#!/usr/bin/env bash
set -euo pipefail

gcsweb_base_url="https://gcsweb.ci.kubevirt.io/gcs/kubevirt-prow"

function test_kubevirt_release() {
    release="$(get_release_tag_for_xy "$1")"
    export DOCKER_TAG="$release"
    deploy_release "$release"

    wait_on_kubevirt_ready
    run_tests
}

function get_release_tag_for_xy() {
    release_xy="$1"

    curl --fail -s https://api.github.com/repos/kubevirt/kubevirt/releases |
        jq -r '(.[].tag_name | select( test("-(rc|alpha|beta)") | not ) )' |
        sort -rV | grep "v$release_xy" | head -1
}

function deploy_release() {
    local release="$1"

    tagged_release_url="https://github.com/kubevirt/kubevirt/releases/download/${release}"

    curl -Lo "$BIN_DIR/tests.test" "${tagged_release_url}/tests.test"
    chmod +x "$BIN_DIR/tests.test"

    curl -L "${tagged_release_url}/kubevirt-operator.yaml" | oc create -f -
    curl -L "${tagged_release_url}/kubevirt-cr.yaml" | oc create -f -

    testing_infra_url="$gcsweb_base_url/devel/release/kubevirt/kubevirt/${release}/manifests/testing/"
    for testinfra_file in $(curl -L "${testing_infra_url}" | grep -oE 'https://[^"]*\.yaml'); do
        curl -L ${testinfra_file} | oc create -f -
    done
}

function test_kubevirt_nightly() {
    export DOCKER_PREFIX='quay.io/kubevirt'
    local release_url
    release_date=$(get_latest_release_date_for_kubevirt_nightly)
    release_url="$(get_release_url_for_kubevirt_nightly "$release_date")"
    DOCKER_TAG="$(get_release_tag_for_kubevirt_nightly "$release_url" "$release_date")"
    export DOCKER_TAG

    deploy_kubevirt_nightly "$release_url"
    wait_on_kubevirt_ready
    run_tests
}

function deploy_kubevirt_nightly() {
    release_url="$1"

    echo "Downloading kubevirt tests binary from nightly build $release_url"
    curl -Lo "$BIN_DIR/tests.test" "${release_url}/testing/tests.test"
    chmod +x "$BIN_DIR/tests.test"

    echo "Deploying kubevirt from nightly build $release_url"
    oc create -f "${release_url}/kubevirt-operator.yaml"
    oc create -f "${release_url}/kubevirt-cr.yaml"

    echo "Deploying test infrastructure from $release_url"
    for testinfra_file in $(curl -L "${release_url}/testing/" | grep -oE 'https://[^"]*\.yaml'); do
        oc create -f "${testinfra_file}"
    done
}

function get_release_tag_for_kubevirt_nightly() {
    release_url="$1"
    release_date="$2"
    commit=$(curl -L "${release_url}/commit")
    echo "${release_date}_$(echo "${commit}" | cut -c 1-9)"
}

function get_release_url_for_kubevirt_nightly() {
    release_base_url="$gcsweb_base_url/devel/nightly/release/kubevirt/kubevirt"
    release_date="$1"
    echo "${release_base_url}/${release_date}"
}

function get_latest_release_date_for_kubevirt_nightly() {
    release_base_url="$gcsweb_base_url/devel/nightly/release/kubevirt/kubevirt"
    release_date=$(curl -L "${release_base_url}/latest")
    echo "${release_date}"
}

function wait_on_kubevirt_ready() {
    oc wait -n kubevirt kv kubevirt --for condition=Available --timeout 15m
}

function get_path_or_empty_string_for_cmd() {
    cmd="$1"
    set +e
    which "$cmd"
    set -e
}

function run_tests() {
    mkdir -p "$ARTIFACT_DIR"
    # required to be set for test binary
    export ARTIFACTS=${ARTIFACT_DIR}

    OC_PATH=$(get_path_or_empty_string_for_cmd oc)
    KUBECTL_PATH=$(get_path_or_empty_string_for_cmd kubectl)

    tests.test -v=5 -kubeconfig=${KUBECONFIG} -container-tag=${DOCKER_TAG} -container-tag-alt= -container-prefix=${DOCKER_PREFIX} -image-prefix-alt=-kv -oc-path=${OC_PATH} -kubectl-path=${KUBECTL_PATH} -gocli-path=$(pwd)/cluster-up/cli.sh -test.timeout 420m -ginkgo.noColor -ginkgo.succinct -ginkgo.slowSpecThreshold=60 ${KUBEVIRT_TESTS_FOCUS} -junit-output=${ARTIFACT_DIR}/junit.functest.xml -installed-namespace=kubevirt -previous-release-tag= -previous-release-registry=quay.io/kubevirt -deploy-testing-infra=false
}

export PATH="$BIN_DIR:$PATH"
eval "$@"
