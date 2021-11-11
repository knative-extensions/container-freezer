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

type FakeContainerdCri struct {
	containers []*cri.Container
	results    []string
	method     string
}

// Container creates a CRI container with the given container ID and name
func Container(id, name string) cri.Container {
	return cri.Container{
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
		return nil, fmt.Errorf("list of containers did not matche expected results: %w", err)
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

	queueProxy := Container("queueproxy", "queue-proxy")
	userContainer := Container("usercontainer", "user-container")
	userContainer2 := Container("usercontainer2", "user-container2")

	tests := []struct {
		containers []*cri.Container
		results    []string
		method     string
	}{{
		containers: []*cri.Container{&queueProxy, &userContainer},
		results:    []string{"usercontainer"},
		method:     "freeze",
	}, {
		containers: []*cri.Container{&queueProxy, &userContainer, &userContainer2},
		results:    []string{"usercontainer", "usercontainer2"},
		method:     "freeze",
	}}
	for _, c := range tests {
		fakeFreezeThawer, err = New(&FakeContainerdCri{
			containers: c.containers,
			results:    c.results,
			method:     c.method,
		})
		if err != nil {
			t.Errorf("expected New() to succeed by got %q", err)
		}
		if err := fakeFreezeThawer.Freeze(nil, ""); err != nil {
			t.Errorf("unable to freeze containers: %v", err)
		}
	}
}

func TestContainerResume(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	queueProxy := Container("queueproxy", "queue-proxy")
	userContainer := Container("usercontainer", "user-container")
	userContainer2 := Container("usercontainer2", "user-container2")

	tests := []struct {
		containers []*cri.Container
		results    []string
		method     string
	}{{
		containers: []*cri.Container{&queueProxy, &userContainer},
		results:    []string{"usercontainer"},
		method:     "thaw",
	}, {
		containers: []*cri.Container{&queueProxy, &userContainer, &userContainer2},
		results:    []string{"usercontainer", "usercontainer2"},
		method:     "thaw",
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
	queueProxy := Container("queueproxy", "queue-proxy")

	containers = []*cri.Container{&queueProxy}
	results = []string{}
	method = ""

	f := FakeContainerdCri{
		containers: containers,
		results:    results,
		method:     method,
	}
	_, err := f.List(nil, nil, "")
	if err == nil {
		t.Errorf("expecting error and got nil")
	}
	if err.Error() != "expected list of containers ids but got no non queue-proxy containers found in pod" {
		t.Errorf("expecting \"no non queue-proxy\" error but got %v", err)
	}
}

func compareContainerLists(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if v != b[k] {
			return false
		}
	}
	return true
}
