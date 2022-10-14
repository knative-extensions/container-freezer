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

# This script runs the end-to-end tests against Knative Serving built from source.
# It is started by prow for each PR. For convenience, it can also be executed manually.

# If you already have a Kubernetes cluster setup and kubectl pointing
# to it, call this script with the --run-tests arguments and it will use
# the cluster and run the tests.

# Calling this script without arguments will create a new cluster in
# project $PROJECT_ID, start knative in it, run the tests and delete the
# cluster.

source $(dirname $0)/../e2e-common.sh

export TIMEOUT=300

###############################################################################################
header "Running performance test"
# Skip installing istio as an add-on.
# Temporarily increasing the cluster size for serving tests to rule out
# resource/eviction as causes of flakiness.
initialize --skip-istio-addon --min-nodes=2 --max-nodes=2 --cluster-version=1.23 "$@"

function run_hey() {
  run_go_tool github.com/rakyll/hey hey "$@"
}

function operate_ksvc() {
  run_go_tool knative.dev/client/cmd/kn kn "$@"
}

function deploy_container_freezer() {
  kubectl label nodes --all knative.dev/container-runtime=containerd
  run_go_tool github.com/google/ko ko "$@"
}

function get_gateway_ip() {
  kubectl get svc "${GATEWAY_OVERRIDE}" -n "${GATEWAY_NAMESPACE_OVERRIDE}"  -oyaml | grep "ip:" | awk '{split($0, ip, ": ");  print ip[2]}'
}

function get_ksvc_url() {
  kubectl get ksvc -oyaml | grep url | awk '{split($0, url, "//");  print url[2]}' | tail -n 1
}

function wait_gateway_ip_ok() {
  end_time=$((SECONDS+TIMEOUT))
  while [ $SECONDS -lt $end_time ];do
    gateway_ip=$(get_gateway_ip)
    if [[ ${gateway_ip} != "" ]];then
      break
    fi
    sleep 1
  done
}

###############################################################################################
header "Wait test env ready"
mkdir -p "${ARTIFACTS}/hey"

setup_ingress_env_vars
wait_gateway_ip_ok

kubectl get pods -A
kubectl get svc -A

###############################################################################################
header "Running latency test when there is not container-freezer"
operate_ksvc service create nginx --image=nginx --port=80 --scale-max=1 --scale-min=1 --concurrency-limit=100 || \
  fail_test "deploy ksvc failed"

run_hey -n 30 -c 1 -host "$(get_ksvc_url)" \
  -o csv http://$(get_gateway_ip) >> "${ARTIFACTS}/hey/without_container_freezer.csv"|| \
  fail_test "run hey test failed"

operate_ksvc service delete nginx || \
  fail_test "delete ksvc failed"

###############################################################################################
header "Running latency test when container-freezer is deployed"
deploy_container_freezer apply -Rf $(dirname $0)/../../config || \
  fail_test "deploy container-freezer failed"

kubectl patch configmap/config-deployment -n knative-serving \
  --type merge -p '{"data":{"concurrencyStateEndpoint":"http://$HOST_IP:9696"}}'

operate_ksvc service create nginx --image=nginx --port=80 --scale-max=1 --scale-min=1 --concurrency-limit=100 || \
  fail_test "deploy ksvc failed"

run_hey -n 30 -c 1 -host "$(get_ksvc_url)" \
  -o csv http://$(get_gateway_ip) >> "${ARTIFACTS}/hey/with_container_freezer.csv" || \
  fail_test "run hey test failed"

success
