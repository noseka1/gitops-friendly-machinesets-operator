package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configapi "github.com/openshift/api/config/v1"
	"go.uber.org/zap/zapcore"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
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
	clientConfig *rest.Config
	k8sClient    client.Client
	testEnv      *envtest.Environment
	ctx          context.Context
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Main Suite",
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

	ctx = context.TODO()

	logf.SetLogger(zap.New(
		zap.WriteTo(GinkgoWriter),
		zap.UseDevMode(true),
		zap.Level(zapcore.Level(-10))))

	By("Bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{
			filepath.Join("..", "config", "crd", "bases"),
			filepath.Join(".", "test", "crds")},
		ErrorIfCRDPathMissing:    false,
		AttachControlPlaneOutput: false,
	}
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	clientConfig = cfg

	By("Creating a scheme")
	scheme := runtime.NewScheme()
	err = clientgoscheme.AddToScheme(scheme)
	Expect(err).ToNot(HaveOccurred())
	err = configapi.Install(scheme)
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
}, 60)

var _ = AfterSuite(func() {
	By("Tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
