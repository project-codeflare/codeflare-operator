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
	mcadoptions "github.com/project-codeflare/multi-cluster-app-dispatcher/cmd/kar-controllers/app/options"
	mcadv1beta1 "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/apis/controller/v1beta1"
	mcad "github.com/project-codeflare/multi-cluster-app-dispatcher/pkg/controller/queuejob"
	"go.uber.org/zap/zapcore"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
	// templatesPath = "config/internal/"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	_ = mcadv1beta1.AddToScheme(scheme)
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var configsNamespace string
	var ocmSecretNamespace string

	// Operator
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// InstScale
	flag.StringVar(&configsNamespace, "configs-namespace", "kube-system", "The namespace containing the Instacale configmap")
	flag.StringVar(&ocmSecretNamespace, "ocm-secret-namespace", "default", "The namespace containing the OCM secret")

	mcadOptions := mcadoptions.NewServerOption()

	zapOptions := zap.Options{
		Development: true,
		TimeEncoder: zapcore.TimeEncoderOfLayout(time.RFC3339),
	}

	flagSet := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flag.CommandLine.VisitAll(func(f *flag.Flag) {
		if f.Name != "kubeconfig" {
			flagSet.Var(f.Value, f.Name, f.Usage)
		}
	})

	zapOptions.BindFlags(flagSet)
	mcadOptions.AddFlags(flagSet)
	_ = flagSet.Parse(os.Args[1:])

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zapOptions)))

	ctx := ctrl.SetupSignalHandler()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5a3ca514.codeflare.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	mcadQueueController := mcad.NewJobController(mgr.GetConfig(), mcadOptions)
	if mcadQueueController == nil {
		// FIXME: update NewJobController so it follows Go idiomatic error handling and return an error instead of a nil object
		os.Exit(1)
	}
	mcadQueueController.Run(ctx.Done())

	instascaleController := &instascale.AppWrapperReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		ConfigsNamespace:   configsNamespace,
		OcmSecretNamespace: ocmSecretNamespace,
	}

	exitOnError(instascaleController.SetupWithManager(mgr), "Error setting up InstaScale controller")

	// if err = (&controllers.MCADReconciler{
	// 	Client:        mgr.GetClient(),
	// 	Scheme:        mgr.GetScheme(),
	// 	Log:           ctrl.Log,
	// 	TemplatesPath: templatesPath,
	// }).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "MCAD")
	// 	os.Exit(1)
	// }
	// if err = (&controllers.InstaScaleReconciler{
	// 	Client:        mgr.GetClient(),
	// 	Scheme:        mgr.GetScheme(),
	// 	Log:           ctrl.Log,
	// 	TemplatesPath: templatesPath,
	// }).SetupWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create controller", "controller", "InstaScale")
	// 	os.Exit(1)
	// }
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func exitOnError(err error, msg string) {
	if err != nil {
		setupLog.Error(err, msg)
		os.Exit(1)
	}
}
