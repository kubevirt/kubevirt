#!/usr/bin/env bash

set -e

source $(dirname "$0")/../../hack/common.sh

cd ${PYTHON_CLIENT_OUT_DIR}
bash git_push.sh
