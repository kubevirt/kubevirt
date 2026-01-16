#!/usr/bin/env bash
set -euo pipefail

SRC="$BASE/$OLD_UUID"
DST="$BASE/$NEW_UUID"

if [ ! -d "$SRC" ]; then
    echo "TPM clone handler: INFO: no existing state for $OLD_UUID, skipping"
    exit 0
fi

if [ -d "$DST" ]; then
    echo "TPM clone handler: INFO: state already exists for $NEW_UUID, skipping"
    exit 0
fi

echo "TPM clone handler: INFO: renaming TPM state $OLD_UUID to $NEW_UUID"
mv "$SRC" "$DST"

command -v restorecon >/dev/null && restorecon -RF "$DST" || true
