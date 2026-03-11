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

package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe("[sig-compute]Plugin domain hooks", Serial, decorators.SigCompute, func() {

	BeforeEach(func() {
		config.EnableFeatureGate(featuregate.PluginsGate)
	})

	createPlugin := func(plugin *pluginv1alpha1.Plugin) {
		pluginClient := kubevirt.Client().GeneratedKubeVirtClient().PluginV1alpha1().Plugins()
		By("Creating Plugin " + plugin.Name)
		_, err := pluginClient.Create(context.Background(), plugin, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(func() {
			_ = pluginClient.Delete(context.Background(), plugin.Name, metav1.DeleteOptions{})
		})
	}

	createCelPlugin := func(name, condition string, expressions ...string) {
		var hooks []pluginv1alpha1.DomainHook
		for _, expr := range expressions {
			hooks = append(hooks, pluginv1alpha1.DomainHook{
				Condition: condition,
				CEL:       &pluginv1alpha1.CELDomainHook{Expression: expr},
			})
		}
		createPlugin(&pluginv1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec:       pluginv1alpha1.PluginSpec{DomainHooks: hooks},
		})
	}

	expectWatchdog := func(vmi *v1.VirtualMachineInstance) {
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
		Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "ls /dev/watchdog && echo watchdog-found || echo watchdog-missing\n"},
			&expect.BExp{R: "watchdog-found"},
		}, 30)).To(Succeed())
	}

	It("should apply a CEL domain hook that adds a watchdog device visible inside the guest", func() {
		createCelPlugin("watchdog-plugin", "", `Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "none"}]}}`)

		By("Starting an Alpine VMI without a watchdog in its spec")
		vmi := libvmifact.NewAlpine()
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the watchdog device is present inside the guest")
		expectWatchdog(vmi)
	})

	It("should preserve existing VMI spec devices when plugin modifies a different domain field", func() {
		createCelPlugin("soundcard-plugin", "", `Domain{Devices: DomainDeviceList{Sounds: [DomainSound{Model: "ich6"}]}}`)

		By("Starting an Alpine VMI with an explicit watchdog in its spec")
		vmi := libvmifact.NewAlpine(libvmi.WithWatchdog(v1.WatchdogActionReset, libnode.GetArch()))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the original watchdog device survived the plugin merge")
		expectWatchdog(vmi)

		By("Verifying the plugin-injected sound card is present in the domain XML")
		domainSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
		Expect(err).NotTo(HaveOccurred())
		Expect(domainSpec.Devices.SoundCards).NotTo(BeEmpty())
		Expect(domainSpec.Devices.SoundCards[0].Model).To(Equal("ich6"))
	})

	It("should compose devices from multiple separate plugins", func() {
		createCelPlugin("plugin-a-watchdog", "", `Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "none"}]}}`)
		createCelPlugin("plugin-b-soundcard", "", `Domain{Devices: DomainDeviceList{Sounds: [DomainSound{Model: "ich6"}]}}`)

		By("Starting an Alpine VMI")
		vmi := libvmifact.NewAlpine()
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the watchdog from plugin-a is present")
		expectWatchdog(vmi)

		By("Verifying the sound card from plugin-b is present in the domain XML")
		domainSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
		Expect(err).NotTo(HaveOccurred())
		Expect(domainSpec.Devices.SoundCards).NotTo(BeEmpty())
		Expect(domainSpec.Devices.SoundCards[0].Model).To(Equal("ich6"))
	})

	It("should apply multiple hooks within a single plugin", func() {
		createCelPlugin("multi-hook-plugin", "",
			`Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "none"}]}}`,
			`Domain{Devices: DomainDeviceList{Sounds: [DomainSound{Model: "ich6"}]}}`,
		)

		By("Starting an Alpine VMI")
		vmi := libvmifact.NewAlpine()
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the watchdog from hook 1 is present")
		expectWatchdog(vmi)

		By("Verifying the sound card from hook 2 is present in the domain XML")
		domainSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
		Expect(err).NotTo(HaveOccurred())
		Expect(domainSpec.Devices.SoundCards).NotTo(BeEmpty())
		Expect(domainSpec.Devices.SoundCards[0].Model).To(Equal("ich6"))
	})

	It("should skip hook when condition does not match", func() {
		createCelPlugin("conditional-watchdog-plugin",
			`"apply-watchdog" in vmi.Labels && vmi.Labels["apply-watchdog"] == "true"`,
			`Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "none"}]}}`,
		)

		By("Starting an Alpine VMI without the matching label")
		vmi := libvmifact.NewAlpine()
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the watchdog device was NOT injected")
		domainSpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
		Expect(err).NotTo(HaveOccurred())
		for _, wd := range domainSpec.Devices.Watchdogs {
			Expect(wd.Model).NotTo(Equal("i6300esb"), "Plugin watchdog should not be injected when condition doesn't match")
		}
	})

	It("should preserve plugin-injected devices after live migration", decorators.RequiresTwoSchedulableNodes, func() {
		createCelPlugin("migration-watchdog-plugin", "", `Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "none"}]}}`)

		By("Starting an Alpine VMI")
		vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking())
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, flags.StartupTimeoutSecondsLarge())

		By("Verifying the watchdog device is present before migration")
		expectWatchdog(vmi)

		By("Performing a live migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

		By("Verifying the watchdog device is still present after migration")
		expectWatchdog(vmi)
	})
})
