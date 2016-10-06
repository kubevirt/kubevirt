#!/bin/bash

binaries="cmd/virt-controller cmd/virt-launcher"
docker_prefix=kubevirt
manifest_templates="`ls contrib/manifest/*.in`"
