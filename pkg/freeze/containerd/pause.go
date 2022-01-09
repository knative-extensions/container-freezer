package containerd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	"google.golang.org/grpc"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	
	"knative.dev/container-freezer/pkg/api"
)

const defaultContainerdAddress = "/var/run/containerd/containerd.sock"

// Containerd freezes and unfreezes containers via containerd.
type Containerd struct {
	cri api.CRI
}

// New return a FreezeThawer based on Containerd.
// Requires /var/run/containerd/containerd.sock to be mounted.
func New(c api.CRI) *Containerd {
	return &Containerd{cri: c}
}

// NewCRI returns a CRI based on Containerd.
func NewCRI() (*ContainerdCRI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, defaultContainerdAddress, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))
	if err != nil {
		return nil, err
	}

	client, err := containerd.NewWithConn(conn)
	if err != nil {
		return nil, err
	}
	return &ContainerdCRI{conn: conn, ctrd: client}, nil
}

// Freeze freezes the user container(s) via the freezer cgroup.
func (f *Containerd) Freeze(ctx context.Context, podName string) error {
	containerIDs, err := f.cri.List(ctx, podName)
	if err != nil {
		return err
	}

	for _, c := range containerIDs {
		if err := f.cri.Pause(ctx, c); err != nil {
			return fmt.Errorf("%s not paused: %v", c, err)
		}
	}
	return nil
}

// Thaw thaws the user container(s) frozen via the Freeze method.
func (f *Containerd) Thaw(ctx context.Context, podName string) error {
	containerIDs, err := f.cri.List(ctx, podName)
	if err != nil {
		return err
	}

	for _, c := range containerIDs {
		if err := f.cri.Resume(ctx, c); err != nil {
			return fmt.Errorf("%s not resumed: %v", c, err)
		}
	}
	return nil
}

type ContainerdCRI struct {
	conn *grpc.ClientConn
	ctrd *containerd.Client
}

// List returns a list of all non queue-proxy container IDs in a given pod
func (c *ContainerdCRI) List(ctx context.Context, podUID string) ([]string, error) {
	client := cri.NewRuntimeServiceClient(c.conn)
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

	containerIDs, err := lookupContainerIDs(ctrs)
	if err != nil {
		return nil, err
	}

	return containerIDs, nil
}

// Pause performs a pause action on a specific container
func (c *ContainerdCRI) Pause(ctx context.Context, container string) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := c.ctrd.TaskService().Pause(ctx, &tasks.PauseTaskRequest{ContainerID: container}); err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}
	return nil
}

// Resume performs a resume action on a specific container
func (c *ContainerdCRI) Resume(ctx context.Context, container string) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := c.ctrd.TaskService().Resume(ctx, &tasks.ResumeTaskRequest{ContainerID: container}); err != nil {
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
		return nil, api.ErrNoNonQueueProxyPods
	}
	return ids, nil
}
