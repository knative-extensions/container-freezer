package containerd

import (
	"context"
	"reflect"
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
	return nil
}

func (c *FakeContainerdCRI) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	c.resumed = append(c.resumed, container)
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
		pods, method, err := fakeFreezeThawer.Freeze(nil, "")
		if !reflect.DeepEqual(pods, c.results) {
			t.Errorf("pod has %s containers, but only %s frozen", c.results, pods)
		}
		if method != c.method {
			t.Errorf("wrong method, expected: %s, got: %s", c.method, method)
		}
		if err != nil {
			t.Errorf("expected freeze to succeed but failed: %v", err)
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
		pods, method, err := fakeFreezeThawer.Thaw(nil, "")
		if !reflect.DeepEqual(pods, c.results) {
			t.Errorf("pod has %s containers, but only %s thawed", c.results, pods)
		}
		if method != c.method {
			t.Errorf("wrong method, expected: %s, got: %s", c.method, method)
		}
		if err != nil {
			t.Errorf("expected thaw to succeed but failed: %v", err)
		}
	}
}
