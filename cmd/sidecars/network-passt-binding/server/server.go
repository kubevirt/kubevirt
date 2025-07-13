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

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"google.golang.org/grpc"

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/cmd/sidecars/network-passt-binding/callback"
	"kubevirt.io/kubevirt/cmd/sidecars/network-passt-binding/domain"

	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"
)

type InfoServer struct {
	Version string
}

func (s InfoServer) Info(_ context.Context, _ *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	return &hooksInfo.InfoResult{
		Name: "network-passt-binding",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
			{
				Name:     hooksInfo.ShutdownHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type V1alpha3Server struct {
	Done chan struct{}
}

func (s V1alpha3Server) OnDefineDomain(
	_ context.Context,
	params *hooksV1alpha3.OnDefineDomainParams,
) (*hooksV1alpha3.OnDefineDomainResult, error) {
	vmi := &vmschema.VirtualMachineInstance{}
	if err := json.Unmarshal(params.GetVmi(), vmi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VMI: %v", err)
	}

	useVirtioTransitional := vmi.Spec.Domain.Devices.UseVirtioTransitional != nil && *vmi.Spec.Domain.Devices.UseVirtioTransitional

	const istioInjectAnnotation = "sidecar.istio.io/inject"
	istioProxyInjectionEnabled := false
	if val, ok := vmi.GetAnnotations()[istioInjectAnnotation]; ok {
		istioProxyInjectionEnabled = strings.EqualFold(val, "true")
	}

	opts := domain.NetworkConfiguratorOptions{
		UseVirtioTransitional:      useVirtioTransitional,
		IstioProxyInjectionEnabled: istioProxyInjectionEnabled,
	}

	passtConfigurator, err := domain.NewPasstNetworkConfigurator(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, opts, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create passt configurator: %v", err)
	}

	newDomainXML, err := callback.OnDefineDomain(params.GetDomainXML(), passtConfigurator)
	if err != nil {
		return nil, err
	}

	return &hooksV1alpha3.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s V1alpha3Server) PreCloudInitIso(
	_ context.Context,
	params *hooksV1alpha3.PreCloudInitIsoParams,
) (*hooksV1alpha3.PreCloudInitIsoResult, error) {
	return &hooksV1alpha3.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func (s V1alpha3Server) Shutdown(_ context.Context, _ *hooksV1alpha3.ShutdownParams) (*hooksV1alpha3.ShutdownResult, error) {
	log.Log.Info("Shutdown passt network binding")
	s.Done <- struct{}{}
	return &hooksV1alpha3.ShutdownResult{}, nil
}

func waitForShutdown(server *grpc.Server, errChan <-chan error, shutdownChan <-chan struct{}) {
	// Handle signals to properly shutdown process
	signalStopChan := make(chan os.Signal, 1)
	signal.Notify(signalStopChan, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	var err error
	select {
	case s := <-signalStopChan:
		log.Log.Infof("passt sidecar received signal: %s", s.String())
	case err = <-errChan:
		log.Log.Reason(err).Error("Failed to run grpc server")
	case <-shutdownChan:
		log.Log.Info("Exiting")
	}

	if err == nil {
		server.GracefulStop()
	}
}

func Serve(server *grpc.Server, socket net.Listener, shutdownChan <-chan struct{}) {
	errChan := make(chan error)
	go func() {
		errChan <- server.Serve(socket)
	}()

	waitForShutdown(server, errChan, shutdownChan)
}
