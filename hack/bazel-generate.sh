#!/usr/bin/env bash
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
# Copyright The KubeVirt Authors.
#

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh

# remove libvirt BUILD file to regenerate it each time
rm -f vendor/libvirt.org/go/libvirt/BUILD.bazel

# generate BUILD files
bazel run \
    --config=${ARCHITECTURE} \
    //:gazelle -- -exclude vendor/google.golang.org/grpc --exclude kubevirtci/cluster-up

# inject changes to libvirt BUILD file
bazel run \
    --config=${ARCHITECTURE} \
    -- :buildozer 'add cdeps //:libvirt-libs' //vendor/libvirt.org/go/libvirt:go_default_library
# align BAZEL files to a single format
bazel run \
    --config=${ARCHITECTURE} \
    //:buildifier
