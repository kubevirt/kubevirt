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

set -ex

vagrant ssh-config master 2>&1 | grep "not yet ready for SSH" >/dev/null \
        && { echo "Master node is not up"; exit 1; }

OPTIONS=`vagrant ssh-config master | grep -v '^Host ' | awk -v ORS=' ' 'NF{print "-o " $1 "=" $2}'`

scp $OPTIONS master:/usr/bin/kubectl ${KUBEVIRT_PATH}cluster/vagrant/.kubectl
chmod u+x cluster/vagrant/.kubectl

vagrant ssh master -c "sudo cat /etc/kubernetes/admin.conf" > ${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig
