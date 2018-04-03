#!/bin/bash
set -e

source hack/common.sh

shfmt -i 4 -w ${KUBEVIRT_DIR}/cluster/ ${KUBEVIRT_DIR}/hack/ ${KUBEVIRT_DIR}/images/
goimports -w -local kubevirt.io ${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/tests/ ${KUBEVIRT_DIR}/tools
(cd ${KUBEVIRT_DIR} && go vet ./cmd/... ./pkg/... ./tests/... ./tools/...)
