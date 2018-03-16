binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/fake-qemu-process"
docker_images="cmd/virt-controller cmd/virt-launcher cmd/virt-handler images/iscsi-demo-target-tgtd images/vm-killer cmd/registry-disk-v1alpha images/cirros-registry-disk-demo images/fedora-cloud-registry-disk-demo images/alpine-registry-disk-demo"
docker_prefix=kubevirt
docker_tag=${DOCKER_TAG:-latest}
master_ip=192.168.200.2
network_provider=flannel
kubeconfig=cluster/vagrant/.kubeconfig
namespace=kube-system
