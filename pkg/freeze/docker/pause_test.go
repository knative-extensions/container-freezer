package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"

	"knative.dev/container-freezer/pkg/daemon"
)

type FakeDockerCRI struct {
	paused     []string
	resumed    []string
	containers []types.Container
	method     string
}

// Container creates a CRI container with the given container ID and name
func Container(id, name string) *cri.Container {
	return &cri.Container{
		Id: id,
		Labels: map[string]string{
			"io.kubernetes.container.name": name,
		},
	}
}

func (f *FakeDockerCRI) List(ctx context.Context, podUID string) ([]string, error) {
	var containerList []string
	for _, c := range f.containers {
		if v, ok := c.Labels["io.kubernetes.container.name"]; ok && v != "queue-proxy" {
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

	tests := []struct {
		containers    []types.Container
		expectedPause []string
	}{{
		containers:    []types.Container{
			{ID: "queueproxy", Labels: map[string]string{"io.kubernetes.container.name": "queue-proxy"}},
			{ID: "usercontainer", Labels: map[string]string{"io.kubernetes.container.name": "user-container"}},
		},
		expectedPause: []string{"usercontainer"},
	}, {
		containers: []types.Container{
			{ID: "queueproxy", Labels: map[string]string{"io.kubernetes.container.name": "queue-proxy"}},
			{ID: "usercontainer", Labels: map[string]string{"io.kubernetes.container.name": "user-container"}},
			{ID: "usercontainer2", Labels: map[string]string{"io.kubernetes.container.name": "user-container2"}},
		},
		expectedPause: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeDockerCRI := &FakeDockerCRI{
			paused:     nil,
			containers: c.containers,
		}
		fakeFreezeThawer = New(fakeDockerCRI)
		if err := fakeFreezeThawer.Freeze(nil, ""); err != nil {
			t.Errorf("expected freeze to succeed but failed: %v", err)
		}
		if !reflect.DeepEqual(fakeDockerCRI.paused, c.expectedPause) {
			t.Errorf("pod has %s containers, but only %s frozen", c.expectedPause, fakeDockerCRI.paused)
		}
		if fakeDockerCRI.method != "pause" {
			t.Errorf("wrong method, expected: %s, got: %s", "pause", fakeDockerCRI.method)
		}
	}
}

func TestContainerResume(t *testing.T) {
	var fakeFreezeThawer daemon.FreezeThawer

	tests := []struct {
		containers     []types.Container
		expectedResume []string
	}{{
		containers:     []types.Container{
			{ID: "queueproxy", Labels: map[string]string{"io.kubernetes.container.name": "queue-proxy"}},
			{ID: "usercontainer", Labels: map[string]string{"io.kubernetes.container.name": "user-container"}},
		},
		expectedResume: []string{"usercontainer"},
	}, {
		containers: []types.Container{
			{ID: "queueproxy", Labels: map[string]string{"io.kubernetes.container.name": "queue-proxy"}},
			{ID: "usercontainer", Labels: map[string]string{"io.kubernetes.container.name": "user-container"}},
			{ID: "usercontainer2", Labels: map[string]string{"io.kubernetes.container.name": "user-container2"}},
		},
		expectedResume: []string{"usercontainer", "usercontainer2"},
	}}
	for _, c := range tests {
		fakeDockerCRI := &FakeDockerCRI{
			resumed:    nil,
			containers: c.containers,
		}
		fakeFreezeThawer = New(fakeDockerCRI)
		if err := fakeFreezeThawer.Thaw(nil, ""); err != nil {
			t.Errorf("expected thaw to succeed but failed: %v", err)
		}
		if !reflect.DeepEqual(fakeDockerCRI.resumed, c.expectedResume) {
			t.Errorf("pod has %s containers, but only %s thawed", c.expectedResume, fakeDockerCRI.resumed)
		}
		if fakeDockerCRI.method != "resume" {
			t.Errorf("wrong method, expected: %s, got: %s", "resume", fakeDockerCRI.method)
		}
	}
}
