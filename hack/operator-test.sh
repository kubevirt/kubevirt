#!/bin/bash

set -e

TMP_ROOT="$(dirname "${BASH_SOURCE[@]}")/.."
REPO_ROOT=$(readlink -e "${TMP_ROOT}" 2> /dev/null || perl -MCwd -e 'print Cwd::abs_path shift' "${TMP_ROOT}")

function clean {
    rm -rf "${TEMP_DIR}"
    echo "Deleted working dir ${TEMP_DIR}"
}

source "${REPO_ROOT}"/hack/config
source "${REPO_ROOT}"/hack/defaults

trap clean EXIT

for manifest in $OPERATOR_MANIFESTS; do
    echo "${manifest}"
    wget -P "${TEMP_DIR}" "${manifest}"
done

for crs in $OPERATOR_CRS; do
    echo "${crs}"
    wget -P "${TEMP_DIR}/crs" "${crs}"
done

kubevirt_sed
cdi_sed
network_addons_sed
ssp_sed
vm_import_sed
echo "Replaced image strings"

set +e
set -x

oc create -f ${TEMP_DIR}/

echo "Give resources time to show up"
sleep 10

VIRT_POD=`oc get pods -n kubevirt | grep virt-operator | head -1 | awk '{ print $1 }'`
CDI_POD=`oc get pods -n cdi | grep cdi-operator | head -1 | awk '{ print $1 }'`
NETWORK_ADDONS_POD=`oc get pods -n cluster-network-addons-operator | grep cluster-network-addons-operator | head -1 | awk '{ print $1 }'`
SSP_POD=`oc get pods -n kubevirt-hyperconverged | grep ssp-operator | head -1 | awk '{ print $1 }'`
VM_IMPORT_POD=`oc get pods -n kubevirt-hyperconverged | grep vm-import-operator | head -1 | awk '{ print $1 }'`
oc wait pod $VIRT_POD --for condition=Ready -n kubevirt --timeout="${WAIT_TIMEOUT}"
oc wait pod $CDI_POD --for condition=Ready -n cdi --timeout="${WAIT_TIMEOUT}"
oc wait pod $NETWORK_ADDONS_POD --for condition=Ready -n cluster-network-addons-operator --timeout="${WAIT_TIMEOUT}"
oc wait pod $SSP_POD --for condition=Ready -n kubevirt --timeout="${WAIT_TIMEOUT}"
oc wait pod $VIRT_POD --for condition=Ready -n kubevirt --timeout="${WAIT_TIMEOUT}"
oc wait pod $CDI_POD --for condition=Ready -n cdi --timeout="${WAIT_TIMEOUT}"
oc wait pod $NETWORK_ADDONS_POD --for condition=Ready -n cluster-network-addons-operator --timeout="${WAIT_TIMEOUT}"
oc wait pod $VM_IMPORT_POD --for condition=Ready -n kubevirt --timeout="${WAIT_TIMEOUT}"

oc create -f ${TEMP_DIR}/crs

echo "Let the API server process the CRs"
sleep 10
oc wait networkaddonsconfig cluster --for condition=Ready --timeout="${WAIT_TIMEOUT}"
oc wait kubevirt kubevirt --for condition=Ready -n kubevirt --timeout="${WAIT_TIMEOUT}"
