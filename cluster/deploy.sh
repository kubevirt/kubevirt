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

source hack/common.sh
source cluster/$PROVIDER/provider.sh
source hack/config.sh

echo "Deploying ..."

# Deploy the right manifests for the right target
if [[ -z $TARGET ]] || [[ $TARGET =~ .*-dev ]]; then
    _kubectl create -f ${MANIFESTS_OUT_DIR}/dev -R $i
elif [[ $TARGET =~ .*-release ]] || [[ $TARGET == windows ]]; then
    for manifest in ${MANIFESTS_OUT_DIR}/release/*; do
        if [[ $manifest =~ .*demo.* ]]; then
            continue
        fi
        _kubectl create -f $manifest
    done
fi

# Deploy additional infra for testing
_kubectl create -f ${MANIFESTS_OUT_DIR}/testing -R $i

if [ "$PROVIDER" = "vagrant-openshift" ] || [ "$PROVIDER" = "os-3.9.0-alpha.4" ]; then
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-controller -n ${namespace}
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-testing -n ${namespace}
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-privileged -n ${namespace}
    _kubectl adm policy add-scc-to-user privileged -z kubevirt-apiserver -n ${namespace}
    # Helpful for development. Allows admin to access everything KubeVirt creates in the web console
    _kubectl adm policy add-scc-to-user privileged admin
fi

echo "Done"
