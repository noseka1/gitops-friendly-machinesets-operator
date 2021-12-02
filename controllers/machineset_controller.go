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
	"context"
	"encoding/json"
	"strings"

	"github.com/go-logr/logr"
	comm "github.com/noseka1/gitops-friendly-machinesets-operator/common"
	machineapi "github.com/openshift/api/machine/v1beta1"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MachineSetReconciler reconciles a MachineSet object
type MachineSetReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	ControllerName     string
	EventRecorder      record.EventRecorder
	InfrastructureName string
}

//+kubebuilder:rbac:groups=machine.openshift.io,resources=machinesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=machine.openshift.io,resources=machinesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=machine.openshift.io,resources=machinesets/finalizers,verbs=update
func (r *MachineSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(2).Info("Reconciling object.")

	// Fetch the MachineSet object from Kubernetes
	machineSet := &unstructured.Unstructured{}
	err := r.Get(ctx, req.NamespacedName, machineSet)
	if err != nil {
		err = processKubernetesError(logger, "get", err)
		return reconcile.Result{}, err
	}

	// Nothing to do if the object is being deleted
	if isObjectBeingDeleted(logger, machineSet) {
		return reconcile.Result{}, nil
	}

	// Is this object enabled for reconciliation?
	enabled, tokenName := comm.EvaluateAnnotations(logger, machineSet)
	if !enabled {
		return reconcile.Result{}, nil
	}

	// Replace tokens in the MachineSet object
	err = r.replaceTokens(ctx, req, machineSet, tokenName)
	if err != nil {
		return reconcile.Result{}, err
	}

	// If the managed MachineSet has at least one node available, check and scale the
	// installer-provisioned MachineSets to zero
	if isWorkerMachineSet(machineSet) && hasNodesAvailable(machineSet) {
		err = r.scaleInstallerProvisionedMachineSetsToZero(ctx, req)
		if err != nil {
			return ctrl.Result{}, err
		}
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
	role, _, _ := unstructured.NestedFieldNoCopy(machineSet.UnstructuredContent(), comm.FieldSpec, comm.FieldTemplate, comm.FieldMetadata, comm.FieldLabels, comm.AnnotationMachineRole)
	roleString, ok := role.(string)
	return ok && roleString == comm.MachineRoleWorker
}

func hasNodesAvailable(machineSet *unstructured.Unstructured) bool {
	availableReplicas, _, _ := unstructured.NestedFieldNoCopy(machineSet.UnstructuredContent(), comm.FieldStatus, comm.FieldAvailableReplicas)
	availableReplicasInt, ok := availableReplicas.(int64)
	return ok && availableReplicasInt > 0
}

func nameStartsWith(machineSet *unstructured.Unstructured, prefix string) bool {
	name := machineSet.GetName()
	return strings.HasPrefix(name, prefix)
}

func isReplicasGreaterThanZero(machineSet *unstructured.Unstructured) bool {
	replicas, _, _ := unstructured.NestedFieldNoCopy(machineSet.UnstructuredContent(), comm.FieldSpec, comm.FieldReplicas)
	replicasInt, ok := replicas.(int64)
	return ok && replicasInt > 0
}

// Look up all the installer-provisioned MachineSets and scale them to zero replicas.
// This will remove all the installer-provisioned Machines from the cluster.
func (r *MachineSetReconciler) scaleInstallerProvisionedMachineSetsToZero(ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx)

	allMachineSetsInNamespace := &unstructured.UnstructuredList{}
	err := r.List(ctx, allMachineSetsInNamespace, &client.ListOptions{Namespace: comm.NamespaceOpenShiftMachineApi})
	if err != nil {
		logger.Error(err, "Failed to retrieve MachineSets from namespace "+comm.NamespaceOpenShiftMachineApi)
		return err
	}

	for _, machineSet := range allMachineSetsInNamespace.Items {
		if isWorkerMachineSet(&machineSet) &&
			nameStartsWith(&machineSet, r.InfrastructureName) &&
			!comm.IsObjectReconciliationEnabled(&machineSet) &&
			isReplicasGreaterThanZero(&machineSet) {
			newLogger := log.FromContext(ctx, "scaled machineset", machineSet.GetNamespace()+"/"+machineSet.GetName())
			err := r.scaleMachineSetToZero(ctx, newLogger, &machineSet)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *MachineSetReconciler) scaleMachineSetToZero(ctx context.Context, logger logr.Logger, machineSet *unstructured.Unstructured) error {
	// Prepare the JSON patch to set replicas = 0
	jsonPatch := []jsonpatch.Operation{{Operation: "replace", Path: "/" + comm.FieldSpec + "/" + comm.FieldReplicas, Value: 0}}
	jsonPatchBytes, err := json.Marshal(jsonPatch)
	if err != nil {
		logger.Error(err, "Failed to marshal patch.")
		return nil
	}

	// Patch the MachineSet object in Kubernetes
	err = r.Patch(ctx, machineSet, client.RawPatch(types.JSONPatchType, jsonPatchBytes), &client.PatchOptions{})
	if err != nil {
		err = processKubernetesError(logger, "patch", err)
		return err
	}

	msg := "Scaling MachineSet provisioned by OpenShift installer to zero."
	r.EventRecorder.Event(machineSet, comm.EventTypeNormal, comm.EventReasonScale, msg)
	logger.Info(msg)

	return nil
}

func (r *MachineSetReconciler) replaceTokens(ctx context.Context, req ctrl.Request, machineSet *unstructured.Unstructured, tokenName string) error {
	logger := log.FromContext(ctx)

	// Compute the JSON patch
	machineSetPatchBytes, err := comm.CreatePatch(logger, machineSet, tokenName, r.InfrastructureName)
	if err != nil || len(machineSetPatchBytes) == 0 {
		return nil
	}

	// Patch the MachineSet object in Kubernetes
	err = r.Patch(ctx, machineSet, client.RawPatch(types.JSONPatchType, machineSetPatchBytes), &client.PatchOptions{})
	if err != nil {
		err = processKubernetesError(logger, "patch", err)
		return err
	}

	logger.Info("Tokens \"" + tokenName + "\" in MachineSet replaced successfully.")
	return nil
}
