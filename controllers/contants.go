package controllers

const (
	AnnotationBase      = "gitops-friendly-machinesets.redhat-cop.io"
	AnnotationEnabled   = AnnotationBase + "/enabled"
	AnnotationTokenName = AnnotationBase + "/token-name"

	DefaultTokenName = "INFRANAME"

	SpecFieldName = "spec"

	EventTypeNormal   = "Normal"
	EventReasonDelete = "Delete"
)
