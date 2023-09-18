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
	"flag"
	"os"
	"time"

	instascale "github.com/project-codeflare/instascale/controllers"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	quotasubtreev1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/quotaplugins/quotasubtree/v1"
	mcadconfig "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/config"
	mcad "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/controller/queuejob"
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	configv1alpha1 "k8s.io/component-base/config/v1alpha1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/project-codeflare/codeflare-operator/pkg/config"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(mcadv1beta1.AddToScheme(scheme))
	utilruntime.Must(quotasubtreev1.AddToScheme(scheme))
}

func main() {
	zapOptions := zap.Options{
		Development: true,
		TimeEncoder: zapcore.TimeEncoderOfLayout(time.RFC3339),
	}
	zapOptions.BindFlags(flag.CommandLine)

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	ctx := ctrl.SetupSignalHandler()

	cfg := config.CodeFlareOperatorConfiguration{
		ControllerManager: config.ControllerManager{
			LeaderElection: &configv1alpha1.LeaderElectionConfiguration{},
		},
		MCAD:       &mcadconfig.MCADConfiguration{},
		InstaScale: &config.InstaScaleConfiguration{},
	}

	kubeConfig, err := ctrl.GetConfig()
	exitOnError(err, "unable to get client config")

	if kubeConfig.UserAgent == "" {
		kubeConfig.UserAgent = "codeflare-operator"
	}
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
		exitOnError(instaScaleController.SetupWithManager(mgr), "Error setting up InstaScale controller")
	}

	exitOnError(mgr.AddHealthzCheck(cfg.Health.LivenessEndpointName, healthz.Ping), "unable to set up health check")
	exitOnError(mgr.AddReadyzCheck(cfg.Health.ReadinessEndpointName, healthz.Ping), "unable to set up ready check")

	setupLog.Info("starting manager")
	exitOnError(mgr.Start(ctx), "error running manager")
}

func exitOnError(err error, msg string) {
	if err != nil {
		setupLog.Error(err, msg)
		os.Exit(1)
	}
}
