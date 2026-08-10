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

	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime/schema"
	genericapiserver "k8s.io/apiserver/pkg/server"

	kubevirtv1 "kubevirt.io/api/core/v1"
)

// very first smoke test for the bootstrap. It only
// verifies that the package can be wired up without panicking and that the
// expected flags are registered. It does not start a server.
func TestNewDoesNotPanic(t *testing.T) {
	s := New()
	if s == nil {
		t.Fatal("apiserver.New() returned nil")
	}
	if s.secureServingOpts == nil || s.authnOpts == nil || s.authzOpts == nil {
		t.Fatal("apiserver.New() did not initialize the option holders")
	}

	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	s.AddFlags(fs)

	for _, expected := range []string{
		"bind-address",
		"secure-port",
		"client-ca-file",
		"authentication-kubeconfig",
		"authorization-kubeconfig",
	} {
		if fs.Lookup(expected) == nil {
			t.Errorf("expected flag %q to be registered", expected)
		}
	}
}

// This guards the one piece of scheme wiring that is different
// from kubevirt/virt-template: the subresources.kubevirt.io group versions
// must end up in the scheme even though virtclientgoscheme.AddToScheme does not register them.
func TestNewSchemeRegistersSubresourceGroupVersions(t *testing.T) {
	scheme := NewScheme()

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		gvk := gv.WithKind("VirtualMachineInstance")
		if !scheme.Recognizes(gvk) {
			t.Errorf("scheme does not recognize %s; the InstallAPIGroup code path will fail later", gvk)
		}
	}
}

func TestWithAlwaysAllowPathsAccumulates(t *testing.T) {
	s := New().WithAlwaysAllowPaths("/a", "/b").WithAlwaysAllowPaths("/c")
	want := []string{"/a", "/b", "/c"}
	if !slices.Equal(s.alwaysAllowPaths, want) {
		t.Fatalf("alwaysAllowPaths = %v, want %v", s.alwaysAllowPaths, want)
	}
}

func TestGetAdditionalAlwaysAllowPaths(t *testing.T) {
	gv := schema.GroupVersion{Group: kubevirtv1.SubresourceGroupName, Version: "v1"}
	got := getAdditionalAlwaysAllowPaths(APIGroups{gv: nil})

	want := []string{
		genericapiserver.APIGroupPrefix + "/" + gv.Group,
		genericapiserver.APIGroupPrefix + "/" + gv.String(),
	}
	for _, path := range want {
		if !slices.Contains(got, path) {
			t.Errorf("getAdditionalAlwaysAllowPaths missing %q; got %v", path, got)
		}
	}
}

func TestCollectAlwaysAllowPaths(t *testing.T) {
	gv := schema.GroupVersion{Group: kubevirtv1.SubresourceGroupName, Version: "v1"}
	s := New().
		WithMuxHandlers(
			MuxHandler{Path: "/metrics", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})},
			MuxHandler{Path: "/healthz", Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})},
		).
		WithAlwaysAllowPaths(
			"/apis/subresources.kubevirt.io/v1/version",
			"/apis/subresources.kubevirt.io/v1/guestfs",
			"/start-profiler",
		)

	got := s.collectAlwaysAllowPaths(APIGroups{gv: nil})

	mustContain := []string{
		"/",
		genericapiserver.APIGroupPrefix,
		"/openapi/v2",
		"/openapi/v3",
		"/openapi/v3/*",
		genericapiserver.APIGroupPrefix + "/" + gv.Group,
		genericapiserver.APIGroupPrefix + "/" + gv.String(),
		"/metrics",
		"/healthz",
		"/apis/subresources.kubevirt.io/v1/version",
		"/apis/subresources.kubevirt.io/v1/guestfs",
		"/start-profiler",
	}
	for _, path := range mustContain {
		if !slices.Contains(got, path) {
			t.Errorf("collectAlwaysAllowPaths missing %q; got %v", path, got)
		}
	}

	// Named subresources must still go through RBAC (SAR), not AlwaysAllow.
	mustNotContain := []string{
		"/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachines/test/start",
		"/apis/subresources.kubevirt.io/v1/namespaces/default/virtualmachineinstances/test/console",
		"virtualmachines/start",
		"virtualmachineinstances/console",
	}
	for _, path := range mustNotContain {
		if slices.Contains(got, path) {
			t.Errorf("collectAlwaysAllowPaths unexpectedly contains %q", path)
		}
	}
}

func TestClusterLevelAllowPaths(t *testing.T) {
	got := ClusterLevelAllowPaths(kubevirtv1.SubresourceGroupVersions)

	for _, gv := range kubevirtv1.SubresourceGroupVersions {
		base := "/apis/" + gv.Group + "/" + gv.Version + "/"
		for _, suffix := range []string{
			"version",
			"guestfs",
			"healthz",
			"start-cluster-profiler",
			"stop-cluster-profiler",
			"dump-cluster-profiler",
		} {
			path := base + suffix
			if !slices.Contains(got, path) {
				t.Errorf("ClusterLevelAllowPaths missing %q; got %v", path, got)
			}
		}

		// expand-vm-spec is authorized via RBAC, not AlwaysAllow.
		if slices.Contains(got, base+"expand-vm-spec") {
			t.Errorf("ClusterLevelAllowPaths unexpectedly allows expand-vm-spec for %s", gv)
		}
	}
}

func TestComponentProfilerPaths(t *testing.T) {
	got := ComponentProfilerPaths()
	for _, path := range []string{"/start-profiler", "/stop-profiler", "/dump-profiler"} {
		if !slices.Contains(got, path) {
			t.Errorf("ComponentProfilerPaths missing %q; got %v", path, got)
		}
	}
}
