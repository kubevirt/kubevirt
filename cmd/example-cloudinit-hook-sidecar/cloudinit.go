/*
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
 * Copyright 2019 StackPath, LLC
 *
 */

// Inspired by cmd/example-hook-sidecar

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	"kubevirt.io/kubevirt/pkg/log"
)

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "cloudinit",
		Versions: []string{
			hooksV1alpha2.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.PreCloudInitIsoHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Warning("Hook's OnDefineDomain callback method has been called which should never happen")
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: params.GetDomainXML(),
	}, nil
}

func (s v1alpha2Server) PreCloudInitIso(ctx context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	log.Log.Info("Hook's PreCloudInitIso callback method has been called")

	vmiJSON := params.GetVmi()
	vmi := v1.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmi)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	cloudInitDataJSON := params.GetCloudInitData()
	cloudInitData := v1.CloudInitNoCloudSource{}
	err = json.Unmarshal(cloudInitDataJSON, &cloudInitData)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given CloudInitNoCloudSource: %s", cloudInitDataJSON)
		panic(err)
	}

	cloudInitData.UserData = "#cloud-config\n"
	cloudInitData.UserDataBase64 = ""

	response, err := json.Marshal(cloudInitData)
	if err != nil {
		return &hooksV1alpha2.PreCloudInitIsoResult{
			CloudInitData: params.GetCloudInitData(),
		}, fmt.Errorf("Failed to marshal CloudInitNoCloudSource: %v", cloudInitData)

	}

	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: response,
	}, nil
}

func main() {
	log.InitializeLogging("cloudinit-hook-sidecar")

	socketPath := hooks.HookSocketsSharedDirectory + "/cloudinit.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha2' services on socket %s", socketPath)
	server.Serve(socket)
}
