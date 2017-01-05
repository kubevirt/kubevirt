#!/bin/bash
set -e

# Temporary hack to export everything into env
eval `cat hack/config.sh | sed -e 's/^/export /'`

if [ $# -eq 0 ]; then
    args=$manifest_templates
else
    args=$@
fi

# Render kubernetes manifests
for arg in $args; do
    env | j2 --format=env $arg > ${arg%%.in}
done
