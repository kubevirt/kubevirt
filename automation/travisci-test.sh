#!/bin/bash -e

export TIMESTAMP=${TIMESTAMP:-1}

# when not on a release do extensive checks
if [ -z "$TRAVIS_TAG" ]; then
	make bazel-build-verify
else
	make
fi

make build-verify # verify that we set version on the packages built by go(goveralls depends on go-build target)
make apidocs
make client-python
make manifests DOCKER_PREFIX="docker.io/kubevirt" DOCKER_TAG=$TRAVIS_TAG # skip getting old CSVs here (no QUAY_REPOSITORY), verification might fail because of stricter rules over time; falls back to latest if not on a tag
make olm-verify
if [[ $TRAVIS_CPU_ARCH == "amd64" ]]; then
	make prom-rules-verify
fi
