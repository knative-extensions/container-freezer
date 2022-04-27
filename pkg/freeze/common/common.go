package common

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	cri "k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const (
	RuntimeTypeContainerd = "containerd"
	RuntimeTypeCrio       = "crio"
)

var ErrNoNonQueueProxyPods = errors.New("no non queue-proxy containers found in pod")

func List(ctx context.Context, conn *grpc.ClientConn, podUID string) ([]string, error) {
	client := cri.NewRuntimeServiceClient(conn)
	pods, err := client.ListPodSandbox(context.Background(), &cri.ListPodSandboxRequest{
		Filter: &cri.PodSandboxFilter{
			LabelSelector: map[string]string{
				"io.kubernetes.pod.uid": podUID,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("pod %s not found", podUID)
	}
	pod := pods.Items[0]

	ctrs, err := client.ListContainers(ctx, &cri.ListContainersRequest{Filter: &cri.ContainerFilter{
		PodSandboxId: pod.Id,
	}})
	if err != nil {
		return nil, err
	}

	containerIDs, err := lookupContainerIDs(ctrs)
	if err != nil {
		return nil, err
	}

	return containerIDs, nil
}

func lookupContainerIDs(ctrs *cri.ListContainersResponse) ([]string, error) {
	ids := make([]string, 0, len(ctrs.Containers)-1)
	for _, c := range ctrs.Containers {
		if c.GetMetadata().GetName() != "queue-proxy" {
			ids = append(ids, c.Id)
		}
	}
	if len(ids) == 0 {
		return nil, ErrNoNonQueueProxyPods
	}
	return ids, nil
}
