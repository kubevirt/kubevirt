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
 * Copyright 2022 Red Hat, Inc.
 *
 */
package components

import (
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetAllRoutes(namespace string) []*routev1.Route {
	return []*routev1.Route{
		NewExportProxyRoute(namespace),
	}
}

func newBlankRoute() *routev1.Route {
	return &routev1.Route{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "route.openshift.io/v1",
			Kind:       "Route",
		},
	}
}

func NewExportProxyRoute(namespace string) *routev1.Route {
	route := newBlankRoute()
	route.Namespace = namespace
	route.Name = VirtExportProxyName

	route.Spec.To.Kind = "Service"
	route.Spec.To.Name = VirtExportProxyName
	route.Spec.TLS = &routev1.TLSConfig{
		Termination: routev1.TLSTerminationReencrypt,
	}

	return route
}
