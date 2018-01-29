binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virtctl cmd/fake-qemu-process cmd/virt-dhcp cmd/fake-dnsmasq-process"
docker_images="cmd/virt-controller cmd/virt-launcher cmd/virt-handler images/iscsi-demo-target-tgtd images/vm-killer images/libvirt-kubevirt cmd/virt-migrator cmd/registry-disk-v1alpha images/cirros-registry-disk-demo cmd/virt-dhcp images/fedora-cloud-registry-disk-demo"
docker_prefix=kubevirt
docker_tag=${DOCKER_TAG:-latest}
master_ip=192.168.200.2
network_provider=weave
kubeconfig=cluster/vagrant/.kubeconfig
