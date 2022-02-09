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
	"fmt"
	"net"
	"os"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	vmSchema "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	diskQosAnnotation =  "disk.vm.kubevirt.io/qos"
	onDefineDomainLoggingMessage    = "Hook's OnDefineDomain callback method has been called"
)

type DiskQos struct {
	TotalIopsSec  uint64 `json:"totalIopsSec,omitempty"`
	ReadIopsSec   uint64 `json:"readIopsSec,omitempty"`
	WriteIopsSec  uint64 `json:"writeIopsSec,omitempty"`
	TotalBytesSec uint64 `json:"totalBytesSec,omitempty"`
	ReadBytesSec  uint64 `json:"readBytesSec,omitempty"`
	WriteBytesSec uint64 `json:"writeBytesSec,omitempty"`
}

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "diskQos",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type v1alpha1Server struct{}
type v1alpha2Server struct{}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
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

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := onDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		return nil, err
	}
	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func onDefineDomain(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	var qosList []DiskQos
	annotations := vmiSpec.GetAnnotations()
	diskQos, found := annotations[diskQosAnnotation]; 
	if !found {
		log.Log.Info("Disk qos hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return domainXML, nil
	}
	err = json.Unmarshal([]byte(diskQos), &qosList)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given disk qos: %s", diskQos)
		panic(err)
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	// have one cloud-init cdrom disk
	if len(qosList) != len(domainSpec.Devices.Disks) - 1 {
		log.Log.Reason(err).Errorf("disk qos not match disk number: %s", diskQos)
		panic(err)
	}

	diskList := make([]domainSchema.Disk, 0, len(domainSpec.Devices.Disks))
	for _, disk := range domainSpec.Devices.Disks {
		if disk.Device == "disk" {
			index := 0
			if len(qosList) <=  index {
				break
			}
			qos := qosList[index]
			ioTune := &domainSchema.IOTune{}
			//  TotalIopsSec cannot appear with read_iops_sec or write_iops_sec
			//  TotalBytesSec cannot appear with read_bytes_sec or write_bytes_sec.
			// https://libvirt.org/formatdomain.html#elementsDisks
			if qos.TotalBytesSec != 0  {
				ioTune.TotalBytesSec = qos.TotalBytesSec
			}else{
				ioTune.ReadBytesSec = qos.ReadBytesSec
				ioTune.WriteBytesSec = qos.WriteBytesSec
			}
			if qos.TotalIopsSec != 0 {
				ioTune.TotalIopsSec = qos.TotalIopsSec
			}else{
				ioTune.ReadIopsSec = qos.ReadBytesSec
				ioTune.WriteIopsSec = qos.WriteIopsSec
			}
			disk.IOTune = ioTune
			log.Log.Infof("disk after set ioTune: %+v", disk)
			
			index++
		}
		diskList = append(diskList, disk)
	}
	domainSpec.Devices.Disks = diskList

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested disk qos attributes")

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging("ecx-hook-sidecar")

	var version string
	pflag.StringVar(&version, "version", "", "hook version to use")
	pflag.Parse()

	socketPath := hooks.HookSocketsSharedDirectory + "/ecx.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	if version == "" {
		panic(fmt.Errorf("usage: \n        /ecx-hook-sidecar --version v1alpha1|v1alpha2"))
	}
	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
