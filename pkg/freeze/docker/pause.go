package docker

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	dockerapi "github.com/docker/docker/client"
)

const defaultDockerUri = "unix:///var/run/docker.sock"
const version = "1.19"

type Docker struct {
	client *dockerapi.Client
}

// New return a FreezeThawer based on Docker.
func New() (*Docker, error) {
	c, err := dockerapi.NewClientWithOpts(dockerapi.WithHost(defaultDockerUri),
		dockerapi.WithVersion(version))
	if err != nil {
		return nil, err
	}
	return &Docker{client: c}, nil
}

func (d *Docker) Freeze(ctx context.Context, podUID, containerName string) error {
	log.Println("Start to freeze container", podUID, containerName)
	containerID, err := d.lookupContainerID(ctx, podUID, containerName)
	if err != nil {
		return err
	}
	err = d.client.ContainerPause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("pause container: %s", err.Error())
	}
	log.Println("Freeze container", podUID, containerName, "success !")
	return nil
}

// Thaw thaws a container which was frozen via the Freeze method.
func (d *Docker) Thaw(ctx context.Context, podUID, containerName string) error {
	log.Println("Start to thaw container", podUID, containerName)
	containerID, err := d.lookupContainerID(ctx, podUID, containerName)
	if err != nil {
		return err
	}

	err = d.client.ContainerUnpause(ctx, containerID)
	if err != nil {
		return fmt.Errorf("pause container: %s", err.Error())
	}
	log.Println("Thaw container", podUID, containerName, "success !")
	return nil
}

func (d *Docker) lookupContainerID(ctx context.Context, podUID, containerName string) (string, error) {
	filter := filters.NewArgs()
	filter.Add("label", fmt.Sprintf("io.kubernetes.pod.uid=%s", podUID))
	filter.Add("label", fmt.Sprintf("io.kubernetes.container.name=%s", containerName))
	containers, err := d.client.ContainerList(context.Background(), types.ContainerListOptions{Filters: filter})
	if err != nil {
		return "", err
	}

	if len(containers) == 0 {
		return "", fmt.Errorf("container %q in pod %q not found", containerName, podUID)
	}

	return containers[0].ID, nil
}
