package containerd

import (
	"context"
	"fmt"
	"testing"

	"github.com/containerd/containerd"
	"google.golang.org/grpc"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
	"knative.dev/container-freezer/pkg/daemon"
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
	c.paused = append(c.paused, container)
	if c.method != "freeze" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "freeze", c.method)
	}
	return nil
}

func (c *FakeContainerdCRI) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	c.resumed = append(c.resumed, container)
	if c.method != "thaw" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "freeze", c.method)
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
