#!/bin/bash
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

# CI considerations: $TARGET is used by the jenkins build, to distinguish what to test
# Currently considered $TARGET values:
#     vagrant-dev: Runs all functional tests on a development vagrant setup (deprecated)
#     vagrant-release: Runs all possible functional tests on a release deployment in vagrant (deprecated)
#     kubernetes-dev: Runs all functional tests on a development kubernetes setup
#     kubernetes-release: Runs all functional tests on a release kubernetes setup
#     openshift-release: Runs all functional tests on a release openshift setup
#     TODO: vagrant-tagged-release: Runs all possible functional tests on a release deployment in vagrant on a tagged release

set -ex

export TIMESTAMP=${TIMESTAMP:-1}

export WORKSPACE="${WORKSPACE:-$PWD}"
readonly ARTIFACTS_PATH="${ARTIFACTS-$WORKSPACE/exported-artifacts}"
readonly TEMPLATES_SERVER="https://templates.ovirt.org/kubevirt/"
readonly BAZEL_CACHE="${BAZEL_CACHE:-http://bazel-cache.kubevirt-prow.svc.cluster.local:8080/kubevirt.io/kubevirt}"

if [ -z $TARGET ]; then
  echo "FATAL: TARGET must be non empty"
  exit 1
fi

if [[ $TARGET =~ windows.* ]]; then
  echo "picking the default provider for windows tests"
elif [[ $TARGET =~ cnao ]]; then
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_PROVIDER=${TARGET/-cnao/}
elif [[ $TARGET =~ sig-network ]]; then
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_PROVIDER=${TARGET/-sig-network/}
elif [[ $TARGET =~ sig-storage ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-storage/}
else
  export KUBEVIRT_PROVIDER=${TARGET}
fi

if [ ! -d "cluster-up/cluster/$KUBEVIRT_PROVIDER" ]; then
  echo "The cluster provider $KUBEVIRT_PROVIDER does not exist"
  exit 1
fi

export KUBEVIRT_NUM_NODES=2
# Give the nodes enough memory to run tests in parallel, including tests which involve fedora
export KUBEVIRT_MEMORY_SIZE=9216M

export RHEL_NFS_DIR=${RHEL_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/rhel7}
export RHEL_LOCK_PATH=${RHEL_LOCK_PATH:-/var/lib/stdci/shared/download_rhel_image.lock}
export WINDOWS_NFS_DIR=${WINDOWS_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/windows2016}
export WINDOWS_LOCK_PATH=${WINDOWS_LOCK_PATH:-/var/lib/stdci/shared/download_windows_image.lock}


kubectl() { cluster-up/kubectl.sh "$@"; }

collect_debug_logs() {
    local containers

    containers=( $(docker ps -a --format '{{ .Names }}') )
    for container in "${containers[@]}"; do
        echo "======== $container ========"
        docker logs "$container"
    done
}

export NAMESPACE="${NAMESPACE:-kubevirt}"


# Build and test images with a custom image name prefix
export IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT:-kv-}

timeout=300
sample=30

#mkdir -p "$ARTIFACTS_PATH"
#export KUBEVIRT_E2E_PARALLEL=true
#if [[ $TARGET =~ .*kind.* ]]; then
#  export KUBEVIRT_E2E_PARALLEL=false
#fi
export KUBEVIRT_E2E_PARALLEL=false

ginko_params="--noColor --seed=42"

# Set KUBEVIRT_E2E_FOCUS and KUBEVIRT_E2E_SKIP only if both of them are not already set
if [[ -z ${KUBEVIRT_E2E_FOCUS} && -z ${KUBEVIRT_E2E_SKIP} ]]; then
  if [[ $TARGET =~ windows.* ]]; then
    # Run only Windows tests
    export KUBEVIRT_E2E_FOCUS=Windows
  elif [[ $TARGET =~ (cnao|multus) ]]; then
    export KUBEVIRT_E2E_FOCUS="Multus|Networking|VMIlifecycle|Expose|Macvtap"
  elif [[ $TARGET =~ sig-network ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-network\\]"
  elif [[ $TARGET =~ sig-storage ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-storage\\]"
  elif [[ $TARGET =~ sriov.* ]]; then
    export KUBEVIRT_E2E_FOCUS=SRIOV
  elif [[ $TARGET =~ gpu.* ]]; then
    export KUBEVIRT_E2E_FOCUS=GPU
  elif [[ $TARGET =~ (okd|ocp).* ]]; then
    export KUBEVIRT_E2E_SKIP="SRIOV|GPU"
  else
    export KUBEVIRT_E2E_SKIP="Multus|SRIOV|GPU|Macvtap"
  fi

  if [[ "$KUBEVIRT_STORAGE" == "rook-ceph" ]]; then
    export KUBEVIRT_E2E_FOCUS=rook-ceph
  fi
fi

# If KUBEVIRT_QUARANTINE is not set, do not run quarantined tests. When it is
# set the whole suite (quarantined and stable) will be run.
if [ -z "$KUBEVIRT_QUARANTINE" ]; then
    if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
        KUBEVIRT_E2E_SKIP="${KUBEVIRT_E2E_SKIP}|QUARANTINE"
    else
        KUBEVIRT_E2E_SKIP="QUARANTINE"
    fi
fi

# Run functional tests
FUNC_TEST_ARGS="$ginko_params -dryRun" make functest