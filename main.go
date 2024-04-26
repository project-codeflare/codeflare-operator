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

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	cert "github.com/open-policy-agent/cert-controller/pkg/rotator"
	dsciv1 "github.com/opendatahub-io/opendatahub-operator/v2/apis/dscinitialization/v1"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	retrywatch "k8s.io/client-go/tools/watch"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/yaml"

	routev1 "github.com/openshift/api/route/v1"
	clientset "github.com/openshift/client-go/config/clientset/versioned"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
	"github.com/project-codeflare/codeflare-operator/pkg/controllers"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	// +kubebuilder:scaffold:imports
)

var (
	scheme          = runtime.NewScheme()
	setupLog        = ctrl.Log.WithName("setup")
	OperatorVersion = "UNKNOWN"
	BuildDate       = "UNKNOWN"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Ray
	utilruntime.Must(rayv1.AddToScheme(scheme))
	// OpenShift Route
	utilruntime.Must(routev1.Install(scheme))
	// ODH
	utilruntime.Must(dsciv1.AddToScheme(scheme))
}

// +kubebuilder:rbac:groups=config.openshift.io,resources=ingresses,verbs=get

func main() {
	var configMapName string
	flag.StringVar(&configMapName, "config", "codeflare-operator-config",
		"The name of the ConfigMap to load the operator configuration from. "+
			"If it does not exist, the operator will create and initialise it.")

	zapOptions := zap.Options{
		Development: true,
		TimeEncoder: zapcore.TimeEncoderOfLayout(time.RFC3339),
	}
	zapOptions.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))
	klog.SetLogger(ctrl.Log)

	setupLog.Info("Build info",
		"operatorVersion", OperatorVersion,
		"date", BuildDate,
	)

	ctx := ctrl.SetupSignalHandler()

	cfg := &config.CodeFlareOperatorConfiguration{
		ClientConnection: &config.ClientConnection{
			QPS:   ptr.To(float32(50)),
			Burst: ptr.To(int32(100)),
		},
		ControllerManager: config.ControllerManager{
			Metrics: config.MetricsConfiguration{
				BindAddress: ":8080",
			},
			Health: config.HealthConfiguration{
				BindAddress:           ":8081",
				ReadinessEndpointName: "readyz",
				LivenessEndpointName:  "healthz",
			},
			LeaderElection: &configv1alpha1.LeaderElectionConfiguration{},
		},
		KubeRay: &config.KubeRayConfiguration{
			RayDashboardOAuthEnabled: ptr.To(true),
			IngressDomain:            "",
			MTLSEnabled:              ptr.To(true),
		},
	}

	kubeConfig, err := ctrl.GetConfig()
	exitOnError(err, "unable to get client config")
	if kubeConfig.UserAgent == "" {
		kubeConfig.UserAgent = "codeflare-operator"
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	exitOnError(err, "unable to create Kubernetes client")

	namespace := namespaceOrDie()

	exitOnError(loadIntoOrCreate(ctx, kubeClient, namespace, configMapName, cfg), "unable to initialise configuration")

	kubeConfig.Burst = int(ptr.Deref(cfg.ClientConnection.Burst, int32(rest.DefaultBurst)))
	kubeConfig.QPS = ptr.Deref(cfg.ClientConnection.QPS, rest.DefaultQPS)
	setupLog.V(2).Info("REST client", "qps", kubeConfig.QPS, "burst", kubeConfig.Burst)

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: cfg.Metrics.BindAddress,
		},
		HealthProbeBindAddress:     cfg.Health.BindAddress,
		LeaderElection:             ptr.Deref(cfg.LeaderElection.LeaderElect, false),
		LeaderElectionID:           cfg.LeaderElection.ResourceName,
		LeaderElectionNamespace:    cfg.LeaderElection.ResourceNamespace,
		LeaderElectionResourceLock: cfg.LeaderElection.ResourceLock,
		LeaseDuration:              &cfg.LeaderElection.LeaseDuration.Duration,
		RetryPeriod:                &cfg.LeaderElection.RetryPeriod.Duration,
		RenewDeadline:              &cfg.LeaderElection.RenewDeadline.Duration,
	})
	exitOnError(err, "unable to create manager")

	certsReady := make(chan struct{})
	exitOnError(setupCertManagement(mgr, namespace, certsReady), "unable to setup cert-controller")

	if cfg.KubeRay.IngressDomain == "" {
		configClient, err := clientset.NewForConfig(kubeConfig)
		exitOnError(err, "unable to create Route Client Set")
		cfg.KubeRay.IngressDomain, err = getClusterDomain(ctx, configClient)
		exitOnError(err, cfg.KubeRay.IngressDomain)
	}

	setupLog.Info("setting up health endpoints")
	exitOnError(setupProbeEndpoints(mgr, cfg, certsReady), "unable to set up health check")

	setupLog.Info("setting up RayCluster controller")
	go waitForRayClusterAPIandSetupController(ctx, mgr, cfg, isOpenShift(ctx, kubeClient.DiscoveryClient), certsReady)

	setupLog.Info("starting manager")
	exitOnError(mgr.Start(ctx), "error running manager")
}

