#!/bin/env bash
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
# Copyright 2024 Red Hat, Inc.
#
set -e

source hack/common.sh
source hack/bootstrap.sh
source hack/config.sh


bazel build \
    --config=${ARCHITECTURE} \
    --@io_bazel_rules_go//go/config:gc_goopts=-N,-l \
    --strip=never \
    --define container_prefix= \
    --define image_prefix= \
    --define container_tag= \
    //cmd/virt-controller:virt-controller-image

bazel run \
    --config=${ARCHITECTURE} \
    --@io_bazel_rules_go//go/config:gc_goopts=-N,-l \
    --strip=never \
    --define container_prefix=${docker_prefix} \
    --define image_prefix=${image_prefix} \
    --define container_tag=debug \
    //:push-virt-controller
