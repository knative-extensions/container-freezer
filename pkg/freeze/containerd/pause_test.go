package containerd

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/containerd/containerd"
	"google.golang.org/grpc"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"knative.dev/container-freezer/pkg/daemon"
)

var (
	containers []*cri.Container
	results    []string
	method     string
)

type FakeContainerdCRI struct {
	paused     []string
	resumed    []string
	containers []*cri.Container
	results    []string
	method     string
}

// Container creates a CRI container with the given container ID and name
func Container(id, name string) *cri.Container {
	return &cri.Container{
		Id: id,
		Metadata: &cri.ContainerMetadata{
			Name: name,
		},
	}
}

func (c *FakeContainerdCRI) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	return &cri.ListContainersResponse{Containers: c.containers}, nil
}

func (c *FakeContainerdCRI) Pause(ctx context.Context, ctrd *containerd.Client, container string) error {
	listContainers, _ := c.List(nil, nil, "")
	containerIDs, err := lookupContainerIDs(listContainers)
	if err != nil {
		return fmt.Errorf("expected list of containers ids but got %w", err)
	}
	c.paused = containerIDs
	if c.method != "freeze" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "freeze", method)
	}
	if !reflect.DeepEqual(c.paused, c.results) {
		return fmt.Errorf("paused list wrong, expected: %s, got %s", c.results, c.paused)
	}
	return nil
}

func (c *FakeContainerdCRI) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	listContainers, _ := c.List(nil, nil, "")
	containerIDs, err := lookupContainerIDs(listContainers)
	if err != nil {
		return fmt.Errorf("expected list of containers ids but got %w", err)
	}
	c.resumed = containerIDs
	if c.method != "thaw" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "freeze", method)
	}
	if !reflect.DeepEqual(c.resumed, c.results) {
		return fmt.Errorf("resumed list wrong, expected: %s, got %s", c.results, c.resumed)
	}
	return nil
}

func TestContainerPause(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	tests := []struct {
		containers []*cri.Container
		results    []string
		method     string
	}{{
		containers: []*cri.Container{Container("queueproxy", "queue-proxy"), Container("usercontainer", "user-container")},
		results:    []string{"usercontainer"},
		method:     "freeze",
	}, {
		containers: []*cri.Container{
			Container("queueproxy", "queue-proxy"),
			Container("usercontainer", "user-container"),
			Container("usercontainer2", "user-container2"),
		},
		results: []string{"usercontainer", "usercontainer2"},
		method:  "freeze",
	}}
	for _, c := range tests {
		fakeFreezeThawer, err = New(&FakeContainerdCRI{
			paused:     nil,
			containers: c.containers,
			results:    c.results,
			method:     c.method,
		})
		if err != nil {
			t.Errorf("expected New() to succeed but got %q", err)
		}
		if err := fakeFreezeThawer.Freeze(nil, ""); err != nil {
			t.Errorf("unable to freeze containers: %v", err)
		}
	}
}

func TestContainerResume(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	tests := []struct {
		containers []*cri.Container
		results    []string
		method     string
	}{{
		containers: []*cri.Container{Container("queueproxy", "queue-proxy"), Container("usercontainer", "user-container")},
		results:    []string{"usercontainer"},
		method:     "thaw",
	}, {
		containers: []*cri.Container{
			Container("queueproxy", "queue-proxy"),
			Container("usercontainer", "user-container"),
			Container("usercontainer2", "user-container2"),
		},
		results: []string{"usercontainer", "usercontainer2"},
		method:  "thaw",
	}}
	for _, c := range tests {
		fakeFreezeThawer, err = New(&FakeContainerdCRI{
			resumed:    nil,
			containers: c.containers,
			results:    c.results,
			method:     c.method,
		})
		if err != nil {
			t.Errorf("expected New() to succeed but got %q", err)
		}
		if err := fakeFreezeThawer.Thaw(nil, ""); err != nil {
			t.Errorf("unable to thaw containers: %v", err)
		}
	}
}
