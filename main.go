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
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	instascale "github.com/project-codeflare/instascale/controllers"
	instascaleconfig "github.com/project-codeflare/instascale/pkg/config"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	quotasubtreev1alpha1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/quotaplugins/quotasubtree/v1alpha1"
	mcadconfig "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/config"
	mcad "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/controller/queuejob"
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
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/yaml"

	configv1 "github.com/openshift/api/config/v1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
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
	McadVersion       = "UNKNOWN"
	InstaScaleVersion = "UNKNOWN"
	BuildDate         = "UNKNOWN"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// MCAD
	utilruntime.Must(mcadv1beta1.AddToScheme(scheme))
	utilruntime.Must(quotasubtreev1alpha1.AddToScheme(scheme))
	// InstaScale
	utilruntime.Must(configv1.Install(scheme))
	utilruntime.Must(machinev1beta1.Install(scheme))
	// Ray
	utilruntime.Must(rayv1.AddToScheme(scheme))
	// OpenShift Route
	utilruntime.Must(routev1.Install(scheme))
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
		"mcadVersion", McadVersion,
		"instaScaleVersion", InstaScaleVersion,
		"date", BuildDate,
	)

	ctx := ctrl.SetupSignalHandler()

	cfg := &config.CodeFlareOperatorConfiguration{
		ClientConnection: &config.ClientConnection{
			QPS:   pointer.Float32(50),
			Burst: pointer.Int32(100),
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
		MCAD: &mcadconfig.MCADConfiguration{},
		InstaScale: &config.InstaScaleConfiguration{
			Enabled: pointer.Bool(false),
			InstaScaleConfiguration: instascaleconfig.InstaScaleConfiguration{
				MaxScaleoutAllowed: 5,
			},
		},
		KubeRay: &config.KubeRayConfiguration{
			RayDashboardOAuthEnabled: pointer.Bool(true),
		},
	}

	kubeConfig, err := ctrl.GetConfig()
	exitOnError(err, "unable to get client config")
	if kubeConfig.UserAgent == "" {
		kubeConfig.UserAgent = "codeflare-operator"
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	exitOnError(err, "unable to create Kubernetes client")

	exitOnError(loadIntoOrCreate(ctx, kubeClient, namespaceOrDie(), configMapName, cfg), "unable to initialise configuration")

	kubeConfig.Burst = int(pointer.Int32Deref(cfg.ClientConnection.Burst, int32(rest.DefaultBurst)))
	kubeConfig.QPS = pointer.Float32Deref(cfg.ClientConnection.QPS, rest.DefaultQPS)
	setupLog.V(2).Info("REST client", "qps", kubeConfig.QPS, "burst", kubeConfig.Burst)

	mgr, err := ctrl.NewManager(kubeConfig, ctrl.Options{
		Scheme:                     scheme,
		MetricsBindAddress:         cfg.Metrics.BindAddress,
		HealthProbeBindAddress:     cfg.Health.BindAddress,
		LeaderElection:             pointer.BoolDeref(cfg.LeaderElection.LeaderElect, false),
		LeaderElectionID:           cfg.LeaderElection.ResourceName,
		LeaderElectionNamespace:    cfg.LeaderElection.ResourceNamespace,
		LeaderElectionResourceLock: cfg.LeaderElection.ResourceLock,
		LeaseDuration:              &cfg.LeaderElection.LeaseDuration.Duration,
		RetryPeriod:                &cfg.LeaderElection.RetryPeriod.Duration,
		RenewDeadline:              &cfg.LeaderElection.RenewDeadline.Duration,
	})
	exitOnError(err, "unable to start manager")

	mcadQueueController := mcad.NewJobController(mgr.GetConfig(), cfg.MCAD, &mcadconfig.MCADConfigurationExtended{})
	if mcadQueueController == nil {
		// FIXME: update NewJobController so it follows Go idiomatic error handling and return an error instead of a nil object
		os.Exit(1)
	}
	mcadQueueController.Run(ctx.Done())

	if pointer.BoolDeref(cfg.InstaScale.Enabled, false) {
		instaScaleController := &instascale.AppWrapperReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
			Config: cfg.InstaScale.InstaScaleConfiguration,
		}
		exitOnError(instaScaleController.SetupWithManager(context.Background(), mgr), "Error setting up InstaScale controller")
	}

	v, err := HasAPIResourceForGVK(kubeClient.DiscoveryClient, rayv1.GroupVersion.WithKind("RayCluster"))
	if v && *cfg.KubeRay.RayDashboardOAuthEnabled {
		rayClusterController := controllers.RayClusterReconciler{Client: mgr.GetClient(), Scheme: mgr.GetScheme()}
		exitOnError(rayClusterController.SetupWithManager(mgr), "Error setting up RayCluster controller")
	} else if err != nil {
		exitOnError(err, "Could not determine if RayCluster CR present on cluster.")
	}

	exitOnError(mgr.AddHealthzCheck(cfg.Health.LivenessEndpointName, healthz.Ping), "unable to set up health check")
	exitOnError(mgr.AddReadyzCheck(cfg.Health.ReadinessEndpointName, healthz.Ping), "unable to set up ready check")

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
