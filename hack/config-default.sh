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
kubevirtci_git_hash="2310230826-107a361"
conn_check_ipv4_address=${CONN_CHECK_IPV4_ADDRESS:-""}
conn_check_ipv6_address=${CONN_CHECK_IPV6_ADDRESS:-""}
conn_check_dns=${CONN_CHECK_DNS:-""}
migration_network_nic=${MIGRATION_NETWORK_NIC:-"eth1"}
infra_replicas=${KUBEVIRT_INFRA_REPLICAS:-0}
common_instancetypes_version=${COMMON_INSTANCETYPES_VERSION:-"v0.3.4"}
cluster_instancetypes_sha256=${CLUSTER_INSTANCETYPES_SHA256:-"80b6ba927a16e2bff8933ff6b0968158372b21633d38dc99573592c4a136073f"}
cluster_preferences_sha256=${CLUSTER_PREFERENCES_SHA256:-"f5bcffc1a94027564b2e69da19e3e4db3532446209ff05c7e5ffd90894bb3e31"}

# try to derive csv_version from docker tag. But it must start with x.y.z, without leading v
default_csv_version="${docker_tag/latest/0.0.0}"
default_csv_version="${default_csv_version/devel/0.0.0}"
[[ $default_csv_version == v* ]] && default_csv_version="${default_csv_version/v/}"
csv_version=${CSV_VERSION:-$default_csv_version}
