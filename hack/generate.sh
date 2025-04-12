#!/usr/bin/env bash
set -ex

K8S_VER=$(grep "k8s.io/api => k8s.io/api" go.mod | xargs | cut -d" " -f4)
KUBEOPENAPI_VER="$(grep "k8s.io/kube-openapi => k8s.io/kube-openapi" go.mod | xargs | cut -d" " -f4)"
PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../)"

PACKAGE=github.com/kubevirt/hyperconverged-cluster-operator
API_FOLDER=api
API_VERSION=v1beta1

go install \
	k8s.io/code-generator/cmd/deepcopy-gen@${K8S_VER} \
	k8s.io/code-generator/cmd/defaulter-gen@${K8S_VER}

go install \
	k8s.io/kube-openapi/cmd/openapi-gen@${KUBEOPENAPI_VER}

deepcopy-gen \
	--output-file zz_generated.deepcopy.go \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	"${PACKAGE}/${API_FOLDER}/${API_VERSION}"

defaulter-gen \
	--output-file zz_generated.defaults.go \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	"${PACKAGE}/${API_FOLDER}/${API_VERSION}"

openapi-gen \
	--output-file zz_generated.openapi.go \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	--output-dir api/v1beta1/ \
	--output-pkg github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	"${PACKAGE}/${API_FOLDER}/${API_VERSION}"

go fmt api/v1beta1/zz_generated.deepcopy.go
go fmt api/v1beta1/zz_generated.defaults.go
go fmt api/v1beta1/zz_generated.openapi.go
