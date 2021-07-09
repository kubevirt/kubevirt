#!/usr/bin/env bash
set -e

./hack/coverage.sh
bazel run :goveralls
