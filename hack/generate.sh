#!/usr/bin/env bash

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

# generate clients
CLIENT_GEN_BASE=kubevirt.io/client-go/generated
rm -rf ${KUBEVIRT_DIR}/staging/src/${CLIENT_GEN_BASE}

# KubeVirt stuff
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go/apis/snapshot/v1alpha1/types.go

deepcopy-gen --input-dirs kubevirt.io/client-go/apis/snapshot/v1alpha1 \
    --bounding-dirs kubevirt.io/client-go/apis \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

openapi-gen --input-dirs kubevirt.io/client-go/apis/snapshot/v1alpha1,k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,kubevirt.io/client-go/api/v1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package kubevirt.io/client-go/apis/snapshot/v1alpha1 \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name versioned \
    --input-base kubevirt.io/client-go/apis \
    --input snapshot/v1alpha1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/kubevirt/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

# dependencies
client-gen --clientset-name versioned \
    --input-base kubevirt.io/containerized-data-importer/pkg/apis \
    --input core/v1alpha1,upload/v1alpha1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/containerized-data-importer/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name versioned \
    --input-base github.com/coreos/prometheus-operator/pkg/apis \
    --input monitoring/v1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/prometheus-operator/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name versioned \
    --input-base github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis \
    --input k8s.cni.cncf.io/v1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/network-attachment-definition-client/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name versioned \
    --input-base github.com/kubernetes-csi/external-snapshotter/v2/pkg/apis \
    --input volumesnapshot/v1beta1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/external-snapshotter/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}
/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh
(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go_build)

${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json

(cd ${KUBEVIRT_DIR}/tools/resource-generator/ && go_build)
(cd ${KUBEVIRT_DIR}/tools/csv-generator/ && go_build)
rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/examples/*
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=priorityclass >${KUBEVIRT_DIR}/manifests/generated/kubevirt-priority-class.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${KUBEVIRT_DIR}/manifests/generated/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace={{.Namespace}} --pullPolicy={{.ImagePullPolicy}} >${KUBEVIRT_DIR}/manifests/generated/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=operator-rbac --namespace={{.Namespace}} >${KUBEVIRT_DIR}/manifests/generated/rbac-operator.authorization.k8s.yaml.in

# used for Image fields in manifests
function getVersion() {
    echo "{{if $1}}@{{$1}}{{else}}:{{.DockerTag}}{{end}}"
}
virtapi_version=$(getVersion ".VirtApiSha")
virtcontroller_version=$(getVersion ".VirtControllerSha")
virthandler_version=$(getVersion ".VirtHandlerSha")
virtlauncher_version=$(getVersion ".VirtLauncherSha")
virtoperator_version=$(getVersion ".VirtOperatorSha")

# used as env var for operator
function getShasum() {
    echo "{{if $1}}@{{$1}}{{end}}"
}

# without the '@' symbole used in 'getShasum'
function getRawShasum() {
    echo "{{if $1}}{{$1}}{{end}}"
}
virtapi_sha=$(getShasum ".VirtApiSha")
virtcontroller_sha=$(getShasum ".VirtControllerSha")
virthandler_sha=$(getShasum ".VirtHandlerSha")
virtlauncher_sha=$(getShasum ".VirtLauncherSha")

virtapi_rawsha=$(getRawShasum ".VirtApiSha")
virtcontroller_rawsha=$(getRawShasum ".VirtControllerSha")
virthandler_rawsha=$(getRawShasum ".VirtHandlerSha")
virtlauncher_rawsha=$(getRawShasum ".VirtLauncherSha")

# The generation code for CSV requires a valid semver to be used.
# But we're trying to generate a template for a CSV here from code
# rather than an actual usable CSV. To work around this, we set the
# versions to something absurd and do a find/replace with our templated
# values after the file is generated.
_fake_replaces_csv_version="1111.1111.1111"
_fake_csv_version="2222.2222.2222"
${KUBEVIRT_DIR}/tools/csv-generator/csv-generator --namespace={{.CSVNamespace}} --dockerPrefix={{.DockerPrefix}} --operatorImageVersion="$virtoperator_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} --apiSha="$virtapi_rawsha" --controllerSha="$virtcontroller_rawsha" --handlerSha="$virthandler_rawsha" --launcherSha="$virtlauncher_rawsha" --kubevirtLogo={{.KubeVirtLogo}} --csvVersion="$_fake_csv_version" --replacesCsvVersion="$_fake_replaces_csv_version" --csvCreatedAtTimestamp={{.CreatedAt}} --kubeVirtVersion={{.DockerTag}} >${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_csv_version/{{.CsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_replaces_csv_version/{{.ReplacesCsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go_build)
vms_docker_prefix=${DOCKER_PREFIX:-registry:5000/kubevirt}
vms_docker_tag=${DOCKER_TAG:-devel}
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --container-prefix=${vms_docker_prefix} --container-tag=${vms_docker_tag} --generated-vms-dir=${KUBEVIRT_DIR}/examples

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=kubevirt_hooks_info:pkg/hooks/info pkg/hooks/info/api.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha1:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api.proto
protoc --proto_path=pkg/hooks/v1alpha2 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha2:pkg/hooks/v1alpha2 pkg/hooks/v1alpha2/api.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/v1/notify.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/info/info.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/v1/cmd.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/info/info.proto

mockgen -source pkg/handler-launcher-com/notify/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/notify/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/cmd/info/generated_mock_info.go
