#!/usr/bin/env bash
set -euo pipefail

# This script modifies the VM state files (TPM and EFI) during clone so they work on the target VM.

TPM_SRC="$TPM_BASE/$OLD_UUID"
TPM_DST="$TPM_BASE/$NEW_UUID"

EFI_SRC="$NVRAM_BASE/${OLD_VM_NAME}_VARS.fd"
EFI_DST="$NVRAM_BASE/${NEW_VM_NAME}_VARS.fd"

clone_successful=false

# Clone TPM state if it exists
if [ -z "$OLD_UUID" ] || [ -z "$NEW_UUID" ]; then
    echo "VM state clone: INFO: TPM UUIDs not set, skipping TPM clone"
elif [ ! -d "$TPM_BASE" ]; then
    echo "VM state clone: INFO: TPM base directory does not exist, skipping TPM clone"
elif [ ! -d "$TPM_SRC" ]; then
    echo "VM state clone: INFO: no TPM state found for $OLD_UUID, skipping TPM clone"
elif [ -d "$TPM_DST" ]; then
    echo "VM state clone: INFO: TPM state already exists for $NEW_UUID, skipping TPM clone"
else
    echo "VM state clone: INFO: cloning TPM state from $OLD_UUID to $NEW_UUID"
    mv "$TPM_SRC" "$TPM_DST"
    command -v restorecon >/dev/null && restorecon -RF "$TPM_DST" || true
    clone_successful=true
fi

# Clone EFI NVRAM if it exists
if [ -z "$OLD_VM_NAME" ] || [ -z "$NEW_VM_NAME" ]; then
    echo "VM state clone: INFO: EFI VM names not set, skipping EFI clone"
elif [ ! -d "$NVRAM_BASE" ]; then
    echo "VM state clone: INFO: NVRAM base directory does not exist, skipping EFI clone"
elif [ ! -f "$EFI_SRC" ]; then
    echo "VM state clone: INFO: no EFI NVRAM found for $OLD_VM_NAME, skipping EFI clone"
elif [ -f "$EFI_DST" ]; then
    echo "VM state clone: INFO: EFI NVRAM already exists for $NEW_VM_NAME, skipping EFI clone"
else
    echo "VM state clone: INFO: cloning EFI NVRAM from $OLD_VM_NAME to $NEW_VM_NAME"
    cp "$EFI_SRC" "$EFI_DST"
    command -v restorecon >/dev/null && restorecon -F "$EFI_DST" || true
    clone_successful=true
fi

if [ "$clone_successful" = true ]; then
    echo "VM state clone: SUCCESS: VM state cloned successfully"
else
    echo "VM state clone: INFO: no state files found to clone"
fi
