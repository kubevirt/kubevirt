VERSION=$(date +"%y%m%d%H%M")-$(git rev-parse --short HEAD)
# TODO: reenable ppc64le when new builds are available
ARCHITECTURES="amd64 arm64"
