package containerd

import (
	"context"
	"errors"
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

type FakeContainerdCri struct {
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

func (c *FakeContainerdCri) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	listContainers := &cri.ListContainersResponse{
		Containers: c.containers,
	}
	containerIDs, err := lookupContainerIDs(listContainers)
	if err != nil {
		return nil, fmt.Errorf("expected list of containers ids but got %w", err)
	}
	if !reflect.DeepEqual(containerIDs, c.results) {
		return nil, fmt.Errorf("list of containers did not match expected results: got %q, want %q", c.results, containerIDs)
	}
	return listContainers, nil
}

func (c *FakeContainerdCri) Pause(ctx context.Context, ctrd *containerd.Client, container string) error {
	if c.method != "freeze" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "freeze", method)
	}
	return nil
}

func (c *FakeContainerdCri) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	if c.method != "thaw" {
		return fmt.Errorf("wrong method, expected: %s, got: %s", "thaw", method)
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
		fakeFreezeThawer, err = New(&FakeContainerdCri{
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
		fakeFreezeThawer, err = New(&FakeContainerdCri{
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

func TestNoQueueProxyPause(t *testing.T) {
	f := FakeContainerdCri{
		containers: []*cri.Container{Container("queueproxy", "queue-proxy")},
	}
	_, err := f.List(nil, nil, "")
	if err == nil {
		t.Errorf("expecting error and got nil")
	}
	if !errors.Is(err, ErrNoNonQueueProxyPods) {
		t.Errorf("expecting %q error but got %q", ErrNoNonQueueProxyPods, err)
	}
}
