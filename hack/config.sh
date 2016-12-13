binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api"
docker_images="$binaries images/haproxy"
docker_prefix=kubevirt
docker_tag=latest
manifest_templates="`ls cluster/manifest/*.in`"
master_ip=192.168.200.2
network_provider=weave
