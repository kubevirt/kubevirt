#!/usr/bin/env bash

set -e

if [[ "$KUBEVIRT_USE_FAKE_VFIO" != "true" ]]; then
  exit 1
fi

SCRIPT_PATH="$(dirname "$(realpath "$0")")"
exec "${SCRIPT_PATH}/../vfio-gpu/config_vfio_cluster.sh" "$@"
