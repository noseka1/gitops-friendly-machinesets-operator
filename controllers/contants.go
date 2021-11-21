package controllers

const (
	AnnotationBase      = "gitops-friendly-machinesets.redhat-cop.io"
	AnnotationEnabled   = AnnotationBase + "/enabled"
	AnnotationTokenName = AnnotationBase + "/token-name"

	DefaultTokenName = "INFRANAME"

	FieldSpec              = "spec"
	FieldTemplate          = "template"
	FieldMetadata          = "metadata"
	FieldLabels            = "labels"
	FieldStatus            = "status"
	FieldAvailableReplicas = "availableReplicas"

	AnnotationMachineRole = "machine.openshift.io/cluster-api-machine-role"

	MachineRoleWorker = "worker"

	EventTypeNormal   = "Normal"
	EventReasonDelete = "Delete"
)
