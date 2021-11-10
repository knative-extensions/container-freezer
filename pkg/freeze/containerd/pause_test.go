package containerd

import (
	"context"
	"testing"

	"google.golang.org/grpc"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

type FakeContainerd struct {
	conn *grpc.ClientConn
}

var containers []*cri.Container

func (f *FakeContainerd) GetContainers(ctx context.Context, conn *grpc.ClientConn, podUID string) (*cri.ListContainersResponse, error) {
	listContainers := &cri.ListContainersResponse{
		Containers: containers,
	}
	return listContainers, nil
}

func TestLookupContainerIDs(t *testing.T) {

	queueProxy := cri.Container{
		Id: "queueproxy",
		Metadata: &cri.ContainerMetadata{
			Name: "queue-proxy",
		},
	}
	userContainer := cri.Container{
		Id: "usercontainer",
		Metadata: &cri.ContainerMetadata{
			Name: "user-container",
		},
	}
	userContainer2 := cri.Container{
		Id: "usercontainer2",
		Metadata: &cri.ContainerMetadata{
			Name: "user-container2",
		},
	}

	tests := []struct {
		containers []*cri.Container
		results    []string
	}{{
		containers: []*cri.Container{&queueProxy, &userContainer},
		results:    []string{"usercontainer"},
	}, {
		containers: []*cri.Container{&queueProxy, &userContainer, &userContainer2},
		results:    []string{"usercontainer", "usercontainer2"},
	}, {
		containers: []*cri.Container{&queueProxy},
		results:    []string{},
	}}

	for _, c := range tests {
		containers = c.containers
		f := FakeContainerd{conn: nil}
		cntr, _ := f.GetContainers(nil, nil, "")
		ids, _ := lookupContainerIDs(cntr)

		if !compareContainerLists(ids, c.results) {
			t.Errorf("expected %s, got %s", ids, c.results)
		}
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
