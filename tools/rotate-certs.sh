#!/bin/bash
set -euo pipefail

namespace=openshift-cnv
_kubectl="${KUBECTL_BINARY:-oc}"

if ! options=$(getopt -o n: -- "$@")
then
    exit 1
fi

eval set -- "$options"

while true; do
    case "$1" in
    -n)
        shift; # The arg is next in position args
        namespace=$1
        ;;
    --)
        shift
        break
        ;;
    esac
    shift
done
shift $((OPTIND-1))

echo "# Rotating kubemacpool certificates ..."
${_kubectl} --namespace "${namespace}" delete pods -l app=kubemacpool

echo "# Rotating cdi certificates ..."
# first rotate the certificates and CAs
${_kubectl} scale --namespace "${namespace}" --replicas=0 deployment/cdi-operator
${_kubectl} delete secrets --namespace "${namespace}" -l cdi.kubevirt.io
# second restart the pods, so that nothing wrong is cached
${_kubectl} delete pods --namespace "${namespace}" -l cdi.kubevirt.io
# then delete registrations
${_kubectl} delete validatingwebhookconfigurations --ignore-not-found=true  --namespace "${namespace}" cdi-api-datavolume-validate
${_kubectl} delete mutatingwebhookconfigurations --ignore-not-found=true  --namespace "${namespace}" cdi-api-datavolume-mutate

# we could use kubectl get api-resources, but if addons are not ready, we just get a general error from kubectl, which would make the query fail.
if ${_kubectl} get routes ;
then 
    ${_kubectl} delete routes --ignore-not-found=true --namespace "${namespace}" cdi-uploadproxy
fi
# finally restart again, so that all registrations get recreated
for ns in $(kubectl.sh get namespaces --no-headers -o custom-columns=":metadata.name") ;
do
    ${_kubectl} delete pods --namespace "${ns}" -l cdi.kubevirt.io
done
${_kubectl} scale --namespace "${namespace}" --replicas=1 deployment/cdi-operator

echo "# Rotating kubevirt certificates ..."
${_kubectl} delete secrets --namespace "${namespace}" -l kubevirt.io
${_kubectl} delete pods --namespace "${namespace}" -l kubevirt.io

echo "# Rotating SSP certificates ..."
${_kubectl} delete secrets --ignore-not-found=true --namespace "${namespace}" virt-template-validator-certs
${_kubectl} delete pods --namespace "${namespace}" -l kubevirt.io=virt-template-validator
