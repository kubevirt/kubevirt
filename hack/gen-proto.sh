#!/usr/bin/env bash

set -e

source $(dirname "$0")/common.sh
source $(dirname "$0")/config.sh

protoc --proto_path=pkg/hooks/info --go_out=plugins=grpc,import_path=kubevirt_hooks_info:pkg/hooks/info pkg/hooks/info/api_info.proto
protoc --proto_path=pkg/hooks/v1alpha1 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha1:pkg/hooks/v1alpha1 pkg/hooks/v1alpha1/api_v1alpha1.proto
protoc --proto_path=pkg/hooks/v1alpha2 --go_out=plugins=grpc,import_path=kubevirt_hooks_v1alpha2:pkg/hooks/v1alpha2 pkg/hooks/v1alpha2/api_v1alpha2.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/v1/notify.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/notify/info/info.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/v1/cmd.proto
protoc --go_out=plugins=grpc:. pkg/handler-launcher-com/cmd/info/info.proto
