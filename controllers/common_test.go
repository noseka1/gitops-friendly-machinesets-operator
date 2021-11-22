package controllers

import (
	"errors"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
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
	ctrl.SetLogger(logger)
}
func TestIsObjectReconciliationEnabled(t *testing.T) {
	assert := assert.New(t)

	var input *unstructured.Unstructured

	input = &unstructured.Unstructured{}
	assert.Equal(false, isObjectReconciliationEnabled(input))

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "false",
	})
	assert.Equal(false, isObjectReconciliationEnabled(input))

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "true",
	})
	assert.Equal(true, isObjectReconciliationEnabled(input))
}
func TestEvaluateAnnotations(t *testing.T) {
	assert := assert.New(t)

	var input *unstructured.Unstructured
	var enabled bool
	var tokenName string

	input = &unstructured.Unstructured{}
	enabled, tokenName = evaluateAnnotations(logger, input)
	assert.Equal(false, enabled)
	assert.Equal("", tokenName)

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled: "true",
	})
	enabled, tokenName = evaluateAnnotations(logger, input)
	assert.Equal(true, enabled)
	assert.Equal(DefaultTokenName, tokenName)

	input = &unstructured.Unstructured{}
	input.SetAnnotations(map[string]string{
		AnnotationEnabled:   "true",
		AnnotationTokenName: "mytoken",
	})
	enabled, tokenName = evaluateAnnotations(logger, input)
	assert.Equal(true, enabled)
	assert.Equal("mytoken", tokenName)
}

func TestProcessKubernetesError(t *testing.T) {
	assert := assert.New(t)

	resource := schema.GroupResource{
		Group:    "machine.openshift.io",
		Resource: "machinesets",
	}
	name := "cluster-3af1-6cnhg-worker-us-east-2a"
	operation := "get"
	var err error

	assert.Equal(nil, processKubernetesError(logger, operation, nil))

	err = apierrors.NewNotFound(resource, name)
	assert.Equal(nil, processKubernetesError(logger, operation, err))

	err = apierrors.NewConflict(resource, name, errors.New("some error"))
	assert.Equal(err, processKubernetesError(logger, operation, err))

	err = apierrors.NewAlreadyExists(resource, name)
	assert.Equal(err, processKubernetesError(logger, operation, err))
}
