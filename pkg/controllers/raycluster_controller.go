/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"

	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreapply "k8s.io/client-go/applyconfigurations/core/v1"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	networkingv1ac "k8s.io/client-go/applyconfigurations/networking/v1"
	rbacapply "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	routev1 "github.com/openshift/api/route/v1"
	routeapply "github.com/openshift/client-go/route/applyconfigurations/route/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
)

// RayClusterReconciler reconciles a RayCluster object
type RayClusterReconciler struct {
	client.Client
	kubeClient  *kubernetes.Clientset
	routeClient *routev1client.RouteV1Client
	Scheme      *runtime.Scheme
	CookieSalt  string
}

const (
	requeueTime             = 10
	controllerName          = "codeflare-raycluster-controller"
	oAuthFinalizer          = "ray.openshift.ai/oauth-finalizer"
	oAuthServicePort        = 443
	oAuthServicePortName    = "oauth-proxy"
	oauthAnnotation         = "codeflare.dev/oauth"
	RegularServicePortName  = "dashboard"
	logRequeueing           = "requeueing"
)

var (
	deletePolicy  = metav1.DeletePropagationForeground
	deleteOptions = client.DeleteOptions{PropagationPolicy: &deletePolicy}
)

// +kubebuilder:rbac:groups=ray.io,resources=rayclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=patch;delete;get
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;patch;delete;get
// +kubebuilder:rbac:groups=core,resources=services,verbs=patch;delete;get
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=patch;delete;get
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=patch;delete;get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RayCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.3/pkg/reconcile

