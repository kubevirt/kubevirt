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
	"testing"

	"k8s.io/kube-openapi/pkg/validation/spec"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

// The spec tools/openapispec checks in must cover every kind of endpoint the
// aggregated API server serves: storage backed subresources, the cluster level
// ConditionalAPIHandlers and the endpoints served off the plain mux.
func TestBuildSubresourceOpenAPISpecCoversEveryEndpointKind(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}
	if swagger.Paths == nil {
		t.Fatal("BuildSubresourceOpenAPISpec() returned a spec without paths")
	}

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		root := GroupVersionRoot(gv)
		want := []string{
			// rest.Connecter storages
			root + "/namespaces/{namespace}/virtualmachineinstances/{name}/console",
			root + "/namespaces/{namespace}/virtualmachineinstances/{name}/vnc",
			root + "/namespaces/{namespace}/virtualmachineinstances/{name}/sev",
			root + "/namespaces/{namespace}/virtualmachines/{name}/start",
			root + "/namespaces/{namespace}/virtualmachines/{name}/stop",
			root + "/namespaces/{namespace}/virtualmachines/{name}/expand-spec",
			// cluster level ConditionalAPIHandlers
			root + "/version",
			root + "/guestfs",
			root + "/healthz",
			root + "/start-cluster-profiler",
			root + "/namespaces/{namespace}/expand-vm-spec",
		}
		for _, path := range want {
			if _, ok := swagger.Paths.Paths[path]; !ok {
				t.Errorf("generated spec is missing %q", path)
			}
		}
	}

	// endpoints served from the NonGoRestfulMux
	for _, path := range append([]string{HealthzPath}, ComponentProfilerPaths()...) {
		if _, ok := swagger.Paths.Paths[path]; !ok {
			t.Errorf("generated spec is missing mux endpoint %q", path)
		}
	}
}

// Every ClusterLevelRoute has to be documented under every subresource version,
// with the method it is actually served with. This is what keeps the generated
// swagger tied to the ClusterLevelRoutes declaration.
func TestBuildSubresourceOpenAPISpecDocumentsAllClusterLevelRoutes(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		for _, route := range ClusterLevelRoutes() {
			path := route.Path(gv)
			item, ok := swagger.Paths.Paths[path]
			if !ok {
				t.Errorf("generated spec is missing cluster level route %q", path)
				continue
			}
			if route.Method == http.MethodGet && item.Get == nil {
				t.Errorf("%q is documented without a GET operation", path)
			}
			if route.Method == http.MethodPut && item.Put == nil {
				t.Errorf("%q is documented without a PUT operation", path)
			}
		}
	}
}

// The aggregated server drops request body schemas for rest.Connecter
// subresources. Storages that decode a body declare it via
// connectRequestBodyDescriber, and BuildSubresourceOpenAPISpec has to document
// it again so the body type stays in the spec like before the migration.
func TestBuildSubresourceOpenAPISpecDocumentsConnecterRequestBodies(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}

	type bodyExpectation struct {
		suffix  string
		method  string
		defName string
	}
	expectations := []bodyExpectation{
		{"/virtualmachines/{name}/start", http.MethodPut, "v1.StartOptions"},
		{"/virtualmachines/{name}/stop", http.MethodPut, "v1.StopOptions"},
		{"/virtualmachines/{name}/restart", http.MethodPut, "v1.RestartOptions"},
		{"/virtualmachines/{name}/migrate", http.MethodPut, "v1.MigrateOptions"},
		{"/virtualmachines/{name}/evacuate", http.MethodPut, "v1.EvacuateCancelOptions"},
		{"/virtualmachines/{name}/objectgraph", http.MethodGet, "v1.ObjectGraphOptions"},
		{"/virtualmachineinstances/{name}/pause", http.MethodPut, "v1.PauseOptions"},
		{"/virtualmachineinstances/{name}/unpause", http.MethodPut, "v1.UnpauseOptions"},
		{"/virtualmachineinstances/{name}/freeze", http.MethodPut, "v1.FreezeUnfreezeTimeout"},
		{"/virtualmachineinstances/{name}/backup", http.MethodPut, "v1alpha1.BackupOptions"},
		{"/virtualmachineinstances/{name}/redefine-checkpoint", http.MethodPut, "v1alpha1.BackupCheckpoint"},
	}

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		root := GroupVersionRoot(gv) + "/namespaces/{namespace}"
		for _, want := range expectations {
			path := root + want.suffix
			item, ok := swagger.Paths.Paths[path]
			if !ok {
				t.Errorf("generated spec is missing %q", path)
				continue
			}
			operation := item.Put
			if want.method == http.MethodGet {
				operation = item.Get
			}
			if operation == nil {
				t.Errorf("%q has no %s operation", path, want.method)
				continue
			}
			ref := "#/definitions/" + want.defName
			found := false
			for _, param := range operation.Parameters {
				if param.In == "body" && param.Schema != nil && param.Schema.Ref.String() == ref {
					found = true
				}
			}
			if !found {
				t.Errorf("%q %s is missing a body parameter referencing %q", path, want.method, ref)
			}
			if _, ok := swagger.Definitions[want.defName]; !ok {
				t.Errorf("generated spec is missing definition %q referenced by %q", want.defName, path)
			}
		}
	}
}

