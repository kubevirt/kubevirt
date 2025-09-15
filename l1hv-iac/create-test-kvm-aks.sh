#!/bin/bash
# Create / update an AKS cluster for KVM e2e tests.
# Reads base settings from common.env then overrides from e2e-kvm.env (if present).
# Creates a resource group and an AKS cluster with two fixed-size node pools:
#  - agentpool: 2 nodes, VM size Standard_D4ds_v5 (system / default)
#  - workers:   1 node, VM size Standard_D8s_v5 (user)
# Both pools have autoscaling disabled. Script is idempotent.

set -euo pipefail

BLUE="\033[0;34m"; GREEN="\033[0;32m"; YELLOW="\033[1;33m"; RED="\033[0;31m"; NC="\033[0m"

log(){ echo -e "${BLUE}[INFO]${NC} $*"; }
success(){ echo -e "${GREEN}[SUCCESS]${NC} $*"; }
warn(){ echo -e "${YELLOW}[WARN]${NC} $*"; }
error(){ echo -e "${RED}[ERROR]${NC} $*"; }

script_dir="$(cd "$(dirname "$0")" && pwd)"

# Load env (common first, then specific overrides)
if [[ -f "$script_dir/common.env" ]]; then
  # shellcheck disable=SC1091
  source "$script_dir/common.env"
else
  error "common.env not found in $script_dir"; exit 1
fi
if [[ -f "$script_dir/e2e-kvm.env" ]]; then
  # shellcheck disable=SC1091
  source "$script_dir/e2e-kvm.env"
fi

# Expected variables (some may come only from env files):
# RESOURCE_GROUP  - Azure resource group name (required)
# LOCATION        - Azure region (required)
# SUBSCRIPTION    - (optional) subscription id or name
# AKS_NAME        - (optional) cluster name (default: kvm-e2e-aks)
# SSH_KEY_PATH    - path to SSH public key (optional; if absent will auto-generate)
# K8S_VERSION     - (optional) k8s version override (e.g. 1.29.4)

AKS_NAME=${AKS_NAME:-kvm-e2e-aks}
AGENTPOOL_NAME="agentpool"  # must stay "agentpool" for default system pool unless explicitly changed
AGENTPOOL_SIZE="Standard_D4ds_v5"
AGENTPOOL_COUNT=2
WORKERPOOL_NAME="workers"
WORKERPOOL_SIZE="Standard_D8s_v5"
WORKERPOOL_COUNT=1

if [[ -z "${RESOURCE_GROUP:-}" || -z "${LOCATION:-}" ]]; then
  error "RESOURCE_GROUP and LOCATION must be set (define them in common.env or export before running)."; exit 1
fi

if ! command -v az >/dev/null 2>&1; then
  error "Azure CLI (az) is required."; exit 1
fi
if ! az account show >/dev/null 2>&1; then
  error "Not logged into Azure. Run 'az login' first."; exit 1
fi

if [[ -n "${SUBSCRIPTION:-}" ]]; then
  log "Switching to subscription: $SUBSCRIPTION"
  az account set --subscription "$SUBSCRIPTION" || { error "Failed to set subscription"; exit 1; }
fi

