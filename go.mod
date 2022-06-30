module knative.dev/container-freezer

go 1.16

require (
	github.com/containerd/containerd v1.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/kelseyhightower/envconfig v1.4.0
	go.uber.org/zap v1.19.1
	google.golang.org/grpc v1.44.0
	k8s.io/api v0.23.8
	k8s.io/apimachinery v0.23.8
	k8s.io/client-go v0.23.8
	k8s.io/cri-api v0.23.1
	knative.dev/hack v0.0.0-20220629134730-e7d63651ce8f
	knative.dev/pkg v0.0.0-20220630112730-85965e1e8eb1
	knative.dev/serving v0.32.0
)
