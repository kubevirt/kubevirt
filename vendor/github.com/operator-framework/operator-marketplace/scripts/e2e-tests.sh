#!/bin/bash
set -e

TEST_NAMESPACE="openshift-marketplace"

# Run the tests through the operator-sdk
echo "Running operator-sdk test"
operator-sdk test local ./test/e2e/ --no-setup --go-test-flags -v --namespace $TEST_NAMESPACE
