/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2019 Red Hat, Inc.
 *
 */
package cmdserver

import (
	"context"

	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

type InfoServer struct{}

func (i InfoServer) Info(context.Context, *info.CmdInfoRequest) (*info.CmdInfoResponse, error) {

	// since this is the first versioned version, we only support the current versions
	// add older versions as soon as they are supported
	return &info.CmdInfoResponse{
		SupportedCmdVersions: []uint32{cmdv1.CmdVersion},
	}, nil

}

func registerInfoServer(grpcServer *grpc.Server) {

	infoServer := &InfoServer{}
	info.RegisterCmdInfoServer(grpcServer, infoServer)

}
