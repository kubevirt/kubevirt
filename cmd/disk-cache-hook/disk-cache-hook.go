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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	onDefineDomainLoggingMessage = "Hook's OnDefineDomain callback method has been called"
	hookName                     = "disk-driver-cache-hook"
	version                      = "v1alpha1"
)

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: hookName,
		Versions: []string{
			version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type hookServer struct {
	cacheType string
}

func (u hookServer) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	domainSpec := domainSchema.DomainSpec{}
	err := xml.Unmarshal(params.GetDomainXML(), &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", params.GetDomainXML())
		panic(err)
	}

	newDomainXML, err := onDefineDomain(domainSpec, u.cacheType)
	if err != nil {
		return nil, err
	}

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func onDefineDomain(domainSpec domainSchema.DomainSpec, cacheType string) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	for index, curDisk := range domainSpec.Devices.Disks {
		diskName := fmt.Sprintf("disk-%d", index)
		if curDisk.Alias != nil {
			diskName = curDisk.Alias.GetName()
		}

		if curDisk.Driver == nil {
			curDisk.Driver = &domainSchema.DiskDriver{}
		}

		curDisk.Driver.Cache = cacheType
		log.Log.Infof("Successfully updated disk %s cache mode to %s", diskName, cacheType)
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging(hookName)

	var cacheType string
	pflag.StringVar(&cacheType, "cache-type", "", "virtual disk driver cache type to apply")
	pflag.Parse()

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, fmt.Sprintf("%s.sock", hookName))
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, hookServer{cacheType})
	server.Serve(socket)
}
