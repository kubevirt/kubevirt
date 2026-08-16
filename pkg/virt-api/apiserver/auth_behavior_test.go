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
	"net/http"
	"net/http/httptest"
	"testing"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/user"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/authorization/path"
	"k8s.io/apiserver/pkg/authorization/union"
	"k8s.io/apiserver/pkg/endpoints/filters"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

// These tests exercise the real authentication and authorization handler chain
// that the GenericAPIServer wires around virt-api (WithRequestInfo ->
// WithAuthentication -> WithAuthorization) using the real AlwaysAllowPaths set
// that collectAlwaysAllowPaths derives from the same declarations api.go feeds
// the server. Only the delegated authenticator/authorizer (which would normally
// talk to the cluster via TokenReview/SAR) are replaced with in-memory fakes,
// so we can assert the wiring behaves as intended without a running cluster.

type spyHandler struct {
	called bool
}

func (s *spyHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	s.called = true
	w.WriteHeader(http.StatusOK)
}

// Reproduces the AlwaysAllowPaths virt-api installs in api.go
func testAlwaysAllowPaths() []string {
	s := New().
		WithAlwaysAllowPaths(ClusterLevelAllowPaths(kubevirtv1.SubresourceGroupVersions)...).
		WithAlwaysAllowPaths(ComponentProfilerPaths()...).
		WithMuxHandlers(
			MuxHandler{Path: "/metrics"},
			MuxHandler{Path: HealthzPath},
		)

	return s.collectAlwaysAllowPaths(testAPIGroups())
}

func testAPIGroups() APIGroups {
	apiGroups := APIGroups{}
	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		apiGroups[gv] = map[string]rest.Storage{}
	}
	return apiGroups
}

// serverAuthorizer mirrors the authorizer virt-api ends up with: the
// AlwaysAllowPaths authorizer and the cluster level exemption sit in front of
// the delegated (RBAC) authorizer in a union.
func serverAuthorizer(t *testing.T, delegate authorizer.Authorizer) authorizer.Authorizer {
	t.Helper()
	pathAuthorizer, err := path.NewAuthorizer(testAlwaysAllowPaths())
	if err != nil {
		t.Fatalf("path.NewAuthorizer failed: %v", err)
	}
	return union.New(
		pathAuthorizer,
		clusterLevelAlwaysAllowAuthorizer(testAPIGroups()),
		delegate,
	)
}

// buildAuthChain assembles the same ordered filters the GenericAPIServer applies
// around the handler: request info resolution, authentication, authorization.
func buildAuthChain(authn authenticator.Request, authz authorizer.Authorizer, backend http.Handler) http.Handler {
	codecs := clientgoscheme.Codecs
	resolver := &request.RequestInfoFactory{
		APIPrefixes:          sets.NewString("api", "apis"),
		GrouplessAPIPrefixes: sets.NewString("api"),
	}

	h := filters.WithAuthorization(backend, authz, codecs)
	h = filters.WithAuthentication(h, authn, filters.Unauthorized(codecs), nil, nil)
	h = filters.WithRequestInfo(h, resolver)
	return h
}

func authnAsUser(name string, groups ...string) authenticator.Request {
	return authenticator.RequestFunc(func(_ *http.Request) (*authenticator.Response, bool, error) {
		return &authenticator.Response{User: &user.DefaultInfo{Name: name, Groups: groups}}, true, nil
	})
}

func authnReject() authenticator.Request {
	return authenticator.RequestFunc(func(_ *http.Request) (*authenticator.Response, bool, error) {
		return nil, false, nil
	})
}

var allowAllDelegate = authorizer.AuthorizerFunc(
	func(context.Context, authorizer.Attributes) (authorizer.Decision, string, error) {
		return authorizer.DecisionAllow, "", nil
	})

var denyAllDelegate = authorizer.AuthorizerFunc(
	func(context.Context, authorizer.Attributes) (authorizer.Decision, string, error) {
		return authorizer.DecisionDeny, "test: RBAC denied", nil
	})

func doRequest(handler http.Handler, method, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(method, target, http.NoBody))
	return rec
}

const startSubresource = "/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachines/testvm/start"

// A request that cannot be authenticated must be rejected with 401 before any
// authorization decision or backend handler runs, regardless of the path.
func TestAuthUnauthenticatedRequestsGet401(t *testing.T) {
	paths := []string{
		startSubresource,
		"/apis/subresources.kubevirt.io/v1/version",
		"/healthz",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			backend := &spyHandler{}
			chain := buildAuthChain(authnReject(), serverAuthorizer(t, allowAllDelegate), backend)

			rec := doRequest(chain, http.MethodGet, p)

			if rec.Code != http.StatusUnauthorized {
				t.Errorf("got status %d, want %d", rec.Code, http.StatusUnauthorized)
			}
			if backend.called {
				t.Error("backend must not be reached for an unauthenticated request")
			}
		})
	}
}

// The delegated authenticator runs with anonymous authentication enabled, so a
// request without credentials arrives as system:anonymous rather than a 401.
// That is what makes the always-allow routes reachable for unauthenticated
// clients, the way they were before the move to the aggregated API server.
func TestAuthAnonymousUsersReachAlwaysAllowRoutes(t *testing.T) {
	anonymous := authnAsUser(user.Anonymous, user.AllUnauthenticated)

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		p := GroupVersionRoot(gv) + "/version"
		t.Run(p, func(t *testing.T) {
			backend := &spyHandler{}
			chain := buildAuthChain(anonymous, serverAuthorizer(t, denyAllDelegate), backend)

			rec := doRequest(chain, http.MethodGet, p)

			if !backend.called {
				t.Errorf("anonymous request to %q must be served (got status %d)", p, rec.Code)
			}
		})
	}

	t.Run("but not protected subresources", func(t *testing.T) {
		backend := &spyHandler{}
		chain := buildAuthChain(anonymous, serverAuthorizer(t, denyAllDelegate), backend)

		rec := doRequest(chain, http.MethodPut, startSubresource)

		if rec.Code != http.StatusForbidden {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusForbidden)
		}
		if backend.called {
			t.Error("anonymous request must not reach a protected subresource")
		}
	})
}

