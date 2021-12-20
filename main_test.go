package main

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	configapi "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Main", func() {

	Context("When Infrastructure object does NOT exist", func() {
		It("Should return an empty infrastructure name", func() {
			Expect(retrieveInfrastructureName(clientConfig)).To(BeEmpty())
		})
	})

	Context("When Infrastructure object does exist", func() {
		It("Should return the infrastructure name", func() {
			By("Defining an infrastructure object")
			infrastructure := &configapi.Infrastructure{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "config.openshift.io/v1",
					Kind:       "Infrastructure",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "cluster",
				},
			}
			By("Creating the infrastructure object in Kubernetes")
			err := k8sClient.Create(ctx, infrastructure, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: "", Name: infrastructure.GetName()},
					infrastructure)
			}).ShouldNot(HaveOccurred())
			By("Setting infrastructure name in Kubernetes")
			infrastructure.Status.ControlPlaneTopology = "SingleReplica"
			infrastructure.Status.InfrastructureTopology = "SingleReplica"
			infrastructure.Status.InfrastructureName = "cluster-test-xyz"
			err = k8sClient.Status().Update(ctx, infrastructure, &client.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() string {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: "", Name: infrastructure.GetName()},
					infrastructure)
				Expect(err).ToNot(HaveOccurred())
				return infrastructure.Status.InfrastructureName
			}).Should(Equal("cluster-test-xyz"))
			By("Checking that infrastructure name is retrieved correctly")
			Expect(retrieveInfrastructureName(clientConfig)).To(Equal("cluster-test-xyz"))
		})
	})
})
