#!/usr/bin/env bash

set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

#Creating temp dir

CACHE_DIR=$(mktemp -d)

export BAZEL_CACHE_DIR=$CACHE_DIR

#Bulding with the bazel cache enabled
bazel build \
    --config=${ARCHITECTURE} \
    --disk_cache=$BAZEL_CACHE_DIR \
    //...

#Cleaning up the temp dir
rm -rf $CACHE_DIR
