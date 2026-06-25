#!/usr/bin/env bash

set -e

: ${MINIKUBE:="minikube --profile=kubevirtci"}
: ${BASE_PATH:=${KUBEVIRTCI_CONFIG_PATH:-$PWD}}
: ${CI_CONFIG:=${BASE_PATH}/${KUBEVIRT_PROVIDER}}
: ${KUBECONFIG:=${CI_CONFIG}/.kubeconfig}
: ${CRI_BIN:="docker"}
: ${REGISTRY_NAME:="kubevirtci-registry"}
: ${HOST_PORT:=5000}

function _kubectl() {
  ${CI_CONFIG}/.kubectl "$@"
}

function prepare_config() {
  # Copy kubectl and kubeconfig
  cp $(which kubectl) ${CI_CONFIG}/.kubectl 2>/dev/null || true
  ${MINIKUBE} kubectl -- config view --flatten > "${CI_CONFIG}/.kubeconfig"
  # Create provider config
  cat >"${CI_CONFIG}/config-provider-${KUBEVIRT_PROVIDER}.sh" <<EOF
master_ip="127.0.0.1"
kubeconfig="${CI_CONFIG}/.kubeconfig"
kubectl="${CI_CONFIG}/.kubectl"
docker_prefix=\${DOCKER_PREFIX:-localhost:${HOST_PORT}/kubevirt}
manifest_docker_prefix=\${DOCKER_PREFIX:-registry:5000/kubevirt}
EOF
}

function remove_registry_if_present() {
  if ${CRI_BIN} ps -a --format '{{.Names}}' | grep -q "^${REGISTRY_NAME}$"; then
    echo "Removing existing registry container..."
    ${CRI_BIN} stop ${REGISTRY_NAME} 2>/dev/null || true
    ${CRI_BIN} rm ${REGISTRY_NAME} 2>/dev/null || true
    echo "Registry container removed"
  fi
}

function run_registry() {
  echo "Setting up container registry..."
  remove_registry_if_present
  echo "Creating registry container on port ${HOST_PORT}..."
  ${CRI_BIN} run -d \
      --name ${REGISTRY_NAME} \
      --restart=always \
      -p ${HOST_PORT}:5000 \
      quay.io/libpod/registry:2.8.2
  echo "Registry container created successfully"
}

function configure_minikube_registry_connection() {
   # Configure containerd using the modern certs.d directory structure
  echo "Configuring containerd registry settings..."

  # Configure both localhost and registry hostnames to point to the host's registry port.
  # This ensures that images using the default 'registry:5000' prefix can be pulled.
  for host in "localhost:${HOST_PORT}" "registry:5000"; do
    ${MINIKUBE} ssh -- "sudo mkdir -p /etc/containerd/certs.d/${host} && cat <<'EOF' | sudo tee /etc/containerd/certs.d/${host}/hosts.toml
server = \"http://host.minikube.internal:${HOST_PORT}\"

[host.\"http://host.minikube.internal:${HOST_PORT}\"]
  capabilities = [\"pull\", \"resolve\", \"push\"]
  skip_verify = true
EOF
"
  done
}

function up() {
  ${MINIKUBE} --driver=${CRI_BIN} --container-runtime=containerd --cni=flannel start
  prepare_config
  run_registry
  configure_minikube_registry_connection
  echo "${KUBEVIRT_PROVIDER} cluster is ready"
}

function down() {
  ${MINIKUBE} delete
  echo "${KUBEVIRT_PROVIDER} cluster is down"
  remove_registry_if_present
}
