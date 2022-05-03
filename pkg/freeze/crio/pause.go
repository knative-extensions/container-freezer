package crio

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"google.golang.org/grpc"

	"knative.dev/container-freezer/pkg/freeze/common"
)

const defaultCrioAddress = "/var/run/crio/crio.sock"

func NewCrioProvider() (*CrioCRI, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, defaultCrioAddress, grpc.WithInsecure(), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*16)), grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", addr)
	}))
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", defaultCrioAddress)
			},
		},
	}

	return &CrioCRI{conn: conn, crioClient: client}, nil
}

type CrioCRI struct {
	conn       *grpc.ClientConn
	crioClient *http.Client
}

// List returns a list of all non queue-proxy container IDs in a given pod
func (c *CrioCRI) List(ctx context.Context, podUID string) ([]string, error) {
	containerIDs, err := common.List(ctx, c.conn, podUID)
	return containerIDs, err
}

// Pause performs a pause action on a specific container
func (c *CrioCRI) Pause(ctx context.Context, container string) error {
	resp, err := c.crioClient.Get("http://localhost/pause/" + container)
	if err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}

	if resp.StatusCode != http.StatusOK {
		errInfo, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%s not paused:%v", container, err)
		}
		return fmt.Errorf("%s not paused:%v", container, string(errInfo))
	}

	return nil
}

// Resume performs a resume action on a specific container
func (c *CrioCRI) Resume(ctx context.Context, container string) error {
	resp, err := c.crioClient.Get("http://localhost/unpause/" + container)
	if err != nil {
		return fmt.Errorf("%s not paused: %v", container, err)
	}

	if resp.StatusCode != http.StatusOK {
		errInfo, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("%s not paused:%v", container, err)
		}
		return fmt.Errorf("%s not paused:%v", container, string(errInfo))
	}

	return nil
}
