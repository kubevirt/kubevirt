#!/bin/bash -xe

bazelisk run //cmd/example-interface-hook-sidecar:example-interface-hook-sidecar-image
container=bazel/cmd/example-interface-hook-sidecar:example-interface-hook-sidecar-image
dnscont=k8s-1.18-dnsmasq
port=$(docker port $dnscont 5000 | awk -F : '{ print $2 }')
image="localhost:$port/kubevirt/example-interface-hook-sidecar:devel"
echo "Tag $container as $image"
docker tag $container $image
echo "Push image to local registry"
docker push $image
