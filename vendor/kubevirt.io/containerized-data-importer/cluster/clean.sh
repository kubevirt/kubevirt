#!/bin/bash -e
  
source ./cluster/gocli.sh
source ./hack/build/config.sh

echo "Cleaning up ..."

# Work around https://github.com/kubernetes/kubernetes/issues/33517
./cluster/kubectl.sh delete ds -l "cdi.kubevirt.io" -n ${NAMESPACE} --cascade=false --grace-period 0 2>/dev/null || :

# Delete all traces of kubevirt
namespaces=(default kube-system ${NAMESPACE})
for i in ${namespaces[@]}; do
    ./cluster/kubectl.sh -n ${i} delete deployment -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete services -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete validatingwebhookconfiguration -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete secrets -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete configmaps -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pv -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pvc -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete ds -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete customresourcedefinitions -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete pods -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete clusterrolebinding -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete rolebinding -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete roles -l 'cdi.kubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete clusterroles -l 'cdikubevirt.io'
    ./cluster/kubectl.sh -n ${i} delete serviceaccounts -l 'cdi.kubevirt.io'
done

sleep 2

echo "Done"
