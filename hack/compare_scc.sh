#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2020 Red Hat, Inc.
#

set -ex

OUTPUT_DIR=${OUTPUT_DIR:-_out}
SCC_OUTPUT_DIR="${OUTPUT_DIR}/scc"

BEFORE_DIR="${SCC_OUTPUT_DIR}/before"
AFTER_DIR="${SCC_OUTPUT_DIR}/after"

SUFFIX=.json

function dump_sccs() {
  TARGET_DIR=$1
  if [ "${CMD}" == "oc" ] && [ "${KUBEVIRT_PROVIDER}" != "external" ]; then
    mkdir -p "${TARGET_DIR}"
    for SCCNAME in $( ${CMD} get scc -o custom-columns=:metadata.name); do
      echo -e "\n--- SCC ${SCCNAME} ---"
      ${CMD} get scc "${SCCNAME}" -o json | jq 'del(.metadata.resourceVersion,.metadata.generation,.metadata.labels,.metadata.annotations)' > "${TARGET_DIR}/${SCCNAME}${SUFFIX}" || true
    done
  else
    echo "Ignoring SCCs on k8s"
  fi
}

function dump_sccs_before() {
  dump_sccs "${BEFORE_DIR}"
}

function dump_sccs_after() {
  dump_sccs "${AFTER_DIR}"

  compare_sccs
}

function compare_sccs() {
  if [ "${CMD}" == "oc" ] && [ "${KUBEVIRT_PROVIDER}" != "external" ]; then
    for f in "${BEFORE_DIR}"/*"${SUFFIX}"; do
      SCCNAME=$(basename --suffix="${SUFFIX}" "$f")
      echo -e "\n--- comparing SCC ${SCCNAME} ---"
      diff "${BEFORE_DIR}/${SCCNAME}${SUFFIX}" "${AFTER_DIR}/${SCCNAME}${SUFFIX}"
    done
  else
    echo "Ignoring SCCs on k8s"
  fi
}
