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
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/authorization/authorizer"
)

// ClusterLevelAllowPaths returns the request paths of the cluster level routes
// declared AlwaysAllow.
//
// These end up in the authorizer's AlwaysAllowPaths, but note that the upstream
// authorizer backing that option
// (k8s.io/apiserver/pkg/authorization/path.NewAuthorizer) deliberately returns
// NoOpinion for resource requests. Every path returned here is served under
// /apis/<group>/<version>/<resource> and is therefore parsed as a resource
// request, so AlwaysAllowPaths alone does not exempt them. The exemption is
// enforced by clusterLevelAlwaysAllowAuthorizer, which the API server unions in
// front of the delegated authorizer.
func ClusterLevelAllowPaths(groupVersions []schema.GroupVersion) []string {
	var paths []string
	for _, gv := range groupVersions {
		for _, route := range ClusterLevelRoutes() {
			if !route.AlwaysAllow {
				continue
			}
			paths = append(paths, route.Path(gv))
		}
	}
	return paths
}

// HealthzPath is the process level health endpoint of virt-api, served from the
// GenericAPIServer's NonGoRestfulMux
const HealthzPath = "/healthz"

func ComponentProfilerPaths() []string {
	return []string{
		"/start-profiler",
		"/stop-profiler",
		"/dump-profiler",
	}
}

// clusterLevelAlwaysAllowAuthorizer allows the collection endpoints of the
// cluster level routes declared AlwaysAllow, e.g. version, guestfs and the
// cluster profiler endpoints.
//
// Before virt-api moved to the aggregated API server, its own authorizer
// exempted these paths from the SubjectAccessReview by exact URL match. The
// AlwaysAllowPaths option cannot express that, because it ignores resource
// requests (see ClusterLevelAllowPaths), so this authorizer restores the
// exemption. It only ever returns Allow or NoOpinion, so it can widen but never
// narrow what the delegated authorizer permits.
func clusterLevelAlwaysAllowAuthorizer(apiGroups APIGroups) authorizer.Authorizer {
	groups := map[string]struct{}{}
	for gv := range apiGroups {
		groups[gv.Group] = struct{}{}
	}

	// resource -> whether the route is served under /namespaces/{namespace}
	allowed := map[string]bool{}
	for _, route := range ClusterLevelRoutes() {
		if route.AlwaysAllow {
			allowed[route.Resource] = route.Namespaced
		}
	}

	return authorizer.AuthorizerFunc(func(_ context.Context, a authorizer.Attributes) (authorizer.Decision, string, error) {
		if !a.IsResourceRequest() {
			return authorizer.DecisionNoOpinion, "", nil
		}
		if _, ok := groups[a.GetAPIGroup()]; !ok {
			return authorizer.DecisionNoOpinion, "", nil
		}
		namespaced, ok := allowed[a.GetResource()]
		if !ok {
			return authorizer.DecisionNoOpinion, "", nil
		}
		// The declared routes are collection endpoints, so anything addressing a
		// concrete object or a subresource below them is not one of them.
		if a.GetName() != "" || a.GetSubresource() != "" {
			return authorizer.DecisionNoOpinion, "", nil
		}
		if namespaced != (a.GetNamespace() != "") {
			return authorizer.DecisionNoOpinion, "", nil
		}
		return authorizer.DecisionAllow, "", nil
	})
}
