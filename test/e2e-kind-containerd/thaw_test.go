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

package e2e_kind_containerd

import (
	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_Thaw(t *testing.T) {
	time.Sleep(time.Second * 5)

	clients, err := NewClient()
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
	req, err := http.NewRequest("GET", reqUrl, nil)
	if err != nil {
		t.Fatalf("Create http request error:%v", err)
	}
	req.Host = ksvcUrl
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("Do request error:%v", err)
	}

	tickLogsByte, err := LogSleepTalkerPod(ctx, clients.KubeClient)
	if err != nil {
		t.Fatalf("Get sleeptalker pod's log error:%v", err)
	}
	lastTimestamp, err := FindTheLastTimestamp(tickLogsByte, t)
	if err != nil {
		t.Fatalf("Get log timestamp error:%v", err)
	}
	if time.Now().Sub(lastTimestamp) > time.Second {
		t.Fatalf("Not paused, Please check:%d", time.Now().Sub(lastTimestamp)/time.Second)
	}
}
