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
 * Copyright The KubeVirt Authors.
 *
 */

package driver

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
	drav1 "k8s.io/kubelet/pkg/apis/dra/v1"
	pluginapi "k8s.io/kubelet/pkg/apis/pluginregistration/v1"
)

type RegistrationServer struct {
	pluginapi.UnimplementedRegistrationServer
	driverName string
	endpoint   string
}

func NewRegistrationServer(driverName, endpoint string) *RegistrationServer {
	return &RegistrationServer{driverName: driverName, endpoint: endpoint}
}

func (r *RegistrationServer) GetInfo(ctx context.Context, req *pluginapi.InfoRequest) (*pluginapi.PluginInfo, error) {
	return &pluginapi.PluginInfo{
		Type:              pluginapi.DRAPlugin,
		Name:              r.driverName,
		Endpoint:          r.endpoint,
		SupportedVersions: []string{drav1.DRAPluginService},
	}, nil
}

func (r *RegistrationServer) NotifyRegistrationStatus(ctx context.Context, status *pluginapi.RegistrationStatus) (*pluginapi.RegistrationStatusResponse, error) {
	if !status.PluginRegistered {
		log.Printf("Registration failed: %s", status.Error)
	}
	return &pluginapi.RegistrationStatusResponse{}, nil
}

func (r *RegistrationServer) Serve(socketPath string) error {
	os.Remove(socketPath)
	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pluginapi.RegisterRegistrationServer(grpcServer, r)
	return grpcServer.Serve(lis)
}
