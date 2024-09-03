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
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	rand2 "math/rand"
	"time"

	dsciv1 "github.com/opendatahub-io/opendatahub-operator/v2/apis/dscinitialization/v1"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	networkingv1ac "k8s.io/client-go/applyconfigurations/networking/v1"
	rbacv1ac "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	routev1 "github.com/openshift/api/route/v1"
	routev1ac "github.com/openshift/client-go/route/applyconfigurations/route/v1"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
)

// RayClusterReconciler reconciles a RayCluster object
type RayClusterReconciler struct {
	client.Client
	kubeClient  *kubernetes.Clientset
	routeClient *routev1client.RouteV1Client
	Scheme      *runtime.Scheme
	CookieSalt  string
	Config      *config.KubeRayConfiguration
	IsOpenShift bool
}

const (
	requeueTime            = 10
	controllerName         = "codeflare-raycluster-controller"
	oAuthFinalizer         = "ray.openshift.ai/oauth-finalizer"
	oAuthServicePort       = 443
	oAuthServicePortName   = "oauth-proxy"
	ingressServicePortName = "dashboard"
	logRequeueing          = "requeueing"

	CAPrivateKeyKey = "ca.key"
	CACertKey       = "ca.crt"

	RayClusterNameLabel = "ray.openshift.ai/cluster-name"
)

var (
	deletePolicy  = metav1.DeletePropagationForeground
	deleteOptions = client.DeleteOptions{PropagationPolicy: &deletePolicy}
)

// +kubebuilder:rbac:groups=ray.io,resources=rayclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ray.io,resources=rayclusters/finalizers,verbs=update
// +kubebuilder:rbac:groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;create;patch;delete;get;watch
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;create;update;patch;delete;watch
// +kubebuilder:rbac:groups=authentication.k8s.io,resources=tokenreviews,verbs=create;
// +kubebuilder:rbac:groups=authorization.k8s.io,resources=subjectaccessreviews,verbs=create;
// +kubebuilder:rbac:groups=dscinitialization.opendatahub.io,resources=dscinitializations,verbs=get;list;watch
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;create;update;patch;delete;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RayCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.15.3/pkg/reconcile

func shouldUseOldName(cluster *rayv1.RayCluster) bool {
	// hashed name code was added in the same commit as the version annotation
	_, ok := cluster.GetAnnotations()[versionAnnotation]
	return !ok
}

