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

# This script provides helper methods to perform cluster actions.
# shellcheck disable=SC1090
source "$(dirname "${BASH_SOURCE[0]}")/../vendor/knative.dev/hack/e2e-tests.sh"
source "$(dirname "${BASH_SOURCE[0]}")/e2e-networking-library.sh"

# Since default is kourier, make default ingress as kourier
export INGRESS_CLASS=${INGRESS_CLASS:-kourier.ingress.networking.knative.dev}

# Searches for the latest version for serving
function latest_serving_version() {
  curl -L --silent "https://api.github.com/repos/knative/serving/releases" | jq -r '[.[].tag_name] | map(select(.)) | sort_by( sub("knative-";"") | sub("v";"") | split(".") | map(tonumber) ) | reverse[0]'
}

# Searches for the latest version for kourier
function latest_net_kourier_version() {
  curl -L --silent "https://api.github.com/repos/knative/net-kourier/releases" | jq -r '[.[].tag_name] | map(select(.)) | sort_by( sub("knative-";"") | sub("v";"") | split(".") | map(tonumber) ) | reverse[0]'
}

# Latest serving release. If user does not supply this as a flag, the latest
# tagged release will be used.
export LATEST_SERVING_RELEASE_VERSION=$(latest_serving_version)

# Latest net-kourier release.
export LATEST_NET_KOURIER_RELEASE_VERSION=$(latest_net_kourier_version)

export SERVING_URL_RELEASE_PREFIX="https://github.com/knative/serving/releases/download"
export KOURIER_URL_RELEASE_PREFIX="https://github.com/knative/net-kourier/releases/download"

function wait_podman_socket_exist() {
  while true
  do
    ls /run/podman/ -al
    if [ -S "/run/podman/podman.sock" ];then
      break
    fi
    sleep 1
  done
}

function knative_setup() {
  local url_serving="${SERVING_URL_RELEASE_PREFIX}/${LATEST_SERVING_RELEASE_VERSION}"
  local url_kourier="${KOURIER_URL_RELEASE_PREFIX}/${LATEST_NET_KOURIER_RELEASE_VERSION}"

  # Install serving
  kubectl apply -f "${url_serving}/serving-crds.yaml"
  kubectl apply -f "${url_serving}/serving-core.yaml"

  # Install kourier
  kubectl apply -f "${url_kourier}/kourier.yaml"
  kubectl patch configmap/config-network \
    --namespace knative-serving \
    --type merge \
    --patch '{"data":{"ingress-class":"kourier.ingress.networking.knative.dev"}}'
}

function knative_teardown() {
  local url_serving="${SERVING_URL_RELEASE_PREFIX}/${LATEST_SERVING_RELEASE_VERSION}"
  local url_kourier="${KOURIER_URL_RELEASE_PREFIX}/${LATEST_NET_KOURIER_RELEASE_VERSION}"

  # Uninstall kourier
  kubectl delete -f "${url_kourier}/kourier.yaml"

  # Uninstall serving
  kubectl delete -f "${url_serving}/serving-core.yaml"
  kubectl delete -f "${url_serving}/serving-crds.yaml"
}
