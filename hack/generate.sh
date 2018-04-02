#!/bin/bash

set -e

source $(dirname "$0")/common.sh

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}
/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh
(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go build)
goimports -w -local kubevirt.io ${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/tests/

tmp_file=$(mktemp)
${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path $tmp_file

# Strip out generation tags from descriptions.
sed -e 's#\+k8s:openapi-gen=.*"#"#g' \
    -e 's#\+k8s:deepcopy-gen:.*"#"#g' \
    $tmp_file >${KUBEVIRT_DIR}/api/openapi-spec/swagger.json
rm -rf $tmp_file

(cd ${KUBEVIRT_DIR}/tools/crd-generator/ && go build)
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vm >${KUBEVIRT_DIR}/manifests/generated/vm-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vmrs >${KUBEVIRT_DIR}/manifests/generated/vmrs-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vmpreset >${KUBEVIRT_DIR}/manifests/generated/vmpreset-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=ovm >${KUBEVIRT_DIR}/manifests/generated/ovm-resource.yaml
