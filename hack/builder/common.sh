DOCKER_PREFIX=${DOCKER_PREFIX:-"quay.io/kubevirt"}
DOCKER_IMAGE=${DOCKER_IMAGE:-"builder"}
DOCKER_CROSS_IMAGE=${DOCKER_CROSS_IMAGE:-"builder-cross"}

# TODO: reenable ppc64le when new builds are available
ARCHITECTURES=${ARCHITECTURES:-"amd64 arm64 s390x"}
