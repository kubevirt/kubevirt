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

SCC_BEFORE_SUFFIX=.yaml.before
SCC_AFTER_SUFFIX=.yaml.after

function dump_sccs_before(){
    if [ "${CMD}" == "oc" ]; then
        mkdir -pv "${SCC_OUTPUT_DIR}"
        for SCCNAME in $( ${CMD} get scc -o custom-columns=:metadata.name )
        do
      echo -e "\n--- SCC ${SCCNAME} ---"
          ${CMD} get scc ${SCCNAME} -o yaml > "${SCC_OUTPUT_DIR}/${SCCNAME}${SCC_BEFORE_SUFFIX}" || true
    done
  else
    echo "Ignoring SCCs on k8s"
  fi
}

function dump_sccs_after(){
  if [ "${CMD}" == "oc" ] && [ "${KUBEVIRT_PROVIDER}" != "external" ]; then
        for f in "${SCC_OUTPUT_DIR}"/*"${SCC_BEFORE_SUFFIX}"; do
           SCCNAME=$(basename --suffix=${SCC_BEFORE_SUFFIX} "$f")
           echo -e "\n--- SCC ${SCCNAME} ---"
           ${CMD} get scc ${SCCNAME} -o yaml > "${SCC_OUTPUT_DIR}/${SCCNAME}${SCC_AFTER_SUFFIX}" || true
           diff -I '^\s*generation:.*$' -I '^\s*resourceVersion:.*$' -I '^\s*time:.*$' "${SCC_OUTPUT_DIR}/${SCCNAME}${SCC_BEFORE_SUFFIX}" "${SCC_OUTPUT_DIR}/${SCCNAME}${SCC_AFTER_SUFFIX}"
    done
  else
    echo "Ignoring SCCs on k8s"
  fi
}
