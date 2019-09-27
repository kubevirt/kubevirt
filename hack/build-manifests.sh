#!/usr/bin/env bash
set -e

# TODO: If we create more hack scripts this should go in common
# and be sourced
PROJECT_ROOT="$(readlink -e $(dirname "$BASH_SOURCE[0]")/../)"

# REPLACES_VERSION is the old CSV_VERSION
#   if REPLACES_VERSION == CSV_VERSION it will be ignored
REPLACES_VERSION="${REPLACES_VERSION:-0.0.2}"
CSV_VERSION="${CSV_VERSION:-0.0.3}"

NAMESPACE="${NAMESPACE:-kubevirt-hyperconverged}"
DEPLOY_DIR="${PROJECT_ROOT}/deploy"
CSV_DIR="${DEPLOY_DIR}/olm-catalog/kubevirt-hyperconverged/${CSV_VERSION}"

CONTAINER_PREFIX="${CONTAINER_PREFIX:-kubevirt}"
CNA_CONTAINER_PREFIX="${CNA_CONTAINER_PREFIX:-quay.io/kubevirt}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

#CNV IMS Images
CONVERSION_CONTAINER="${CONVERSION_CONTAINER:-quay.io/kubevirt/kubevirt-v2v-conversion:v2.0.0}"
VMWARE_CONTAINER="${VMWARE_CONTAINER:-quay.io/kubevirt/kubevirt-vmware:v2.0.0}"

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

	NETWORK_ADDONS_TAG="$(dep status -f='{{if eq .ProjectRoot "github.com/kubevirt/cluster-network-addons-operator"}}{{.Version}} {{end}}')"
	echo "Network Addons: ${NETWORK_ADDONS_TAG}"

	if [ -z "${GITHUB_TOKEN}" ]; then
		NMO_TAG=$(curl --silent "https://api.github.com/repos/kubevirt/node-maintenance-operator/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')
	else
		NMO_TAG=$(curl -H "Authorization: token $GITHUB_TOKEN" --silent "https://api.github.com/repos/kubevirt/node-maintenance-operator/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')
	fi
	if [ -z "${NMO_TAG}" ]; then
		echo "Failed to get WEB_UI_TAG"
		exit 1
	fi
	echo "NMO: ${NMO_TAG}	WARNING: Not using Gopkg.toml version"
}

function buildFlags {

	BUILD_FLAGS="--hco-tag=latest \
	--namespace=${NAMESPACE} \
	--csv-version=${CSV_VERSION} \
	--container-prefix=${CONTAINER_PREFIX} \
	--replaces-version=${REPLACES_VERSION} \
	--image-pull-policy=${IMAGE_PULL_POLICY} \
	--ims-conversion-container=${CONVERSION_CONTAINER} \
	--ims-vmware-container=${VMWARE_CONTAINER}"

	if [ -z "${CONTAINER_TAG}" ]; then
		versions

		BUILD_FLAGS="${BUILD_FLAGS} \
		--kubevirt-tag=${KUBEVIRT_TAG} \
		--cdi-tag=${CDI_TAG} \
		--ssp-tag=${SSP_TAG} \
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

	out_dir="$(dirname ${DEPLOY_DIR}/${template})"
	out_dir=${out_dir/VERSION/$CSV_VERSION}
	mkdir -p ${out_dir}

	out_file="${out_dir}/$(basename -s .in $template)"
	out_file=${out_file/VERSION/v$CSV_VERSION}

	rendered=$( \
		 ${PROJECT_ROOT}/tools/manifest-templator/manifest-templator \
		 ${BUILD_FLAGS} \
		 --converged \
		 --cna-container-prefix=${CNA_CONTAINER_PREFIX} \
		 --input-file=${infile} \
	)
	if [[ ! -z "$rendered" ]]; then
		echo -e "$rendered" > $out_file
		if [[ "${infile}" =~ .*crd.yaml.in ]]; then
			csv_out_dir="${CSV_DIR}"
			mkdir -p ${csv_out_dir}
			csv_out_file="${csv_out_dir}/$(basename -s .in $template)"

			echo -e "$rendered" > $csv_out_file
		fi
	fi
done

(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go clean)
