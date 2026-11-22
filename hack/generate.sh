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
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/pool/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/migrations/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/export/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/export/v1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/clone/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/clone/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/backup/v1alpha1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/backup/v1beta1/types.go
swagger-doc -in ${KUBEVIRT_DIR}/staging/src/kubevirt.io/api/plugin/v1alpha1/types.go

deepcopy-gen \
    --bounding-dirs kubevirt.io/api \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    --output-file deepcopy_generated.go \
    kubevirt.io/api/snapshot/v1alpha1 \
    kubevirt.io/api/snapshot/v1beta1 \
    kubevirt.io/api/export/v1beta1 \
    kubevirt.io/api/export/v1 \
    kubevirt.io/api/instancetype/v1beta1 \
    kubevirt.io/api/pool/v1alpha1 \
    kubevirt.io/api/pool/v1beta1 \
    kubevirt.io/api/migrations/v1alpha1 \
    kubevirt.io/api/clone/v1alpha1 \
    kubevirt.io/api/clone/v1beta1 \
    kubevirt.io/api/backup/v1alpha1 \
    kubevirt.io/api/backup/v1beta1 \
    kubevirt.io/api/plugin/v1alpha1 \
    kubevirt.io/api/core/v1

defaulter-gen \
    --output-file zz_generated.defaults.go \
    --go-header-file ${KUBEVIRT_DIR}/hack/boilerplate/boilerplate.go.txt \
    kubevirt.io/api/core/v1

openapi-gen \
    --output-dir ${KUBEVIRT_DIR}/staging/src/kubevirt.io/client-go/api/ \
    --output-pkg kubevirt.io/client-go/api/ \
    --output-file openapi_generated.go \
    --output-model-name-file zz_generated.model_name.go \
    --readonly-pkg k8s.io/api/core/v1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/runtime,k8s.io/apimachinery/pkg/util/intstr,kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1 \
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
    kubevirt.io/api/export/v1beta1 \
    kubevirt.io/api/export/v1 \
    kubevirt.io/api/instancetype/v1beta1 \
    kubevirt.io/api/migrations/v1alpha1 \
    kubevirt.io/api/pool/v1alpha1 \
    kubevirt.io/api/pool/v1beta1 \
    kubevirt.io/api/snapshot/v1alpha1 \
    kubevirt.io/api/snapshot/v1beta1 \
    kubevirt.io/api/backup/v1alpha1 \
    kubevirt.io/api/backup/v1beta1 \
    kubevirt.io/api/plugin/v1alpha1 \
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
    --input core/v1,export/v1beta1,export/v1,snapshot/v1alpha1,snapshot/v1beta1,instancetype/v1beta1,pool/v1alpha1,pool/v1beta1,migrations/v1alpha1,clone/v1alpha1,clone/v1beta1,backup/v1alpha1,backup/v1beta1,plugin/v1alpha1 \
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
    GOFLAGS= controller-gen crd paths=../api/export/v1beta1/
    GOFLAGS= controller-gen crd paths=../api/export/v1/

    #include instancetype
    GOFLAGS= controller-gen crd paths=../api/instancetype/v1beta1/

    #include pool
    GOFLAGS= controller-gen crd paths=../api/pool/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/pool/v1beta1/

    #include migrations
    GOFLAGS= controller-gen crd paths=../api/migrations/v1alpha1/

    #include clone
    GOFLAGS= controller-gen crd paths=../api/clone/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/clone/v1beta1/

    #include backup
    GOFLAGS= controller-gen crd paths=../api/backup/v1alpha1/
    GOFLAGS= controller-gen crd paths=../api/backup/v1beta1/

    #include plugin
    GOFLAGS= controller-gen crd paths=../api/plugin/v1alpha1/

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

${KUBEVIRT_DIR}/hack/generate-tools.sh

${KUBEVIRT_DIR}/hack/gen-proto.sh

mockgen -source pkg/handler-launcher-com/notify/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/notify/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/info/info.pb.go -package=info -destination=pkg/handler-launcher-com/cmd/info/generated_mock_info.go
mockgen -source pkg/handler-launcher-com/cmd/v1/cmd.pb.go -package=v1 -destination=pkg/handler-launcher-com/cmd/v1/generated_mock_cmd.go

${KUBEVIRT_DIR}/hack/bazel-race.sh
