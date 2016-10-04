#!/bin/bash

binaries="pkg/virt-controller pkg/virt-launcher"
docker_prefix=kubevirt
manifest_templates="`ls contrib/manifest/*.in`"