func setupRayClusterController(mgr ctrl.Manager, cfg *config.CodeFlareOperatorConfiguration, isOpenShift bool, certsReady chan struct{}) error {
	setupLog.Info("Waiting for certificate generation to complete")
	<-certsReady
	setupLog.Info("Certs ready")

	err := controllers.SetupRayClusterWebhookWithManager(mgr, cfg.KubeRay)
	if err != nil {
		return err
	}

	rayClusterController := controllers.RayClusterReconciler{
		Client:      mgr.GetClient(),
		Scheme:      mgr.GetScheme(),
		Config:      cfg.KubeRay,
		IsOpenShift: isOpenShift,
	}
	return rayClusterController.SetupWithManager(mgr)
}

// +kubebuilder:rbac:groups="apiextensions.k8s.io",resources=customresourcedefinitions,verbs=get;list;watch

func waitForRayClusterAPIandSetupController(ctx context.Context, mgr ctrl.Manager, cfg *config.CodeFlareOperatorConfiguration, isOpenShift bool, certsReady chan struct{}) {
	crdClient, err := apiextensionsclientset.NewForConfig(mgr.GetConfig())
	exitOnError(err, "unable to create CRD client")

	crdList, err := crdClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
	exitOnError(err, "unable to list CRDs")

	if slices.ContainsFunc(crdList.Items, func(crd apiextensionsv1.CustomResourceDefinition) bool {
		return crd.Name == "rayclusters.ray.io"
	}) {
		exitOnError(setupRayClusterController(mgr, cfg, isOpenShift, certsReady), "unable to setup RayCluster controller")
	}

	retryWatcher, err := retrywatch.NewRetryWatcher(crdList.ResourceVersion, &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return crdClient.ApiextensionsV1().CustomResourceDefinitions().List(ctx, metav1.ListOptions{})
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return crdClient.ApiextensionsV1().CustomResourceDefinitions().Watch(ctx, metav1.ListOptions{})
		},
	})
	exitOnError(err, "unable to create retry watcher")

	defer retryWatcher.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-retryWatcher.ResultChan():
			switch event.Type {
			case watch.Error:
				exitOnError(apierrors.FromObject(event.Object), "error watching for RayCluster API")

			case watch.Added, watch.Modified:
				if crd := event.Object.(*apiextensionsv1.CustomResourceDefinition); crd.Name == "rayclusters.ray.io" &&
					slices.ContainsFunc(crd.Status.Conditions, func(condition apiextensionsv1.CustomResourceDefinitionCondition) bool {
						return condition.Type == apiextensionsv1.Established && condition.Status == apiextensionsv1.ConditionTrue
					}) {
					setupLog.Info("RayCluster API installed, setting up controller")
					exitOnError(setupRayClusterController(mgr, cfg, isOpenShift, certsReady), "unable to setup RayCluster controller")
					return
				}
			}
		}
	}
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;update
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;update

