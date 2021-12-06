package containerd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	"google.golang.org/grpc"

	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const defaultContainerdAddress = "/var/run/containerd/containerd.sock"

type CRI interface {
	List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error)
	Pause(ctx context.Context, ctrd *containerd.Client, container string) error
	Resume(ctx context.Context, ctrd *containerd.Client, container string) error
}

// Containerd freezes and unfreezes containers via containerd.
type Containerd struct {
	conn       *grpc.ClientConn
	containerd CRI
}

var ErrNoNonQueueProxyPods = errors.New("no non queue-proxy containers found in pod")

// New return a FreezeThawer based on Containerd.
// Requires /var/run/containerd/containerd.sock to be mounted.
func New(c CRI) (*Containerd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, defaultContainerdAddress, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))
	if err != nil {
		return nil, err
	}

	return &Containerd{
		conn:       conn,
		containerd: c,
	}, nil
}

// Freeze freezes the user container(s) via the freezer cgroup.
func (f *Containerd) Freeze(ctx context.Context, podName string) ([]string, string, error) {
	var frozen []string
	method := "freeze"
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return frozen, method, err
	}

	containers, err := f.containerd.List(ctx, f.conn, podName)
	containerIDs, err := lookupContainerIDs(containers)
	if err != nil {
		return frozen, method, err
	}

	for _, c := range containerIDs {
		if err := f.containerd.Pause(ctx, ctrd, c); err != nil {
			return frozen, method, fmt.Errorf("%s not paused: %v", c, err)
		}
		frozen = append(frozen, c)
	}
	if !reflect.DeepEqual(frozen, containerIDs) {
		return frozen, method, fmt.Errorf("pod has %s containers, but only %s frozen", containerIDs, frozen)
	}
	return frozen, method, nil
}

// Thaw thaws the user container(s) frozen via the Freeze method.
func (f *Containerd) Thaw(ctx context.Context, podName string) ([]string, string, error) {
	var thawed []string
	method := "thaw"
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return thawed, method, err
	}

	containers, err := f.containerd.List(ctx, f.conn, podName)
	containerIDs, err := lookupContainerIDs(containers)
	if err != nil {
		return thawed, method, err
	}

	for _, c := range containerIDs {
		if err := f.containerd.Resume(ctx, ctrd, c); err != nil {
			return thawed, method, fmt.Errorf("%s not resumed: %v", c, err)
		}
		thawed = append(thawed, c)
	}
	if !reflect.DeepEqual(thawed, containerIDs) {
		return thawed, method, fmt.Errorf("pod has %s containers, but only %s thawed", containerIDs, thawed)
	}
	return thawed, method, nil
}

type ContainerdCRI struct{}

// List returns all containers in a given pod
func (c *ContainerdCRI) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	client := cri.NewRuntimeServiceClient(conn)
	pods, err := client.ListPodSandbox(context.Background(), &cri.ListPodSandboxRequest{
		Filter: &cri.PodSandboxFilter{
			LabelSelector: map[string]string{
				"io.kubernetes.pod.uid": podUID,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("pod %s not found", podUID)
	}
	pod := pods.Items[0]

	ctrs, err := client.ListContainers(ctx, &cri.ListContainersRequest{Filter: &cri.ContainerFilter{
		PodSandboxId: pod.Id,
	}})
	if err != nil {
		return nil, err
	}

	return ctrs, nil
}

// Pause performs a pause action on a specific container
func (c *ContainerdCRI) Pause(ctx context.Context, ctrd *containerd.Client, container string) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := ctrd.TaskService().Pause(ctx, &tasks.PauseTaskRequest{ContainerID: container}); err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}
	return nil
}

// Resume performs a resume action on a specific container
func (c *ContainerdCRI) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := ctrd.TaskService().Resume(ctx, &tasks.ResumeTaskRequest{ContainerID: container}); err != nil {
		return fmt.Errorf("%s not resumed: %v", container, err)
	}
	return nil
}

func lookupContainerIDs(ctrs *cri.ListContainersResponse) ([]string, error) {
	ids := make([]string, 0, len(ctrs.Containers)-1)
	for _, c := range ctrs.Containers {
		if c.GetMetadata().GetName() != "queue-proxy" {
			ids = append(ids, c.Id)
		}
	}
	if len(ids) == 0 {
		return nil, ErrNoNonQueueProxyPods
	}
	return ids, nil
}