# Ensure SSH key (public)
if [[ -n "${SSH_KEY_PATH:-}" ]]; then
  # Expand tilde
  SSH_KEY_PATH=${SSH_KEY_PATH/#~/$HOME}
  if [[ ! -f "$SSH_KEY_PATH" ]]; then
    warn "SSH key $SSH_KEY_PATH not found; generating new ed25519 key pair."
    ssh-keygen -t ed25519 -N "" -f "$SSH_KEY_PATH" || { error "Failed to generate SSH key"; exit 1; }
  fi
  if grep -q "BEGIN OPENSSH PRIVATE KEY" "$SSH_KEY_PATH" 2>/dev/null; then
    error "Provided SSH_KEY_PATH points to a private key. Provide the .pub file."; exit 1
  fi
  SSH_PUB_KEY_CONTENT=$(cat "$SSH_KEY_PATH")
else
  warn "SSH_KEY_PATH not provided; creating ephemeral key in ./aks-temporary-key"
  tmp_key="$script_dir/aks-temporary-key"
  ssh-keygen -t ed25519 -N "" -f "$tmp_key" >/dev/null 2>&1
  SSH_KEY_PATH="${tmp_key}.pub"
  SSH_PUB_KEY_CONTENT=$(cat "$SSH_KEY_PATH")
fi

log "Resource Group: $RESOURCE_GROUP"
log "Region: $LOCATION"
log "Cluster Name: $AKS_NAME"
log "System Pool: $AGENTPOOL_NAME ($AGENTPOOL_SIZE x$AGENTPOOL_COUNT)"
log "Worker Pool: $WORKERPOOL_NAME ($WORKERPOOL_SIZE x$WORKERPOOL_COUNT)"

# Create or verify RG
if az group show --name "$RESOURCE_GROUP" >/dev/null 2>&1; then
  existing_loc=$(az group show --name "$RESOURCE_GROUP" --query location -o tsv)
  if [[ "$existing_loc" != "$LOCATION" ]]; then
    warn "Existing resource group is in $existing_loc, requested $LOCATION (cannot change). Proceeding."
  else
    success "Resource group $RESOURCE_GROUP exists."
  fi
else
  log "Creating resource group $RESOURCE_GROUP in $LOCATION..."
  az group create --name "$RESOURCE_GROUP" --location "$LOCATION" --output none
  success "Resource group created."
fi

# Check if AKS cluster exists
if az aks show -g "$RESOURCE_GROUP" -n "$AKS_NAME" >/dev/null 2>&1; then
  success "AKS cluster $AKS_NAME already exists. Skipping cluster creation."
else
  log "Creating AKS cluster $AKS_NAME..."
  create_args=(
    aks create
    --resource-group "$RESOURCE_GROUP"
    --name "$AKS_NAME"
    --location "$LOCATION"
    --nodepool-name "$AGENTPOOL_NAME"
    --node-count $AGENTPOOL_COUNT
    --node-vm-size "$AGENTPOOL_SIZE"
    --generate-ssh-keys
    --enable-managed-identity
    --network-plugin azure
    --node-os-upgrade-channel NodeImage
    --nodepool-tags role=system
    --only-show-errors
    --output none
  )
  if [[ -n "${K8S_VERSION:-}" ]]; then
    create_args+=(--kubernetes-version "$K8S_VERSION")
  fi
  # If we have a custom public key, override the generated one
  if [[ -n "${SSH_PUB_KEY_CONTENT:-}" ]]; then
    create_args+=(--ssh-key-value "$SSH_PUB_KEY_CONTENT")
  fi
  az "${create_args[@]}"
  success "AKS cluster created."
fi

# Ensure worker pool exists
if az aks nodepool show --cluster-name "$AKS_NAME" --resource-group "$RESOURCE_GROUP" --name "$WORKERPOOL_NAME" >/dev/null 2>&1; then
  success "Node pool $WORKERPOOL_NAME already exists."
else
  log "Creating worker node pool $WORKERPOOL_NAME..."
  az aks nodepool add \
    --cluster-name "$AKS_NAME" \
    --resource-group "$RESOURCE_GROUP" \
    --name "$WORKERPOOL_NAME" \
    --node-count $WORKERPOOL_COUNT \
    --node-vm-size "$WORKERPOOL_SIZE" \
    --mode User \
    --nodepool-tags role=worker \
    --only-show-errors \
    --output none
  success "Worker node pool created."
fi

# Get credentials (admin by default)
log "Fetching kubeconfig..."
az aks get-credentials -g "$RESOURCE_GROUP" -n "$AKS_NAME" --overwrite-existing --only-show-errors --output none
success "Kubeconfig merged (current context set to $AKS_NAME)."

# Show cluster nodes summary
log "Cluster nodes:"; kubectl get nodes -o wide || warn "kubectl query failed"

cat <<SUMMARY
============================================================
AKS Provisioning Complete
------------------------------------------------------------
Resource Group : $RESOURCE_GROUP
Cluster        : $AKS_NAME
Region         : $LOCATION
Subscription   : ${SUBSCRIPTION:-<active>}
System Pool    : $AGENTPOOL_NAME ($AGENTPOOL_SIZE x$AGENTPOOL_COUNT)
Worker Pool    : $WORKERPOOL_NAME ($WORKERPOOL_SIZE x$WORKERPOOL_COUNT)
Kubeconfig     : merged into ~/.kube/config
To switch context: kubectl config use-context $AKS_NAME
============================================================
SUMMARY
