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

if [[ $TARGET =~ openshift-.* ]]; then
  if [[ $TARGET =~ .*-crio-.* ]]; then
    export KUBEVIRT_PROVIDER="os-3.10.0-crio"
  elif [[ $TARGET =~ .*-multus-.* ]]; then
    export KUBEVIRT_PROVIDER="os-3.10.0-multus"
  else
    export KUBEVIRT_PROVIDER="os-3.10.0"
  fi
elif [[ $TARGET =~ .*-1.10.4-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-1.10.4"
elif [[ $TARGET =~ .*-multus-1.11.1-.* ]]; then
  export KUBEVIRT_PROVIDER="k8s-multus-1.11.1"
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

release_download_lock() { 
  if [[ -e "$1" ]]; then
    rm -f "$1"
    echo "Released lock: $1"
  fi
}

if [[ $TARGET =~ openshift.* ]]; then
    # Create images directory
    if [[ ! -d $RHEL_NFS_DIR ]]; then
        mkdir -p $RHEL_NFS_DIR
    fi

    # Download RHEL image
    if wait_for_download_lock $RHEL_LOCK_PATH; then
        if [[ ! -f "$RHEL_NFS_DIR/disk.img" ]]; then
            curl http://templates.ovirt.org/kubevirt/rhel7.img > $RHEL_NFS_DIR/disk.img
        fi
        release_download_lock $RHEL_LOCK_PATH
    else
        exit 1
    fi
fi

if [[ $TARGET =~ windows.* ]]; then
  # Create images directory
  if [[ ! -d $WINDOWS_NFS_DIR ]]; then
    mkdir -p $WINDOWS_NFS_DIR
  fi

  # Download Windows image
  if wait_for_download_lock $WINDOWS_LOCK_PATH; then
    if [[ ! -f "$WINDOWS_NFS_DIR/disk.img" ]]; then
      curl http://templates.ovirt.org/kubevirt/win01.img > $WINDOWS_NFS_DIR/disk.img
    fi
    release_download_lock $WINDOWS_LOCK_PATH
  else
    exit 1
  fi
fi

kubectl() { cluster/kubectl.sh "$@"; }

export NAMESPACE="${NAMESPACE:-kube-system}"

# Make sure that the VM is properly shut down on exit
trap '{ release_download_lock $RHEL_LOCK_PATH; release_download_lock $WINDOWS_LOCK_PATH; make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

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

make cluster-sync

# OpenShift is running important containers under default namespace
namespaces=(kube-system default)
if [[ $NAMESPACE != "kube-system" ]]; then
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
