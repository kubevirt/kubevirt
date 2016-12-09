#!/bin/bash
set -e

source hack/config.sh

if [ $# -eq 0 ]; then
    args=$manifest_templates
else
    args=$@
fi

# Render kubernetes manifests
for arg in $args; do
    j2 --format=env $arg hack/config.sh > ${arg%%.in}
done
