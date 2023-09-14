#!/usr/bin/env bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

# generate clients
CLIENT_GEN_BASE=kubevirt.io/client-go/generated
rm -rf ${KUBEVIRT_DIR}/staging/src/${CLIENT_GEN_BASE}

# KubeVirt stuff
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/core/v1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/core/v1/schema.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/snapshot/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/instancetype/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/instancetype/v1alpha2/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/instancetype/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/pool/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/migrations/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/export/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/clone/v1alpha1/types.go

deepcopy-gen --input-dirs kubevirt.io/api/snapshot/v1alpha1,kubevirt.io/api/export/v1alpha1,kubevirt.io/api/instancetype/v1alpha1,kubevirt.io/api/instancetype/v1alpha2,kubevirt.io/api/instancetype/v1beta1,kubevirt.io/api/pool/v1alpha1,kubevirt.io/api/migrations/v1alpha1,kubevirt.io/api/clone/v1alpha1,kubevirt.io/api/core/v1 \
    --bounding-dirs kubevirt.io/api \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

defaulter-gen --input-dirs kubevirt.io/api/core/v1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package kubevirt.io/api/core/v1 \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

openapi-gen --input-dirs kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1,k8s.io/apimachinery/pkg/util/intstr,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/runtime,k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,kubevirt.io/api/core/v1,kubevirt.io/api/export/v1alpha1,kubevirt.io/api/snapshot/v1alpha1,kubevirt.io/api/instancetype/v1alpha1,kubevirt.io/api/instancetype/v1alpha2,kubevirt.io/api/instancetype/v1beta1,kubevirt.io/api/pool/v1alpha1,kubevirt.io/api/migrations/v1alpha1,kubevirt.io/api/clone/v1alpha1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package kubevirt.io/client-go/api/ \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt >${KUBEVIRT_DIR}/api/api-rule-violations.list

conversion-gen --input-dirs kubevirt.io/api/instancetype/v1alpha1,kubevirt.io/api/instancetype/v1alpha2,kubevirt.io/api/instancetype/v1beta1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

if cmp ${KUBEVIRT_DIR}/api/api-rule-violations.list ${KUBEVIRT_DIR}/api/api-rule-violations-known.list; then
    echo "openapi generated"
else
    diff -u ${KUBEVIRT_DIR}/api/api-rule-violations-known.list ${KUBEVIRT_DIR}/api/api-rule-violations.list || true
    echo "You introduced new API rule violation"
    diff ${KUBEVIRT_DIR}/api/api-rule-violations.list ${KUBEVIRT_DIR}/api/api-rule-violations-known.list
    exit 2
fi

client-gen --clientset-name versioned \
    --input-base kubevirt.io/api \
    --input export/v1alpha1,snapshot/v1alpha1,instancetype/v1alpha1,instancetype/v1alpha2,instancetype/v1beta1,pool/v1alpha1,migrations/v1alpha1,clone/v1alpha1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/kubevirt/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

# dependencies
client-gen --clientset-name versioned \
    --input-base kubevirt.io/containerized-data-importer-api/pkg/apis \
    --input core/v1beta1,upload/v1beta1 \
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
    --input-base github.com/kubernetes-csi/external-snapshotter/client/v4/apis \
    --input volumesnapshot/v1 \
    --output-base ${KUBEVIRT_DIR}/staging/src \
    --output-package ${CLIENT_GEN_BASE}/external-snapshotter/clientset \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}

deepcopy-gen --input-dirs ./pkg/virt-launcher/virtwrap/api \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

# Genearte validation with controller-gen and create go file for them
(
    cd ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go &&
        # suppress -mod=vendor
        GOFLAGS= controller-gen crd:allowDangerousTypes=true paths=../api/core/v1/
    #include snapshot
    GOFLAGS= controller-gen crd paths=../api/snapshot/v1alpha1/

    #include export
    GOFLAGS= controller-gen crd paths=../api/export/v1alpha1/

    #include instancetype
    GOFLAGS= controller-gen crd paths=../api/instancetype/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/instancetype/v1alpha2/
    GOFLAGS= controller-gen crd paths=../api/instancetype/v1beta1/

    #include pool
    GOFLAGS= controller-gen crd paths=../api/pool/v1alpha1/

    #include migrations
    GOFLAGS= controller-gen crd paths=../api/migrations/v1alpha1/

    #include clone
    GOFLAGS= controller-gen crd paths=../api/clone/v1alpha1/

    #remove some weird stuff from controller-gen
    cd config/crd
    for file in *; do
        tail -n +3 $file >$file"new"
        mv $file"new" $file
    done
    cd ${KUBEVIRT_DIR}/tools/crd-validation-generator/ && go_build

    cd ${KUBEVIRT_DIR}
    ${KUBEVIRT_DIR}/tools/crd-validation-generator/crd-validation-generator
)
rm -rf ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go/config

/${KUBEVIRT_DIR}/hack/bootstrap-ginkgo.sh

