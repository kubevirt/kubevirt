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

PROVIDER=${PROVIDER:-vagrant}
source cluster/$PROVIDER/provider.sh
source ${KUBEVIRT_PATH}hack/config.sh

if [ "$1" == "console" ] || [ "$1" == "spice" ]; then
    cmd/virtctl/virtctl "$@" -s http://${master_ip}:8184 
    exit
fi

# Print usage from virtctl and kubectl
if [ "$1" == "--help" ]  || [ "$1" == "-h" ] ; then
    cmd/virtctl/virtctl "$@"
fi

if [ -e  ${KUBEVIRT_PATH}cluster/vagrant/.kubeconfig ] &&
   [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ] &&
   [ "x$1" == "x--core" ]; then
    shift
    _kubectl "$@"
elif [ -e ${KUBEVIRT_PATH}cluster/vagrant/.kubectl ];then
    _kubectl -s http://${master_ip}:8184 "$@"
else
    echo "Did you already run 'cluster/up.sh' to deploy kubevirt?"
fi
