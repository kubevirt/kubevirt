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

package apiserver

import (
	"net/http"
	"slices"
	"testing"

	"k8s.io/apiserver/pkg/endpoints/request"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

func TestClusterLevelRoutesAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, route := range ClusterLevelRoutes() {
		if seen[route.Resource] {
			t.Errorf("resource %q is declared more than once", route.Resource)
		}
		seen[route.Resource] = true

		// Only GET and PUT can be mapped onto an OpenAPI operation verb.
		if route.Method != http.MethodGet && route.Method != http.MethodPut {
			t.Errorf("route %q uses method %q, which openapispec cannot map", route.Resource, route.Method)
		}
		if route.Doc == "" {
			t.Errorf("route %q has no Doc, so it would be documented without a summary", route.Resource)
		}
		if _, err := clusterLevelOperationID(route); err != nil {
			t.Errorf("route %q has no usable operation ID: %v", route.Resource, err)
		}
	}
}

// expand-vm-spec carries a VirtualMachine of the caller's choosing and must stay
// behind RBAC. Everything else on this list is public discovery or health data.
func TestClusterLevelRoutesOnlyExemptPublicEndpointsFromRBAC(t *testing.T) {
	for _, route := range ClusterLevelRoutes() {
		if route.Resource == "expand-vm-spec" && route.AlwaysAllow {
			t.Error("expand-vm-spec must be authorized through RBAC, not AlwaysAllow")
		}
		if route.Namespaced && route.AlwaysAllow {
			t.Errorf("namespaced route %q must not bypass RBAC", route.Resource)
		}
	}
}

func TestClusterLevelRoutePath(t *testing.T) {
	gv := kubevirtv1.SubresourceGroupVersions[0]

	clusterScoped := ClusterLevelRoute{Resource: "version"}
	if got, want := clusterScoped.Path(gv), GroupVersionRoot(gv)+"/version"; got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}

	namespaced := ClusterLevelRoute{Resource: "expand-vm-spec", Namespaced: true}
	if got, want := namespaced.Path(gv), GroupVersionRoot(gv)+"/namespaces/{namespace}/expand-vm-spec"; got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestClusterLevelRouteMatches(t *testing.T) {
	route := ClusterLevelRoute{Resource: "guestfs"}
	matches := route.Matches(kubevirtv1.SubresourceGroupName)

	if !matches(&request.RequestInfo{IsResourceRequest: true, APIGroup: kubevirtv1.SubresourceGroupName, Resource: "guestfs"}) {
		t.Error("predicate rejected a guestfs request of the subresources group")
	}
	// A different resource, a different group or a non resource request must not
	// be swallowed by the handler.
	if matches(&request.RequestInfo{IsResourceRequest: true, APIGroup: kubevirtv1.SubresourceGroupName, Resource: "version"}) {
		t.Error("predicate accepted a request for another resource")
	}
	if matches(&request.RequestInfo{IsResourceRequest: true, APIGroup: "kubevirt.io", Resource: "guestfs"}) {
		t.Error("predicate accepted a request for another API group")
	}
	if matches(&request.RequestInfo{IsResourceRequest: false, APIGroup: kubevirtv1.SubresourceGroupName, Resource: "guestfs"}) {
		t.Error("predicate accepted a non resource request")
	}
}

// ClusterLevelAllowPaths is derived from ClusterLevelRoutes, so the two must
// stay consistent for every served version.
func TestClusterLevelAllowPathsMatchesRoutes(t *testing.T) {
	got := ClusterLevelAllowPaths(kubevirtv1.SubresourceGroupVersions)

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		for _, route := range ClusterLevelRoutes() {
			path := route.Path(gv)
			if route.AlwaysAllow != slices.Contains(got, path) {
				t.Errorf("route %q has AlwaysAllow=%v but ClusterLevelAllowPaths contains(%q)=%v",
					route.Resource, route.AlwaysAllow, path, slices.Contains(got, path))
			}
		}
	}
}
