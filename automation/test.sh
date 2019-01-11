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

export WORKSPACE="${WORKSPACE:-$PWD}"
readonly ARTIFACTS_PATH="$WORKSPACE/exported-artifacts"
readonly TEMPLATES_SERVER="https://templates.ovirt.org/kubevirt/"

if [[ $TARGET =~ openshift-.* ]]; then
  # when testing on slow CI system cleanup sometimes takes very long.
  # openshift clusters are more memory demanding. If the cleanup
  # of old vms does not go fast enough they run out of memory.
  # To still allow continuing with the tests, give more memory in CI.
  export KUBEVIRT_MEMORY_SIZE=6144M
  if [[ $TARGET =~ .*-crio-.* ]]; then
    export KUBEVIRT_PROVIDER="os-3.11.0-crio"
  elif [[ $TARGET =~ .*-multus-.* ]]; then
    export KUBEVIRT_PROVIDER="os-3.11.0-multus"
  else
    export KUBEVIRT_PROVIDER="os-3.11.0"
  fi
elif [[ $TARGET =~ .*-1.10.4-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-1.10.11"
elif [[ $TARGET =~ .*-multus-1.12.2-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-multus-1.12.2"
elif [[ $TARGET =~ .*-genie-1.11.1-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-genie-1.11.1"
else
  export KUBEVIRT_PROVIDER="k8s-1.11.0"
fi

export KUBEVIRT_NUM_NODES=2
export RHEL_NFS_DIR=${RHEL_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/rhel7}
export RHEL_LOCK_PATH=${RHEL_LOCK_PATH:-/var/lib/stdci/shared/download_rhel_image.lock}
export WINDOWS_NFS_DIR=${WINDOWS_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/windows2016}
export WINDOWS_LOCK_PATH=${WINDOWS_LOCK_PATH:-/var/lib/stdci/shared/download_windows_image.lock}

wait_for_download_lock() {
  local max_lock_attempts=60
  local lock_wait_interval=60

  for ((i = 0; i < $max_lock_attempts; i++)); do
      if (set -o noclobber; > $1) 2> /dev/null; then
          echo "Acquired lock: $1"
          return
      fi
      sleep $lock_wait_interval
  done
  echo "Timed out waiting for lock: $1" >&2
  exit 1
}

safe_download() (
    # Download files into shared locations using a lock.
    # The lock will be released as soon as this subprocess will exit
    local lockfile="${1:?Lockfile was not specified}"
    local download_from="${2:?Download from was not specified}"
    local download_to="${3:?Download to was not specified}"
    local timeout_sec="${4:-3600}"

    touch "$lockfile"
    exec {fd}< "$lockfile"
    flock -e  -w "$timeout_sec" "$fd" || {
        echo "ERROR: Timed out after $timeout_sec seconds waiting for lock" >&2
        exit 1
    }

    local remote_sha1_url="${download_from}.sha1"
    local local_sha1_file="${download_to}.sha1"
    local remote_sha1
    # Remote file includes only sha1 w/o filename suffix
    remote_sha1="$(curl -s "${remote_sha1_url}")"
    if [[ "$(cat "$local_sha1_file")" != "$remote_sha1" ]]; then
        echo "${download_to} is not up to date, corrupted or doesn't exist."
        echo "Downloading file from: ${remote_sha1_url}"
        curl "$download_from" --output "$download_to"
        sha1sum "$download_to" | cut -d " " -f1 > "$local_sha1_file"
        [[ "$(cat "$local_sha1_file")" == "$remote_sha1" ]] || {
            echo "${download_to} is corrupted"
            return 1
        }
    else
        echo "${download_to} is up to date"
    fi
)

if [[ $TARGET =~ openshift.* ]]; then
    # Create images directory
    if [[ ! -d $RHEL_NFS_DIR ]]; then
        mkdir -p $RHEL_NFS_DIR
    fi

    # Download RHEL image
    rhel_image_url="${TEMPLATES_SERVER}/rhel7.img"
    rhel_image="$RHEL_NFS_DIR/disk.img"
    safe_download "$RHEL_LOCK_PATH" "$rhel_image_url" "$rhel_image" || exit 1
fi

if [[ $TARGET =~ windows.* ]]; then
  # Create images directory
  if [[ ! -d $WINDOWS_NFS_DIR ]]; then
    mkdir -p $WINDOWS_NFS_DIR
  fi

  # Download Windows image
  win_image_url="${TEMPLATES_SERVER}/win01.img"
  win_image="$WINDOWS_NFS_DIR/disk.img"
  safe_download "$WINDOWS_LOCK_PATH" "$win_image_url" "$win_image" || exit 1
fi

kubectl() { cluster/kubectl.sh "$@"; }


# If run on CI use random kubevirt system-namespaces
if [ -n "${JOB_NAME}" ]; then
  export NAMESPACE="kubevirt-system-$(cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 10 | head -n 1)"
  cat >hack/config-local.sh <<EOF
namespace=${NAMESPACE}
EOF
else
  export NAMESPACE="${NAMESPACE:-kubevirt}"
fi


# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP


# Check if we are on a pull request in jenkins.
export KUBEVIRT_CACHE_FROM=${ghprbTargetBranch}
if [ -n "${KUBEVIRT_CACHE_FROM}" ]; then
    make pull-cache
fi

make cluster-down
make cluster-up

# Wait for nodes to become ready
set +e
kubectl get nodes --no-headers
kubectl_rc=$?
while [ $kubectl_rc -ne 0 ] || [ -n "$(kubectl get nodes --no-headers | grep NotReady)" ]; do
    echo "Waiting for all nodes to become ready ..."
    kubectl get nodes --no-headers
    kubectl_rc=$?
    sleep 10
done
set -e

echo "Nodes are ready:"
kubectl get nodes

make cluster-sync-operator

# OpenShift is running important containers under default namespace
namespaces=(kubevirt default)
if [[ $NAMESPACE != "kubevirt" ]]; then
  namespaces+=($NAMESPACE)
fi

timeout=300
sample=30

for i in ${namespaces[@]}; do
  # Wait until kubevirt pods are running
  current_time=0
  while [ -n "$(kubectl get pods -n $i --no-headers | grep -v Running)" ]; do
    echo "Waiting for kubevirt pods to enter the Running state ..."
    kubectl get pods -n $i --no-headers | >&2 grep -v Running || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      exit 1
    fi
  done

  # Make sure all containers are ready
  current_time=0
  while [ -n "$(kubectl get pods -n $i -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false)" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -n $i -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      exit 1
    fi
  done
  kubectl get pods -n $i
done

kubectl version

mkdir -p "$ARTIFACTS_PATH"

ginko_params="--ginkgo.noColor --junit-output=$ARTIFACTS_PATH/tests.junit.xml"

# Prepare PV for Windows testing
if [[ $TARGET =~ windows.* ]]; then
  kubectl create -f - <<EOF
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: disk-windows
  labels:
    kubevirt.io/test: "windows"
spec:
  capacity:
    storage: 30Gi
  accessModes:
    - ReadWriteOnce
  nfs:
    server: "nfs"
    path: /
  storageClassName: local
EOF
  # Run only Windows tests
  ginko_params="$ginko_params --ginkgo.focus=Windows"

elif [[ $TARGET =~ multus.* ]] || [[ $TARGET =~ genie.* ]]; then
  # Run networking tests only (general networking, multus, genie, ...)
  # If multus or genie is not present the test will  be skipped base on per-test checks
  ginko_params="$ginko_params --ginkgo.focus=Networking|VMIlifecycle|Expose|Networkpolicy"
fi

# Prepare RHEL PV for Template testing
if [[ $TARGET =~ openshift-.* ]]; then
  kubectl create -f - <<EOF
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: disk-rhel
  labels:
    kubevirt.io/test: "rhel"
spec:
  capacity:
    storage: 15Gi
  accessModes:
    - ReadWriteOnce
  nfs:
    server: "nfs"
    path: /
  storageClassName: local
EOF
fi


# Run functional tests
FUNC_TEST_ARGS=$ginko_params make functest