// Delegated authentication can reject any subresource request, so every
// generated operation has to document a 401 like the pre-migration routes did.
func TestBuildSubresourceOpenAPISpecDocumentsUnauthorized(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}

	for path, item := range swagger.Paths.Paths {
		if _, ok := subresourceSuffix(path); !ok {
			continue
		}
		for _, operation := range []*spec.Operation{item.Get, item.Put, item.Post, item.Delete, item.Patch} {
			if operation == nil {
				continue
			}
			if operation.Responses == nil {
				t.Errorf("%q operation %q has no responses", path, operation.ID)
				continue
			}
			if _, ok := operation.Responses.StatusCodeResponses[http.StatusUnauthorized]; !ok {
				t.Errorf("%q operation %q is missing a 401 response", path, operation.ID)
			}
		}
	}
}

// The restored descriptions and error responses must land on real served
// operations, and every annotation has to describe an endpoint the server
// actually serves (so the table can never document a phantom endpoint).
func TestBuildSubresourceOpenAPISpecRestoresDescriptionsAndErrors(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}

	servedSuffixes := map[string]bool{}
	for path := range swagger.Paths.Paths {
		if suffix, ok := subresourceSuffix(path); ok {
			servedSuffixes[suffix] = true
		}
	}

	for suffix, doc := range subresourceOperationDocs() {
		if !servedSuffixes[suffix] {
			t.Errorf("subresourceOperationDocs documents %q but no such endpoint is served", suffix)
			continue
		}
		for _, gv := range kubevirtv1.SubresourceGroupVersions {
			path := GroupVersionRoot(gv) + suffix
			item, ok := swagger.Paths.Paths[path]
			if !ok {
				continue
			}
			for _, operation := range []*spec.Operation{item.Get, item.Put, item.Post, item.Delete, item.Patch} {
				if operation == nil {
					continue
				}
				if doc.description != "" && operation.Description != doc.description {
					t.Errorf("%q description = %q, want %q", path, operation.Description, doc.description)
				}
				for _, code := range doc.responseCodes {
					if operation.Responses == nil || operation.Responses.StatusCodeResponses == nil {
						t.Errorf("%q is missing a %d response", path, code)
						continue
					}
					if _, ok := operation.Responses.StatusCodeResponses[code]; !ok {
						t.Errorf("%q is missing a %d response", path, code)
					}
				}
			}
		}
	}
}

// Named subresources are reached through the wildcard route the rest.Connecter
// installs, so no operation may be documented twice under one path.
func TestBuildSubresourceOpenAPISpecHasUniqueOperationIDs(t *testing.T) {
	swagger, err := BuildSubresourceOpenAPISpec()
	if err != nil {
		t.Fatalf("BuildSubresourceOpenAPISpec() failed: %v", err)
	}

	seen := map[string]string{}
	for path, item := range swagger.Paths.Paths {
		for _, operation := range []*spec.Operation{item.Get, item.Put, item.Post, item.Delete, item.Patch} {
			if operation == nil || operation.ID == "" {
				continue
			}
			if previous, duplicate := seen[operation.ID]; duplicate {
				t.Errorf("operation ID %q is used by both %q and %q", operation.ID, previous, path)
				continue
			}
			seen[operation.ID] = path
		}
	}
}
