DOCKER_PREFIX=${DOCKER_PREFIX:-"quay.io/kubevirt"}
DOCKER_IMAGE=${DOCKER_IMAGE:-"builder"}

# TODO: reenable ppc64le when new builds are available
ARCHITECTURES="amd64 arm64"
