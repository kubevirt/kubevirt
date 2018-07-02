#!/usr/bin/env bash

source $(dirname "$0")/../common.sh

set -o errexit
set -o nounset
set -o pipefail

SWAGGER_CODEGEN_CLI_SRC=http://central.maven.org/maven2/io/swagger/swagger-codegen-cli/2.2.3/swagger-codegen-cli-2.2.3.jar
SWAGGER_CODEGEN_CLI="/tmp/swagger-codegen-cli.jar"
KUBEVIRT_SPEC="${KUBEVIRT_DIR}/api/openapi-spec/swagger.json"
CODEGEN_CONFIG_SRC="${KUBEVIRT_DIR}/hack/gen-client-python/swagger-codegen-config.json.in"
CODEGEN_CONFIG="${PYTHON_CLIENT_OUT_DIR}/swagger-codegen-config.json"
HARD_CODED_MODULES="${KUBEVIRT_DIR}/hack/gen-client-python/hard-coded-modules"

# Define version of client
if [ -n "${TRAVIS_TAG:-}" ]; then
    CLIENT_PYTHON_VERSION="$TRAVIS_TAG"
else
    CLIENT_PYTHON_VERSION="$(git describe || echo 'none')"
fi

mkdir -p "${PYTHON_CLIENT_OUT_DIR}"

# Download swagger code generator
curl "$SWAGGER_CODEGEN_CLI_SRC" -o "$SWAGGER_CODEGEN_CLI"

# Generate config file for swagger code generator
sed -e "s/[\$]VERSION/${CLIENT_PYTHON_VERSION}/" \
    "${CODEGEN_CONFIG_SRC}" >"${CODEGEN_CONFIG}"

# Generate python client
java -jar "$SWAGGER_CODEGEN_CLI" generate \
    -i "$KUBEVIRT_SPEC" \
    -l python \
    -o "${PYTHON_CLIENT_OUT_DIR}" \
    --git-user-id kubevirt \
    --git-repo-id client-python \
    --release-note "Auto-generated client ${CLIENT_PYTHON_VERSION}" \
    -c "${CODEGEN_CONFIG}" &>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt-pysdk-codegen.log

cp "${HARD_CODED_MODULES}"/* "${PYTHON_CLIENT_OUT_DIR}"/kubevirt/models

echo "from .v1_interface_bridge import V1InterfaceBridge" >>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt/models/__init__.py
echo "from .v1_interface_slirp import V1InterfaceSlirp" >>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt/models/__init__.py

echo "from .models.v1_interface_bridge import V1InterfaceBridge" >>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt/__init__.py
echo "from .models.v1_interface_slirp import V1InterfaceSlirp" >>"${PYTHON_CLIENT_OUT_DIR}"/kubevirt/__init__.py
