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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"fmt"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksv1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/clientcmd"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	hookSidecarImage     = "example-hook-sidecar"
	sidecarContainerName = "hook-sidecar-0"
)

var _ = Describe("[sig-compute]HookSidecars", decorators.SigCompute, func() {

	var (
		err        error
		virtClient kubecli.KubevirtClient

		vmi *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
		vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha1.Version)
	})

	Describe("[rfe_id:2667][crit:medium][vendor:cnv-qe@redhat.com][level:component] VMI definition", func() {
		getVMIPod := func(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, bool, error) {
			podSelector := tests.UnfinishedVMIPodSelector(vmi)
			vmiPods, err := virtClient.CoreV1().Pods(vmi.GetNamespace()).List(context.Background(), podSelector)

			if err != nil {
				return nil, false, fmt.Errorf("could not retrieve the VMI pod: %v", err)
			} else if len(vmiPods.Items) == 0 {
				return nil, false, nil
			}
			return &vmiPods.Items[0], true, nil
		}

		Context("set sidecar resources", func() {
			var originalConfig v1.KubeVirtConfiguration
			BeforeEach(func() {
				originalConfig = *util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			})

			AfterEach(func() {
				tests.UpdateKubeVirtConfigValueAndWait(originalConfig)
			})

			It("[test_id:3155][serial]should successfully start with hook sidecar annotation", Serial, func() {
				resources := k8sv1.ResourceRequirements{
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("1m"),
						k8sv1.ResourceMemory: resource.MustParse("10M"),
					},
					Limits: k8sv1.ResourceList{
						k8sv1.ResourceCPU:    resource.MustParse("201m"),
						k8sv1.ResourceMemory: resource.MustParse("74M"),
					},
				}
				config := originalConfig.DeepCopy()
				config.SupportContainerResources = []v1.SupportContainerResources{
					{
						Type:      v1.SideCar,
						Resources: resources,
					},
				}
				tests.UpdateKubeVirtConfigValueAndWait(*config)
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
				By("Finding virt-launcher pod")
				var virtlauncherPod *k8sv1.Pod
				Eventually(func() *k8sv1.Pod {
					podList, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
					if err != nil {
						return nil
					}
					for _, pod := range podList.Items {
						for _, ownerRef := range pod.GetOwnerReferences() {
							if ownerRef.UID == vmi.GetUID() {
								virtlauncherPod = &pod
								break
							}
						}
					}
					return virtlauncherPod
				}, 30*time.Second, 1*time.Second).ShouldNot(BeNil())
				Expect(virtlauncherPod.Spec.Containers).To(HaveLen(4))
				foundContainer := false
				for _, container := range virtlauncherPod.Spec.Containers {
					if container.Name == "hook-sidecar-0" {
						foundContainer = true
						Expect(container.Resources.Requests.Cpu().Value()).To(Equal(resources.Requests.Cpu().Value()))
						Expect(container.Resources.Requests.Memory().Value()).To(Equal(resources.Requests.Memory().Value()))
						Expect(container.Resources.Limits.Cpu().Value()).To(Equal(resources.Limits.Cpu().Value()))
						Expect(container.Resources.Limits.Memory().Value()).To(Equal(resources.Limits.Memory().Value()))
					}
				}
				Expect(foundContainer).To(BeTrue())
			})
		})

		Context("with SM BIOS hook sidecar", func() {
			It("[test_id:3156]should successfully start with hook sidecar annotation for v1alpha2", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha2.Version)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:3157]should call Collect and OnDefineDomain on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())
				logs := func() string { return getHookSidecarLogs(virtClient, vmi) }
				libwait.WaitForSuccessfulVMIStart(vmi)
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Info method has been called"))
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("OnDefineDomain method has been called"))
			})

			It("[test_id:3158]should update domain XML with SM BIOS properties", func() {
				By("Reading domain XML using virsh")
				clientcmd.SkipIfNoCmd("kubectl")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				libwait.WaitForSuccessfulVMIStart(vmi)
				domainXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(domainXml).Should(ContainSubstring("<sysinfo type='smbios'>"))
				Expect(domainXml).Should(ContainSubstring("<smbios mode='sysinfo'/>"))
				Expect(domainXml).Should(ContainSubstring("<entry name='manufacturer'>Radical Edward</entry>"))
			})

			It("should not start with hook sidecar annotation when the version is not provided", func() {
				By("Starting a VMI")
				vmi.ObjectMeta.Annotations = RenderInvalidSMBiosSidecar()
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).NotTo(HaveOccurred(), "the request to create the VMI should be accepted")

				Eventually(func() bool {
					vmiPod, exists, err := getVMIPod(vmi)
					if err != nil {
						Expect(err).NotTo(HaveOccurred(), "must be able to retrieve the VMI virt-launcher pod")
					} else if !exists {
						return false
					}

					for _, container := range vmiPod.Status.ContainerStatuses {
						if container.Name == sidecarContainerName && container.State.Terminated != nil {
							terminated := container.State.Terminated
							return terminated.ExitCode != 0 && terminated.Reason == "Error"
						}
					}
					return false
				}, 30*time.Second, time.Second).Should(
					BeTrue(),
					fmt.Sprintf("the %s container must fail if it was not provided the hook version to advertise itself", sidecarContainerName))
			})
		})

		Context("[Serial]with sidecar feature gate disabled", Serial, func() {
			BeforeEach(func() {
				tests.DisableFeatureGate(virtconfig.SidecarGate)
			})

			It("[test_id:2666]should not start with hook sidecar annotation", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).To(HaveOccurred(), "should not create a VMI without sidecar feature gate")
				Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("invalid entry metadata.annotations.%s", hooks.HookSidecarListAnnotationName)))
			})
		})
	})
})

func getHookSidecarLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	podName := tests.GetVmPodName(virtCli, vmi)

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(podName, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: sidecarContainerName,
		}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred())

	return string(logsRaw)
}

func RenderSidecar(version string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf(`[{"args": ["--version", "%s"],"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`, version, flags.KubeVirtUtilityRepoPrefix, hookSidecarImage, flags.KubeVirtUtilityVersionTag),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}

func RenderInvalidSMBiosSidecar() map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf(`[{"image": "%s/%s:%s", "imagePullPolicy": "IfNotPresent"}]`, flags.KubeVirtUtilityRepoPrefix, hookSidecarImage, flags.KubeVirtUtilityVersionTag),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}