(cd ${KUBEVIRT_DIR}/tools/openapispec/ && go_build)

${KUBEVIRT_DIR}/tools/openapispec/openapispec --dump-api-spec-path ${KUBEVIRT_DIR}/api/openapi-spec/swagger.json

(cd ${KUBEVIRT_DIR}/tools/resource-generator/ && go_build)
(cd ${KUBEVIRT_DIR}/tools/csv-generator/ && go_build)
(cd ${KUBEVIRT_DIR}/tools/doc-generator/ && go_build)
(
    cd ${KUBEVIRT_DIR}/docs
    ${KUBEVIRT_DIR}/tools/doc-generator/doc-generator
    mv newmetrics.md metrics.md
)

rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/examples/*

ResourceDir=${KUBEVIRT_DIR}/manifests/generated
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=priorityclass >${ResourceDir}/kubevirt-priority-class.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${ResourceDir}/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace='{{.Namespace}}' --pullPolicy='{{.ImagePullPolicy}}' \
    --featureGates='{{.FeatureGates}}' --infraReplicas='{{.InfraReplicas}}' >${ResourceDir}/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=operator-rbac --namespace='{{.Namespace}}' >${ResourceDir}/rbac-operator.authorization.k8s.yaml.in

# used for Image fields in manifests
function getVersion() {
    echo "{{if $1}}@{{$1}}{{else}}:{{.DockerTag}}{{end}}"
}
virtapi_version=$(getVersion ".VirtApiSha")
virtcontroller_version=$(getVersion ".VirtControllerSha")
virthandler_version=$(getVersion ".VirtHandlerSha")
virtlauncher_version=$(getVersion ".VirtLauncherSha")
virtoperator_version=$(getVersion ".VirtOperatorSha")
virtexportproxy_version=$(getVersion ".VirtExportProxySha")
virtexportserver_version=$(getVersion ".VirtExportServerSha")

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
virtexportproxy_sha=$(getShasum ".VirtExportProxySha")
virtexportserver_sha=$(getShasum ".VirtExportServerSha")
gs_sha=$(getShasum ".GsSha")
pr_helper_sha=$(getShasum ".PrHelperSha")

virtapi_rawsha=$(getRawShasum ".VirtApiSha")
virtcontroller_rawsha=$(getRawShasum ".VirtControllerSha")
virthandler_rawsha=$(getRawShasum ".VirtHandlerSha")
virtlauncher_rawsha=$(getRawShasum ".VirtLauncherSha")
virtexportproxy_rawsha=$(getRawShasum ".VirtExportProxySha")
virtexportserver_rawsha=$(getRawShasum ".VirtExportServerSha")
gs_rawsha=$(getRawShasum ".GsSha")
prhelper_rawsha=$(getRawShasum ".PrHelperSha")

# The generation code for CSV requires a valid semver to be used.
# But we're trying to generate a template for a CSV here from code
# rather than an actual usable CSV. To work around this, we set the
# versions to something absurd and do a find/replace with our templated
# values after the file is generated.
_fake_replaces_csv_version="1111.1111.1111"
_fake_csv_version="2222.2222.2222"
${KUBEVIRT_DIR}/tools/csv-generator/csv-generator --namespace={{.CSVNamespace}} --dockerPrefix={{.DockerPrefix}} --operatorImageVersion="$virtoperator_version" --pullPolicy={{.ImagePullPolicy}} --verbosity={{.Verbosity}} --apiSha="$virtapi_rawsha" --controllerSha="$virtcontroller_rawsha" --handlerSha="$virthandler_rawsha" --launcherSha="$virtlauncher_rawsha" --exportProxySha="$virtexportproxy_rawsha" --exportServerSha="$virtexportserver_rawsha" --gsSha="$gs_rawsha" --prHelperSha="$prhelper_rawsha" --kubevirtLogo={{.KubeVirtLogo}} --csvVersion="$_fake_csv_version" --replacesCsvVersion="$_fake_replaces_csv_version" --csvCreatedAtTimestamp={{.CreatedAt}} --kubeVirtVersion={{.DockerTag}} >${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_csv_version/{{.CsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in
sed -i "s/$_fake_replaces_csv_version/{{.ReplacesCsvVersion}}/g" ${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

(cd ${KUBEVIRT_DIR}/tools/vms-generator/ && go_build)
vms_docker_prefix=${DOCKER_PREFIX:-registry:5000/kubevirt}
vms_docker_tag=${DOCKER_TAG:-devel}
${KUBEVIRT_DIR}/tools/vms-generator/vms-generator --container-prefix=${vms_docker_prefix} --container-tag=${vms_docker_tag} --generated-vms-dir=${KUBEVIRT_DIR}/examples

${KUBEVIRT_DIR}/hack/gen-proto.sh

mockgen -source pkg/handler-launcher-com/notify/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/notify/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/cmd/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/v1/cmd.pb.go -package=v1 -destination=pkg/handler-launcher-com/cmd/v1/generated_mock_cmd.go
