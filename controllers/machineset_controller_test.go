package controllers

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	machineapi "github.com/openshift/api/machine/v1beta1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestIsWorkerMachineSet(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{}
	assert.Equal(false, isWorkerMachineSet(machineSet))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "worker", "spec", "template", "metadata", "labels", "machine.openshift.io/cluster-api-machine-role")
	assert.Equal(true, isWorkerMachineSet(machineSet))
}

func TestHasNodesAvailable(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{}
	assert.Equal(false, hasNodesAvailable(machineSet))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), int64(0), "status", "availableReplicas")
	assert.Equal(false, hasNodesAvailable(machineSet))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), int64(1), "status", "availableReplicas")

	bytes, _ := json.Marshal(machineSet)
	logger.Info(string(bytes))
	assert.Equal(true, hasNodesAvailable(machineSet))
}

func TestNameStartsWith(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{}
	assert.Equal(false, nameStartsWith(machineSet, "mycluster"))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "mycluster", "metadata", "name")
	assert.Equal(false, nameStartsWith(machineSet, "myprefix"))
	assert.Equal(true, nameStartsWith(machineSet, "my"))
	assert.Equal(true, nameStartsWith(machineSet, "mycluster"))
}

func TestIsReplicasGreaterThanZero(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{}
	assert.Equal(false, isReplicasGreaterThanZero(machineSet))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), int64(0), "spec", "replicas")
	assert.Equal(false, isReplicasGreaterThanZero(machineSet))

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), int64(1), "spec", "replicas")
	assert.Equal(true, isReplicasGreaterThanZero(machineSet))
}

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
			By("Waiting until the tokens have been resolved")
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
				Expect(err).NotTo(HaveOccurred())
				value, found := machineSet.GetLabels()["machine.openshift.io/cluster-api-cluster"]
				return found && (value == "cluster-test-xyz")
			}).Should(BeTrue())
		})
	})

	Context("When MachineSet has unresolved tokens but reconciliation is disabled", func() {
		It("Should not resolve the tokens", func() {
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
			By("Checking that MachineSet is not deleted")
			Consistently(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
			}).ShouldNot(HaveOccurred())
		})
	})

	Context("When nodes of managed MachineSet become available", func() {
		It("Should scale the installer-provisioned MachineSets down to zero", func() {
			By("Defining an installer-provisioned MachineSet")
			var installerMachineSetReplicas int32 = 3
			installerMachineSet := &machineapi.MachineSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cluster-test-xyz-az1",
					Namespace: "openshift-machine-api",
				},
				Spec: machineapi.MachineSetSpec{
					Template: machineapi.MachineTemplateSpec{
						ObjectMeta: machineapi.ObjectMeta{
							Labels: map[string]string{
								"machine.openshift.io/cluster-api-machine-role": "worker",
							},
						},
					},
					Replicas: &installerMachineSetReplicas,
				},
			}
			By("Creating an installer-provisioned MachineSet")
			err := k8sClient.Create(ctx, installerMachineSet, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: installerMachineSet.GetNamespace(), Name: installerMachineSet.GetName()},
					installerMachineSet)
			}).ShouldNot(HaveOccurred())
			By("Checking that the installer-provisioned MachineSet has 3 replicas")
			Consistently(func() int {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: installerMachineSet.GetNamespace(), Name: installerMachineSet.GetName()},
					installerMachineSet)
				Expect(err).ToNot(HaveOccurred())
				return int(*installerMachineSet.Spec.Replicas)
			}).Should(Equal(3))
			By("Defining a managed MachineSet with available nodes")
			machineSet := &machineapi.MachineSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "MachineSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machineset3",
					Namespace: "openshift-machine-api",
					Annotations: map[string]string{
						"gitops-friendly-machinesets.redhat-cop.io/enabled": "true"},
				},
				Spec: machineapi.MachineSetSpec{
					Template: machineapi.MachineTemplateSpec{
						ObjectMeta: machineapi.ObjectMeta{
							Labels: map[string]string{
								"machine.openshift.io/cluster-api-machine-role": "worker",
							},
						},
					},
				},
			}
			By("Creating a managed MachineSet")
			err = k8sClient.Create(ctx, machineSet, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
			}).ShouldNot(HaveOccurred())
			By("Setting the available nodes in MachineSet")
			machineSet.Status.AvailableReplicas = 1
			err = k8sClient.Status().Update(ctx, machineSet, &client.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() int {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machineSet.GetNamespace(), Name: machineSet.GetName()},
					machineSet)
				Expect(err).ToNot(HaveOccurred())
				return int(machineSet.Status.AvailableReplicas)
			}).Should(Equal(1))
			By("Checking that installer-provisioned MachineSet has been scaled down to zero")
			Eventually(func() int {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: installerMachineSet.GetNamespace(), Name: installerMachineSet.GetName()},
					installerMachineSet)
				Expect(err).ToNot(HaveOccurred())
				return int(*installerMachineSet.Spec.Replicas)
			}).Should(Equal(0))
		})
	})
})
