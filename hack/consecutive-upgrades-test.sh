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
# Copyright 2023 Red Hat, Inc.
#
# Usage:
# make upgrade-test-operator-sdk
#
# Use Openshift-CI "optional-operators-ci-*" workflow to:
# - Use the operator-sdk to upgrade a pre deployed bundle; use a new bundle,
#   named 100.0.0 with the contents of the open PR (this can include new
#   dependent images, new CRDs...).
# - the script then performs various validations against the upgraded version.

MAX_STEPS=$(( $(grep -c "Msg " "$0") - 2)) # subtract self line and the function name
CUR_STEP=1
HCO_DEPLOYMENT_NAME=hco-operator
HCO_WH_DEPLOYMENT_NAME=hco-webhook
HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"
MID_VERSION=1.16.0
TARGET_VERSION=100.0.0
VMS_NAMESPACE=vmsns

OUTPUT_DIR=${OUTPUT_DIR:-_out}
OO_MID_BUNDLE=${OO_MID_BUNDLE}
OO_LAST_BUNDLE=${OO_LAST_BUNDLE}

echo "INITIAL_VERSION: $INITIAL_VERSION"

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

export CMD="oc"

echo "oc version"
${CMD} version || true

function cleanup() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during upgrade: exit status: $rv"
        make dump-state
        echo "*** Upgrade test failed ***"
    fi
    exit $rv
}

function upgrade() {
  I_VERSION=$1
  T_VERSION=$2
  BUNDLE=$3

  source ./hack/check-uptime.sh
  sleep 5
  INITIAL_BOOTTIME=$(check_uptime 10 60)

  Msg "HCO deployOVS annotation and OVS state in CNAO CR before the upgrade"
  PREVIOUS_OVS_ANNOTATION=$(${CMD} get ${HCO_KIND} ${HCO_RESOURCE_NAME} -n ${HCO_NAMESPACE} -o jsonpath='{.metadata.annotations.deployOVS}')
  PREVIOUS_OVS_STATE=$(${CMD} get networkaddonsconfigs cluster -o jsonpath='{.spec.ovs}')

  # Before starting the upgrade, make sure the CSV is installed properly.
  Msg "Read the CSV to make sure the deployment is done"
  # Make sure the CSV is in Succeeded phase
  ./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"
  # Make sure the CSV is in the correct version
  ./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .spec.version }' | grep ${I_VERSION}"

  HCO_SUBSCRIPTION=$(${CMD} get subscription -n ${HCO_NAMESPACE} -o name -l operators.coreos.com/community-kubevirt-hyperconverged.${HCO_NAMESPACE}=)
  OLD_INSTALL_PLAN=$(${CMD} -n "${HCO_NAMESPACE}" get "${HCO_SUBSCRIPTION}" -o jsonpath='{.status.installplan.name}')

  Msg "Perform the upgrade, using operator-sdk"
  operator-sdk run bundle-upgrade -n "${HCO_NAMESPACE}" --verbose --timeout=15m "${BUNDLE}" --security-context-config=restricted

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

  ## Verify the subscription has changed to the new version
  #  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
  #  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
  Msg "Verify the subscription's currentCSV and installedCSV have moved to the new version"

  ${CMD} get pods -n ${HCO_NAMESPACE}
  ./hack/retry.sh 30 60 "${CMD} get deployment -n ${HCO_NAMESPACE} | grep ${HCO_DEPLOYMENT_NAME}"

  ${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"
  ${CMD} wait deployment ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"

  Msg "operator conditions during upgrade"
  KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${I_VERSION}"
  KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${T_VERSION}"

  ./hack/retry.sh 30 60 "${CMD} get ${HCO_SUBSCRIPTION} -n ${HCO_NAMESPACE} -o yaml | grep currentCSV   | grep v${T_VERSION}"
  ./hack/retry.sh  2 30 "${CMD} get ${HCO_SUBSCRIPTION} -n ${HCO_NAMESPACE} -o yaml | grep installedCSV | grep v${T_VERSION}"

  Msg "Verify the hyperconverged-cluster deployment is using the new image"

  set -x
  SEARCH_PHRASE="${OPENSHIFT_BUILD_NAMESPACE}/pipeline"
  ./hack/retry.sh 60 30 "${CMD} get -n ${HCO_NAMESPACE} deployment ${HCO_DEPLOYMENT_NAME} -o jsonpath=\"{ .spec.template.spec.containers[0].image }\" | grep ${SEARCH_PHRASE}"

  Msg "Wait that cluster is operational after upgrade"
  timeout 20m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

  # Make sure the CSV is installed properly.
  Msg "Read the CSV to make sure the deployment is done"
  CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${T_VERSION})
  # Make sure the CSV is in Succeeded phase
  ./hack/retry.sh 90 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"
  # Make sure the CSV is in the correct version
  ./hack/retry.sh 30 10 "${CMD} get ${CSV} -n ${HCO_NAMESPACE} -o jsonpath='{ .spec.version }' | grep ${T_VERSION}"

  echo "----- Pod after upgrade"
  Msg "Verify that the hyperconverged-cluster Pod is using the new image"
  ./hack/retry.sh 10 30 "CMD=${CMD} HCO_NAMESPACE=${HCO_NAMESPACE} ./hack/check_pod_upgrade.sh"

  Msg "Verify new operator version reported after the upgrade"
  ./hack/retry.sh 15 30 "CMD=${CMD} HCO_RESOURCE_NAME=${HCO_RESOURCE_NAME} HCO_NAMESPACE=${HCO_NAMESPACE} TARGET_VERSION=${T_VERSION} hack/check_hco_version.sh"

  Msg "Ensure that HCO got upgraded"
  for hco_pod in $( ${CMD} get pods -n ${HCO_NAMESPACE} -l "name=hyperconverged-cluster-operator" --field-selector=status.phase=Running -o name); do
    pod_version=$( ${CMD} get ${hco_pod} -n ${HCO_NAMESPACE} -o json | jq -r '.spec.containers[0].env[] | select(.name=="HCO_KV_IO_VERSION") | .value')
    if [[ ${pod_version} == ${T_VERSION} ]]; then
      found_new_running_hco_pod="true"
    fi
  done

  Msg "operator conditions after upgrade"
  KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${T_VERSION}"

  [[ -n ${found_new_running_hco_pod} ]]

  echo "----- Images after upgrade"
  # TODO: compare all of them with the list of images in RelatedImages in the new CSV
  ${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

  OUTPUT_DIR=${OUTPUT_DIR} dump_sccs_after

  Msg "make sure that the VM is still running, after the upgrade"
  ${CMD} get vm -n ${VMS_NAMESPACE} -o yaml testvm
  ${CMD} get vmi -n ${VMS_NAMESPACE} -o yaml testvm
  ${CMD} get vmi -n ${VMS_NAMESPACE} testvm -o jsonpath='{ .status.phase }' | grep 'Running'
  CURRENT_BOOTTIME=$(check_uptime 10 60)

  if ((INITIAL_BOOTTIME - CURRENT_BOOTTIME > 3)) || ((CURRENT_BOOTTIME - INITIAL_BOOTTIME > 3)); then
      echo "ERROR: The test VM got restarted during the upgrade process."
      exit 1
  else
      echo "The test VM survived the upgrade process."
  fi
}

