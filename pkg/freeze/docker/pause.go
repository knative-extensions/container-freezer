package docker

import (
	"context"
	"errors"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerapi "github.com/docker/docker/client"
)

const defaultDockerUri = "unix:///var/run/docker.sock"
const version = "1.19"

var ErrNoNonQueueProxyPods = errors.New("no non queue-proxy containers found in pod")

type CRI interface {
	List(ctx context.Context, podUID string) ([]string, error)
	Pause(ctx context.Context, container string) error
	Resume(ctx context.Context, container string) error
}

type Docker struct {
	cri CRI
}

// New return a FreezeThawer based on Docker.
func New(c CRI) *Docker {
	return &Docker{cri: c}
}

func NewCRI() (*DockerCRI, error) {
	c, err := dockerapi.NewClientWithOpts(dockerapi.WithHost(defaultDockerUri),
		dockerapi.WithVersion(version))
	if err != nil {
		return nil, err
	}
	return &DockerCRI{client: c}, nil
}

func (d *Docker) Freeze(ctx context.Context, podUID string) error {
	containerIDs, err := d.cri.List(ctx, podUID)
	if err != nil {
		return err
	}
	for _, c := range containerIDs {
		if err := d.cri.Pause(ctx, c); err != nil {
			return fmt.Errorf("%s not paused: %v", c, err)
		}
	}
	return nil
}

// Thaw thaws a container which was frozen via the Freeze method.
func (d *Docker) Thaw(ctx context.Context, podUID string) error {
	containerIDs, err := d.cri.List(ctx, podUID)
	if err != nil {
		return err
	}
	for _, c := range containerIDs {
		if err := d.cri.Resume(ctx, c); err != nil {
			return fmt.Errorf("%s not resumed: %v", c, err)
		}
	}
	return nil
}

type DockerCRI struct {
	client *dockerapi.Client
}

func (d *DockerCRI) List(ctx context.Context, podUID string) ([]string, error) {
	var containerIDs []string
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("io.kubernetes.pod.uid=%s", podUID))
	containers, err := d.client.ContainerList(context.Background(), types.ContainerListOptions{Filters: filter})
	if err != nil {
		return nil, err
	}

	for _, c := range containers {
		if c.Labels["io.kubernetes.container.name"] != "queue-proxy" && c.Labels["io.kubernetes.container.name"] != "POD" {
			containerIDs = append(containerIDs, c.ID)
		}
	}
	if len(containerIDs) == 0 {
		return nil, ErrNoNonQueueProxyPods
	}
	return containerIDs, nil
}

func (d *DockerCRI) Pause(ctx context.Context, container string) error {
	if err := d.client.ContainerPause(ctx, container); err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}
	return nil
}

func (d *DockerCRI) Resume(ctx context.Context, container string) error {
	if err := d.client.ContainerUnpause(ctx, container); err != nil {
		return fmt.Errorf("%s not resumed: %v", container, err)
	}
	return nil
}
