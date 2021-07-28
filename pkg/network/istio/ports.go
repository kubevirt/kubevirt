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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package istio

import "fmt"

const (
	EnvoyAdminPort                     = 15000
	EnvoyOutboundPort                  = 15001
	EnvoyInboundPort                   = 15006
	EnvoyTunnelPort                    = 15008
	EnvoyMergedPrometheusTelemetryPort = 15020
	EnvoyHealthCheckPort               = 15021
	EnvoyPrometheusTelemetryPort       = 15090
)

func ReservedPorts() []string {
	return []string{
		fmt.Sprint(EnvoyAdminPort),
		fmt.Sprint(EnvoyOutboundPort),
		fmt.Sprint(EnvoyInboundPort),
		fmt.Sprint(EnvoyTunnelPort),
		fmt.Sprint(EnvoyMergedPrometheusTelemetryPort),
		fmt.Sprint(EnvoyHealthCheckPort),
		fmt.Sprint(EnvoyPrometheusTelemetryPort),
	}
}
