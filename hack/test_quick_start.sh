#!/usr/bin/env bash

set -ex

if ${KUBECTL_BINARY} get CustomResourceDefinition consolequickstarts.console.openshift.io -o name; then
  echo "Check if the ConsoleQuickStart test-quick-start was deployed"
  QS_DISPLAY_NAME=$(${KUBECTL_BINARY} get ConsoleQuickStart test-quick-start -o json | jq -r ".spec.displayName")
  if [[ ${QS_DISPLAY_NAME} == "Test Quickstart Tour" ]]; then
    echo "ConsoleQuickStart test-quick-start successfully deployed"
  else
    echo "can't find ConsoleQuickStart test-quick-start"
    exit 1
  fi
else
  echo "This cluster does not support Quick Start; skipping quick-start test"
fi
