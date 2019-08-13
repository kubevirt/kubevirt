#!/usr/bin/env bash

if [ -z "$KUBEVIRTCI_PATH" ]; then
    KUBEVIRTCI_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/../"
        echo "$(pwd)/"
    )"
fi


if [ -z "$KUBEVIRTCI_CONFIG_PATH" ]; then
    KUBEVIRTCI_CONFIG_PATH="$(
        cd "$(dirname "$BASH_SOURCE[0]")/../../"
        echo "$(pwd)/_ci-configs"
    )"
fi

KUBEVIRT_PROVIDER=${KUBEVIRT_PROVIDER:-k8s-1.13.3}
KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES:-1}
KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE:-5120M}
KUBEVIRT_NUM_SECONDARY_NICS=${KUBEVIRT_NUM_SECONDARY_NICS:-0}

# If on a developer setup, expose ocp on 8443, so that the openshift web console can be used (the port is important because of auth redirects)
if [ -z "${JOB_NAME}" ]; then
    KUBEVIRT_PROVIDER_EXTRA_ARGS="${KUBEVIRT_PROVIDER_EXTRA_ARGS} --ocp-port 8443"
fi

#If run on jenkins, let us create isolated environments based on the job and
# the executor number
provider_prefix=${JOB_NAME:-${KUBEVIRT_PROVIDER}}${EXECUTOR_NUMBER}
job_prefix=${JOB_NAME:-kubevirt}${EXECUTOR_NUMBER}

provider_prefix=${KUBEVIRT_PROVIDER}${EXECUTOR_NUMBER}
job_prefix=${kubevirt}${EXECUTOR_NUMBER}

mkdir -p $KUBEVIRTCI_CONFIG_PATH/$KUBEVIRT_PROVIDER
