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
# Copyright 2017 Red Hat, Inc.
#

set -e

if [[ -z "$push_log_file" ]]; then
    echo "PUSH_LOG_FILE is empty: won't use shasums, falling back to tags"
    return
fi

if [[ ! -f "$push_log_file" ]]; then
    echo "$push_log_file not found: won't use shasums, falling back to tags"
    return
fi

# example log line: index.docker.io/kubevirt/virt-handler:v0.19.0 was published with digest: sha256:48104bcbc1d5f11b8f98d3f6ad875871ec3714482771b2c2ac7406aa02a05b00

# 1. find virt-* images
# 2. remove repo
# 3. remove tag
# 4. replace - with _
# 5. print uppercase image_SHA=shasum
# 6. source everything
#
# results in e.g. $VIRT_HANDLER_SHA = sha256:48104bcbc1d5f11b8f98d3f6ad875871ec3714482771b2c2ac7406aa02a05b00
#
source <(awk '$1 ~ /.*virt-.*/ { sub(".*/", "", $1) ; sub(":.*", "", $1) ; sub("-", "_", $1) ; print toupper($1) "_SHA=" $6}' "$push_log_file")
