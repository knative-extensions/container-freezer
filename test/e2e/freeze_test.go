//go:build e2e
// +build e2e

/*
Copyright 2022 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/environment"
	"knative.dev/serving/test"
)

const (
	ksvcUrl           = "sleeptalker.default.example.com"
	deployNamespace   = "default"
	workerNodeName    = "kind-worker"
	lbsvcNamespace    = "kourier-system"
	lbsvcName         = "kourier"
	crioRuntime       = "crio"
	containerdRuntime = "containerd"
)

func newClient() (*test.Clients, error) {
	env := environment.ClientConfig{}
	flag.Parse()
	cfg, err := env.GetRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig %v", err)
	}

	clients, err := test.NewClients(cfg, "default")
	if err != nil {
		return nil, fmt.Errorf("failed to setup clients: %v", err)
	}

	return clients, nil
}

func findTheLastTimestamp(ctx context.Context, clients *test.Clients) (time.Time, error) {
	logs, err := logSleepTalkerPod(ctx, clients.KubeClient)
	if err != nil {
		return time.Time{}, err
	}
	lines := strings.Split(string(logs), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			timeStamp := strings.Split(lines[i], "at: ")
			// case: Ticking at: Wed Jun 29 15:42:54 2022
			return time.Parse(time.ANSIC, timeStamp[1])
		}
	}

	return time.Time{}, fmt.Errorf("no log lines")
}

func logSleepTalkerPod(ctx context.Context, client kubernetes.Interface) ([]byte, error) {
	pods, err := client.CoreV1().Pods(deployNamespace).
		List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("get pod failed:%v", err)
	}
	if len(pods.Items) != 1 {
		return nil, fmt.Errorf("pods are more than one, please check")
	}

	var logTailNums int64 = 3
	sleepTalkerPod := pods.Items[0]
	tickLogsByte, err := client.CoreV1().Pods(deployNamespace).
		GetLogs(sleepTalkerPod.Name, &v1.PodLogOptions{Container: "user-container", TailLines: &logTailNums}).
		Do(ctx).Raw()
	if err != nil {
		return nil, fmt.Errorf("get pod:%s log error:%v", sleepTalkerPod.Name, err)
	}

	return tickLogsByte, nil
}

func isPausedState(ctx context.Context, clients *test.Clients) error {
	timeBefore, err := findTheLastTimestamp(ctx, clients)
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 5)

	timeAfter, err := findTheLastTimestamp(ctx, clients)
	if err != nil {
		return err
	}

	if timeAfter != timeBefore {
		return fmt.Errorf("time log is not same as 5 seconds ago")
	}

	return nil
}

func getRequestIP(ctx context.Context, clients *test.Clients) (string, error) {
	runtimeType := os.Getenv("RUNTIME_TYPE")
	switch runtimeType {
	case crioRuntime:
		return "127.0.0.1", nil
	case containerdRuntime:
		node, err := clients.KubeClient.CoreV1().Nodes().Get(ctx, workerNodeName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("get worker node error:%v", err)
		}

		nodeIP := ""
		for _, v := range node.Status.Addresses {
			if v.Type == v1.NodeInternalIP {
				nodeIP = v.Address
			}
		}
		if nodeIP == "" {
			return "", fmt.Errorf("node ip is empty")
		}
		return nodeIP, nil
	}

	return "", fmt.Errorf("wrong type runtime")
}

func requestService(ctx context.Context, clients *test.Clients) error {
	requestIP, err := getRequestIP(ctx, clients)
	if err != nil {
		return fmt.Errorf("get request ip error")
	}

	var nodePort int32
	svc, err := clients.KubeClient.CoreV1().Services(lbsvcNamespace).
		Get(ctx, lbsvcName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get lb svc error:%v", err)
	}
	for _, v := range svc.Spec.Ports {
		if v.Port == 80 {
			nodePort = v.NodePort
		}
	}
	if nodePort == 0 {
		return fmt.Errorf("lb's nodePort is 0")
	}

	reqUrl := "http://" + requestIP + ":" + strconv.Itoa(int(nodePort))
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		return fmt.Errorf("create http request error:%v", err)
	}
	req.Host = ksvcUrl
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request error:%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("response code is:%d, request url:%s, requrst host:%s", resp.StatusCode, reqUrl, ksvcUrl)
	}

	return nil
}

func requestShouldBeResumed(ctx context.Context, clients *test.Clients) error {
	timeBefore, err := findTheLastTimestamp(ctx, clients)
	if err != nil {
		return fmt.Errorf("get log timestamp error:%v", err)
	}

	time.Sleep(time.Second * 5)

	if err := requestService(ctx, clients); err != nil {
		return fmt.Errorf("request svc error:%v", err)
	}

	timeAfter, err := findTheLastTimestamp(ctx, clients)
	if err != nil {
		return fmt.Errorf("get log timestamp error:%v", err)
	}

	if timeBefore == timeAfter {
		return fmt.Errorf("not resumed, please check")
	}

	return nil
}

func TestFreezerBasedContainerd(t *testing.T) {
	ctx := context.Background()
	clients, err := newClient()
	if err != nil {
		t.Fatalf("Init clients error:%v", err)
	}

	// First check the init state
	if err := isPausedState(ctx, clients); err != nil {
		t.Fatalf("Check init paused state error:%v", err)
	}

	// Second do request and check the state
	if err := requestShouldBeResumed(ctx, clients); err != nil {
		t.Fatalf("Check resume state error:%v", err)
	}

	// Third check the state after request
	if err := isPausedState(ctx, clients); err != nil {
		t.Fatalf("Check state after request error:%v", err)
	}
}
