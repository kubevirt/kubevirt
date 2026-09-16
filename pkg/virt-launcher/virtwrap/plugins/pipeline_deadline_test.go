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

	const testReadinessTimeout = 300 * time.Millisecond
	const testSharedReadinessTimeout = 2 * time.Second

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

	AfterEach(func() {
		Expect(os.RemoveAll(pluginSocketBaseDir)).To(Succeed())
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
	// Plugin "a" has a socket that never appears and burns the shared budget. Plugin "b"'s socket
	// appears after that deadline but within a fresh per-hook window. A shared deadline must reject
	// "b" before its socket appears; a per-hook deadline would wait for the socket and fail later
	// while dialing it. The two outcomes are distinguishable by error alone.
	It("should be shared by all sidecar hooks rather than recomputed per hook", func() {
		missingSocket := filepath.Join(pluginSocketBaseDir, "a", "hook.sock")

		lateSocketDir := filepath.Join(pluginSocketBaseDir, "b")
		lateSocket := filepath.Join(lateSocketDir, "hook.sock")
		socketCreated := make(chan error, 1)
		go func() {
			time.Sleep(testSharedReadinessTimeout + 1500*time.Millisecond)
			if err := os.MkdirAll(lateSocketDir, 0755); err != nil {
				socketCreated <- err
				return
			}
			socketCreated <- os.WriteFile(lateSocket, nil, 0600)
		}()

		plugins := []pluginv1alpha1.Plugin{
			sidecarPlugin("a", missingSocket, pluginv1alpha1.FailureStrategyIgnore),
			sidecarPlugin("b", lateSocket, pluginv1alpha1.FailureStrategyFail),
		}

		_, _, err := applyGuestDefinitionHooks(plugins, vmi, spec, pluginv1alpha1.InvocationContextBoot, testSharedReadinessTimeout)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(lateSocket))
		Expect(err.Error()).To(ContainSubstring("not ready after"),
			"the second sidecar hook must inherit the exhausted pipeline deadline, not a fresh window")
		Expect(err.Error()).To(ContainSubstring(testSharedReadinessTimeout.String()))
		Expect(<-socketCreated).To(Succeed())
	})

	It("should let a sidecar hook run when the shared deadline has not been exhausted", func() {
		socketDir := filepath.Join(pluginSocketBaseDir, "b")
		Expect(os.MkdirAll(socketDir, 0755)).To(Succeed())
		socketPath := filepath.Join(socketDir, "hook.sock")
		Expect(os.WriteFile(socketPath, nil, 0600)).To(Succeed())

		plugins := []pluginv1alpha1.Plugin{sidecarPlugin("b", socketPath, pluginv1alpha1.FailureStrategyFail)}

		_, _, err := applyGuestDefinitionHooks(plugins, vmi, spec, pluginv1alpha1.InvocationContextBoot, testReadinessTimeout)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).NotTo(ContainSubstring("not ready after"),
			"readiness must succeed here, so the failure has to come from the call itself")
	})

	It("should fail promptly when a sidecar hook with FailureStrategy Fail is unreachable", func() {
		socketPath := filepath.Join(pluginSocketBaseDir, "sidecar-plugin", "hook.sock")
		plugin := sidecarPlugin("sidecar-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)

		_, _, err := applyGuestDefinitionHooks(
			[]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot, testReadinessTimeout,
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("not ready after"))
	})

	It("should ignore an unreachable sidecar hook when FailureStrategy is Ignore", func() {
		socketPath := filepath.Join(pluginSocketBaseDir, "sidecar-plugin", "hook.sock")
		plugin := sidecarPlugin("sidecar-plugin", socketPath, pluginv1alpha1.FailureStrategyIgnore)

		result, xmlStr, err := applyGuestDefinitionHooks(
			[]pluginv1alpha1.Plugin{plugin}, vmi, spec, pluginv1alpha1.InvocationContextBoot, testReadinessTimeout,
		)

		Expect(err).NotTo(HaveOccurred())
		Expect(xmlStr).NotTo(BeEmpty())
		Expect(result).NotTo(BeNil())
	})
})
