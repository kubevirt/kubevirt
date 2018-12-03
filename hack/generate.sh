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

(cd ${KUBEVIRT_DIR}/tools/resource-generator/ && go build)
rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/cluster/examples/*
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmi >${KUBEVIRT_DIR}/manifests/generated/vmi-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmirs >${KUBEVIRT_DIR}/manifests/generated/vmirs-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmipreset >${KUBEVIRT_DIR}/manifests/generated/vmipreset-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vm >${KUBEVIRT_DIR}/manifests/generated/vm-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=vmim >${KUBEVIRT_DIR}/manifests/generated/vmim-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=rbac --namespace=${namespace} >${KUBEVIRT_DIR}/manifests/generated/rbac.authorization.k8s.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=prometheus --namespace=${namespace} >${KUBEVIRT_DIR}/manifests/generated/prometheus.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-api --namespace=${namespace} --repository=${docker_prefix} --version=${docker_tag} --pullPolicy=${image_pull_policy} >${KUBEVIRT_DIR}/manifests/generated/virt-api.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-controller --namespace=${namespace} --repository=${docker_prefix} --version=${docker_tag} --pullPolicy=${image_pull_policy} >${KUBEVIRT_DIR}/manifests/generated/virt-controller.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-handler --namespace=${namespace} --repository=${docker_prefix} --version=${docker_tag} --pullPolicy=${image_pull_policy} >${KUBEVIRT_DIR}/manifests/generated/virt-handler.yaml

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go build)
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --generated-vms-dir=${KUBEVIRT_DIR}/cluster/examples

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=info:pkg/hooks/info pkg/hooks/info/api.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=v1alpha:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api.proto
