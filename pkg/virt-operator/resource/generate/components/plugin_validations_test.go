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

package components

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "kubevirt.io/api/plugin/v1alpha1"
)

func pluginObject(name string, hooks ...v1alpha1.LauncherHook) *v1alpha1.Plugin {
	return &v1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       v1alpha1.PluginSpec{LauncherHooks: hooks},
	}
}

func sidecar(socketPath string) *v1alpha1.SidecarLauncherHook {
	return &v1alpha1.SidecarLauncherHook{SocketPath: socketPath}
}

func celHook(hookPoint v1alpha1.LauncherHookPoint) *v1alpha1.CELLauncherHook {
	return &v1alpha1.CELLauncherHook{HookPoint: hookPoint, Expression: "true"}
}

func unstructuredSelf(obj any) any {
	switch obj := obj.(type) {
	case *v1alpha1.Plugin, *v1alpha1.LauncherHook, *v1alpha1.SidecarLauncherHook, *v1alpha1.NodeHook:
		return unstructuredOf(obj)
	default:
		return obj
	}
}

var _ = Describe("Plugin CRD validation rules", func() {
	var schema *extv1.JSONSchemaProps

	BeforeEach(func() {
		schema = crdSchema(NewPluginCrd, "v1alpha1")
	})

	DescribeTable("should accept", func(path string, self interface{}) {
		Expect(validateCELAt(schema, path, unstructuredSelf(self))).To(Succeed())
	},
		Entry("a sidecar socketPath confined to the plugin's own directory", "",
			pluginObject("my-plugin", v1alpha1.LauncherHook{Sidecar: sidecar("/var/run/kubevirt-plugin/my-plugin/hook.sock")})),
		Entry("a plugin with no launcher hooks at all", "",
			pluginObject("my-plugin")),
		Entry("a launcher hook with only cel", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{CEL: celHook(v1alpha1.LauncherHookGuestDefinition)}),
		Entry("a launcher hook with only sidecar", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{Sidecar: sidecar("/var/run/kubevirt-plugin/p/h.sock")}),
		Entry("a launcher hook with a positive timeout", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{
				CEL:     celHook(v1alpha1.LauncherHookGuestDefinition),
				Timeout: &metav1.Duration{Duration: 30 * time.Second},
			}),
		Entry("a launcher hook with no timeout", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{CEL: celHook(v1alpha1.LauncherHookGuestDefinition)}),
		Entry("a clean sidecar socketPath", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/p/h.sock"}),
		Entry("a node hook with a clean socket and a positive timeout", "spec.nodeHooks[]",
			&v1alpha1.NodeHook{
				Socket:  "/var/run/kubevirt-plugin/p/node.sock",
				Timeout: &metav1.Duration{Duration: time.Minute},
			}),
		Entry("a node hook with no timeout", "spec.nodeHooks[]",
			&v1alpha1.NodeHook{Socket: "/var/run/kubevirt-plugin/p/node.sock"}),

		Entry("launcher hook point GuestDefinition", "spec.launcherHooks[].cel.hookPoint", v1alpha1.LauncherHookGuestDefinition),

		Entry("node hook point PreVMStart", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPreVMStart),
		Entry("node hook point PostVMStart", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPostVMStart),
		Entry("node hook point OnVMStop", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookOnVMStop),
		Entry("node hook point PostVMStop", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPostVMStop),
		Entry("node hook point PreMigrationSource", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPreMigrationSource),
		Entry("node hook point PreMigrationTarget", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPreMigrationTarget),
		Entry("node hook point PostMigrationTarget", "spec.nodeHooks[].permittedHooks[]", v1alpha1.NodeHookPostMigrationTarget),

		Entry("failure strategy Fail on the plugin", "spec.failureStrategy", v1alpha1.FailureStrategyFail),
		Entry("failure strategy Ignore on the plugin", "spec.failureStrategy", v1alpha1.FailureStrategyIgnore),
		Entry("failure strategy Fail on a launcher hook", "spec.launcherHooks[].failureStrategy", v1alpha1.FailureStrategyFail),
		Entry("failure strategy Ignore on a node hook", "spec.nodeHooks[].failureStrategy", v1alpha1.FailureStrategyIgnore),
	)

	DescribeTable("should reject", func(path string, self interface{}, expectedMessage string) {
		Expect(validateCELAt(schema, path, unstructuredSelf(self))).To(MatchError(ContainSubstring(expectedMessage)))
	},
		Entry("a sidecar socketPath under another plugin's directory", "",
			pluginObject("my-plugin", v1alpha1.LauncherHook{Sidecar: sidecar("/var/run/kubevirt-plugin/other-plugin/hook.sock")}),
			"must start with /var/run/kubevirt-plugin/"),
		Entry("a sidecar socketPath outside the plugin socket base directory", "",
			pluginObject("my-plugin", v1alpha1.LauncherHook{Sidecar: sidecar("/tmp/hook.sock")}),
			"must start with /var/run/kubevirt-plugin/"),
		Entry("a sidecar socketPath that only prefix-matches the plugin name", "",
			pluginObject("my-plugin", v1alpha1.LauncherHook{Sidecar: sidecar("/var/run/kubevirt-plugin/my-plugin-evil/hook.sock")}),
			"must start with /var/run/kubevirt-plugin/"),

		Entry("a launcher hook with neither cel nor sidecar", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{}, "exactly one of cel or sidecar"),
		Entry("a launcher hook with both cel and sidecar", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{
				CEL:     celHook(v1alpha1.LauncherHookGuestDefinition),
				Sidecar: sidecar("/var/run/kubevirt-plugin/p/h.sock"),
			}, "exactly one of cel or sidecar"),
		Entry("a launcher hook with a zero timeout", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{
				CEL:     celHook(v1alpha1.LauncherHookGuestDefinition),
				Timeout: &metav1.Duration{},
			}, "timeout must be greater than zero"),
		Entry("a launcher hook with a negative timeout", "spec.launcherHooks[]",
			&v1alpha1.LauncherHook{
				CEL:     celHook(v1alpha1.LauncherHookGuestDefinition),
				Timeout: &metav1.Duration{Duration: -5 * time.Second},
			}, "timeout must be greater than zero"),
		Entry("a node hook with a zero timeout", "spec.nodeHooks[]",
			&v1alpha1.NodeHook{Socket: "/var/run/kubevirt-plugin/p/node.sock", Timeout: &metav1.Duration{}},
			"timeout must be greater than zero"),

		Entry("a sidecar socketPath with a parent-directory segment", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/p/../other/h.sock"}, "clean path"),
		Entry("a sidecar socketPath with two dots in its name", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/p/hook..sock"}, "clean path"),
		Entry("a sidecar socketPath with a repeated separator", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin//p/h.sock"}, "clean path"),
		Entry("a sidecar socketPath with a trailing separator", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/p/"}, "clean path"),
		Entry("a sidecar socketPath without a .sock suffix", "spec.launcherHooks[].sidecar",
			&v1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/p/hook"}, "must end with .sock"),
		Entry("a node hook socket with a parent-directory segment", "spec.nodeHooks[]",
			&v1alpha1.NodeHook{Socket: "/var/run/kubevirt-plugin/p/../x.sock"}, "clean path"),

		Entry("launcher hook point PreBoot is not implemented yet", "spec.launcherHooks[].cel.hookPoint",
			v1alpha1.LauncherHookPoint("PreBoot"), "hook point must be GuestDefinition"),
		Entry("launcher hook point PreMigrationSource is not implemented yet", "spec.launcherHooks[].sidecar.permittedHooks[]",
			v1alpha1.LauncherHookPoint("PreMigrationSource"), "hook point must be GuestDefinition"),
		Entry("a node hook point used by a CEL hook", "spec.launcherHooks[].cel.hookPoint",
			v1alpha1.NodeHookPostVMStart, "hook point must be GuestDefinition"),
		Entry("a node hook point used by a sidecar hook", "spec.launcherHooks[].sidecar.permittedHooks[]",
			v1alpha1.NodeHookPreMigrationTarget, "hook point must be GuestDefinition"),
		Entry("a misspelled launcher hook point", "spec.launcherHooks[].sidecar.permittedHooks[]",
			"guestdefinition", "hook point must be GuestDefinition"),
		Entry("an unknown node hook point", "spec.nodeHooks[].permittedHooks[]",
			v1alpha1.NodeHookPoint("PreBoot"), "hook point must be one of"),
		Entry("an empty node hook point", "spec.nodeHooks[].permittedHooks[]",
			"", "hook point must be one of"),

		Entry("an unknown failure strategy on the plugin", "spec.failureStrategy",
			"Retry", "failureStrategy must be one of"),
		Entry("a lowercase failure strategy on a launcher hook", "spec.launcherHooks[].failureStrategy",
			"fail", "failureStrategy must be one of"),
		Entry("an unknown failure strategy on a node hook", "spec.nodeHooks[].failureStrategy",
			"Ignore ", "failureStrategy must be one of"),
	)
})
