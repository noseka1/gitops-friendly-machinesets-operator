package common

import (
	"bytes"
	"encoding/json"

	"github.com/go-logr/logr"
	"github.com/wI2L/jsondiff"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func IsObjectReconciliationEnabled(obj *unstructured.Unstructured) bool {
	annotations := obj.GetAnnotations()
	enabledString, enabledFound := annotations[AnnotationEnabled]
	return enabledFound && enabledString == "true"
}

func EvaluateAnnotations(logger logr.Logger, obj *unstructured.Unstructured) (bool, string) {
	if !IsObjectReconciliationEnabled(obj) {
		logger.V(2).Info("Skipping object. Annotation that allows object patching was not found.")
		return false, ""
	}

	annotations := obj.GetAnnotations()
	tokenName, tokenNameFound := annotations[AnnotationTokenName]

	if !tokenNameFound {
		tokenName = DefaultTokenName
	}

	return true, tokenName
}

func MarshalObjectSections(logger logr.Logger, obj *unstructured.Unstructured) ([]byte, error) {
	section := unstructured.Unstructured{Object: map[string]interface{}{}}

	labelsField := obj.GetLabels()
	section.SetLabels(labelsField)

	specField, _, _ := unstructured.NestedFieldNoCopy(obj.UnstructuredContent(), FieldSpec)
	unstructured.SetNestedField(section.UnstructuredContent(), specField, FieldSpec)

	sectionBytes, err := section.MarshalJSON()
	if err != nil {
		logger.Error(err, "Failed to marshall object sections to JSON")
	}
	return sectionBytes, err
}
func CreatePatch(logger logr.Logger, machineSet *unstructured.Unstructured, tokenName, infrastructureName string) ([]byte, error) {
	// Extract MachineSet sections that are going to be patched
	machineSetBytes, err := MarshalObjectSections(logger, machineSet)
	if err != nil {
		return []byte{}, err
	}

	// Replace the token in the serialized JSON
	machineSetUpdatedBytes := bytes.ReplaceAll(machineSetBytes, []byte(tokenName), []byte(infrastructureName))

	// Compute the JSON patch
	jsonPatch, err := jsondiff.CompareJSON(machineSetBytes, machineSetUpdatedBytes)
	if err != nil {
		logger.Error(err, "Failed to generate patch.")
		return []byte{}, err
	}

	if len(jsonPatch) == 0 {
		logger.V(2).Info("Nothing to patch for object.")
		return []byte{}, nil
	}

	jsonPatchBytes, err := json.Marshal(jsonPatch)
	if err != nil {
		logger.Error(err, "Failed to marshal patch.")
		return []byte{}, err
	}

	logger.V(3).Info("Generated JSON Patch: " + string(jsonPatchBytes))

	return jsonPatchBytes, nil
}
