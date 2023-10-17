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
 * Copyright the KubeVirt Authors.
 *
 */

package vmi_configuration

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	k8sv1 "k8s.io/api/core/v1"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/util"

	kubevirt_hooks_v1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = ConfigDescribe("", func() {
	const enoughMemForSafeBiosEmulation = "32Mi"
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("with all devices on the root PCI bus", func() {
		It("[test_id:4623]should start run the guest as usual", func() {
			vmi := libvmi.NewCirros(
				libvmi.WithAnnotation(v1.PlacePCIDevicesOnRootComplex, "true"),
				libvmi.WithRng(),
				libvmi.WithWatchdog(v1.WatchdogActionPoweroff),
			)
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "tablet", Bus: v1.VirtIO, Type: "tablet"}, {Name: "tablet1", Bus: "usb", Type: "tablet"}}
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(console.LoginToCirros(vmi)).To(Succeed())
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			rootPortController := []api.Controller{}
			for _, c := range domSpec.Devices.Controllers {
				if c.Model == "pcie-root-port" {
					rootPortController = append(rootPortController, c)
				}
			}
			Expect(rootPortController).To(BeEmpty(), "libvirt should not add additional buses to the root one")
		})
	})

	Context("when requesting virtio-transitional models", func() {
		It("[test_id:6957]should start and run the guest", func() {
			vmi := libvmi.NewCirros(
				libvmi.WithRng(),
				libvmi.WithWatchdog(v1.WatchdogActionPoweroff),
			)
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{{Name: "tablet", Bus: v1.VirtIO, Type: "tablet"}, {Name: "tablet1", Bus: "usb", Type: "tablet"}}
			vmi.Spec.Domain.Devices.UseVirtioTransitional = pointer.BoolPtr(true)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(console.LoginToCirros(vmi)).To(Succeed())
			domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			testutils.ExpectVirtioTransitionalOnly(domSpec)
		})
	})

	Context("[rfe_id:897][crit:medium][vendor:cnv-qe@redhat.com][level:component]for CPU and memory limits should", func() {

		It("[test_id:3110]lead to get the burstable QOS class assigned when limit and requests differ", func() {
			vmi := libvmi.NewAlpine()
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSBurstable))
		})

		It("[test_id:3111]lead to get the guaranteed QOS class assigned when limit and requests are identical", func() {
			vmi := libvmi.NewAlpine()
			By("specifying identical limits and requests")
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
				Limits: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}

			By("adding a sidecar to ensure it gets limits assigned too")
			vmi.ObjectMeta.Annotations = libvmi.RenderSidecar(kubevirt_hooks_v1alpha2.Version)
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))
		})

		It("[test_id:3112]lead to get the guaranteed QOS class assigned when only limits are set", func() {
			vmi := libvmi.NewAlpine()
			By("specifying identical limits and requests")
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: kubev1.ResourceList{},
				Limits: kubev1.ResourceList{
					kubev1.ResourceCPU:    resource.MustParse("1"),
					kubev1.ResourceMemory: resource.MustParse("64M"),
				},
			}

			By("adding a sidecar to ensure it gets limits assigned too")
			vmi.ObjectMeta.Annotations = libvmi.RenderSidecar(kubevirt_hooks_v1alpha2.Version)
			vmi = tests.RunVMIAndExpectScheduling(vmi, 60)

			Eventually(func() kubev1.PodQOSClass {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.IsFinal()).To(BeFalse())
				if vmi.Status.QOSClass == nil {
					return ""
				}
				return *vmi.Status.QOSClass
			}, 10*time.Second, 1*time.Second).Should(Equal(kubev1.PodQOSGuaranteed))

			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Domain.Resources.Requests.Cpu().Cmp(*vmi.Spec.Domain.Resources.Limits.Cpu())).To(BeZero())
			Expect(vmi.Spec.Domain.Resources.Requests.Memory().Cmp(*vmi.Spec.Domain.Resources.Limits.Memory())).To(BeZero())
		})

	})

	Context("[Serial][rfe_id:2869][crit:medium][vendor:cnv-qe@redhat.com][level:component]with machine type settings", Serial, func() {
		testEmulatedMachines := []string{"q35*", "pc-q35*", "pc*"}

		BeforeEach(func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			config.MachineType = ""
			config.ArchitectureConfiguration = &v1.ArchConfiguration{Amd64: &v1.ArchSpecificConfiguration{}, Arm64: &v1.ArchSpecificConfiguration{}, Ppc64le: &v1.ArchSpecificConfiguration{}}
			config.ArchitectureConfiguration.Amd64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Arm64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Ppc64le.EmulatedMachines = testEmulatedMachines

			tests.UpdateKubeVirtConfigValueAndWait(config)
		})

		It("[test_id:3124]should set machine type from VMI spec", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				libvmi.WithMachineType("pc"),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))

			Expect(vmi.Status.Machine).ToNot(BeNil())
			Expect(vmi.Status.Machine.Type).To(Equal(runningVMISpec.OS.Type.Machine))
		})

		It("[test_id:3125]should allow creating VM without Machine defined", func() {
			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine = nil
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("[test_id:6964]should allow creating VM defined with Machine with an empty Type", func() {
			// This is needed to provide backward compatibility since our example VMIs used to be defined in this way
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				libvmi.WithMachineType(""),
			)

			vmi = tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("q35"))
		})

		It("[Serial][test_id:3126]should set machine type from kubevirt-config", Serial, func() {
			kv := util.GetCurrentKv(virtClient)
			testEmulatedMachines := []string{"pc"}

			config := kv.Spec.Configuration

			config.ArchitectureConfiguration = &v1.ArchConfiguration{Amd64: &v1.ArchSpecificConfiguration{}, Arm64: &v1.ArchSpecificConfiguration{}, Ppc64le: &v1.ArchSpecificConfiguration{}}
			config.ArchitectureConfiguration.Amd64.MachineType = "pc"
			config.ArchitectureConfiguration.Arm64.MachineType = "pc"
			config.ArchitectureConfiguration.Ppc64le.MachineType = "pc"
			config.ArchitectureConfiguration.Amd64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Arm64.EmulatedMachines = testEmulatedMachines
			config.ArchitectureConfiguration.Ppc64le.EmulatedMachines = testEmulatedMachines
			tests.UpdateKubeVirtConfigValueAndWait(config)

			vmi := tests.NewRandomVMI()
			vmi.Spec.Domain.Machine = nil
			tests.RunVMIAndExpectLaunch(vmi, 30)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)

			Expect(err).ToNot(HaveOccurred())
			Expect(runningVMISpec.OS.Type.Machine).To(ContainSubstring("pc-i440"))
		})
	})

	Context("with a custom scheduler", func() {
		It("[test_id:4631]should set the custom scheduler on the pod", func() {
			vmi := libvmi.New(
				libvmi.WithResourceMemory(enoughMemForSafeBiosEmulation),
				libvmi.WithSchedulerName("my-custom-scheduler"),
			)
			runningVMI := tests.RunVMIAndExpectScheduling(vmi, 30)
			launcherPod, err := libvmi.GetPodByVirtualMachineInstance(runningVMI, testsuite.GetTestNamespace(vmi))
			Expect(err).ToNot(HaveOccurred())
			Expect(launcherPod.Spec.SchedulerName).To(Equal("my-custom-scheduler"))
		})
	})

	Context("[Serial][rfe_id:904][crit:medium][vendor:cnv-qe@redhat.com][level:component][storage-req]with driver cache and io settings and PVC", Serial, decorators.StorageReq, func() {
		var dataVolume *cdiv1.DataVolume

		BeforeEach(func() {
			var err error
			if !checks.HasFeature(virtconfig.HostDiskGate) {
				Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
			}

			dataVolume, err = createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))
		})

		AfterEach(func() {
			libstorage.DeleteDataVolume(&dataVolume)
		})

		It("[test_id:1681]should set appropriate cache modes", func() {
			vmi := tests.NewRandomVMI()

			By("adding disks to a VMI")
			tests.AddEphemeralDisk(vmi, "ephemeral-disk1", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone

			tests.AddEphemeralDisk(vmi, "ephemeral-disk2", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[1].Cache = v1.CacheWriteThrough

			tests.AddEphemeralDisk(vmi, "ephemeral-disk5", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[2].Cache = v1.CacheWriteBack

			tests.AddEphemeralDisk(vmi, "ephemeral-disk3", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			tmpHostDiskDir := tests.RandTmpDir()
			tests.AddHostDisk(vmi, filepath.Join(tmpHostDiskDir, "test-disk.img"), v1.HostDiskExistsOrCreate, "hostdisk")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, testsuite.GetTestNamespace(vmi))
			defer tests.RemoveHostDiskImage(tmpHostDiskDir, vmiPod.Spec.NodeName)

			disks := runningVMISpec.Devices.Disks
			By("checking if number of attached disks is equal to real disks number")
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			cacheNone := string(v1.CacheNone)
			cacheWritethrough := string(v1.CacheWriteThrough)
			cacheWriteback := string(v1.CacheWriteBack)

			By("checking if requested cache 'none' has been set")
			Expect(disks[0].Alias.GetName()).To(Equal("ephemeral-disk1"))
			Expect(disks[0].Driver.Cache).To(Equal(cacheNone))

			By("checking if requested cache 'writethrough' has been set")
			Expect(disks[1].Alias.GetName()).To(Equal("ephemeral-disk2"))
			Expect(disks[1].Driver.Cache).To(Equal(cacheWritethrough))

			By("checking if requested cache 'writeback' has been set")
			Expect(disks[2].Alias.GetName()).To(Equal("ephemeral-disk5"))
			Expect(disks[2].Driver.Cache).To(Equal(cacheWriteback))

			By("checking if default cache 'none' has been set to ephemeral disk")
			Expect(disks[3].Alias.GetName()).To(Equal("ephemeral-disk3"))
			Expect(disks[3].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'none' has been set to cloud-init disk")
			Expect(disks[4].Alias.GetName()).To(Equal("cloud-init"))
			Expect(disks[4].Driver.Cache).To(Equal(cacheNone))

			By("checking if default cache 'writethrough' has been set to fs which does not support direct I/O")
			Expect(disks[5].Alias.GetName()).To(Equal("hostdisk"))
			Expect(disks[5].Driver.Cache).To(Equal(cacheWritethrough))

		})

		It("[test_id:5360]should set appropriate IO modes", func() {
			vmi := tests.NewRandomVMI()

			By("adding disks to a VMI")
			// disk[0]:  File, sparsed, no user-input, cache=none
			tests.AddEphemeralDisk(vmi, "ephemeral-disk1", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone

			// disk[1]:  Block, no user-input, cache=none
			tests.AddPVCDisk(vmi, "block-pvc", v1.DiskBusVirtio, dataVolume.Name)

			// disk[2]: File, not-sparsed, no user-input, cache=none
			tests.AddPVCDisk(vmi, "hostpath-pvc", v1.DiskBusVirtio, tests.DiskAlpineHostPath)

			// disk[3]:  File, sparsed, user-input=threads, cache=none
			tests.AddEphemeralDisk(vmi, "ephemeral-disk2", v1.DiskBusVirtio, cd.ContainerDiskFor(cd.ContainerDiskCirros))
			vmi.Spec.Domain.Devices.Disks[3].Cache = v1.CacheNone
			vmi.Spec.Domain.Devices.Disks[3].IO = v1.IOThreads

			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			disks := runningVMISpec.Devices.Disks
			By("checking if number of attached disks is equal to real disks number")
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			ioNative := v1.IONative
			ioThreads := v1.IOThreads
			ioNone := ""

			By("checking if default io has not been set for sparsed file")
			Expect(disks[0].Alias.GetName()).To(Equal("ephemeral-disk1"))
			Expect(string(disks[0].Driver.IO)).To(Equal(ioNone))

			By("checking if default io mode has been set to 'native' for block device")
			Expect(disks[1].Alias.GetName()).To(Equal("block-pvc"))
			Expect(disks[1].Driver.IO).To(Equal(ioNative))

			By("checking if default cache 'none' has been set to pvc disk")
			Expect(disks[2].Alias.GetName()).To(Equal("hostpath-pvc"))
			// PVC is mounted as tmpfs on kind, which does not support direct I/O.
			// As such, it behaves as plugging in a hostDisk - check disks[6].
			if checks.IsRunningOnKindInfra() {
				// The cache mode is set to cacheWritethrough
				Expect(string(disks[2].Driver.IO)).To(Equal(ioNone))
			} else {
				// The cache mode is set to cacheNone
				Expect(disks[2].Driver.IO).To(Equal(ioNative))
			}

			By("checking if requested io mode 'threads' has been set")
			Expect(disks[3].Alias.GetName()).To(Equal("ephemeral-disk2"))
			Expect(disks[3].Driver.IO).To(Equal(ioThreads))

		})
	})

	Context("Block size configuration set", func() {

		It("[test_id:6965][storage-req]Should set BlockIO when using custom block sizes", decorators.StorageReq, func() {
			By("creating a block volume")
			dataVolume, err := createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

			vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

			By("setting the disk to use custom block sizes")
			logicalSize := uint(16384)
			physicalSize := uint(16384)
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				Custom: &v1.CustomBlockSize{
					Logical:  logicalSize,
					Physical: physicalSize,
				},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set to the custom block size")
			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(logicalSize))
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(Equal(physicalSize))
		})

		It("[test_id:6966][storage-req]Should set BlockIO when set to match volume block sizes on block devices", decorators.StorageReq, func() {
			By("creating a block volume")
			dataVolume, err := createBlockDataVolume(virtClient)
			Expect(err).ToNot(HaveOccurred())
			if dataVolume == nil {
				Skip("Skip test when Block storage is not present")
			}

			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

			vmi := tests.NewRandomVMIWithPVC(dataVolume.Name)

			By("setting the disk to match the volume block sizes")
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				MatchVolume: &v1.FeatureState{},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set for the disk")
			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			// Block devices should be one of 512n, 512e or 4096n so accept 512 and 4096 values.
			expectedDiskSizes := SatisfyAny(Equal(uint(512)), Equal(uint(4096)))
			Expect(disks[0].BlockIO.LogicalBlockSize).To(expectedDiskSizes)
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(expectedDiskSizes)
		})

		It("[test_id:6967]Should set BlockIO when set to match volume block sizes on files", func() {
			if !checks.HasFeature(virtconfig.HostDiskGate) {
				Skip("Cluster has the HostDisk featuregate disabled, skipping  the tests")
			}

			By("creating a disk image")
			var nodeName string
			tmpHostDiskDir := tests.RandTmpDir()
			tmpHostDiskPath := filepath.Join(tmpHostDiskDir, fmt.Sprintf("disk-%s.img", uuid.NewRandom().String()))

			job := tests.CreateHostDiskImage(tmpHostDiskPath)
			job, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Create(context.Background(), job, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(ThisPod(job), 30*time.Second, 1*time.Second).Should(BeInPhase(k8sv1.PodSucceeded))
			pod, err := ThisPod(job)()
			Expect(err).NotTo(HaveOccurred())
			nodeName = pod.Spec.NodeName
			defer tests.RemoveHostDiskImage(tmpHostDiskDir, nodeName)

			vmi := tests.NewRandomVMIWithHostDisk(tmpHostDiskPath, v1.HostDiskExistsOrCreate, nodeName)

			By("setting the disk to match the volume block sizes")
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				MatchVolume: &v1.FeatureState{},
			}

			By("initializing the VM")
			tests.RunVMIAndExpectLaunch(vmi, 60)
			runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set for the disk")
			Expect(disks[0].Alias.GetName()).To(Equal("host-disk"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			// The default for most filesystems nowadays is 4096 but it can be changed.
			// As such, relying on a specific value is flakey.
			// As long as we have a value, the exact value doesn't matter.
			Expect(disks[0].BlockIO.LogicalBlockSize).ToNot(BeZero())
			// A filesystem only has a single size so logical == physical
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(disks[0].BlockIO.PhysicalBlockSize))
		})
	})

	Context("[rfe_id:898][crit:medium][vendor:cnv-qe@redhat.com][level:component]New VirtualMachineInstance with all supported drives", func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			// ordering:
			// use a small disk for the other ones
			containerImage := cd.ContainerDiskFor(cd.ContainerDiskCirros)
			// virtio - added by NewRandomVMIWithEphemeralDisk
			vmi = tests.NewRandomVMIWithEphemeralDiskAndUserdata(containerImage, "echo hi!\n")
			// sata
			tests.AddEphemeralDisk(vmi, "disk2", v1.DiskBusSATA, containerImage)
			// NOTE: we have one disk per bus, so we expect vda, sda
		})
		checkPciAddress := func(vmi *v1.VirtualMachineInstance, expectedPciAddress string) {
			err := console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "grep DEVNAME /sys/bus/pci/devices/" + expectedPciAddress + "/*/block/vda/uevent|awk -F= '{ print $2 }'\n"},
				&expect.BExp{R: "vda"},
			}, 15)
			Expect(err).ToNot(HaveOccurred())
		}

		It("[test_id:1682]should have all the device nodes", func() {
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			Expect(console.LoginToCirros(vmi)).To(Succeed())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// keep the ordering!
				&expect.BSnd{S: "ls /dev/vda  /dev/vdb\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 10)).To(Succeed())
		})

		It("[test_id:3906]should configure custom Pci address", func() {
			By("checking disk1 Pci address")
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:00:10.0"
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = v1.DiskBusVirtio
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

			checkPciAddress(vmi, vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress)
		})

		It("[test_id:1020]should not create the VM with wrong PCI address", func() {
			By("setting disk1 Pci address")

			wrongPciAddress := "0000:04:10.0"

			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = wrongPciAddress
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = v1.DiskBusVirtio
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())

			var vmiCondition v1.VirtualMachineInstanceCondition
			// TODO
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, cond := range vmi.Status.Conditions {
					if cond.Type == v1.VirtualMachineInstanceConditionType(v1.VirtualMachineInstanceSynchronized) && cond.Status == kubev1.ConditionFalse {
						vmiCondition = cond
						return true
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue())

			Expect(vmiCondition.Message).To(ContainSubstring("Invalid PCI address " + wrongPciAddress))
			Expect(vmiCondition.Reason).To(Equal("Synchronizing with the Domain failed."))
		})
	})

	Context("[rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check Chassis value", func() {

		It("[Serial][test_id:2927]Test Chassis value in a newly created VM", Serial, func() {
			vmi := tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
			vmi.Spec.Domain.Chassis = &v1.Chassis{
				Asset: "Test-123",
			}

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values on domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='asset'>Test-123</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check value in VM with dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s chassis-asset-tag | tr -s ' ') = Test-123 ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 10)).To(Succeed())
		})
	})

	Context("[Serial][rfe_id:2926][crit:medium][vendor:cnv-qe@redhat.com][level:component]Check SMBios with default and custom values", Serial, func() {

		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		It("[test_id:2751]test default SMBios", func() {
			kv := util.GetCurrentKv(virtClient)

			config := kv.Spec.Configuration
			// Clear SMBios values if already set in kubevirt-config, for testing default values.
			test_smbios := &v1.SMBiosConfiguration{Family: "", Product: "", Manufacturer: ""}
			config.SMBIOSConfig = test_smbios
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values in domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>KubeVirt</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>KubeVirt</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check values in dmidecode")
			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') = KubeVirt ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') = None ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') = KubeVirt ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1)).To(Succeed())
		})

		It("[test_id:2752]test custom SMBios values", func() {
			kv := util.GetCurrentKv(virtClient)
			config := kv.Spec.Configuration
			// Set a custom test SMBios
			test_smbios := &v1.SMBiosConfiguration{Family: "test", Product: "test", Manufacturer: "None", Sku: "1.0", Version: "1.0"}
			config.SMBIOSConfig = test_smbios
			tests.UpdateKubeVirtConfigValueAndWait(config)

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<entry name='family'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='product'>test</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='manufacturer'>None</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='sku'>1.0</entry>"))
			Expect(domXml).To(ContainSubstring("<entry name='version'>1.0</entry>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check values in dmidecode")

			// Check on the VM, if expected values are there with dmidecode
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-family | tr -s ' ') = test ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-product-name | tr -s ' ') = test ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "[ $(sudo dmidecode -s system-manufacturer | tr -s ' ') = None ] && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1)).To(Succeed())
		})
	})

	Context("With ephemeral CD-ROM", func() {
		var vmi *v1.VirtualMachineInstance
		var DiskBusIDE v1.DiskBus = "ide"

		BeforeEach(func() {
			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		DescribeTable("For various bus types", func(bus v1.DiskBus, errMsg string) {
			tests.AddEphemeralCdrom(vmi, "cdrom-0", bus, cd.ContainerDiskFor(cd.ContainerDiskCirros))

			By(fmt.Sprintf("Starting a VMI with a %s CD-ROM", bus))
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			if errMsg == "" {
				Expect(err).ToNot(HaveOccurred())
			} else {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errMsg))
			}
		},
			Entry("[test_id:3777] Should be accepted when using sata", v1.DiskBusSATA, ""),
			Entry("[test_id:3809] Should be accepted when using scsi", v1.DiskBusSCSI, ""),
			Entry("[test_id:3776] Should be rejected when using virtio", v1.DiskBusVirtio, "Bus type virtio is invalid"),
			Entry("[test_id:3808] Should be rejected when using ide", DiskBusIDE, "IDE bus is not supported"),
		)
	})

	Context("Custom PCI Addresses configuration", func() {
		// The aim of the test is to validate the configurability of a range of PCI slots
		// on the root PCI bus 0. We would like to test slots 2..1a (slots 0,1 and beyond 1a are reserved).
		// In addition , we test usage of PCI functions on a single slot
		// by occupying all the functions 1..7 on random port 2.

		addrPrefix := "0000:00" // PCI bus 0
		numOfSlotsToTest := 24  // slots 2..1a
		numOfFuncsToTest := 8
		var vmi *v1.VirtualMachineInstance

		createDisks := func(numOfDisks int, vmi *v1.VirtualMachineInstance) {
			for i := 0; i < numOfDisks; i++ {
				vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks,
					v1.Disk{
						Name: fmt.Sprintf("test%v", i),
						DiskDevice: v1.DiskDevice{
							Disk: &v1.DiskTarget{
								Bus: v1.DiskBusVirtio,
							},
						},
					})
				vmi.Spec.Volumes = append(vmi.Spec.Volumes,
					v1.Volume{
						Name: fmt.Sprintf("test%v", i),
						VolumeSource: v1.VolumeSource{
							EmptyDisk: &v1.EmptyDiskSource{
								Capacity: resource.MustParse("1Mi"),
							},
						},
					})
			}
		}
		assignDisksToSlots := func(startIndex int, vmi *v1.VirtualMachineInstance) {
			var addr string

			for i, disk := range vmi.Spec.Domain.Devices.Disks {
				addr = fmt.Sprintf("%x", i+startIndex)
				if len(addr) == 1 {
					disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:0%v.0", addrPrefix, addr)
				} else {
					disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:%v.0", addrPrefix, addr)
				}
			}
		}

		assignDisksToFunctions := func(startIndex int, vmi *v1.VirtualMachineInstance) {
			for i, disk := range vmi.Spec.Domain.Devices.Disks {
				disk.DiskDevice.Disk.PciAddress = fmt.Sprintf("%s:02.%v", addrPrefix, fmt.Sprintf("%x", i+startIndex))
			}
		}

		BeforeEach(func() {
			var bootOrder uint = 1
			vmi = tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Resources.Requests[kubev1.ResourceMemory] = resource.MustParse("1024M")
			vmi.Spec.Domain.Devices.Disks[0].BootOrder = &bootOrder
		})

		DescribeTable("should configure custom pci address", func(startIndex, numOfDevices int, testingPciFunctions bool) {
			currentDisks := len(vmi.Spec.Domain.Devices.Disks)
			numOfDisksToAdd := numOfDevices - currentDisks

			createDisks(numOfDisksToAdd, vmi)
			if testingPciFunctions {
				assignDisksToFunctions(startIndex, vmi)
			} else {
				tests.DisableFeatureGate(virtconfig.ExpandDisksGate)
				assignDisksToSlots(startIndex, vmi)
			}
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
			Expect(vmi.Spec.Domain.Devices.Disks).Should(HaveLen(numOfDevices))

			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("[Serial][test_id:5269]across all available PCI root bus slots", Serial, 2, numOfSlotsToTest, false),
			Entry("[test_id:5270]across all available PCI functions of a single slot", 0, numOfFuncsToTest, true),
		)
	})

	Context("Check KVM CPUID advertisement", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			checks.SkipIfRunningOnKindInfra("Skip KVM MSR prescence test on kind")

			vmi = tests.NewRandomFedoraVMIWithEphemeralDiskHighMemory()
		})

		It("[test_id:5271]test cpuid hidden", func() {
			vmi.Spec.Domain.Features = &v1.Features{
				KVM: &v1.FeatureKVM{Hidden: true},
			}

			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Check values in domain XML")
			domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(domXml).To(ContainSubstring("<hidden state='on'/>"))

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check virt-what-cpuid-helper does not match KVM")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "/usr/libexec/virt-what-cpuid-helper > /dev/null 2>&1 && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "$(sudo /usr/libexec/virt-what-cpuid-helper | grep -q KVMKVMKVM) || echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 2*time.Second)).To(Succeed())
		})

		It("[test_id:5272]test cpuid default", func() {
			By("Starting a VirtualMachineInstance")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Check virt-what-cpuid-helper matches KVM")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "/usr/libexec/virt-what-cpuid-helper > /dev/null 2>&1 && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
				&expect.BSnd{S: "$(sudo /usr/libexec/virt-what-cpuid-helper | grep -q KVMKVMKVM) && echo 'pass'\n"},
				&expect.BExp{R: console.RetValue("pass")},
			}, 1*time.Second)).To(Succeed())
		})
	})
	Context("virt-launcher processes memory usage", func() {
		doesntExceedMemoryUsage := func(processRss *map[string]resource.Quantity, process string, memoryLimit resource.Quantity) {
			actual := (*processRss)[process]
			ExpectWithOffset(1, (&actual).Cmp(memoryLimit)).To(Equal(-1),
				"the %s process is taking too much RAM! (%s > %s). All processes: %v",
				process, actual.String(), memoryLimit.String(), processRss)
		}
		It("should be lower than allocated size", func() {
			By("Starting a VirtualMachineInstance")
			vmi := tests.NewRandomFedoraVMI()
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Expecting console")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Running ps in virt-launcher")
			pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
			})
			Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
			var stdout, stderr string
			errorMassageFormat := "failed after running the `ps` command with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
			Eventually(func() error {
				stdout, stderr, err = exec.ExecuteCommandOnPodWithResults(virtClient, &pods.Items[0], "compute",
					[]string{
						"ps",
						"--no-header",
						"axo",
						"rss,command",
					})
				return err
			}, time.Second, 50*time.Millisecond).Should(BeNil(), fmt.Sprintf(errorMassageFormat, stdout, stderr, err))

			By("Parsing the output of ps")
			processRss := make(map[string]resource.Quantity)
			scanner := bufio.NewScanner(strings.NewReader(stdout))
			for scanner.Scan() {
				fields := strings.Fields(scanner.Text())
				Expect(len(fields)).To(BeNumerically(">=", 2))
				rss := fields[0]
				command := filepath.Base(fields[1])
				// Handle the qemu binary: e.g. qemu-kvm or qemu-system-x86_64
				if command == "qemu-kvm" || strings.HasPrefix(command, "qemu-system-") {
					command = "qemu"
				}
				switch command {
				case "virt-launcher-monitor", "virt-launcher", "virtlogd", "virtqemud", "qemu":
					Expect(processRss).ToNot(HaveKey(command), "multiple %s processes found", command)
					value := resource.MustParse(rss + "Ki")
					processRss[command] = value
				}
			}
			for _, process := range []string{"virt-launcher-monitor", "virt-launcher", "virtlogd", "virtqemud", "qemu"} {
				Expect(processRss).To(HaveKey(process), "no %s process found", process)
			}

			By("Ensuring no process is using too much ram")
			doesntExceedMemoryUsage(&processRss, "virt-launcher-monitor", resource.MustParse(services.VirtLauncherMonitorOverhead))
			doesntExceedMemoryUsage(&processRss, "virt-launcher", resource.MustParse(services.VirtLauncherOverhead))
			doesntExceedMemoryUsage(&processRss, "virtlogd", resource.MustParse(services.VirtlogdOverhead))
			doesntExceedMemoryUsage(&processRss, "virtqemud", resource.MustParse(services.VirtqemudOverhead))
			qemuExpected := resource.MustParse(services.QemuOverhead)
			qemuExpected.Add(vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory])
			doesntExceedMemoryUsage(&processRss, "qemu", qemuExpected)
		})
	})

	Context("When topology spread constraints are defined for the VMI", func() {
		It("they should be applied to the launcher pod", func() {
			vmi := libvmi.NewCirros()
			tsc := []k8sv1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "zone",
					WhenUnsatisfiable: "DoNotSchedule",
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"foo": "bar",
						},
					},
				},
			}
			vmi.Spec.TopologySpreadConstraints = tsc

			By("Starting a VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectScheduling(vmi, 30)

			By("Ensuring that pod has expected topologySpreadConstraints")
			pod := tests.GetPodByVirtualMachineInstance(vmi)
			Expect(pod.Spec.TopologySpreadConstraints).To(Equal(tsc))
		})
	})
})
