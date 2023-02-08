#!/bin/bash -e

readonly PROM_IMAGE="quay.io/prometheus/prometheus:v2.42.0"

function cleanup() {
    local cleanup_files=("${@:?}")
    for file in "${cleanup_files[@]}"; do
        rm -f "$file"
    done
}

function lint() {
    local target_file="${1:?}"
    podman run --rm --entrypoint=/bin/promtool \
        -v "$target_file":/tmp/rules.verify:ro,Z "$PROM_IMAGE" \
        check rules /tmp/rules.verify
}

function unit_test() {
    local target_file="${1:?}"
    local tests_file="${2:?}"
    podman run --rm --entrypoint=/bin/promtool \
        -v "$tests_file":/tmp/rules.test:ro,Z \
        -v "$target_file":/tmp/rules.verify:ro,Z \
        "$PROM_IMAGE" \
        test rules /tmp/rules.test
}

function main() {
    local prom_spec_dumper="${1:?}"
    local tests_file="${2:?}"
    local target_file
    target_file="$(mktemp --tmpdir -u tmp.prom_rules.XXXXX)"
    trap "cleanup $target_file" RETURN EXIT INT
    "$prom_spec_dumper" "$target_file"
    echo "INFO: Rules file content:"
    cat "$target_file"
    echo
    lint "$target_file"
    unit_test "$target_file" "$tests_file"
}

main "$@"
