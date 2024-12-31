#!/bin/bash -ex

source ./hack/common.sh
source ./_kubevirtci/cluster-up/cluster/ephemeral-provider-common.sh

function set_env() {
    if [ ${KUBEVIRT_PROVIDER} != "external" ]; then
        registry_port=$(${_cri_bin} ps | grep -Po '(?<=0.0.0.0:)\d+(?=->5000\/tcp)' | head -n 1)
        if [ -z "$registry_port" ]; then
            >&2 echo "unable to get the registry port"
            exit 1
        fi
        export IMAGE_REGISTRY=localhost:$registry_port
        export REGISTRY=registry:5000
        export CMD="./cluster/kubectl.sh"
    else
        if [ "${REGISTRY_NAMESPACE}" == "kubevirt" ]; then
            echo "REGISTRY_NAMESPACE cant be kubevirt when using KUBEVIRT_PROVIDER=external"
            exit 1
        fi
        export REGISTRY=$IMAGE_REGISTRY
        export CMD="oc"
    fi
}

component_name=$1
if [[ "${component_name}" != "operator" && "${component_name}" != "webhook" ]]; then
  echo "must use $0 with \"operator\" or \"webhook\""
  exit 1
fi

deployment_name="hyperconverged-cluster-${component_name}"

set_env

# get original number of replicas
replicas=$(${CMD} get deployment -n kubevirt-hyperconverged ${deployment_name} -o jsonpath='{ .spec.replicas }')
# make sure the restarting the pod, will force it to use the new image
num_cont=$(${CMD} get deployment -n kubevirt-hyperconverged ${deployment_name} -o json | jq '.spec.template.spec.containers | length')
for i in  $(seq $num_cont); do
  ind=$((i-1))
  ${CMD} patch deployment -n kubevirt-hyperconverged ${deployment_name} --type=json --patch '[{"op": "replace", "path": "/spec/template/spec/containers/'${ind}'/imagePullPolicy", "value": "Always"}]'
done
# rebuild the image and replace it in the registry
make container-build-${component_name} container-push-${component_name}
# restarting the pod, to force taking the new image
${CMD} scale deployment -n kubevirt-hyperconverged ${deployment_name} --replicas=0
${CMD} scale deployment -n kubevirt-hyperconverged ${deployment_name} --replicas=${replicas}
# make sure the new deployment is ready
${CMD} wait deployment -n kubevirt-hyperconverged ${deployment_name} --for=condition=Available
