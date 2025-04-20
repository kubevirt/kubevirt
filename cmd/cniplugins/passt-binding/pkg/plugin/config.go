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
 * Copyright The KubeVirt Authors.
 *
 */

package plugin

import (
	"encoding/json"
	"fmt"

	"github.com/containernetworking/cni/pkg/types"
)

// A NetConf structure represents a Multus network attachment definition configuration
type NetConf struct {
	types.NetConf

	Args struct {
		Cni CniArgs `json:"cni,omitempty"`
	} `json:"args,omitempty"`
}

// CniArgs represents extended parameters that are passed as `cni-args`.
type CniArgs struct {
	LogicNetworkName string `json:"logicNetworkName,omitempty"`
}

func loadConf(bytes []byte) (NetConf, string, error) {
	n := NetConf{}
	if err := json.Unmarshal(bytes, &n); err != nil {
		return n, "", fmt.Errorf("failed to load netconf: %v", err)
	}

	return n, n.CNIVersion, nil
}
