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

package storage

import (
	"context"
	"fmt"
	"path/filepath"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] Storage configuration", decorators.SigStorage, decorators.StorageReq, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("volumes and disks validation (with volumes, disks and filesystem defined)", func() {
		It("[test_id:6960]should reject disk with missing volume", func() {
			vmi := libvmifact.NewGuestless()
			const diskName = "testdisk"
			vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
				Name: diskName,
			})
			_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).To(HaveOccurred())
			const expectedErrMessage = "denied the request: spec.domain.devices.disks[0].Name '" + diskName + "' not found."
			Expect(err.Error()).To(ContainSubstring(expectedErrMessage))
		})
	})

	Context("driver cache and io settings and PVC", func() {
		It("[test_id:1681]should set appropriate cache modes", decorators.HostDiskGate, func() {
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithContainerDisk("ephemeral-disk1", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
				libvmi.WithContainerDisk("ephemeral-disk2", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
				libvmi.WithContainerDisk("ephemeral-disk5", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
				libvmi.WithContainerDisk("ephemeral-disk3", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudUserData("#!/bin/bash\necho 'hello'\n")),
			)
			By("setting disk caches")
			// ephemeral-disk1
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone
			// ephemeral-disk2
			vmi.Spec.Domain.Devices.Disks[1].Cache = v1.CacheWriteThrough
			// ephemeral-disk5
			vmi.Spec.Domain.Devices.Disks[2].Cache = v1.CacheWriteBack

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			runningVMISpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

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
			Expect(disks[4].Alias.GetName()).To(Equal(libvmi.CloudInitDiskName))
			Expect(disks[4].Driver.Cache).To(Equal(cacheNone))
		})

		It("[test_id:5360]should set appropriate IO modes", decorators.RequiresBlockStorage, func() {
			By("Creating block Datavolume")
			sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
			if !foundSC {
				Fail("Block storage RWO is not present")
			}

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithBlockVolumeMode()),
			)
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))

			const alpineHostPath = "alpine-host-path"
			libstorage.CreateHostPathPv(alpineHostPath, testsuite.GetTestNamespace(nil), testsuite.HostPathAlpine)
			libstorage.CreateHostPathPVC(alpineHostPath, testsuite.GetTestNamespace(nil), "1Gi")
			vmi := libvmi.New(
				libvmi.WithMemoryRequest("128Mi"),
				// disk[0]
				libvmi.WithContainerDisk("ephemeral-disk1", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
				// disk[1]:  Block, no user-input, cache=none
				libvmi.WithPersistentVolumeClaim("block-pvc", dataVolume.Name),
				// disk[2]: File, not-sparsed, no user-input, cache=none
				libvmi.WithPersistentVolumeClaim("hostpath-pvc", fmt.Sprintf("disk-%s", alpineHostPath)),
				// disk[3]
				libvmi.WithContainerDisk("ephemeral-disk2", cd.ContainerDiskFor(cd.ContainerDiskCirros)),
			)
			// disk[0]:  File, sparsed, no user-input, cache=none
			vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheNone
			// disk[3]:  File, sparsed, user-input=threads, cache=none
			vmi.Spec.Domain.Devices.Disks[3].Cache = v1.CacheNone
			vmi.Spec.Domain.Devices.Disks[3].IO = v1.IOThreads

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			runningVMISpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
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

		It("[test_id:6965]Should set BlockIO when using custom block sizes", decorators.RequiresBlockStorage, func() {
			sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
			if !foundSC {
				Fail(`Block storage is not present. You can filter by "RequiresBlockStorage" label`)
			}

			By("creating a block volume")
			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithBlockVolumeMode()),
			)
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))

			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithPersistentVolumeClaim("disk0", dataVolume.Name),
				libvmi.WithMemoryRequest("128Mi"),
			)

			By("setting the disk to use custom block sizes")
			logicalSize := uint(16384)
			physicalSize := uint(16384)
			discardGranularity := uint(16384)
			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{
				Custom: &v1.CustomBlockSize{
					Logical:            logicalSize,
					Physical:           physicalSize,
					DiscardGranularity: &discardGranularity,
				},
			}

			By("initializing the VM")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			runningVMISpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			By("checking if number of attached disks is equal to real disks number")
			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			By("checking if BlockIO is set to the custom block size")
			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(logicalSize))
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(Equal(physicalSize))
			Expect(disks[0].BlockIO.DiscardGranularity).To(Equal(&discardGranularity))
		})

		It("[test_id:6966]Should set BlockIO when set to match volume block sizes on block devices", decorators.RequiresBlockStorage, func() {
			sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteOnce)
			if !foundSC {
				Fail(`Block storage is not present. You can skip by "RequiresBlockStorage" label`)
			}

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithStorage(libdv.StorageWithStorageClass(sc), libdv.StorageWithBlockVolumeMode()),
			)
			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), WaitForFirstConsumer()))

			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithPersistentVolumeClaim("disk0", dataVolume.Name),
				libvmi.WithMemoryRequest("128Mi"),
			)

			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{MatchVolume: &v1.FeatureState{}}

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			runningVMISpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			Expect(disks[0].BlockIO.LogicalBlockSize).To(SatisfyAny(Equal(uint(512)), Equal(uint(4096))))
			Expect(disks[0].BlockIO.PhysicalBlockSize).To(SatisfyAny(Equal(uint(512)), Equal(uint(4096))))
			if discard := disks[0].BlockIO.DiscardGranularity; discard != nil {
				Expect(*discard%disks[0].BlockIO.LogicalBlockSize).To(Equal(uint(0)),
					"Discard granularity must align with logical block size")
			}

		})

		It("[test_id:6967]Should set BlockIO when set to match volume block sizes on files", decorators.HostDiskGate, func() {
			var nodeName string
			tmpHostDiskDir := RandHostDiskDir()
			tmpHostDiskPath := filepath.Join(tmpHostDiskDir, fmt.Sprintf("disk-%s.img", uuid.NewString()))

			pod := CreateHostDisk(tmpHostDiskPath)
			pod = runPodAndExpectPhase(pod, k8sv1.PodSucceeded)
			nodeName = pod.Spec.NodeName
			defer func() { Expect(RemoveHostDisk(tmpHostDiskDir, nodeName)).To(Succeed()) }()

			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithHostDisk("host-disk", tmpHostDiskPath, v1.HostDiskExists),
				libvmi.WithNodeAffinityFor(nodeName),
				libvmi.WithNamespace(testsuite.NamespacePrivileged),
			)

			vmi.Spec.Domain.Devices.Disks[0].BlockSize = &v1.BlockSize{MatchVolume: &v1.FeatureState{}}

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			runningVMISpec, err := libdomain.GetRunningVMIDomainSpec(vmi)
			Expect(err).ToNot(HaveOccurred())

			disks := runningVMISpec.Devices.Disks
			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(len(disks)))

			Expect(disks[0].Alias.GetName()).To(Equal("host-disk"))
			Expect(disks[0].BlockIO).ToNot(BeNil())
			Expect(disks[0].BlockIO.LogicalBlockSize).ToNot(BeZero())
			Expect(disks[0].BlockIO.LogicalBlockSize).To(Equal(disks[0].BlockIO.PhysicalBlockSize))
		})
	})

	Context("virtio queues", func() {
		It("[test_id:1664]should map cores to virtio block queues", Serial, func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithMemoryRequest("128Mi"),
				libvmi.WithCPURequest("3"),
			)
			vmi.Spec.Domain.Devices.BlockMultiQueue = pointer.P(true)

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /sys/block/vda/mq | wc -l\n"},
				&expect.BExp{R: console.RetValue("3")},
			}, 15)).To(Succeed())
		})

		It("[test_id:1667]should not enforce explicitly rejected virtio block queues without cores", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithMemoryRequest("128Mi"),
			)
			vmi.Spec.Domain.Resources = v1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}
			vmi.Spec.Domain.Devices.BlockMultiQueue = pointer.P(false)

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "ls /sys/block/vda/mq | wc -l\n"},
				&expect.BExp{R: console.RetValue("1")},
			}, 15)).To(Succeed())
		})
	})

})
