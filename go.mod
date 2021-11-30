module github.com/noseka1/gitops-friendly-machinesets-operator

go 1.16

require (
	github.com/go-logr/logr v0.4.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.15.0
	github.com/openshift/api v0.0.0-20211108165917-be1be0e89115
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.0
	gomodules.xyz/jsonpatch/v2 v2.2.0 // indirect
	k8s.io/api v0.22.1 // indirect
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/controller-runtime v0.10.0
)
