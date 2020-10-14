#!/usr/bin/env bash

INFRA=$(cat <<EOF
  infra:
    nodePlacement:
      nodeSelector:
        node.kubernetes.io/instance-type: "infra"
EOF
)
INFRA=$(echo "${INFRA}" | sed '$!s|$|\\|g')

WORKLOADS=$(cat <<EOF
  workloads:
    nodePlacement:
      nodeSelector:
        node.kubernetes.io/instance-type: "workloads"
EOF
)
WORKLOADS=$(echo "${WORKLOADS}" | sed '$!s|$|\\|g')


sed -i -r "s|^  infra:.*$|${INFRA}|" _out/hco.cr.yaml
sed -i -r "s|^  workloads:.*$|${WORKLOADS}|" _out/hco.cr.yaml
