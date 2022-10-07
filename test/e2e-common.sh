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

# Since default is istio, make default ingress as istio
export INGRESS_CLASS=${INGRESS_CLASS:-istio.ingress.networking.knative.dev}

function wait_podman_socket_exist() {
    while true
    do
        ls /run/podman/ -al
        if [ -S "/run/podman/podman.sock" ]
        then
            break
        fi
        sleep 1
    done
}
