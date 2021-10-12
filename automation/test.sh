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

export KUBEVIRT_DEPLOY_CDI=true
if [[ $TARGET =~ windows.* ]]; then
  echo "picking the default provider for windows tests"
elif [[ $TARGET =~ cnao ]]; then
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_PROVIDER=${TARGET/-cnao/}
  export KUBEVIRT_DEPLOY_CDI=false
elif [[ $TARGET =~ sig-network ]]; then
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_PROVIDER=${TARGET/-sig-network/}
  export KUBEVIRT_DEPLOY_ISTIO=true
  export KUBEVIRT_DEPLOY_CDI=false
  if [[ $TARGET =~ k8s-1\.1.* ]]; then
    export KUBEVIRT_DEPLOY_ISTIO=false
  fi
elif [[ $TARGET =~ sig-storage ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-storage/}
  export KUBEVIRT_STORAGE="rook-ceph-default"
elif [[ $TARGET =~ sig-compute-realtime ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-compute-realtime/}
  export KUBEVIRT_HUGEPAGES_2M=512
  export KUBEVIRT_REALTIME_SCHEDULER=true
elif [[ $TARGET =~ sig-compute ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-compute/}
elif [[ $TARGET =~ sig-operator ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-operator/}
else
  export KUBEVIRT_PROVIDER=${TARGET}
fi

if [ ! -d "cluster-up/cluster/$KUBEVIRT_PROVIDER" ]; then
  echo "The cluster provider $KUBEVIRT_PROVIDER does not exist"
  exit 1
fi

if [[ $TARGET =~ sriov.* ]]; then
  export KUBEVIRT_NUM_NODES=3
  export KUBEVIRT_DEPLOY_CDI="false"
elif [[ $TARGET =~ vgpu.* ]]; then
  export KUBEVIRT_NUM_NODES=1
else
  export KUBEVIRT_NUM_NODES=${KUBEVIRT_NUM_NODES:-2}
fi

# Give the nodes enough memory to run tests in parallel, including tests which involve fedora
export KUBEVIRT_MEMORY_SIZE=${KUBEVIRT_MEMORY_SIZE:-9216M}

export RHEL_NFS_DIR=${RHEL_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/rhel7}
export RHEL_LOCK_PATH=${RHEL_LOCK_PATH:-/var/lib/stdci/shared/download_rhel_image.lock}
export WINDOWS_NFS_DIR=${WINDOWS_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/windows2016}
export WINDOWS_LOCK_PATH=${WINDOWS_LOCK_PATH:-/var/lib/stdci/shared/download_windows_image.lock}
export WINDOWS_SYSPREP_NFS_DIR=${WINDOWS_SYSPREP_NFS_DIR:-/var/lib/stdci/shared/kubevirt-images/windows2012_syspreped}
export WINDOWS_SYSPREP_LOCK_PATH=${WINDOWS_SYSPREP_LOCK_PATH:-/var/lib/stdci/shared/download_windows_syspreped_image.lock}

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
    local retry=3
    # Remote file includes only sha1 w/o filename suffix
    for i in $(seq 1 $retry);
    do
      remote_sha1="$(curl -s "${remote_sha1_url}")"
      if [[ "$remote_sha1" != "" ]]; then
        break
      fi
    done

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

if [[ $TARGET =~ os-.* ]] || [[ $TARGET =~ (okd|ocp)-.* ]]; then
    # Create images directory
    if [[ ! -d $RHEL_NFS_DIR ]]; then
        mkdir -p $RHEL_NFS_DIR
    fi

    # Download RHEL image
    rhel_image_url="${TEMPLATES_SERVER}/rhel7.img"
    rhel_image="$RHEL_NFS_DIR/disk.img"
    safe_download "$RHEL_LOCK_PATH" "$rhel_image_url" "$rhel_image" || exit 1
fi

if [[ $TARGET =~ windows_sysprep.* ]]; then
  # Create images directory
  if [[ ! -d $WINDOWS_SYSPREP_NFS_DIR ]]; then
    mkdir -p $WINDOWS_SYSPREP_NFS_DIR
  fi

  # Download Windows image
  win_image_url="${TEMPLATES_SERVER}/windows2012_syspreped.img"
  win_image="$WINDOWS_SYSPREP_NFS_DIR/disk.img"
  safe_download "$WINDOWS_SYSPREP_LOCK_PATH" "$win_image_url" "$win_image" || exit 1
elif [[ $TARGET =~ windows.* ]]; then
  # Create images directory
  if [[ ! -d $WINDOWS_NFS_DIR ]]; then
    mkdir -p $WINDOWS_NFS_DIR
  fi

  # Download Windows image
  win_image_url="${TEMPLATES_SERVER}/win01.img"
  win_image="$WINDOWS_NFS_DIR/disk.img"
  safe_download "$WINDOWS_LOCK_PATH" "$win_image_url" "$win_image" || exit 1
fi

kubectl() { KUBEVIRTCI_VERBOSE=false cluster-up/kubectl.sh "$@"; }
cli() { cluster-up/cli.sh "$@"; }

determine_cri_bin() {
    if [ "${KUBEVIRTCI_RUNTIME}" = "podman" ]; then
        echo podman
    elif [ "${KUBEVIRTCI_RUNTIME}" = "docker" ]; then
        echo docker
    else
        if curl --unix-socket /${HOME}/podman.sock http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
            echo podman
        elif docker ps >/dev/null; then
            echo docker
        else
            >&2 echo "no working container runtime found. Neither docker nor podman seems to work."
            exit 1
        fi
    fi
}

collect_debug_logs() {
    local containers

    local cri_bin="$(determine_cri_bin)"

    containers=( $("${cri_bin}" ps -a --format '{{ .Names }}') )
    for container in "${containers[@]}"; do
        echo "======== $container ========"
        "${cri_bin}" logs "$container"
    done
}

build_images() {
    # build all images with the basic repeat logic
    # probably because load on the node, possible situation when the bazel
    # fails to download artifacts, to avoid job fails because of it,
    # we repeat the build images action
    local tries=3
    for i in $(seq 1 $tries); do
        make bazel-build-images && return
        rc=$?
    done

    return $rc
}

export NAMESPACE="${NAMESPACE:-kubevirt}"

# Make sure that the VM is properly shut down on exit
trap '{ make cluster-down; }' EXIT SIGINT SIGTERM SIGSTOP

make cluster-down

# Create .bazelrc to use remote cache
cat >ci.bazelrc <<EOF
build --jobs=4
build --remote_download_toplevel
EOF

# Build and test images with a custom image name prefix
export IMAGE_PREFIX_ALT=${IMAGE_PREFIX_ALT:-kv-}

build_images

trap '{ collect_debug_logs; echo "Dump kubevirt state:"; make dump; }' ERR
make cluster-up
trap - ERR

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
namespaces=(kubevirt default)
if [[ $NAMESPACE != "kubevirt" ]]; then
  namespaces+=($NAMESPACE)
fi

timeout=300
sample=30

for i in ${namespaces[@]}; do
  # Wait until kubevirt pods are running or completed
  current_time=0
  while [ -n "$(kubectl get pods -n $i --no-headers | grep -v -E 'Running|Completed')" ]; do
    echo "Waiting for kubevirt pods to enter the Running/Completed state ..."
    kubectl get pods -n $i --no-headers | >&2 grep -v -E 'Running|Completed' || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      echo "Dump kubevirt state:"
      make dump
      exit 1
    fi
  done

  # Make sure all containers are ready
  current_time=0
  while [ -n "$(kubectl get pods -n $i --field-selector=status.phase==Running -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false)" ]; do
    echo "Waiting for KubeVirt containers to become ready ..."
    kubectl get pods -n $i --field-selector=status.phase==Running -o'custom-columns=status:status.containerStatuses[*].ready' --no-headers | grep false || true
    sleep $sample

    current_time=$((current_time + sample))
    if [ $current_time -gt $timeout ]; then
      echo "Dump kubevirt state:"
      make dump
      exit 1
    fi
  done
  kubectl get pods -n $i
done

kubectl version

mkdir -p "$ARTIFACTS_PATH"
export KUBEVIRT_E2E_PARALLEL=true
if [[ $TARGET =~ .*kind.* ]]; then
  export KUBEVIRT_E2E_PARALLEL=false
fi

ginko_params="--noColor --seed=42"

# Prepare PV for Windows testing
if [[ $TARGET =~ windows.* ]]; then
  if [[ $TARGET =~ windows_sysprep.* ]]; then
    disk_name=disk-windows-sysprep
    os_label=windows-sysprep
  else
    disk_name=disk-windows
    os_label=windows
  fi
  kubectl create -f - <<EOF
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: $disk_name
  labels:
    kubevirt.io/test: $os_label
spec:
  capacity:
    storage: 35Gi
  accessModes:
    - ReadWriteOnce
  nfs:
    server: "nfs"
    path: /
  storageClassName: windows
EOF
fi

# Set KUBEVIRT_E2E_FOCUS and KUBEVIRT_E2E_SKIP only if both of them are not already set
if [[ -z ${KUBEVIRT_E2E_FOCUS} && -z ${KUBEVIRT_E2E_SKIP} ]]; then
  if [[ $TARGET =~ windows_sysprep.* ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[Sysprep\\]"
  elif [[ $TARGET =~ windows.* ]]; then
    # Run only Windows tests
    export KUBEVIRT_E2E_FOCUS=Windows
  elif [[ $TARGET =~ (cnao|multus) ]]; then
    export KUBEVIRT_E2E_FOCUS="Multus|Networking|VMIlifecycle|Expose|Macvtap"
  elif [[ $TARGET =~ sig-network ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-network\\]"
  elif [[ $TARGET =~ sig-storage ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-storage\\]|\\[rook-ceph\\]"
  elif [[ $TARGET =~ vgpu.* ]]; then
    export KUBEVIRT_E2E_FOCUS=MediatedDevices
  elif [[ $TARGET =~ sig-compute-realtime ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-compute-realtime\\]"
  elif [[ $TARGET =~ sig-compute ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-compute\\]"
    export KUBEVIRT_E2E_SKIP="GPU|MediatedDevices"
  elif [[ $TARGET =~ sig-operator ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[sig-operator\\]"
  elif [[ $TARGET =~ sriov.* ]]; then
    export KUBEVIRT_E2E_FOCUS=SRIOV
  elif [[ $TARGET =~ gpu.* ]]; then
    export KUBEVIRT_E2E_FOCUS=GPU
  elif [[ $TARGET =~ (okd|ocp).* ]]; then
    export KUBEVIRT_E2E_SKIP="SRIOV|GPU|MediatedDevices"
  else
    export KUBEVIRT_E2E_SKIP="Multus|SRIOV|GPU|Macvtap|MediatedDevices"
  fi

  if ! [[ $TARGET =~ sig-storage ]]; then
    if [[ "$KUBEVIRT_STORAGE" == "rook-ceph-default" ]]; then
        export KUBEVIRT_E2E_FOCUS=rook-ceph
    fi
  fi
fi

if [[ $KUBEVIRT_NONROOT =~ true ]]; then
  if [[ -z $KUBEVIRT_E2E_FOCUS ]]; then
    export KUBEVIRT_E2E_FOCUS="\\[verify-nonroot\\]"
  else
    export KUBEVIRT_E2E_FOCUS="$KUBEVIRT_E2E_FOCUS|\\[verify-nonroot\\]"
  fi
else
  if [[ -z $KUBEVIRT_E2E_SKIP ]]; then
    export KUBEVIRT_E2E_SKIP="\\[verify-nonroot\\]"
  else
    export KUBEVIRT_E2E_SKIP="$KUBEVIRT_E2E_SKIP|\\[verify-nonroot\\]"
  fi
fi

# If KUBEVIRT_QUARANTINE is not set, do not run quarantined tests. When it is
# set the whole suite (quarantined and stable) will be run.
if [ -z "$KUBEVIRT_QUARANTINE" ]; then
    if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
        export KUBEVIRT_E2E_SKIP="${KUBEVIRT_E2E_SKIP}|QUARANTINE"
    else
        export KUBEVIRT_E2E_SKIP="QUARANTINE"
    fi
    # quarantine test_id:3145 only for nonroot lanes
    if [[ $KUBEVIRT_NONROOT =~ true ]]; then
        export KUBEVIRT_E2E_SKIP="${KUBEVIRT_E2E_SKIP}|test_id:3145"
    fi
fi

# Prepare RHEL PV for Template testing
if [[ $TARGET =~ os-.* ]]; then
  ginko_params="$ginko_params|Networkpolicy"

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
  storageClassName: rhel
EOF
fi


# Run functional tests
FUNC_TEST_ARGS=$ginko_params make functest

# Run REST API coverage based on k8s audit log and openapi spec
if [ -n "$RUN_REST_COVERAGE" ]; then
  echo "Generating REST API coverage report"
  wget https://github.com/mfranczy/crd-rest-coverage/releases/download/v0.1.3/rest-coverage -O _out/rest-coverage
  chmod +x _out/rest-coverage
  AUDIT_LOG_PATH=${AUDIT_LOG_PATH-/var/log/k8s-audit/k8s-audit.log}
  log_dest="$ARTIFACTS_PATH/cluster-audit.log"
  cli scp "$AUDIT_LOG_PATH" - > $log_dest
  _out/rest-coverage \
    --swagger-path "api/openapi-spec/swagger.json" \
    --audit-log-path $log_dest \
    --output-path "$ARTIFACTS_PATH/rest-coverage.json" \
    --ignore-resource-version
  echo "REST API coverage report generated"
fi

# Sanity check test execution by looking at results file
./automation/assert-not-all-tests-skipped.sh "${ARTIFACTS}/junit.functest.xml"
