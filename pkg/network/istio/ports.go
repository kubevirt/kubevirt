/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package istio

const (
	EnvoyAdminPort                     = 15000
	EnvoyOutboundPort                  = 15001
	EnvoyDebugPort                     = 15004
	EnvoyInboundPort                   = 15006
	EnvoyTunnelPort                    = 15008
	EnvoySecureNetworkPort             = 15009
	EnvoyMergedPrometheusTelemetryPort = 15020
	EnvoyHealthCheckPort               = 15021
	EnvoyDNSPort                       = 15053
	EnvoyPrometheusTelemetryPort       = 15090
	SSHPort                            = 22
)

func ReservedPorts() []uint {
	return []uint{
		EnvoyAdminPort,
		EnvoyOutboundPort,
		EnvoyDebugPort,
		EnvoyInboundPort,
		EnvoyTunnelPort,
		EnvoySecureNetworkPort,
		EnvoyMergedPrometheusTelemetryPort,
		EnvoyHealthCheckPort,
		EnvoyDNSPort,
		EnvoyPrometheusTelemetryPort,
	}
}

func NonProxiedPorts() []int {
	return []int{
		SSHPort,
	}
}
