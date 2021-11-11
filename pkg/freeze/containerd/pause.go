package containerd

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/api/services/tasks/v1"
	"github.com/containerd/containerd/namespaces"
	types1 "github.com/gogo/protobuf/types"
	"google.golang.org/grpc"

	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const defaultContainerdAddress = "/var/run/containerd/containerd.sock"

type CRI interface {
	List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error)
	Pause(ctx context.Context, ctrd *containerd.Client, containerList []string) error
	Resume(ctx context.Context, ctrd *containerd.Client, container string) (*types1.Empty, error)
}

// Containerd freezes and unfreezes containers via containerd.
type Containerd struct {
	conn       *grpc.ClientConn
	containerd CRI
}

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

// Freeze freezes the user container via the freezer cgroup.
func (f *Containerd) Freeze(ctx context.Context, podName string) error {
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return err
	}

	containers, err := f.containerd.List(ctx, f.conn, podName)
	containerIDs, err := lookupContainerIDs(containers)
	if err != nil {
		return err
	}

	if err := f.containerd.Pause(ctx, ctrd, containerIDs); err != nil {
		return err
	}
	return nil
}

// Thaw thats a container which was freezed via the Freeze method.
func (f *Containerd) Thaw(ctx context.Context, podName string) error {
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return err
	}

	containers, err := f.containerd.List(ctx, f.conn, podName)
	containerIDs, err := lookupContainerIDs(containers)
	if err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	for _, c := range containerIDs {
		if _, err := f.containerd.Resume(ctx, ctrd, c); err != nil {
			return err
		}
	}
	return nil
}

type ContainerdCri struct{}

// List returns all containers in a given pod
func (c *ContainerdCri) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
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
func (c *ContainerdCri) Pause(ctx context.Context, ctrd *containerd.Client, containerList []string) error {
	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	for _, c := range containerList {
		if _, err := ctrd.TaskService().Pause(ctx, &tasks.PauseTaskRequest{ContainerID: c}); err != nil {
			return fmt.Errorf("%s not paused: %v", c, err)
		}
	}
	return nil
}

// Resume performs a resume action on a specific container
func (c *ContainerdCri) Resume(ctx context.Context, ctrd *containerd.Client, container string) (*types1.Empty, error) {
	return ctrd.TaskService().Resume(ctx, &tasks.ResumeTaskRequest{ContainerID: container})
}

func lookupContainerIDs(ctrs *cri.ListContainersResponse) ([]string, error) {
	ids := make([]string, 0, len(ctrs.Containers)-1)
	for _, c := range ctrs.Containers {
		if c.GetMetadata().GetName() != "queue-proxy" {
			ids = append(ids, c.Id)
		}
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no non queue-proxy containers found in pod")
	}

	return ids, nil
}
