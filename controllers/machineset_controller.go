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

	machineapi "github.com/openshift/api/machine/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// MachineSetReconciler reconciles a MachineSet object
type MachineSetReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	MachineSetInterface dynamic.NamespaceableResourceInterface
}

//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=machine.openshift.io.redhat.com,resources=machinesets/finalizers,verbs=update
func (r *MachineSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling MachinePool object " + req.String())

	ms, err := r.MachineSetInterface.Namespace(req.Namespace).Get(ctx, req.Name, v1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(4).Error(err, "object "+req.NamespacedName.String()+" no longer exists")
			return reconcile.Result{}, nil
		}
		logger.Error(err, "failed to get "+req.NamespacedName.String())
		return reconcile.Result{}, err
	}
	marshall, _ := json.Marshal(ms.Object)
	logger.Info("JSON: " + string(marshall))

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&machineapi.MachineSet{}).
		Complete(r)
}
