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
	"testing"

	"github.com/spf13/pflag"

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
