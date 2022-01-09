package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerapi "github.com/docker/docker/client"

	"knative.dev/container-freezer/pkg/api"
)

const defaultDockerAPIVersion = "1.19"
const dockerUri = "unix:///var/run/docker.sock"

// Docker freezes and unfreezes containers via docker.
type Docker struct {
	cri api.CRI
}

// New return a FreezeThawer based on Docker.
// Requires unix:///var/run/docker.sock to be mounted.
func New(c api.CRI) *Docker {
	return &Docker{cri: c}
}

// NewCRI returns a CRI based on Docker.
func NewCRI(dockerAPIVersion string) (*DockerCRI, error) {
	var version string

	if dockerAPIVersion == "" {
		version = defaultDockerAPIVersion
	} else {
		version = dockerAPIVersion
	}
	client, err := dockerapi.NewClientWithOpts(dockerapi.WithHost(dockerUri),
		dockerapi.WithVersion(version))
	if err != nil {
		return nil, err
	}
	return &DockerCRI{client: client}, nil
}

// Freeze freezes the user container(s) via the freezer cgroup.
func (f *Docker) Freeze(ctx context.Context, podName string) error {
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
func (f *Docker) Thaw(ctx context.Context, podName string) error {
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

type DockerCRI struct {
	client *dockerapi.Client
}

// List returns a list of all non queue-proxy container IDs in a given pod
func (c *DockerCRI) List(ctx context.Context, podUID string) ([]string, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("io.kubernetes.pod.uid=%s", podUID))
	containers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{Filters: filter})
	if err != nil {
		return nil, err
	}
	if len(containers) == 0 {
		return nil, fmt.Errorf("pod %s not found", podUID)
	}
	containerIDs, err := lookupContainerIDs(containers)
	if err != nil {
		return nil, err
	}

	return containerIDs, nil
}

// Pause performs a pause action on a specific container
func (c *DockerCRI) Pause(ctx context.Context, container string) error {
	err := c.client.ContainerPause(ctx, container)
	if err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}
	return nil
}

// Resume performs a resume action on a specific container
func (c *DockerCRI) Resume(ctx context.Context, container string) error {
	err := c.client.ContainerUnpause(ctx, container)
	if err != nil {
		return fmt.Errorf("%s not resumed: %v", container, err)
	}
	return nil
}

func lookupContainerIDs(ctrs []types.Container) ([]string, error) {
	ids := make([]string, 0, len(ctrs)-1)
	for _, c := range ctrs {
		if v, ok := c.Labels["io.kubernetes.container.name"]; ok && v != "queue-proxy" {
			ids = append(ids, c.ID)
		}
	}
	if len(ids) == 0 {
		return nil, api.ErrNoNonQueueProxyPods
	}
	return ids, nil
}