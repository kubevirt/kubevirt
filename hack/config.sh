#!/bin/bash

binaries="cmd/virt-controller cmd/virt-launcher cmd/virt-handler"
docker_prefix=kubevirt
docker_tag=latest
manifest_templates="`ls contrib/manifest/*.in`"
