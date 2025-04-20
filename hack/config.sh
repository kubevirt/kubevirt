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

unset binaries docker_images docker_tag docker_tag_alt image_prefix image_prefix_alt manifest_templates \
    namespace image_pull_policy verbosity \
    csv_version package_name

source ${KUBEVIRT_PATH}hack/config-default.sh
source ${KUBEVIRT_PATH}kubevirtci/cluster-up/hack/config.sh

export binaries docker_images docker_tag docker_tag_alt image_prefix image_prefix_alt manifest_templates \
    namespace image_pull_policy verbosity \
    csv_version package_name
