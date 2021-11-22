/*
Copyright 2021 Ales Nosek.

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
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	configapi "github.com/openshift/api/config/v1"
	machineapi "github.com/openshift/api/machine/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/noseka1/gitops-friendly-machinesets-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	controllerName = "gitops-friendly-machinesets"
)

func init() {
	utilruntime.Must(machineapi.Install(scheme))

	//+kubebuilder:scaffold:scheme
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

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "123eec1d.openshift.io",
	})
	if err != nil {
		setupLog.Error(err, "Unable to start manager")
		os.Exit(1)
	}

	restConfig := mgr.GetConfig()

	infrastructureName := retrieveInfrastructureName(restConfig)

	machineSetResource := retrieveMachineApiResourceForKind(restConfig, "MachineSet")
	machineSetInterface := retrieveMachineApiInterfaceForResource(restConfig, machineSetResource)
	machineResource := retrieveMachineApiResourceForKind(restConfig, "Machine")
	machineInterface := retrieveMachineApiInterfaceForResource(restConfig, machineResource)

	if err = (&controllers.MachineSetReconciler{
		Client:              mgr.GetClient(),
		Scheme:              mgr.GetScheme(),
		ControllerName:      controllerName,
		EventRecorder:       mgr.GetEventRecorderFor(controllerName),
		InfrastructureName:  infrastructureName,
		MachineSetInterface: machineSetInterface,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "MachineSet")
		os.Exit(1)
	}
	if err = (&controllers.MachineReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ControllerName:   controllerName,
		EventRecorder:    mgr.GetEventRecorderFor(controllerName),
		MachineInterface: machineInterface,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "Machine")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

// Retrieve unique infrastructure name of this OpenShift cluster (something like mycluster-jfnx7).
// The code performs an equivalent of: oc get infrastructure cluster -o jsonpath='{.status.infrastructureName}'
func retrieveInfrastructureName(clientConfig *rest.Config) string {

	configScheme := runtime.NewScheme()
	utilruntime.Must(configapi.Install(configScheme))

	kubeClient, err := client.New(clientConfig, client.Options{Scheme: configScheme})
	if err != nil {
		setupLog.Error(err, "Failed to create kube client")
		os.Exit(1)
	}

	infraObjectName := client.ObjectKey{
		Namespace: "",
		Name:      "cluster",
	}
	infraObject := &configapi.Infrastructure{}

	if err = kubeClient.Get(context.TODO(), infraObjectName, infraObject); err != nil {
		setupLog.Error(err, "Unable retrieve object "+infraObjectName.String()+" of kind Infrastructure")
		os.Exit(1)
	}
	infraName := infraObject.Status.InfrastructureName

	if infraName == "" {
		setupLog.Error(err, "Infrastructure.status.infrastructureName must not be empty")
		os.Exit(1)
	}

	setupLog.Info("Infrastructure name is " + infraName)

	return infraName
}

func retrieveMachineApiResourceForKind(clientConfig *rest.Config, kind string) string {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(clientConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create a discovery client")
		os.Exit(1)
	}

	machineSetGv := schema.GroupVersion{
		Group:   machineapi.SchemeGroupVersion.Group,
		Version: machineapi.SchemeGroupVersion.Version,
	}

	resourceList, err := discoveryClient.ServerResourcesForGroupVersion(machineSetGv.String())
	if err != nil {
		setupLog.Error(err, "Unable to retrieve resouce list for "+machineSetGv.String())
		os.Exit(1)
	}

	var resource string
	for _, res := range resourceList.APIResources {
		if res.Kind == kind {
			resource = res.Name
			break
		}
	}

	if resource == "" {
		setupLog.Info("Cannot find resource for kind " + kind)
		os.Exit(1)
	}

	return resource
}

func retrieveMachineApiInterfaceForResource(clientConfig *rest.Config, resource string) dynamic.NamespaceableResourceInterface {

	dynamicClient, err := dynamic.NewForConfig(clientConfig)
	if err != nil {
		setupLog.Error(err, "Unable to create a dynamic client")
		os.Exit(1)
	}

	machineSetGvr := schema.GroupVersionResource{
		Group:    machineapi.SchemeGroupVersion.Group,
		Version:  machineapi.SchemeGroupVersion.Version,
		Resource: resource,
	}

	return dynamicClient.Resource(machineSetGvr)
}