trap "cleanup" INT TERM EXIT

source hack/compare_scc.sh

Msg "Check that cluster is operational before upgrade"
timeout 10m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

${CMD} get subscription -n ${HCO_NAMESPACE} -o yaml
${CMD} get pods -n ${HCO_NAMESPACE}

Msg "Images before upgrade"
${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

Msg "Get virtctl"
KV_VERSION=$( ${CMD} get kubevirt.kubevirt.io/kubevirt-kubevirt-hyperconverged -n ${HCO_NAMESPACE} -o=jsonpath="{.status.observedKubeVirtVersion}")
ARCH=$(uname -s | tr A-Z a-z)-$(uname -m | sed 's/x86_64/amd64/') || windows-amd64.exe
echo ${ARCH}
curl -L -o ~/virtctl https://github.com/kubevirt/kubevirt/releases/download/${KV_VERSION}/virtctl-${KV_VERSION}-${ARCH}
chmod +x ~/virtctl
###################

ssh-keygen -t ecdsa -f ./hack/test_ssh -q -N ""
cat << END > ./hack/cloud-init.sh
#!/bin/sh
export NEW_USER="cirros"
export SSH_PUB_KEY="$(cat ./hack/test_ssh.pub)"
sudo mkdir /home/\${NEW_USER}/.ssh
sudo echo "\${SSH_PUB_KEY}" > /home/\${NEW_USER}/.ssh/authorized_keys
sudo chown -R \${NEW_USER}: /home/\${NEW_USER}/.ssh
sudo chmod 600 /home/\${NEW_USER}/.ssh/authorized_keys
END

CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep "kubevirt-hyperconverged-operator")

# Patch the default CPU model to ensure a successful live migration
${CMD} patch hco kubevirt-hyperconverged -n ${HCO_NAMESPACE} --type=json -p='[{"op": "add", "path": "/spec/defaultCPUModel", "value": "Westmere"}]'

Msg "operator conditions before upgrade"
source ./hack/check_operator_condition.sh
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} printOperatorCondition "${INITIAL_VERSION}"

### Create a VM ###
Msg "Create a simple VM on the previous version cluster, before the upgrade"
${CMD} get namespace | grep "^${VMS_NAMESPACE}" || ${CMD} create namespace ${VMS_NAMESPACE}
${CMD} get secret -n ${VMS_NAMESPACE} | grep "^testvm-secret" || ${CMD} create secret -n ${VMS_NAMESPACE} generic testvm-secret --from-file=userdata=./hack/cloud-init.sh
${CMD} apply -n ${VMS_NAMESPACE} -f ./hack/vm.yaml
${CMD} get vm -n ${VMS_NAMESPACE} -o yaml testvm
~/virtctl start testvm -n ${VMS_NAMESPACE}
./hack/retry.sh 30 10 "${CMD} get vmi -n ${VMS_NAMESPACE} testvm -o jsonpath='{ .status.phase }' | grep 'Running'"
${CMD} get vmi -n ${VMS_NAMESPACE} -o yaml testvm

upgrade $INITIAL_VERSION $MID_VERSION $OO_MID_BUNDLE
upgrade $MID_VERSION $TARGET_VERSION $OO_LAST_BUNDLE


Msg "make sure that we don't have outdated VMs"

${CMD} get vmim -n ${VMS_NAMESPACE} -o yaml

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

Msg "Read the HCO operator log before it is deleted"
LOG_DIR="${ARTIFACT_DIR}/logs"
mkdir -p "${LOG_DIR}"
HCO_POD=$( ${CMD} get -n ${HCO_NAMESPACE} pods -l "name=hyperconverged-cluster-operator" -o name)
${CMD} logs -n ${HCO_NAMESPACE} "${HCO_POD}" > "${LOG_DIR}/hyperconverged-cluster-operator.log"

Msg "Read the HCO webhook log before it been deleted"
WH_POD=$( ${CMD} get -n ${HCO_NAMESPACE} pods -l "name=hyperconverged-cluster-webhook" -o name)
${CMD} logs -n ${HCO_NAMESPACE} "${WH_POD}" > "${LOG_DIR}/hyperconverged-cluster-webhook.log"

echo "upgrade-test completed successfully."
