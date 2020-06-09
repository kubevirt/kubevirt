#!/bin/bash -e


make generate
if [[ -n "$(git status --porcelain)" ]] ; then
    echo "It seems like you need to run 'make generate'. Please run it and commit the changes"
    git status --porcelain; false
fi

if diff <(git grep -c '') <(git grep -cI '') | egrep -v -e 'docs/.*\.png|swagger-ui' -e 'vendor/*' -e 'assets/*' | grep '^<'; then
    echo "Binary files are present in git repostory."; false
fi

make

if [[ -n "$(git status --porcelain)" ]] ; then
    echo "It seems like you need to run 'make'. Please run it and commit the changes"; git status --porcelain; false
fi

make build-verify # verify that we set version on the packages built by bazel

# The make bazel-test might take longer then the current timeout for a command in Travis-CI of 10 min, so adding a keep alive loop while it runs
while sleep 9m; do echo "Long running job - keep alive"; done & LOOP_PID=$!

if [[ $TRAVIS_REPO_SLUG == "kubevirt/kubevirt" && $TRAVIS_CPU_ARCH == "amd64" ]]; then
    make goveralls
else
    make bazel-test
fi

kill $LOOP_PID

make build-verify # verify that we set version on the packages built by go(goveralls depends on go-build target)
make apidocs
make client-python
make manifests DOCKER_PREFIX="docker.io/kubevirt" DOCKER_TAG=$TRAVIS_TAG # skip getting old CSVs here (no QUAY_REPOSITORY), verification might fail because of stricter rules over time; falls back to latest if not on a tag
make olm-verify
if [[ $TRAVIS_CPU_ARCH == "amd64" ]]; then
    make prom-rules-verify
fi

