package test

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	ctrdv1 "github.com/containerd/containerd/api/services/tasks/v1"
	types1 "github.com/gogo/protobuf/types"
	"google.golang.org/grpc"
	"k8s.io/cri-api/pkg/apis/runtime/v1alpha2"
)

const tmpSocketpath = "/tmp/"

type MockCtr struct {
	Id    string
	Name  string
	State string
}

type MockPod struct {
	Id   string
	Ctrs []MockCtr
}

type CRIServer struct {
	Pod []MockPod
}

func NewCriRuntimeServer() *CRIServer {
	return &CRIServer{}
}

func NewCrioRuntimerServer() *CrioServer {
	return &CrioServer{}
}

func NewCtrdRuntimeServer() *CtrdServer {
	return &CtrdServer{}
}

func (c *CRIServer) PodSandboxStats(ctx context.Context, request *v1alpha2.PodSandboxStatsRequest) (*v1alpha2.PodSandboxStatsResponse, error) {
	return &v1alpha2.PodSandboxStatsResponse{}, nil
}

func (c *CRIServer) ListPodSandboxStats(ctx context.Context, request *v1alpha2.ListPodSandboxStatsRequest) (*v1alpha2.ListPodSandboxStatsResponse, error) {
	return &v1alpha2.ListPodSandboxStatsResponse{}, nil
}

func (c *CRIServer) AddPodSandboxForCRI(pod MockPod) {
	c.Pod = append(c.Pod, pod)
}

func (c *CtrdServer) AddCtrForCtrd(ctr MockCtr) {
	c.Ctrs = append(c.Ctrs, ctr)
}

func (c *CrioServer) AddCrioForCtrd(ctr MockCtr) {
	c.Ctrs = append(c.Ctrs, ctr)
}

func (c *CRIServer) Version(ctx context.Context,
	req *v1alpha2.VersionRequest) (*v1alpha2.VersionResponse, error) {
	return nil, nil
}

func (c *CRIServer) RunPodSandbox(ctx context.Context,
	req *v1alpha2.RunPodSandboxRequest) (*v1alpha2.RunPodSandboxResponse, error) {
	return nil, nil
}

func (c *CRIServer) StopPodSandbox(ctx context.Context,
	req *v1alpha2.StopPodSandboxRequest) (*v1alpha2.StopPodSandboxResponse, error) {
	return nil, nil
}

func (c *CRIServer) RemovePodSandbox(ctx context.Context,
	req *v1alpha2.RemovePodSandboxRequest) (*v1alpha2.RemovePodSandboxResponse, error) {
	return nil, nil
}

func (c *CRIServer) PodSandboxStatus(ctx context.Context,
	req *v1alpha2.PodSandboxStatusRequest) (*v1alpha2.PodSandboxStatusResponse, error) {
	return nil, nil
}

func (c *CRIServer) ListPodSandbox(ctx context.Context,
	req *v1alpha2.ListPodSandboxRequest) (*v1alpha2.ListPodSandboxResponse, error) {
	data := &v1alpha2.ListPodSandboxResponse{
		Items: []*v1alpha2.PodSandbox{},
	}

	for _, v := range c.Pod {
		if v.Id == req.Filter.LabelSelector["io.kubernetes.pod.uid"] {
			item := &v1alpha2.PodSandbox{
				Id: v.Id,
			}
			data.Items = append(data.Items, item)
		}
	}

	return data, nil
}

func (c *CRIServer) CreateContainer(ctx context.Context,
	req *v1alpha2.CreateContainerRequest) (*v1alpha2.CreateContainerResponse, error) {
	return nil, nil
}

func (c *CRIServer) StartContainer(ctx context.Context,
	req *v1alpha2.StartContainerRequest) (*v1alpha2.StartContainerResponse, error) {
	return nil, nil
}

func (c *CRIServer) StopContainer(ctx context.Context,
	req *v1alpha2.StopContainerRequest) (*v1alpha2.StopContainerResponse, error) {
	return nil, nil
}

func (c *CRIServer) RemoveContainer(ctx context.Context,
	req *v1alpha2.RemoveContainerRequest) (*v1alpha2.RemoveContainerResponse, error) {
	return nil, nil
}

