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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package storage

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	util2 "kubevirt.io/kubevirt/tests/util"
)

var _ = SIGDescribe("Volumes update with migration", func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
		originalKv := util2.GetCurrentKv(virtClient)
		updateStrategy := &v1.KubeVirtWorkloadUpdateStrategy{
			WorkloadUpdateMethods: []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate},
		}
		rolloutStrategy := pointer.P(v1.VMRolloutStrategyLiveUpdate)
		tests.PatchWorkloadUpdateMethodAndRolloutStrategy(originalKv.Name, virtClient, updateStrategy, rolloutStrategy)

		currentKv := util2.GetCurrentKv(virtClient)
		tests.WaitForConfigToBePropagatedToComponent(
			"kubevirt.io=virt-controller",
			currentKv.ResourceVersion,
			tests.ExpectResourceVersionToBeLessEqualThanConfigVersion,
			time.Minute)
	})

	Describe("Update volumes with the migration updateVolumesStrategy", func() {
		const addDiskPrefix = "disk"

		var (
			ns      string
			destPVC string
		)
		const (
			fsPVC    = "filesystem"
			blockPVC = "block"
		)

		setSC := func(mode string) string {
			var sc string
			var exists bool
			switch mode {
			case fsPVC:
				sc, exists = libstorage.GetRWOFileSystemStorageClass()
			case blockPVC:
				sc, exists = libstorage.GetRWOBlockStorageClass()
			}
			if !exists {
				Skip(fmt.Sprintf("Skip test when %s storage is not present", mode))
			}
			return sc
		}

		waitForMigrationToSucceed := func(vmiName, ns string) {
			Eventually(func() bool {
				ls := labels.Set{
					virtv1.VolumesUpdateMigration: vmiName,
				}
				migList, err := virtClient.VirtualMachineInstanceMigration(ns).List(
					&metav1.ListOptions{
						LabelSelector: ls.String(),
					})
				Expect(err).ToNot(HaveOccurred())
				if len(migList.Items) < 0 {
					return false
				}
				vmi, err := virtClient.VirtualMachineInstance(ns).Get(context.Background(), vmiName,
					&metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if vmi.Status.MigrationState == nil {
					return false
				}
				if !vmi.Status.MigrationState.Completed {
					return false
				}
				Expect(vmi.Status.MigrationState.Failed).To(BeFalse())
				return true
			}, 120*time.Second, time.Second).Should(BeTrue())
		}

		BeforeEach(func() {
			ns = testsuite.GetTestNamespace(nil)
			destPVC = "dest-" + rand.String(5)

		})

		DescribeTable("volume migration with a single VM", func(mode string) {
			sc := setSC(mode)
			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros)),
				libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
			)
			_, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(ns).Create(context.Background(),
				dv, metav1.CreateOptions{})
			vmi := libvmi.New(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithDataVolume("disk0", dv.Name),
				libvmi.WithCloudInitNoCloudEncodedUserData(("#!/bin/bash\necho hello\n")),
			)
			vmi.Namespace = ns
			vm := libvmi.NewVirtualMachine(vmi,
				libvmi.WithRunning(),
				libvmi.WithDataVolumeTemplate(dv),
			)
			vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm)
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVM(vm), 360*time.Second, 1*time.Second).Should(beReady())
			libwait.WaitForSuccessfulVMIStart(vmi)

			// Create dest PVC
			size := dv.Spec.PVC.Resources.Requests.Storage()
			Expect(size).ShouldNot(BeNil())
			switch mode {
			case fsPVC:
				libstorage.CreateFSPVC(destPVC, ns, size.String(), nil)
			case blockPVC:
				libstorage.CreateBlockPVC(destPVC, ns, size.String())
			}
			By("Update volumes")
			vm, err = virtClient.VirtualMachine(ns).Get(context.Background(), vm.Name, &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Remove datavolume templates
			vm.Spec.DataVolumeTemplates = []virtv1.DataVolumeTemplateSpec{}
			// Replace dst pvc
			vm.Spec.Template.Spec.Volumes[0].VolumeSource.PersistentVolumeClaim = &virtv1.PersistentVolumeClaimVolumeSource{
				PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
					ClaimName: destPVC,
				},
			}
			vm.Spec.Template.Spec.Volumes[0].VolumeSource.DataVolume = nil
			vm.Spec.UpdateVolumesStrategy = pointer.P(virtv1.UpdateVolumesStrategyMigration)
			vm, err = virtClient.VirtualMachine(ns).Update(context.Background(), vm)
			Expect(err).ShouldNot(HaveOccurred())
			// wait VirtualMachineInstanceMigration to complete
			waitForMigrationToSucceed(vmi.Name, ns)
		},
			Entry("single filesystem volume", fsPVC),
			Entry("single block volume", blockPVC),
		)
	})
})

func beReady() gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Ready": BeTrue(),
		}),
	}))
}
