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

	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/tests/util"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	hooksv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksv1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
)

const (
	hookSidecarImage     = "example-hook-sidecar"
	sidecarContainerName = "hook-sidecar-0"
)

var _ = Describe("[sig-compute]HookSidecars", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		tests.BeforeTestCleanup()
		vmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
		vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha1.Version)
	})

	Describe("[rfe_id:2667][crit:medium][vendor:cnv-qe@redhat.com][level:component] VMI definition", func() {
		getVMIPod := func(vmi *v1.VirtualMachineInstance) (*k8sv1.Pod, error) {
			podSelector := tests.UnfinishedVMIPodSelector(vmi)
			vmiPods, err := virtClient.CoreV1().Pods(vmi.GetNamespace()).List(context.Background(), podSelector)

			if err != nil || len(vmiPods.Items) != 1 {
				return nil, fmt.Errorf("could not retrieve the VMI pod: %v", err)
			}
			return &vmiPods.Items[0], nil
		}

		Context("with SM BIOS hook sidecar", func() {
			It("[test_id:3155]should successfully start with hook sidecar annotation", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)
			}, 300)

			It("[test_id:3156]should successfully start with hook sidecar annotation for v1alpha2", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha2.Version)
				Expect(err).ToNot(HaveOccurred())
				tests.WaitForSuccessfulVMIStart(vmi)
			}, 300)

			It("[test_id:3157]should call Collect and OnDefineDomain on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).ToNot(HaveOccurred())
				logs := func() string { return getHookSidecarLogs(virtClient, vmi) }
				tests.WaitForSuccessfulVMIStart(vmi)
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's Info method has been called"))
				Eventually(logs,
					11*time.Second,
					500*time.Millisecond).
					Should(ContainSubstring("Hook's OnDefineDomain callback method has been called"))
			}, 300)

			It("[test_id:3158]should update domain XML with SM BIOS properties", func() {
				By("Reading domain XML using virsh")
				tests.SkipIfNoCmd("kubectl")
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				tests.WaitForSuccessfulVMIStart(vmi)
				domainXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(domainXml).Should(ContainSubstring("<sysinfo type='smbios'>"))
				Expect(domainXml).Should(ContainSubstring("<smbios mode='sysinfo'/>"))
				Expect(domainXml).Should(ContainSubstring("<entry name='manufacturer'>Radical Edward</entry>"))
			}, 300)

			It("should not start with hook sidecar annotation when the version is not provided", func() {
				By("Starting a VMI")
				vmi.ObjectMeta.Annotations = RenderInvalidSMBiosSidecar()
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).NotTo(HaveOccurred(), "the request to create the VMI should be accepted")

				Eventually(func() bool {
					vmiPod, err := getVMIPod(vmi)
					Expect(err).NotTo(HaveOccurred(), "must be able to retrieve the VMI virt-launcher pod")

					for _, container := range vmiPod.Status.ContainerStatuses {
						if container.Name == sidecarContainerName && container.State.Terminated != nil {
							terminated := container.State.Terminated
							return terminated.ExitCode != 0 && terminated.Reason == "Error"
						}
					}
					return false
				}, 15*time.Second, time.Second).Should(
					BeTrue(),
					fmt.Sprintf("the %s container must fail if it was not provided the hook version to advertise itself", sidecarContainerName))
			}, 30)
		})

		Context("[Serial]with sidecar feature gate disabled", func() {
			BeforeEach(func() {
				tests.DisableFeatureGate(virtconfig.SidecarGate)
			})

			It("[test_id:2666]should not start with hook sidecar annotation", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
				Expect(err).To(HaveOccurred(), "should not create a VMI without sidecar feature gate")
				Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("invalid entry metadata.annotations.%s", hooks.HookSidecarListAnnotationName)))
			}, 30)
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
	Expect(err).To(BeNil())

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
