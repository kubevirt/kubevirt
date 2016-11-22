#!/bin/bash

binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler cmd/virt-api"
docker_images="$binaries contrib/haproxy"
docker_prefix=kubevirt
docker_tag=latest
manifest_templates="`ls contrib/manifest/*.in`"
