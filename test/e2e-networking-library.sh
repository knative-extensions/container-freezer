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

function is_ingress_class() {
  [[ "${INGRESS_CLASS}" == *"${1}"* ]]
}

function setup_ingress_env_vars() {
  if is_ingress_class istio; then
    export GATEWAY_OVERRIDE=istio-ingressgateway
    export GATEWAY_NAMESPACE_OVERRIDE=istio-system
  fi
  if is_ingress_class kourier; then
    export GATEWAY_OVERRIDE=kourier
    export GATEWAY_NAMESPACE_OVERRIDE=kourier-system
  fi
  if is_ingress_class contour; then
    export GATEWAY_OVERRIDE=envoy
    export GATEWAY_NAMESPACE_OVERRIDE=contour-external
  fi
  if is_ingress_class gateway-api; then
    export GATEWAY_OVERRIDE=istio-ingressgateway
    export GATEWAY_NAMESPACE_OVERRIDE=istio-system
  fi
}

