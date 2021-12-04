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

package controllers

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	machineapi "github.com/openshift/api/machine/v1beta1"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	ctxCancel context.CancelFunc
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func() {

	const (
		apiTimeout  = 5 * time.Second
		apiDuration = 5 * time.Second
		apiInterval = 250 * time.Millisecond
	)

	SetDefaultConsistentlyDuration(apiDuration)
	SetDefaultConsistentlyPollingInterval(apiInterval)
	SetDefaultEventuallyTimeout(apiTimeout)
	SetDefaultEventuallyPollingInterval(apiInterval)

	ctx, ctxCancel = context.WithCancel(context.TODO())

	logf.SetLogger(zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(true),
		zap.Level(zapcore.Level(-10))))

	By("Bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join("..", "test", "crds")},
		ErrorIfCRDPathMissing:    false,
		AttachControlPlaneOutput: false,
	}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	By("Creating a scheme")
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).ToNot(HaveOccurred())
	err = machineapi.Install(scheme)
	Expect(err).ToNot(HaveOccurred())

	//+kubebuilder:scaffold:scheme

	By("Creating a Kubernetes client")
	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	By("Creating an openshift-machine-api namespace")
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "openshift-machine-api",
		},
	}
	err = k8sClient.Create(ctx, ns, &client.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		return k8sClient.Get(ctx, types.NamespacedName{Name: ns.GetName()}, ns)
	}).ShouldNot(HaveOccurred())

	By("Creating a controller manager")
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	controllerName := "gitops-friendly-machinesets"

	err = (&MachineSetReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		EventRecorder:      mgr.GetEventRecorderFor(controllerName),
		InfrastructureName: "cluster-test-xyz",
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (NewMachineReconciler(MachineReconcilerConfig{
		Client:        mgr.GetClient(),
		Scheme:        mgr.GetScheme(),
		EventRecorder: mgr.GetEventRecorderFor(controllerName),
	},
		func(mr *machineReconciler) {
			mr.DeleteMachineMinAgeSeconds = -1
		})).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	By("Starting the manager")
	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "Failed to run manager")
	}()
}, 60)

var _ = AfterSuite(func() {
	By("Stopping the controller manager")
	ctxCancel()
	By("Tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
