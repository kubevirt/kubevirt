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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	v1 "kubevirt.io/api/core/v1"
	virtv1 "kubevirt.io/api/core/v1"
	virtstoragev1alpha1 "kubevirt.io/api/storage/v1alpha1"
	"kubevirt.io/client-go/kubecli"

	storagetypes "kubevirt.io/kubevirt/pkg/storage/types"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = SIGDescribe("Volume migration", func() {
	var virtClient kubecli.KubevirtClient
	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
	})
	waitForVolumeMigrationForVMIToComplete := func(smName, ns string, seconds int) {
		gomega.EventuallyWithOffset(1, func() bool {
			sm, err := virtClient.VolumeMigration(ns).Get(context.Background(), smName, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			return sm.Status.Phase == virtstoragev1alpha1.VolumeMigrationPhaseSucceeded ||
				sm.Status.Phase == virtstoragev1alpha1.VolumeMigrationPhaseFailed
		}, seconds, 1*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("The volume migration %s should be completed", smName))

	}

	checkVolumeMigrationFailed := func(smName, vmiName, ns string) bool {
		volMig, err := virtClient.VolumeMigration(ns).Get(context.Background(), smName, metav1.GetOptions{})
		Expect(err).ShouldNot(HaveOccurred())
		Expect(volMig).ShouldNot(BeNil())
		if volMig.Status.Phase == virtstoragev1alpha1.VolumeMigrationPhaseFailed {
			return true
		}
		return false

	}

	checkPVCVMI := func(vmi *virtv1.VirtualMachineInstance, claimName string, seconds int) {
		gomega.EventuallyWithOffset(1, func() bool {
			for _, v := range vmi.Spec.Volumes {
				name := storagetypes.PVCNameFromVirtVolume(&v)
				if name == claimName {
					return true
				}
			}
			return false
		}, seconds, 1*time.Second).Should(gomega.BeTrue(), fmt.Sprintf("The VMI %s should have the destination PVC %s", vmi.Name, claimName))
	}
	getVMIMigrationConditions := func(migName, ns string) string {
		var str strings.Builder
		mig, err := virtClient.VirtualMachineInstanceMigration(ns).Get(migName, &metav1.GetOptions{})
		if errors.IsNotFound(err) {
			return ""
		}
		Expect(err).ShouldNot(HaveOccurred())
		str.WriteString(fmt.Sprintf("VMI migrations %s conditions:\n", mig.Name))
		for _, c := range mig.Status.Conditions {
			str.WriteString(fmt.Sprintf("%s: %s: %s\n", c.Status, c.Reason, c.Message))
		}
		return str.String()

	}
	Describe("Creation of volume migration", func() {
		const addDiskPrefix = "disk"

		var (
			ns      string
			smName  string
			destPVC string
		)

		type modePVC int
		const (
			fsPVC modePVC = iota
			blockPVC
		)

		createSourceDestinationPVCcouples := func(n int, mode modePVC) map[string]string {
			disks := make(map[string]string)
			for i := 0; i < n; i++ {
				suffix := rand.String(5)
				src := fmt.Sprintf("%s-%s", addDiskPrefix, suffix)
				dst := fmt.Sprintf("destdisk-%s", suffix)
				switch mode {
				case fsPVC:
					libstorage.CreateFSPVC(src, ns, "500M", nil)
					libstorage.CreateFSPVC(dst, ns, "500M", nil)
				case blockPVC:
					libstorage.CreateBlockPVC(src, ns, "500M")
					libstorage.CreateBlockPVC(dst, ns, "500M")
				}
				disks[src] = dst
			}
			return disks
		}

		setSC := func(mode modePVC) string {
			var sc string
			var exists bool
			switch mode {
			case fsPVC:
				sc, exists = libstorage.GetRWOFileSystemStorageClass()
			case blockPVC:
				sc, exists = libstorage.GetRWOBlockStorageClass()
			}
			if !exists {
				Skip("Skip test when Filesystem storage is not present")
			}
			return sc
		}

		BeforeEach(func() {
			ns = testsuite.GetTestNamespace(nil)
			smName = "test-" + rand.String(5)
			destPVC = "dest-" + rand.String(5)

		})

		DescribeTable("volume migration with a single VM", func(noAddDisks int, mode modePVC) {
			sc := setSC(mode)
			// Additional disks
			addDisks := createSourceDestinationPVCcouples(noAddDisks, mode)

			// Create VM with a filesystem DV
			vmi, dataVolume := tests.NewRandomVirtualMachineInstanceWithDisk(
				cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros),
				ns, sc, k8sv1.ReadWriteOnce, k8sv1.PersistentVolumeFilesystem)
			tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
			By(fmt.Sprintf("Volume migration with %d disks", 1+len(addDisks)))
			for src, _ := range addDisks {
				libvmi.WithPersistentVolumeClaim(src, src)(vmi)
			}

			vmi = tests.RunVMIAndExpectLaunch(vmi, 500)

			By("Expecting the VirtualMachineInstance console")
			Expect(console.LoginToCirros(vmi)).To(Succeed())

			// Create dest PVC
			size := dataVolume.Spec.PVC.Resources.Requests.Storage()
			Expect(size).ShouldNot(BeNil())
			switch mode {
			case fsPVC:
				libstorage.CreateFSPVC(destPVC, ns, size.String(), nil)
			case blockPVC:
				libstorage.CreateBlockPVC(destPVC, ns, size.String())
			}

			// Create Storage Migration
			sm := virtstoragev1alpha1.VolumeMigration{
				ObjectMeta: metav1.ObjectMeta{Name: smName},
				Spec: virtstoragev1alpha1.VolumeMigrationSpec{
					MigratedVolume: []virtstoragev1alpha1.MigratedVolume{
						{
							SourceClaim:         dataVolume.Name,
							DestinationClaim:    destPVC,
							SourceReclaimPolicy: virtstoragev1alpha1.SourceReclaimPolicyDelete,
						},
					},
				},
			}
			for src, dst := range addDisks {
				sm.Spec.MigratedVolume = append(sm.Spec.MigratedVolume, virtstoragev1alpha1.MigratedVolume{
					SourceClaim:         src,
					DestinationClaim:    dst,
					SourceReclaimPolicy: virtstoragev1alpha1.SourceReclaimPolicyDelete,
				})
			}
			By("Creating the volume migration")
			_, err := virtClient.VolumeMigration(ns).Create(context.Background(), &sm, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Wait Storage Migration to complete
			waitForVolumeMigrationForVMIToComplete(smName, ns, 180)
			migName := sm.GetVirtualMachiheInstanceMigrationName(vmi.Name)
			Expect(checkVolumeMigrationFailed(smName, vmi.Name, ns)).To(BeFalse(), getVMIMigrationConditions(migName, ns))

			// Check status for the migration for the VMI
			vmi, err = virtClient.VirtualMachineInstance(ns).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			// Check the VMI is running after the migration
			Expect(vmi.Status.Phase).To(Equal(v1.Running))

			// Check if the source volume have been replaced with the
			// destination PVCs
			checkPVCVMI(vmi, destPVC, 90)
			for _, dst := range addDisks {
				checkPVCVMI(vmi, dst, 90)
			}
		},
			Entry("single filesystem volume", 0, fsPVC),
			Entry("multiple filesystem volumes", 2, fsPVC),
			Entry("single block volume", 0, blockPVC),
		)
	})
})
