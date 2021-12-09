package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"knative.dev/container-freezer/pkg/daemon"
)

type FakeDockerCRI struct {
	paused     []string
	resumed    []string
	containers []types.Container
	method     string
}

func (f *FakeDockerCRI) List(ctx context.Context, podUID string) ([]string, error) {
	var containerList []string
	for _, c := range f.containers {
		if c.Names[0] != "queue-proxy" {
			containerList = append(containerList, c.ID)
		}
	}
	return containerList, nil
}

func (f *FakeDockerCRI) Pause(ctx context.Context, container string) error {
	f.paused = append(f.paused, container)
	f.method = "pause"
	return nil
}

func (f *FakeDockerCRI) Resume(ctx context.Context, container string) error {
	f.resumed = append(f.resumed, container)
	f.method = "resume"
	return nil
}

func TestContainerPause(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer
	var err error

	tests := []struct {
		// TODO change containers type to be for Docker
		containers    []types.Container
		expectedPause []string
	}{{
		containers: []types.Container{
			{Names: []string{"queue-proxy"}, ID: "queueproxy"},
			{Names: []string{"user-container"}, ID: "usercontainer"},
		},
		expectedPause: []string{"usercontainer"},
	}, {
		containers: []types.Container{
			{Names: []string{"queue-proxy"}, ID: "queueproxy"},
			{Names: []string{"user-container"}, ID: "usercontainer"},
			{Names: []string{"user-container2"}, ID: "usercontainer2"},
		},
		expectedPause: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeContainerCRI := &FakeDockerCRI{
			paused:     nil,
			containers: c.containers,
		}
		fakeFreezeThawer = New(fakeContainerCRI)
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
		containers     []types.Container
		expectedResume []string
	}{{
		containers: []types.Container{
			{Names: []string{"queue-proxy"}, ID: "queueproxy"},
			{Names: []string{"user-container"}, ID: "usercontainer"},
		},
		expectedResume: []string{"usercontainer"},
	}, {
		containers: []types.Container{
			{Names: []string{"queue-proxy"}, ID: "queueproxy"},
			{Names: []string{"user-container"}, ID: "usercontainer"},
			{Names: []string{"user-container2"}, ID: "usercontainer2"},
		},
		expectedResume: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeContainerdCRI := &FakeDockerCRI{
			resumed:    nil,
			containers: c.containers,
		}
		fakeFreezeThawer = New(fakeContainerdCRI)
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
