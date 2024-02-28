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
 * Copyright 2024 Red Hat, Inc.
 *
 */

package libipam

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

type IPAMClaimParams struct {
	ClaimName   string
	NetworkName string
}

type netConf struct {
	AllowPersistentIPs bool   `json:"allowPersistentIPs,omitempty"`
	Name               string `json:"name,omitempty"`
}

func GetPersistentIPsConf(nad *networkv1.NetworkAttachmentDefinition) (netConf, error) {
	if nad.Spec.Config == "" {
		return netConf{}, nil
	}

	var conf netConf
	err := json.Unmarshal([]byte(nad.Spec.Config), &conf)
	if err != nil {
		return netConf{}, fmt.Errorf("failed to unmarshal NAD spec.config JSON: %v", err)
	}

	if conf.Name == "" {
		return netConf{}, fmt.Errorf("failed to obtain network name: missing required field")
	}

	return conf, nil
}
