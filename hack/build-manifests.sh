#!/bin/bash
set -e

# Temporary hack to export everything into env
eval `cat hack/config.sh | sed -e 's/^/export /'`

if [ $# -eq 0 ]; then
    args=$manifest_templates
else
    args=$@
fi

# Delete all generated manifests in case an input file was deleted or renamed
rm -f "manifests/*.yaml"

# Render kubernetes manifests
for arg in $args; do
    sed -e "s/{{ master_ip }}/$master_ip/g" \
        -e "s/{{ docker_tag }}/$docker_tag/g" \
        -e "s/{{ docker_prefix }}/$docker_prefix/g" \
        $arg > ${arg%%.in}
done
