#!/usr/bin/env bash
set -exuo pipefail

gcsweb_base_url="https://gcsweb.ci.kubevirt.io/gcs/kubevirt-prow"
testing_resources=(disks-images-provider.yaml local-block-storage.yaml rbac-for-testing.yaml uploadproxy-nodeport.yaml)

function test_kubevirt_release() {
    release="$(get_release_tag_for_xy "$1")"
    export DOCKER_TAG="$release"
    deploy_release "$release"
    run_tests
}

function get_release_tag_for_xy() {
    release_xy="$1"

    curl --fail -s https://api.github.com/repos/kubevirt/kubevirt/releases |
        jq -r '(.[].tag_name | select( test("-(rc|alpha|beta)") | not ) )' |
        sort -rV | grep "v$release_xy" | head -1
}

function deploy_latest_cdi_release() {
    cdi_release_tag=$(curl -L -H'Accept: application/json' 'https://github.com/kubevirt/containerized-data-importer/releases/latest' | jq -r '.tag_name')
    oc create -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${cdi_release_tag}/cdi-operator.yaml"
    oc create -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${cdi_release_tag}/cdi-cr.yaml"

    # enable featuregate
    oc patch cdi cdi --type merge -p '{"spec": {"config": {"featureGates": [ "HonorWaitForFirstConsumer" ]}}}'
}

function undeploy_latest_cdi_release() {
    cdi_release_tag=$(curl -L -H'Accept: application/json' 'https://github.com/kubevirt/containerized-data-importer/releases/latest' | jq -r '.tag_name')
    oc delete --ignore-not-found=true -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${cdi_release_tag}/cdi-cr.yaml" || true
    oc delete --ignore-not-found=true -f "https://github.com/kubevirt/containerized-data-importer/releases/download/${cdi_release_tag}/cdi-operator.yaml"
}

function deploy_release() {
    local release="$1"

    tagged_release_url="https://github.com/kubevirt/kubevirt/releases/download/${release}"

    oc create -f ./hack/ci/resources/disk-rhel.yaml

    curl -Lo "$BIN_DIR/tests.test" "${tagged_release_url}/tests.test"
    chmod +x "$BIN_DIR/tests.test"

    curl -L "${tagged_release_url}/kubevirt-operator.yaml" | oc create -f -
    curl -L "${tagged_release_url}/kubevirt-cr.yaml" | oc create -f -

    deploy_latest_cdi_release

    testing_infra_url="$gcsweb_base_url/devel/release/kubevirt/kubevirt/${release}/manifests/testing"
    for testing_resource in "${testing_resources[@]}"; do
        curl -L "${testing_infra_url}/${testing_resource}" | oc create -f -
    done

    until wait_on_cdi_ready && wait_on_kubevirt_ready; do sleep 5; done
}

function undeploy_release() {
    local release="$1"

    tagged_release_url="https://github.com/kubevirt/kubevirt/releases/download/${release}"

    oc delete --ignore-not-found=true -f ./hack/ci/resources/disk-rhel.yaml

    testing_infra_url="$gcsweb_base_url/devel/release/kubevirt/kubevirt/${release}/manifests/testing"
    for testing_resource in "${testing_resources[@]}"; do
        curl -L "${testing_infra_url}/${testing_resource}" | oc delete --ignore-not-found=true -f -
    done

    undeploy_latest_cdi_release

    curl -L "${tagged_release_url}/kubevirt-cr.yaml" | oc delete --ignore-not-found=true -f - || true
    curl -L "${tagged_release_url}/kubevirt-operator.yaml" | oc delete --ignore-not-found=true -f -

    oc delete --ignore-not-found=true -f ./hack/ci/resources/disk-rhel.yaml
}

function test_kubevirt_nightly() {
    local release_url
    release_date=$(get_latest_release_date_for_kubevirt_nightly)
    release_url="$(get_release_url_for_kubevirt_nightly "$release_date")"

    export DOCKER_PREFIX='quay.io/kubevirt'
    DOCKER_TAG="$(get_release_tag_for_kubevirt_nightly "$release_url" "$release_date")"
    export DOCKER_TAG

    deploy_kubevirt_nightly_test_setup
    run_tests
}

