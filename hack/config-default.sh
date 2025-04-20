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

binaries_and_docker_images="
    cmd/virt-operator
    cmd/virt-controller
    cmd/virt-launcher
    cmd/virt-handler
    cmd/virt-api
    cmd/sidecars
    cmd/sidecars/smbios
    cmd/sidecars/cloudinit
"

binaries="
    ${binaries_and_docker_images}
    cmd/virtctl
    cmd/fake-qemu-process
    cmd/virt-chroot
    cmd/virt-tail
"

docker_images="
    ${binaries_and_docker_images}
    images/disks-images-provider
    images/vm-killer
    images/winrmcli
    tests/conformance
"

docker_tag=${DOCKER_TAG:-latest}
docker_tag_alt=${DOCKER_TAG_ALT}
image_prefix=${IMAGE_PREFIX}
image_prefix_alt=${IMAGE_PREFIX_ALT}
namespace=${KUBEVIRT_INSTALLED_NAMESPACE:-kubevirt}
deploy_testing_infra=${DEPLOY_TESTING_INFRA:-false}
csv_namespace=placeholder
cdi_namespace=cdi
image_pull_policy=${IMAGE_PULL_POLICY:-IfNotPresent}
verbosity=${VERBOSITY:-2}
package_name=${PACKAGE_NAME:-kubevirt-dev}
kubevirtci_git_hash="2504171552-a558e3fe"
conn_check_ipv4_address=${CONN_CHECK_IPV4_ADDRESS:-""}
conn_check_ipv6_address=${CONN_CHECK_IPV6_ADDRESS:-""}
conn_check_dns=${CONN_CHECK_DNS:-""}
migration_network_nic=${MIGRATION_NETWORK_NIC:-"eth1"}
infra_replicas=${KUBEVIRT_INFRA_REPLICAS:-0}
test_image_replicas=${KUBEVIRT_E2E_PARALLEL_NODES:-6}
base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
kubevirt_test_config=${KUBEVIRT_TEST_CONFIG:-$(if [[ "${KUBEVIRT_STORAGE:-}" == rook-ceph* ]]; then echo "${base_dir}/tests/default-ceph-config.json"; else echo "${base_dir}/tests/default-config.json"; fi)}

# try to derive csv_version from docker tag. But it must start with x.y.z, without leading v
default_csv_version="${docker_tag/latest/0.0.0}"
default_csv_version="${default_csv_version/devel/0.0.0}"
[[ $default_csv_version == v* ]] && default_csv_version="${default_csv_version/v/}"
csv_version=${CSV_VERSION:-$default_csv_version}