func (c *CRIServer) ListContainers(ctx context.Context,
	req *v1alpha2.ListContainersRequest) (*v1alpha2.ListContainersResponse, error) {
	data := &v1alpha2.ListContainersResponse{
		Containers: []*v1alpha2.Container{},
	}

	for _, v := range c.Pod {
		if v.Id == req.Filter.PodSandboxId {
			for _, ctr := range v.Ctrs {
				item := &v1alpha2.Container{
					Id: ctr.Id,
					Metadata: &v1alpha2.ContainerMetadata{
						Name: ctr.Name,
					},
				}
				data.Containers = append(data.Containers, item)
			}
		}
	}

	return data, nil
}

func (c *CRIServer) ContainerStatus(ctx context.Context,
	req *v1alpha2.ContainerStatusRequest) (*v1alpha2.ContainerStatusResponse, error) {
	return nil, nil
}

func (c *CRIServer) UpdateContainerResources(ctx context.Context,
	req *v1alpha2.UpdateContainerResourcesRequest) (*v1alpha2.UpdateContainerResourcesResponse, error) {
	return nil, nil
}

func (c *CRIServer) ReopenContainerLog(ctx context.Context,
	req *v1alpha2.ReopenContainerLogRequest) (*v1alpha2.ReopenContainerLogResponse, error) {
	return nil, nil
}

func (c *CRIServer) ExecSync(ctx context.Context,
	req *v1alpha2.ExecSyncRequest) (*v1alpha2.ExecSyncResponse, error) {
	return nil, nil
}

func (c *CRIServer) Exec(ctx context.Context,
	req *v1alpha2.ExecRequest) (*v1alpha2.ExecResponse, error) {
	return nil, nil
}

func (c *CRIServer) Attach(ctx context.Context,
	req *v1alpha2.AttachRequest) (*v1alpha2.AttachResponse, error) {
	return nil, nil
}

func (c *CRIServer) PortForward(ctx context.Context,
	req *v1alpha2.PortForwardRequest) (*v1alpha2.PortForwardResponse, error) {
	return nil, nil
}

func (c *CRIServer) ContainerStats(ctx context.Context,
	req *v1alpha2.ContainerStatsRequest) (*v1alpha2.ContainerStatsResponse, error) {
	return nil, nil
}

func (c *CRIServer) ListContainerStats(ctx context.Context,
	req *v1alpha2.ListContainerStatsRequest) (*v1alpha2.ListContainerStatsResponse, error) {
	return nil, nil
}
func (c *CRIServer) UpdateRuntimeConfig(ctx context.Context,
	req *v1alpha2.UpdateRuntimeConfigRequest) (*v1alpha2.UpdateRuntimeConfigResponse, error) {
	return nil, nil
}

func (c *CRIServer) Status(ctx context.Context,
	req *v1alpha2.StatusRequest) (*v1alpha2.StatusResponse, error) {
	return nil, nil
}

func GetRandomSocketPath() string {
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	bytes := []byte(str)
	result := []byte{}
	rand.Seed(time.Now().UnixNano() + int64(rand.Intn(100)))
	for i := 0; i < 8; i++ {
		result = append(result, bytes[rand.Intn(len(bytes))])
	}
	return tmpSocketpath + string(result) + "-test.sock"
}

func RunCriServer(c *CRIServer, socketPath string) {
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	criLis, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}

	s := grpc.NewServer()
	v1alpha2.RegisterRuntimeServiceServer(s, c)

	if err := s.Serve(criLis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}

type CtrdServer struct {
	Ctrs []MockCtr
}

