/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package istio

const (
	// InjectSidecarLabel Specifies whether an Envoy sidecar should be automatically injected into the workload
	// https://istio.io/latest/docs/reference/config/labels/#SidecarInject
	InjectSidecarLabel = "sidecar.istio.io/inject"
	// InjectSidecarAnnotation in VM/VMI API, propagates sidecar.istio.io/inject label to the virt-launcher pod
	InjectSidecarAnnotation = "sidecar.istio.io/inject"

	// KubeVirtTrafficAnnotation Specifies a comma separated list of virtual interfaces
	// whose inbound traffic (from VM) will be treated as outbound
	// https://istio.io/latest/docs/reference/config/annotations/#SidecarTrafficKubevirtInterfaces
	// This annotation was deprecated in Istio 1.25 in favor of RerouteVirtualInterfacesAnnotation
	// https://istio.io/latest/news/releases/1.25.x/announcing-1.25/change-notes/#deprecation-notices
	KubeVirtTrafficAnnotation = "traffic.sidecar.istio.io/kubevirtInterfaces"

	// RerouteVirtualInterfacesAnnotation Specifies a comma separated list of virtual interfaces
	// whose inbound traffic (from VM) will be treated as outbound
	// https://istio.io/latest/docs/reference/config/annotations/#IoIstioRerouteVirtualInterfaces
	// Introduced in Istio v1.25
	RerouteVirtualInterfacesAnnotation = "istio.io/reroute-virtual-interfaces"
)
