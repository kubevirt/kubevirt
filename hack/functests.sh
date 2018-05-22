#!/bin/bash
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
# Copyright 2017 Red Hat, Inc.
#

set -e

 source hack/common.sh
 source hack/config.sh

functest_docker_prefix=${manifest_docker_prefix-${docker_prefix}}

START_TIME=$(vagrant ssh master -c 'date +"%x %T"' -- -T)
${TESTS_OUT_DIR}/tests.test -kubeconfig=${kubeconfig} -tag=${docker_tag} -prefix=${functest_docker_prefix} -kubectl-path=${kubectl} -test.timeout 60m ${FUNC_TEST_ARGS}
STOP_TIME=$(vagrant ssh master -c 'date +"%x %T"' -- -T)

# get denials from master
echo "------ master ------" >> selinux.log
set +e
vagrant ssh master -c "sudo ausearch -m avc --start ${START_TIME} --end ${STOP_TIME}" >> selinux.log
RESULT=$?
if [ $RESULT != 0 ] && [ $RESULT != 1 ]; then
  echo ausearch failed
  exit 1
fi

# get denials from nodes
if [ -n "${VAGRANT_NUM_NODES}" ]; then 
  for i in $(seq 0 $(($VAGRANT_NUM_NODES-1))); do
    echo "------ node${i} -------" >> selinux.log
    vagrant ssh node${i} -c "sudo ausearch -m avc --start ${START_TIME} --end ${STOP_TIME}" >> selinux.log
    RESULT=$?
    if [ $RESULT != 0 ] && [ $RESULT != 1 ]; then
      echo ausearch failed
      exit 1
    fi
  done
fi

set -e
