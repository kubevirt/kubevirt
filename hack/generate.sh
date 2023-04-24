#!/usr/bin/env bash
set -ex

K8S_VER=$(grep "k8s.io/api => k8s.io/api" go.mod | xargs | cut -d" " -f4)
PROJECT_ROOT="$(readlink -e "$(dirname "${BASH_SOURCE[0]}")"/../)"

go install \
k8s.io/code-generator/cmd/deepcopy-gen@${K8S_VER} \
k8s.io/code-generator/cmd/defaulter-gen@${K8S_VER} \
k8s.io/code-generator/cmd/openapi-gen@${K8S_VER}

deepcopy-gen \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	--output-base "${PROJECT_ROOT}" \
	--output-file-base zz_generated.deepcopy \
	--input-dirs github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--output-package github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--trim-path-prefix="${PROJECT_ROOT}/github.com/kubevirt/hyperconverged-cluster-operator/"

defaulter-gen \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	--output-base "${PROJECT_ROOT}" \
	--output-file-base zz_generated.defaults \
	--input-dirs github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--output-package github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--trim-path-prefix="${PROJECT_ROOT}/github.com/kubevirt/hyperconverged-cluster-operator/"

openapi-gen \
	--go-header-file "${PROJECT_ROOT}/hack/boilerplate.go.txt" \
	--output-base "${PROJECT_ROOT}" \
	--output-file-base zz_generated.openapi \
	--input-dirs github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--output-package github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1 \
	--trim-path-prefix="${PROJECT_ROOT}/github.com/kubevirt/hyperconverged-cluster-operator/"

go fmt api/v1beta1/zz_generated.deepcopy.go
go fmt api/v1beta1/zz_generated.defaults.go
go fmt api/v1beta1/zz_generated.openapi.go
