package controllers

import (
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
