package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func evaluateAnnotations(logger logr.Logger, obj *unstructured.Unstructured) (bool, string) {
	annotations := obj.GetAnnotations()
	enabledString, enabledFound := annotations[AnnotationEnabled]
	tokenName, tokenNameFound := annotations[AnnotationTokenName]

	if !enabledFound || enabledString != "true" {
		logger.V(2).Info("Skipping object. Annotation that allows object patching was not found.")
		return false, ""
	}

	enabled := true

	if !tokenNameFound {
		tokenName = DefaultTokenName
	}

	return enabled, tokenName
}

func marshalObjectSections(logger logr.Logger, obj *unstructured.Unstructured) ([]byte, error) {
	section := unstructured.Unstructured{}

	labelsField := obj.GetLabels()
	section.SetLabels(labelsField)

	specField, _, _ := unstructured.NestedFieldNoCopy(obj.Object, SpecFieldName)
	unstructured.SetNestedField(section.Object, specField, SpecFieldName)

	sectionBytes, err := section.MarshalJSON()
	if err != nil {
		logger.Error(err, "Failed to marshall object sections to JSON")
	}
	return sectionBytes, err
}
