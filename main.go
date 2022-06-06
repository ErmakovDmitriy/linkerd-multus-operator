/*
Copyright 2022.

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
	"errors"
	"flag"
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	podwebhook "github.com/ErmakovDmitriy/linkerd-cni-attach-operator/api/v1"
	cniv1alpha1 "github.com/ErmakovDmitriy/linkerd-cni-attach-operator/api/v1alpha1"

	netattachv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"

	"github.com/ErmakovDmitriy/linkerd-cni-attach-operator/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	EnvVarPrefix = "LINKERD_CNI_ATTACH_OPERATOR_"

	EnvInstanceName          = EnvVarPrefix + "INSTANCE"
	EnvCNIConfigMapNamespace = EnvVarPrefix + "CNI_CM_NAMESPACE"
	EnvCNIConfigMapName      = EnvVarPrefix + "CNI_CM_NAME"
	EnvCNIConfigMapKey       = EnvVarPrefix + "CNI_CM_KEY"
	EnvCNIKubeconfig         = EnvVarPrefix + "KUBECONFIG"
)

const (
	DefaultInstanceName = "default"

	DefaultLinkerdCNICMNamespace = "linkerd-cni"
	DefaultLinkerdCNICMName      = "linkerd-cni-config"
	DefaultLinkerdCNICMKey       = "cni_network_config"

	// DefaultLinkerdCNIKubeconfigName - used the name which is created by
	// Linkerd-CNI DaemonSet.
	DefaultLinkerdCNIKubeconfigPath = "/etc/cni/net.d/ZZZ-linkerd-cni-kubeconfig"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(cniv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme

	utilruntime.Must(netattachv1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Configure environment.
	var configMapNamespace = os.Getenv(EnvCNIConfigMapNamespace)
	if configMapNamespace == "" {
		configMapNamespace = DefaultLinkerdCNICMNamespace
	}

	var configMapName = os.Getenv(EnvCNIConfigMapName)
	if configMapName == "" {
		configMapName = DefaultLinkerdCNICMName
	}

	var configMapKey = os.Getenv(EnvCNIConfigMapKey)
	if configMapKey == "" {
		configMapKey = DefaultLinkerdCNICMKey
	}

	var instanceName = os.Getenv(EnvInstanceName)
	if instanceName == "" {
		instanceName = DefaultInstanceName
	}

	var cniKubeconfig = os.Getenv(EnvCNIKubeconfig)
	if cniKubeconfig == "" {
		cniKubeconfig = DefaultLinkerdCNIKubeconfigPath
	}

	// Check that instance name is not too long.
	const maxInstanceNameLen = 30
	if len(instanceName) > maxInstanceNameLen {
		setupLog.Error(
			errors.New("instance-name is too long"),
			fmt.Sprintf("instance name must be no longer than %d symbols", len(instanceName)),
			"current_length", len(instanceName),
		)
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "cni-attach-operator.linkerd.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Linkerd AttachDefinition controller.
	var attachReconciler = &controllers.AttachDefinitionReconciler{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		InstanceName:  instanceName,
		CNIKubeconfig: cniKubeconfig,
		CNIConfigMapRef: controllers.CNIConfigMapRef{
			ObjectKey: types.NamespacedName{
				Namespace: configMapNamespace,
				Name:      configMapName,
			},
			Key: configMapKey,
		},
	}

	if err = attachReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AttachDefinition")
		os.Exit(1)
	}

	// Multus NetAttachDefinition controller.
	if err = (&controllers.MultusNetAttachDefinitionReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		InstanceName:           instanceName,
		AttachReconcileTrigger: attachReconciler.Reconcile,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MultusNetAttachDefinitionReconciler")
		os.Exit(1)
	}

	// Create mutating webhook.
	mgr.GetWebhookServer().Register("/annotate-v1-pod", &webhook.Admission{
		Handler: &podwebhook.PodAnnotator{
			Client: mgr.GetClient(),
		},
	})

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