function deploy_kubevirt_nightly_test_setup() {
    local release_url
    release_date=$(get_latest_release_date_for_kubevirt_nightly)
    release_url="$(get_release_url_for_kubevirt_nightly "$release_date")"

    oc create -f ./hack/ci/resources/disk-rhel.yaml

    curl -Lo "$BIN_DIR/tests.test" "${release_url}/testing/tests.test"
    chmod +x "$BIN_DIR/tests.test"

    oc create -f "${release_url}/kubevirt-operator.yaml"
    oc create -f "${release_url}/kubevirt-cr.yaml"

    deploy_latest_cdi_release

    for testing_resource in "${testing_resources[@]}"; do
        oc create -f "${release_url}/testing/${testing_resource}"
    done

    until wait_on_cdi_ready && wait_on_kubevirt_ready; do sleep 5; done
}

function wait_on_cdi_ready() {
    oc wait -n cdi cdi cdi --for=condition=Available --timeout=180s
}

function wait_on_kubevirt_ready() {
    oc wait -n kubevirt kv kubevirt --for condition=Available --timeout 15m
}

function undeploy_kubevirt_nightly_test_setup() {
    local release_url
    release_date=$(get_latest_release_date_for_kubevirt_nightly)
    release_url="$(get_release_url_for_kubevirt_nightly "$release_date")"

    oc delete --ignore-not-found=true -f ./hack/ci/resources/disk-rhel.yaml

    for testing_resource in "${testing_resources[@]}"; do
        oc delete --ignore-not-found=true -f "${release_url}/testing/${testing_resource}"
    done

    undeploy_latest_cdi_release

    oc delete --ignore-not-found=true -f "${release_url}/kubevirt-cr.yaml"
    oc delete --ignore-not-found=true -f "${release_url}/kubevirt-operator.yaml"

    oc delete --ignore-not-found=true -f ./hack/ci/resources/disk-rhel.yaml
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

    set +u
    additional_test_args=""
    if [ -n "$KUBEVIRT_E2E_SKIP" ] || [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
        if [ -n "$KUBEVIRT_E2E_SKIP" ]; then
            additional_test_args="${additional_test_args} -ginkgo.skip=${KUBEVIRT_E2E_SKIP}"
        fi
        if [ -n "$KUBEVIRT_E2E_FOCUS" ]; then
            additional_test_args="${additional_test_args} -ginkgo.focus=${KUBEVIRT_E2E_FOCUS}"
        fi
    elif [ -n "$KUBEVIRT_TESTS_FOCUS" ]; then
        additional_test_args="$KUBEVIRT_TESTS_FOCUS"
    fi
    kubevirt_testing_configuration=${KUBEVIRT_TESTING_CONFIGURATION:-./hack/ci/resources/kubevirt-testing-configuration.json}
    set -u

    tests.test -v=5 \
        -config=${kubevirt_testing_configuration} \
        -kubeconfig=${KUBECONFIG} \
        -container-tag=${DOCKER_TAG} \
        -container-tag-alt= \
        -container-prefix=${DOCKER_PREFIX} \
        -image-prefix-alt=-kv \
        -oc-path=${OC_PATH} \
        -kubectl-path=${KUBECTL_PATH} \
        -gocli-path=$(pwd)/cluster-up/cli.sh \
        -test.timeout 420m \
        -ginkgo.noColor \
        -ginkgo.succinct \
        -ginkgo.slowSpecThreshold=60 \
        ${additional_test_args} \
        -junit-output=${ARTIFACT_DIR}/junit.functest.xml \
        -installed-namespace=kubevirt \
        -previous-release-tag= \
        -previous-release-registry=quay.io/kubevirt \
        -deploy-testing-infra=false \
        -apply-default-e2e-configuration=true
}

export PATH="$BIN_DIR:$PATH"
eval "$@"
