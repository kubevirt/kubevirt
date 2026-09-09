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

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/endpoints/request"
)

// This is the only place where such an endpoint is declared. virt-api derives
// the ConditionalAPIHandler predicates and the authorizer's AlwaysAllowPaths
// from it, and tools/openapispec derives their OpenAPI routes from it, so the
// generated swagger cannot drift from what the API server actually serves.
type ClusterLevelRoute struct {
	Resource    string
	Method      string
	Namespaced  bool
	AlwaysAllow bool
	Doc         string
}

// ClusterLevelRoutes returns every cluster level ConditionalAPIHandler endpoint
// of the subresources.kubevirt.io group.
func ClusterLevelRoutes() []ClusterLevelRoute {
	return []ClusterLevelRoute{
		{
			Resource:   "expand-vm-spec",
			Method:     http.MethodPut,
			Namespaced: true,
			Doc:        "Expands instancetype and preference into the passed VirtualMachine object.",
		},
		{
			Resource:    "version",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Returns the version of the KubeVirt API server.",
		},
		{
			Resource:    "guestfs",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Returns the guestfs image tag and registry used by virtctl guestfs.",
		},
		{
			Resource:    "healthz",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Health endpoint reporting whether virt-api can reach the cluster.",
		},
		{
			Resource:    "start-cluster-profiler",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Starts the CPU profiler on every KubeVirt component.",
		},
		{
			Resource:    "stop-cluster-profiler",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Stops the CPU profiler on every KubeVirt component.",
		},
		{
			Resource:    "dump-cluster-profiler",
			Method:      http.MethodGet,
			AlwaysAllow: true,
			Doc:         "Collects the profiler results of every KubeVirt component.",
		},
	}
}

// SubPath returns the route path relative to the group version root.
func (r ClusterLevelRoute) SubPath() string {
	if r.Namespaced {
		return "/namespaces/{namespace}/" + r.Resource
	}
	return "/" + r.Resource
}

// Path returns the full request path of the route for the given group version.
func (r ClusterLevelRoute) Path(gv schema.GroupVersion) string {
	return GroupVersionRoot(gv) + r.SubPath()
}

// Matches is the ConditionalAPIHandler predicate selecting requests for the
// route. Method is deliberately not matched so unsupported methods still reach
// the handler and get its own error instead of a generic 404.
func (r ClusterLevelRoute) Matches(group string) func(*request.RequestInfo) bool {
	return func(info *request.RequestInfo) bool {
		return info.IsResourceRequest &&
			info.APIGroup == group &&
			info.Resource == r.Resource
	}
}

// GroupVersionRoot returns the path the GenericAPIServer serves a group version
// under.
func GroupVersionRoot(gv schema.GroupVersion) string {
	return "/apis/" + gv.Group + "/" + gv.Version
}
