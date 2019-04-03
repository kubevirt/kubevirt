#!/bin/bash
set -e

TEST_NAMESPACE="openshift-marketplace"

# Run the tests through the operator-sdk
echo "Running operator-sdk test"
operator-sdk test local ./test/e2e/ --no-setup --namespace $TEST_NAMESPACE
