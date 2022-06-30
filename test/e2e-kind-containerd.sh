#!/usr/bin/env bash

# Copyright 2022 The Knative Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script runs e2e tests on a local kind environment.
set -euo pipefail

CLUSTER_SUFFIX=${CLUSTER_SUFFIX:-cluster.local}

echo ">> Setup test resources"
kubectl patch configmap/config-deployment -n knative-serving --type merge -p '{"data":{"concurrencyStateEndpoint":"http://$HOST_IP:9696"}}'
ko apply -f test/config/sleeptalker.yaml
kubectl wait ksvc --timeout 300s --for=condition=Ready -n default sleeptalker

echo ">> Check init status, should be paused"
go test -race -count=1 -timeout=20m -tags=e2e-containerd ./test/e2e-kind-containerd/... -test.run Test_Freeze

echo ">> Check curl test, should be resumed"
go test -race -count=1 -timeout=20m -tags=e2e-containerd ./test/e2e-kind-containerd/... -test.run Test_Thaw

echo ">> check after curl, should be paused"
go test -race -count=1 -timeout=20m -tags=e2e-containerd ./test/e2e-kind-containerd/... -test.run Test_Freeze
