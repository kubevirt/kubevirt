#!/usr/bin/env bash
# This file is part of the KubeVirt project
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
#
# Copyright 2026 The KubeVirt Authors
#
# Utility functions for podman/docker builds

# Create buildah unshare context for rootless builds if needed (Podman only)
setup_buildah_context() {
  if [[ "${KUBEVIRT_CRI}" == "podman" ]]; then
    if ! podman system info | grep -q "rootless: false"; then
      echo "Running in rootless mode"
      export BUILDAH_ISOLATION=chroot
    fi
  fi
}

# Save image digest to file similar to Bazel
save_image_digest() {
  local image_name=$1
  local full_tag=$2
  local arch=$3

  mkdir -p ${DIGESTS_DIR}/${arch}/${image_name}

  # Get image digest
  digest=$(${KUBEVIRT_CRI} inspect ${full_tag} --format '{{.Digest}}' 2>/dev/null || echo "no-digest")
  echo "${digest}" >${DIGESTS_DIR}/${arch}/${image_name}/${image_name}.json.sha256
}

# Build multi-arch manifest
create_manifest() {
  local image_name=$1
  local tags=("$@")

  local manifest_name="${DOCKER_PREFIX}/${IMAGE_PREFIX}${image_name}:${DOCKER_TAG}"

  if [[ "${KUBEVIRT_CRI}" == "podman" ]]; then
    podman manifest create ${manifest_name}

    for tag in "${tags[@]}"; do
      podman manifest add ${manifest_name} ${tag}
    done
  elif [[ "${KUBEVIRT_CRI}" == "docker" ]]; then
    docker buildx imagetools create -t ${manifest_name} "${tags[@]}"
  fi

  echo "Created manifest: ${manifest_name}"
}
