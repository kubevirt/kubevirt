binaries="cmd/virt-operator cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/fake-qemu-process cmd/virt-api cmd/subresource-access-test cmd/example-hook-sidecar cmd/example-cloudinit-hook-sidecar"
docker_images="cmd/virt-operator cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api images/disks-images-provider images/vm-killer images/nfs-server cmd/subresource-access-test images/winrmcli cmd/example-hook-sidecar cmd/example-cloudinit-hook-sidecar images/cdi-http-import-server"
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
kubevirtci_git_hash="426443f01f541258d17d8abd005c1f9296f614d6"

# try to derive csv_version from docker tag. But it must start with x.y.z, without leading v
default_csv_version="${docker_tag/latest/0.0.0}"
default_csv_version="${default_csv_version/devel/0.0.0}"
[[ $default_csv_version == v* ]] && default_csv_version="${default_csv_version/v/}"
csv_version=${CSV_VERSION:-$default_csv_version}
