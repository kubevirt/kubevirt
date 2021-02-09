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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"

	"google.golang.org/grpc"

	vmSchema "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var done = 0

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "interface",
		Versions: []string{
			hooksV1alpha2.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")
	if done > 1 {
		log.Log.Info("OnDefineDomain already called")
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}
	log.Log.Info("OnDefineDomain first call")
	defer func() {
		done -= 1
	}()

	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func onDefineDomain(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")

	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	for i, _ := range domainSpec.Devices.Interfaces {
		domainSpec.Devices.Interfaces[i].MTU = nil
		if done == 0 {
			domainSpec.Devices.Interfaces[i].Target = &domainSchema.InterfaceTarget{Device: ""}
		}
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested network interfaces attributes")

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging("interface-hook-sidecar")

	socketPath := hooks.HookSocketsSharedDirectory + "/interface.sock"
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
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
