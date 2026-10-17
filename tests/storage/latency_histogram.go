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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"libvirt.org/go/libvirtxml"

	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe(SIG("Block I/O latency histogram lifecycle", Serial, func() {
	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()
	})

	It("should restore a removed histogram after a VM restart", func() {
		vm := createRunningVMWithRunStrategy(
			virtClient,
			libvmifact.NewAlpine(),
			v1.RunStrategyAlways,
			true,
		)

		vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(
			context.Background(),
			vm.Name,
			metav1.GetOptions{},
		)
		Expect(err).ToNot(HaveOccurred())

		expectBlockLatencyHistograms(vmi, "disk0", true)

		removeBlockLatencyHistograms(vmi, "disk0")

		expectBlockLatencyHistograms(vmi, "disk0", false)

		By("Restarting the VM after removing its latency histograms")
		vm = libvmops.StopVirtualMachine(vm)
		vm = libvmops.StartVirtualMachine(vm)

		vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(
			context.Background(),
			vm.Name,
			metav1.GetOptions{},
		)
		Expect(err).ToNot(HaveOccurred())

		expectBlockLatencyHistograms(vmi, "disk0", true)
	})

	It("should restore a removed histogram after a disk is detached and reattached", decorators.StorageReq, func() {
		const (
			volumeName         = "latency-hotplug-disk"
			randomSuffixLength = 5
		)

		vmi := libvmops.RunVMIAndExpectLaunch(
			libvmifact.NewAlpine(),
			flags.StartupTimeoutSecondsHuge(),
		)

		dv := libstorage.CreateBlankFSDataVolume(
			"latency-hotplug-"+rand.String(randomSuffixLength),
			vmi.Namespace,
			"512Mi",
			nil,
		)

		addVolumeOptions := &v1.AddVolumeOptions{
			Name: volumeName,
			Disk: &v1.Disk{
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.DiskBusSCSI,
					},
				},
			},
			VolumeSource: &v1.HotplugVolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: dv.Name,
				},
			},
		}

		By("Hotplugging a data volume")
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vmi.Namespace).AddVolume(
				context.Background(),
				vmi.Name,
				addVolumeOptions,
			)
		}, 10*time.Second, time.Second).Should(Succeed())

		libstorage.VerifyVolumeStatus(
			virtClient,
			vmi,
			v1.VolumeReady,
			"",
			true,
			volumeName,
		)

		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(
			context.Background(),
			vmi.Name,
			metav1.GetOptions{},
		)
		Expect(err).ToNot(HaveOccurred())

		expectBlockLatencyHistograms(vmi, volumeName, true)

		removeBlockLatencyHistograms(vmi, volumeName)

		expectBlockLatencyHistograms(vmi, volumeName, false)

		By("Detaching the disk whose histograms were removed")
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vmi.Namespace).RemoveVolume(
				context.Background(),
				vmi.Name,
				&v1.RemoveVolumeOptions{Name: volumeName},
			)
		}, 10*time.Second, time.Second).Should(Succeed())

		Eventually(func() bool {
			currentVMI, getErr := virtClient.VirtualMachineInstance(vmi.Namespace).Get(
				context.Background(),
				vmi.Name,
				metav1.GetOptions{},
			)
			if getErr != nil {
				return false
			}

			for _, volumeStatus := range currentVMI.Status.VolumeStatus {
				if volumeStatus.Name == volumeName {
					return false
				}
			}

			return true
		}, 3*time.Minute, 2*time.Second).Should(BeTrue())

		By("Reattaching the disk and waiting for fresh histograms")
		Eventually(func() error {
			return virtClient.VirtualMachineInstance(vmi.Namespace).AddVolume(
				context.Background(),
				vmi.Name,
				addVolumeOptions,
			)
		}, 10*time.Second, time.Second).Should(Succeed())

		libstorage.VerifyVolumeStatus(
			virtClient,
			vmi,
			v1.VolumeReady,
			"",
			true,
			volumeName,
		)

		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(
			context.Background(),
			vmi.Name,
			metav1.GetOptions{},
		)
		Expect(err).ToNot(HaveOccurred())

		expectBlockLatencyHistograms(vmi, volumeName, true)
	})

	It("should restore a removed histogram on the target after live migration", decorators.RequiresTwoSchedulableNodes, func() {
		vmi := libvmifact.NewAlpine(
			libnet.WithMasqueradeNetworking(),
		)
		vmi = libvmops.RunVMIAndExpectLaunch(
			vmi,
			flags.StartupTimeoutSecondsHuge(),
		)

		expectBlockLatencyHistograms(vmi, "disk0", true)

		removeBlockLatencyHistograms(vmi, "disk0")

		expectBlockLatencyHistograms(vmi, "disk0", false)

		sourceNode := vmi.Status.NodeName

		By("Migrating the VMI")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(
			virtClient,
			migration,
		)
		libmigration.ConfirmVMIPostMigration(
			virtClient,
			vmi,
			migration,
		)

		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(
			context.Background(),
			vmi.Name,
			metav1.GetOptions{},
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.NodeName).ToNot(Equal(sourceNode))

		expectBlockLatencyHistograms(vmi, "disk0", true)
	})
}))

