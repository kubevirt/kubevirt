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
