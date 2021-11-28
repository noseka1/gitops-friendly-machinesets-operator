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

package v1beta1

import (
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var machinesetlog = logf.Log.WithName("machineset-resource")

func (r *MachineSet) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-machine-openshift-io-v1beta1-machineset,mutating=true,failurePolicy=fail,sideEffects=None,groups=machine.openshift.io,resources=machinesets,verbs=create;update,versions=v1beta1,name=mmachineset.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &MachineSet{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *MachineSet) Default() {
	machinesetlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}
