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

func (f *FakeContainerdCRI) List(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	return &cri.ListContainersResponse{Containers: f.containers}, nil
}

func (f *FakeContainerdCRI) Pause(ctx context.Context, ctrd *containerd.Client, container string) error {
	f.paused = append(f.paused, container)
	f.method = "pause"
	return nil
}

func (f *FakeContainerdCRI) Resume(ctx context.Context, ctrd *containerd.Client, container string) error {
	f.resumed = append(f.resumed, container)
	f.method = "resume"
	return nil
}

func TestContainerPause(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	tests := []struct {
		containers    []*cri.Container
		expectedPause []string
	}{{
		containers:    []*cri.Container{Container("queueproxy", "queue-proxy"), Container("usercontainer", "user-container")},
		expectedPause: []string{"usercontainer"},
	}, {
		containers: []*cri.Container{
			Container("queueproxy", "queue-proxy"),
			Container("usercontainer", "user-container"),
			Container("usercontainer2", "user-container2"),
		},
		expectedPause: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeContainerCRI := &FakeContainerdCRI{
			paused:     nil,
			containers: c.containers,
		}
		fakeFreezeThawer, err = New(fakeContainerCRI)
		if err != nil {
			t.Errorf("expected New() to succeed but got %q", err)
		}
		if err := fakeFreezeThawer.Freeze(nil, ""); err != nil {
			t.Errorf("expected freeze to succeed but failed: %v", err)
		}
		if !reflect.DeepEqual(fakeContainerCRI.paused, c.expectedPause) {
			t.Errorf("pod has %s containers, but only %s frozen", c.expectedPause, fakeContainerCRI.paused)
		}
		if fakeContainerCRI.method != "pause" {
			t.Errorf("wrong method, expected: %s, got: %s", "pause", fakeContainerCRI.method)
		}
	}
}

func TestContainerResume(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	tests := []struct {
		containers     []*cri.Container
		expectedResume []string
	}{{
		containers:     []*cri.Container{Container("queueproxy", "queue-proxy"), Container("usercontainer", "user-container")},
		expectedResume: []string{"usercontainer"},
	}, {
		containers: []*cri.Container{
			Container("queueproxy", "queue-proxy"),
			Container("usercontainer", "user-container"),
			Container("usercontainer2", "user-container2"),
		},
		expectedResume: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeContainerdCRI := &FakeContainerdCRI{
			resumed:    nil,
			containers: c.containers,
		}
		fakeFreezeThawer, err = New(fakeContainerdCRI)
		if err != nil {
			t.Errorf("expected New() to succeed but got %q", err)
		}
		if err := fakeFreezeThawer.Thaw(nil, ""); err != nil {
			t.Errorf("expected thaw to succeed but failed: %v", err)
		}
		if !reflect.DeepEqual(fakeContainerdCRI.resumed, c.expectedResume) {
			t.Errorf("pod has %s containers, but only %s thawed", c.expectedResume, fakeContainerdCRI.resumed)
		}
		if fakeContainerdCRI.method != "resume" {
			t.Errorf("wrong method, expected: %s, got: %s", "resume", fakeContainerdCRI.method)
		}
	}
}