func (r *RayClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var cluster rayv1.RayCluster

	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Error getting RayCluster resource")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	isLocalInteractive := annotationBoolVal(logger, &cluster, "sdk.codeflare.dev/local_interactive")
	isOpenShift, ingressHost := getClusterType(logger, r.kubeClient, &cluster)
	ingressDomain := cluster.ObjectMeta.Annotations["sdk.codeflare.dev/ingress_domain"]

	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&cluster, oAuthFinalizer) {
			logger.Info("Add a finalizer", "finalizer", oAuthFinalizer)
			controllerutil.AddFinalizer(&cluster, oAuthFinalizer)
			if err := r.Update(ctx, &cluster); err != nil {
				// this log is info level since errors are not fatal and are expected
				logger.Info("WARN: Failed to update RayCluster with finalizer", "error", err.Error(), logRequeueing, true)
				return ctrl.Result{RequeueAfter: requeueTime}, err
			}
		}
	} else if controllerutil.ContainsFinalizer(&cluster, oAuthFinalizer) {
		err := client.IgnoreNotFound(r.Client.Delete(
			ctx,
			&rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: crbNameFromCluster(&cluster),
				},
			},
			&deleteOptions,
		))
		if err != nil {
			logger.Error(err, "Failed to remove OAuth ClusterRoleBinding.", logRequeueing, true)
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		controllerutil.RemoveFinalizer(&cluster, oAuthFinalizer)
		if err := r.Update(ctx, &cluster); err != nil {
			logger.Error(err, "Failed to remove finalizer from RayCluster", logRequeueing, true)
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		logger.Info("Successfully removed finalizer.", logRequeueing, false)
		return ctrl.Result{}, nil
	}

	if cluster.Status.State != "suspended" && annotationBoolVal(logger, &cluster, oauthAnnotation) && isOpenShift {
		logger.Info("Creating OAuth Objects")
		_, err := r.routeClient.Routes(cluster.Namespace).Apply(ctx, desiredClusterRoute(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth Route")
		}

		_, err = r.kubeClient.CoreV1().Secrets(cluster.Namespace).Apply(ctx, desiredOAuthSecret(&cluster, r), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to create OAuth Secret")
		}

		_, err = r.kubeClient.CoreV1().Services(cluster.Namespace).Apply(ctx, desiredOAuthService(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth Service")
		}

		_, err = r.kubeClient.CoreV1().ServiceAccounts(cluster.Namespace).Apply(ctx, desiredServiceAccount(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth ServiceAccount")
		}

		_, err = r.kubeClient.RbacV1().ClusterRoleBindings().Apply(ctx, desiredOAuthClusterRoleBinding(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth ClusterRoleBinding")
		}

	} else if cluster.Status.State != "suspended" && !annotationBoolVal(logger, &cluster, oauthAnnotation) && isOpenShift {
		logger.Info("Creating Dashboard Route")
		_, err := r.routeClient.Routes(cluster.Namespace).Apply(ctx, createRoute(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update Dashboard Route")
		}
		if isLocalInteractive && ingressDomain != "" {
			logger.Info("Creating RayClient Route")
			_, err := r.routeClient.Routes(cluster.Namespace).Apply(ctx, createRayClientRoute(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
			if err != nil {
				logger.Error(err, "Failed to update RayClient Route")
			}
		}
		return ctrl.Result{}, nil

	} else if cluster.Status.State != "suspended" && !annotationBoolVal(logger, &cluster, oauthAnnotation) && !isOpenShift {
		logger.Info("Creating Dashboard Ingress")
		_, err := r.kubeClient.NetworkingV1().Ingresses(cluster.Namespace).Apply(ctx, createIngressApplyConfiguration(&cluster, ingressHost), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			// This log is info level since errors are not fatal and are expected
			logger.Info("WARN: Failed to update Dashboard Ingress", "error", err.Error(), logRequeueing, true)
		}
		if isLocalInteractive && ingressDomain != "" {
			logger.Info("Creating RayClient Ingress")
			_, err := r.kubeClient.NetworkingV1().Ingresses(cluster.Namespace).Apply(ctx, createRayClientIngress(&cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
			if err != nil {
				logger.Error(err, "Failed to update RayClient Ingress")
			}
		}
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

func annotationBoolVal(logger logr.Logger, cluster *rayv1.RayCluster, annotation string) bool {
	val := cluster.ObjectMeta.Annotations[annotation]
	boolVal, err := strconv.ParseBool(val)
	if err != nil {
		logger.Error(err, "Could not convert", annotation, "value to bool", val)
	}
	if boolVal {
		return true
	} else {
		return false
	}
}

func crbNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-" + cluster.Namespace + "-auth" // NOTE: potential naming conflicts ie {name: foo, ns: bar-baz} and {name: foo-bar, ns: baz}
}

func desiredOAuthClusterRoleBinding(cluster *rayv1.RayCluster) *rbacapply.ClusterRoleBindingApplyConfiguration {
	return rbacapply.ClusterRoleBinding(
		crbNameFromCluster(cluster)).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSubjects(
			rbacapply.Subject().
				WithKind("ServiceAccount").
				WithName(oauthServiceAccountNameFromCluster(cluster)).
				WithNamespace(cluster.Namespace),
		).
		WithRoleRef(
			rbacapply.RoleRef().
				WithAPIGroup("rbac.authorization.k8s.io").
				WithKind("ClusterRole").
				WithName("system:auth-delegator"),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func oauthServiceAccountNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-oauth-proxy"
}

func desiredServiceAccount(cluster *rayv1.RayCluster) *coreapply.ServiceAccountApplyConfiguration {
	return coreapply.ServiceAccount(oauthServiceAccountNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithAnnotations(map[string]string{
			"serviceaccounts.openshift.io/oauth-redirectreference.first": "" +
				`{"kind":"OAuthRedirectReference","apiVersion":"v1",` +
				`"reference":{"kind":"Route","name":"` + dashboardNameFromCluster(cluster) + `"}}`,
		}).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func dashboardNameFromCluster(cluster *rayv1.RayCluster) string {
	return "ray-dashboard-" + cluster.Name
}

func rayClientNameFromCluster(cluster *rayv1.RayCluster) string {
	return "rayclient-" + cluster.Name
}

func desiredClusterRoute(cluster *rayv1.RayCluster) *routeapply.RouteApplyConfiguration {
	return routeapply.Route(dashboardNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSpec(routeapply.RouteSpec().
			WithTo(routeapply.RouteTargetReference().WithKind("Service").WithName(oauthServiceNameFromCluster(cluster))).
			WithPort(routeapply.RoutePort().WithTargetPort(intstr.FromString((oAuthServicePortName)))).
			WithTLS(routeapply.TLSConfig().
				WithInsecureEdgeTerminationPolicy(routev1.InsecureEdgeTerminationPolicyRedirect).
				WithTermination(routev1.TLSTerminationReencrypt),
			),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func oauthServiceNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-oauth"
}

func oauthServiceTLSSecretName(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-proxy-tls-secret"
}

func desiredOAuthService(cluster *rayv1.RayCluster) *coreapply.ServiceApplyConfiguration {
	return coreapply.Service(oauthServiceNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithAnnotations(map[string]string{"service.beta.openshift.io/serving-cert-secret-name": oauthServiceTLSSecretName(cluster)}).
		WithSpec(
			coreapply.ServiceSpec().
				WithPorts(
					coreapply.ServicePort().
						WithName(oAuthServicePortName).
						WithPort(oAuthServicePort).
						WithTargetPort(intstr.FromString(oAuthServicePortName)).
						WithProtocol(corev1.ProtocolTCP),
				).
				WithSelector(map[string]string{"ray.io/cluster": cluster.Name, "ray.io/node-type": "head"}),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func oauthSecretNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-oauth-config"
}

// desiredOAuthSecret defines the desired OAuth secret object
func desiredOAuthSecret(cluster *rayv1.RayCluster, r *RayClusterReconciler) *coreapply.SecretApplyConfiguration {
	// Generate the cookie secret for the OAuth proxy
	hasher := sha1.New() // REVIEW is SHA1 okay here?
	hasher.Write([]byte(cluster.Name + r.CookieSalt))
	cookieSecret := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return coreapply.Secret(oauthSecretNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithStringData(map[string]string{"cookie_secret": cookieSecret}).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
	// Create a Kubernetes secret to store the cookie secret
}

// SetupWithManager sets up the controller with the Manager.
func (r *RayClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.kubeClient = kubernetes.NewForConfigOrDie(mgr.GetConfig())
	r.routeClient = routev1client.NewForConfigOrDie(mgr.GetConfig())
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return err
	}
	r.CookieSalt = string(b)
	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&rayv1.RayCluster{}).
		Complete(r)
}

func serviceNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-head-svc"
}

func createRoute(cluster *rayv1.RayCluster) *routeapply.RouteApplyConfiguration {
	return routeapply.Route(dashboardNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSpec(routeapply.RouteSpec().
			WithTo(routeapply.RouteTargetReference().WithKind("Service").WithName(serviceNameFromCluster(cluster))).
			WithPort(routeapply.RoutePort().WithTargetPort(intstr.FromString(RegularServicePortName))).
			WithTLS(routeapply.TLSConfig().
				WithTermination("edge")),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func createRayClientRoute(cluster *rayv1.RayCluster) *routeapply.RouteApplyConfiguration {
	ingress_domain := cluster.ObjectMeta.Annotations["sdk.codeflare.dev/ingress_domain"]
	return routeapply.Route(rayClientNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSpec(routeapply.RouteSpec().
			WithHost(rayClientNameFromCluster(cluster) + "-" + cluster.Namespace + "." + ingress_domain).
			WithTo(routeapply.RouteTargetReference().WithKind("Service").WithName(serviceNameFromCluster(cluster)).WithWeight(100)).
			WithPort(routeapply.RoutePort().WithTargetPort(intstr.FromString("client"))).
			WithTLS(routeapply.TLSConfig().WithTermination("passthrough")),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

// Create an Ingress object for the RayCluster
func createRayClientIngress(cluster *rayv1.RayCluster) *networkingv1ac.IngressApplyConfiguration {
	ingress_domain := cluster.ObjectMeta.Annotations["sdk.codeflare.dev/ingress_domain"]
	return networkingv1ac.Ingress(rayClientNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithAnnotations(map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target":  "/",
			"nginx.ingress.kubernetes.io/ssl-redirect":    "true",
			"nginx.ingress.kubernetes.io/ssl-passthrough": "true",
		}).
		WithOwnerReferences(v1.OwnerReference().
			WithAPIVersion(cluster.APIVersion).
			WithKind(cluster.Kind).
			WithName(cluster.Name).
			WithUID(types.UID(cluster.UID))).
		WithSpec(networkingv1ac.IngressSpec().
			WithIngressClassName("nginx").
			WithRules(networkingv1ac.IngressRule().
				WithHost(rayClientNameFromCluster(cluster) + "-" + cluster.Namespace + "." + ingress_domain).
				WithHTTP(networkingv1ac.HTTPIngressRuleValue().
					WithPaths(networkingv1ac.HTTPIngressPath().
						WithPath("/").
						WithPathType(networkingv1.PathTypeImplementationSpecific).
						WithBackend(networkingv1ac.IngressBackend().
							WithService(networkingv1ac.IngressServiceBackend().
								WithName(serviceNameFromCluster(cluster)).
								WithPort(networkingv1ac.ServiceBackendPort().
									WithNumber(10001),
								),
							),
						),
					),
				),
			),
		)
	// Optionally, add TLS configuration here if needed
}

// Create an Ingress object for the RayCluster
func createIngressApplyConfiguration(cluster *rayv1.RayCluster, ingressHost string) *networkingv1ac.IngressApplyConfiguration {
	return networkingv1ac.Ingress(dashboardNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithOwnerReferences(v1.OwnerReference().
			WithAPIVersion(cluster.APIVersion).
			WithKind(cluster.Kind).
			WithName(cluster.Name).
			WithUID(types.UID(cluster.UID))).
		WithSpec(networkingv1ac.IngressSpec().
			WithRules(networkingv1ac.IngressRule().
				WithHost(ingressHost). // kind host name or ingress_domain
				WithHTTP(networkingv1ac.HTTPIngressRuleValue().
					WithPaths(networkingv1ac.HTTPIngressPath().
						WithPath("/").
						WithPathType(networkingv1.PathTypePrefix).
						WithBackend(networkingv1ac.IngressBackend().
							WithService(networkingv1ac.IngressServiceBackend().
								WithName(serviceNameFromCluster(cluster)).
								WithPort(networkingv1ac.ServiceBackendPort().
									WithName(RegularServicePortName),
								),
							),
						),
					),
				),
			),
		)
	// Optionally, add TLS configuration here if needed
}

// isOnKindCluster checks if the current cluster is a KinD cluster.
// It searches for a node with a label commonly used by KinD clusters.
func isOnKindCluster(clientset *kubernetes.Clientset) (bool, error) {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kubernetes.io/hostname=kind-control-plane",
	})
	if err != nil {
		return false, err
	}
	// If we find one or more nodes with the label, assume it's a KinD cluster.
	return len(nodes.Items) > 0, nil
}

// getDiscoveryClient returns a discovery client for the current reconciler
func getDiscoveryClient(config *rest.Config) (*discovery.DiscoveryClient, error) {
	return discovery.NewDiscoveryClientForConfig(config)
}

// Check where we are running. We are trying to distinguish here whether
// this is vanilla kubernetes cluster or Openshift
func getClusterType(logger logr.Logger, clientset *kubernetes.Clientset, cluster *rayv1.RayCluster) (bool, string) {
	// The discovery package is used to discover APIs supported by a Kubernetes API server.
	ingress_domain := cluster.ObjectMeta.Annotations["sdk.codeflare.dev/ingress_domain"]
	config, err := ctrl.GetConfig()
	if err == nil && config != nil {
		dclient, err := getDiscoveryClient(config)
		if err == nil && dclient != nil {
			apiGroupList, err := dclient.ServerGroups()
			if err != nil {
				logger.Info("Error while querying ServerGroups, assuming we're on Vanilla Kubernetes")
				return false, ""
			} else {
				for i := 0; i < len(apiGroupList.Groups); i++ {
					if strings.HasSuffix(apiGroupList.Groups[i].Name, ".openshift.io") {
						logger.Info("We detected being on OpenShift!")
						return true, ""
					}
				}
				onKind, _ := isOnKindCluster(clientset)
				if onKind && ingress_domain == "" {
					logger.Info("We detected being on a KinD cluster!")
					return false, "kind"
				} else {
					logger.Info("We detected being on Vanilla Kubernetes!")
					return false, fmt.Sprintf("ray-dashboard-%s-%s.%s", cluster.Name, cluster.Namespace, ingress_domain)
				}
			}
		} else {
			logger.Info("Cannot retrieve a DiscoveryClient, assuming we're on Vanilla Kubernetes")
			return false, fmt.Sprintf("ray-dashboard-%s-%s.%s", cluster.Name, cluster.Namespace, ingress_domain)
		}
	} else {
		logger.Info("Cannot retrieve config, assuming we're on Vanilla Kubernetes")
		return false, fmt.Sprintf("ray-dashboard-%s-%s.%s", cluster.Name, cluster.Namespace, ingress_domain)
	}
}

// No more ingress_options - Removing completely.
// What to do about ingress_domain? Needed for local_interactive?
