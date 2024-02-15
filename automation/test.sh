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
readonly TEMPLATES_SERVER="gs://kubevirt-vm-images"
readonly BAZEL_CACHE="${BAZEL_CACHE:-http://bazel-cache.kubevirt-prow.svc.cluster.local:8080/kubevirt.io/kubevirt}"


# Skip if it's docs changes only
# Only if we are in CI, and this is a non-batch change
if [[ ${CI} == "true" && -n "$PULL_BASE_SHA" && -n "$PULL_PULL_SHA" ]]; then
    SKIP_PATTERN="^(docs/)|(OWNERS|OWNERS_ALIASES|.*\.(md|txt))$"
    CI_GIT_ALL_CHANGES=$(git diff --name-only ${PULL_BASE_SHA}...${PULL_PULL_SHA})
    CI_GIT_NO_DOCS_CHANGES=$(cat <<<$CI_GIT_ALL_CHANGES | grep -vE "$SKIP_PATTERN" || :)
    if [[ -z "$CI_GIT_NO_DOCS_CHANGES" ]]; then
        echo "Aborting as there were only none-code related changes detected."
        exit 0
    fi
 fi

if [[ ${CI} == "true" ]]; then
  if [[ ! $TARGET =~ .*kind.* ]] && [[ ! $TARGET =~ .*k3d.* ]]; then
    _delay="$(( ( RANDOM % 180 )))"
    echo "INFO: Sleeping for ${_delay}s to randomize job startup slighty"
    sleep ${_delay}
  fi
fi

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
  export KUBEVIRT_WITH_MULTUS_V3="${KUBEVIRT_WITH_MULTUS_V3:-true}"
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_DEPLOY_NET_BINDING_CNI=true
  export KUBEVIRT_DEPLOY_CDI=false
  # FIXME: https://github.com/kubevirt/kubevirt/issues/9158
  if [[ $TARGET =~ no-istio ]]; then
    export KUBEVIRT_DEPLOY_ISTIO=false
  else
    export KUBEVIRT_DEPLOY_ISTIO=true
  fi
  export KUBEVIRT_PROVIDER=${TARGET/-sig-network*/}
