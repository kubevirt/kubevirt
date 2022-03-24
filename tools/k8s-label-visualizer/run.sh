#!/bin/bash

set -euox pipefail

echo KUBECONFIG=${KUBECONFIG}

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_OUT_PATH="${TEST_OUT_PATH:-"$DIR"}"
NAMESPACE="${NAMESPACE:-"kubevirt-hyperconverged"}"

python3 -mplatform | grep -qEi "Ubuntu|Debian" && apt-get update && apt-get install -y python3-venv graphviz || true

python3 -m venv "${DIR}/venv"
source "${DIR}/venv/bin/activate"
pip3 install -r "${DIR}/requirements.txt"

OUTDIR="${TEST_OUT_PATH}/output/"
if [[ -n "${ARTIFACTS-}" ]]; then
  OUTDIR=${ARTIFACTS}
fi

python3 "${DIR}/main.py" --namespace "${NAMESPACE}" --conf "${DIR}/conf.json" --output "$OUTDIR"