func removeBlockLatencyHistograms(
	vmi *v1.VirtualMachineInstance,
	drive string,
) {
	By(fmt.Sprintf(
		"Removing block I/O latency histograms from drive %s",
		drive,
	))

	domain := getRunningDomain(vmi)
	disk := findDomainDisk(domain, drive)

	ExpectWithOffset(
		1,
		disk,
	).ToNot(BeNil(), "drive %s was not found in the domain XML", drive)

	ExpectWithOffset(1, disk.Driver).ToNot(BeNil())
	ExpectWithOffset(1, disk.Driver.Statistics).ToNot(BeNil())
	ExpectWithOffset(
		1,
		disk.Driver.Statistics.LatencyHistogram,
	).ToNot(BeEmpty())

	disk.Driver.Statistics.LatencyHistogram = nil

	if len(disk.Driver.Statistics.Statistic) == 0 {
		disk.Driver.Statistics = nil
	}

	diskXML, err := disk.Marshal()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	const diskXMLPath = "/tmp/disk-without-latency-histograms.xml"

	libpod.RunCommandOnVmiPod(vmi, []string{
		"/bin/bash",
		"-c",
		`trap 'rm -f "$2"' EXIT; printf "%s" "$1" > "$2" && virsh update-device 1 "$2" --live`,
		"remove-latency-histograms",
		diskXML,
		diskXMLPath,
	})

	expectBlockLatencyHistograms(vmi, drive, false)
}

func expectBlockLatencyHistograms(
	vmi *v1.VirtualMachineInstance,
	drive string,
	expected bool,
) {
	disk := findDomainDisk(
		getRunningDomain(vmi),
		drive,
	)

	ExpectWithOffset(
		1,
		disk,
	).ToNot(BeNil(), "drive %s was not found in the domain XML", drive)

	hasHistograms := disk.Driver != nil &&
		disk.Driver.Statistics != nil &&
		len(disk.Driver.Statistics.LatencyHistogram) > 0

	ExpectWithOffset(
		1,
		hasHistograms,
	).To(Equal(expected))
}

func getRunningDomain(
	vmi *v1.VirtualMachineInstance,
) *libvirtxml.Domain {
	domainXML := libpod.RunCommandOnVmiPod(
		vmi,
		[]string{
			"virsh",
			"dumpxml",
			"1",
		},
	)

	domain := &libvirtxml.Domain{}

	ExpectWithOffset(
		1,
		domain.Unmarshal(domainXML),
	).To(Succeed())

	return domain
}

func findDomainDisk(
	domain *libvirtxml.Domain,
	drive string,
) *libvirtxml.DomainDisk {
	if domain.Devices == nil {
		return nil
	}

	for i := range domain.Devices.Disks {
		disk := &domain.Devices.Disks[i]

		if disk.Alias != nil &&
			strings.TrimPrefix(disk.Alias.Name, "ua-") == drive {
			return disk
		}
	}

	return nil
}

func createRunningVMWithRunStrategy(
	virtClient kubecli.KubevirtClient,
	vmi *v1.VirtualMachineInstance,
	runStrategy v1.VirtualMachineRunStrategy,
	waitForVMIStart bool,
) *v1.VirtualMachine {
	By("Creating a running VirtualMachine")

	vm := libvmi.NewVirtualMachine(
		vmi,
		libvmi.WithRunStrategy(runStrategy),
	)

	var err error

	vm, err = virtClient.VirtualMachine(
		testsuite.GetTestNamespace(vm),
	).Create(
		context.Background(),
		vm,
		metav1.CreateOptions{},
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	const vmReadyTimeout = 300 * time.Second

	if runStrategy != v1.RunStrategyHalted {
		EventuallyWithOffset(
			1,
			matcher.ThisVM(vm),
		).
			WithTimeout(vmReadyTimeout).
			WithPolling(time.Second).
			Should(matcher.BeReady())
	}

	if waitForVMIStart {
		vmi.Namespace = vm.Namespace
		vmi.Name = vm.Name

		libwait.WaitForSuccessfulVMIStart(vmi)
	}

	return vm
}
