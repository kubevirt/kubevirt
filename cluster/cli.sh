#!/bin/bash
set -e

source hack/common.sh
test -t 1 && USE_TTY="-it"
source ${KUBEVIRT_DIR}/cluster/$KUBEVIRT_PROVIDER/provider.sh
source hack/config.sh

${_cli} --prefix $provider_prefix "$@"