// Named subresources (start, console, ...) are not in AlwaysAllowPaths and are
// parsed as resource requests, so they must be authorized by the delegated
// authorizer: denied means 403, allowed means the backend is reached.
func TestAuthNamedSubresourcesAreAuthorizedByRBAC(t *testing.T) {
	t.Run("denied -> 403", func(t *testing.T) {
		backend := &spyHandler{}
		chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, denyAllDelegate), backend)

		rec := doRequest(chain, http.MethodPut, startSubresource)

		if rec.Code != http.StatusForbidden {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusForbidden)
		}
		if backend.called {
			t.Error("backend must not be reached when RBAC denies the request")
		}
	})

	t.Run("allowed -> backend reached", func(t *testing.T) {
		backend := &spyHandler{}
		chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, allowAllDelegate), backend)

		rec := doRequest(chain, http.MethodPut, startSubresource)

		if rec.Code != http.StatusOK {
			t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
		}
		if !backend.called {
			t.Error("backend must be reached when RBAC allows the request")
		}
	})
}

// Non-resource AlwaysAllowPaths (discovery, openapi, the plain-mux health,
// metrics and profiler endpoints) must bypass authorization: the backend is
// reached even though the delegated authorizer denies everything.
func TestAuthNonResourceAlwaysAllowPathsBypassAuthorization(t *testing.T) {
	paths := []string{
		"/",
		"/apis",
		"/openapi/v2",
		"/openapi/v3",
		"/healthz",
		"/metrics",
		"/start-profiler",
		"/stop-profiler",
		"/dump-profiler",
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			backend := &spyHandler{}
			chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, denyAllDelegate), backend)

			rec := doRequest(chain, http.MethodGet, p)

			if !backend.called {
				t.Errorf("backend must be reached for always-allow path %q (got status %d)", p, rec.Code)
			}
			if rec.Code != http.StatusOK {
				t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
			}
		})
	}
}

// The cluster level routes declared AlwaysAllow (version, guestfs, healthz,
// *-cluster-profiler) were exempt from the SubjectAccessReview before virt-api
// moved to the aggregated API server, and must stay exempt. They are parsed as
// resource requests, so AlwaysAllowPaths cannot cover them and
// clusterLevelAlwaysAllowAuthorizer has to: the backend is reached even though
// the delegated authorizer denies everything.
func TestAuthAlwaysAllowClusterRoutesBypassAuthorization(t *testing.T) {
	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		for _, route := range ClusterLevelRoutes() {
			if !route.AlwaysAllow {
				continue
			}
			p := route.Path(gv)
			t.Run(p, func(t *testing.T) {
				backend := &spyHandler{}
				chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, denyAllDelegate), backend)

				rec := doRequest(chain, route.Method, p)

				if !backend.called {
					t.Errorf("backend must be reached for always-allow route %q (got status %d)", p, rec.Code)
				}
				if rec.Code != http.StatusOK {
					t.Errorf("got status %d, want %d", rec.Code, http.StatusOK)
				}
			})
		}
	}
}

// The exemption must be limited to the declared routes: cluster level routes
// that are not AlwaysAllow (expand-vm-spec) keep requiring RBAC.
func TestAuthClusterRoutesWithoutAlwaysAllowRequireAuthorization(t *testing.T) {
	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		for _, route := range ClusterLevelRoutes() {
			if route.AlwaysAllow {
				continue
			}
			p := GroupVersionRoot(gv) + "/namespaces/default/" + route.Resource
			t.Run(p, func(t *testing.T) {
				backend := &spyHandler{}
				chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, denyAllDelegate), backend)

				rec := doRequest(chain, route.Method, p)

				if rec.Code != http.StatusForbidden {
					t.Errorf("got status %d, want %d", rec.Code, http.StatusForbidden)
				}
				if backend.called {
					t.Errorf("backend must not be reached for %q when RBAC denies", p)
				}
			})
		}
	}
}

// The exemption keys off the resource of a cluster scoped collection request, so
// it must not leak to requests that merely look similar: a namespaced variant, a
// named object or a subresource below an exempt resource, or the same resource
// name in a foreign API group.
func TestAuthAlwaysAllowExemptionDoesNotLeak(t *testing.T) {
	paths := []string{
		"/apis/subresources.kubevirt.io/v1/namespaces/default/version",
		"/apis/subresources.kubevirt.io/v1/version/extra",
		"/apis/subresources.kubevirt.io/v1/guestfs/somename",
		"/apis/kubevirt.io/v1/version",
		startSubresource,
	}
	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			backend := &spyHandler{}
			chain := buildAuthChain(authnAsUser("alice"), serverAuthorizer(t, denyAllDelegate), backend)

			rec := doRequest(chain, http.MethodGet, p)

			if rec.Code != http.StatusForbidden {
				t.Errorf("got status %d, want %d", rec.Code, http.StatusForbidden)
			}
			if backend.called {
				t.Errorf("backend must not be reached for %q", p)
			}
		})
	}
}