elif [[ $TARGET =~ sig-storage ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-storage/}
  export KUBEVIRT_STORAGE="rook-ceph-default"
  export KUBEVIRT_DEPLOY_NFS_CSI=true
elif [[ $TARGET =~ sig-compute-realtime ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-compute-realtime/}
  export KUBEVIRT_HUGEPAGES_2M=512
  export KUBEVIRT_REALTIME_SCHEDULER=true
elif [[ $TARGET =~ sig-compute-migrations ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-compute-migrations/}
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_NUM_SECONDARY_NICS=1
  export KUBEVIRT_DEPLOY_NFS_CSI=true
elif [[ $TARGET =~ sig-compute ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-compute/}
elif [[ $TARGET =~ sig-operator ]]; then
  export KUBEVIRT_PROVIDER=${TARGET/-sig-operator*/}
  export KUBEVIRT_WITH_CNAO=true
  export KUBEVIRT_NUM_SECONDARY_NICS=1
elif [[ $TARGET =~ sig-monitoring ]]; then
    export KUBEVIRT_PROVIDER=${TARGET/-sig-monitoring/}
    export KUBEVIRT_DEPLOY_PROMETHEUS=true
else
  export KUBEVIRT_PROVIDER=${TARGET}
fi

# Single-node single-replica test lanes need nfs csi to run sig-storage tests
if [[ $KUBEVIRT_NUM_NODES = "1" && $KUBEVIRT_INFRA_REPLICAS = "1" ]]; then
  export KUBEVIRT_DEPLOY_NFS_CSI=true
fi

if [ ! -d "cluster-up/cluster/$KUBEVIRT_PROVIDER" ]; then
  echo "The cluster provider $KUBEVIRT_PROVIDER does not exist"
  exit 1
fi

if [[ $TARGET =~ sriov.* ]]; then
  if [[ $TARGET =~ kind.* ]]; then
    export KUBEVIRT_NUM_NODES=3
  fi
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
      remote_sha1="$(gsutil cat ${remote_sha1_url})"
      if [[ "$remote_sha1" != "" ]]; then
        break
      fi
    done

    if [[ "$(cat "$local_sha1_file")" != "$remote_sha1" ]]; then
        echo "${download_to} is not up to date, corrupted or doesn't exist."
        echo "Downloading file from: ${remote_sha1_url}"
        gsutil cp $download_from $download_to
        sha1sum "$download_to" | cut -d " " -f1 > "$local_sha1_file"
        [[ "$(cat "$local_sha1_file")" == "$remote_sha1" ]] || {
            echo "${download_to} is corrupted"
            return 1
        }
    else
        echo "${download_to} is up to date"
    fi
)


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
        if curl --unix-socket "${XDG_RUNTIME_DIR}/podman/podman.sock" http://d/v3.0.0/libpod/info >/dev/null 2>&1; then
            echo podman
        elif docker ps >/dev/null 2>&1; then
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

if [ "$CI" != "true" ]; then
  make cluster-down
fi

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
if [[ $TARGET =~ .*kind.* ]] || [[ $TARGET =~ .*k3d.* ]]; then
  export KUBEVIRT_E2E_PARALLEL=false
fi

ginko_params="--no-color --seed=42"

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

# add_to_label_filter appends the given label and separator to
# $label_filter which is passed to Ginkgo --filter-label flag.
# How to use:
# - Run tests with label
#     add_to_label_filter '(mylabel)' ','
# - Dont run tests with label:
#     add_to_label_filter '(!mylabel)' '&&'
add_to_label_filter() {
  local label=$1
  local separator=$2
  if [[ -z $label_filter ]]; then
    label_filter="${1}"
  else
    label_filter="${label_filter}${separator}${1}"
  fi
}

label_filter="${KUBEVIRT_LABEL_FILTER}"
# Set label_filter only if KUBEVIRT_E2E_FOCUS, KUBEVIRT_E2E_SKIP and KUBEVIRT_LABEL_FILTER are not set.
if [[ -z ${KUBEVIRT_E2E_FOCUS} && -z ${KUBEVIRT_E2E_SKIP} && -z ${label_filter} ]]; then
  echo "WARN: Ongoing deprecation of the keyword matchers and updating them with ginkgo Label decorators"
  if [[ $TARGET =~ windows_sysprep.* ]]; then
    label_filter='(Sysprep)'
  elif [[ $TARGET =~ windows.* ]]; then
    # Run only Windows tests
    label_filter='(Windows)'
  elif [[ $TARGET =~ (cnao|multus) ]]; then
    label_filter='(Multus,Networking,VMIlifecycle,Expose,Macvtap)'
  elif [[ $TARGET =~ sig-network ]]; then
    label_filter='(sig-network,netCustomBindingPlugins)'
    # FIXME: https://github.com/kubevirt/kubevirt/issues/9158
    if [[ $TARGET =~ no-istio ]]; then
      add_to_label_filter "(!Istio)" "&&"
    fi
    if [[ $KUBEVIRT_WITH_MULTUS_V3 == "true" ]]; then
      add_to_label_filter "(!in-place-hotplug-NICs)" "&&"
    else
      add_to_label_filter "(!migration-based-hotplug-NICs)" "&&"
    fi
  elif [[ $TARGET =~ sig-storage ]]; then
    label_filter='(sig-storage)'
  elif [[ $TARGET =~ vgpu.* ]]; then
    label_filter='(VGPU)'
  elif [[ $TARGET =~ sig-compute-realtime ]]; then
    label_filter='(sig-compute-realtime)'
  elif [[ $TARGET =~ sig-compute-migrations ]]; then
    label_filter='(sig-compute-migrations && !(GPU,VGPU))'
  elif [[ $TARGET =~ sig-compute ]]; then
    label_filter='(sig-compute && !(GPU,VGPU,sig-compute-migrations))'
  elif [[ $TARGET =~ sig-monitoring ]]; then
    label_filter='(sig-monitoring)'
  elif [[ $TARGET =~ sig-operator ]]; then
    if [[ $TARGET =~ sig-operator-upgrade ]]; then
      label_filter='(Upgrade)'
    elif [[ $TARGET =~ sig-operator-configuration ]]; then
      label_filter='(sig-operator && !(Upgrade))'
    else
      label_filter='(sig-operator)'
    fi
  elif [[ $TARGET =~ sriov.* ]]; then
    label_filter='(SRIOV)'
  elif [[ $TARGET =~ gpu.* ]]; then
    label_filter='(GPU)'
  elif [[ $TARGET =~ (okd|ocp).* ]]; then
    label_filter='(!(SRIOV,GPU,VGPU))'
  else
    label_filter='(!(Multus,SRIOV,Macvtap,GPU,VGPU,netCustomBindingPlugins))'
  fi
fi

# We do not want to run tests which exclude native SSH functionality
add_to_label_filter '(!exclude-native-ssh)' '&&'

# Single-node single-replica test lanes obviously can't run live migrations,
# but also currently lack the requirements for SRIOV, GPU, Macvtap and MDEVs.
if [[ $KUBEVIRT_NUM_NODES = "1" && $KUBEVIRT_INFRA_REPLICAS = "1" ]]; then
  add_to_label_filter '(!(SRIOV,GPU,Macvtap,VGPU,sig-compute-migrations,requires-two-schedulable-nodes))' '&&'
fi

# If KUBEVIRT_QUARANTINE is not set, do not run quarantined tests. When it is
# set the whole suite (quarantined and stable) will be run.
if [ -z "$KUBEVIRT_QUARANTINE" ]; then
  add_to_label_filter '(!QUARANTINE)' '&&'
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
FUNC_TEST_ARGS=$ginko_params FUNC_TEST_LABEL_FILTER='--label-filter='${label_filter} make functest

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
