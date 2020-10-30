#!/usr/bin/env bash
set -euo pipefail

echo "Waiting for KubeVirt to get ready"
oc wait -n kubevirt kv kubevirt --for condition=Available --timeout 15m
