#!/bin/bash

MARKETPLACE="${MARKETPLACE:-openshift-marketplace}"
TARGET_NAMESPACE="${TARGET_NAMESPACE:-kubevirt-hyperconverged}"
APP_REGISTRY="${APP_REGISTRY:-kubevirt-hyperconverged}"
HCO_VERSION="${HCO_VERSION:-1.0.0}"

oc delete csc hco-catalogsource-config -n $MARKETPLACE
oc delete catalogsource $APP_REGISTRY -n $MARKETPLACE
oc delete operatorsource $APP_REGISTRY -n $MARKETPLACE
oc delete hco kubevirt-hyperconverged -n $TARGET_NAMESPACE
sleep 10
oc delete sub hco-operatorhub -n $TARGET_NAMESPACE
oc delete csv kubevirt-hyperconverged-operator.v${HCO_VERSION} -n $TARGET_NAMESPACE
oc delete operatorgroup $TARGET_NAMESPACE-group -n $TARGET_NAMESPACE
oc delete secret $(oc get secret -n $MARKETPLACE | grep quay-registry | awk '{print $1}') -n $MARKETPLACE
