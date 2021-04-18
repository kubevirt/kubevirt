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


MAX_STEPS=15
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


echo "----- HCO deployOVS annotation and OVS state in CNAO CR before the upgrade"
PREVIOUS_OVS_ANNOTATION=$(${CMD} get ${HCO_KIND} ${HCO_RESOURCE_NAME} -n ${HCO_NAMESPACE} -o jsonpath='{.metadata.annotations.deployOVS}')
PREVIOUS_OVS_STATE=$(${CMD} get networkaddonsconfigs cluster -o jsonpath='{.spec.ovs}')

# Before starting the upgrade, make sure the CSV is installed properly.
Msg "Read the CSV to make sure the deployment is done"
# Make sure the CSV is in Succeeded phase
./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"
# Make sure the CSV is in the correct version
./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .spec.version }' | grep ${INITIAL_CHANNEL}"

# When upgrading from 1.3.0, we expect to have the KV configMap, that will be dropped during upgrade
if ${CMD} get cm kubevirt-config -n ${HCO_NAMESPACE}; then
  KV_CM_FOUND=TRUE
fi

# Create a new version based off of latest. The new version appends ".1" to the latest version.
# The new version replaces the hco-operator image from quay.io with the image pushed to the local registry.
# We create a new CSV based off of the latest version and update the replaces attribute so that the new
# version updates the latest version.
# The currentCSV in the package manifest is also updated to point to the new version.

Msg "Patch the subscription to move to the new channel"
HCO_SUBSCRIPTION_NAME=$(${CMD} get subscription -n ${HCO_NAMESPACE} -o name)
${CMD} patch ${HCO_SUBSCRIPTION_NAME} -n ${HCO_NAMESPACE} -p "{\"spec\": {\"channel\": \"${TARGET_CHANNEL}\"}}"  --type merge

# Patch the OperatorGroup to match the required InstallMode of the new version
sleep 60
source hack/patch_og.sh
patch_og ${TARGET_CHANNEL}
sleep 30
CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${INITIAL_CHANNEL})
if [ -n "${CSV}" ] && [ ${OG_PATCHED} -eq 1 ]
then
  ${CMD} delete "${CSV}" -n ${HCO_NAMESPACE}
fi

sleep 30

# Verify the subscription has changed to the new version
#  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
#  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
Msg "Verify the subscription's currentCSV and installedCSV have moved to the new version"

${CMD} get pods -n ${HCO_NAMESPACE}
./hack/retry.sh 30 60 "${CMD} get deployment -n ${HCO_NAMESPACE} | grep ${HCO_DEPLOYMENT_NAME}"

${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"
${CMD} wait deployment ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"

./hack/retry.sh 30 60 "${CMD} get subscriptions -n ${HCO_NAMESPACE} -o yaml | grep currentCSV   | grep v${TARGET_VERSION}"
./hack/retry.sh  2 30 "${CMD} get subscriptions -n ${HCO_NAMESPACE} -o yaml | grep installedCSV | grep v${TARGET_VERSION}"

Msg "Verify the hyperconverged-cluster deployment is using the new image"

set -x
SEARCH_PHRASE="${OPENSHIFT_BUILD_NAMESPACE}/stable"
./hack/retry.sh 60 30 "${CMD} get -n ${HCO_NAMESPACE} deployment ${HCO_DEPLOYMENT_NAME} -o jsonpath=\"{ .spec.template.spec.containers[0].image }\" | grep ${SEARCH_PHRASE}"

Msg "Wait that cluster is operational after upgrade"
timeout 20m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

# Make sure the CSV is installed properly.
Msg "Read the CSV to make sure the deployment is done"
CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE})
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

Msg "Ensure that old SSP operator resources are removed from the cluster"
./hack/retry.sh 5 30 "CMD=${CMD} HCO_RESOURCE_NAME=${HCO_RESOURCE_NAME} HCO_NAMESPACE=${HCO_NAMESPACE} ./hack/check_old_ssp_removed.sh"

[[ -n ${found_new_running_hco_pod} ]]

echo "----- Images after upgrade"
# TODO: compare all of them with the list of images in RelatedImages in the new CSV
${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

dump_sccs_after

KUBECTL_BINARY=${CMD} ./hack/test_quick_start.sh

Msg "Check that OVS is deployed or not deployed according to deployOVS annotation in HCO CR."
./hack/retry.sh 40 15 "CMD=${CMD} PREVIOUS_OVS_ANNOTATION=${PREVIOUS_OVS_ANNOTATION}\
 PREVIOUS_OVS_STATE=${PREVIOUS_OVS_STATE} ./hack/check_upgrade_ovs.sh"

Msg "Check that managed objects has correct labels"
./hack/retry.sh 10 30 "KUBECTL_BINARY=${CMD} ./hack/check_labels.sh"

# If we found the KV config map before the upgrade, let's check that it's not
# exists anymore, and its backup was created.
Msg "Check that the kubevirt-config ConfigMap was removed"
if [[ -n ${KV_CM_FOUND} ]]; then
  if ${CMD} get cm kubevirt-config -n ${HCO_NAMESPACE}; then
    echo "The kubevirt-config ConfigMap should not be found; it had to be removed."
    exit 1
  else
    echo "kubevirt-config ConfigMap was removed"
  fi
  ${CMD} get cm kubevirt-config-backup -n ${HCO_NAMESPACE}
fi

Msg "Brutally delete HCO removing the namespace where it's running"
source hack/test_delete_ns.sh
test_delete_ns

echo "upgrade-test completed successfully."
