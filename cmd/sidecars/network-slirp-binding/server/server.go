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

	vmschema "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/log"

	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"

	"kubevirt.io/kubevirt/cmd/sidecars/network-slirp-binding/callback"
	"kubevirt.io/kubevirt/cmd/sidecars/network-slirp-binding/domain"
)

type InfoServer struct {
	Version string
}

func (s InfoServer) Info(_ context.Context, _ *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Infof("Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "network-slirp-binding",
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

type V1alpha2Server struct {
	SearchDomains []string
}

func (s V1alpha2Server) OnDefineDomain(_ context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info("OnDefineDomain callback method has been called")

	vmi := &vmschema.VirtualMachineInstance{}
	if err := json.Unmarshal(params.GetVmi(), vmi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VMI: %v", err)
	}

	slirpConfigurator, err := domain.NewSlirpNetworkConfigurator(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, s.SearchDomains)
	if err != nil {
		return nil, fmt.Errorf("failed to create slirp configurator: %v", err)
	}

	newDomainXML, err := callback.OnDefineDomain(params.GetDomainXML(), slirpConfigurator)
	if err != nil {
		return nil, err
	}

	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s V1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	log.Log.Info("PreCloudInitIso method has been called")

	return &hooksV1alpha2.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}
