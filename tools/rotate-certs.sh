#!/bin/bash
set -euo pipefail

namespace=openshift-cnv
cdi_namespace=""
webui_namespace="kubevirt-web-ui"

_kubectl="${KUBECTL_BINARY:-oc}"
if ! options=$(getopt -o n: --long namespace:,cdi-namespace:,webui-namespace: -- "$@")
then
    exit 1
fi

eval set -- "$options"

while true; do
    case "$1" in
    --namespace | -n)
        shift;
        namespace=$1
        ;;
    --cdi-namespace)
        shift;
        cdi_namespace=$1
        ;;
    --webui-namespace)
        shift;
        webui_namespace=$1
        ;;
    --)
        shift
        break
        ;;
    esac
    shift
done
shift $((OPTIND-1))

if [ -z "$cdi_namespace" ]; then
    cdi_namespace=${namespace}
fi

echo "# Rotating kubemacpool certificates ..."
${_kubectl} --namespace "${namespace}" delete pods -l app=kubemacpool

echo "# Rotating cdi certificates ..."
# first rotate the certificates and CAs
${_kubectl} scale --namespace "${cdi_namespace}" --replicas=0 deployment/cdi-operator
${_kubectl} delete secrets --namespace "${cdi_namespace}" -l cdi.kubevirt.io
# second restart the pods, so that nothing wrong is cached
${_kubectl} delete pods --namespace "${cdi_namespace}" -l cdi.kubevirt.io
# then delete registrations
${_kubectl} delete validatingwebhookconfigurations --ignore-not-found=true  --namespace "${cdi_namespace}" cdi-api-datavolume-validate
${_kubectl} delete mutatingwebhookconfigurations --ignore-not-found=true  --namespace "${cdi_namespace}" cdi-api-datavolume-mutate

# we could use kubectl get api-resources, but if addons are not ready, we just get a general error from kubectl, which would make the query fail.
if ${_kubectl} get routes ;
then 
    ${_kubectl} delete routes --ignore-not-found=true --namespace "${cdi_namespace}" cdi-uploadproxy
fi

namespaces=$(${_kubectl} get namespaces --no-headers -o custom-columns=":metadata.name")
for ns in ${namespaces} ;
do
    ${_kubectl} delete pods --namespace "${ns}" -l cdi.kubevirt.io
done
# finally restart again, so that all registrations get recreated
${_kubectl} scale --namespace "${cdi_namespace}" --replicas=1 deployment/cdi-operator

echo "# Rotating kubevirt certificates ..."
${_kubectl} delete secrets --namespace "${namespace}" -l kubevirt.io
${_kubectl} delete pods --namespace "${namespace}" -l kubevirt.io

echo "# Rotating SSP certificates ..."
${_kubectl} delete secrets --ignore-not-found=true --namespace "${namespace}" virt-template-validator-certs
${_kubectl} delete pods --namespace "${namespace}" -l kubevirt.io=virt-template-validator

if (${_kubectl} get crd kwebuis.kubevirt.io >/dev/null 2>&1); then
  # valid for 1.4.Z only
  echo "# Rotating Web UI certificates ..."
  formerVersion=$(${_kubectl} get kwebui kubevirt-web-ui -n ${webui_namespace} -o yaml | grep '  version: '|sed 's/^.*: *\(.*\)$/\1/g')
  echo Detected former Web UI version: ${formerVersion}
  ${_kubectl} patch kwebui kubevirt-web-ui -n ${webui_namespace} --patch '{"spec": {"version": ""}}' --type=merge # undeploy
  while (${_kubectl} get deployment console -n ${webui_namespace} --no-headers=true 2>/dev/null); do
    echo "Waiting for Web UI ..." # to undeploy
    sleep 5
  done
  patch=$(echo '{"spec": {"version": "'${formerVersion}'"}}')
  ${_kubectl} patch kwebui kubevirt-web-ui --patch "${patch}" -n ${webui_namespace} --type=merge # deploy
fi

