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
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/util/rand"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
)

const (
	onDefineDomainLoggingMessage  = "OnDefineDomain method has been called"
	preCloudInitIsoLoggingMessage = "PreCloudInitIso method has been called"

	onDefineDomainBin  = "onDefineDomain"
	preCloudInitIsoBin = "preCloudInitIso"
)

type infoServer struct {
	Version string
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Info method has been called")
	supportedHookPoints := map[string]string{
		hooksInfo.OnDefineDomainHookPointName:  onDefineDomainBin,
		hooksInfo.PreCloudInitIsoHookPointName: preCloudInitIsoBin,
	}
	var hookPoints = []*hooksInfo.HookPoint{}

	for hookPointName, binName := range supportedHookPoints {
		if _, err := exec.LookPath(binName); err != nil {
			if errors.Is(err, exec.ErrNotFound) {
				log.Log.Infof("Info: %s has not been found", binName)
			}
			continue
		}

		hookPoints = append(hookPoints, &hooksInfo.HookPoint{
			Name:     hookPointName,
			Priority: 0,
		})
	}

	return &hooksInfo.InfoResult{
		Name: "shim",
		Versions: []string{
			s.Version,
		},
		HookPoints: hookPoints,
	}, nil
}

type v1Alpha1Server struct{}
type v1Alpha2Server struct{}

func (s v1Alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := runOnDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		log.Log.Reason(err).Error("Failed OnDefineDomain")
		return nil, err
	}
	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s v1Alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	log.Log.Info(preCloudInitIsoLoggingMessage)
	cloudInitData, err := runPreCloudInitIso(params.GetVmi(), params.GetCloudInitData())
	if err != nil {
		log.Log.Reason(err).Error("Failed ProCloudInitIso")
		return nil, err
	}
	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: cloudInitData,
	}, nil
}

func (s v1Alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)
	newDomainXML, err := runOnDefineDomain(params.GetVmi(), params.GetDomainXML())
	if err != nil {
		log.Log.Reason(err).Error("Failed OnDefineDomain")
		return nil, err
	}
	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func runPreCloudInitIso(vmiJSON []byte, cloudInitDataJSON []byte) ([]byte, error) {
	// Check binary exists
	if _, err := exec.LookPath(preCloudInitIsoBin); err != nil {
		return nil, fmt.Errorf("Failed in finding %s in $PATH: %v", preCloudInitIsoBin, err)
	}

	// Validate params before calling hook script
	vmiSpec := virtv1.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal given VMI spec: %s due %v", vmiJSON, err)
	}

	cloudInitData := cloudinit.CloudInitData{}
	err := json.Unmarshal(cloudInitDataJSON, &cloudInitData)
	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal given CloudInitData: %s due %v", cloudInitDataJSON, err)
	}

	args := append([]string{},
		"--vmi", string(vmiJSON),
		"--cloud-init", string(cloudInitDataJSON))

	log.Log.Infof("Executing %s", preCloudInitIsoBin)
	command := exec.Command(preCloudInitIsoBin, args...)
	return command.Output()
}

func runOnDefineDomain(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	if _, err := exec.LookPath(onDefineDomainBin); err != nil {
		return nil, fmt.Errorf("Failed in finding %s in $PATH due %v", onDefineDomainBin, err)
	}

	vmiSpec := virtv1.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal given VMI spec: %s due %v", vmiJSON, err)
	}

	args := append([]string{},
		"--vmi", string(vmiJSON),
		"--domain", string(domainXML))

	log.Log.Infof("Executing %s", onDefineDomainBin)
	command := exec.Command(onDefineDomainBin, args...)
	return command.Output()
}

func parseCommandLineArgs() (string, error) {
	supportedVersions := []string{"v1alpha1", "v1alpha2"}
	version := ""

	pflag.StringVar(&version, "version", "", "hook version to use")
	pflag.Parse()
	if version == "" {
		return "", fmt.Errorf("Missing --version parameter. Supported options are %s.", supportedVersions)
	}

	supported := false
	for _, v := range supportedVersions {
		if v == version {
			supported = true
			break
		}
	}
	if !supported {
		return "", fmt.Errorf("Version %s is not supported. Supported options are %s.", version, supportedVersions)
	}

	return version, nil
}

func getSocketPath() (string, error) {
	if _, err := os.Stat(hooks.HookSocketsSharedDirectory); err != nil {
		return "", fmt.Errorf("Failed dir %s due %s", hooks.HookSocketsSharedDirectory, err.Error())
	}

	// In case there are multiple shims being used, append random string and try a few times
	for i := 0; i < 10; i++ {
		socketName := fmt.Sprintf("shim-%s.sock", rand.String(4))
		socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, socketName)
		if _, err := os.Stat(socketPath); !errors.Is(err, os.ErrNotExist) {
			log.Log.Infof("Failed socket %s due %s", socketName, err.Error())
			continue
		}
		return socketPath, nil
	}

	return "", fmt.Errorf("Failed generate socket path")
}

func main() {
	log.InitializeLogging("shim-sidecar")

	// Shim arguments
	version, err := parseCommandLineArgs()
	if err != nil {
		log.Log.Reason(err).Errorf("Input error")
		os.Exit(1)
	}

	socketPath, err := getSocketPath()
	if err != nil {
		log.Log.Reason(err).Errorf("Enviroment error")
		os.Exit(1)
	}

	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		os.Exit(1)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha1.RegisterCallbacksServer(server, v1Alpha1Server{})
	hooksV1alpha2.RegisterCallbacksServer(server, v1Alpha2Server{})
	log.Log.Infof("shim is now exposing its services on socket %s", socketPath)
	server.Serve(socket)
}
