#!/bin/bash

set -e

source $(dirname "$0")/common.sh

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}
/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh
(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go build)
goimports -w -local kubevirt.io ${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/tests/
${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json
