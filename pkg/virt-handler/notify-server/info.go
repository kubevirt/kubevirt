/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package eventsserver

import (
	"context"

	"google.golang.org/grpc"

	"kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/info"
	notifyv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/notify/v1"
)

type InfoServer struct{}

func (i InfoServer) Info(context.Context, *info.NotifyInfoRequest) (*info.NotifyInfoResponse, error) {

	// since this is the first versioned version, we only support the current versions
	// add older versions as soon as they are supported
	return &info.NotifyInfoResponse{
		SupportedNotifyVersions: []uint32{notifyv1.NotifyVersion},
	}, nil

}

func registerInfoServer(grpcServer *grpc.Server) {

	infoServer := &InfoServer{}
	info.RegisterNotifyInfoServer(grpcServer, infoServer)

}
