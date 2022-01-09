module knative.dev/container-freezer

go 1.16

require (
	github.com/containerd/containerd v1.5.7
	github.com/docker/docker v20.10.12+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/morikuni/aec v1.0.0 // indirect
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.42.0
	k8s.io/api v0.22.5
	k8s.io/apimachinery v0.22.5
	k8s.io/client-go v0.22.5
	k8s.io/cri-api v0.21.4
	knative.dev/hack v0.0.0-20211222071919-abd085fc43de
	knative.dev/pkg v0.0.0-20220104185830-52e42b760b54
)