func setupCertManagement(mgr ctrl.Manager, namespace string, certsReady chan struct{}) error {
	return cert.AddRotator(mgr, &cert.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: namespace,
			Name:      "codeflare-operator-webhook-server-cert",
		},
		CertDir:        "/tmp/k8s-webhook-server/serving-certs",
		CAName:         "codeflare",
		CAOrganization: "openshift.ai",
		DNSName:        fmt.Sprintf("%s.%s.svc", "codeflare-operator-webhook-service", namespace),
		IsReady:        certsReady,
		Webhooks: []cert.WebhookInfo{
			{
				Type: cert.Validating,
				Name: "codeflare-operator-validating-webhook-configuration",
			},
			{
				Type: cert.Mutating,
				Name: "codeflare-operator-mutating-webhook-configuration",
			},
		},
		// When the controller is running in the leader election mode,
		// we expect webhook server will run in primary and secondary instance
		RequireLeaderElection: false,
	})
}

func setupProbeEndpoints(mgr ctrl.Manager, cfg *config.CodeFlareOperatorConfiguration, certsReady chan struct{}) error {
	err := mgr.AddHealthzCheck(cfg.Health.LivenessEndpointName, healthz.Ping)
	if err != nil {
		return err
	}

	return mgr.AddReadyzCheck(cfg.Health.ReadinessEndpointName, func(req *http.Request) error {
		select {
		case <-certsReady:
			return mgr.GetWebhookServer().StartedChecker()(req)
		default:
			return errors.New("certificates are not ready")
		}
	})
}

func loadIntoOrCreate(ctx context.Context, client kubernetes.Interface, ns, name string, cfg *config.CodeFlareOperatorConfiguration) error {
	configMap, err := client.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return createConfigMap(ctx, client, ns, name, cfg)
	} else if err != nil {
		return err
	}

	if len(configMap.Data) != 1 {
		return fmt.Errorf("cannot resolve config from ConfigMap %s/%s", configMap.Namespace, configMap.Name)
	}

	for _, data := range configMap.Data {
		return yaml.Unmarshal([]byte(data), cfg)
	}

	return nil
}

func createConfigMap(ctx context.Context, client kubernetes.Interface, ns, name string, cfg *config.CodeFlareOperatorConfiguration) error {
	content, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Data: map[string]string{
			"config.yaml": string(content),
		},
	}

	_, err = client.CoreV1().ConfigMaps(ns).Create(ctx, configMap, metav1.CreateOptions{})
	return err
}

func namespaceOrDie() string {
	// This way assumes you've set the NAMESPACE environment variable either manually, when running
	// the operator standalone, or using the downward API, when running the operator in-cluster.
	if ns := os.Getenv("NAMESPACE"); ns != "" {
		return ns
	}

	// Fall back to the namespace associated with the service account token, if available
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		if ns := strings.TrimSpace(string(data)); len(ns) > 0 {
			return ns
		}
	}

	panic("unable to determine current namespace")
}

func exitOnError(err error, msg string) {
	if err != nil {
		setupLog.Error(err, msg)
		os.Exit(1)
	}
}

func isOpenShift(ctx context.Context, dc discovery.DiscoveryInterface) bool {
	logger := ctrl.LoggerFrom(ctx)
	apiGroupList, err := dc.ServerGroups()
	if err != nil {
		logger.Info("Error while querying ServerGroups, assuming we're on Vanilla Kubernetes")
		return false
	}
	for i := 0; i < len(apiGroupList.Groups); i++ {
		if strings.HasSuffix(apiGroupList.Groups[i].Name, ".openshift.io") {
			logger.Info("We detected being on OpenShift!")
			return true
		}
	}
	logger.Info("We detected being on Vanilla Kubernetes!")
	return false
}

func getClusterDomain(ctx context.Context, configClient *clientset.Clientset) (string, error) {
	ingress, err := configClient.ConfigV1().Ingresses().Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get Ingress object: %v", err)
	}

	domain := ingress.Spec.Domain
	if domain == "" {
		return "", fmt.Errorf("domain is not set in the Ingress object")
	}

	return domain, nil
}
