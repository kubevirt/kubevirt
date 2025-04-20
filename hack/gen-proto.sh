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


set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=info:pkg/hooks/info pkg/hooks/info/api_info.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=v1alpha1:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api_v1alpha1.proto
protoc --proto_path=pkg/hooks/v1alpha2 --go_out=plugins=grpc,import_path=v1alpha2:pkg/hooks/v1alpha2 pkg/hooks/v1alpha2/api_v1alpha2.proto
protoc --proto_path=pkg/hooks/v1alpha3 --go_out=plugins=grpc,import_path=v1alpha3:pkg/hooks/v1alpha3 pkg/hooks/v1alpha3/api_v1alpha3.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/v1/notify.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/info/info.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/v1/cmd.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/info/info.proto
protoc --go_out=plugins=grpc:. pkg/vsock/system/v1/system.proto
