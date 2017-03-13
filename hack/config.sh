binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api"
docker_images="$binaries images/haproxy images/libvirtd images/iscsi-demo-target-tgtd images/cockpit"
docker_prefix=kubevirt
docker_tag=${DOCKER_TAG:-latest}
manifest_templates="`ls manifests/*.in`"
master_ip=192.168.200.2
master_port=8184
network_provider=weave
