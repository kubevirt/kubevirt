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
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${KUBEVIRT_DIR}/manifests/generated/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace={{.Namespace}} --pullPolicy={{.ImagePullPolicy}} >${KUBEVIRT_DIR}/manifests/generated/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=rbac --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/rbac.authorization.k8s.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=prometheus --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/prometheus.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-api --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version={{.DockerTag}} --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-api.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-controller --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version={{.DockerTag}} --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-controller.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=virt-handler --namespace={{.Namespace}} --repository={{.DockerPrefix}} --version={{.DockerTag}} --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} >${KUBEVIRT_DIR}/manifests/generated/virt-handler.yaml.in

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go build)
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --generated-vms-dir=${KUBEVIRT_DIR}/cluster/examples

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=info:pkg/hooks/info pkg/hooks/info/api.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=v1alpha:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api.proto
protoc --proto_path=pkg/hooks/v1alpha2 --go_out=plugins=grpc,import_path=v1alpha2:pkg/hooks/v1alpha2 pkg/hooks/v1alpha2/api.proto
