package controllers

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wI2L/jsondiff"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestCreatePatch(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME", "metadata", "labels", "machine.openshift.io/cluster-api-cluster")
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME", "spec", "selector", "matchLabels", "machine.openshift.io/cluster-api-cluster")
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME-worker-us-east-2c", "spec", "selector", "matchLabels", "machine.openshift.io/cluster-api-machineset")

	patchBytes, err := createPatch(logger, machineSet, "INFRANAME", "MYCLUSTER")
	assert.Equal(nil, err)

	patch := jsondiff.Patch{}
	err = json.Unmarshal(patchBytes, &patch)

	expectedPatch := jsondiff.Patch{jsondiff.Operation{Type: "replace", Path: "/metadata/labels/machine.openshift.io~1cluster-api-cluster", Value: "MYCLUSTER"}, jsondiff.Operation{Type: "replace", Path: "/spec/selector/matchLabels/machine.openshift.io~1cluster-api-cluster", Value: "MYCLUSTER"}, jsondiff.Operation{Type: "replace", Path: "/spec/selector/matchLabels/machine.openshift.io~1cluster-api-machineset", Value: "MYCLUSTER-worker-us-east-2c"}}
	assert.Equal(nil, err)
	assert.Equal(3, len(patch))
	assert.Equal(expectedPatch, patch)
}
