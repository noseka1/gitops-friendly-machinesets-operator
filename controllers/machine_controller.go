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
	"fmt"
	"time"

	"github.com/go-logr/logr"
	machineapi "github.com/openshift/api/machine/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ControllerName   string
	EventRecorder    record.EventRecorder
	MachineInterface dynamic.NamespaceableResourceInterface
}

//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machines/finalizers,verbs=update
func (r *MachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.V(2).Info("Reconciling object.")

	// Fetch the Machine object from Kubernetes
	machine, err := r.MachineInterface.Namespace(req.Namespace).Get(ctx, req.Name, v1.GetOptions{})
	if err != nil {
		err = processKubernetesError(logger, "get", err)
		return reconcile.Result{}, err
	}

	// Nothing to do if the object is being deleted
	if isObjectBeingDeleted(logger, machine) {
		return reconcile.Result{}, nil
	}

	// Is this object enabled for reconciliation?
	enabled, tokenName := evaluateAnnotations(logger, machine)
	if !enabled {
		return reconcile.Result{}, nil
	}

	// Extract Machine sections that should have been patched
	machineBytes, err := marshalObjectSections(logger, machine)
	if err != nil {
		return reconcile.Result{}, nil
	}

	// If we cannot find the token in the Machine object, we are going to leave this object alone
	if !bytes.Contains(machineBytes, []byte(tokenName)) {
		return reconcile.Result{}, nil
	}

	if deleteMachineNow(logger, machine) {
		// Delete the Machine object in Kubernetes. Object contained tokens that were not replaced.
		err = r.MachineInterface.Namespace(req.Namespace).Delete(ctx, req.Name, v1.DeleteOptions{})
		if err != nil {
			err = processKubernetesError(logger, "delete", err)
			return reconcile.Result{}, err
		}

		msg := "Machine contains unresolved tokens \"" + tokenName + "\". Deleting it."
		r.EventRecorder.Event(machine, EventTypeNormal, EventReasonDelete, msg)
		logger.Info(msg)
		return ctrl.Result{}, nil
	}

	// Requeue the request
	return ctrl.Result{RequeueAfter: DeleteMachineRequeueAfter}, nil
}

// Check if we should send the delete request at this time. If the Machine was created based on the MachineSet
// that our controller haven't updated on time, we want to delay the deletion of this Machine.
// We want to give machine-api-controller enough time to notice the MachineSet update. After we delete the Machine,
// a new Machine will immediately be created. Leave enough time so that the replacement Machine is created based
// on the udpated MachineSet.
func deleteMachineNow(logger logr.Logger, machine *unstructured.Unstructured) bool {
	now := v1.NewTime(time.Now())
	creationTime := machine.GetCreationTimestamp().Time
	age := int(now.Sub(creationTime).Seconds())
	result := age > DeleteMachineMinAgeSeconds
	if !result {
		logger.V(3).Info("Not deleting machine that is only " + fmt.Sprint(age) + " secs old.")
	}
	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machineapi.Machine{}).
		Complete(r)
}
