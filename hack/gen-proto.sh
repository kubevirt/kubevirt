#!/usr/bin/env bash

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
protoc --go_out=plugins=grpc:. pkg/synchronizer-com/synchronization/v1/synchronization.proto
