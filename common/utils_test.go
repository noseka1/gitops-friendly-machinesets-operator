package common

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	logger logr.Logger
)

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func setup() {
	logger = zap.New(zap.Level(zapcore.Level(-10)))
}
func TestIsObjectReconciliationEnabled(t *testing.T) {
	assert := assert.New(t)

	var input *unstructured.Unstructured

	input = &unstructured.Unstructured{}
	assert.Equal(false, IsObjectReconciliationEnabled(input))

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "false",
	})
	assert.Equal(false, IsObjectReconciliationEnabled(input))

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "true",
	})
	assert.Equal(true, IsObjectReconciliationEnabled(input))
}
func TestEvaluateAnnotations(t *testing.T) {
	assert := assert.New(t)

	var input *unstructured.Unstructured
	var enabled bool
	var tokenName string

	input = &unstructured.Unstructured{}
	enabled, tokenName = EvaluateAnnotations(logger, input)
	assert.Equal(false, enabled)
	assert.Equal("", tokenName)

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "true",
	})
	enabled, tokenName = EvaluateAnnotations(logger, input)
	assert.Equal(true, enabled)
	assert.Equal(DefaultTokenName, tokenName)

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled:   "true",
		AnnotationTokenName: "mytoken",
	})
	enabled, tokenName = EvaluateAnnotations(logger, input)
	assert.Equal(true, enabled)
	assert.Equal("mytoken", tokenName)
}

func TestCreatePatch(t *testing.T) {
	assert := assert.New(t)

	var machineSet *unstructured.Unstructured

	machineSet = &unstructured.Unstructured{Object: map[string]interface{}{}}
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME", "metadata", "labels", "machine.openshift.io/cluster-api-cluster")
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME", "spec", "selector", "matchLabels", "machine.openshift.io/cluster-api-cluster")
	unstructured.SetNestedField(machineSet.UnstructuredContent(), "INFRANAME-worker-us-east-2c", "spec", "selector", "matchLabels", "machine.openshift.io/cluster-api-machineset")

	patchBytes, err := CreatePatch(logger, machineSet, "INFRANAME", "MYCLUSTER")
	assert.Equal(nil, err)

	patch := []jsonpatch.Operation{}
	err = json.Unmarshal(patchBytes, &patch)

	expectedPatch := []jsonpatch.Operation{
		{Operation: "replace",
			Path:  "/metadata/labels/machine.openshift.io~1cluster-api-cluster",
			Value: "MYCLUSTER"},
		{Operation: "replace",
			Path:  "/spec/selector/matchLabels/machine.openshift.io~1cluster-api-cluster",
			Value: "MYCLUSTER"},
		{Operation: "replace",
			Path:  "/spec/selector/matchLabels/machine.openshift.io~1cluster-api-machineset",
			Value: "MYCLUSTER-worker-us-east-2c"}}
	assert.Equal(nil, err)
	assert.Equal(3, len(patch))
	assert.Equal(expectedPatch, patch)
}
