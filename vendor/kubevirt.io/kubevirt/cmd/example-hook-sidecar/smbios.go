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

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	"kubevirt.io/kubevirt/pkg/log"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "smbios",
		Versions: []string{
			hooksV1alpha1.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")

	vmiJSON := params.GetVmi()
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()

	if _, found := annotations[baseBoardManufacturerAnnotation]; !found {
		log.Log.Info("SM BIOS hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return &hooksV1alpha1.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	domainXML := params.GetDomainXML()
	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	domainSpec.OS.SMBios = &domainSchema.SMBios{Mode: "sysinfo"}

	if domainSpec.SysInfo == nil {
		domainSpec.SysInfo = &domainSchema.SysInfo{}
	}
	domainSpec.SysInfo.Type = "smbios"
	if baseBoardManufacturer, found := annotations[baseBoardManufacturerAnnotation]; found {
		domainSpec.SysInfo.BaseBoard = append(domainSpec.SysInfo.BaseBoard, domainSchema.Entry{
			Name:  "manufacturer",
			Value: baseBoardManufacturer,
		})
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested SMBIOS attributes")

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	log.InitializeLogging("smbios-hook-sidecar")

	socketPath := hooks.HookSocketsSharedDirectory + "/smbios.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
