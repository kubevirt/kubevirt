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
# Copyright 2019 Red Hat, Inc.
#
# Usage:
# export KUBEVIRT_PROVIDER=okd-4.1
# make cluster-up
# make upgrade-test
#
# Start deploying the HCO cluster using the latest images shipped
# in quay.io with latest tag:
# - quay.io/kubevirt/hyperconverged-cluster-operator:latest
# - quay.io/kubevirt/hco-container-registry:latest
#
# A new bundle, named 100.0.0, is then created with the content of
# the open PR (this can include new dependent images, new CRDs...).
# A new hco-operator image is created based off of the code in the
# current checkout.  
#
# Both the hco-operator image and new registry image is pushed
# to the local registry.
#
# The subscription is checked to verify that it progresses
# to the new version. 
# 
# The hyperconverged-cluster deployment's image is also checked
# to verify that it is updated to the new operator image from 
# the local registry.

MAX_STEPS=19
CUR_STEP=1
RELEASE_DELTA="${RELEASE_DELTA:-1}"
HCO_DEPLOYMENT_NAME=hco-operator
HCO_WH_DEPLOYMENT_NAME=hco-webhook
HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"
HCO_SUBSCRIPTION_NAME="hco-subscription-example"
HCO_CATALOGSOURCE_NAME="hco-catalogsource-example"
HCO_OPERATORGROUP_NAME="hco-operatorgroup"
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

echo "KUBEVIRT_PROVIDER: $KUBEVIRT_PROVIDER"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  echo "Running on STDCI ${KUBEVIRT_PROVIDER}"
  source ./hack/upgrade-stdci-config
else
  echo "Running on OpenShift CI"
  source ./hack/upgrade-openshiftci-config
fi

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


Msg "Clean cluster"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  make cluster-clean
fi

"${CMD}" delete ${HCO_KIND} ${HCO_RESOURCE_NAME} -n ${HCO_NAMESPACE} || true
"${CMD}" delete subscription ${HCO_SUBSCRIPTION_NAME} -n ${HCO_NAMESPACE} || true
"${CMD}" delete catalogsource ${HCO_CATALOGSOURCE_NAME} -n ${HCO_CATALOG_NAMESPACE} || true
"${CMD}" delete operatorgroup ${HCO_OPERATORGROUP_NAME} -n ${HCO_NAMESPACE} || true

source hack/compare_scc.sh
dump_sccs_before

${CMD} wait deployment packageserver --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"
${CMD} wait deployment catalog-operator --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  Msg "Build images for STDCI"
  ./hack/upgrade-test-build-images.sh
else
  Msg "Openshift CI detected." "Image build skipped. Images are built through Prow."
fi

Msg "Create catalogsource and subscription to install HCO"

${CMD} create ns ${HCO_NAMESPACE} || true
${CMD} get pods -n ${HCO_NAMESPACE}

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: ${HCO_OPERATORGROUP_NAME}
  namespace: ${HCO_NAMESPACE}
spec: {}
EOF

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: ${HCO_CATALOGSOURCE_NAME}
  namespace: ${HCO_CATALOG_NAMESPACE}
spec:
  sourceType: grpc
  image: ${REGISTRY_IMAGE_UPGRADE}
  displayName: KubeVirt HyperConverged
  publisher: Red Hat
EOF

sleep 15

HCO_CATALOGSOURCE_POD=`${CMD} get pods -n ${HCO_CATALOG_NAMESPACE} | grep hco-catalogsource | head -1 | awk '{ print $1 }'`
${CMD} wait pod $HCO_CATALOGSOURCE_POD --for condition=Ready -n ${HCO_CATALOG_NAMESPACE} --timeout="120s"

CATALOG_OPERATOR_POD=`${CMD} get pods -n openshift-operator-lifecycle-manager | grep catalog-operator | head -1 | awk '{ print $1 }'`
${CMD} wait pod $CATALOG_OPERATOR_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

