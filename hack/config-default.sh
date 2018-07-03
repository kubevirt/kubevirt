binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/fake-qemu-process cmd/virt-api cmd/subresource-access-test"
docker_images="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api images/disks-images-provider images/vm-killer cmd/registry-disk-v1alpha images/cirros-registry-disk-demo images/fedora-cloud-registry-disk-demo images/alpine-registry-disk-demo cmd/subresource-access-test images/winrmcli"
docker_prefix=${DOCKER_PREFIX:-kubevirt}
docker_tag=${DOCKER_TAG:-latest}
master_ip=192.168.200.2
network_provider=flannel
namespace=kube-system
