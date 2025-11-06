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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdomain"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] Storage configuration", decorators.SigStorage, decorators.StorageReq, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	Context("Block size configuration set", func() {
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
	})
})
