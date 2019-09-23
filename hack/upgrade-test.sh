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
# Copyright 2017 Red Hat, Inc.
#
# Usage:
# export KUBEVIRT_PROVIDER=okd-4.1
# make cluster-up
# make upgrade-test
#
# Deploys the HCO cluster using the latest version in the git repo.
# This latest version deploys the hco-operator using the :latest tag 
# on quay.io.
#
# A new version, named 100.0.0, is then created. A new hco-operator
# image is created based off of the code in the current checkout.  
# A CSV and registry image is created for this version. The CSV
# uses the new hco-operator image in the hco deployment.
#
# Both the hco-operator image and new registry image is pushed
# to the local registry.
#
# The hco-catalogsource pod is then patched to use the new registry
# image.
#
# The subscription is checked to verify that it progresses
# to the new version. 
# 
# The hyperconverged-cluster deployment's image is also checked
# to verify that it is updated to the new operator image from 
# the local registry.

echo "-- Upgrade Step 1/8: clean cluster"
make cluster-clean

echo "-- Upgrade Step 2/8: build registry image (tag:latest)"

container_id=$(docker ps | grep kubevirtci | cut -d ' ' -f 1)
registry_port=$(docker port $container_id | grep 5000 | cut -d ':' -f 2)
registry=localhost:$registry_port

echo "INFO: registry: $registry"

export REGISTRY_NAMESPACE=kubevirt
export IMAGE_REGISTRY=$registry
export CONTAINER_TAG=latest
make bundleRegistry

# check images are accessible
CLUSTER_NODES=$(./cluster-up/kubectl.sh get nodes | grep Ready | cut -d ' ' -f 1)
for NODE in $CLUSTER_NODES; do
    ./cluster-up/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hco-registry:latest'
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster-up/ssh.sh $NODE 'sudo sysctl -w user.max_user_namespaces=1024'
done

./cluster-up/kubectl.sh wait deployment packageserver --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"
./cluster-up/kubectl.sh wait deployment catalog-operator --for condition=Available -n openshift-operator-lifecycle-manager --timeout="1200s"

echo "-- Upgrade Step 3/8: create catalogsource and subscription to install HCO"

./cluster-up/kubectl.sh create ns kubevirt-hyperconverged | true

cat <<EOF | ./cluster-up/kubectl.sh create -f -
apiVersion: operators.coreos.com/v1alpha2
kind: OperatorGroup
metadata:
  name: hco-operatorgroup
  namespace: kubevirt-hyperconverged
EOF

# TODO: The catalog source image here should point to the latest version in quay.io
# once that is published.
cat <<EOF | ./cluster-up/kubectl.sh create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: CatalogSource
metadata:
  name: hco-catalogsource-example
  namespace: openshift-operator-lifecycle-manager
spec:
  sourceType: grpc
  image: registry:5000/kubevirt/hco-registry
  displayName: KubeVirt HyperConverged
  publisher: Red Hat
EOF

sleep 15

HCO_CATALOGSOURCE_POD=`./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep hco-catalogsource | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $HCO_CATALOGSOURCE_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

CATALOG_OPERATOR_POD=`./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep catalog-operator | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $CATALOG_OPERATOR_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

PACKAGESERVER_POD=`./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep packageserver | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $PACKAGESERVER_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

# Creating a subscription immediately after the catalog
# source is ready can cause delays. Sometimes the catalog-operator
# isn't ready to create the install plan. As a temporary workaround
# we wait for 15 seconds here. 
sleep 15

cat <<EOF | ./cluster-up/kubectl.sh create -f -
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: hco-subscription-example
  namespace: kubevirt-hyperconverged
spec:
  channel: 0.0.3
  name: kubevirt-hyperconverged
  source: hco-catalogsource-example
  sourceNamespace: openshift-operator-lifecycle-manager
EOF

# Allow time for the install plan to be created a for the
# hco-operator to be created. Otherwise kubectl wait will report EOF.
./hack/retry.sh 20 30 "./cluster-up/kubectl.sh get subscription -n kubevirt-hyperconverged | grep -v EOF"
./hack/retry.sh 20 30 "./cluster-up/kubectl.sh get pods -n kubevirt-hyperconverged | grep hco-operator"

HCO_OPERATOR_POD=`./cluster-up/kubectl.sh get pods -n kubevirt-hyperconverged | grep hco-operator | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $HCO_OPERATOR_POD --for condition=Ready -n kubevirt-hyperconverged --timeout="600s"
./cluster-up/kubectl.sh get pods -n kubevirt-hyperconverged 

./cluster-up/kubectl.sh create -f ./deploy/hco.cr.yaml -n kubevirt-hyperconverged

./cluster-up/kubectl.sh get subscription -n kubevirt-hyperconverged -o yaml

echo "----- Images before upgrade"
./cluster-up/kubectl.sh get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep -v imagePullPolicy
./cluster-up/kubectl.sh get pod $HCO_CATALOGSOURCE_POD -n openshift-operator-lifecycle-manager -o yaml | grep image | grep -v imagePullPolicy

# Create a new version based off of latest. The new version appends ".1" to the latest version.
# The new version replaces the hco-operator image from quay.io with the image pushed to the local registry.
# We create a new CSV based off of the latest version and update the replaces attribute so that the new
# version updates the latest version.
# The currentCSV in the package manifest is also updated to point to the new version.

