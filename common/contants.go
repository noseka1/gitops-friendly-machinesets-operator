package common

import "time"

const (
	AnnotationBase      = "gitops-friendly-machinesets.redhat-cop.io"
	AnnotationEnabled   = AnnotationBase + "/enabled"
	AnnotationTokenName = AnnotationBase + "/token-name"

	DefaultTokenName = "INFRANAME"

	FieldName              = "name"
	FieldNamespace         = "namespace"
	FieldSpec              = "spec"
	FieldTemplate          = "template"
	FieldMetadata          = "metadata"
	FieldLabels            = "labels"
	FieldStatus            = "status"
	FieldAvailableReplicas = "availableReplicas"
	FieldReplicas          = "replicas"

	AnnotationMachineRole = "machine.openshift.io/cluster-api-machine-role"

	MachineRoleWorker = "worker"

	EventTypeNormal   = "Normal"
	EventReasonDelete = "Delete"
	EventReasonScale  = "Scale"

	NamespaceOpenShiftMachineApi = "openshift-machine-api"

	DeleteMachineMinAgeSeconds = 60
	DeleteMachineRequeueAfter  = 20 * time.Second
)
