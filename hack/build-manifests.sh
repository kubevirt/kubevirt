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
CONTAINER_TAG="${CONTAINER_TAG:-latest}"
IMAGE_PULL_POLICY="${IMAGE_PULL_POLICY:-IfNotPresent}"

(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go build)

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
		--namespace=${NAMESPACE} \
		--csv-version=${CSV_VERSION} \
		--container-prefix=${CONTAINER_PREFIX} \
		--container-tag=${CONTAINER_TAG} \
		--image-pull-policy=${IMAGE_PULL_POLICY} \
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
		--converged \
		--namespace=${NAMESPACE} \
		--csv-version=${CSV_VERSION} \
		--container-prefix=${CONTAINER_PREFIX} \
		--container-tag=${CONTAINER_TAG} \
		--image-pull-policy=${IMAGE_PULL_POLICY} \
		--input-file=${infile} \
	)
	if [[ ! -z "$rendered" ]]; then
		echo -e "$rendered" > $converged_out_file
	fi
done

(cd ${PROJECT_ROOT}/tools/manifest-templator/ && go clean)
