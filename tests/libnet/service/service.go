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

package service

import (
	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func BuildHeadlessSpec(serviceName string, exposedPort, portToExpose int32, selectorKey, selectorValue string) *k8sv1.Service {
	service := BuildSpec(serviceName, exposedPort, portToExpose, selectorKey, selectorValue)
	service.Spec.ClusterIP = k8sv1.ClusterIPNone
	return service
}

func BuildIPv6Spec(serviceName string, exposedPort, portToExpose int32, selectorKey, selectorValue string) *k8sv1.Service {
	service := BuildSpec(serviceName, exposedPort, portToExpose, selectorKey, selectorValue)
	ipv6Family := k8sv1.IPv6Protocol
	service.Spec.IPFamilies = []k8sv1.IPFamily{ipv6Family}

	return service
}

func BuildSpec(serviceName string, exposedPort, portToExpose int32, selectorKey, selectorValue string) *k8sv1.Service {
	return &k8sv1.Service{
		ObjectMeta: k8smetav1.ObjectMeta{
			Name: serviceName,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				selectorKey: selectorValue,
			},
			Ports: []k8sv1.ServicePort{
				{Protocol: k8sv1.ProtocolTCP, Port: portToExpose, TargetPort: intstr.FromInt32(exposedPort)},
			},
		},
	}
}
