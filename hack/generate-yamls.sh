#! /usr/bin/env bash
#
# This script builds all the YAMLs that Knative container-freezer publishes.
# It may be varied between different branches, of what it does, but the
# following usage must be observed:
#
# generate-yamls.sh  <repo-root-dir> <generated-yaml-list>
#     repo-root-dir         the root directory of the repository.
#     generated-yaml-list   an output file that will contain the list of all
#                           YAML files. The first file listed must be our
#                           manifest that contains all images to be tagged.

# Different versions of our scripts should be able to call this script with
# such assumption so that the test/publishing/tagging steps can evolve
# differently than how the YAMLs are built.

# The following environment variables affect the behavior of this script:
# * `$KO_FLAGS` Any extra flags that will be passed to ko.
# * `$YAML_OUTPUT_DIR` Where to put the generated YAML files, otherwise a
#   random temporary directory will be created. **All existing YAML files in
#   this directory will be deleted.**
# * `$KO_DOCKER_REPO` If not set, use ko.local as the registry.

set -o errexit
set -o pipefail

readonly YAML_REPO_ROOT=${1:?"First argument must be the repo root dir"}
readonly YAML_LIST_FILE=${2:?"Second argument must be the output file"}
readonly YAML_ENV_FILE=${3:-$(mktemp)}

# Set output directory
if [[ -z "${YAML_OUTPUT_DIR:-}" ]]; then
  readonly YAML_OUTPUT_DIR="$(mktemp -d)"
fi
rm -fr ${YAML_OUTPUT_DIR}/*.yaml

# Generated Knative component YAML files
readonly CONTAINER_FREEZER_YAML=${YAML_OUTPUT_DIR}/container-freezer.yaml

# Flags for all ko commands
KO_YAML_FLAGS="-P"
KO_FLAGS="${KO_FLAGS:-}"
[[ "${KO_DOCKER_REPO}" != gcr.io/* ]] && KO_YAML_FLAGS=""

if [[ "${KO_FLAGS}" != *"--platform"* ]]; then
  KO_YAML_FLAGS="${KO_YAML_FLAGS} --platform=all"
fi

readonly KO_YAML_FLAGS="${KO_YAML_FLAGS} ${KO_FLAGS}"

if [[ -n "${TAG:-}" ]]; then
  LABEL_YAML_CMD=(sed -e "s|serving.knative.dev/release: devel|serving.knative.dev/release: \"${TAG}\"|" -e "s|app.kubernetes.io/version: devel|app.kubernetes.io/version: \"${TAG:1}\"|")
else
  LABEL_YAML_CMD=(cat)
fi

: ${KO_DOCKER_REPO:="ko.local"}
export KO_DOCKER_REPO

cd "${YAML_REPO_ROOT}"

echo "Building Knative Container-Freezer"
ko resolve ${KO_YAML_FLAGS} -f config/ | "${LABEL_YAML_CMD[@]}" > "${CONTAINER_FREEZER_YAML}"

echo "All manifests generated"
