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

	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	coreapply "k8s.io/client-go/applyconfigurations/core/v1"
	v1 "k8s.io/client-go/applyconfigurations/meta/v1"
	rbacapply "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/kubernetes"
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
	oauthAnnotation         = "codeflare.dev/oauth=true"
	CodeflareOAuthFinalizer = "codeflare.dev/oauth-finalizer"
	OAuthServicePort        = 443
	OAuthServicePortName    = "oauth-proxy"
	OAuthProxyImage         = "registry.redhat.io/openshift4/ose-oauth-proxy:latest"
	strTrue                 = "true"
	strFalse                = "false"
	logRequeueing           = "requeueing"
)

var (
	deletePolicy  = metav1.DeletePropagationForeground
	deleteOptions = client.DeleteOptions{PropagationPolicy: &deletePolicy}
)

//+kubebuilder:rbac:groups=ray.io,resources=rayclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ray.io,resources=rayclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ray.io,resources=rayclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups=route.openshift.io,resources=routes,verbs=patch;delete;get
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;create;patch;delete;get
//+kubebuilder:rbac:groups=core,resources=services,verbs=patch;delete;get
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=patch;delete;get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RayCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.4/pkg/reconcile

func (r *RayClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := ctrl.LoggerFrom(ctx)

	var cluster rayv1.RayCluster

	if err := r.Get(ctx, req.NamespacedName, &cluster); err != nil {
		if !errors.IsNotFound(err) {
			logger.Error(err, "Error getting RayCluster resource")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(&cluster, CodeflareOAuthFinalizer) {
			logger.Info("Add a finalizer", "finalizer", CodeflareOAuthFinalizer)
			controllerutil.AddFinalizer(&cluster, CodeflareOAuthFinalizer)
			if err := r.Update(ctx, &cluster); err != nil {
				logger.Error(err, "Failed to update RayCluster with finalizer", logRequeueing, strTrue)
				return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, err
			}
		}
	} else if controllerutil.ContainsFinalizer(&cluster, CodeflareOAuthFinalizer) {
		err := r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: crbNameFromCluster(&cluster)}, &rbacv1.ClusterRoleBinding{},
		)
		if err != nil {
			logger.Error(err, "Failed to remove OAuth ClusterRoleBinding.", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, err
		}
		controllerutil.RemoveFinalizer(&cluster, CodeflareOAuthFinalizer)
		if err := r.Update(ctx, &cluster); err != nil {
			logger.Error(err, "Failed to remove finalizer from RayCluster", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, err
		}
		logger.Info("Successfully removed finalizer.", logRequeueing, strFalse)
		return ctrl.Result{}, nil
	}

	if val, ok := cluster.ObjectMeta.Annotations["codeflare.dev/oauth"]; !ok || val != "true" {
		logger.Info("Removing all OAuth Objects")
		err := r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: oauthSecretNameFromCluster(&cluster)}, &corev1.Secret{},
		)
		if err != nil {
			logger.Error(err, "Error deleting OAuth Secret, retrying", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
		}
		err = r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: oauthServiceNameFromCluster(&cluster)}, &corev1.Service{},
		)
		if err != nil {
			logger.Error(err, "Error deleting OAuth Service, retrying", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
		}
		err = r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, &corev1.ServiceAccount{},
		)
		if err != nil {
			logger.Error(err, "Error deleting OAuth ServiceAccount, retrying", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
		}
		err = r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: crbNameFromCluster(&cluster)}, &rbacv1.ClusterRoleBinding{},
		)
		if err != nil {
			logger.Error(err, "Error deleting OAuth CRB, retrying", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
		}
		err = r.deleteIfNotExist(
			ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Name}, &routev1.Route{},
		)
		if err != nil {
			logger.Error(err, "Error deleting OAuth Route, retrying", logRequeueing, strTrue)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueTime}, nil
		}
		return ctrl.Result{}, nil
	}

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

	return ctrl.Result{}, nil
}

func crbNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-" + cluster.Namespace + "-auth" // NOTE: potential naming conflicts ie {name: foo, ns: bar-baz} and {name: foo-bar, ns: baz}
}

func (r *RayClusterReconciler) deleteIfNotExist(ctx context.Context, namespacedName types.NamespacedName, obj client.Object) error {
	err := r.Client.Get(ctx, namespacedName, obj)
	if err != nil && errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	}
	return r.Client.Delete(ctx, obj, &deleteOptions)
}

func desiredOAuthClusterRoleBinding(cluster *rayv1.RayCluster) *rbacapply.ClusterRoleBindingApplyConfiguration {
	return rbacapply.ClusterRoleBinding(
		crbNameFromCluster(cluster)).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSubjects(
			rbacapply.Subject().
				WithKind("ServiceAccount").
				WithName(cluster.Name).
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

func desiredServiceAccount(cluster *rayv1.RayCluster) *coreapply.ServiceAccountApplyConfiguration {
	return coreapply.ServiceAccount(cluster.Name, cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithAnnotations(map[string]string{
			"serviceaccounts.openshift.io/oauth-redirectreference.first": "" +
				`{"kind":"OAuthRedirectReference","apiVersion":"v1",` +
				`"reference":{"kind":"Route","name":"` + cluster.Name + `"}}`,
		}).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func desiredClusterRoute(cluster *rayv1.RayCluster) *routeapply.RouteApplyConfiguration {
	return routeapply.Route(cluster.Name, cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithSpec(routeapply.RouteSpec().
			WithTo(routeapply.RouteTargetReference().WithKind("Service").WithName(oauthServiceNameFromCluster(cluster))).
			WithPort(routeapply.RoutePort().WithTargetPort(intstr.FromString((OAuthServicePortName)))).
			WithTLS(routeapply.TLSConfig().
				WithInsecureEdgeTerminationPolicy(routev1.InsecureEdgeTerminationPolicyRedirect).
				// WithKey().WithCACertificate().WithCACertificate().WithDestinationCACertificate().
				WithTermination(routev1.TLSTerminationReencrypt),
			),
		).
		WithOwnerReferences(
			v1.OwnerReference().WithUID(cluster.UID).WithName(cluster.Name).WithKind(cluster.Kind).WithAPIVersion(cluster.APIVersion),
		)
}

func oauthServiceNameFromCluster(cluster *rayv1.RayCluster) string {
	return cluster.Name + "-tls"
}

func desiredOAuthService(cluster *rayv1.RayCluster) *coreapply.ServiceApplyConfiguration {
	return coreapply.Service(oauthServiceNameFromCluster(cluster), cluster.Namespace).
		WithLabels(map[string]string{"ray.io/cluster-name": cluster.Name}).
		WithAnnotations(map[string]string{"service.beta.openshift.io/serving-cert-secret-name": cluster.Name + "-tls"}).
		WithSpec(
			coreapply.ServiceSpec().
				WithPorts(
					coreapply.ServicePort().
						WithName(OAuthServicePortName).
						WithPort(OAuthServicePort).
						WithTargetPort(intstr.FromString(OAuthServicePortName)).
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

// NewClusterOAuthSecret defines the desired OAuth secret object
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
