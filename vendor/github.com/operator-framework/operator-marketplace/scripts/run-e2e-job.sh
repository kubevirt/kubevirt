set -o errexit
set -o nounset
set -o pipefail

MARKETPLACE_OPERATOR_ROOT=$(dirname "${BASH_SOURCE}")/..
SDK_VERSION=v0.3.0

# Get operator-sdk binary.
wget -O /tmp/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/${SDK_VERSION}/operator-sdk-${SDK_VERSION}-x86_64-linux-gnu && chmod +x /tmp/operator-sdk

PATH=$PATH:/tmp
cd $MARKETPLACE_OPERATOR_ROOT
. ./scripts/e2e-tests.sh
