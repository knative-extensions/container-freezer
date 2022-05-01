package containerd

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/containerd/containerd"

	"knative.dev/container-freezer/pkg/freeze/test"
)

func runServerAndCreateProvider(ctx context.Context) (*ContainerdCRI, *test.CRIServer, *test.CtrdServer, error) {
	ctrdSever := test.NewCtrdRuntimeServer()
	ctrdSocketPath := test.GetRandomSocketPath()
	go test.RunCtrdServer(ctrdSever, ctrdSocketPath)

	criServer := test.NewCriRuntimeServer()
	criSocketPath := test.GetRandomSocketPath()
	go test.RunCriServer(criServer, criSocketPath)
	time.Sleep(time.Millisecond * 50)

	criGrpc, err := test.NewCRIGrpcClient(ctx, criSocketPath)
	if err != nil {
		return nil, nil, nil, err
	}

	ctrdGrpc, err := test.NewCtrdGrpcClient(ctx, ctrdSocketPath)
	if err != nil {
		return nil, nil, nil, err
	}

	ctrdConn, err := containerd.NewWithConn(ctrdGrpc)
	if err != nil {
		return nil, nil, nil, err
	}

	provider := &ContainerdCRI{
		conn: criGrpc,
		ctrd: ctrdConn,
	}

	return provider, criServer, ctrdSever, nil
}

func TestNewContainerdProvider(t *testing.T) {
	_, err := NewContainerdProvider()
	if !reflect.DeepEqual(err, nil) {
		t.Errorf("want error nil, but get:%v", err)
	}
}

func TestList(t *testing.T) {
	ctx := context.Background()
	provider, criServer, _, err := runServerAndCreateProvider(ctx)
	if err != nil {
		t.Errorf("init error:%v", err)
	}

	podAdd := test.MockPod{
		Id: "pod1",
		Ctrs: []test.MockCtr{
			{Id: "ctr1", Name: "ctr1"},
			{Id: "ctr2", Name: "queue-proxy"},
		},
	}
	criServer.AddPodSandboxForCRI(podAdd)

	resp, err := provider.List(ctx, "pod1")
	if !reflect.DeepEqual(err, nil) {
		t.Errorf("want error nil, but get:%v", err)
	}
	if resp[0] != "ctr1" {
		t.Errorf("want ctr:%v, but get:%v", "ctr1", resp)
	}
}

func TestPause(t *testing.T) {
	tests := []struct {
		ctrsAdd     test.MockCtr
		reqCtrId    string
		expectError bool
	}{
		//has related ctr with running state
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "running"},
			reqCtrId:    "ctr1",
			expectError: false,
		},
		//has related ctr but request other
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "running"},
			reqCtrId:    "ctr2",
			expectError: true,
		},
		//has related ctr but wrong state
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "paused"},
			reqCtrId:    "ctr1",
			expectError: true,
		},
	}

	for _, v := range tests {
		ctx := context.Background()
		provider, _, ctrdServer, err := runServerAndCreateProvider(ctx)
		if err != nil {
			t.Errorf("init error:%v", err)
		}

		ctrdServer.AddCtrForCtrd(v.ctrsAdd)
		err = provider.Pause(ctx, v.reqCtrId)
		if (err != nil) != v.expectError {
			t.Errorf("expect error exist:%v, but get:%v", v.expectError, err)
		}
	}
}

func TestResume(t *testing.T) {
	tests := []struct {
		ctrsAdd     test.MockCtr
		reqCtrId    string
		expectError bool
	}{
		//has related ctr with paused state
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "paused"},
			reqCtrId:    "ctr1",
			expectError: false,
		},
		//has related ctr but request other
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "paused"},
			reqCtrId:    "ctr2",
			expectError: true,
		},
		//has related ctr but wrong state
		{
			ctrsAdd:     test.MockCtr{Id: "ctr1", Name: "ctr1", State: "running"},
			reqCtrId:    "ctr1",
			expectError: true,
		},
	}

	for _, v := range tests {
		ctx := context.Background()
		provider, _, ctrdServer, err := runServerAndCreateProvider(ctx)
		if err != nil {
			t.Errorf("init error:%v", err)
		}

		ctrdServer.AddCtrForCtrd(v.ctrsAdd)
		err = provider.Resume(ctx, v.reqCtrId)
		if (err != nil) != v.expectError {
			t.Errorf("expect error exist:%v, but get:%v", v.expectError, err)
		}
	}
}
