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
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	expect "github.com/google/goexpect"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"

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
	"kubevirt.io/kubevirt/tests/testsuite"
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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

		By("Verifying the watchdog device is present inside the guest")
		expectWatchdog(vmi)
	})

	It("should preserve existing VMI spec devices when plugin modifies a different domain field", func() {
		createCelPlugin("soundcard-plugin", "", `Domain{Devices: DomainDeviceList{Sounds: [DomainSound{Model: "ich6"}]}}`)

		By("Starting an Alpine VMI with an explicit watchdog in its spec")
		vmi := libvmifact.NewAlpine(libvmi.WithWatchdog(v1.WatchdogActionReset, libnode.GetArch()))
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

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
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsLarge)

		By("Verifying the watchdog device is present before migration")
		expectWatchdog(vmi)

		By("Performing a live migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

		By("Verifying the watchdog device is still present after migration")
		expectWatchdog(vmi)
	})

	Context("sidecar domain hooks", func() {
		BeforeEach(func() {
			enabled, err := util.IsMutatingAdmissionPolicyEnabled(kubevirt.Client())
			Expect(err).NotTo(HaveOccurred())
			if !enabled {
				Skip("MutatingAdmissionPolicy not available in this cluster")
			}
		})

		deploySidecarInjectorMAP := func(pluginName, socketPath string, sidecarCommand []string) {
			virtClient := kubevirt.Client()
			testNamespace := testsuite.GetTestNamespace(nil)
			policyName := fmt.Sprintf("plugin-sidecar-injector-%s", pluginName)
			sidecarImage := fmt.Sprintf("%s/test-domain-hook-sidecar:%s", flags.KubeVirtRepoPrefix, flags.KubeVirtVersionTag)

			mapGVR := schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingadmissionpolicies"}
			bindingGVR := schema.GroupVersionResource{Group: "admissionregistration.k8s.io", Version: "v1", Resource: "mutatingadmissionpolicybindings"}

			commandCEL := "["
			for i, c := range sidecarCommand {
				if i > 0 {
					commandCEL += ", "
				}
				commandCEL += fmt.Sprintf("%q", c)
			}
			commandCEL += "]"

			By("Creating MutatingAdmissionPolicy for sidecar injection")
			policy := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "admissionregistration.k8s.io/v1",
					"kind":       "MutatingAdmissionPolicy",
					"metadata":   map[string]interface{}{"name": policyName},
					"spec": map[string]interface{}{
						"failurePolicy":      "Fail",
						"reinvocationPolicy": "Never",
						"matchConstraints": map[string]interface{}{
							"resourceRules": []interface{}{
								map[string]interface{}{
									"operations":  []interface{}{"CREATE"},
									"apiGroups":   []interface{}{""},
									"apiVersions": []interface{}{"v1"},
									"resources":   []interface{}{"pods"},
								},
							},
						},
						"matchConditions": []interface{}{
							map[string]interface{}{
								"name":       "is-virt-launcher-pod",
								"expression": `has(object.metadata.labels) && "kubevirt.io" in object.metadata.labels && object.metadata.labels["kubevirt.io"] == "virt-launcher"`,
							},
							map[string]interface{}{
								"name":       "is-test-namespace",
								"expression": fmt.Sprintf(`object.metadata.namespace == %q`, testNamespace),
							},
						},
						"mutations": []interface{}{
							map[string]interface{}{
								"patchType": "JSONPatch",
								"jsonPatch": map[string]interface{}{
									"expression": fmt.Sprintf(`[
    JSONPatch{
        op: "add",
        path: "/spec/containers/-",
        value: Object.spec.containers{
            name: "plugin-sidecar-%s",
            image: %q,
            imagePullPolicy: "Always",
            command: %s,
            securityContext: Object.spec.containers.securityContext{
                allowPrivilegeEscalation: false,
                runAsNonRoot: true,
                seccompProfile: Object.spec.containers.securityContext.seccompProfile{
                    type: "RuntimeDefault"
                },
                capabilities: Object.spec.containers.securityContext.capabilities{
                    drop: ["ALL"]
                }
            },
            volumeMounts: [Object.spec.containers.volumeMounts{
                name: "kubevirt-plugin-sockets",
                mountPath: "/var/run/kubevirt-plugin/%s/",
                subPath: %q
            }]
        }
    }
]`, pluginName, sidecarImage, commandCEL, pluginName, pluginName),
								},
							},
						},
					},
				},
			}
			_, err := virtClient.DynamicClient().Resource(mapGVR).Create(context.Background(), policy, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = virtClient.DynamicClient().Resource(mapGVR).Delete(context.Background(), policyName, metav1.DeleteOptions{})
			})

			By("Creating MutatingAdmissionPolicyBinding")
			binding := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "admissionregistration.k8s.io/v1",
					"kind":       "MutatingAdmissionPolicyBinding",
					"metadata":   map[string]interface{}{"name": policyName + "-binding"},
					"spec": map[string]interface{}{
						"policyName": policyName,
					},
				},
			}
			_, err = virtClient.DynamicClient().Resource(bindingGVR).Create(context.Background(), binding, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				_ = virtClient.DynamicClient().Resource(bindingGVR).Delete(context.Background(), policyName+"-binding", metav1.DeleteOptions{})
			})

			By("Waiting for MutatingAdmissionPolicy to become active")
			expectedSidecarName := fmt.Sprintf("plugin-sidecar-%s", pluginName)
			Eventually(func(g Gomega) {
				probePod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "map-readiness-probe", Namespace: testNamespace, Labels: map[string]string{"kubevirt.io": "virt-launcher"}},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "compute", Image: "busybox"}},
						Volumes:    []corev1.Volume{{Name: "kubevirt-plugin-sockets", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}},
					},
				}
				result, err := virtClient.CoreV1().Pods(testNamespace).Create(context.Background(), probePod, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
				g.Expect(err).NotTo(HaveOccurred())
				var containerNames []string
				for _, c := range result.Spec.Containers {
					containerNames = append(containerNames, c.Name)
				}
				g.Expect(containerNames).To(ContainElement(expectedSidecarName), "MAP has not injected the sidecar container yet")
			}).WithTimeout(30 * time.Second).WithPolling(time.Second).Should(Succeed())
		}

		deploySleepingSidecarInjectorMAP := func(pluginName string) {
			socketPath := fmt.Sprintf("/var/run/kubevirt-plugin/%s/hook.sock", pluginName)
			deploySidecarInjectorMAP(pluginName, socketPath, []string{"/usr/bin/test-domain-hook-sidecar", "--sleep"})
		}

		It("should apply both CEL and sidecar domain hooks", func() {
			const pluginName = "composition-test"
			socketPath := fmt.Sprintf("/var/run/kubevirt-plugin/%s/hook.sock", pluginName)

			deploySidecarInjectorMAP(pluginName, socketPath, []string{"/usr/bin/test-domain-hook-sidecar", socketPath})

			By("Creating Plugin with CEL watchdog hook and sidecar domain hook")
			createPlugin(&pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: pluginName},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{
						{
							CEL: &pluginv1alpha1.CELDomainHook{
								Expression: `Domain{Devices: DomainDeviceList{Watchdogs: [DomainWatchdog{Model: "i6300esb", Action: "poweroff"}]}}`,
							},
						},
						{
							Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: socketPath},
						},
					},
				},
			})

			By("Starting an Alpine VMI")
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXLarge)

			By("Verifying the watchdog device from CEL hook is visible inside the guest")
			expectWatchdog(vmi)

			By("Verifying the sidecar hook set sys_vendor via SMBIOS")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			output, err := console.RunCommandAndStoreOutput(vmi, "cat /sys/class/dmi/id/sys_vendor\n", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("KubeVirt-Plugin-CPU-"))
		})

		It("should preserve sidecar hook mutations after live migration", decorators.RequiresTwoSchedulableNodes, func() {
			const pluginName = "migration-test"
			socketPath := fmt.Sprintf("/var/run/kubevirt-plugin/%s/hook.sock", pluginName)

			deploySidecarInjectorMAP(pluginName, socketPath, []string{"/usr/bin/test-domain-hook-sidecar", socketPath})

			By("Creating Plugin with sidecar domain hook")
			createPlugin(&pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: pluginName},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar: &pluginv1alpha1.SidecarDomainHook{SocketPath: socketPath},
					}},
				},
			})

			By("Starting an Alpine VMI")
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking())
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXLarge)

			By("Verifying sidecar hook applied before migration")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			output, err := console.RunCommandAndStoreOutput(vmi, "cat /sys/class/dmi/id/sys_vendor\n", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("KubeVirt-Plugin-CPU-"))

			By("Performing a live migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(kubevirt.Client(), migration)

			By("Verifying sidecar hook mutations are preserved after migration")
			output, err = console.RunCommandAndStoreOutput(vmi, "cat /sys/class/dmi/id/sys_vendor\n", 30*time.Second)
			Expect(err).NotTo(HaveOccurred())
			Expect(output).To(ContainSubstring("KubeVirt-Plugin-CPU-"))
		})

		It("should skip sidecar hook when socket is not ready and failureStrategy is Ignore", func() {
			const pluginName = "ignore-test"
			socketPath := fmt.Sprintf("/var/run/kubevirt-plugin/%s/hook.sock", pluginName)

			deploySleepingSidecarInjectorMAP(pluginName)

			By("Creating Plugin with sidecar hook and FailureStrategy Ignore")
			createPlugin(&pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: pluginName},
				Spec: pluginv1alpha1.PluginSpec{
					DomainHooks: []pluginv1alpha1.DomainHook{{
						Sidecar:         &pluginv1alpha1.SidecarDomainHook{SocketPath: socketPath},
						FailureStrategy: pluginv1alpha1.FailureStrategyIgnore,
					}},
				},
			})

			By("Starting an Alpine VMI - should succeed even though sidecar socket never appears")
			vmi := libvmifact.NewAlpine(libnet.WithMasqueradeNetworking())
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, libvmops.StartupTimeoutSecondsHuge)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
		})
	})
})
