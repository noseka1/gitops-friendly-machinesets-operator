package controllers

import (
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

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
