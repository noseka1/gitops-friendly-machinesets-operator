package webhooks

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	machineapi "github.com/openshift/api/machine/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("MachineSet controller", func() {

	Context("When MachineSet has unresolved tokens", func() {
		It("Should resolve the tokens", func() {
			By("Defining a MachineSet with unresolved tokens")
			machineSet := &machineapi.MachineSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machineset1",
					Namespace: "openshift-machine-api",
					Annotations: map[string]string{
						"gitops-friendly-machinesets.redhat-cop.io/enabled": "true"},
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-cluster": "INFRANAME",
					},
				},
			}
			By("Creating a MachineSet with unresolved tokens in Kubernetes")
			err := k8sClient.Create(ctx, machineSet, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
			}).ShouldNot(HaveOccurred())
			By("Checking that tokens have been resolved")
			value, found := machineSet.GetLabels()["machine.openshift.io/cluster-api-cluster"]
			Expect(found).To(BeTrue())
			Expect(value).To(Equal("cluster-test-xyz"))
		})
	})

	Context("When MachineSet has unresolved tokens but reconciliation is disabled", func() {
		It("Should NOT resolve the tokens", func() {
			By("Defining a MachineSet with unresolved tokens and reconciliation disabled")
			machineSet := &machineapi.MachineSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machineset2",
					Namespace: "openshift-machine-api",
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-cluster": "INFRANAME",
					},
				},
			}
			By("Creating a MachineSet with unresolved tokens and reconciliation disabled in Kubernetes")
			err := k8sClient.Create(ctx, machineSet, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
			}).ShouldNot(HaveOccurred())
			By("Checking that tokens have NOT been resolved")
			value, found := machineSet.GetLabels()["machine.openshift.io/cluster-api-cluster"]
			Expect(found).To(BeTrue())
			Expect(value).To(Equal("INFRANAME"))
		})
	})
})
