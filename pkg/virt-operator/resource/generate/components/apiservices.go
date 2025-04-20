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
 */

package components

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	v1 "kubevirt.io/api/core/v1"
)

func NewVirtAPIAPIServices(installNamespace string) []*apiregv1.APIService {
	apiservices := []*apiregv1.APIService{}

	for _, version := range v1.SubresourceGroupVersions {
		subresourceAggregatedApiName := version.Version + "." + version.Group

		apiservices = append(apiservices, &apiregv1.APIService{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apiregistration.k8s.io/v1",
				Kind:       "APIService",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: subresourceAggregatedApiName,
				Labels: map[string]string{
					v1.AppLabel:       "virt-api-aggregator",
					v1.ManagedByLabel: v1.ManagedByLabelOperatorValue,
				},
				Annotations: map[string]string{
					certificatesSecretAnnotationKey: VirtApiCertSecretName,
				},
			},
			Spec: apiregv1.APIServiceSpec{
				Service: &apiregv1.ServiceReference{
					Namespace: installNamespace,
					Name:      VirtApiServiceName,
				},
				Group:                version.Group,
				Version:              version.Version,
				GroupPriorityMinimum: 1000,
				VersionPriority:      15,
			},
		})
	}
	return apiservices
}