LATEST_VERSION=`ls -d ./deploy/olm-catalog/kubevirt-hyperconverged/*/ | sort -r | head -1 | cut -d '/' -f 5`
UPGRADE_VERSION=100.0.0

echo "-- Upgrade Step 4/8: create version $UPGRADE_VERSION, the target for upgrade"

cp -r ./deploy/olm-catalog/kubevirt-hyperconverged/$LATEST_VERSION ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION

mv ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$LATEST_VERSION.clusterserviceversion.yaml ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml
sed -i "s|name: kubevirt-hyperconverged-operator.v$LATEST_VERSION|name: kubevirt-hyperconverged-operator.v$UPGRADE_VERSION|g" ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml
REPLACES_LINE=`grep "replaces" ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml`
sed -i "s|$REPLACES_LINE|  replaces: kubevirt-hyperconverged-operator.v$LATEST_VERSION|g" ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml
sed -i "s|  version: $LATEST_VERSION|  version: $UPGRADE_VERSION|g" ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml
sed -i "s|quay.io/kubevirt/hyperconverged-cluster-operator:latest|registry:5000/kubevirt/hyperconverged-cluster-operator:latest|g" ./deploy/olm-catalog/kubevirt-hyperconverged/$UPGRADE_VERSION/kubevirt-hyperconverged-operator.v$UPGRADE_VERSION.clusterserviceversion.yaml

sed -i "s|currentCSV: kubevirt-hyperconverged-operator.v$LATEST_VERSION|currentCSV: kubevirt-hyperconverged-operator.v$UPGRADE_VERSION|g" ./deploy/olm-catalog/kubevirt-hyperconverged/kubevirt-hyperconverged.package.yaml

echo "-- Upgrade Step 5/8: build new HCO operator image and HCO registry image (tag:upgrade) and push to local registry"

# Build a new registry image for the new version.
export CONTAINER_TAG=upgrade
make container-build-operator container-push-operator bundleRegistry
CLUSTER_NODES=$(./cluster-up/kubectl.sh get nodes | grep Ready | cut -d ' ' -f 1)
for NODE in $CLUSTER_NODES; do
    ./cluster-up/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hyperconverged-cluster-operator'
    ./cluster-up/ssh.sh $NODE 'sudo podman pull registry:5000/kubevirt/hco-registry:upgrade'
    # Temporary until image is updated with provisioner that sets this field
    # This field is required by buildah tool
    ./cluster-up/ssh.sh $NODE 'sudo sysctl -w user.max_user_namespaces=1024'
done

echo "-- Upgrade Step 6/8: patch existing catalog source with new registry image"
echo "-- and wait for hco-catalogsource pod to be in Ready state"

# Patch the HCO catalogsource image to the upgrade version
./cluster-up/kubectl.sh patch catalogsource hco-catalogsource-example -n openshift-operator-lifecycle-manager -p '{"spec":{"image": "registry:5000/kubevirt/hco-registry:upgrade"}}' --type merge
sleep 5
./hack/retry.sh 20 30 "./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep hco-catalogsource | grep -v Terminating"
HCO_CATALOGSOURCE_POD=`./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep hco-catalogsource | grep -v Terminating | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $HCO_CATALOGSOURCE_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

sleep 15
CATALOG_OPERATOR_POD=`./cluster-up/kubectl.sh get pods -n openshift-operator-lifecycle-manager | grep catalog-operator | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $CATALOG_OPERATOR_POD --for condition=Ready -n openshift-operator-lifecycle-manager --timeout="120s"

# Verify the subscription has changed to the new version
#  currentCSV: kubevirt-hyperconverged-operator.v100.0.0
#  installedCSV: kubevirt-hyperconverged-operator.v100.0.0
echo "-- Upgrade Step 7/8: verify the subscription's currentCSV and installedCSV have moved to the new version"
sleep 10
HCO_OPERATOR_POD=`./cluster-up/kubectl.sh get pods -n kubevirt-hyperconverged | grep hco-operator | head -1 | awk '{ print $1 }'`
./cluster-up/kubectl.sh wait pod $HCO_OPERATOR_POD --for condition=Ready -n kubevirt-hyperconverged --timeout="600s"
./hack/retry.sh 30 60 "./cluster-up/kubectl.sh get subscriptions -n kubevirt-hyperconverged -o yaml | grep currentCSV | grep v100.0.0"
./hack/retry.sh 2 30 "./cluster-up/kubectl.sh get subscriptions -n kubevirt-hyperconverged -o yaml | grep installedCSV | grep v100.0.0"

# Verify hco-operator has updated to the new version from the local registry
# registry:5000/kubevirt/hyperconverged-cluster-operator:latest
echo "-- Upgrade Step 8/8: verify the hyperconverged-cluster deployment is using the new image from local registry"
./hack/retry.sh 2 30 "./cluster-up/kubectl.sh get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep hyperconverged-cluster | grep registry:5000"

echo "----- Images after upgrade"
./cluster-up/kubectl.sh get deployments -n kubevirt-hyperconverged -o yaml | grep image | grep -v imagePullPolicy
./cluster-up/kubectl.sh get pod $HCO_CATALOGSOURCE_POD -n openshift-operator-lifecycle-manager -o yaml | grep image | grep -v imagePullPolicy
