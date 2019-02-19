#!/bin/bash -e
  
source ./cluster/gocli.sh
source ./hack/build/config.sh

echo "Cleaning up ..."

OPERATOR_CR_MANIFEST=./manifests/generated/cdi-operator-cr.yaml
OPERATOR_MANIFEST=./manifests/generated/cdi-operator.yaml

if [ -f "${OPERATOR_CR_MANIFEST}" ]; then
    if ./cluster/kubectl.sh get crd cdis.cdi.kubevirt.io ; then
        ./cluster/kubectl.sh delete --ignore-not-found -f "${OPERATOR_CR_MANIFEST}"
        ./cluster/kubectl.sh wait cdis.cdi.kubevirt.io/cdi --for=delete | echo "this is fine"
    fi
fi

if [ -f "${OPERATOR_MANIFEST}" ]; then
    ./cluster/kubectl.sh delete --ignore-not-found -f "${OPERATOR_MANIFEST}"
fi

# Work around https://github.com/kubernetes/kubernetes/issues/33517
./cluster/kubectl.sh delete ds -l 'operator.cdi.kubevirt.io' -l "cdi.kubevirt.io" -n ${NAMESPACE} --cascade=false --grace-period 0 2>/dev/null || :

# Delete all traces of kubevirt
namespaces=(default kube-system ${NAMESPACE})
for i in ${namespaces[@]}; do
    ./cluster/kubectl.sh -n ${i} delete deployment -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete services -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete validatingwebhookconfiguration -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete secrets -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete configmaps -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pv -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pvc -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete ds -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete customresourcedefinitions -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pods -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete clusterrolebinding -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete rolebinding -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete roles -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete clusterroles -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete serviceaccounts -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
done

sleep 2

echo "Done"
