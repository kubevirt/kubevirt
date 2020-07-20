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
# Copyright 2020 Red Hat, Inc.
#

MAX_STEPS=3
CUR_STEP=1
RELEASE_DELTA="${RELEASE_DELTA:-0}"
HCO_DEPLOYMENT_NAME=hco-operator

function Msg {
    { set +x; } 2>/dev/null
    echo "--"
    for a in "$@"; do
        echo "Deploy Step ${CUR_STEP}/${MAX_STEPS}: $a"
    done
    echo "--"
    ((CUR_STEP += 1))
    set -x
}

echo "Running on OpenShift CI"
source ./hack/kubevirt-nightly-openshiftci-config

function cleanup() {
    rv=$?
    if [ "x$rv" != "x0" ]; then
        echo "Error during deployment: exit status: $rv"
        make dump-state
        echo "*** Kubevirt-nightly test failed ***"
    fi
    exit $rv
}

trap "cleanup" INT TERM EXIT


${CMD} wait deployment packageserver --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"
${CMD} wait deployment catalog-operator --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"

Msg "create catalogsource and subscription to install HCO"

HCO_NAMESPACE="kubevirt-hyperconverged"
HCO_KIND="hyperconvergeds"
HCO_RESOURCE_NAME="kubevirt-hyperconverged"
PACKAGE_DIR="./deploy/olm-catalog/kubevirt-hyperconverged"
INITIAL_CHANNEL=$(ls -d ${PACKAGE_DIR}/*/ | sort -rV | awk "NR==$((RELEASE_DELTA+1))" | cut -d '/' -f 5)
echo "INITIAL_CHANNEL: $INITIAL_CHANNEL"

${CMD} create ns ${HCO_NAMESPACE} | true
${CMD} get pods -n ${HCO_NAMESPACE}

cat <<EOF | ${CMD} create -f -
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: hco-operatorgroup
  namespace: ${HCO_NAMESPACE}
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
  channel: ${INITIAL_CHANNEL}
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
  namespace: ${HCO_NAMESPACE}
spec:
  channel: ${LATEST_VERSION}
  name: kubevirt-hyperconverged
  source: hco-catalogsource-example
  sourceNamespace: ${HCO_CATALOG_NAMESPACE}
${SUBSCRIPTION_CONFIG}
EOF

# Allow time for the install plan to be created a for the
# hco-operator to be created. Otherwise kubectl wait will report EOF.
./hack/retry.sh 20 30 "${CMD} get subscription -n ${HCO_NAMESPACE} | grep -v EOF"
./hack/retry.sh 20 30 "${CMD} get pods -n ${HCO_NAMESPACE} | grep hco-operator"

${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="1200s"

${CMD} create -f ./deploy/hco.cr.yaml -n ${HCO_NAMESPACE}

${CMD} wait -n ${HCO_NAMESPACE} ${HCO_KIND} ${HCO_RESOURCE_NAME} --for condition=Available --timeout=30m
${CMD} wait deployment ${HCO_DEPLOYMENT_NAME} --for condition=Available -n ${HCO_NAMESPACE} --timeout="30m"

Msg "check that cluster is operational"
timeout 10m bash -c 'export CMD="${CMD}";exec ./hack/check-state.sh' 

${CMD} get subscription -n ${HCO_NAMESPACE} -o yaml
${CMD} get pods -n ${HCO_NAMESPACE} 

echo "----- Image"
${CMD} get deployments -n ${HCO_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy
${CMD} get pod $HCO_CATALOGSOURCE_POD -n ${HCO_CATALOG_NAMESPACE} -o yaml | grep image | grep -v imagePullPolicy

echo "kubevirt-nightly-test completed successfully."
