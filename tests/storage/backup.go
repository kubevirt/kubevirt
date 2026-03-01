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
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"

	backupapi "kubevirt.io/api/backup"
	backupv1 "kubevirt.io/api/backup/v1alpha1"
	"kubevirt.io/api/core"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	backendstorage "kubevirt.io/kubevirt/pkg/storage/backend-storage"
	backup "kubevirt.io/kubevirt/pkg/storage/cbt"
	"kubevirt.io/kubevirt/pkg/storage/velero"

	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Backup", func() {
	var (
		err        error
		virtClient kubecli.KubevirtClient
		vm         *v1.VirtualMachine
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	DescribeTable("Full Backup with source VirtualMachine", func(pvcSize string, expectedBackupCount int) {
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		targetPVC := libstorage.CreateFSPVC("target-pvc", testsuite.GetTestNamespace(vm), pvcSize, libstorage.WithStorageProfile())

		By("Creating the backup")
		backupName := backupName(vm.Name)
		createAndVerifyFullVMBackup(virtClient, backupName, vm.Name, vm.Namespace, targetPVC.Name, waitBackupSucceeded)
		if expectedBackupCount > 1 {
			By("Deleting the backup")
			deleteVMBackup(virtClient, vm.Namespace, backupName)
			By("Creating another backup")
			createAndVerifyFullVMBackup(virtClient, backupName, vm.Name, vm.Namespace, targetPVC.Name, waitBackupSucceeded)
		}
		expectedDiskSize := resource.MustParse(cd.AlpineVolumeSize)
		expectedDiskSizes := []int64{expectedDiskSize.Value()}
		verifyBackupTargetPVCOutput(virtClient, targetPVC, vm.Name, expectedBackupCount, expectedDiskSizes)
	},
		Entry("should succeed", getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), 1),
		Entry("2 backups to the same PVC should succeed", getDoubleTargetPVCSize(cd.AlpineVolumeSize), 2),
	)

	It("[QUARANTINE]Full and Incremental Backup with BackupTracker", func() {
		const (
			testDataSizeMB    = 50
			testDataSizeBytes = testDataSizeMB * 1024 * 1024
		)

		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm.Spec.Template.ObjectMeta.Annotations["kubevirt.io/libvirt-log-filters"] = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		fullBackupPVC := libstorage.CreateFSPVC("full-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())
		incrementalBackupPVC := libstorage.CreateFSPVC("incremental-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())

		expectedDiskSize := resource.MustParse(cd.AlpineVolumeSize)

		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating first full backup with tracker reference")
		fullBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, fullBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(fullBackup.Status.Type).To(Equal(backupv1.Full), "First backup should be Full")
		Expect(fullBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(fullBackup.Status.IncludedVolumes).To(HaveLen(1), "Should have one included volume")

		By("Verifying full backup size matches disk size")
		verifyBackupTargetPVCOutput(virtClient, fullBackupPVC, vm.Name, 1, []int64{expectedDiskSize.Value()})

		By("Verifying BackupTracker was updated with first checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		firstCheckpoint := tracker.Status.LatestCheckpoint
		Expect(firstCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after first backup")
		Expect(firstCheckpoint.Name).To(Equal(*fullBackup.Status.CheckpointName), "First checkpoint should match backup checkpoint")
		Expect(firstCheckpoint.CreationTime).ToNot(BeNil())

		By(fmt.Sprintf("Writing %dMB of data to VM disk before incremental backup", testDataSizeMB))
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should be able to login to Alpine VM")
		// Write random data to root home directory (on disk) not /tmp (which is tmpfs/RAM)
		// Use /dev/urandom instead of /dev/zero to ensure data is actually written to QCOW2 (not optimized as sparse zeros)
		err = console.RunCommand(vmi, fmt.Sprintf("dd if=/dev/urandom of=/root/testfile bs=1M count=%d && sync", testDataSizeMB), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Creating second incremental backup with same tracker reference")
		incrementalBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, incrementalBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(incrementalBackup.Status.Type).To(Equal(backupv1.Incremental), "Second backup should be Incremental")
		Expect(incrementalBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(incrementalBackup.Status.IncludedVolumes).To(HaveLen(1), "Should have one included volume")

		By("Verifying BackupTracker was updated with new checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after second backup")
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(*incrementalBackup.Status.CheckpointName), "Second checkpoint should match backup checkpoint")
		Expect(tracker.Status.LatestCheckpoint.Name).ToNot(Equal(firstCheckpoint.Name), "Second checkpoint should have a different name")
		Expect(tracker.Status.LatestCheckpoint.CreationTime).ToNot(BeNil())
		Expect(tracker.Status.LatestCheckpoint.CreationTime).ToNot(Equal(firstCheckpoint.CreationTime))

		By("Verifying incremental backup size matches the amount of data written")
		// Expected size should be around the amount of data we wrote
		verifyBackupTargetPVCOutput(virtClient, incrementalBackupPVC, vm.Name, 1, []int64{testDataSizeBytes})
	})

	FIt("Full and Incremental Backup with 2 disks", MustPassRepeatedly(100), func() {
		const (
			testDataSizeMB    = 50
			testDataSizeBytes = testDataSizeMB * 1024 * 1024
			blankDiskSize     = "256Mi"
		)

		bootDv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)

		blankDv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(blankDiskSize),
			),
		)

		vm = libstorage.RenderVMWithDataVolumeTemplate(bootDv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm.Spec.Template.ObjectMeta.Annotations["kubevirt.io/libvirt-log-filters"] = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

		libstorage.AddDataVolumeTemplate(vm, blankDv)
		libstorage.AddDataVolume(vm, "disk1", blankDv)

		By(fmt.Sprintf("Creating VM %s with 2 disks", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		// Calculate PVC sizes for full and incremental backups
		totalSize := resource.MustParse(cd.AlpineVolumeSize)
		blankSize := resource.MustParse(blankDiskSize)
		totalSize.Add(blankSize)
		fullBackupPVCSize := getTargetPVCSizeWithOverhead(totalSize.String())

		fullBackupPVC := libstorage.CreateFSPVC("full-backup-pvc", testsuite.GetTestNamespace(vm), fullBackupPVCSize, libstorage.WithStorageProfile())
		incrementalBackupPVC := libstorage.CreateFSPVC("incremental-backup-pvc", testsuite.GetTestNamespace(vm), fullBackupPVCSize, libstorage.WithStorageProfile())

		expectedBootDiskSize := resource.MustParse(cd.AlpineVolumeSize)
		expectedBlankDiskSize := resource.MustParse(blankDiskSize)

		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating first full backup with tracker reference")
		fullBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, fullBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(fullBackup.Status.Type).To(Equal(backupv1.Full), "First backup should be Full")
		Expect(fullBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(fullBackup.Status.IncludedVolumes).To(HaveLen(2), "Should have two included volumes")

		By("Verifying full backup has 2 qcow2 files with correct sizes")
		expectedDiskSizes := []int64{expectedBootDiskSize.Value(), expectedBlankDiskSize.Value()}
		verifyBackupTargetPVCOutput(virtClient, fullBackupPVC, vm.Name, 1, expectedDiskSizes)

		By("Verifying BackupTracker was updated with first checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		firstCheckpoint := tracker.Status.LatestCheckpoint
		Expect(firstCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after first backup")
		Expect(firstCheckpoint.Name).To(Equal(*fullBackup.Status.CheckpointName), "First checkpoint should match backup checkpoint")

		By(fmt.Sprintf("Writing %dMB of data to boot disk before incremental backup", testDataSizeMB))
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should be able to login to Alpine VM")

		// Write random data to root home directory on boot disk
		err = console.RunCommand(vmi, fmt.Sprintf("dd if=/dev/urandom of=/root/testfile bs=1M count=%d && sync", testDataSizeMB), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Writing data directly to second disk")
		// Write random data directly to the raw block device (no formatting needed)
		err = console.RunCommand(vmi, fmt.Sprintf("dd if=/dev/urandom of=/dev/vdb bs=1M count=%d && sync", testDataSizeMB), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Creating second incremental backup with same tracker reference")
		incrementalBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, incrementalBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(incrementalBackup.Status.Type).To(Equal(backupv1.Incremental), "Second backup should be Incremental")
		Expect(incrementalBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(incrementalBackup.Status.IncludedVolumes).To(HaveLen(2), "Should have two included volumes")

		By("Verifying BackupTracker was updated with new checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after second backup")
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(*incrementalBackup.Status.CheckpointName), "Second checkpoint should match backup checkpoint")
		Expect(tracker.Status.LatestCheckpoint.Name).ToNot(Equal(firstCheckpoint.Name), "Second checkpoint should have a different name")

		By("Verifying incremental backup has 2 qcow2 files with sizes matching data written")
		// Both disks should have approximately testDataSizeBytes of changed data
		incrementalExpectedSizes := []int64{testDataSizeBytes, testDataSizeBytes}
		verifyBackupTargetPVCOutput(virtClient, incrementalBackupPVC, vm.Name, 1, incrementalExpectedSizes)
	})

	It("Incremental Backup after VM shutdown and restart", func() {
		const (
			testDataSizeMB    = 50
			testDataSizeBytes = testDataSizeMB * 1024 * 1024
		)

		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm.Spec.Template.ObjectMeta.Annotations["kubevirt.io/libvirt-log-filters"] = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		fullBackupPVC := libstorage.CreateFSPVC("full-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())
		incrementalBackupPVC := libstorage.CreateFSPVC("incremental-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())

		expectedDiskSize := resource.MustParse(cd.AlpineVolumeSize)

		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating first full backup with tracker reference")
		fullBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, fullBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(fullBackup.Status.Type).To(Equal(backupv1.Full), "First backup should be Full")
		Expect(fullBackup.Status.CheckpointName).ToNot(BeNil())

		By("Verifying full backup size matches disk size")
		verifyBackupTargetPVCOutput(virtClient, fullBackupPVC, vm.Name, 1, []int64{expectedDiskSize.Value()})

		By("Verifying BackupTracker was updated with first checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		firstCheckpoint := tracker.Status.LatestCheckpoint
		Expect(firstCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after first backup")
		Expect(firstCheckpoint.Name).To(Equal(*fullBackup.Status.CheckpointName), "First checkpoint should match backup checkpoint")
		Expect(firstCheckpoint.Volumes).ToNot(BeEmpty(), "Checkpoint should have disk info for redefinition")

		By(fmt.Sprintf("Writing %dMB of data to VM disk before shutdown", testDataSizeMB))
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should be able to login to Alpine VM")
		err = console.RunCommand(vmi, fmt.Sprintf("dd if=/dev/urandom of=/root/testfile bs=1M count=%d && sync", testDataSizeMB), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Listing checkpoints before VM stop")
		checkpointsBeforeStop := listDomainCheckpoints(vmi)
		Expect(checkpointsBeforeStop).To(HaveLen(1), "Should have exactly one checkpoint before stop")
		Expect(checkpointsBeforeStop[0]).To(Equal(firstCheckpoint.Name), "Checkpoint name should match tracker checkpoint")

		By("Stopping the VM gracefully")
		err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VMI to be deleted (VM stopped)")
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			return err
		}, 180*time.Second, 2*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"),
			"VMI should be deleted after VM stop")

		By("Starting the VM again")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for VM to be running and guest agent connected")
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		By("Verifying BackupTracker still has the checkpoint after VM restart")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil(), "Tracker should still have checkpoint after VM restart")
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(firstCheckpoint.Name), "Checkpoint should be the same after restart")

		By("Verifying checkpoint was redefined in libvirt after VM restart")
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		checkpointsAfterRestart := listDomainCheckpoints(vmi)
		Expect(checkpointsAfterRestart).To(Equal(checkpointsBeforeStop),
			"Checkpoint should be redefined with the same name after VM restart")

		By("Creating second backup after VM restart - this should be incremental")
		incrementalBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, incrementalBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(incrementalBackup.Status.Type).To(Equal(backupv1.Incremental),
			"Backup after VM restart should be Incremental (checkpoint was redefined)")
		Expect(incrementalBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(incrementalBackup.Status.IncludedVolumes).To(HaveLen(1), "Should have one included disk")

		By("Verifying BackupTracker was updated with new checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after second backup")
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(*incrementalBackup.Status.CheckpointName), "Second checkpoint should match backup checkpoint")
		Expect(tracker.Status.LatestCheckpoint.Name).ToNot(Equal(firstCheckpoint.Name), "Second checkpoint should have a different name")

		By("Verifying incremental backup size matches the amount of data written")
		verifyBackupTargetPVCOutput(virtClient, incrementalBackupPVC, vm.Name, 1, []int64{testDataSizeBytes})
	})

	It("Backup falls back to Full when checkpoint is corrupted", func() {
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm.Spec.Template.ObjectMeta.Annotations["kubevirt.io/libvirt-log-filters"] = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		fullBackupPVC := libstorage.CreateFSPVC("full-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())
		secondBackupPVC := libstorage.CreateFSPVC("second-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())

		expectedDiskSize := resource.MustParse(cd.AlpineVolumeSize)

		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating first full backup")
		fullBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, fullBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(fullBackup.Status.Type).To(Equal(backupv1.Full), "First backup should be Full")
		Expect(fullBackup.Status.CheckpointName).ToNot(BeNil())
		checkpointName := *fullBackup.Status.CheckpointName

		By("Verifying BackupTracker has checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil())
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(checkpointName))

		By("Stopping the VM to access the disk")
		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		pvcName := backendstorage.CurrentPVCName(vmi)
		Expect(pvcName).ToNot(BeEmpty(), "Backend storage PVC name should not be empty")
		volumeName := vm.Spec.Template.Spec.Volumes[0].Name
		cbtOverlayPath := fmt.Sprintf("/cbt/%s.qcow2", volumeName)

		err = virtClient.VirtualMachine(vm.Namespace).Stop(context.Background(), vm.Name, &v1.StopOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(func() error {
			_, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
			return err
		}, 180*time.Second, 2*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

		By("Corrupting the checkpoint by deleting the QCOW2 overlay")
		corruptBitmap(virtClient, vm.Namespace, pvcName, cbtOverlayPath)

		By("Starting the VM again")
		err = virtClient.VirtualMachine(vm.Namespace).Start(context.Background(), vm.Name, &v1.StartOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		By("Verifying CheckpointRedefinitionFailed event was emitted on the tracker")
		events.ExpectEvent(tracker, corev1.EventTypeWarning, "CheckpointRedefinitionFailed")

		By("Verifying checkpoint redefinition failed and checkpoint was cleared")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).To(BeNil(),
			"BackupTracker checkpoint should be cleared after bitmap corruption detected")

		By("Creating second backup - should be Full since checkpoint was corrupted")
		secondBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, secondBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(secondBackup.Status.Type).To(Equal(backupv1.Full),
			"Backup should fall back to Full when checkpoint was corrupted")

		By("Verifying second backup is a full backup")
		verifyBackupTargetPVCOutput(virtClient, secondBackupPVC, vm.Name, 1, []int64{expectedDiskSize.Value()})
	})

	It("[QUARANTINE] Checkpoint redefinition succeeds after hotplug volume removal", decorators.Quarantine, func() {
		const (
			testDataSizeMB    = 50
			testDataSizeBytes = testDataSizeMB * 1024 * 1024
			bootDiskName      = "disk0"
			hotplugDiskSize   = "256Mi"
			hotplugDiskName   = "hotplug-disk"
		)

		By("Creating boot disk DataVolume")
		bootDv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)

		By("Creating VM with boot disk and CBT enabled")
		vm = libstorage.RenderVMWithDataVolumeTemplate(bootDv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)
		vm.Spec.Template.ObjectMeta.Annotations["kubevirt.io/libvirt-log-filters"] = "3:remote 4:event 3:util.json 3:util.object 3:util.dbus 3:util.netlink 3:node_device 3:rpc 3:access 1:*"

		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		By("Creating hotplug DataVolume")
		hotplugDv := libdv.NewDataVolume(
			libdv.WithBlankImageSource(),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(hotplugDiskSize),
			),
		)
		hotplugDv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(hotplugDv.Namespace).Create(context.Background(), hotplugDv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Waiting for hotplug DataVolume to be ready")
		libstorage.EventuallyDV(hotplugDv, 240, matcher.HaveSucceeded())

		By("Hotplugging volume to running VM")
		hotplugVolumeName := "hotplug-volume"
		vm = libstorage.AddHotplugDiskAndVolume(virtClient, vm, hotplugVolumeName, hotplugDv.Name)

		By("Waiting for hotplug volume to be ready")
		libstorage.WaitForHotplugToComplete(virtClient, vm, hotplugVolumeName, hotplugDv.Name, true)

		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		totalSize := resource.MustParse(cd.AlpineVolumeSize)
		hotplugSize := resource.MustParse(hotplugDiskSize)
		totalSize.Add(hotplugSize)
		fullBackupPVCSize := getTargetPVCSizeWithOverhead(totalSize.String())

		fullBackupPVC := libstorage.CreateFSPVC("full-backup-pvc", testsuite.GetTestNamespace(vm), fullBackupPVCSize, libstorage.WithStorageProfile())

		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating full backup with both boot disk and hotplug volume")
		fullBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, fullBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(fullBackup.Status.Type).To(Equal(backupv1.Full), "First backup should be Full")
		Expect(fullBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(fullBackup.Status.IncludedVolumes).To(HaveLen(2), "Should have two included volumes (boot + hotplug)")

		By("Verifying BackupTracker has checkpoint with 2 volumes")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		firstCheckpoint := tracker.Status.LatestCheckpoint
		Expect(firstCheckpoint).ToNot(BeNil(), "Tracker should have checkpoint after first backup")
		Expect(firstCheckpoint.Volumes).To(HaveLen(2), "Checkpoint should have 2 volumes")

		By("Getting disk targets from VMI volume status")
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		var bootDiskTarget, hotplugDiskTarget string
		var allDiskTargets []string
		for _, volStatus := range vmi.Status.VolumeStatus {
			if volStatus.Target == "" {
				continue
			}
			if volStatus.Name == hotplugVolumeName {
				hotplugDiskTarget = volStatus.Target
			}
			if volStatus.Name == bootDiskName {
				bootDiskTarget = volStatus.Target
			}
			allDiskTargets = append(allDiskTargets, volStatus.Target)
		}
		Expect(bootDiskTarget).ToNot(BeEmpty(), "Boot disk target should be found in VMI status")
		Expect(hotplugDiskTarget).ToNot(BeEmpty(), "Hotplug disk target should be found in VMI status")

		By("Verifying tracker checkpoint has matching disk targets")
		trackerTargets := make(map[string]bool)
		for _, vol := range firstCheckpoint.Volumes {
			trackerTargets[vol.DiskTarget] = true
		}
		Expect(trackerTargets).To(HaveKey(bootDiskTarget), "Tracker checkpoint should have boot disk target")
		Expect(trackerTargets).To(HaveKey(hotplugDiskTarget), "Tracker checkpoint should have hotplug disk target")

		By("Listing checkpoints before VM stop")
		checkpointsBeforeStop := listDomainCheckpoints(vmi)
		Expect(checkpointsBeforeStop).To(HaveLen(1), "Should have exactly one checkpoint before stop")

		By("Verifying libvirt checkpoint has bitmaps for both boot and hotplug disks")
		expectCheckpointDisks(vmi, firstCheckpoint.Name, allDiskTargets, nil)

		By("Writing data to boot disk")
		Expect(console.LoginToAlpine(vmi)).To(Succeed(), "Should be able to login to Alpine VM")
		err = console.RunCommand(vmi, fmt.Sprintf("dd if=/dev/urandom of=/root/testfile bs=1M count=%d && sync", testDataSizeMB), 2*time.Minute)
		Expect(err).ToNot(HaveOccurred())

		By("Removing the hotplug volume from VM")
		vm = libstorage.RemoveHotplugDiskAndVolume(virtClient, vm, hotplugVolumeName)
		libstorage.WaitForHotplugToComplete(virtClient, vm, hotplugVolumeName, hotplugDv.Name, false)

		By("Restarting the VM")
		oldVMIUID := vmi.UID
		err = virtClient.VirtualMachine(vm.Namespace).Restart(context.Background(), vm.Name, &v1.RestartOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMI(vmi), 240*time.Second, time.Second).Should(matcher.BeRestarted(oldVMIUID))
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		By("Verifying checkpoint redefinition succeeded")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil(),
			"Checkpoint should not change after VM restart")
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(firstCheckpoint.Name),
			"Checkpoint name should be the same after restart")
		Expect(tracker.Status.LatestCheckpoint.Volumes).To(Equal(firstCheckpoint.Volumes),
			"Checkpoint volumes should be the same after restart even if one of them is not redefined")

		By("Verifying checkpoint was redefined in libvirt with remaining disk")
		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		checkpointsAfterRestart := listDomainCheckpoints(vmi)
		Expect(checkpointsAfterRestart).To(HaveLen(1),
			"Should still have one checkpoint after restart")
		Expect(checkpointsAfterRestart[0]).To(Equal(firstCheckpoint.Name),
			"Checkpoint should be redefined with the same name")

		By("Verifying libvirt checkpoint only has bitmap for boot disk")
		expectCheckpointDisks(vmi, firstCheckpoint.Name, []string{bootDiskTarget}, []string{hotplugDiskTarget})

		By("Creating incremental backup after VM restart")
		incrementalBackupPVC := libstorage.CreateFSPVC("incremental-backup-pvc", testsuite.GetTestNamespace(vm), getTargetPVCSizeWithOverhead(cd.AlpineVolumeSize), libstorage.WithStorageProfile())
		incrementalBackup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), vm.Namespace, incrementalBackupPVC.Name, tracker.Name, waitBackupSucceeded)
		Expect(incrementalBackup.Status.Type).To(Equal(backupv1.Incremental),
			"Backup after VM restart should be Incremental (checkpoint was redefined with remaining disk)")
		Expect(incrementalBackup.Status.CheckpointName).ToNot(BeNil())
		Expect(incrementalBackup.Status.IncludedVolumes).To(HaveLen(1),
			"Should have one included volume")

		By("Verifying BackupTracker was updated with new checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status.LatestCheckpoint).ToNot(BeNil())
		Expect(tracker.Status.LatestCheckpoint.Name).To(Equal(*incrementalBackup.Status.CheckpointName))
		Expect(tracker.Status.LatestCheckpoint.Name).ToNot(Equal(firstCheckpoint.Name),
			"Second checkpoint should have a different name")
	})

	It("Should handle backup failure due to insufficient target PVC size", func() {
		dv := libdv.NewDataVolume(
			libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpineTestTooling)),
			libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			libdv.WithStorage(
				libdv.StorageWithVolumeSize(cd.AlpineVolumeSize),
			),
		)
		vm = libstorage.RenderVMWithDataVolumeTemplate(dv,
			libvmi.WithLabels(backup.CBTLabel),
			libvmi.WithRunStrategy(v1.RunStrategyAlways),
		)

		By(fmt.Sprintf("Creating VM %s", vm.Name))
		vm, err = virtClient.VirtualMachine(vm.Namespace).Create(context.Background(), vm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
		libstorage.WaitForCBTEnabled(virtClient, vm.Namespace, vm.Name)

		smallFSDv := libdv.NewDataVolume(
			libdv.WithNamespace(vm.Namespace),
			libdv.WithForceBindAnnotation(),
			libdv.WithBlankImageSource(),
			libdv.WithStorage(
				libdv.StorageWithFilesystemVolumeMode(),
				libdv.StorageWithAccessMode(corev1.ReadWriteOnce),
				libdv.StorageWithVolumeSize(cd.BlankVolumeSize),
				libdv.StorageWithStorageClass(libstorage.Config.StorageClassCSI),
			),
		)
		smallFSDv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(smallFSDv.Namespace).Create(context.Background(), smallFSDv, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())

		libstorage.EventuallyDV(smallFSDv, 180, matcher.HaveSucceeded())
		By("Creating BackupTracker")
		tracker := createBackupTracker(virtClient, vm)

		By("Creating full backup and wait for it to fail")
		backup := createAndVerifyBackupWithTracker(virtClient, backupName(vm.Name), tracker.Namespace, smallFSDv.Name, tracker.Name, waitBackupFailed)
		Expect(backup).ToNot(BeNil())
		for _, cond := range backup.Status.Conditions {
			if cond.Type == backupv1.ConditionDone {
				Expect(cond.Reason).To(ContainSubstring("No space left on device"))
			}
		}

		By("Verifying BackupTracker was not updated with a checkpoint")
		tracker, err = virtClient.VirtualMachineBackupTracker(tracker.Namespace).Get(context.Background(), tracker.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(tracker.Status).To(BeNil())
	})

	Context("Velero backup hooks injection", Serial, func() {
		It("should dynamically sync hooks annotations based on KubeVirt CR annotation", func() {
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)))

			By("Creating VMI without skip-backup-hooks annotation")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 300*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))

			By("Verifying launcher pod has Velero backup hooks annotations")
			pod := getPodByVMI(vmi)
			Expect(pod.Annotations).To(HaveKey(velero.PreBackupHookContainerAnnotation))

			kv := libkubevirt.GetCurrentKv(virtClient)
			originalKvAnnotations := kv.Annotations
			if originalKvAnnotations == nil {
				originalKvAnnotations = make(map[string]string)
			}
			_, hadSkipAnnotation := originalKvAnnotations[velero.SkipHooksAnnotation]

			By("Adding skip-backup-hooks annotation to KubeVirt CR")
			patchData := fmt.Appendf(nil, `{"metadata":{"annotations":{%q:"true"}}}`, velero.SkipHooksAnnotation)
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying launcher pod Velero annotations are removed")
			Eventually(func() bool {
				pod = getPodByVMI(vmi)
				_, hasPreHook := pod.Annotations[velero.PreBackupHookContainerAnnotation]
				_, hasPostHook := pod.Annotations[velero.PostBackupHookContainerAnnotation]
				return !hasPreHook && !hasPostHook
			}, 60*time.Second, 1*time.Second).Should(BeTrue(), "Velero hook annotations should be removed from launcher pod when KubeVirt CR annotation is set")

			By("Restoring KubeVirt CR annotations to original state")
			if hadSkipAnnotation {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:%q}}}`, velero.SkipHooksAnnotation, originalKvAnnotations[velero.SkipHooksAnnotation])
			} else {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:null}}}`, velero.SkipHooksAnnotation)
			}
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Verifying launcher pod Velero annotations are added back")
			Eventually(func() bool {
				pod = getPodByVMI(vmi)
				return pod.Annotations[velero.PreBackupHookContainerAnnotation] == "compute" &&
					pod.Annotations[velero.PostBackupHookContainerAnnotation] == "compute"
			}, 60*time.Second, 1*time.Second).Should(BeTrue())
		})

		It("VMI annotation should take precedence over KubeVirt CR annotation", func() {
			By("Getting KubeVirt CR and setting skip annotation to true")
			kv := libkubevirt.GetCurrentKv(virtClient)
			originalKvAnnotations := kv.Annotations
			if originalKvAnnotations == nil {
				originalKvAnnotations = make(map[string]string)
			}
			_, hadSkipAnnotation := originalKvAnnotations[velero.SkipHooksAnnotation]

			patchData := fmt.Appendf(nil, `{"metadata":{"annotations":{%q:"true"}}}`, velero.SkipHooksAnnotation)
			kv, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Creating VMI with skip-backup-hooks=false annotation (opposite of KubeVirt CR)")
			vmi := libvmifact.NewAlpineWithTestTooling(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithAnnotation(velero.SkipHooksAnnotation, "false"))

			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(matcher.ThisVMI(vmi), 300*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))

			By("Verifying launcher pod has Velero annotations (VMI annotation takes precedence)")
			pod := getPodByVMI(vmi)
			Expect(pod.Annotations).To(HaveKey(velero.PreBackupHookContainerAnnotation), "VMI annotation should override KubeVirt CR annotation")
			Expect(pod.Annotations).To(HaveKey(velero.PostBackupHookContainerAnnotation))
			Expect(pod.Annotations[velero.PreBackupHookContainerAnnotation]).To(Equal("compute"))

			By("Restoring KubeVirt CR annotations to original state")
			if hadSkipAnnotation {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:%q}}}`, velero.SkipHooksAnnotation, originalKvAnnotations[velero.SkipHooksAnnotation])
			} else {
				patchData = fmt.Appendf(nil, `{"metadata":{"annotations":{%q:null}}}`, velero.SkipHooksAnnotation)
			}
			_, err = virtClient.KubeVirt(kv.Namespace).Patch(context.Background(), kv.Name, types.MergePatchType, patchData, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
}))

func getPodByVMI(vmi *v1.VirtualMachineInstance) *corev1.Pod {
	pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pod).ToNot(BeNil())
	return pod
}

func backupName(vmName string) string {
	return "vmbackup-" + vmName + rand.String(5)
}

func trackerName(vmName string) string {
	return "vmbackuptracker-" + vmName + rand.String(5)
}

func createBackupTracker(virtClient kubecli.KubevirtClient, vm *v1.VirtualMachine) *backupv1.VirtualMachineBackupTracker {
	tracker := &backupv1.VirtualMachineBackupTracker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      trackerName(vm.Name),
			Namespace: vm.Namespace,
		},
		Spec: backupv1.VirtualMachineBackupTrackerSpec{
			Source: corev1.TypedLocalObjectReference{
				APIGroup: pointer.P(core.GroupName),
				Kind:     "VirtualMachine",
				Name:     vm.Name,
			},
		},
	}

	tracker, err := virtClient.VirtualMachineBackupTracker(tracker.Namespace).Create(context.Background(), tracker, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return tracker
}

func NewBackup(backupName, namespace, pvcName string) *backupv1.VirtualMachineBackup {
	return &backupv1.VirtualMachineBackup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backupName,
			Namespace: namespace,
		},
		Spec: backupv1.VirtualMachineBackupSpec{
			PvcName: &pvcName,
		},
	}
}

func newBackupWithSource(backupName, vmName, namespace, pvcName string) *backupv1.VirtualMachineBackup {
	vmBackup := NewBackup(backupName, namespace, pvcName)
	vmBackup.Spec.Source = corev1.TypedLocalObjectReference{
		APIGroup: pointer.P(core.GroupName),
		Kind:     "VirtualMachine",
		Name:     vmName,
	}
	return vmBackup
}

func newBackupWithTracker(backupName, namespace, pvcName, trackerName string) *backupv1.VirtualMachineBackup {
	vmBackup := NewBackup(backupName, namespace, pvcName)
	vmBackup.Spec.Source = corev1.TypedLocalObjectReference{
		APIGroup: pointer.P(backupapi.GroupName),
		Kind:     backupv1.VirtualMachineBackupTrackerGroupVersionKind.Kind,
		Name:     trackerName,
	}
	return vmBackup
}

type verifyBackupFunc func(virtClient kubecli.KubevirtClient, namespace string, backupName string) *backupv1.VirtualMachineBackup

func createAndVerifyFullVMBackup(virtClient kubecli.KubevirtClient, backupName, vmName, namespace, pvcName string, verifyBackup verifyBackupFunc) *backupv1.VirtualMachineBackup {
	vmbackup := newBackupWithSource(backupName, vmName, namespace, pvcName)

	_, err := virtClient.VirtualMachineBackup(vmbackup.Namespace).Create(context.Background(), vmbackup, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	vmbackup = verifyBackup(virtClient, namespace, vmbackup.Name)
	Expect(vmbackup.Status.Type).To(Equal(backupv1.Full))

	return vmbackup
}

func createAndVerifyBackupWithTracker(virtClient kubecli.KubevirtClient, backupName, namespace, pvcName, trackerName string, verifyBackup verifyBackupFunc) *backupv1.VirtualMachineBackup {
	vmbackup := newBackupWithTracker(backupName, namespace, pvcName, trackerName)

	vmbackup, err := virtClient.VirtualMachineBackup(vmbackup.Namespace).Create(context.Background(), vmbackup, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return verifyBackup(virtClient, namespace, vmbackup.Name)
}

func deleteVMBackup(virtClient kubecli.KubevirtClient, namespace string, backupName string) {
	err := virtClient.VirtualMachineBackup(namespace).Delete(context.Background(), backupName, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
	Eventually(func() error {
		_, err := virtClient.VirtualMachineBackup(namespace).Get(context.Background(), backupName, metav1.GetOptions{})
		return err
	}, 180*time.Second, 2*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
}

func waitBackupSucceeded(virtClient kubecli.KubevirtClient, namespace string, backupName string) *backupv1.VirtualMachineBackup {
	var vmbackup *backupv1.VirtualMachineBackup

	By(fmt.Sprintf("Waiting for VirtualMachineBackup %s/%s to succeed", namespace, backupName))
	Eventually(func() *backupv1.VirtualMachineBackupStatus {
		var err error
		vmbackup, err = virtClient.VirtualMachineBackup(namespace).Get(context.Background(), backupName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return vmbackup.Status
	}, 180*time.Second, 2*time.Second).Should(And(
		Not(BeNil()),
		gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Conditions": ContainElements(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(backupv1.ConditionDone),
					"Status": Equal(corev1.ConditionTrue),
					"Reason": ContainSubstring("Successfully completed VirtualMachineBackup")}),
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(backupv1.ConditionProgressing),
					"Status": Equal(corev1.ConditionFalse)}),
			),
		})),
	))

	events.ExpectEvent(vmbackup, corev1.EventTypeNormal, "VirtualMachineBackupCompletedSuccessfully")
	return vmbackup
}

func waitBackupFailed(virtClient kubecli.KubevirtClient, namespace string, backupName string) *backupv1.VirtualMachineBackup {
	var vmbackup *backupv1.VirtualMachineBackup

	By(fmt.Sprintf("Waiting for VirtualMachineBackup %s/%s to succeed", namespace, backupName))
	Eventually(func() *backupv1.VirtualMachineBackupStatus {
		var err error
		vmbackup, err = virtClient.VirtualMachineBackup(namespace).Get(context.Background(), backupName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		return vmbackup.Status
	}, 180*time.Second, 2*time.Second).Should(And(
		Not(BeNil()),
		gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"Conditions": ContainElements(
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(backupv1.ConditionDone),
					"Status": Equal(corev1.ConditionTrue),
					"Reason": ContainSubstring("Backup has failed")}),
				gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"Type":   Equal(backupv1.ConditionProgressing),
					"Status": Equal(corev1.ConditionFalse)}),
			),
		})),
	))

	events.ExpectEvent(vmbackup, corev1.EventTypeWarning, "VirtualMachineBackupFailed")
	return vmbackup
}

func getTargetPVCSizeWithOverhead(originalSize string) string {
	originalQuantity := resource.MustParse(originalSize)
	smallerQuantity := originalQuantity.DeepCopy()
	smallerQuantity.Set(int64(float64(originalQuantity.Value()) * 1.2))
	return smallerQuantity.String()
}

func getDoubleTargetPVCSize(originalSize string) string {
	originalQuantity := resource.MustParse(originalSize)
	smallerQuantity := originalQuantity.DeepCopy()
	smallerQuantity.Set(int64(float64(originalQuantity.Value()) * 2.2))
	return smallerQuantity.String()
}

func createExecutorPod(targetPVC *corev1.PersistentVolumeClaim) *corev1.Pod {
	pod := libstorage.RenderPodWithPVC("verifier", []string{"/bin/bash", "-c", "touch /tmp/startup; while true; do echo hello; sleep 2; done"}, nil, targetPVC)
	pod.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/cat", "/tmp/startup"},
			},
		},
	}
	return runPodAndExpectPhase(pod, corev1.PodRunning)
}

func corruptBitmap(virtClient kubecli.KubevirtClient, namespace, pvcName, cbtOverlayPath string) {
	pvc, err := virtClient.CoreV1().PersistentVolumeClaims(namespace).Get(context.Background(), pvcName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	executorPod := createExecutorPod(pvc)

	fullPath := fmt.Sprintf("%s%s", libstorage.DefaultPvcMountPath, cbtOverlayPath)

	By("Listing backend storage PVC contents before deletion")
	lsOutput, err := exec.ExecuteCommandOnPod(
		executorPod,
		executorPod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("find %s -type f -name '*.qcow2' 2>/dev/null || ls -laR %s", libstorage.DefaultPvcMountPath, libstorage.DefaultPvcMountPath)},
	)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("Backend storage PVC contents before deletion:\n%s\n", lsOutput)

	By(fmt.Sprintf("Deleting the QCOW2 overlay file at %s to corrupt the checkpoint", fullPath))
	_, err = exec.ExecuteCommandOnPod(
		executorPod,
		executorPod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("rm -f %s", fullPath)},
	)
	Expect(err).ToNot(HaveOccurred(), "Should be able to delete the QCOW2 overlay file")

	By("Verifying overlay was deleted")
	lsOutput, err = exec.ExecuteCommandOnPod(
		executorPod,
		executorPod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("find %s -type f -name '*.qcow2' 2>/dev/null || echo 'No qcow2 files found'", libstorage.DefaultPvcMountPath)},
	)
	Expect(err).ToNot(HaveOccurred())
	fmt.Printf("Backend storage PVC contents after deletion:\n%s\n", lsOutput)
	Expect(lsOutput).ToNot(ContainSubstring(cbtOverlayPath), "QCOW2 overlay should have been deleted")

	By("Cleaning up executor pod")
	err = virtClient.CoreV1().Pods(executorPod.Namespace).Delete(context.Background(), executorPod.Name, metav1.DeleteOptions{})
	Expect(err).ToNot(HaveOccurred())
}

func verifyBackupTargetPVCOutput(virtClient kubecli.KubevirtClient, targetPVC *corev1.PersistentVolumeClaim, vmName string, numBackups int, expectedDiskSizes []int64) {
	By(fmt.Sprintf("Verifying backup target PVC output: expecting %d backup(s) with %d disk(s) each", numBackups, len(expectedDiskSizes)))
	executorPod := createExecutorPod(targetPVC)

	backupOutputPath := fmt.Sprintf("%s/%s", libstorage.DefaultPvcMountPath, vmName)

	lsOutput, err := exec.ExecuteCommandOnPod(
		executorPod,
		executorPod.Spec.Containers[0].Name,
		[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s", backupOutputPath)},
	)
	Expect(err).ToNot(HaveOccurred())

	lsOutput = strings.TrimSpace(lsOutput)
	lsOutputList := []string{}
	if lsOutput != "" {
		lsOutputList = strings.Split(lsOutput, "\n")
	}

	Expect(lsOutputList).To(HaveLen(numBackups), fmt.Sprintf("Should have exactly %d backup directory(ies)", numBackups))

	for backupIdx, backupDir := range lsOutputList {
		Expect(backupDir).To(ContainSubstring("vmbackup"))

		fullBackupPath := fmt.Sprintf("%s/%s", backupOutputPath, backupDir)
		By(fmt.Sprintf("Verifying backup %d/%d: %s", backupIdx+1, numBackups, backupDir))

		lsQcow2Output, err := exec.ExecuteCommandOnPod(
			executorPod,
			executorPod.Spec.Containers[0].Name,
			[]string{"/bin/sh", "-c", fmt.Sprintf("ls -1 %s/*.qcow2 2>/dev/null | sort || echo", fullBackupPath)},
		)
		Expect(err).ToNot(HaveOccurred())

		qcow2Files := []string{}
		if strings.TrimSpace(lsQcow2Output) != "" {
			qcow2Files = strings.Split(strings.TrimSpace(lsQcow2Output), "\n")
		}
		Expect(qcow2Files).To(HaveLen(len(expectedDiskSizes)),
			fmt.Sprintf("Backup %s should have exactly %d qcow2 backup file(s)", backupDir, len(expectedDiskSizes)))

		// Verify size of each disk
		for diskIdx, qcow2File := range qcow2Files {
			By(fmt.Sprintf("Verifying disk %d/%d in backup %s", diskIdx+1, len(qcow2Files), backupDir))

			sizeOutput, err := exec.ExecuteCommandOnPod(
				executorPod,
				executorPod.Spec.Containers[0].Name,
				[]string{"/bin/sh", "-c", fmt.Sprintf("stat -c %%s %s", qcow2File)},
			)
			Expect(err).ToNot(HaveOccurred())
			actualSize, err := strconv.ParseInt(strings.TrimSpace(sizeOutput), 10, 64)
			Expect(err).ToNot(HaveOccurred())

			expectedSizeBytes := expectedDiskSizes[diskIdx]
			// Allow for 20% variance (80% minimum) to account for compression and sparse files
			minExpectedSize := int64(float64(expectedSizeBytes) * 0.8)
			Expect(actualSize).To(BeNumerically(">=", minExpectedSize),
				fmt.Sprintf("Disk %d backup file %s size (%d bytes / %.2f GB) should be at least %.2f GB (80%% of expected %.2f GB)",
					diskIdx+1, qcow2File, actualSize,
					float64(actualSize)/(1024*1024*1024),
					float64(minExpectedSize)/(1024*1024*1024),
					float64(expectedSizeBytes)/(1024*1024*1024)))
		}
	}

	Eventually(func() error {
		return virtClient.CoreV1().Pods(executorPod.Namespace).Delete(context.Background(), executorPod.Name, metav1.DeleteOptions{})
	}, 180*time.Second, time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))
}

func listDomainCheckpoints(vmi *v1.VirtualMachineInstance) []string {
	domainName := fmt.Sprintf("%s_%s", vmi.Namespace, vmi.Name)

	output := libpod.RunCommandOnVmiPod(vmi, []string{
		"virsh",
		"checkpoint-list",
		domainName,
		"--name",
	})

	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}
	}

	var checkpoints []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			checkpoints = append(checkpoints, line)
		}
	}
	return checkpoints
}

func getCheckpointXML(vmi *v1.VirtualMachineInstance, checkpointName string) string {
	domainName := fmt.Sprintf("%s_%s", vmi.Namespace, vmi.Name)
	return libpod.RunCommandOnVmiPod(vmi, []string{
		"virsh",
		"checkpoint-dumpxml",
		domainName,
		checkpointName,
	})
}

func expectCheckpointDisks(vmi *v1.VirtualMachineInstance, checkpointName string, includedDisks []string, excludedDisks []string) {
	xml := getCheckpointXML(vmi, checkpointName)
	for _, disk := range includedDisks {
		ExpectWithOffset(1, xml).To(ContainSubstring(fmt.Sprintf("name='%s' checkpoint='bitmap'", disk)),
			"Checkpoint %s should have bitmap for disk %s", checkpointName, disk)
	}
	for _, disk := range excludedDisks {
		ExpectWithOffset(1, xml).ToNot(ContainSubstring(fmt.Sprintf("name='%s'", disk)),
			"Checkpoint %s should not have bitmap for disk %s", checkpointName, disk)
	}
}
