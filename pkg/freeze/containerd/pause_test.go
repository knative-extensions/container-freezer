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

const (
	freezeMethod = "freeze"
	thawMethod   = "thaw"
)

var (
	containers []*cri.Container
	results    []string
	method     string
)

var (
	queueProxy     = cri.Container{Id: "queueproxy", Metadata: &cri.ContainerMetadata{Name: "queue-proxy"}}
	userContainer  = cri.Container{Id: "usercontainer", Metadata: &cri.ContainerMetadata{Name: "user-container"}}
	userContainer2 = cri.Container{Id: "usercontainer2", Metadata: &cri.ContainerMetadata{Name: "user-container2"}}
)

type FakeContainerdCri struct{}

func (c *FakeContainerdCri) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	listContainers := &cri.ListContainersResponse{
		Containers: containers,
	}
	return listContainers, nil
}

func (c *FakeContainerdCri) Pause(ctx context.Context, ctrd *containerd.Client, containerList []string) error {
	if method != freezeMethod {
		return fmt.Errorf("wrong method, expected: %s, got: %s", freezeMethod, method)
	}
	if !compareContainerLists(containerList, results) {
		return fmt.Errorf("paused container list did not match")
	}
	return nil
}

func (c *FakeContainerdCri) Resume(ctx context.Context, ctrd *containerd.Client, containerList []string) error {
	if method != thawMethod {
		return fmt.Errorf("wrong method, expected: %s, got: %s", thawMethod, method)
	}
	if !compareContainerLists(containerList, results) {
		return fmt.Errorf("resume container list did not match")
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
		containers: []*cri.Container{&queueProxy, &userContainer},
		results:    []string{"usercontainer"},
		method:     "freeze",
	}, {
		containers: []*cri.Container{&queueProxy, &userContainer, &userContainer2},
		results:    []string{"usercontainer", "usercontainer2"},
		method:     "freeze",
	}}
	for _, c := range tests {
		containers = c.containers
		results = c.results
		method = c.method

		fakeFreezeThawer, err = New(&FakeContainerdCri{})
		if err != nil {
			t.Errorf("unable to create freezeThawer: %v", err)
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
		containers: []*cri.Container{&queueProxy, &userContainer},
		results:    []string{"usercontainer"},
		method:     "thaw",
	}, {
		containers: []*cri.Container{&queueProxy, &userContainer, &userContainer2},
		results:    []string{"usercontainer", "usercontainer2"},
		method:     "thaw",
	}}
	for _, c := range tests {
		containers = c.containers
		results = c.results
		method = c.method

		fakeFreezeThawer, err = New(&FakeContainerdCri{})
		if err != nil {
			t.Errorf("unable to create freezeThawer: %v", err)
		}
		if err := fakeFreezeThawer.Thaw(nil, ""); err != nil {
			t.Errorf("unable to thaw containers: %v", err)
		}
	}
}

func TestNoQueueProxyPause(t *testing.T) {
	containers = []*cri.Container{&queueProxy}
	results = []string{}

	f := FakeContainerdCri{}
	cntr, _ := f.List(nil, nil, "")
	ids, _ := lookupContainerIDs(cntr)
	if !compareContainerLists(ids, results) {
		t.Errorf("expected %s, got %s", ids, results)
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
