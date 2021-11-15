module knative.dev/container-freezer

go 1.16

require (
	github.com/containerd/containerd v1.5.7
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.42.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/cri-api v0.21.4
	knative.dev/hack v0.0.0-20211112192837-128cf0150a69
	knative.dev/pkg v0.0.0-20211111114938-0b0c3390a475
)
