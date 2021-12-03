package controllers

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestDeleteMachineNow(t *testing.T) {
	assert := assert.New(t)

	var machine *unstructured.Unstructured

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(v1.Now())
	assert.Equal(false, deleteMachineNow(logger, machine))

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(v1.NewTime(time.Now().Add(-50 * time.Second)))
	assert.Equal(false, deleteMachineNow(logger, machine))

	machine = &unstructured.Unstructured{Object: map[string]interface{}{}}
	machine.SetCreationTimestamp(v1.NewTime(time.Now().Add(-2 * time.Minute)))
	assert.Equal(true, deleteMachineNow(logger, machine))
}
