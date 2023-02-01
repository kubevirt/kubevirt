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

package multus

import (
	"encoding/json"
	"fmt"

	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

func MapInterfaceNameToNetworkStatus(networkStatusAnnotationValue string) (map[string]networkv1.NetworkStatus, error) {
	if networkStatusAnnotationValue == "" {
		return nil, fmt.Errorf("network-status annotation is not present")
	}
	var networkStatusList []networkv1.NetworkStatus
	if err := json.Unmarshal([]byte(networkStatusAnnotationValue), &networkStatusList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal network-status annotation: %v", err)
	}

	multusInterfaceNameToNetworkStatusMap := map[string]networkv1.NetworkStatus{}
	for _, networkStatus := range networkStatusList {
		multusInterfaceNameToNetworkStatusMap[networkStatus.Interface] = networkStatus
	}

	return multusInterfaceNameToNetworkStatusMap, nil
}
