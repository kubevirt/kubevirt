# source: https://github.com/rpardini/docker-registry-proxy#kind-cluster
#
# This script execute docker-registry-proxy cluster nodes
# setup script on each cluster node.
# Basically what the setup script does is loading the proxy certificate
# and set HTTP_PROXY and NO_PROXY env vars to enable direct communication
# between cluster components (e.g: pods, nodes and services).
#
# Args:
# KIND_BIN - KinD binary path.
# PROXY_HOSTNAME - docker-registry-proxy endpoint hostname.
# CLUSTER_NAME - KinD cluster name.
#
# Usage example:
# KIND_BIN="./kind" CLUSTER_NAME="test" PROXY_HOSTNAME="proxy.ci.com" \
#   ./configure-registry-proxy.sh
#

#! /bin/bash

set -ex

CRI_BIN=${CRI_BIN:-docker}

KIND_BIN="${KIND_BIN:-./kind}"
PROXY_HOSTNAME="${PROXY_HOSTNAME:-docker-registry-proxy}"
CLUSTER_NAME="${CLUSTER_NAME:-sriov}"

SETUP_URL="http://${PROXY_HOSTNAME}:3128/setup/systemd"
pids=""
for node in $($KIND_BIN get nodes --name "$CLUSTER_NAME"); do
   $CRI_BIN exec "$node" sh -c "\
      curl $SETUP_URL | \
      sed s/docker\.service/containerd\.service/g | \
      sed '/Environment/ s/$/ \"NO_PROXY=127.0.0.0\/8,10.0.0.0\/8,172.16.0.0\/12,192.168.0.0\/16\"/' | \
      bash" &
   pids="$pids $!"
done
wait $pids

