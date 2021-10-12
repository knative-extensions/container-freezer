module knative.dev/container-freezer

go 1.16

require (
	github.com/containerd/containerd v1.5.7
	github.com/docker/docker v20.10.9+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.41.0
	k8s.io/api v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	k8s.io/cri-api v0.21.4
	knative.dev/hack v0.0.0-20210806075220-815cd312d65c
	knative.dev/pkg v0.0.0-20211006213726-9179f78dcf97
)
