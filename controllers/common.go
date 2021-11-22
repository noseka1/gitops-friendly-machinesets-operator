package controllers

import (
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func isObjectReconciliationEnabled(obj *unstructured.Unstructured) bool {
	annotations := obj.GetAnnotations()
	enabledString, enabledFound := annotations[AnnotationEnabled]
	return enabledFound && enabledString == "true"
}

func evaluateAnnotations(logger logr.Logger, obj *unstructured.Unstructured) (bool, string) {
	if !isObjectReconciliationEnabled(obj) {
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

func marshalObjectSections(logger logr.Logger, obj *unstructured.Unstructured) ([]byte, error) {
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

func processKubernetesError(logger logr.Logger, operation string, err error) error {
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(2).Info("Object no longer exists.", "err", err)
			return nil
		} else if apierrors.IsConflict(err) {
			logger.V(2).Info("Update coflict.", "err", err)
			return err
		} else {
			logger.Error(err, "Failed to "+operation+" Kubernetes object.")
			return err
		}
	}
	return nil
}

func isObjectBeingDeleted(logger logr.Logger, obj *unstructured.Unstructured) bool {
	if obj.GetDeletionTimestamp() != nil {
		logger.V(2).Info("Skipping reconciliation of object as it is being deleted.")
		return true
	}
	return false
}
