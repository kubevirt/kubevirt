source $(dirname "$0")/../hack/common.sh

source ${KUBEVIRT_DIR}/cluster/$PROVIDER/provider.sh
source hack/config.sh

${_cli} "$@" --prefix $provider_prefix
