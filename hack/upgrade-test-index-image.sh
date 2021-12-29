#!/bin/bash -e
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2021 Red Hat, Inc.
#
# Usage:
# make upgrade-test-index-image
#
# Use Openshift-CI "optional-operators-ci-*" workflow to:
# - Build an internal index image based off of the index image at:
#   quay.io/kubevirt/hyperconverged-cluster-index
#   with the appropriate tag of the version
# - Add to that index a new bundle, named 100.0.0 with the contents
#   of the open PR (this can include new dependent images, new CRDs...).
# - Subscribe to the initial channel, using the "optional-operators-ci-subscribe"
#   step.
# This script is then upgrading HCO to 100.0.0 version, by patching the subscripiton,
# and performs various validations against the upgraded version.


MAX_STEPS=$(( $(grep -c "Msg " "$0") - 2)) # subtract self line and the function name
CUR_STEP=1
RELEASE_DELTA="${RELEASE_DELTA:-1}"
HCO_DEPLOYMENT_NAME=hco-operator
HCO_WH_DEPLOYMENT_NAME=hco-webhook
HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"
PACKAGE_DIR="./deploy/olm-catalog/community-kubevirt-hyperconverged"
INITIAL_CHANNEL=$(ls -d ${PACKAGE_DIR}/*/ | sort -rV | awk "NR==${RELEASE_DELTA}" | cut -d '/' -f 5)
TARGET_VERSION=100.0.0
TARGET_CHANNEL=${TARGET_VERSION}
VMS_NAMESPACE=vmsns

echo "INITIAL_CHANNEL: $INITIAL_CHANNEL"

function Msg {
    { set +x; } 2>/dev/null
    echo "--"
    for a in "$@"; do
        echo "Upgrade Step ${CUR_STEP}/${MAX_STEPS}: $a"
    done
    echo "--"
    ((CUR_STEP += 1))
    set -x
}

source ./hack/upgrade-openshiftci-config

function cleanup() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during upgrade: exit status: $rv"
        make dump-state
        echo "*** Upgrade test failed ***"
    fi
    exit $rv
}

trap "cleanup" INT TERM EXIT

echo "----- Set KVM_EMULATION if needed"
if [[ -n "${KVM_EMULATION}" ]]; then
  ${CMD} patch $(${CMD} get subscription -n "${HCO_NAMESPACE}" -o name) -n "${HCO_NAMESPACE}" -p '{"spec":{"config":{"selector":{"matchLabels":{"name":"hyperconverged-cluster-operator"}},"env":[{"name":"KVM_EMULATION","value":"true"}]}}}' --type=merge
  sleep 30
  ${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="30m"
  ${CMD} wait deployment ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="30m"
fi

source hack/compare_scc.sh
dump_sccs_before

CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE})
HCO_API_VERSION=$( ${CMD} get -n ${HCO_NAMESPACE} "${CSV}" -o jsonpath="{ .spec.customresourcedefinitions.owned[?(@.kind=='HyperConverged')].version }")
sed -e "s|hco.kubevirt.io/v1beta1|hco.kubevirt.io/${HCO_API_VERSION}|g" deploy/hco.cr.yaml | ${CMD} apply -n kubevirt-hyperconverged -f -

${CMD} wait -n ${HCO_NAMESPACE} ${HCO_KIND} ${HCO_RESOURCE_NAME} --for condition=Available --timeout="30m"
${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="30m"
${CMD} wait deployment ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="30m"

Msg "Check that cluster is operational before upgrade"
timeout 10m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

${CMD} get subscription -n ${HCO_NAMESPACE} -o yaml
${CMD} get pods -n ${HCO_NAMESPACE}

echo "----- Images before upgrade"
${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

echo "----- Get virtctl"
KV_VERSION=$( ${CMD} get kubevirt.kubevirt.io/kubevirt-kubevirt-hyperconverged -n ${HCO_NAMESPACE} -o=jsonpath="{.status.observedKubeVirtVersion}")
ARCH=$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/') || windows-amd64.exe
echo ${ARCH}
curl -L -o ~/virtctl https://github.com/kubevirt/kubevirt/releases/download/${KV_VERSION}/virtctl-${KV_VERSION}-${ARCH}
chmod +x ~/virtctl
###################

Msg "operator conditions before upgrade"
source ./hack/check_operator_condition.sh
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${INITIAL_CHANNEL}"

### Create a VM ###
Msg "Create a simple VM on the previous version cluster, before the upgrade"
${CMD} create namespace ${VMS_NAMESPACE}
${CMD} apply -n ${VMS_NAMESPACE} -f ./hack/vm.yaml
${CMD} get vm -n ${VMS_NAMESPACE} -o yaml testvm
~/virtctl start testvm -n ${VMS_NAMESPACE}
./hack/retry.sh 30 10 "${CMD} get vmi -n ${VMS_NAMESPACE} testvm -o jsonpath='{ .status.phase }' | grep 'Running'"
${CMD} get vmi -n ${VMS_NAMESPACE} -o yaml testvm
INITIAL_BOOTTIME=$(./hack/vmuptime.ext | grep "^BOOTTIME" | cut -d= -f2 | tr -dc '[:digit:]')

echo "----- HCO deployOVS annotation and OVS state in CNAO CR before the upgrade"
PREVIOUS_OVS_ANNOTATION=$(${CMD} get ${HCO_KIND} ${HCO_RESOURCE_NAME} -n ${HCO_NAMESPACE} -o jsonpath='{.metadata.annotations.deployOVS}')
PREVIOUS_OVS_STATE=$(${CMD} get networkaddonsconfigs cluster -o jsonpath='{.spec.ovs}')

# Before starting the upgrade, make sure the CSV is installed properly.
Msg "Read the CSV to make sure the deployment is done"
# Make sure the CSV is in Succeeded phase
./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"
# Make sure the CSV is in the correct version
./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .spec.version }' | grep ${INITIAL_CHANNEL}"

# Create a new version based off of latest. The new version appends ".1" to the latest version.
# The new version replaces the hco-operator image from quay.io with the image pushed to the local registry.
# We create a new CSV based off of the latest version and update the replaces attribute so that the new
# version updates the latest version.
# The currentCSV in the package manifest is also updated to point to the new version.

Msg "Patch the subscription to move to the new channel"
HCO_SUBSCRIPTION=$(${CMD} get subscription -n ${HCO_NAMESPACE} -o name)
OLD_INSTALL_PLAN=$(oc -n "${HCO_NAMESPACE}" get "${HCO_SUBSCRIPTION}" -o jsonpath='{.status.installplan.name}')
${CMD} patch ${HCO_SUBSCRIPTION} -n ${HCO_NAMESPACE} -p "{\"spec\": {\"channel\": \"${TARGET_CHANNEL}\"}}"  --type merge

Msg "Wait up to 5 minutes for the new installPlan to appear, and approve it to begin upgrade"
INSTALL_PLAN_APPROVED=false
for _ in $(seq 1 60); do
    INSTALL_PLAN=$(oc -n "${HCO_NAMESPACE}" get "${HCO_SUBSCRIPTION}" -o jsonpath='{.status.installplan.name}' || true)
    if [[ "${INSTALL_PLAN}" != "${OLD_INSTALL_PLAN}" ]]; then
      ${CMD} -n "${HCO_NAMESPACE}" patch installPlan "${INSTALL_PLAN}" --type merge --patch '{"spec":{"approved":true}}'
      INSTALL_PLAN_APPROVED=true
      break
    fi
    sleep 5
done

[[ "${INSTALL_PLAN_APPROVED}" = true ]]

# Patch the OperatorGroup to match the required InstallMode of the new version
sleep 60
HCO_OPERATORGROUP_NAME=$(${CMD} get og -n ${HCO_NAMESPACE} -o jsonpath='{.items[].metadata.name}')
source hack/patch_og.sh
patch_og ${TARGET_CHANNEL}
sleep 30
CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${INITIAL_CHANNEL})
if [ -n "${CSV}" ] && [ ${OG_PATCHED} -eq 1 ]
then
  ${CMD} delete "${CSV}" -n ${HCO_NAMESPACE}
  sleep 30
fi

# Verify the subscription has changed to the new version
#  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
#  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
Msg "Verify the subscription's currentCSV and installedCSV have moved to the new version"

${CMD} get pods -n ${HCO_NAMESPACE}
./hack/retry.sh 30 60 "${CMD} get deployment -n ${HCO_NAMESPACE} | grep ${HCO_DEPLOYMENT_NAME}"

${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"
${CMD} wait deployment ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"

Msg "operator conditions during upgrade"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${INITIAL_CHANNEL}"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${TARGET_VERSION}"

./hack/retry.sh 30 60 "${CMD} get subscriptions -n ${HCO_NAMESPACE} -o yaml | grep currentCSV   | grep v${TARGET_VERSION}"
./hack/retry.sh  2 30 "${CMD} get subscriptions -n ${HCO_NAMESPACE} -o yaml | grep installedCSV | grep v${TARGET_VERSION}"

Msg "Verify the hyperconverged-cluster deployment is using the new image"

set -x
SEARCH_PHRASE="${OPENSHIFT_BUILD_NAMESPACE}/pipeline"
./hack/retry.sh 60 30 "${CMD} get -n ${HCO_NAMESPACE} deployment ${HCO_DEPLOYMENT_NAME} -o jsonpath=\"{ .spec.template.spec.containers[0].image }\" | grep ${SEARCH_PHRASE}"

Msg "Wait that cluster is operational after upgrade"
timeout 20m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

# Make sure the CSV is installed properly.
Msg "Read the CSV to make sure the deployment is done"
CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${TARGET_VERSION})
# Make sure the CSV is in Succeeded phase
./hack/retry.sh 90 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"
# Make sure the CSV is in the correct version
./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .spec.version }' | grep ${TARGET_VERSION}"

echo "----- Pod after upgrade"
Msg "Verify that the hyperconverged-cluster Pod is using the new image"
./hack/retry.sh 10 30 "CMD=${CMD} HCO_NAMESPACE=${HCO_NAMESPACE} ./hack/check_pod_upgrade.sh"

Msg "Verify new operator version reported after the upgrade"
./hack/retry.sh 15 30 "CMD=${CMD} HCO_RESOURCE_NAME=${HCO_RESOURCE_NAME} HCO_NAMESPACE=${HCO_NAMESPACE} TARGET_VERSION=${TARGET_VERSION} hack/check_hco_version.sh"

Msg "Ensure that HCO detected the cluster as OpenShift"
for hco_pod in $( ${CMD} get pods -n ${HCO_NAMESPACE} -l "name=hyperconverged-cluster-operator" --field-selector=status.phase=Running -o name); do
  pod_version=$( ${CMD} get ${hco_pod} -n ${HCO_NAMESPACE} -o json | jq -r '.spec.containers[0].env[] | select(.name=="HCO_KV_IO_VERSION") | .value')
  if [[ ${pod_version} == ${TARGET_VERSION} ]]; then
    ${CMD} logs -n ${HCO_NAMESPACE} "${hco_pod}" | grep "Cluster type = openshift"
    found_new_running_hco_pod="true"
  fi
done

Msg "operator conditions after upgrade"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${TARGET_VERSION}"

Msg "Ensure that old SSP operator resources are removed from the cluster"
./hack/retry.sh 5 30 "CMD=${CMD} HCO_RESOURCE_NAME=${HCO_RESOURCE_NAME} HCO_NAMESPACE=${HCO_NAMESPACE} ./hack/check_old_ssp_removed.sh"

[[ -n ${found_new_running_hco_pod} ]]

echo "----- Images after upgrade"
# TODO: compare all of them with the list of images in RelatedImages in the new CSV
${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

dump_sccs_after

Msg "make sure that the VM is still running, after the upgrade"
${CMD} get vm -n ${VMS_NAMESPACE} -o yaml testvm
${CMD} get vmi -n ${VMS_NAMESPACE} -o yaml testvm
${CMD} get vmi -n ${VMS_NAMESPACE} testvm -o jsonpath='{ .status.phase }' | grep 'Running'
CURRENT_BOOTTIME=$(./hack/vmuptime.ext | grep "^BOOTTIME" | cut -d= -f2 | tr -dc '[:digit:]')

if ((INITIAL_BOOTTIME - CURRENT_BOOTTIME > 3)) || ((CURRENT_BOOTTIME - INITIAL_BOOTTIME > 3)); then
    echo "ERROR: The test VM got restarted during the upgrade process."
    exit 1
else
    echo "The test VM survived the upgrade process."
fi

Msg "make sure that we don't have outdated VMs"

INFRASTRUCTURETOPOLOGY=$(${CMD} get infrastructure.config.openshift.io cluster -o json | jq -j '.status.infrastructureTopology')
UPDATE_METHODS=$(${CMD} get hco ${HCO_RESOURCE_NAME} -n ${HCO_NAMESPACE} -o jsonpath='{.spec .workloadUpdateStrategy .workloadUpdateMethods}')

if [[ "${INFRASTRUCTURETOPOLOGY}" == "SingleReplica" ]]; then
  echo "Skipping the check on SNO clusters"
elif [[ "${UPDATE_METHODS}" == "" || "${UPDATE_METHODS}" == "[]" ]]; then
  echo "Skipping while workloadUpdateMethods methods are empty "
else
  ./hack/retry.sh 10 30 "[[ \$(${CMD} get vmi -l kubevirt.io/outdatedLauncherImage -A --no-headers | wc -l) -eq 0 ]]" "${CMD} get vmi -l kubevirt.io/outdatedLauncherImage -A"
  echo "All the running VMs got upgraded"
fi

./hack/retry.sh 5 30 "~/virtctl stop testvm -n ${VMS_NAMESPACE}"
${CMD} delete vm -n ${VMS_NAMESPACE} testvm

KUBECTL_BINARY=${CMD} ./hack/test_quick_start.sh

Msg "Check that OVS is deployed or not deployed according to deployOVS annotation in HCO CR."
./hack/retry.sh 40 15 "CMD=${CMD} PREVIOUS_OVS_ANNOTATION=${PREVIOUS_OVS_ANNOTATION}\
 PREVIOUS_OVS_STATE=${PREVIOUS_OVS_STATE} ./hack/check_upgrade_ovs.sh"

Msg "Check that managed objects has correct labels"
./hack/retry.sh 10 30 "KUBECTL_BINARY=${CMD} ./hack/check_labels.sh"

Msg "Check the defaulting mechanism"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} ./hack/check_defaults.sh

Msg "Check that the v2v CRDs and deployments were removed"
if ${CMD} get crd | grep -q v2v.kubevirt.io; then
    echo "The v2v CRDs should not be found; they had to be removed."
    exit 1
else
    echo "v2v CRDs removed"
fi
if ${CMD} get deployments -n ${HCO_NAMESPACE} | grep -q vm-import; then
    echo "v2v deployments should not be found; they had to be removed."
    exit 1
else
    echo "v2v deployments removed"
fi

Msg "Check that the v2v references were removed from .status.relatedObjects"
if ${CMD} -n ${HCO_NAMESPACE} ${HCO_KIND} ${HCO_RESOURCE_NAME} -o=jsonpath={.status.relatedObjects[*].apiVersion} | grep -q v2v.kubevirt.io; then
    echo "v2v references should not be found in relatedObjects; they had to be removed."
    exit 1
else
    echo "v2v references removed from .status.relatedObjects"
fi

Msg "check golden images"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} ./hack/check_golden_images.sh

Msg "check virtio-win image is in configmap"
VIRTIOWIN_IMAGE_CSV=$(${CMD} get ${CSV} -n ${HCO_NAMESPACE} \
  -o jsonpath='{.spec.install.spec.deployments[?(@.name=="hco-operator")].spec.template.spec.containers[0].env[?(@.name=="VIRTIOWIN_CONTAINER")].value}')
VIRTIOWIN_IMAGE_CM=$(${CMD} get cm virtio-win -n ${HCO_NAMESPACE} -o jsonpath='{.data.virtio-win-image}')

[[ "${VIRTIOWIN_IMAGE_CSV}" == "${VIRTIOWIN_IMAGE_CM}" ]]

Msg "Read the HCO operator log before it been deleted"
HCO_POD=$( ${CMD} get -n ${HCO_NAMESPACE} pods -l "name=hyperconverged-cluster-operator" -o name)
${CMD} logs -n ${HCO_NAMESPACE} "${HCO_POD}"

Msg "Read the HCO webhook log before it been deleted"
WH_POD=$( ${CMD} get -n ${HCO_NAMESPACE} pods -l "name=hyperconverged-cluster-webhook" -o name)
${CMD} logs -n ${HCO_NAMESPACE} "${WH_POD}"

Msg "Brutally delete HCO removing the namespace where it's running"
source hack/test_delete_ns.sh
test_delete_ns

echo "upgrade-test completed successfully."
