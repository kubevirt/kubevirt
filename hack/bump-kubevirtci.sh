#!/usr/bin/env bash

set -ex

KUBEVIRTCI_TAG=$(curl -L -Ss https://storage.googleapis.com/kubevirt-prow/release/kubevirt/kubevirtci/latest)

for file in cluster/kubevirtci.sh automation/nightly/test-nightly-build.sh; do
  if ! grep '^export KUBEVIRTCI_TAG=' "${file}" | grep "${KUBEVIRTCI_TAG}"; then
    sed -i -E "s|(^export KUBEVIRTCI_TAG=).*$|\1\${KUBEVIRTCI_TAG:-\"$KUBEVIRTCI_TAG\"}|g" ${file}
  fi
done

