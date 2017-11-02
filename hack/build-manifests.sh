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

source hack/config.sh

if [ $# -eq 0 ]; then
    args=$manifest_templates
else
    args=$@
fi

# Delete all generated manifests in case an input file was deleted or renamed
rm -f "manifests/*.yaml"

# Render kubernetes manifests
for arg in $args; do
    sed -e "s/{{ master_ip }}/$master_ip/g" \
        -e "s/{{ primary_nic }}/$primary_nic/g" \
        -e "s/{{ docker_tag }}/$docker_tag/g" \
        -e "s/{{ docker_prefix }}/$docker_prefix/g" \
        $arg > ${arg%%.in}
done
