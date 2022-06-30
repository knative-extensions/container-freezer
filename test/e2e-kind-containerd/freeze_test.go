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
	"testing"
	"time"
)

func Test_Freeze(t *testing.T) {
	time.Sleep(time.Second * 5)

	clients, err := NewClient()
	if err != nil {
		t.Fatalf("Create client failed:%v", err)
	}

	ctx := context.Background()
	tickLogsByte, err := LogSleepTalkerPod(ctx, clients.KubeClient)
	if err != nil {
		t.Fatalf("Get sleeptalker pod's log error:%v", err)
	}
	lastTimestamp, err := FindTheLastTimestamp(tickLogsByte, t)
	if err != nil {
		t.Fatalf("Get log timestamp error:%v", err)
	}
	//we wait 5 second before, so the time difference should not smaller than 5 seconds
	if time.Now().Sub(lastTimestamp) < time.Second*5 {
		t.Errorf("Not paused, Please check:%d", time.Now().Sub(lastTimestamp)/time.Second)
	}
}
