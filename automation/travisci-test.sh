#!/bin/bash -e

# when not on a release do extensive checks
if [ -z "$TRAVIS_TAG" ]; then
	make bazel-build-verify

	# The make bazel-test might take longer then the current timeout for a command in Travis-CI of 10 min, so adding a keep alive loop while it runs
	while sleep 9m; do echo "Long running job - keep alive"; done &
	LOOP_PID=$!

	if [[ $TRAVIS_REPO_SLUG == "kubevirt/kubevirt" && $TRAVIS_CPU_ARCH == "amd64" ]]; then
		make goveralls
	else
		make bazel-test
	fi

	kill $LOOP_PID

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
