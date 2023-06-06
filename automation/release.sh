#!/usr/bin/env bash

set -exuo pipefail

function cleanup_gh_install() {
    [ -n "${gh_cli_dir}" ] && [ -d "${gh_cli_dir}" ] && rm -rf "${gh_cli_dir:?}/"
}

function ensure_gh_cli_installed() {
    if command -V gh; then
        return
    fi

    trap 'cleanup_gh_install' EXIT SIGINT SIGTERM

    # install gh cli for uploading release artifacts, with prompt disabled to enforce non-interactive mode
    gh_cli_dir=$(mktemp -d)
    (
        cd  "$gh_cli_dir/"
        curl -sSL "https://github.com/cli/cli/releases/download/v${GH_CLI_VERSION}/gh_${GH_CLI_VERSION}_linux_amd64.tar.gz" -o "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
        tar xvf "gh_${GH_CLI_VERSION}_linux_amd64.tar.gz"
    )
    export PATH="$gh_cli_dir/gh_${GH_CLI_VERSION}_linux_amd64/bin:$PATH"
    if ! command -V gh; then
        echo "gh cli not installed successfully"
        exit 1
    fi
    gh config set prompt disabled
}

export BUILD_ARCH=aarch64,x86_64
export KUBEVIRT_RELEASE=true

function build_release_artifacts() {
    make
    make build-verify
    make apidocs
    make client-python
    make manifests
    make olm-verify
    make prom-rules-verify

    BUILD_ARCH="${BUILD_ARCH}" QUAY_REPOSITORY="kubevirt" PACKAGE_NAME="kubevirt-operatorhub" make bazel-push-images

    make build-functests
}

function update_github_release() {
    # note: for testing purposes we set the target repository, gh cli seems to always automatically choose the
    # upstream repository automatically, even when you are in a fork

    set +e
    if ! gh release view --repo "$GITHUB_REPOSITORY" "$DOCKER_TAG" ; then
        set -e
        git show "$DOCKER_TAG" --format=format:%B > /tmp/tag_notes
        gh release create --repo "$GITHUB_REPOSITORY" "$DOCKER_TAG" --prerelease --title="$DOCKER_TAG" --notes-file /tmp/tag_notes
    else
        set -e
    fi

    gh release upload --repo "$GITHUB_REPOSITORY" --clobber "$DOCKER_TAG" _out/cmd/virtctl/virtctl-v* \
        _out/manifests/release/demo-content.yaml \
        _out/manifests/release/kubevirt-operator.yaml \
        _out/manifests/release/kubevirt-cr.yaml \
        _out/manifests/release/olm/kubevirt-operatorsource.yaml \
        "_out/manifests/release/olm/bundle/kubevirtoperator.$DOCKER_TAG.clusterserviceversion.yaml" \
        _out/tests/tests.test \
        _out/manifests/release/conformance.yaml \
        _out/manifests/testing/*
        _out/cmd/dump/dump*
}

function upload_testing_manifests() {
    # replaces periodic-kubevirt-update-release-x.y-testing-manifests periodics
    gsutil -m rm -r "gs://kubevirt-prow/devel/release/kubevirt/kubevirt/$DOCKER_TAG" || true
    gsutil cp -r "_out/manifests/testing" "gs://kubevirt-prow/devel/release/kubevirt/kubevirt/$DOCKER_TAG/manifests/"
}

function generate_stable_version_file() {
    # will be available under http://storage.googleapis.com/kubevirt-prow/devel/release/kubevirt/kubevirt/stable.txt
    (
        gh release list --repo "$GITHUB_REPOSITORY" --limit 1000 |
        awk '{ print $1 }' |
        grep -v -E '\-(rc|alpha|beta)' |
        sort -rV |
        head -1
    ) > _out/stable.txt
    # this place is deprecated. Combining "devel" and stable is not optimal
    gsutil cp "_out/stable.txt" "gs://kubevirt-prow/devel/release/kubevirt/kubevirt/"
    gsutil cp "_out/stable.txt" "gs://kubevirt-prow/release/kubevirt/kubevirt/"
}

function main() {
    DOCKER_TAG="$(git tag --points-at HEAD | head -1)"
    if [ -z "$DOCKER_TAG" ]; then
        echo "commit $(git show -s --format=%h) doesn't have a tag, exiting..."
        exit 0
    fi

    export DOCKER_TAG

    GIT_ASKPASS="$(pwd)/automation/git-askpass.sh"
    [ -f "$GIT_ASKPASS" ] || exit 1
    export GIT_ASKPASS

    ensure_gh_cli_installed

    gh auth login --with-token <"$GITHUB_TOKEN_PATH"

    build_release_artifacts
    update_github_release
    upload_testing_manifests
    generate_stable_version_file

    bash hack/gen-swagger-doc/deploy.sh
    hack/publish-staging.sh
}

main "$@"