PACKAGESERVER_POD=`${CMD} get pods -n openshift-operator-lifecycle-manager | grep packageserver | head -1 | awk '{ print $1 }'`
${CMD} wait pod $PACKAGESERVER_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

# Creating a subscription immediately after the catalog
# source is ready can cause delays. Sometimes the catalog-operator
# isn't ready to create the install plan. As a temporary workaround
# we wait for 15 seconds here.
sleep 15

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-subscription-example
  namespace: ${HCO_NAMESPACE}
spec:
  channel: ${INITIAL_CHANNEL}
  name: community-kubevirt-hyperconverged
  source: ${HCO_CATALOGSOURCE_NAME}
  sourceNamespace: ${HCO_CATALOG_NAMESPACE}
${SUBSCRIPTION_CONFIG}
EOF

# Allow time for the install plan to be created a for the
# hco-operator to be created. Otherwise kubectl wait will report EOF.
./hack/retry.sh 20 30 "${CMD} get subscription -n ${HCO_NAMESPACE} | grep -v EOF"

# Wait for the CSV to be created
./hack/retry.sh 20 30 "${CMD} get csv -n ${HCO_NAMESPACE} | grep -v EOF"
# Adjust the OperatorGroup to the supported InstallMode of the CSV
source hack/patch_og.sh
patch_og ${INITIAL_CHANNEL}

./hack/retry.sh 20 30 "${CMD} get pods -n ${HCO_NAMESPACE} | grep hco-operator"

${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} ${HCO_WH_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"

# Creating a CR immediately after HCO pod started can
# cause a connection error "validate-hco.kubevirt.io" webhook.
# Give it a bit of time to correctly start the webhook.
sleep 30
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
./hack/retry.sh 30 10 "${CMD} get ClusterServiceVersion  -n ${HCO_NAMESPACE} kubevirt-hyperconverged-operator.v${INITIAL_CHANNEL} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"

# Create a new version based off of latest. The new version appends ".1" to the latest version.
# The new version replaces the hco-operator image from quay.io with the image pushed to the local registry.
# We create a new CSV based off of the latest version and update the replaces attribute so that the new
# version updates the latest version.
# The currentCSV in the package manifest is also updated to point to the new version.

Msg "Patch the subscription to move to the new channel"
${CMD} patch subscription ${HCO_SUBSCRIPTION_NAME} -n ${HCO_NAMESPACE} -p "{\"spec\": {\"channel\": \"${TARGET_CHANNEL}\"}}"  --type merge

# Verify the subscription has changed to the new version
#  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
#  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
Msg "Verify the subscription's currentCSV and installedCSV have moved to the new version"

sleep 60
patch_og ${TARGET_CHANNEL}
sleep 30
CSV=$( ${CMD} get csv -o name -n ${HCO_NAMESPACE} | grep ${INITIAL_CHANNEL})
if [ -n "${CSV}" ] && [ ${OG_PATCHED} -eq 1 ]
then
  ${CMD} delete "${CSV}" -n ${HCO_NAMESPACE}
fi

sleep 30

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
./hack/retry.sh 90 10 "${CMD} get ClusterServiceVersion  -n ${HCO_NAMESPACE} kubevirt-hyperconverged-operator.v${TARGET_VERSION} -o jsonpath='{ .status.phase }' | grep 'Succeeded'"


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

Msg "Check the defaulting mechanism"
KUBECTL_BINARY=${CMD} INSTALLED_NAMESPACE=${HCO_NAMESPACE} ./hack/check_defaults.sh

######
# TODO: remove this, workaround for https://issues.redhat.com/browse/OCPBUGS-2219
${CMD} patch ConsolePlugin kubevirt-plugin -o yaml --type=json -p '[{ "op": "replace", "path": "/spec/i18n/loadType", "value": "Preload" }]' || true
sleep 3
######

Msg "Brutally delete HCO removing the namespace where it's running"
source hack/test_delete_ns.sh
test_delete_ns

echo "upgrade-test completed successfully."
