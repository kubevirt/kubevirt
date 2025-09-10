#!/usr/bin/env bash

set -ex

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

# generate clients
CLIENT_GEN_BASE=kubevirt.io/client-go

# KubeVirt stuff
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/core/v1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/core/v1/schema.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/snapshot/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/snapshot/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/instancetype/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/pool/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/migrations/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/export/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/export/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/clone/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/clone/v1beta1/types.go

deepcopy-gen \
    --bounding-dirs kubevirt.io/api \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    --output-file deepcopy_generated.go \
    kubevirt.io/api/snapshot/v1alpha1 \
    kubevirt.io/api/snapshot/v1beta1 \
    kubevirt.io/api/export/v1alpha1 \
    kubevirt.io/api/export/v1beta1 \
    kubevirt.io/api/instancetype/v1beta1 \
    kubevirt.io/api/pool/v1alpha1 \
    kubevirt.io/api/migrations/v1alpha1 \
    kubevirt.io/api/clone/v1alpha1 \
    kubevirt.io/api/clone/v1beta1 \
    kubevirt.io/api/core/v1

defaulter-gen \
    --output-file zz_generated.defaults.go \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    kubevirt.io/api/core/v1

openapi-gen \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go/api/ \
    --output-pkg kubevirt.io/client-go/api/ \
    --output-file openapi_generated.go \
    --report-filename ${KUBEVIRT_DIR}/api/api-rule-violations.list \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    k8s.io/api/core/v1 \
    k8s.io/apimachinery/pkg/api/resource \
    k8s.io/apimachinery/pkg/apis/meta/v1 \
    k8s.io/apimachinery/pkg/runtime \
    k8s.io/apimachinery/pkg/util/intstr \
    kubevirt.io/api/core/v1 \
    kubevirt.io/api/clone/v1alpha1 \
    kubevirt.io/api/clone/v1beta1 \
    kubevirt.io/api/export/v1alpha1 \
    kubevirt.io/api/export/v1beta1 \
    kubevirt.io/api/instancetype/v1beta1 \
    kubevirt.io/api/migrations/v1alpha1 \
    kubevirt.io/api/pool/v1alpha1 \
    kubevirt.io/api/snapshot/v1alpha1 \
    kubevirt.io/api/snapshot/v1beta1 \
    kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1

conversion-gen \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    --output-file conversion_generated.go \
    kubevirt.io/api/instancetype/v1beta1

if cmp ${KUBEVIRT_DIR}/api/api-rule-violations.list ${KUBEVIRT_DIR}/api/api-rule-violations-known.list; then
    echo "openapi generated"
else
    diff -u ${KUBEVIRT_DIR}/api/api-rule-violations-known.list ${KUBEVIRT_DIR}/api/api-rule-violations.list || true
    echo "You introduced new API rule violation"
    diff ${KUBEVIRT_DIR}/api/api-rule-violations.list ${KUBEVIRT_DIR}/api/api-rule-violations-known.list
    exit 2
fi

client-gen --clientset-name kubevirt \
    --input-base kubevirt.io/api \
    --input core/v1,export/v1alpha1,export/v1beta1,snapshot/v1alpha1,snapshot/v1beta1,instancetype/v1beta1,pool/v1alpha1,migrations/v1alpha1,clone/v1alpha1,clone/v1beta1 \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go \
    --output-pkg ${CLIENT_GEN_BASE} \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

# dependencies
client-gen --clientset-name containerizeddataimporter \
    --input-base kubevirt.io/containerized-data-importer-api/pkg/apis \
    --input core/v1beta1,upload/v1beta1 \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go \
    --output-pkg ${CLIENT_GEN_BASE} \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name prometheusoperator \
    --input-base github.com/prometheus-operator/prometheus-operator/pkg/apis \
    --input monitoring/v1 \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go \
    --output-pkg ${CLIENT_GEN_BASE} \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name networkattachmentdefinitionclient \
    --input-base github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis \
    --input k8s.cni.cncf.io/v1 \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go \
    --output-pkg ${CLIENT_GEN_BASE} \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

client-gen --clientset-name externalsnapshotter \
    --input-base github.com/kubernetes-csi/external-snapshotter/client/v4/apis \
    --input volumesnapshot/v1 \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go \
    --output-pkg ${CLIENT_GEN_BASE} \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt

find ${KUBEVIRT_DIR}/pkg/ -name "*generated*.go" -exec rm {} -f \;

${KUBEVIRT_DIR}/hack/build-go.sh generate ${WHAT}

deepcopy-gen \
    --output-file deepcopy_generated.go \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    ./pkg/virt-launcher/virtwrap/api

# Generate validation with controller-gen and create go file for them
(
    cd ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go &&
        # suppress -mod=vendor
        GOFLAGS= controller-gen crd:allowDangerousTypes=true paths=../api/core/v1/
    #include snapshot
    GOFLAGS= controller-gen crd paths=../api/snapshot/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/snapshot/v1beta1/

    #include export
    GOFLAGS= controller-gen crd paths=../api/export/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/export/v1beta1/

    #include instancetype
    GOFLAGS= controller-gen crd paths=../api/instancetype/v1beta1/

    #include pool
    GOFLAGS= controller-gen crd paths=../api/pool/v1alpha1/

    #include migrations
    GOFLAGS= controller-gen crd paths=../api/migrations/v1alpha1/

    #include clone
    GOFLAGS= controller-gen crd paths=../api/clone/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/clone/v1beta1/

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
    cd ${KUBEVIRT_DIR}/docs/observability
    ${KUBEVIRT_DIR}/tools/doc-generator/doc-generator >metrics.md
)

rm -f ${KUBEVIRT_DIR}/manifests/generated/*
rm -f ${KUBEVIRT_DIR}/examples/*

ResourceDir=${KUBEVIRT_DIR}/manifests/generated
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=priorityclass >${ResourceDir}/kubevirt-priority-class.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv >${ResourceDir}/kv-resource.yaml
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=kv-cr --namespace='{{.Namespace}}' --pullPolicy='{{.ImagePullPolicy}}' \
    --featureGates='{{.FeatureGates}}' --infraReplicas='{{.InfraReplicas}}' >${ResourceDir}/kubevirt-cr.yaml.in
${KUBEVIRT_DIR}/tools/resource-generator/resource-generator --type=operator-rbac --namespace='{{.Namespace}}' >${ResourceDir}/rbac-operator.authorization.k8s.yaml.in

# The generation code for CSV requires a valid semver to be used.
# But we're trying to generate a template for a CSV here from code
# rather than an actual usable CSV. To work around this, we set the
# versions to something absurd and do a find/replace with our templated
# values after the file is generated.
_fake_replaces_csv_version="1111.1111.1111"
_fake_csv_version="2222.2222.2222"
${KUBEVIRT_DIR}/tools/csv-generator/csv-generator \
    --operatorImageVersion="{{.DockerTag}}" \
    --csvCreatedAtTimestamp={{.CreatedAt}} \
    --csvVersion="$_fake_csv_version" \
    --dockerPrefix={{.DockerPrefix}} \
    --kubevirtLogo={{.KubeVirtLogo}} \
    --kubeVirtVersion={{.DockerTag}} \
    --namespace={{.CSVNamespace}} \
    --pullPolicy={{.ImagePullPolicy}} \
    --replacesCsvVersion="$_fake_replaces_csv_version" \
    --verbosity={{.Verbosity}} \
    >${KUBEVIRT_DIR}/manifests/generated/operator-csv.yaml.in

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

${KUBEVIRT_DIR}/hack/bazel-race.sh
