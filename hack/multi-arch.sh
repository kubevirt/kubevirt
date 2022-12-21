#!/usr/bin/env bash

CONTAINER_BUILD="${CONTAINER_BUILD:-docker}"
image_list="virt-controller virt-handler virt-launcher virt-api virt-operator"

function get_tag {
    local tag
    tag=${DOCKER_TAG}
    # Strip trailing $arch from the tag
    for arch in ${BUILD_ARCH//,/ }; do
	tag=${tag%"-$arch"}
    done
    echo $tag
}

for image in ${image_list}; do
    amend=""
    for arch in ${BUILD_ARCH//,/ }; do
	tag="$(get_tag)"
	amend+=" --amend $DOCKER_PREFIX/$image:$tag-$arch"
    done

    cmd="$CONTAINER_BUILD manifest create $DOCKER_PREFIX/$image:$tag-multi-arch $amend"
    echo $cmd
    $cmd
done
