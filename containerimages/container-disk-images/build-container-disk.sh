#!/usr/bin/env bash
set -ex

SCRIPT_DIR="$(
    cd "$(dirname "$BASH_SOURCE[0]")"
    pwd
)"

image_name=$1
tag=$2
vm_image_file=$3

export CONTAINER_DISK_DOCKERFILE=${CONTAINER_DISK_DOCKERFILE:-$SCRIPT_DIR/Dockerfile.template}
readonly IMAGE_PLACEHOLDER="IMAGE"

sed s?$IMAGE_PLACEHOLDER?"${vm_image_file}"?g "$CONTAINER_DISK_DOCKERFILE" > "Dockerfile"

docker build -t "${image_name}:${tag}" .

rm -rf build
docker save --output "${image_name}-${tag}.tar" "${image_name}:${tag}"
