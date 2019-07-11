#!/usr/bin/env bash
set -e

# TODO: If we create more hack scripts this should go in common
# and be sourced
PROJECT_ROOT="$(readlink -e $(dirname "$BASH_SOURCE[0]")/../)"

# TODO: Move this to deploy
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
STD_DEPLOY_DIR="${DEPLOY_DIR}/standard"
CONVERGED_DEPLOY_DIR="${DEPLOY_DIR}/converged"

NAMESPACE="${NAMESPACE:-kubevirt-hyperconverged}"
CSV_VERSION="${CSV_VERSION:-0.0.1}"
CONTAINER_PREFIX="${CONTAINER_PREFIX:-kubevirt}"
CNA_CONTAINER_PREFIX="${CNA_CONTAINER_PREFIX:-quay.io/kubevirt}"
WEBUI_CONTAINER_PREFIX="${WEBUI_CONTAINER_PREFIX:-quay.io/kubevirt}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

# HCO Tag hardcoded to latest
CONTAINER_TAG="${CONTAINER_TAG:-}"

(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go build)

function versions {
    KUBEVIRT_TAG="$(dep status -f='{{if eq .ProjectRoot "kubevirt.io/kubevirt"}}{{.Version}} {{end}}')"
    echo "KubeVirt: ${KUBEVIRT_TAG}"

    CDI_TAG="$(dep status -f='{{if eq .ProjectRoot "kubevirt.io/containerized-data-importer"}}{{.Version}} {{end}}')"
    echo "CDI: ${CDI_TAG}"

    SSP_TAG="$(dep status -f='{{if eq .ProjectRoot "github.com/MarSik/kubevirt-ssp-operator"}}{{.Version}} {{end}}')"
    echo "SSP: ${SSP_TAG}"

    WEB_UI_OPERATOR_TAG="$(dep status -f='{{if eq .ProjectRoot "github.com/kubevirt/web-ui-operator"}}{{.Version}} {{end}}')"
    echo "Web UI operator: ${WEB_UI_OPERATOR_TAG}"

    WEB_UI_TAG=$(curl --silent "https://api.github.com/repos/kubevirt/web-ui/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")' | sed 's/kubevirt-/v/g' )
    echo "Web UI: ${WEB_UI_TAG}"

    NETWORK_ADDONS_TAG="$(dep status -f='{{if eq .ProjectRoot "github.com/kubevirt/cluster-network-addons-operator"}}{{.Version}} {{end}}')"
    echo "Network Addons: ${NETWORK_ADDONS_TAG}"

    NMO_TAG=$(curl --silent "https://api.github.com/repos/kubevirt/node-maintenance-operator/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')
    echo "NMO: ${NMO_TAG}   WARNING: Not using Gopkg.toml version"
}

function buildFlags {

    BUILD_FLAGS="--hco-tag=latest \
    --namespace=${NAMESPACE} \
    --csv-version=${CSV_VERSION} \
    --container-prefix=${CONTAINER_PREFIX} \
    --image-pull-policy=${IMAGE_PULL_POLICY}"

    if [ -z "${CONTAINER_TAG}" ]; then
	versions

	BUILD_FLAGS="${BUILD_FLAGS} \
	--kubevirt-tag=${KUBEVIRT_TAG} \
	--cdi-tag=${CDI_TAG} \
	--ssp-tag=${SSP_TAG} \
	--web-ui-tag=${WEB_UI_TAG} \
	--web-ui-operator-tag=${WEB_UI_OPERATOR_TAG} \
	--nmo-tag=${NMO_TAG} \
	--network-addons-tag=${NETWORK_ADDONS_TAG}"
    else
	BUILD_FLAGS="${BUILD_FLAGS} \
	--container-tag=${CONTAINER_TAG}"
    fi
}

buildFlags

templates=$(cd ${PROJECT_ROOT}/templates && find . -type f -name "*.yaml.in")
for template in $templates; do
	infile="${PROJECT_ROOT}/templates/${template}"

	std_out_dir="$(dirname ${STD_DEPLOY_DIR}/${template})"
	std_out_dir=${std_out_dir/VERSION/$CSV_VERSION}
	mkdir -p ${std_out_dir}

	std_out_file="${std_out_dir}/$(basename -s .in $template)"
	std_out_file=${std_out_file/VERSION/v$CSV_VERSION}

	rendered=$( \
                 ${PROJECT_ROOT}/tools/manifest-templator/manifest-templator \
                 ${BUILD_FLAGS} \
                 --input-file=${infile} \
	)
	if [[ ! -z "$rendered" ]]; then
		echo -e "$rendered" > $std_out_file
	fi

	converged_out_dir="$(dirname ${CONVERGED_DEPLOY_DIR}/${template})"
	converged_out_dir=${converged_out_dir/VERSION/$CSV_VERSION}
	mkdir -p ${converged_out_dir}

	converged_out_file="${converged_out_dir}/$(basename -s .in $template)"
	converged_out_file=${converged_out_file/VERSION/v$CSV_VERSION}

	rendered=$( \
                 ${PROJECT_ROOT}/tools/manifest-templator/manifest-templator \
                 ${BUILD_FLAGS} \
                 --converged \
                 --cna-container-prefix=${CNA_CONTAINER_PREFIX} \
                 --webui-container-prefix=${WEBUI_CONTAINER_PREFIX} \
                 --input-file=${infile} \
	)
	if [[ ! -z "$rendered" ]]; then
		echo -e "$rendered" > $converged_out_file
	fi
done

(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go clean)
