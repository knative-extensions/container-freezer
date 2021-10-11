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
)

const defaultContainerdAddress = "/var/run/containerd/containerd.sock"

// Containerd freezes and unfreezes containers via containerd.
type Containerd struct {
	conn *grpc.ClientConn
}

// New return a FreezeThawer based on Containerd.
// Requires /var/run/containerd/containerd.sock to be mounted.
func New() (*Containerd, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, defaultContainerdAddress, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))
	if err != nil {
		return nil, err
	}

	return &Containerd{
		conn: conn,
	}, nil
}

// Freeze freezes the user container via the freezer cgroup.
func (f *Containerd) Freeze(ctx context.Context, podName, containerName string) error {
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return err
	}

	containerID, err := lookupContainerID(ctx, f.conn, podName, containerName)
	if err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := ctrd.TaskService().Pause(ctx, &tasks.PauseTaskRequest{ContainerID: containerID}); err != nil {
		return err
	}

	return nil
}

// Thaw thats a container which was freezed via the Freeze method.
func (f *Containerd) Thaw(ctx context.Context, podName, containerName string) error {
	ctrd, err := containerd.NewWithConn(f.conn)
	if err != nil {
		return err
	}

	containerID, err := lookupContainerID(ctx, f.conn, podName, containerName)
	if err != nil {
		return err
	}

	ctx = namespaces.WithNamespace(ctx, "k8s.io")
	if _, err := ctrd.TaskService().Resume(ctx, &tasks.ResumeTaskRequest{ContainerID: containerID}); err != nil {
		return err
	}

	return nil
}

func lookupContainerID(ctx context.Context, conn *grpc.ClientConn, podUID, containerName string) (string, error) {
	client := cri.NewRuntimeServiceClient(conn)
	pods, err := client.ListPodSandbox(context.Background(), &cri.ListPodSandboxRequest{
		Filter: &cri.PodSandboxFilter{
			LabelSelector: map[string]string{
				"io.kubernetes.pod.uid": podUID,
			},
		},
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("pod %s not found", podUID)
	}
	pod := pods.Items[0]

	ctrs, err := client.ListContainers(ctx, &cri.ListContainersRequest{Filter: &cri.ContainerFilter{
		PodSandboxId: pod.Id,
		LabelSelector: map[string]string{
			"io.kubernetes.container.name": containerName,
		},
	}})
	if err != nil {
		return "", err
	}

	if len(ctrs.Containers) == 0 {
		return "", fmt.Errorf("container %q in pod %q not found", containerName, podUID)
	}

	return ctrs.Containers[0].Id, nil
}
