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

package containerd

import (
	"context"
	"flag"
	"fmt"
	"net/http"
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
	ksvcUrl         = "sleeptalker.default.example.com"
	deployNamespace = "default"
	workerNodeName  = "kind-worker"
	lbsvcNamespace  = "kourier-system"
	lbsvcName       = "kourier"
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

func findTheLastTimestamp(logs []byte) (time.Time, error) {
	lines := strings.Split(string(logs), "\n")
	//The last line has \n, so last array element is empty, we choose the penultimate element
	lastLine := lines[len(lines)-2]
	// case: Ticking at: Wed Jun 29 15:42:54 2022
	timeStamp := strings.Split(lastLine, "at: ")
	return time.Parse(time.ANSIC, timeStamp[1])
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

	sleepTalkerPod := pods.Items[0]
	tickLogsByte, err := client.CoreV1().Pods(deployNamespace).
		GetLogs(sleepTalkerPod.Name, &v1.PodLogOptions{Container: "user-container"}).Do(ctx).Raw()
	if err != nil {
		return nil, fmt.Errorf("get pod:%s log error:%v", sleepTalkerPod.Name, err)
	}

	return tickLogsByte, nil
}

func IsPausedState() (bool, error) {
	time.Sleep(time.Second * 5)

	clients, err := newClient()
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	tickLogsByte, err := logSleepTalkerPod(ctx, clients.KubeClient)
	if err != nil {
		return false, err
	}
	lastTimestamp, err := findTheLastTimestamp(tickLogsByte)
	if err != nil {
		return false, err
	}
	//we wait 5 second before, so the time difference should not smaller than 5 seconds
	if time.Now().Sub(lastTimestamp) < time.Second*5 {
		return false, nil
	}

	return true, nil
}

func Test_InitShouldBePaused(t *testing.T) {
	var ok bool
	var err error
	if ok, err = IsPausedState(); err != nil {
		t.Fatalf("Check paused state error:%v", err)
	}

	if !ok {
		t.Fatalf("The init sate is not paused, please check it")
	}
}

func Test_RequestShouldBeResumed(t *testing.T) {
	time.Sleep(time.Second * 5)

	clients, err := newClient()
	if err != nil {
		t.Fatalf("Create client failed:%v", err)
	}

	ctx := context.Background()
	node, err := clients.KubeClient.CoreV1().Nodes().Get(ctx, workerNodeName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get worker node error:%v", err)
	}

	nodeIP := ""
	for _, v := range node.Status.Addresses {
		if v.Type == v1.NodeInternalIP {
			nodeIP = v.Address
		}
	}
	if nodeIP == "" {
		t.Fatalf("node ip is empty")
	}

	var nodePort int32
	svc, err := clients.KubeClient.CoreV1().Services(lbsvcNamespace).Get(ctx, lbsvcName, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Get lb svc error:%v", err)
	}
	for _, v := range svc.Spec.Ports {
		if v.Port == 80 {
			nodePort = v.NodePort
		}
	}
	if nodePort == 0 {
		t.Fatalf("Lb's nodePort is 0")
	}

	reqUrl := "http://" + nodeIP + ":" + strconv.Itoa(int(nodePort))
	t.Logf("Do request for:%s", reqUrl)
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		t.Fatalf("Create http request error:%v", err)
	}
	req.Host = ksvcUrl
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do request error:%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Response code is:%d", resp.StatusCode)
	}

	tickLogsByte, err := logSleepTalkerPod(ctx, clients.KubeClient)
	if err != nil {
		t.Fatalf("Get sleeptalker pod's log error:%v", err)
	}
	lastTimestamp, err := findTheLastTimestamp(tickLogsByte)
	if err != nil {
		t.Fatalf("Get log timestamp error:%v", err)
	}
	if time.Now().Sub(lastTimestamp) > time.Second {
		t.Fatalf("Not resumed, Please check:%d", time.Now().Sub(lastTimestamp)/time.Second)
	}
}

func Test_AfterRequestShouldBePaused(t *testing.T) {
	var ok bool
	var err error
	if ok, err = IsPausedState(); err != nil {
		t.Fatalf("Check paused state error:%v", err)
	}

	if !ok {
		t.Fatalf("After request finished, the sate is not paused, please check it")
	}
}