func (c *CtrdServer) Create(ctx context.Context,
	req *ctrdv1.CreateTaskRequest) (*ctrdv1.CreateTaskResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Start(ctx context.Context,
	req *ctrdv1.StartRequest) (*ctrdv1.StartResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Delete(ctx context.Context,
	req *ctrdv1.DeleteTaskRequest) (*ctrdv1.DeleteResponse, error) {
	return nil, nil
}

func (c *CtrdServer) DeleteProcess(ctx context.Context,
	req *ctrdv1.DeleteProcessRequest) (*ctrdv1.DeleteResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Get(ctx context.Context,
	req *ctrdv1.GetRequest) (*ctrdv1.GetResponse, error) {
	return nil, nil
}

func (c *CtrdServer) List(ctx context.Context,
	req *ctrdv1.ListTasksRequest) (*ctrdv1.ListTasksResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Kill(ctx context.Context,
	req *ctrdv1.KillRequest) (*types1.Empty, error) {
	return nil, nil
}

func (c *CtrdServer) Exec(ctx context.Context,
	req *ctrdv1.ExecProcessRequest) (*types1.Empty, error) {
	return nil, nil
}

func (c *CtrdServer) ResizePty(ctx context.Context,
	req *ctrdv1.ResizePtyRequest) (*types1.Empty, error) {
	return nil, nil
}

func (c *CtrdServer) CloseIO(ctx context.Context,
	req *ctrdv1.CloseIORequest) (*types1.Empty, error) {
	return nil, nil
}

func (c *CtrdServer) Pause(ctx context.Context,
	req *ctrdv1.PauseTaskRequest) (*types1.Empty, error) {
	for _, v := range c.Ctrs {
		if req.ContainerID == v.Id {
			if v.State == "running" {
				data := &types1.Empty{}
				return data, nil
			} else {
				return nil, fmt.Errorf("not in running state")
			}

		}
	}
	return nil, fmt.Errorf("can't found ctr")
}

func (c *CtrdServer) Resume(ctx context.Context,
	req *ctrdv1.ResumeTaskRequest) (*types1.Empty, error) {
	for _, v := range c.Ctrs {
		if req.ContainerID == v.Id {
			if v.State == "paused" {
				data := &types1.Empty{}
				return data, nil
			} else {
				return nil, fmt.Errorf("not in paused state")
			}

		}
	}
	return nil, fmt.Errorf("can't found ctr")
}

func (c *CtrdServer) ListPids(ctx context.Context,
	req *ctrdv1.ListPidsRequest) (*ctrdv1.ListPidsResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Checkpoint(ctx context.Context,
	req *ctrdv1.CheckpointTaskRequest) (*ctrdv1.CheckpointTaskResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Update(ctx context.Context,
	req *ctrdv1.UpdateTaskRequest) (*types1.Empty, error) {
	return nil, nil
}

func (c *CtrdServer) Metrics(ctx context.Context,
	req *ctrdv1.MetricsRequest) (*ctrdv1.MetricsResponse, error) {
	return nil, nil
}

func (c *CtrdServer) Wait(ctx context.Context,
	req *ctrdv1.WaitRequest) (*ctrdv1.WaitResponse, error) {
	return nil, nil
}

func RunCtrdServer(c *CtrdServer, socketPath string) {
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	ctrdLis, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}

	s := grpc.NewServer()
	ctrdv1.RegisterTasksServer(s, c)

	if err := s.Serve(ctrdLis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}

type CrioServer struct {
	Ctrs []MockCtr
}

func getCtrId(path string) string {
	info := strings.Split(path, "/")
	return info[2]
}

func (c *CrioServer) pauseFunc(w http.ResponseWriter, r *http.Request) {
	ctrId := getCtrId(r.URL.Path)
	for _, v := range c.Ctrs {
		if ctrId == v.Id {
			if v.State == "running" {
				w.Write([]byte("200 OK"))
				return
			} else {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte("not in running state"))
				return
			}

		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found ctr"))
}

func (c *CrioServer) unpauseFunc(w http.ResponseWriter, r *http.Request) {
	ctrId := getCtrId(r.URL.Path)
	for _, v := range c.Ctrs {
		if ctrId == v.Id {
			if v.State == "paused" {
				w.Write([]byte("200 OK"))
				return
			} else {
				w.WriteHeader(http.StatusConflict)
				w.Write([]byte("not in paused state"))
				return
			}

		}
	}
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("not found ctr"))
}

func RunCrioServer(c *CrioServer, socketPath string) {
	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/pause/", c.pauseFunc)
	mux.HandleFunc("/unpause/", c.unpauseFunc)
	server := http.Server{
		Handler: mux,
	}

	crioLis, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(fmt.Sprintf("failed to listen: %v", err))
	}

	if err := server.Serve(crioLis); err != nil {
		panic(fmt.Sprintf("failed to serve: %v", err))
	}
}

func NewCRIGrpcClient(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))

	return conn, err
}

func NewCtrdGrpcClient(ctx context.Context, socketPath string) (*grpc.ClientConn, error) {
	conn, err := grpc.DialContext(ctx, socketPath, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))

	return conn, err
}

func NewCrioHttpClient(socketPath string) *http.Client {
	conn := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	return conn
}
