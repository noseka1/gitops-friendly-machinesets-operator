package controllers

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	machineapi "github.com/openshift/api/machine/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestDeleteMachineNow(t *testing.T) {
	assert := assert.New(t)

	mr := NewMachineReconciler(MachineReconcilerConfig{})
	var machine *unstructured.Unstructured

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(metav1.Now())
	assert.Equal(false, mr.deleteMachineNow(logger, machine))

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-50 * time.Second)))
	assert.Equal(false, mr.deleteMachineNow(logger, machine))

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(metav1.NewTime(time.Now().Add(-2 * time.Minute)))
	assert.Equal(true, mr.deleteMachineNow(logger, machine))
}

var _ = Describe("Machine controller", func() {

	Context("When Machine has unresolved tokens", func() {
		It("Should delete the Machine", func() {
			By("Defining a Machine with unresolved tokens")
			machine := &machineapi.Machine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "machine.openshift.io/v1beta1",
					Kind:       "Machine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "machine1",
					Namespace: "openshift-machine-api",
					Annotations: map[string]string{
						"gitops-friendly-machinesets.redhat-cop.io/enabled": "true"},
					Labels: map[string]string{
						"machine.openshift.io/cluster-api-cluster": "INFRANAME",
					},
				},
				Spec: machineapi.MachineSpec{},
			}
			By("Creating a Machine with unresolved tokens in Kubernetes")
			err := k8sClient.Create(ctx, machine, &client.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				return k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machine.GetNamespace(), Name: machine.GetName()},
					machine)
			}).ShouldNot(HaveOccurred())
			By("Waiting until the Machine has been deleted")
			Eventually(func() bool {
				err := k8sClient.Get(ctx,
					types.NamespacedName{Namespace: machine.GetNamespace(), Name: machine.GetName()},
					machine)
				return apierrors.IsNotFound(err)
			}).Should(BeTrue())
		})
	})

})
