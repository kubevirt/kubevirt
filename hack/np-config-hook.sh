#!/usr/bin/env bash

set -x

INFRA=$(cat <<EOF
  infra:
    nodePlacement:
      nodeSelector:
        node.kubernetes.io/hco-test-node-type: "infra"
EOF
)
INFRA=$(echo "${INFRA}" | sed '$!s|$|\\|g')

WORKLOADS=$(cat <<EOF
  workloads:
    nodePlacement:
      nodeSelector:
        node.kubernetes.io/hco-test-node-type: "workloads"
EOF
)
WORKLOADS=$(echo "${WORKLOADS}" | sed '$!s|$|\\|g')

sed -i -r "s|^  infra:.*$|${INFRA}|; s|^  workloads:.*$|${WORKLOADS}|" _out/hco.cr.yaml

echo HCO CR after modification:
cat _out/hco.cr.yaml
