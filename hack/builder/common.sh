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
# Copyright The KubeVirt Authors.
#

DOCKER_PREFIX=${DOCKER_PREFIX:-"quay.io/kubevirt"}
DOCKER_IMAGE=${DOCKER_IMAGE:-"builder"}
DOCKER_CROSS_IMAGE=${DOCKER_CROSS_IMAGE:-"builder-cross"}

# TODO: reenable ppc64le when new builds are available
ARCHITECTURES="amd64 arm64 s390x"
