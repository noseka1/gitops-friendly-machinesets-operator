/*
Copyright 2021 Ales Nosek.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	machineapi "github.com/openshift/api/machine/v1beta1"
	"github.com/wI2L/jsondiff"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MachineSetReconciler reconciles a MachineSet object
type MachineSetReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	ControllerName      string
	EventRecorder       record.EventRecorder
	InfrastructureName  string
	MachineSetInterface dynamic.NamespaceableResourceInterface
}

//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets/finalizers,verbs=update
func (r *MachineSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(2).Info("Reconciling object.")

	// Fetch the MachineSet object from Kubernetes
	machineSet, err := r.MachineSetInterface.Namespace(req.Namespace).Get(ctx, req.Name, v1.GetOptions{})
	if err != nil {
		err = processKubernetesError(logger, "get", err)
		return reconcile.Result{}, err
	}

	// Nothing to do if the object is being deleted
	if isObjectBeingDeleted(logger, machineSet) {
		return reconcile.Result{}, nil
	}

	// Is this object enabled for reconciliation?
	enabled, tokenName := evaluateAnnotations(logger, machineSet)
	if !enabled {
		return reconcile.Result{}, nil
	}

	// Replace tokens in the MachineSet object
	err = r.replaceTokens(logger, machineSet, tokenName, req, ctx)
	if err != nil {
		return reconcile.Result{}, err
	}

	if isWorkerMachineSet(machineSet) && hasNodesAvailable(machineSet) {
		downscaleInstallerProvisionedMachineSets()
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machineapi.MachineSet{}).
		Complete(r)
}

func isWorkerMachineSet(machineSet *unstructured.Unstructured) bool {
	role, _, _ := unstructured.NestedFieldNoCopy(machineSet.UnstructuredContent(), FieldSpec, FieldTemplate, FieldMetadata, FieldLabels, AnnotationMachineRole)
	roleString, ok := role.(string)
	return ok && roleString == MachineRoleWorker
}

func hasNodesAvailable(machineSet *unstructured.Unstructured) bool {
	availableReplicas, _, _ := unstructured.NestedFieldNoCopy(machineSet.UnstructuredContent(), FieldStatus, FieldAvailableReplicas)
	availableReplicasInt, ok := availableReplicas.(int64)
	return ok && availableReplicasInt > 0
}

func downscaleInstallerProvisionedMachineSets() {

}

func (r *MachineSetReconciler) replaceTokens(logger logr.Logger, machineSet *unstructured.Unstructured, tokenName string, req ctrl.Request, ctx context.Context) error {
	// Extract MachineSet sections that are going to be patched
	machineSetBytes, err := marshalObjectSections(logger, machineSet)
	if err != nil {
		return nil
	}

	// Compute the JSON patch
	machineSetPatchBytes, err := createPatch(logger, machineSetBytes, tokenName, r.InfrastructureName)
	if err != nil || len(machineSetPatchBytes) == 0 {
		return nil
	}

	// Patch the MachineSet object in Kubernetes
	_, err = r.MachineSetInterface.Namespace(req.Namespace).Patch(ctx, req.Name, types.JSONPatchType, machineSetPatchBytes, v1.PatchOptions{})
	if err != nil {
		err = processKubernetesError(logger, "patch", err)
		return err
	}

	logger.Info("MachineSet updated successfully.")
	return nil
}

func createPatch(logger logr.Logger, machineSetBytes []byte, tokenName, infrastructureName string) ([]byte, error) {
	machineSetUpdatedBytes := bytes.ReplaceAll(machineSetBytes, []byte(tokenName), []byte(infrastructureName))

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
