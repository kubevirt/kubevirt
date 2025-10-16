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
	_ "embed"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/hooks"
	hooksv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	hooksv1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	hooksv1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libregistry"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	hookSidecarImage     = "example-hook-sidecar"
	sidecarShimImage     = "sidecar-shim"
	sidecarContainerName = "hook-sidecar-0"
	configMapKey         = "my_script"
)

//go:embed testdata/sidecar-hook-configmap.sh
var configMapData string

var _ = Describe("[sig-compute]HookSidecars", decorators.SigCompute, func() {

	var (
		err        error
		virtClient kubecli.KubevirtClient

		vmi *v1.VirtualMachineInstance
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		vmi = libvmifact.NewAlpine(libvmi.WithInterface(
			libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)
		vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha1.Version)
	})

	Describe("[rfe_id:2667][crit:medium][vendor:cnv-qe@redhat.com][level:component] VMI definition", func() {
		Context("set sidecar resources", func() {
			var originalConfig v1.KubeVirtConfiguration
			BeforeEach(func() {
				originalConfig = *libkubevirt.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			})

			AfterEach(func() {
				kvconfig.UpdateKubeVirtConfigValueAndWait(originalConfig)
			})

			It("[test_id:3155]should successfully start with hook sidecar annotation", Serial, func() {
				resources := v1.ResourceRequirementsWithoutClaims{
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
				kvconfig.UpdateKubeVirtConfigValueAndWait(*config)
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
				By("Finding virt-launcher pod")
				virtlauncherPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())
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
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				vmi.ObjectMeta.Annotations = RenderSidecar(hooksv1alpha2.Version)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi)
			})

			It("[test_id:3157]should call Collect and OnDefineDomain on the hook sidecar", func() {
				By("Getting hook-sidecar logs")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
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

			It("should not start with hook sidecar annotation when the version is not provided", func() {
				By("Starting a VMI")
				vmi.ObjectMeta.Annotations = RenderInvalidSMBiosSidecar()
				vmi = libvmops.RunVMIAndExpectScheduling(vmi, 90)
				Expect(err).NotTo(HaveOccurred(), "the request to create the VMI should be accepted")

				Eventually(func() bool {
					vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
					Expect(err).NotTo(HaveOccurred(), "must be able to retrieve the VMI virt-launcher pod")

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

		Context("with sidecar-shim", func() {
			It("should receive Terminal signal on VMI deletion", func() {
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

				err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				Eventually(func(g Gomega) {
					vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
					Expect(err).ToNot(HaveOccurred())

					var tailLines int64 = 100
					logsRaw, err := virtClient.CoreV1().
						Pods(vmiPod.GetObjectMeta().GetNamespace()).
						GetLogs(vmiPod.GetObjectMeta().GetName(), &k8sv1.PodLogOptions{
							TailLines: &tailLines,
							Container: sidecarContainerName,
						}).
						DoRaw(context.Background())
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(string(logsRaw)).To(ContainSubstring("sidecar-shim received signal: terminated"))
				}, 30*time.Second, time.Second).Should(
					Succeed(),
					fmt.Sprintf("container %s should terminate", sidecarContainerName))
			})

			DescribeTable("migrate VMI with sidecar", decorators.RequiresTwoSchedulableNodes, func(hookVersion string, sidecarShouldTerminate bool) {
				vmi.ObjectMeta.Annotations = RenderSidecar(hookVersion)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)

				sourcePod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				sourcePodName := sourcePod.GetObjectMeta().GetName()
				sourcePodUID := sourcePod.GetObjectMeta().GetUID()

				migration := libmigration.New(vmi.Name, testsuite.GetTestNamespace(vmi))
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				targetPod, err := libpod.GetPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
				Expect(err).ToNot(HaveOccurred())

				targetPodUID := targetPod.GetObjectMeta().GetUID()
				Expect(sourcePodUID).ToNot(Equal(targetPodUID))

				Eventually(func(g Gomega) {
					pods, err := virtClient.CoreV1().Pods("").List(
						context.Background(),
						metav1.ListOptions{
							FieldSelector: fields.ParseSelectorOrDie("metadata.name=" + sourcePodName).String(),
						})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(pods.Items).To(HaveLen(1))
					computeTerminated := false
					sidecarTerminated := false
					for _, container := range pods.Items[0].Status.ContainerStatuses {
						hasTerminated := container.State.Terminated != nil
						switch container.Name {
						case "compute":
							computeTerminated = hasTerminated
						case sidecarContainerName:
							sidecarTerminated = hasTerminated
						}
					}
					g.Expect(computeTerminated).To(BeTrue())
					g.Expect(sidecarTerminated).To(Equal(sidecarShouldTerminate))
				}, 30*time.Second, 1*time.Second).Should(Succeed())
			},
				// See: https://github.com/kubevirt/kubevirt/issues/8395#issuecomment-1619187827
				Entry("Fails to terminate on migration with < v1alpha3", hooksv1alpha2.Version, false),
				Entry("Terminates properly on migration with >= v1alpha3", hooksv1alpha3.Version, true),
			)
		})

		Context("with ConfigMap in sidecar hook annotation", func() {

			DescribeTable("should update domain XML with SM BIOS properties", func(withImage bool) {
				cm, err := virtClient.CoreV1().ConfigMaps(testsuite.GetTestNamespace(vmi)).Create(context.TODO(), RenderConfigMap(), metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				if withImage {
					vmi.ObjectMeta.Annotations = RenderSidecarWithConfigMapPlusImage(hooksv1alpha2.Version, cm.Name)
				} else {
					vmi.ObjectMeta.Annotations = RenderSidecarWithConfigMapWithoutImage(hooksv1alpha2.Version, cm.Name)
				}
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsXHuge)
				domainXml, err := libdomain.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).NotTo(HaveOccurred())
				Expect(domainXml).Should(ContainSubstring("<sysinfo type='smbios'>"))
				Expect(domainXml).Should(ContainSubstring("<smbios mode='sysinfo'/>"))
				Expect(domainXml).Should(ContainSubstring("<entry name='manufacturer'>Radical Edward</entry>"))
			},
				Entry("when sidecar image is specified", true),
				Entry("when sidecar image is not specified", false),
			)
		})

		Context("with sidecar feature gate disabled", Serial, func() {
			BeforeEach(func() {
				kvconfig.DisableFeatureGate(featuregate.SidecarGate)
			})

			It("[test_id:2666]should not start with hook sidecar annotation", func() {
				By("Starting a VMI")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred(), "should not create a VMI without sidecar feature gate")
				Expect(err.Error()).Should(ContainSubstring(fmt.Sprintf("invalid entry metadata.annotations.%s", hooks.HookSidecarListAnnotationName)))
			})
		})
	})
})

func getHookSidecarLogs(virtCli kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) string {
	namespace := vmi.GetObjectMeta().GetNamespace()
	pod, err := libpod.GetPodByVirtualMachineInstance(vmi, namespace)
	Expect(err).ToNot(HaveOccurred())

	var tailLines int64 = 100
	logsRaw, err := virtCli.CoreV1().
		Pods(namespace).
		GetLogs(pod.Name, &k8sv1.PodLogOptions{
			TailLines: &tailLines,
			Container: sidecarContainerName,
		}).
		DoRaw(context.Background())
	Expect(err).ToNot(HaveOccurred())

	return string(logsRaw)
}

func RenderSidecar(version string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf(`[{"args": ["--version", "%s"],"image": "%s", "imagePullPolicy": "IfNotPresent"}]`, version, libregistry.GetUtilityImageFromRegistry(hookSidecarImage)),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}

func RenderInvalidSMBiosSidecar() map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars":              fmt.Sprintf(`[{"image": "%s", "imagePullPolicy": "IfNotPresent"}]`, libregistry.GetUtilityImageFromRegistry(hookSidecarImage)),
		"smbios.vm.kubevirt.io/baseBoardManufacturer": "Radical Edward",
	}
}

func RenderSidecarWithConfigMapPlusImage(version, name string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(`[{"args": ["--version", "%s"], "image":"%s", "configMap": {"name": "%s","key": "%s", "hookPath": "/usr/bin/onDefineDomain"}}]`,
			version, libregistry.GetUtilityImageFromRegistry(sidecarShimImage), name, configMapKey),
	}
}

func RenderSidecarWithConfigMapWithoutImage(version, name string) map[string]string {
	return map[string]string{
		"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(`[{"args": ["--version", "%s"], "configMap": {"name": "%s","key": "%s", "hookPath": "/usr/bin/onDefineDomain"}}]`,
			version, name, configMapKey),
	}
}

func RenderConfigMap() *k8sv1.ConfigMap {
	return &k8sv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cm-",
		},
		Data: map[string]string{
			configMapKey: configMapData,
		},
	}
}
