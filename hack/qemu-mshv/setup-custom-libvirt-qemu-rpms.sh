#!/bin/bash

#set -e

while getopts q:l: flag; do
    case "${flag}" in
    q) QEMU_IMAGE=${OPTARG} ;;
    l) LIBVIRT_IMAGE=${OPTARG} ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
    esac
done

if [ -z "$QEMU_IMAGE" ] || [ -z "$LIBVIRT_IMAGE" ]; then
    echo "Usage: $0 -q <QEMU_IMAGE> -l <LIBVIRT_IMAGE>"
    exit 1
fi

# Start libvirt repo HTTP server (port 8080) if not running
if [ "$(docker ps -q -f name=libvirt-rpms-http-server)" ]; then
    echo "Libvirt RPM server already running"
else
    docker run --rm -dit \
        --name libvirt-rpms-http-server \
        -p 8080:80 \
        $LIBVIRT_IMAGE
    sleep 5
fi

# Start qemu repo HTTP server (port 9090) if not running
if [ "$(docker ps -q -f name=qemu-rpms-http-server)" ]; then
    echo "QEMU RPM server already running"
else
    docker run --rm -dit \
        --name qemu-rpms-http-server \
        -p 9090:80 \
        $QEMU_IMAGE
    sleep 5
fi

docker images --digests

LIBVIRT_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' libvirt-rpms-http-server)
QEMU_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' qemu-rpms-http-server)
echo "Libvirt repo IP: $LIBVIRT_IP"
echo "QEMU repo IP:    $QEMU_IP"

# Verify both repos (use their mapped host ports)
curl -f "http://localhost:8080/x86_64/repodata/repomd.xml" || {
    echo 'Libvirt repo not accessible'
    exit 1
}
curl -f "http://localhost:9090/x86_64/repodata/repomd.xml" || {
    echo 'QEMU repo not accessible'
    exit 1
}

# Extract versions (tolerate missing build-info fields)
LIBVIRT_VERSION=$(curl -s "http://localhost:8080/build-info.json" | jq -r '.libvirt_version // empty') || true
QEMU_VERSION=$(curl -s "http://localhost:9090/build-info.json" | jq -r '.qemu_version // empty') || true
echo "Detected libvirt version: ${LIBVIRT_VERSION:-<none>}"
echo "Detected qemu version:    ${QEMU_VERSION:-<none>}"

# Build combined repo descriptor so rpm-deps sees both
cat >custom-repo.yaml <<EOF
repositories:
- arch: x86_64
  baseurl: http://$LIBVIRT_IP:80/x86_64/
  name: custom-libvirt
  gpgcheck: 0
  repo_gpgcheck: 0
- arch: x86_64
  baseurl: http://$QEMU_IP:80/x86_64/
  name: custom-qemu
  gpgcheck: 0
  repo_gpgcheck: 0
EOF

echo "Combined custom-repo.yaml:"
cat custom-repo.yaml

make CUSTOM_REPO=custom-repo.yaml LIBVIRT_VERSION="$LIBVIRT_VERSION" QEMU_VERSION="$QEMU_VERSION" SINGLE_ARCH="x86_64" rpm-deps

ls -l /tmp/123455

echo "rpm-deps completed with custom libvirt & qemu"

#make bazel-build-images
