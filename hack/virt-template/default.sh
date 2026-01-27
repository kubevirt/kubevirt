#!/bin/bash

virt_template_version=${VIRT_TEMPLATE_VERSION:-"v0.1.3"}
virt_template_yaml_sha256=${VIRT_TEMPLATE_YAML_SHA256:-"63d2fcd851043b526f0c03ff814a9d6ad34e62111410851bc72dc19db745dcf2"}

virt_template_targets="virt-template-apiserver virt-template-controller"
function is_virt_template_target() {
    local target=$1
    for vt_target in ${virt_template_targets}; do
        if [[ "${target}" == "${vt_target}" ]]; then
            return 0
        fi
    done
    return 1
}
