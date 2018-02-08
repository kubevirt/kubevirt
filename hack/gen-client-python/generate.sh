#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

source $(dirname "$0")/../common.sh

SWAGGER_CODEGEN_CLI_SRC=http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.2.3/swagger-codegen-cli-2.2.3.jar
SWAGGER_CODEGEN_CLI="/tmp/swagger-codegen-cli.jar"
KUBEVIRT_SPEC="${KUBEVIRT_DIR}/api/openapi-spec/swagger.json"
CODEGEN_CONFIG_SRC="${KUBEVIRT_DIR}/hack/gen-client-python/swagger-codegen-config.json.in"
CODEGEN_CONFIG="${PYTHON_CLIENT_OUT_DIR}/swagger-codegen-config.json"

if [ -n "${TRAVIS_TAG:-}" ]; then
    CLIENT_PYTHON_VERSION="$TRAVIS_TAG"
else
    CLIENT_PYTHON_VERSION="$(git describe || echo 'none')"
fi

mkdir -p "${PYTHON_CLIENT_OUT_DIR}"

curl "$SWAGGER_CODEGEN_CLI_SRC" -o "$SWAGGER_CODEGEN_CLI"

sed -e "s/[\$]VERSION/${CLIENT_PYTHON_VERSION}/" \
    "${CODEGEN_CONFIG_SRC}" >"${CODEGEN_CONFIG}"

java -jar "$SWAGGER_CODEGEN_CLI" generate \
    -i "$KUBEVIRT_SPEC" \
    -l python \
    -o "${PYTHON_CLIENT_OUT_DIR}" \
    --git-user-id kubevirt \
    --git-repo-id client-python \
    --release-note "Auto-generated client ${CLIENT_PYTHON_VERSION}" \
    -c "${CODEGEN_CONFIG}" &>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt-pysdk-codegen.log

# Replace token name
sed -i \
    -e 's/GIT_TOKEN/API_REFERENCE_PUSH_TOKEN/' \
    "$PYTHON_CLIENT_OUT_DIR"/git_push.sh

# If it is taggged commit create tag in client-python as well.
if [ -n "${TRAVIS_TAG:-}" ]; then
    sed -i \
        -e "/git pull/a \\git tag -a -m \"New KubeVirt release $TRAVIS_TAG\" $TRAVIS_TAG" \
        -e 's/git push/git push --tags/' \
        "$PYTHON_CLIENT_OUT_DIR"/git_push.sh
fi
