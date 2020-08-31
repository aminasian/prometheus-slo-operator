module github.com/aminasian/prometheus-slo-operator

go 1.13

require (
	github.com/coreos/prometheus-operator v0.41.1 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	k8s.io/apimachinery v0.18.6
	k8s.io/autoscaler/vertical-pod-autoscaler v0.0.0-20200831110820-bbc60b10d7d0 // indirect
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)
