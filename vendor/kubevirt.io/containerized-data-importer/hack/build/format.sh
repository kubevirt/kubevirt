#!/usr/bin/env bash

#Copyright 2018 The CDI Authors.
#
#Licensed under the Apache License, Version 2.0 (the "License");
#you may not use this file except in compliance with the License.
#You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#Unless required by applicable law or agreed to in writing, software
#distributed under the License is distributed on an "AS IS" BASIS,
#WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#See the License for the specific language governing permissions and
#limitations under the License.

set -eo pipefail

script_dir="$(readlink -f $(dirname $0))"
source "${script_dir}"/common.sh
source "${script_dir}"/config.sh

shfmt -i 4 -w ${CDI_DIR}/hack ${CLONER_MAIN}
goimports -w -local kubevirt.io ${CDI_DIR}/cmd/ ${CDI_DIR}/pkg/ ${CDI_DIR}/tests/
(cd ${CDI_DIR} && go vet $(go list ./... | grep -v -E "vendor|pkg/client" | sort -u))
