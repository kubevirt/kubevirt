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

package types

import (
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
)

const (
	// ExportServerPort is the port the export server listens on inside the pod.
	ExportServerPort = 8443
	// ExportClusterIPServicePort is the Service port used by ClusterIP export
	// Services created before headless migration. kube-proxy remaps this port
	// to ExportServerPort.
	ExportClusterIPServicePort = 443
)

// ExportServiceDialPort returns the TCP port clients should dial on the
// export Service DNS name.
//
// Headless Services (clusterIP: None) resolve to the pod IP, so the container
// port must be used. ClusterIP Services are remapped by kube-proxy, so the
// Service port (historically 443) must be used.
func ExportServiceDialPort(service *k8sv1.Service) int32 {
	if service != nil && service.Spec.ClusterIP != k8sv1.ClusterIPNone {
		return ExportClusterIPServicePort
	}
	return ExportServerPort
}

// ExportServiceHost is host:port for internal export URLs and ConfigMap
// internal_host entries.
func ExportServiceHost(service *k8sv1.Service) string {
	return fmt.Sprintf("%s.%s.svc:%d", service.Name, service.Namespace, ExportServiceDialPort(service))
}
