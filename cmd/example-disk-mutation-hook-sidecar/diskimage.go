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
 * Copyright 2022 Nvidia
 *
 */

// Inspired by cmd/example-hook-sidecar

package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	vmSchema "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const bootDiskImageNameAnnotation = "diskimage.vm.kubevirt.io/bootImageName"

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "bootdiskimage",
		Versions: []string{
			hooksV1alpha2.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info("Disk mutation hook's OnDefineDomain callback method has been called")

	domainXML := params.GetDomainXML()
	vmiJSON := params.GetVmi()
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()
	if _, found := annotations[bootDiskImageNameAnnotation]; !found {
		log.Log.Info("Boot disk hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: domainXML,
		}, nil
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	// Get Boot disk
	bootDiskLocation := domainSpec.Devices.Disks[0].Source.File
	dir, diskName := filepath.Split(bootDiskLocation)
	newDiskName := annotations[bootDiskImageNameAnnotation]
	if newDiskName == diskName {
		log.Log.Infof("Boot disk image name is already %q. Returning original domain spec", newDiskName)
		return &hooksV1alpha2.OnDefineDomainResult{
			DomainXML: domainXML,
		}, nil
	}

	bootDiskLocation = filepath.Join(dir, newDiskName)
	domainSpec.Devices.Disks[0].Source.File = bootDiskLocation

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested boot disk attribute")
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func main() {
	log.InitializeLogging("bootdisk-hook-sidecar")

	var version string
	pflag.StringVar(&version, "version", "", "hook version to use")
	pflag.Parse()

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, "bootdisk.sock")
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha2' services on socket %s", socketPath)
	server.Serve(socket)
}
