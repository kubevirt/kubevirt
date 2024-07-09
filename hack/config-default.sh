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
    images/nfs-server
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
kubevirtci_git_hash="2407081227-7c042242"
conn_check_ipv4_address=${CONN_CHECK_IPV4_ADDRESS:-""}
conn_check_ipv6_address=${CONN_CHECK_IPV6_ADDRESS:-""}
conn_check_dns=${CONN_CHECK_DNS:-""}
migration_network_nic=${MIGRATION_NETWORK_NIC:-"eth1"}
infra_replicas=${KUBEVIRT_INFRA_REPLICAS:-0}
common_instancetypes_version=${COMMON_INSTANCETYPES_VERSION:-"v1.0.1"}
cluster_instancetypes_sha256=${CLUSTER_INSTANCETYPES_SHA256:-"ebd37dae0ed07ae1a79bb09b27b4fe8b971ff371ee900119e1906e1574ecf829"}
cluster_preferences_sha256=${CLUSTER_PREFERENCES_SHA256:-"fa7e6cb6b21116d4ef200f4e9e774e38f26862055f62a23c7f926610f0d25ba4"}

# try to derive csv_version from docker tag. But it must start with x.y.z, without leading v
default_csv_version="${docker_tag/latest/0.0.0}"
default_csv_version="${default_csv_version/devel/0.0.0}"
[[ $default_csv_version == v* ]] && default_csv_version="${default_csv_version/v/}"
csv_version=${CSV_VERSION:-$default_csv_version}
