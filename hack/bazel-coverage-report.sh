#!/usr/bin/env bash
set -e

./hack/coverage.sh ${WHAT}
bazel run :coverage-report