func (r *RayClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	cluster := &rayv1.RayCluster{}

	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Error getting RayCluster resource")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(cluster, oAuthFinalizer) {
			logger.Info("Add a finalizer", "finalizer", oAuthFinalizer)
			controllerutil.AddFinalizer(cluster, oAuthFinalizer)
			if err := r.Update(ctx, cluster); err != nil {
				// this log is info level since errors are not fatal and are expected
				logger.Info("WARN: Failed to update RayCluster with finalizer", "error", err.Error(), logRequeueing, true)
				return ctrl.Result{RequeueAfter: requeueTime}, err
			}
		}
	} else if controllerutil.ContainsFinalizer(cluster, oAuthFinalizer) {
		err := client.IgnoreNotFound(r.Client.Delete(
			ctx,
			&rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: crbNameFromCluster(cluster),
				},
			},
			&deleteOptions,
		))
		if err != nil {
			logger.Error(err, "Failed to remove OAuth ClusterRoleBinding.", logRequeueing, true)
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		controllerutil.RemoveFinalizer(cluster, oAuthFinalizer)
		if err := r.Update(ctx, cluster); err != nil {
			logger.Error(err, "Failed to remove finalizer from RayCluster", logRequeueing, true)
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		logger.Info("Successfully removed finalizer.", logRequeueing, false)
		return ctrl.Result{}, nil
	}

	if isMTLSEnabled(r.Config) {
		caSecretName := caSecretNameFromCluster(cluster)
		caSecret, err := r.kubeClient.CoreV1().Secrets(cluster.Namespace).Get(ctx, caSecretName, metav1.GetOptions{})
		if errors.IsNotFound(err) {
			key, cert, err := generateCACertificate()
			if err != nil {
				logger.Error(err, "Failed to generate CA certificate")
				return ctrl.Result{RequeueAfter: requeueTime}, err
			}
			_, err = r.kubeClient.CoreV1().Secrets(cluster.Namespace).Apply(ctx, desiredCASecret(cluster, key, cert), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
			if err != nil {
				logger.Error(err, "Failed to apply CA Secret")
				return ctrl.Result{RequeueAfter: requeueTime}, err
			}
		} else if err != nil {
			logger.Error(err, "Failed to get CA Secret")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		} else {
			key := caSecret.Data[CAPrivateKeyKey]
			cert := caSecret.Data[CACertKey]
			_, err = r.kubeClient.CoreV1().Secrets(cluster.Namespace).Apply(ctx, desiredCASecret(cluster, key, cert), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
			if err != nil {
				logger.Error(err, "Failed to apply CA Secret")
				return ctrl.Result{RequeueAfter: requeueTime}, err
			}
		}
	}

	if cluster.Status.State != "suspended" && isRayDashboardOAuthEnabled(r.Config) && r.IsOpenShift {
		logger.Info("Creating OAuth Objects")
		_, err := r.routeClient.Routes(cluster.Namespace).Apply(ctx, desiredClusterRoute(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth Route")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		_, err = r.kubeClient.CoreV1().Secrets(cluster.Namespace).Apply(ctx, desiredOAuthSecret(cluster, r.CookieSalt), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to create OAuth Secret")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		_, err = r.kubeClient.CoreV1().Services(cluster.Namespace).Apply(ctx, desiredOAuthService(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth Service")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		_, err = r.kubeClient.CoreV1().ServiceAccounts(cluster.Namespace).Apply(ctx, desiredServiceAccount(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth ServiceAccount")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		if err := r.deleteHeadPodIfMissingImagePullSecrets(ctx, cluster); err != nil {
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		_, err = r.kubeClient.RbacV1().ClusterRoleBindings().Apply(ctx, desiredOAuthClusterRoleBinding(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update OAuth ClusterRoleBinding")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

		logger.Info("Creating RayClient Route")
		_, err = r.routeClient.Routes(cluster.Namespace).Apply(ctx, desiredRayClientRoute(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update RayClient Route")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}

	} else if cluster.Status.State != "suspended" && !isRayDashboardOAuthEnabled(r.Config) && !r.IsOpenShift {
		logger.Info("We detected being on Vanilla Kubernetes!")
		logger.Info("Creating Dashboard Ingress")
		dashboardName := dashboardNameFromCluster(cluster)
		dashboardIngressHost, err := getIngressHost(r.Config, cluster, dashboardName)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		_, err = r.kubeClient.NetworkingV1().Ingresses(cluster.Namespace).Apply(ctx, desiredClusterIngress(cluster, dashboardIngressHost), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			// This log is info level since errors are not fatal and are expected
			logger.Info("WARN: Failed to update Dashboard Ingress", "error", err.Error(), logRequeueing, true)
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		logger.Info("Creating RayClient Ingress")
		rayClientName := rayClientNameFromCluster(cluster)
		rayClientIngressHost, err := getIngressHost(r.Config, cluster, rayClientName)
		if err != nil {
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
		_, err = r.kubeClient.NetworkingV1().Ingresses(cluster.Namespace).Apply(ctx, desiredRayClientIngress(cluster, rayClientIngressHost), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
		if err != nil {
			logger.Error(err, "Failed to update RayClient Ingress")
			return ctrl.Result{RequeueAfter: requeueTime}, err
		}
	}

	// Locate the KubeRay operator deployment:
	// - First try to get the ODH / RHOAI application namespace from the DSCInitialization
	// - Or fallback to the well-known defaults
	var kubeRayNamespaces []string
	dsci := &dsciv1.DSCInitialization{}
	err := r.Client.Get(ctx, client.ObjectKey{Name: "default-dsci"}, dsci)
	if errors.IsNotFound(err) {
		kubeRayNamespaces = []string{"opendatahub", "redhat-ods-applications"}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		kubeRayNamespaces = []string{dsci.Spec.ApplicationsNamespace}
	}

	_, err = r.kubeClient.NetworkingV1().NetworkPolicies(cluster.Namespace).Apply(ctx, desiredHeadNetworkPolicy(cluster, r.Config, kubeRayNamespaces), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
	if err != nil {
		logger.Error(err, "Failed to update NetworkPolicy")
	}

	_, err = r.kubeClient.NetworkingV1().NetworkPolicies(cluster.Namespace).Apply(ctx, desiredWorkersNetworkPolicy(cluster), metav1.ApplyOptions{FieldManager: controllerName, Force: true})
	if err != nil {
		logger.Error(err, "Failed to update NetworkPolicy")
	}

	return ctrl.Result{}, nil
}

// getIngressHost generates the cluster URL string based on the cluster type, RayCluster, and ingress domain.
func getIngressHost(cfg *config.KubeRayConfiguration, cluster *rayv1.RayCluster, ingressNameFromCluster string) (string, error) {
	ingressDomain := ""
	if cfg != nil && cfg.IngressDomain != "" {
		ingressDomain = cfg.IngressDomain
	} else {
		return "", fmt.Errorf("missing IngressDomain configuration in ConfigMap 'codeflare-operator-config'")
	}
	return fmt.Sprintf("%s-%s.%s", ingressNameFromCluster, cluster.Namespace, ingressDomain), nil
}

func isRayDashboardOAuthEnabled(cfg *config.KubeRayConfiguration) bool {
	return cfg == nil || ptr.Deref(cfg.RayDashboardOAuthEnabled, true)
}

func isMTLSEnabled(cfg *config.KubeRayConfiguration) bool {
	return cfg == nil || ptr.Deref(cfg.MTLSEnabled, true)
}

func crbNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-" + cluster.Namespace + "-auth"
	}
	return RCCUniqueName(cluster.Name + "-" + cluster.Namespace + "-auth")
}

func desiredOAuthClusterRoleBinding(cluster *rayv1.RayCluster) *rbacv1ac.ClusterRoleBindingApplyConfiguration {
	return rbacv1ac.ClusterRoleBinding(
		crbNameFromCluster(cluster)).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name, "ray.openshift.ai/cluster-namespace": cluster.Namespace}).
		WithSubjects(
			rbacv1ac.Subject().
				WithKind("ServiceAccount").
				WithName(oauthServiceAccountNameFromCluster(cluster)).
				WithNamespace(cluster.Namespace),
		).
		WithRoleRef(
			rbacv1ac.RoleRef().
				WithAPIGroup("rbac.authorization.k8s.io").
				WithKind("ClusterRole").
				WithName("system:auth-delegator"),
		)
}

func oauthServiceAccountNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-oauth-proxy"
	}
	return RCCUniqueName(cluster.Name + "-oauth-proxy")
}

func desiredServiceAccount(cluster *rayv1.RayCluster) *corev1ac.ServiceAccountApplyConfiguration {
	return corev1ac.ServiceAccount(oauthServiceAccountNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithAnnotations(map[string]string{
			"serviceaccounts.openshift.io/oauth-redirectreference.first": "" +
				`{"kind":"OAuthRedirectReference","apiVersion":"v1",` +
				`"reference":{"kind":"Route","name":"` + dashboardNameFromCluster(cluster) + `"}}`,
		}).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func dashboardNameFromCluster(cluster *rayv1.RayCluster) string {
	return "ray-dashboard-" + cluster.Name
}

func rayClientNameFromCluster(cluster *rayv1.RayCluster) string {
	return "rayclient-" + cluster.Name
}

func desiredClusterRoute(cluster *rayv1.RayCluster) *routev1ac.RouteApplyConfiguration {
	return routev1ac.Route(dashboardNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithSpec(routev1ac.RouteSpec().
			WithTo(routev1ac.RouteTargetReference().WithKind("Service").WithName(oauthServiceNameFromCluster(cluster))).
			WithPort(routev1ac.RoutePort().WithTargetPort(intstr.FromString((oAuthServicePortName)))).
			WithTLS(routev1ac.TLSConfig().
				WithInsecureEdgeTerminationPolicy(routev1.InsecureEdgeTerminationPolicyRedirect).
				WithTermination(routev1.TLSTerminationReencrypt),
			),
		).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func oauthServiceNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-oauth"
	}
	return RCCUniqueName(cluster.Name + "-oauth")
}

func oauthServiceTLSSecretName(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-proxy-tls-secret"
	}
	return RCCUniqueName(cluster.Name + "-proxy-tls-secret")
}

func desiredOAuthService(cluster *rayv1.RayCluster) *corev1ac.ServiceApplyConfiguration {
	return corev1ac.Service(oauthServiceNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithAnnotations(map[string]string{"service.beta.openshift.io/serving-cert-secret-name": oauthServiceTLSSecretName(cluster)}).
		WithSpec(
			corev1ac.ServiceSpec().
				WithPorts(
					corev1ac.ServicePort().
						WithName(oAuthServicePortName).
						WithPort(oAuthServicePort).
						WithTargetPort(intstr.FromString(oAuthServicePortName)).
						WithProtocol(corev1.ProtocolTCP),
				).
				WithSelector(map[string]string{"ray.io/cluster": cluster.Name, "ray.io/node-type": "head"}),
		).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func oauthSecretNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-oauth-config"
	}
	return RCCUniqueName(cluster.Name + "-oauth-config")
}

// desiredOAuthSecret defines the desired OAuth secret object
func desiredOAuthSecret(cluster *rayv1.RayCluster, cookieSalt string) *corev1ac.SecretApplyConfiguration {
	// Generate the cookie secret for the OAuth proxy
	hasher := sha1.New() // REVIEW is SHA1 okay here?
	hasher.Write([]byte(cluster.Name + cookieSalt))
	cookieSecret := base64.StdEncoding.EncodeToString(hasher.Sum(nil))

	return corev1ac.Secret(oauthSecretNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithStringData(map[string]string{"cookie_secret": cookieSecret}).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func caSecretNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return "ca-secret-" + cluster.Name
	}
	return RCCUniqueName(cluster.Name + "-ca-secret")
}

func desiredCASecret(cluster *rayv1.RayCluster, key, cert []byte) *corev1ac.SecretApplyConfiguration {
	return corev1ac.Secret(caSecretNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithData(map[string][]byte{
			CAPrivateKeyKey: key,
			CACertKey:       cert,
		}).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func generateCACertificate() ([]byte, []byte, error) {
	serialNumber := big.NewInt(rand2.Int63())
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "root-ca",
		},
		Issuer: pkix.Name{
			CommonName: "root-ca",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	certPrivateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(certPrivateKey)
	privateKeyPem := pem.EncodeToMemory(
		&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		},
	)
	certBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &certPrivateKey.PublicKey, certPrivateKey)
	if err != nil {
		return nil, nil, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	return privateKeyPem, certPem, nil
}

func workerNWPNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-workers"
	}
	return RCCUniqueName(cluster.Name + "-workers")
}

func desiredWorkersNetworkPolicy(cluster *rayv1.RayCluster) *networkingv1ac.NetworkPolicyApplyConfiguration {
	return networkingv1ac.NetworkPolicy(
		workerNWPNameFromCluster(cluster), cluster.Namespace,
	).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithSpec(networkingv1ac.NetworkPolicySpec().
			WithPodSelector(metav1ac.LabelSelector().WithMatchLabels(map[string]string{"ray.io/cluster": cluster.Name, "ray.io/node-type": "worker"})).
			WithIngress(
				networkingv1ac.NetworkPolicyIngressRule().
					WithFrom(
						networkingv1ac.NetworkPolicyPeer().WithPodSelector(metav1ac.LabelSelector().WithMatchLabels(map[string]string{"ray.io/cluster": cluster.Name})),
					),
			),
		).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func headNWPNameFromCluster(cluster *rayv1.RayCluster) string {
	if shouldUseOldName(cluster) {
		return cluster.Name + "-head"
	}
	return RCCUniqueName(cluster.Name + "-head")
}

func desiredHeadNetworkPolicy(cluster *rayv1.RayCluster, cfg *config.KubeRayConfiguration, kubeRayNamespaces []string) *networkingv1ac.NetworkPolicyApplyConfiguration {
	allSecuredPorts := []*networkingv1ac.NetworkPolicyPortApplyConfiguration{
		networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(8443)),
	}
	if ptr.Deref(cfg.MTLSEnabled, true) {
		allSecuredPorts = append(allSecuredPorts, networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(10001)))
	}
	return networkingv1ac.NetworkPolicy(headNWPNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{RayClusterNameLabel: cluster.Name}).
		WithSpec(networkingv1ac.NetworkPolicySpec().
			WithPodSelector(metav1ac.LabelSelector().WithMatchLabels(map[string]string{"ray.io/cluster": cluster.Name, "ray.io/node-type": "head"})).
			WithIngress(
				networkingv1ac.NetworkPolicyIngressRule().
					WithFrom(
						networkingv1ac.NetworkPolicyPeer().WithPodSelector(metav1ac.LabelSelector().WithMatchLabels(map[string]string{"ray.io/cluster": cluster.Name})),
					),
				networkingv1ac.NetworkPolicyIngressRule().
					WithPorts(
						networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(10001)),
						networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(8265)),
					).WithFrom(
					networkingv1ac.NetworkPolicyPeer().WithPodSelector(metav1ac.LabelSelector()),
				),
				networkingv1ac.NetworkPolicyIngressRule().
					WithFrom(
						networkingv1ac.NetworkPolicyPeer().WithPodSelector(metav1ac.LabelSelector().
							WithMatchLabels(map[string]string{"app.kubernetes.io/component": "kuberay-operator"})).
							WithNamespaceSelector(metav1ac.LabelSelector().
								WithMatchExpressions(metav1ac.LabelSelectorRequirement().
									WithKey(corev1.LabelMetadataName).
									WithOperator(metav1.LabelSelectorOpIn).
									WithValues(kubeRayNamespaces...)))).
					WithPorts(
						networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(8265)),
						networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(10001)),
					),
				networkingv1ac.NetworkPolicyIngressRule().
					WithPorts(
						networkingv1ac.NetworkPolicyPort().WithProtocol(corev1.ProtocolTCP).WithPort(intstr.FromInt(8080)),
					).
					WithFrom(
						networkingv1ac.NetworkPolicyPeer().WithNamespaceSelector(metav1ac.LabelSelector().
							WithMatchExpressions(metav1ac.LabelSelectorRequirement().
								WithKey(corev1.LabelMetadataName).
								WithOperator(metav1.LabelSelectorOpIn).
								WithValues("openshift-monitoring"))),
					),
				networkingv1ac.NetworkPolicyIngressRule().
					WithPorts(
						allSecuredPorts...,
					),
			),
		).
		WithOwnerReferences(ownerRefForRayCluster(cluster))
}

func (r *RayClusterReconciler) deleteHeadPodIfMissingImagePullSecrets(ctx context.Context, cluster *rayv1.RayCluster) error {
	serviceAccount, err := r.kubeClient.CoreV1().ServiceAccounts(cluster.Namespace).Get(ctx, oauthServiceAccountNameFromCluster(cluster), metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get OAuth ServiceAccount: %w", err)
	}

	headPod, err := getHeadPod(ctx, r, cluster)
	if err != nil {
		return fmt.Errorf("failed to get head pod: %w", err)
	}

	if headPod == nil {
		return nil
	}

	missingSecrets := map[string]bool{}
	for _, secret := range serviceAccount.ImagePullSecrets {
		missingSecrets[secret.Name] = true
	}
	for _, secret := range headPod.Spec.ImagePullSecrets {
		delete(missingSecrets, secret.Name)
	}
	if len(missingSecrets) > 0 {
		if err := r.kubeClient.CoreV1().Pods(headPod.Namespace).Delete(ctx, headPod.Name, metav1.DeleteOptions{}); err != nil {
			return fmt.Errorf("failed to delete head pod: %w", err)
		}
	}
	return nil
}

func getHeadPod(ctx context.Context, r *RayClusterReconciler, cluster *rayv1.RayCluster) (*corev1.Pod, error) {
	podList, err := r.kubeClient.CoreV1().Pods(cluster.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("ray.io/node-type=head,ray.io/cluster=%s", cluster.Name),
	})
	if err != nil {
		return nil, err
	}
	if len(podList.Items) > 0 {
		return &podList.Items[0], nil
	}
	return nil, nil
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
	// despite ownership, we need to check for labels because we can't use
	controller := ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		For(&rayv1.RayCluster{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.Secret{}).
		Owns(&networkingv1.Ingress{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Watches(&rbacv1.ClusterRoleBinding{}, handler.EnqueueRequestsFromMapFunc(
			func(c context.Context, o client.Object) []reconcile.Request {
				name, ok := o.GetLabels()[RayClusterNameLabel]
				if !ok {
					return []reconcile.Request{}
				}
				namespace, ok := o.GetLabels()["ray.openshift.ai/cluster-namespace"]
				if !ok {
					return []reconcile.Request{}
				}
				return []reconcile.Request{{
					NamespacedName: client.ObjectKey{
						Name:      name,
						Namespace: namespace,
					},
				}}
			}),
		)
	if r.IsOpenShift {
		controller.Owns(&routev1.Route{})
	}

	return controller.Complete(r)
}

func RCCUniqueName(s string) string {
	return s + "-" + seededHash(controllerName, s)
}
