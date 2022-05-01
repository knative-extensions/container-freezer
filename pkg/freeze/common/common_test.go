package common

import (
	"context"
	"testing"
	"time"

	"knative.dev/container-freezer/pkg/freeze/test"
)

func checkCtrListEqual(expect, act []string) bool {
	if len(expect) != len(act) {
		return false
	}

	for i := 0; i < len(expect); i++ {
		if expect[i] != act[i] {
			return false
		}
	}

	return true
}

func TestListContainers(t *testing.T) {
	tests := []struct {
		requestPodId string
		pod          test.MockPod
		expectCtrs   []string
	}{
		//no related pod
		{
			requestPodId: "none",
			pod: test.MockPod{
				Id: "pod1",
				Ctrs: []test.MockCtr{
					{Id: "ctr1", Name: "ctr1"},
					{Id: "ctr2", Name: "queue-proxy"},
				},
			},
			expectCtrs: []string{},
		},
		//with queue-proxy and one user container
		{
			requestPodId: "pod1",
			pod: test.MockPod{
				Id: "pod1",
				Ctrs: []test.MockCtr{
					{Id: "ctr1", Name: "ctr1"},
					{Id: "ctr2", Name: "queue-proxy"},
				},
			},
			expectCtrs: []string{"ctr1"},
		},
		//with queue-proxy and two user container
		{
			requestPodId: "pod1",
			pod: test.MockPod{
				Id: "pod1",
				Ctrs: []test.MockCtr{
					{Id: "ctr1", Name: "ctr1"},
					{Id: "ctr2", Name: "ctr2"},
					{Id: "ctr3", Name: "queue-proxy"},
				},
			},
			expectCtrs: []string{"ctr1", "ctr2"},
		},
		//with queue-proxy only
		{
			requestPodId: "pod1",
			pod: test.MockPod{
				Id: "pod1",
				Ctrs: []test.MockCtr{
					{Id: "ctr1", Name: "queue-proxy"},
				},
			},
			expectCtrs: []string{},
		},
		//with user container only
		{
			requestPodId: "pod1",
			pod: test.MockPod{
				Id: "pod1",
				Ctrs: []test.MockCtr{
					{Id: "ctr1", Name: "ctr1"},
				},
			},
			expectCtrs: []string{"ctr1"},
		},
	}

	for _, v := range tests {
		socketPath := test.GetRandomSocketPath()
		criServer := test.NewCriRuntimeServer()
		go test.RunCriServer(criServer, socketPath)
		time.Sleep(time.Millisecond * 50)

		ctx := context.Background()
		criServer.AddPodSandboxForCRI(v.pod)
		conn, err := test.NewCRIGrpcClient(ctx, socketPath)
		if err != nil {
			t.Errorf("New grpc client error:%v", err)
		}

		ctrs, err := List(ctx, conn, v.requestPodId)
		if !checkCtrListEqual(ctrs, v.expectCtrs) {
			t.Errorf("expect ctrs:%v, but get: %v", v.expectCtrs, ctrs)
		}
	}
}
