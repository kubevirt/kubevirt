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

# Everything should be deleted by now, but just to be sure
namespaces=(default kube-system)
for i in ${namespaces[@]}; do
    ./cluster/kubectl.sh -n ${i} delete deployment -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete services -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete secrets -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete configmaps -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pvc -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pods -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete rolebinding -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete roles -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete serviceaccounts -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
done

./cluster/kubectl.sh delete pv -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
./cluster/kubectl.sh delete validatingwebhookconfiguration -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
./cluster/kubectl.sh delete clusterrolebinding -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
./cluster/kubectl.sh delete clusterroles -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'
./cluster/kubectl.sh delete customresourcedefinitions -l 'operator.cdi.kubevirt.io' -l 'cdi.kubevirt.io'

sleep 2

echo "Done"
