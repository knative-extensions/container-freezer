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
	"flag"
	"fmt"
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

func NewClient() (*test.Clients, error) {
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

func FindTheLastTimestamp(logs []byte, t *testing.T) (time.Time, error) {
	lines := strings.Split(string(logs), "\n")
	//The last line has \n, so last array element is empty, we choose the penultimate element
	lastLine := lines[len(lines)-2]
	// case: Ticking at: Wed Jun 29 15:42:54 2022
	timeStamp := strings.Split(lastLine, "at: ")
	return time.Parse(time.ANSIC, timeStamp[1])
}

func LogSleepTalkerPod(ctx context.Context, client kubernetes.Interface) ([]byte, error) {
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
