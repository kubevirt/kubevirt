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

package plugins_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/plugins"
)

var _ = Describe("NodeHookManager", func() {
	var (
		pluginStore cache.Store
		vmi         *virtv1.VirtualMachineInstance
	)

	BeforeEach(func() {
		pluginStore = cache.NewStore(cache.MetaNamespaceKeyFunc)
		vmi = &virtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
			},
			Status: virtv1.VirtualMachineInstanceStatus{
				NodeName: "test-node",
			},
		}
	})

	Context("with feature gate disabled", func() {
		It("should return nil without processing", func() {
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{})
			manager, err := plugins.NewNodeHookManager(pluginStore, config)
			Expect(err).ToNot(HaveOccurred())

			Expect(pluginStore.Add(newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart))).To(Succeed())
			Expect(manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")).To(Succeed())
		})
	})

	Context("with feature gate enabled", func() {
		var manager plugins.NodeHookExecutor

		BeforeEach(func() {
			config, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(&virtv1.KubeVirtConfiguration{
				DeveloperConfiguration: &virtv1.DeveloperConfiguration{
					FeatureGates: []string{"Plugins"},
				},
			})
			var err error
			manager, err = plugins.NewNodeHookManager(pluginStore, config)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should return nil when no plugins exist", func() {
			Expect(manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")).To(Succeed())
		})

		It("should skip plugins without matching hook point", func() {
			Expect(pluginStore.Add(newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStop))).To(Succeed())
			// CallNodeHooks for PreVMStart should not attempt to dial a socket for a
			// plugin that only permits PreVMStop. Since no socket exists, a dial
			// attempt would fail - success here proves the plugin was skipped.
			Expect(manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")).To(Succeed())
		})

		It("should attempt to call matching plugins and fail on missing socket", func() {
			Expect(pluginStore.Add(newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart))).To(Succeed())
			err := manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to dial plugin"))
		})

		It("should ignore errors when FailureStrategy is Ignore", func() {
			plugin := newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart)
			plugin.Spec.NodeHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyIgnore
			Expect(pluginStore.Add(plugin)).To(Succeed())

			Expect(manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")).To(Succeed())
		})

		It("should fail on error when FailureStrategy is Fail", func() {
			plugin := newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart)
			plugin.Spec.NodeHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyFail
			Expect(pluginStore.Add(plugin)).To(Succeed())

			err := manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")
			Expect(err).To(HaveOccurred())
		})

		It("should skip plugin when plugin-level condition does not match", func() {
			plugin := newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart)
			plugin.Spec.Condition = "vmi.metadata.name == 'nonexistent'"
			Expect(pluginStore.Add(plugin)).To(Succeed())

			Expect(manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")).To(Succeed())
		})

		It("should attempt to call plugin when plugin-level condition matches", func() {
			plugin := newPlugin("test-plugin", pluginv1alpha1.NodeHookPreVMStart)
			plugin.Spec.Condition = "vmi.metadata.name == 'test-vmi'"
			Expect(pluginStore.Add(plugin)).To(Succeed())

			err := manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to dial plugin"))
		})

		It("should process plugins in alphabetical order", func() {
			pluginC := newPlugin("charlie", pluginv1alpha1.NodeHookPreVMStart)
			pluginC.Spec.NodeHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyFail
			pluginA := newPlugin("alpha", pluginv1alpha1.NodeHookPreVMStart)
			pluginA.Spec.NodeHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyFail
			pluginB := newPlugin("bravo", pluginv1alpha1.NodeHookPreVMStart)
			pluginB.Spec.NodeHooks[0].FailureStrategy = pluginv1alpha1.FailureStrategyFail

			Expect(pluginStore.Add(pluginC)).To(Succeed())
			Expect(pluginStore.Add(pluginA)).To(Succeed())
			Expect(pluginStore.Add(pluginB)).To(Succeed())

			// With FailureStrategy=Fail, the first plugin (alphabetically "alpha") will
			// be dialed first and fail. The error should reference "alpha".
			err := manager.CallNodeHooks(pluginv1alpha1.NodeHookPreVMStart, vmi, "test-node")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("alpha"))
		})
	})
})

func newPlugin(name string, hookPoints ...pluginv1alpha1.NodeHookPoint) *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: pluginv1alpha1.PluginSpec{
			NodeHooks: []pluginv1alpha1.NodeHook{
				{
					Socket:         "/var/run/kubevirt/plugins/" + name + ".sock",
					PermittedHooks: hookPoints,
				},
			},
		},
	}
}
