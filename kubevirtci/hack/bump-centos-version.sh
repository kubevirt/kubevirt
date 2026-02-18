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
# Copyright 2022 Red Hat, Inc.
#
#

(
    curl --no-progress-meter -L https://cloud.centos.org/centos/9-stream/x86_64/images/ |
        grep -oE 'a href="(CentOS-Stream-Vagrant-9-[^"]+)"' |
        grep -oE '[0-9]{8}\.[0-9]+' | sort -rV | uniq | head -1
) >./cluster-provision/centos9/version
