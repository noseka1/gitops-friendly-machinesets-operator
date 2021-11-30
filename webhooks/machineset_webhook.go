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

package webhooks

import (
	"context"
	"net/http"

	comm "github.com/noseka1/gitops-friendly-machinesets-operator/common"
	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-machine-openshift-io-v1beta1-machineset,mutating=true,failurePolicy=fail,sideEffects=None,groups=machine.openshift.io,resources=machinesets,verbs=create;update,versions=v1beta1,name=gitops-friendly-machinesets,admissionReviewVersions={v1,v1beta1}

const (
	webhookPath string = "/mutate-machine-openshift-io-v1beta1-machineset"
)

type MachineSetWebhook struct {
	decoder            *admission.Decoder
	InfrastructureName string
}

// SetupWithManager sets up the webhook with the Manager.
func (m *MachineSetWebhook) SetupWithManager(mgr ctrl.Manager) {
	webhookServer := mgr.GetWebhookServer()
	webhookServer.Register(webhookPath, &webhook.Admission{Handler: m})
}

// A decoder will be automatically injected.
func (m *MachineSetWebhook) InjectDecoder(decoder *admission.Decoder) error {
	m.decoder = decoder
	return nil
}

// Replace tokens in MachineSet object
func (m *MachineSetWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithName("webhook.machineset").WithValues(
		comm.FieldNamespace, req.Namespace, comm.FieldName, req.Name)

	logger.V(2).Info("Called for object.")

	// Parse the MachineSet object
	machineSet := &unstructured.Unstructured{}
	err := m.decoder.Decode(req, machineSet)
	if err != nil {
		logger.Error(err, "Failed to decode the MachineSet object.")
		return admission.Errored(http.StatusBadRequest, err)
	}

	// Is this object enabled for reconciliation?
	enabled, tokenName := comm.EvaluateAnnotations(logger, machineSet)
	if !enabled {
		return admission.Allowed("")
	}

	// Compute the JSON patch
	machineSetPatchBytes, err := comm.CreatePatch(logger, machineSet, tokenName, m.InfrastructureName)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	// Nothing to patch
	if len(machineSetPatchBytes) == 0 {
		return admission.Allowed("")
	}

	logger.Info("Tokens \"" + tokenName + "\" in MachineSet replaced successfully.")

	return admission.Response{
		AdmissionResponse: admissionv1.AdmissionResponse{
			Allowed: true,
			Patch:   machineSetPatchBytes,
			PatchType: func() *admissionv1.PatchType {
				pt := admissionv1.PatchTypeJSONPatch
				return &pt
			}(),
		},
	}
}
