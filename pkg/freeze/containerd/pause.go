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

	"knative.dev/container-freezer/pkg/freeze/common"
)

const defaultContainerdAddress = "/var/run/containerd/containerd.sock"

func NewContainerdProvider() (*ContainerdCRI, error) {
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

type ContainerdCRI struct {
	conn *grpc.ClientConn
	ctrd *containerd.Client
}

// List returns a list of all non queue-proxy container IDs in a given pod
func (c *ContainerdCRI) List(ctx context.Context, podUID string) ([]string, error) {
	containerIDs, err := common.List(ctx, c.conn, podUID)
	return containerIDs, err
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
