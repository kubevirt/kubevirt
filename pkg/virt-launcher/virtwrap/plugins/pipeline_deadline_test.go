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

package plugins

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	virtwrapApi "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("Sidecar readiness deadline", func() {

	const testReadinessTimeout = 30 * time.Millisecond
	const testSharedReadinessTimeout = 40 * time.Millisecond

	var (
		vmi  *v1.VirtualMachineInstance
		spec *virtwrapApi.DomainSpec
	)

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"},
		}
		spec = &virtwrapApi.DomainSpec{}
		spec.Type = "kvm"
		spec.Name = "test-vm"
	})

	sidecarPlugin := func(name, socketPath string, failureStrategy pluginv1alpha1.FailureStrategy) pluginv1alpha1.Plugin {
		return pluginv1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: pluginv1alpha1.PluginSpec{
				LauncherHooks: []pluginv1alpha1.LauncherHook{
					{
						Sidecar: &pluginv1alpha1.SidecarLauncherHook{
							SocketPath:     socketPath,
							PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookGuestDefinition},
						},
						FailureStrategy: failureStrategy,
					},
				},
			},
		}
	}

	// All sidecars are containers of the same virt-launcher pod and start at roughly the same
	// time, so the readiness budget belongs to the pipeline, not to each hook.
	//
	// Plugin "a" never becomes ready and burns the shared budget. Plugin "dev" reports not-ready on
	// its first probe, then ready on its next. With a shared deadline it reaches the exhausted
	// budget check after one probe and skips polling. A per-hook deadline would poll, accept
	// /dev/null as ready, and fail later while dialing it.
	It("should be shared by all sidecar hooks rather than recomputed per hook", func() {
		missingSocket := filepath.Join(pluginSocketBaseDir, "a", "hook.sock")
		readySocket := "/dev/null"
		readySocketProbes := 0
		socketReady := func(path string) (bool, error) {
			if path == readySocket {
				readySocketProbes++
				return readySocketProbes > 1, nil
			}
			return false, nil
		}

		plugins := []pluginv1alpha1.Plugin{
			sidecarPlugin("a", missingSocket, pluginv1alpha1.FailureStrategyIgnore),
			sidecarPlugin("dev", readySocket, pluginv1alpha1.FailureStrategyFail),
		}

		_, _, err := applyGuestDefinitionHooks(plugins, vmi, spec, pluginv1alpha1.InvocationContextBoot, testSharedReadinessTimeout, "/", socketReady)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(readySocket))
		Expect(err.Error()).To(ContainSubstring("not ready after"),
			"the second sidecar hook must inherit the exhausted pipeline deadline, not a fresh window")
		Expect(err.Error()).To(ContainSubstring(testSharedReadinessTimeout.String()))
		Expect(readySocketProbes).To(Equal(1), "the exhausted shared budget must skip polling the second hook")
	})

	It("should check readiness with os.Stat before calling a ready sidecar", func() {
		socketBaseDir := GinkgoT().TempDir()
		socketDir := filepath.Join(socketBaseDir, "b")
		Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
		socketPath := filepath.Join(socketDir, "hook.sock")
		Expect(os.WriteFile(socketPath, nil, 0600)).To(Succeed())

		plugins := []pluginv1alpha1.Plugin{sidecarPlugin("b", socketPath, pluginv1alpha1.FailureStrategyFail)}

		_, _, err := applyGuestDefinitionHooks(plugins, vmi, spec, pluginv1alpha1.InvocationContextBoot, testReadinessTimeout, socketBaseDir, osStatSocketReady)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).NotTo(ContainSubstring("not ready after"),
			"readiness must succeed here, so the failure has to come from the call itself")
		Expect(err.Error()).To(ContainSubstring("dialing sidecar socket"),
			"the injected temp base directory must pass socket path validation and reach the dial")
	})

	It("should fail promptly when a sidecar hook with FailureStrategy Fail is unreachable", func() {
		socketPath := filepath.Join(pluginSocketBaseDir, "sidecar-plugin", "hook.sock")
		plugin := sidecarPlugin("sidecar-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)
		socketReady := func(string) (bool, error) { return false, nil }

		_, _, err := applyGuestDefinitionHooks(
			[]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot, 0, pluginSocketBaseDir, socketReady,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not ready after"))
	})

	It("should ignore an unreachable sidecar hook when FailureStrategy is Ignore", func() {
		socketPath := filepath.Join(pluginSocketBaseDir, "sidecar-plugin", "hook.sock")
		plugin := sidecarPlugin("sidecar-plugin", socketPath, pluginv1alpha1.FailureStrategyIgnore)
		socketReady := func(string) (bool, error) { return false, nil }

		result, xmlStr, err := applyGuestDefinitionHooks(
			[]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot, 0, pluginSocketBaseDir, socketReady,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(xmlStr).NotTo(BeEmpty())
		Expect(result).NotTo(BeNil())
	})
})
