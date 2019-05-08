binaries="cmd/virt-operator cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/fake-qemu-process cmd/virt-api cmd/subresource-access-test cmd/example-hook-sidecar cmd/example-cloudinit-hook-sidecar"
docker_images="cmd/virt-operator cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api images/disks-images-provider images/vm-killer images/nfs-server cmd/subresource-access-test images/winrmcli cmd/example-hook-sidecar cmd/example-cloudinit-hook-sidecar images/cdi-http-import-server cmd/container-disk-v1alpha"
docker_tag=${DOCKER_TAG:-latest}
docker_tag_alt=${DOCKER_TAG_ALT}
namespace=kubevirt
cdi_namespace=cdi
image_pull_policy=${IMAGE_PULL_POLICY:-IfNotPresent}
verbosity=${VERBOSITY:-2}
package_name=${PACKAGE_NAME:-kubevirt-dev}
kubevirtci_git_hash="45cdf80da619866cdab36295de25a8aeea7eccbc"

# try to derive csv_version from docker tag. But it must start with x.y.z, without leading v
default_csv_version="${docker_tag/latest/0.0.0}"
default_csv_version="${default_csv_version/devel/0.0.0}"
[[ $default_csv_version == v* ]] && default_csv_version="${default_csv_version/v/}"
csv_version=${CSV_VERSION:-$default_csv_version}
