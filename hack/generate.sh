#!/bin/bash

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}
/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh
(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go build)
goimports -w -local kubevirt.io ${KUBEVIRT_DIR}/cmd/ ${KUBEVIRT_DIR}/pkg/ ${KUBEVIRT_DIR}/tests/

${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json

(cd ${KUBEVIRT_DIR}/tools/crd-generator/ && go build)
rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/cluster/examples/*
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vmi >${KUBEVIRT_DIR}/manifests/generated/vmi-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vmirs >${KUBEVIRT_DIR}/manifests/generated/vmirs-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vmipreset >${KUBEVIRT_DIR}/manifests/generated/vmipreset-resource.yaml
${KUBEVIRT_DIR}/tools/crd-generator/crd-generator --crd-type=vm >${KUBEVIRT_DIR}/manifests/generated/vm-resource.yaml

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go build)
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --generated-vms-dir=${KUBEVIRT_DIR}/cluster/examples

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=info:pkg/hooks/info pkg/hooks/info/api.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=v1alpha:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api.proto
