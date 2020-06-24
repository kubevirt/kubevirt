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

MAX_STEPS=8
CUR_STEP=1

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


Msg "clean cluster"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  make cluster-clean
fi

"${CMD}" delete -f ./deploy/hco.cr.yaml -n kubevirt-hyperconverged | true
"${CMD}" delete subscription hco-subscription-example -n kubevirt-hyperconverged | true
"${CMD}" delete catalogsource hco-catalogsource-example -n ${HCO_CATALOG_NAMESPACE} | true
"${CMD}" delete operatorgroup hco-operatorgroup -n kubevirt-hyperconverged | true



${CMD} wait deployment packageserver --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"
${CMD} wait deployment catalog-operator --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"

if [ -n "$KUBEVIRT_PROVIDER" ]; then
  Msg "build images for STDCI"
  ./hack/upgrade-test-build-images.sh
else
  Msg "Openshift CI detected." "Image build skipped. Images are built through Prow."
fi

Msg "create catalogsource and subscription to install HCO"

HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"

${CMD} create ns ${HCO_NAMESPACE} | true
${CMD} get pods -n ${HCO_NAMESPACE}

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: hco-operatorgroup
  namespace: kubevirt-hyperconverged
spec:
  targetNamespaces:
  - ${HCO_NAMESPACE}
EOF

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: hco-catalogsource-example
  namespace: ${HCO_CATALOG_NAMESPACE}
spec:
  sourceType: grpc
  image: ${REGISTRY_IMAGE}
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

LATEST_VERSION=$(ls -d ./deploy/olm-catalog/kubevirt-hyperconverged/*/ | sort -r | head -1 | cut -d '/' -f 5);

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-subscription-example
  namespace: kubevirt-hyperconverged
spec:
  channel: ${LATEST_VERSION}
  name: kubevirt-hyperconverged
  source: hco-catalogsource-example
  sourceNamespace: ${HCO_CATALOG_NAMESPACE}
${SUBSCRIPTION_CONFIG}
EOF

# Allow time for the install plan to be created a for the
# hco-operator to be created. Otherwise kubectl wait will report EOF.
./hack/retry.sh 20 30 "${CMD} get subscription -n kubevirt-hyperconverged | grep -v EOF"
./hack/retry.sh 20 30 "${CMD} get pods -n kubevirt-hyperconverged | grep hco-operator"

HCO_OPERATOR_POD=`${CMD} get pods -n kubevirt-hyperconverged | grep hco-operator  | grep 'Running\|ContainerCreating' | head -1 | awk '{ print $1 }'`
${CMD} wait pod $HCO_OPERATOR_POD --for condition=Ready -n kubevirt-hyperconverged --timeout="1200s"

${CMD} create -f ./deploy/hco.cr.yaml -n kubevirt-hyperconverged

HCO_OPERATOR_POD=`${CMD} get pods -n ${HCO_NAMESPACE} | grep hco-operator | grep 'Running\|ContainerCreating' | head -1 | awk '{ print $1 }'`

${CMD} wait -n ${HCO_NAMESPACE} ${HCO_KIND} ${HCO_RESOURCE_NAME} --for condition=Available --timeout=30m
${CMD} wait pod $HCO_OPERATOR_POD --for condition=Ready -n ${HCO_NAMESPACE} --timeout=30m

Msg "check that cluster is operational before upgrade"
timeout 10m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh' 

${CMD} get subscription -n kubevirt-hyperconverged -o yaml
${CMD} get pods -n kubevirt-hyperconverged 

echo "----- Images before upgrade"
${CMD} get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

# Create a new version based off of latest. The new version appends ".1" to the latest version.
# The new version replaces the hco-operator image from quay.io with the image pushed to the local registry.
# We create a new CSV based off of the latest version and update the replaces attribute so that the new
# version updates the latest version.
# The currentCSV in the package manifest is also updated to point to the new version.

Msg "patch existing catalog source with new registry image" "and wait for hco-catalogsource pod to be in Ready state"

# Patch the HCO catalogsource image to the upgrade version
${CMD} patch catalogsource hco-catalogsource-example -n ${HCO_CATALOG_NAMESPACE} -p "{\"spec\": {\"image\": \"${REGISTRY_IMAGE_UPGRADE}\"}}"  --type merge
sleep 5
./hack/retry.sh 20 30 "${CMD} get pods -n ${HCO_CATALOG_NAMESPACE} | grep hco-catalogsource | grep -v Terminating"
HCO_CATALOGSOURCE_POD=`${CMD} get pods -n ${HCO_CATALOG_NAMESPACE} | grep hco-catalogsource | grep -v Terminating | head -1 | awk '{ print $1 }'`
${CMD} wait pod $HCO_CATALOGSOURCE_POD --for condition=Ready -n ${HCO_CATALOG_NAMESPACE} --timeout="1800s"

sleep 15
CATALOG_OPERATOR_POD=`${CMD} get pods -n openshift-operator-lifecycle-manager | grep catalog-operator | head -1 | awk '{ print $1 }'`
${CMD} wait pod $CATALOG_OPERATOR_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

# Verify the subscription has changed to the new version
#  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
#  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
Msg "verify the subscription's currentCSV and installedCSV have moved to the new version"

sleep 60
HCO_OPERATOR_POD=`${CMD} get pods -n kubevirt-hyperconverged | grep hco-operator | head -1 | awk '{ print $1 }'`
${CMD} get pods -n kubevirt-hyperconverged
${CMD} wait pod $HCO_OPERATOR_POD --for condition=Ready -n kubevirt-hyperconverged --timeout="1200s"

./hack/retry.sh 30 60 "${CMD} get subscriptions -n kubevirt-hyperconverged -o yaml | grep currentCSV | grep v100.0.0"
./hack/retry.sh 2 30 "${CMD} get subscriptions -n kubevirt-hyperconverged -o yaml | grep installedCSV | grep v100.0.0"

Msg "verify the hyperconverged-cluster deployment is using the new image"

./hack/retry.sh 6 30 "${CMD} get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep hyperconverged-cluster | grep ${REGISTRY_IMAGE_URL_PREFIX}"

echo "----- Images after upgrade"
${CMD} get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

Msg "wait that cluster is operational after upgrade"
timeout 10m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh'

echo "upgrade-test completed successfully."
