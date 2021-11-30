package controllers

import (
	"errors"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
