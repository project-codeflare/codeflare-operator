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
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mcadv1beta2 "github.com/project-codeflare/appwrapper/api/v1beta2"
	awconfig "github.com/project-codeflare/appwrapper/pkg/config"
	awctrl "github.com/project-codeflare/appwrapper/pkg/controller"
	rayv1 "github.com/ray-project/kuberay/ray-operator/apis/ray/v1"
	"go.uber.org/zap/zapcore"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/klog/v2"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	kueue "sigs.k8s.io/kueue/apis/kueue/v1beta1"
	"sigs.k8s.io/yaml"

	routev1 "github.com/openshift/api/route/v1"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
	"github.com/project-codeflare/codeflare-operator/pkg/controllers"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	// +kubebuilder:scaffold:imports
)

var (
	scheme            = runtime.NewScheme()
	setupLog          = ctrl.Log.WithName("setup")
	OperatorVersion   = "UNKNOWN"
	AppWrapperVersion = "UNKNOWN"
	BuildDate         = "UNKNOWN"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Ray
	utilruntime.Must(rayv1.AddToScheme(scheme))
	// OpenShift Route
	utilruntime.Must(routev1.Install(scheme))
	// AppWrapper
	utilruntime.Must(mcadv1beta2.AddToScheme(scheme))
	utilruntime.Must(kueue.AddToScheme(scheme))
}

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
		"appwrapperVersion", AppWrapperVersion,
		"date", BuildDate,
	)

	ctx := ctrl.SetupSignalHandler()

	namespace := namespaceOrDie()
	cfg := &config.CodeFlareOperatorConfiguration{
		ClientConnection: &config.ClientConnection{
			QPS:   ptr.To(float32(50)),
			Burst: ptr.To(int32(100)),
		},
		AppWrapper: &config.AppWrapperConfiguration{
			Enabled: ptr.To(true),
			Config:  awconfig.NewConfig(namespace),
		},
		ControllerManager: config.ControllerManager{
			Metrics: config.MetricsConfiguration{
				BindAddress:   ":8080",
				SecureServing: true,
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
		},
	}
	cfg.AppWrapper.Config.CertManagement.WebhookSecretName = "codeflare-operator-webhook-server-cert"
	cfg.AppWrapper.Config.CertManagement.WebhookServiceName = "codeflare-operator-webhook-service"
	cfg.AppWrapper.Config.CertManagement.MutatingWebhookConfigName = "codeflare-operator-mutating-webhook-configuration"
	cfg.AppWrapper.Config.CertManagement.ValidatingWebhookConfigName = "codeflare-operator-validating-webhook-configuration"

	kubeConfig, err := ctrl.GetConfig()
	exitOnError(err, "unable to get client config")
	if kubeConfig.UserAgent == "" {
		kubeConfig.UserAgent = "codeflare-operator"
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	exitOnError(err, "unable to create Kubernetes client")

	exitOnError(loadIntoOrCreate(ctx, kubeClient, namespaceOrDie(), configMapName, cfg), "unable to initialise configuration")
	exitOnError(awconfig.ValidateConfig(cfg.AppWrapper.Config), "invalid AppWrapper configuration")

	kubeConfig.Burst = int(ptr.Deref(cfg.ClientConnection.Burst, int32(rest.DefaultBurst)))
	kubeConfig.QPS = ptr.Deref(cfg.ClientConnection.QPS, rest.DefaultQPS)
	setupLog.V(2).Info("REST client", "qps", kubeConfig.QPS, "burst", kubeConfig.Burst)

	// if the cfg.EnableHTTP2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancelation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}
	tlsOpts := []func(*tls.Config){}
	if !cfg.Metrics.EnableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   cfg.Metrics.BindAddress,
			SecureServing: cfg.Metrics.SecureServing,
			TLSOpts:       tlsOpts,
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
	exitOnError(err, "unable to start manager")

	certsReady := make(chan struct{})
	if os.Getenv("ENABLE_WEBHOOKS") == "false" {
		close(certsReady)
	} else {
		exitOnError(awctrl.SetupCertManagement(mgr, &cfg.AppWrapper.Config.CertManagement, certsReady), "unable to set up cert rotation")
	}

	v, err := HasAPIResourceForGVK(kubeClient.DiscoveryClient, rayv1.GroupVersion.WithKind("RayCluster"))
	if v && *cfg.KubeRay.RayDashboardOAuthEnabled {
		rayClusterController := controllers.RayClusterReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
		exitOnError(rayClusterController.SetupWithManager(mgr), "Error setting up RayCluster controller")
	} else if err != nil {
		exitOnError(err, "Could not determine if RayCluster CR present on cluster.")
	}

	v1, err1 := HasAPIResourceForGVK(kubeClient.DiscoveryClient, kueue.GroupVersion.WithKind("Workload"))
	v2, err2 := HasAPIResourceForGVK(kubeClient.DiscoveryClient, mcadv1beta2.GroupVersion.WithKind("AppWrapper"))
	if v1 && v2 && *cfg.AppWrapper.Enabled {
		// Ascynchronous because controllers need to wait for certificate to be ready for webhooks to work
		go awctrl.SetupControllers(ctx, mgr, cfg.AppWrapper.Config, certsReady, setupLog)
		exitOnError(awctrl.SetupIndexers(ctx, mgr, cfg.AppWrapper.Config), "unable to setup indexers")
	} else if err1 != nil || err2 != nil {
		exitOnError(err, "Could not determine if AppWrapper and Workload CRDs present on cluster.")
	} else {
		setupLog.Info("AppWrapper controller disabled", "Workload CRD present", v1,
			"AppWrapper CRD present", v2, "Config flag value", *cfg.AppWrapper.Enabled)
	}

	exitOnError(awctrl.SetupProbeEndpoints(mgr, certsReady), "unable to setup probe endpoints")

	setupLog.Info("starting manager")
	exitOnError(mgr.Start(ctx), "error running manager")
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

func HasAPIResourceForGVK(dc discovery.DiscoveryInterface, gvk schema.GroupVersionKind) (bool, error) {
	gv, kind := gvk.ToAPIVersionAndKind()
	if resources, err := dc.ServerResourcesForGroupVersion(gv); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	} else {
		for _, res := range resources.APIResources {
			if res.Kind == kind {
				return true, nil
			}
		}
	}
	return false, nil
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
