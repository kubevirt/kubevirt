#!/bin/bash

function main() {
    local registry="${1:?}"
    local username="${2:?}"
    local password="${3:?}"

    local dockerconfig_patch original_pull_secret patched_pull_secret
    local encoded_pull_secret auth

    auth="$(create_auth_entry "$username" "$password")"
    dockerconfig_patch="$(create_dockerconfig "$auth" "$registry")"
    original_pull_secret="$(get_original_pull_secret)"
    patched_pull_secret="$(
        patch_pull_secret "$original_pull_secret" "$dockerconfig_patch"
    )"
    pull_secret_b64="$(b64_encode "$patched_pull_secret")"
    apply_pull_secret_patch "$pull_secret_b64"
}

function create_auth_entry() {
    local username="${1:?}"
    local password="${2:?}"

    echo -n "${username}:${password}" | base64 -w 0
}

function create_dockerconfig() {
    local auth="${1:?}"
    local registry="${2:?}"

    echo -n "{\"auths\":{\"${registry}\":{\"auth\":\"${auth}\",\"email\":\"\"}}}"
}

function apply_pull_secret_patch() {
    local encoded_pull_secret="${1:?}"

    oc patch secret/pull-secret -n openshift-config --type merge --patch \
	"{\"data\":{\".dockerconfigjson\":\"${encoded_pull_secret}\"}}"
}

function b64_encode() {
    local data="${1:?}"

    echo -n "$data" | base64 -w 0
}

function patch_pull_secret() {
    local orig="${1:?}"
    local patch="${2:?}"

    local data
    data="$(jq -c ".auths" <<<"$patch")"
    jq -c ".auths += $data" <<<"$orig"
}

function generate_dockerconfig_patch() {
    local org="${1:?}"
    local username="${2:?}"
    local password="${3:?}"

    cat << EOF
{
    "${org}": {
        "username": "${username}",
        "password": "${password}"
    }
}
EOF
}

function get_original_pull_secret() {
    oc get secret/pull-secret -n openshift-config -o json | \
        jq '.data.".dockerconfigjson"' | tr -d '"' | base64 -d
}

main "$@"
