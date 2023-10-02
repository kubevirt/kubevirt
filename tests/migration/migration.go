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

package migration

import (
	"context"
	"crypto/tls"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"kubevirt.io/kubevirt/tests/libmigration"

	migrationsv1 "kubevirt.io/api/migrations/v1alpha1"

	kvpointer "kubevirt.io/kubevirt/pkg/pointer"

	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libnet/service"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"

	"kubevirt.io/kubevirt/tests/events"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/testsuite"

	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"

	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/tests/clientcmd"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libdv"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/util"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"

	. "kubevirt.io/kubevirt/tests/framework/matcher"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/watcher"
)

const (
	fedoraVMSize               = "256M"
	secretDiskSerial           = "D23YZ9W6WA5DJ487"
	stressDefaultVMSize        = "100M"
	stressLargeVMSize          = "400M"
	stressDefaultSleepDuration = 1600
)

var _ = SIGMigrationDescribe("VM Live Migration", func() {
	var (
		virtClient              kubecli.KubevirtClient
		migrationBandwidthLimit resource.Quantity
		err                     error
	)

	const (
		downwardTestLabelKey = "downwardTestLabelKey"
		downwardTestLabelVal = "downwardTestLabelVal"
	)

	createConfigMap := func(namespace string) string {
		name := "configmap-" + rand.String(5)
		data := map[string]string{
			"config1": "value1",
			"config2": "value2",
		}
		tests.CreateConfigMap(name, namespace, data)
		return name
	}

	createSecret := func(namespace string) string {
		name := "secret-" + rand.String(5)
		data := map[string]string{
			"user":     "admin",
			"password": "redhat",
		}
		tests.CreateSecret(name, namespace, data)
		return name
	}

	withKernelBoot := func() libvmi.Option {
		return func(vmi *v1.VirtualMachineInstance) {
			kernelBootFirmware := utils.GetVMIKernelBootWithRandName().Spec.Domain.Firmware
			if vmiFirmware := vmi.Spec.Domain.Firmware; vmiFirmware == nil {
				vmiFirmware = kernelBootFirmware
			} else {
				vmiFirmware.KernelBoot = kernelBootFirmware.KernelBoot
			}
		}
	}

	prepareVMIWithAllVolumeSources := func(namespace string) *v1.VirtualMachineInstance {
		secretName := createSecret(namespace)
		configMapName := createConfigMap(namespace)

		return libvmi.NewFedora(
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithLabel(downwardTestLabelKey, downwardTestLabelVal),
			libvmi.WithDownwardAPIDisk("downwardapi-"+rand.String(5)),
			libvmi.WithServiceAccountDisk("default"),
			withKernelBoot(),
			libvmi.WithSecretDisk(secretName, secretName),
			libvmi.WithConfigMapDisk(configMapName, configMapName),
			libvmi.WithEmptyDisk("usb-disk", v1.DiskBusUSB, resource.MustParse("64Mi")),
			libvmi.WithCloudInitNoCloudUserData("#!/bin/bash\necho 'hello'\n", true),
		)
	}

	BeforeEach(func() {
		checks.SkipIfMigrationIsNotPossible()
		virtClient = kubevirt.Client()
		migrationBandwidthLimit = resource.MustParse("1Ki")
	})

	confirmMigrationMode := func(vmi *v1.VirtualMachineInstance, expectedMode v1.MigrationMode) {
		By("Retrieving the VMI post migration")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("Verifying the VMI's migration mode")
		Expect(vmi.Status.MigrationState.Mode).To(Equal(expectedMode))
	}

	getVirtqemudPid := func(pod *k8sv1.Pod) string {
		stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "compute",
			[]string{
				"pidof",
				"virtqemud",
			})
		errorMessageFormat := "faild after running `pidof virtqemud` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
		Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(errorMessageFormat, stdout, stderr, err))
		pid := strings.TrimSuffix(stdout, "\n")
		return pid
	}

	Context("with Headless service", func() {
		const subdomain = "mysub"

		AfterEach(func() {
			err := virtClient.CoreV1().Services(util.NamespaceTestDefault).Delete(context.Background(), subdomain, metav1.DeleteOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should remain to able resolve the VM IP", func() {
			withHostnameAndSubdomain := func(hostname, subdomain string) libvmi.Option {
				return func(vmi *v1.VirtualMachineInstance) {
					vmi.Spec.Hostname = hostname
					vmi.Spec.Subdomain = subdomain

				}
			}
			const hostname = "alpine"
			const port int = 1500
			const labelKey = "subdomain"
			const labelValue = "mysub"

			vmi := libvmi.NewCirros(
				withHostnameAndSubdomain(hostname, subdomain),
				libvmi.WithLabel(labelKey, labelValue),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Starting hello world in the VM")
			tests.StartTCPServer(vmi, port, console.LoginToCirros)

			By("Exposing headless service matching subdomain")
			service := service.BuildHeadlessSpec(subdomain, port, port, labelKey, labelValue)
			_, err = virtClient.CoreV1().Services(vmi.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			assertConnectivityToService := func(msg string) {
				By(msg)
				tcpJob := job.NewHelloWorldJobTCP(fmt.Sprintf("%s.%s", hostname, subdomain), strconv.FormatInt(int64(port), 10))
				tcpJob.Spec.BackoffLimit = pointer.Int32(3)
				tcpJob, err := virtClient.BatchV1().Jobs(vmi.Namespace).Create(context.Background(), tcpJob, k8smetav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = job.WaitForJobToSucceed(tcpJob, 90*time.Second)
				Expect(err).ToNot(HaveOccurred(), msg)
			}

			assertConnectivityToService("Asserting connectivity through service before migration")

			By("Executing a migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			assertConnectivityToService("Asserting connectivity through service after migration")

		})
	})

	Describe("Starting a VirtualMachineInstance ", func() {
		guestAgentMigrationTestFunc := func(pvName string, memoryRequestSize resource.Quantity, migrationPolicy *migrationsv1.MigrationPolicy) {
			By("Creating the VMI")
			vmi := tests.NewRandomVMIWithPVC(pvName)
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = memoryRequestSize
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

			mode := v1.MigrationPreCopy
			if migrationPolicy != nil && migrationPolicy.Spec.AllowPostCopy != nil && *migrationPolicy.Spec.AllowPostCopy {
				mode = v1.MigrationPostCopy
			}

			// postcopy needs a privileged namespace
			if mode == v1.MigrationPostCopy {
				vmi.Namespace = testsuite.NamespacePrivileged
			}

			// add userdata for guest agent and service account mount
			mountSvcAccCommands := fmt.Sprintf(`#!/bin/bash
					mkdir /mnt/servacc
					mount /dev/$(lsblk --nodeps -no name,serial | grep %s | cut -f1 -d' ') /mnt/servacc
				`, secretDiskSerial)
			tests.AddUserData(vmi, "cloud-init", mountSvcAccCommands)

			tests.AddServiceAccountDisk(vmi, "default")
			disks := vmi.Spec.Domain.Devices.Disks
			disks[len(disks)-1].Serial = secretDiskSerial

			if migrationPolicy != nil {
				AlignPolicyAndVmi(vmi, migrationPolicy)
				migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
			}
			vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

			// Wait for cloud init to finish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed(), "Should be able to login to the Fedora VM")

			if mode == v1.MigrationPostCopy {
				By("Running stress test to allow transition to post-copy")
				runStressTest(vmi, stressLargeVMSize, stressDefaultSleepDuration)
			}

			// execute a migration, wait for finalized state
			By("Starting the Migration for iteration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			By("Checking VMI, confirm migration state")
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			confirmMigrationMode(vmi, mode)

			By("Is agent connected after migration")
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			By("Checking that the migrated VirtualMachineInstance console has expected output")
			Expect(console.OnPrivilegedPrompt(vmi, 60)).To(BeTrue(), "Should stay logged in to the migrated VM")

			By("Checking that the service account is mounted")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "cat /mnt/servacc/namespace\n"},
				&expect.BExp{R: vmi.Namespace},
			}, 30)).To(Succeed(), "Should be able to access the mounted service account file")
		}

		Context("with a bridge network interface", func() {
			It("[test_id:3226]should reject a migration of a vmi with a bridge interface", func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
					{
						Name: "default",
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Bridge: &v1.InterfaceBridge{},
						},
					},
				}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(vmi).To(HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("InterfaceNotLiveMigratable"))
			})
		})
		Context("[Serial] with bandwidth limitations", Serial, func() {

			var repeatedlyMigrateWithBandwidthLimitation = func(vmi *v1.VirtualMachineInstance, bandwidth string, repeat int) time.Duration {
				var migrationDurationTotal time.Duration
				config := getCurrentKvConfig(virtClient)
				limit := resource.MustParse(bandwidth)
				config.MigrationConfiguration.BandwidthPerMigration = &limit
				tests.UpdateKubeVirtConfigValueAndWait(config)

				for x := 0; x < repeat; x++ {
					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())

					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					migrationDuration := vmi.Status.MigrationState.EndTimestamp.Sub(vmi.Status.MigrationState.StartTimestamp.Time)
					log.DefaultLogger().Infof("Migration with bandwidth %v took: %v", bandwidth, migrationDuration)
					migrationDurationTotal += migrationDuration
				}
				return migrationDurationTotal
			}

			It("[test_id:6968]should apply them and result in different migration durations", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				durationLowBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, "10Mi", 3)
				durationHighBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, "128Mi", 3)
				Expect(durationHighBandwidth.Seconds() * 2).To(BeNumerically("<", durationLowBandwidth.Seconds()))
			})
		})
		Context("with a Alpine disk", func() {
			It("[test_id:6969]should be successfully migrate with a tablet device", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				vmi.Spec.Domain.Devices.Inputs = []v1.Input{
					{
						Name: "tablet0",
						Type: "tablet",
						Bus:  "usb",
					},
				}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
			It("should be successfully migrate with a WriteBack disk cache", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheWriteBack

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				runningVMISpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())

				disks := runningVMISpec.Devices.Disks
				By("checking if requested cache 'writeback' has been set")
				Expect(disks[0].Alias.GetName()).To(Equal("disk0"))
				Expect(disks[0].Driver.Cache).To(Equal(string(v1.CacheWriteBack)))
			})

			It("[test_id:6970]should migrate vmi with cdroms on various bus types", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				tests.AddEphemeralCdrom(vmi, "cdrom-0", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				tests.AddEphemeralCdrom(vmi, "cdrom-1", v1.DiskBusSCSI, cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("should migrate vmi with LiveMigrateIfPossible eviction strategy", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				strategy := v1.EvictionStrategyLiveMigrateIfPossible
				vmi.Spec.EvictionStrategy = &strategy

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("should migrate vmi and use Live Migration method with read-only disks", func() {
				By("Defining a VMI with PVC disk and read-only CDRoms")
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)
				vmi.Spec.Hostname = string(cd.ContainerDiskAlpine)

				tests.AddEphemeralCdrom(vmi, "cdrom-0", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				tests.AddEphemeralCdrom(vmi, "cdrom-1", v1.DiskBusSCSI, cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Ensuring migration is using Live Migration method")
				Eventually(func() v1.VirtualMachineInstanceMigrationMethod {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					return vmi.Status.MigrationMethod
				}, 20*time.Second, 1*time.Second).Should(Equal(v1.LiveMigration), "migration method is expected to be Live Migration")
			})

			DescribeTable("should migrate with a downwardMetrics", func(via libvmi.Option, metricsGetter libinfra.MetricsGetter) {
				vmi := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					via,
				)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("checking if the metrics are still updated after the migration")
				Eventually(func() error {
					_, err := metricsGetter(vmi)
					return err
				}, 20*time.Second, 1*time.Second).ShouldNot(HaveOccurred())
				metrics, err := metricsGetter(vmi)
				Expect(err).ToNot(HaveOccurred())
				timestamp := libinfra.GetTimeFromMetrics(metrics)
				Eventually(func() int {
					metrics, err := metricsGetter(vmi)
					Expect(err).ToNot(HaveOccurred())
					return libinfra.GetTimeFromMetrics(metrics)
				}, 10*time.Second, 1*time.Second).ShouldNot(Equal(timestamp))

				By("checking that the new nodename is reflected in the downward metrics")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(libinfra.GetHostnameFromMetrics(metrics)).To(Equal(vmi.Status.NodeName))

			},
				Entry("[test_id:6971]disk", libvmi.WithDownwardMetricsVolume("vhostmd"), libinfra.GetDownwardMetricsDisk),
				Entry("channel", libvmi.WithDownwardMetricsChannel(), libinfra.GetDownwardMetricsVirtio),
			)

			It("[test_id:6842]should migrate with TSC frequency set", decorators.Invtsc, decorators.TscFrequencies, func() {
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				vmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{
							Name:   "invtsc",
							Policy: "require",
						},
					},
				}
				// only with this strategy will the frequency be set
				strategy := v1.EvictionStrategyLiveMigrate
				vmi.Spec.EvictionStrategy = &strategy

				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Checking the TSC frequency on the Domain XML")
				domainSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				timerFrequency := ""
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						timerFrequency = timer.Frequency
					}
				}
				Expect(timerFrequency).ToNot(BeEmpty())

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Checking the TSC frequency on the Domain XML on the new node")
				domainSpec, err = tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				timerFrequency = ""
				for _, timer := range domainSpec.Clock.Timer {
					if timer.Name == "tsc" {
						timerFrequency = timer.Frequency
					}
				}
				Expect(timerFrequency).ToNot(BeEmpty())
			})

			It("[test_id:4113]should be successfully migrate with cloud-init disk with devices on the root bus", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)
				vmi.Annotations = map[string]string{
					v1.PlacePCIDevicesOnRootComplex: "true",
				}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("checking that we really migrated a VMI with only the root bus")
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

			It("should migrate vmi with a usb disk", func() {

				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithEmptyDisk("uniqueusbdisk", v1.DiskBusUSB, resource.MustParse("128Mi")),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:1783]should be successfully migrated multiple times with cloud-init disk", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				num := 4

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
					libmigration.ConfirmMigrationDataIsStored(virtClient, migration, vmi)

					By("Check if Migrated VMI has updated IP and IPs fields")
					Eventually(func() error {
						newvmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod := tests.GetRunningPodByVirtualMachineInstance(newvmi, newvmi.Namespace)
						return libnet.ValidateVMIandPodIPMatch(newvmi, vmiPod)
					}, 180*time.Second, time.Second).Should(Succeed(), "Should have updated IP and IPs fields")
				}
			})

			// We had a bug that prevent migrations and graceful shutdown when the libvirt connection
			// is reset. This can occur for many reasons, one easy way to trigger it is to
			// force virtqemud down, which will result in virt-launcher respawning it.
			// Previously, we'd stop getting events after libvirt reconnect, which
			// prevented things like migration. This test verifies we can migrate after
			// resetting virtqemud
			It("[test_id:4746]should migrate even if virtqemud has restarted at some point.", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")

				// find virtqemud pid
				pid := getVirtqemudPid(&pods.Items[0])

				// kill virtqemud
				By(fmt.Sprintf("Killing virtqemud with pid %s", pid))
				stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, &pods.Items[0], "compute",
					[]string{
						"kill",
						"-9",
						pid,
					})
				errorMessageFormat := "failed after running `kill -9 %v` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(errorMessageFormat, pid, stdout, stderr, err))

				// wait for both virtqemud to respawn and all connections to re-establish
				time.Sleep(30 * time.Second)

				// ensure new pid comes online
				newPid := getVirtqemudPid(&pods.Items[0])
				Expect(pid).ToNot(Equal(newPid), fmt.Sprintf("expected virtqemud to be cycled. original pid %s new pid %s", pid, newPid))

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:6972]should migrate to a persistent (non-transient) libvirt domain.", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				// ensure the libvirt domain is persistent
				persistent, err := libvirtDomainIsPersistent(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should list libvirt domains successfully")
				Expect(persistent).To(BeTrue(), "The VMI was not found in the list of libvirt persistent domains")
				libmigration.EnsureNoMigrationMetadataInPersistentXML(vmi)
			})
			It("[test_id:6973]should be able to successfully migrate with a paused vmi", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Pausing the VirtualMachineInstance")
				err := virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

				By("verifying that the vmi is still paused before migration")
				isPausedb, err := tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPausedb).To(BeTrue(), "The VMI should be paused before migration, but it is not.")

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("verifying that the vmi is still paused after migration")
				isPaused, err := tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPaused).To(BeTrue(), "The VMI should be paused after migration, but it is not.")

				By("verify that VMI can be unpaused after migration")
				command := clientcmd.NewRepeatableVirtctlCommand("unpause", "vmi", "--namespace", vmi.Namespace, vmi.Name)
				Expect(command()).To(Succeed(), "should successfully unpause tthe vmi")
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

				By("verifying that the vmi is running")
				isPaused, err = tests.LibvirtDomainIsPaused(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred(), "Should get domain state successfully")
				Expect(isPaused).To(BeFalse(), "The VMI should be running, but it is not.")
			})
		})

		Context("with an pending target pod", func() {
			var nodes *k8sv1.NodeList
			BeforeEach(func() {
				Eventually(func() []k8sv1.Node {
					nodes = libnode.GetAllSchedulableNodes(virtClient)
					return nodes.Items
				}, 60*time.Second, 1*time.Second).ShouldNot(BeEmpty(), "There should be some compute node")
			})

			It("should automatically cancel unschedulable migration after a timeout period", func() {
				// Add node affinity to ensure VMI affinity rules block target pod from being created
				vmi := tests.NewRandomFedoraVMI(libvmi.WithNodeAffinityFor(&nodes.Items[0]))
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "130"}

				var err error
				Eventually(func() error {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
					return err
				}, 5, 1*time.Second).Should(Succeed(), "migration creation should succeed")

				By("Should receive warning event that target pod is currently unschedulable")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				watcher.New(migration).
					Timeout(60*time.Second).
					SinceWatchedObjectResourceVersion().
					WaitFor(ctx, watcher.WarningEvent, "migrationTargetPodUnschedulable")

				By("Migration should observe a timeout period before canceling unschedulable target pod")
				Consistently(func() error {

					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase == v1.MigrationFailed {
						return fmt.Errorf("Migration should observe timeout period before transitioning to failed state")
					}
					return nil

				}, 1*time.Minute, 10*time.Second).Should(Succeed())

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(func() error {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase != v1.MigrationFailed {
						return fmt.Errorf("Waiting on migration with phase %s to reach phase Failed", migration.Status.Phase)
					}
					return nil
				}, 2*time.Minute, 5*time.Second).Should(Succeed(), "migration creation should fail")
			})

			It("should automatically cancel pending target pod after a catch all timeout period", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationPendingPodTimeoutSecondsAnnotation: "130"}

				// Add a fake continer image to the target pod to force a image pull failure which
				// keeps the target pod in pending state
				// Make sure to actually use an image repository we own here so no one
				// can somehow figure out a way to execute custom logic in our func tests.
				migration.Annotations[v1.FuncTestMigrationTargetImageOverrideAnnotation] = "quay.io/kubevirtci/some-fake-image:" + rand.String(12)

				By("Starting a Migration")
				var err error
				Eventually(func() error {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
					return err
				}, 5, 1*time.Second).Should(Succeed(), "migration creation should succeed")

				By("Migration should observe a timeout period before canceling pending target pod")
				Consistently(func() error {

					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase == v1.MigrationFailed {
						return fmt.Errorf("Migration should observe timeout period before transitioning to failed state")
					}
					return nil

				}, 1*time.Minute, 10*time.Second).Should(Succeed())

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(func() error {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					if err != nil {
						return err
					}

					if migration.Status.Phase != v1.MigrationFailed {
						return fmt.Errorf("Waiting on migration with phase %s to reach phase Failed", migration.Status.Phase)
					}
					return nil
				}, 2*time.Minute, 5*time.Second).Should(Succeed(), "migration creation should fail")
			})
		})
		Context("[Serial] with auto converge enabled", Serial, func() {
			BeforeEach(func() {

				// set autoconverge flag
				config := getCurrentKvConfig(virtClient)
				allowAutoConverage := true
				config.MigrationConfiguration.AllowAutoConverge = &allowAutoConverage
				tests.UpdateKubeVirtConfigValueAndWait(config)
			})

			It("[test_id:3237]should complete a migration", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				runStressTest(vmi, stressDefaultVMSize, stressDefaultSleepDuration)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
		})
		Context("with setting guest time", func() {
			It("[test_id:4114]should set an updated time after a migration", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Set wrong time on the guest")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "date +%T -s 23:26:00\n"},
					&expect.BExp{R: console.PromptExpression},
				}, 15)).To(Succeed(), "should set guest time")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the migrated VirtualMachineInstance has an updated time")
				if !console.OnPrivilegedPrompt(vmi, 60) {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
				}

				By("Waiting for the agent to set the right time")
				Eventually(func() error {
					// get current time on the node
					output := tests.RunCommandOnVmiPod(vmi, []string{"date", "+%H:%M"})
					expectedTime := strings.TrimSpace(output)
					log.DefaultLogger().Infof("expoected time: %v", expectedTime)

					By("Checking that the guest has an updated time")
					return console.SafeExpectBatch(vmi, []expect.Batcher{
						&expect.BSnd{S: "date +%H:%M\n"},
						&expect.BExp{R: expectedTime},
					}, 30)
				}, 240*time.Second, 1*time.Second).Should(Succeed())
			})
		})

		Context("with an Alpine DataVolume", func() {
			BeforeEach(func() {
				if !libstorage.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}
			})

			It("[test_id:3239]should reject a migration of a vmi with a non-shared data volume", func() {
				sc, foundSC := libstorage.GetRWOFileSystemStorageClass()
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithPVC(libdv.PVCWithStorageClass(sc)),
				)

				vmi := tests.NewRandomVMIWithDataVolume(dataVolume.Name)

				dataVolume, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				libstorage.EventuallyDV(dataVolume, 240, Or(HaveSucceeded(), BeInPhase(cdiv1.WaitForFirstConsumer)))

				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(vmi).Should(matcher.HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))
			})
			It("[test_id:1479][storage-req] should migrate a vmi with a shared block disk", decorators.StorageReq, func() {
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 300)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
			It("[test_id:6974]should reject additional migrations on the same VMI if the first one is not finished", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				// Only stressing the VMI for 60 seconds to ensure the first migration eventually succeeds
				By("Stressing the VMI")
				runStressTest(vmi, stressDefaultVMSize, 60)

				By("Starting a first migration")
				migration1 := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration1, err = virtClient.VirtualMachineInstanceMigration(migration1.Namespace).Create(migration1, &metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Successfully tested with 40, but requests start getting throttled above 10, which is better to avoid to prevent flakyness
				By("Starting 10 more migrations expecting all to fail to create")
				var wg sync.WaitGroup
				for n := 0; n < 10; n++ {
					wg.Add(1)
					go func(n int) {
						defer GinkgoRecover()
						defer wg.Done()
						migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
						_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
						Expect(err).To(HaveOccurred(), fmt.Sprintf("Extra migration %d should have failed to create", n))
						Expect(err.Error()).To(ContainSubstring(`admission webhook "migration-create-validator.kubevirt.io" denied the request: in-flight migration detected.`))
					}(n)
				}
				wg.Wait()

				libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration1)
			})
		})
		Context("[storage-req]with an Alpine shared block volume PVC", decorators.StorageReq, func() {

			It("[test_id:1854]should migrate a VMI with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)
				image := cd.ContainerDiskFor(cd.ContainerDiskAlpine)
				tests.AddEphemeralDisk(vmi, "myephemeral", v1.VirtIO, image)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
			It("[release-blocker][test_id:1377]should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:3240]should be successfully with a cloud init", func() {
				// Start the VirtualMachineInstance with the PVC attached

				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskCirros), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteMany)
				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\necho 'hello'\n")
				vmi.Spec.Hostname = fmt.Sprintf("%s", cd.ContainerDiskCirros)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Checking that MigrationMethod is set to BlockMigration")
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.BlockMigration))

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
		})

		Context("with a Fedora shared NFS PVC (using nfs ipv4 address), cloud init and service account", func() {
			var vmi *v1.VirtualMachineInstance
			var dv *cdiv1.DataVolume
			var storageClass string

			createDV := func(namespace string) {
				url := "docker://" + cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling)
				dv = libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(url, cdiv1.RegistryPullNode),
					libdv.WithPVC(
						libdv.PVCWithStorageClass(storageClass),
						libdv.PVCWithVolumeSize(cd.FedoraVolumeSize),
						libdv.PVCWithReadWriteManyAccessMode(),
					),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			BeforeEach(func() {
				var foundSC bool
				storageClass, foundSC = libstorage.GetRWXFileSystemStorageClass()
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}
			})

			AfterEach(func() {
				libstorage.DeleteDataVolume(&dv)
			})

			It("[test_id:2653] should be migrated successfully, using guest agent on VM with default migration configuration", func() {
				By("Creating the DV")
				createDV(testsuite.NamespacePrivileged)
				guestAgentMigrationTestFunc(dv.Name, resource.MustParse(fedoraVMSize), nil)
			})

			It("[test_id:6975] should have guest agent functional after migration", func() {
				By("Creating the DV")
				createDV(testsuite.GetTestNamespace(nil))
				By("Creating the VMI")
				vmi = tests.NewRandomVMIWithPVC(dv.Name)
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}

				tests.AddUserData(vmi, "cloud-init", "#!/bin/bash\n echo hello\n")
				vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Checking guest agent")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Starting the Migration for iteration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				By("Agent stays connected")
				Consistently(matcher.ThisVMI(vmi), 5*time.Minute, 10*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))
			})
		})

		createDataVolumePVCAndChangeDiskImgPermissions := func(namespace, size string) *cdiv1.DataVolume {
			// Create DV and alter permission of disk.img
			sc, foundSC := libstorage.GetRWXFileSystemStorageClass()
			if !foundSC {
				Skip("Skip test when Filesystem storage is not present")
			}

			dv := libdv.NewDataVolume(
				libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
				libdv.WithPVC(
					libdv.PVCWithStorageClass(sc),
					libdv.PVCWithVolumeSize(size),
					libdv.PVCWithReadWriteManyAccessMode(),
				),
				libdv.WithForceBindAnnotation(),
			)

			dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			var pvc *k8sv1.PersistentVolumeClaim
			Eventually(func() *k8sv1.PersistentVolumeClaim {
				pvc, err = virtClient.CoreV1().PersistentVolumeClaims(dv.Namespace).Get(context.Background(), dv.Name, metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return pvc
			}, 30*time.Second).Should(Not(BeNil()))
			By("waiting for the dv import to pvc to finish")
			libstorage.EventuallyDV(dv, 180, HaveSucceeded())
			tests.ChangeImgFilePermissionsToNonQEMU(pvc)
			return dv
		}

		Context("[Serial] migration to nonroot", Serial, func() {
			var dv *cdiv1.DataVolume
			size := "256Mi"
			var clusterIsRoot bool

			BeforeEach(func() {
				clusterIsRoot = checks.HasFeature(virtconfig.Root)
				if !clusterIsRoot {
					tests.EnableFeatureGate(virtconfig.Root)
				}
			})
			AfterEach(func() {
				if !clusterIsRoot {
					tests.DisableFeatureGate(virtconfig.Root)
				} else {
					tests.EnableFeatureGate(virtconfig.Root)
				}
				libstorage.DeleteDataVolume(&dv)
			})

			DescribeTable("should migrate root implementation to nonroot", func(createVMI func() *v1.VirtualMachineInstance, loginFunc func(*v1.VirtualMachineInstance) error) {
				By("Create a VMI that will run root(default)")
				vmi := createVMI()

				By("Starting the VirtualMachineInstance")
				// Resizing takes too long and therefor a warning is thrown
				vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				By("Checking that the launcher is running as root")
				Expect(tests.GetIdOfLauncher(vmi)).To(Equal("0"))

				tests.DisableFeatureGate(virtconfig.Root)

				By("Starting new migration and waiting for it to succeed")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Checking that the launcher is running as qemu")
				Expect(tests.GetIdOfLauncher(vmi)).To(Equal("107"))
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				vmi, err := ThisVMI(vmi)()
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).To(HaveKey(v1.DeprecatedNonRootVMIAnnotation))
			},
				Entry("[test_id:8609] with simple VMI", func() *v1.VirtualMachineInstance {
					return libvmi.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()))
				}, console.LoginToAlpine),

				Entry("[test_id:8610] with DataVolume", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the DataVolume
					return tests.NewRandomVMIWithDataVolume(dv.Name)
				}, console.LoginToAlpine),

				Entry("[test_id:8611] with CD + CloudInit + SA + ConfigMap + Secret + DownwardAPI + Kernel Boot", func() *v1.VirtualMachineInstance {
					return prepareVMIWithAllVolumeSources(testsuite.NamespacePrivileged)
				}, console.LoginToFedora),

				Entry("[test_id:8612] with PVC", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the Underlying PVC
					return tests.NewRandomVMIWithPVC(dv.Name)
				}, console.LoginToAlpine),
			)
		})
		Context("[Serial] migration to root", Serial, func() {
			var dv *cdiv1.DataVolume
			var clusterIsRoot bool
			size := "256Mi"

			BeforeEach(func() {
				clusterIsRoot = checks.HasFeature(virtconfig.Root)
				if clusterIsRoot {
					tests.DisableFeatureGate(virtconfig.Root)
				}
			})
			AfterEach(func() {
				if clusterIsRoot {
					tests.EnableFeatureGate(virtconfig.Root)
				} else {
					tests.DisableFeatureGate(virtconfig.Root)
				}
				if dv != nil {
					libstorage.DeleteDataVolume(&dv)
					dv = nil
				}
			})

			DescribeTable("should migrate nonroot implementation to root", func(createVMI func() *v1.VirtualMachineInstance, loginFunc func(*v1.VirtualMachineInstance) error) {
				By("Create a VMI that will run root(default)")
				vmi := createVMI()
				// force VMI on privileged namespace since we will be migrating to a root VMI
				vmi.Namespace = testsuite.NamespacePrivileged

				By("Starting the VirtualMachineInstance")
				// Resizing takes too long and therefor a warning is thrown
				vmi = tests.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				By("Checking that the launcher is running as root")
				Expect(tests.GetIdOfLauncher(vmi)).To(Equal("107"))

				tests.EnableFeatureGate(virtconfig.Root)

				By("Starting new migration and waiting for it to succeed")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Checking that the launcher is running as qemu")
				Expect(tests.GetIdOfLauncher(vmi)).To(Equal("0"))
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				vmi, err := ThisVMI(vmi)()
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).ToNot(HaveKey(v1.DeprecatedNonRootVMIAnnotation))
			},
				Entry("with simple VMI", func() *v1.VirtualMachineInstance {
					return libvmi.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()))
				}, console.LoginToAlpine),

				Entry("with DataVolume", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the DataVolume
					return tests.NewRandomVMIWithDataVolume(dv.Name)
				}, console.LoginToAlpine),

				Entry("with CD + CloudInit + SA + ConfigMap + Secret + DownwardAPI + Kernel Boot", func() *v1.VirtualMachineInstance {
					return prepareVMIWithAllVolumeSources(testsuite.NamespacePrivileged)
				}, console.LoginToFedora),

				Entry("with PVC", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the underlying PVC
					return tests.NewRandomVMIWithPVC(dv.Name)
				}, console.LoginToAlpine),
			)
		})
		Context("migration security", func() {
			Context("[Serial] with TLS disabled", Serial, func() {
				It("[test_id:6976] should be successfully migrated", func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration.DisableTLS = pointer.BoolPtr(true)
					tests.UpdateKubeVirtConfigValueAndWait(cfg)

					vmi := libvmi.NewAlpineWithTestTooling(
						libvmi.WithMasqueradeNetworking()...,
					)

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())

					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				})

				It("[test_id:6977]should not secure migrations with TLS", func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration.BandwidthPerMigration = resource.NewQuantity(1, resource.BinarySI)
					cfg.MigrationConfiguration.DisableTLS = pointer.BoolPtr(true)
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
					vmi := tests.NewRandomFedoraVMI()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					runStressTest(vmi, stressDefaultVMSize, stressDefaultSleepDuration)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})

					By("Waiting for the proxy connection details to appear")
					Eventually(func() bool {
						migratingVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if migratingVMI.Status.MigrationState == nil {
							return false
						}

						if migratingVMI.Status.MigrationState.TargetNodeAddress == "" || len(migratingVMI.Status.MigrationState.TargetDirectMigrationNodePorts) == 0 {
							return false
						}
						vmi = migratingVMI
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue())

					By("checking if we fail to connect with our own cert")
					tlsConfig := temporaryTLSConfig()

					handler, err := libnode.GetVirtHandlerPod(virtClient, vmi.Status.MigrationState.TargetNode)
					Expect(err).ToNot(HaveOccurred())

					var wg sync.WaitGroup
					wg.Add(len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))

					i := 0
					errors := make(chan error, len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))
					for port := range vmi.Status.MigrationState.TargetDirectMigrationNodePorts {
						portI, _ := strconv.Atoi(port)
						go func(i int, port int) {
							defer GinkgoRecover()
							defer wg.Done()
							stopChan := make(chan struct{})
							defer close(stopChan)
							Expect(tests.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
							_, err := tls.Dial("tcp", fmt.Sprintf("localhost:4321%d", i), tlsConfig)
							Expect(err).To(HaveOccurred())
							errors <- err
						}(i, portI)
						i++
					}
					wg.Wait()
					close(errors)

					By("checking that we were never able to connect")
					for err := range errors {
						Expect(err.Error()).To(Or(ContainSubstring("EOF"), ContainSubstring("first record does not look like a TLS handshake")))
					}
				})
			})
			Context("with TLS enabled", func() {
				BeforeEach(func() {
					cfg := getCurrentKvConfig(virtClient)
					tlsEnabled := cfg.MigrationConfiguration.DisableTLS == nil || *cfg.MigrationConfiguration.DisableTLS == false
					if !tlsEnabled {
						Skip("test requires secure migrations to be enabled")
					}
				})

				It("[test_id:2303][posneg:negative] should secure migrations with TLS", func() {
					vmi := tests.NewRandomFedoraVMI()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

					By("Limiting the bandwidth of migrations in the test namespace")
					CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for the proxy connection details to appear")
					Eventually(func() bool {
						migratingVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if migratingVMI.Status.MigrationState == nil {
							return false
						}

						if migratingVMI.Status.MigrationState.TargetNodeAddress == "" || len(migratingVMI.Status.MigrationState.TargetDirectMigrationNodePorts) == 0 {
							return false
						}
						vmi = migratingVMI
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue())

					By("checking if we fail to connect with our own cert")
					tlsConfig := temporaryTLSConfig()

					handler, err := libnode.GetVirtHandlerPod(virtClient, vmi.Status.MigrationState.TargetNode)
					Expect(err).ToNot(HaveOccurred())

					var wg sync.WaitGroup
					wg.Add(len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))

					i := 0
					errors := make(chan error, len(vmi.Status.MigrationState.TargetDirectMigrationNodePorts))
					for port := range vmi.Status.MigrationState.TargetDirectMigrationNodePorts {
						portI, _ := strconv.Atoi(port)
						go func(i int, port int) {
							defer GinkgoRecover()
							defer wg.Done()
							stopChan := make(chan struct{})
							defer close(stopChan)
							Expect(tests.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
							conn, err := tls.Dial("tcp", fmt.Sprintf("localhost:4321%d", i), tlsConfig)
							if conn != nil {
								b := make([]byte, 1)
								_, err = conn.Read(b)
							}
							Expect(err).To(HaveOccurred())
							errors <- err
						}(i, portI)
						i++
					}
					wg.Wait()
					close(errors)

					By("checking that we were never able to connect")
					tlsErrorFound := false
					for err := range errors {
						if strings.Contains(err.Error(), "remote error: tls: bad certificate") {
							tlsErrorFound = true
						}
						Expect(err.Error()).To(Or(ContainSubstring("remote error: tls: bad certificate"), ContainSubstring("EOF")))
					}

					Expect(tlsErrorFound).To(BeTrue())
				})
			})
		})

		Context("migration postcopy", func() {

			var migrationPolicy *migrationsv1.MigrationPolicy

			BeforeEach(func() {
				By("Allowing post-copy and limit migration bandwidth")
				policyName := fmt.Sprintf("testpolicy-%s", rand.String(5))
				migrationPolicy = kubecli.NewMinimalMigrationPolicy(policyName)
				migrationPolicy.Spec.AllowPostCopy = kvpointer.P(true)
				migrationPolicy.Spec.CompletionTimeoutPerGiB = kvpointer.P(int64(1))
				migrationPolicy.Spec.BandwidthPerMigration = kvpointer.P(resource.MustParse("5Mi"))
			})

			Context("with datavolume", func() {
				var dv *cdiv1.DataVolume

				BeforeEach(func() {
					sc, foundSC := libstorage.GetRWXFileSystemStorageClass()
					if !foundSC {
						Skip("Skip test when Filesystem storage is not present")
					}

					dv = libdv.NewDataVolume(
						libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling), cdiv1.RegistryPullNode),
						libdv.WithPVC(
							libdv.PVCWithStorageClass(sc),
							libdv.PVCWithVolumeSize(cd.FedoraVolumeSize),
							libdv.PVCWithReadWriteManyAccessMode(),
						),
					)

					dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.NamespacePrivileged).Create(context.Background(), dv, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
				})

				AfterEach(func() {
					libstorage.DeleteDataVolume(&dv)
				})

				It("[test_id:5004] should be migrated successfully, using guest agent on VM with postcopy", func() {
					guestAgentMigrationTestFunc(dv.Name, resource.MustParse("1Gi"), migrationPolicy)
				})

			})

			Context("should migrate using for postcopy", func() {

				applyMigrationPolicy := func(vmi *v1.VirtualMachineInstance) {
					AlignPolicyAndVmi(vmi, migrationPolicy)
					migrationPolicy = CreateMigrationPolicy(virtClient, migrationPolicy)
				}

				applyKubevirtCR := func() {
					config := getCurrentKvConfig(virtClient)
					config.MigrationConfiguration.AllowPostCopy = migrationPolicy.Spec.AllowPostCopy
					config.MigrationConfiguration.CompletionTimeoutPerGiB = migrationPolicy.Spec.CompletionTimeoutPerGiB
					config.MigrationConfiguration.BandwidthPerMigration = migrationPolicy.Spec.BandwidthPerMigration
					tests.UpdateKubeVirtConfigValueAndWait(config)
				}

				type applySettingsType string
				const (
					applyWithMigrationPolicy applySettingsType = "policy"
					applyWithKubevirtCR      applySettingsType = "kubevirt"
				)

				DescribeTable("[test_id:4747] using", func(settingsType applySettingsType) {
					vmi := tests.NewRandomFedoraVMI()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512Mi")
					vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
					vmi.Namespace = testsuite.NamespacePrivileged

					switch settingsType {
					case applyWithMigrationPolicy:
						applyMigrationPolicy(vmi)
					case applyWithKubevirtCR:
						applyKubevirtCR()
					}

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					runStressTest(vmi, "350M", stressDefaultSleepDuration)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 150)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
					confirmMigrationMode(vmi, v1.MigrationPostCopy)
				},
					Entry("a migration policy", applyWithMigrationPolicy),
					Entry("[Serial] Kubevirt CR", Serial, applyWithKubevirtCR),
				)
			})
		})

		Context("[Serial] migration monitor", Serial, func() {
			var createdPods []string
			AfterEach(func() {
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), podName, metav1.DeleteOptions{})

						if err != nil && errors.IsNotFound(err) {
							return nil
						}
						return err
					}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete helper pod")
				}
			})
			BeforeEach(func() {
				createdPods = []string{}
				cfg := getCurrentKvConfig(virtClient)
				var timeout int64 = 5
				cfg.MigrationConfiguration = &v1.MigrationConfiguration{
					CompletionTimeoutPerGiB: &timeout,
				}
				tests.UpdateKubeVirtConfigValueAndWait(cfg)
			})
			Context("without progress", func() {

				BeforeEach(func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration = &v1.MigrationConfiguration{
						ProgressTimeout:         pointer.Int64(5),
						CompletionTimeoutPerGiB: pointer.Int64(5),
						BandwidthPerMigration:   resource.NewQuantity(1, resource.BinarySI),
					}
					tests.UpdateKubeVirtConfigValueAndWait(cfg)
				})

				It("[test_id:2227] should abort a vmi migration", func() {
					vmi := tests.NewRandomFedoraVMI()
					vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					runStressTest(vmi, stressLargeVMSize, stressDefaultSleepDuration)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)
				})

			})
			It("[test_id:6978] Should detect a failed migration", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				emulator := filepath.Base(strings.TrimPrefix(domSpec.Devices.Emulator, "/"))
				// ensure that we only match the process
				emulator = "[" + emulator[0:1] + "]" + emulator[1:]

				// launch killer pod on every node that isn't the vmi's node
				By("Starting our migration killer pods")
				nodes := libnode.GetAllSchedulableNodes(virtClient)
				Expect(nodes.Items).ToNot(BeEmpty(), "There should be some compute node")
				for idx, entry := range nodes.Items {
					if entry.Name == vmi.Status.NodeName {
						continue
					}

					podName := fmt.Sprintf("migration-killer-pod-%d", idx)

					// kill the handler right as we detect the qemu target process come online
					pod := tests.RenderPrivilegedPod(podName, []string{"/bin/bash", "-c"}, []string{fmt.Sprintf("while true; do ps aux | grep -v \"defunct\" | grep -v \"D\" | grep \"%s\" && pkill -9 virt-handler && sleep 5; done", emulator)})

					pod.Spec.NodeName = entry.Name
					createdPod, err := virtClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should create helper pod")
					createdPods = append(createdPods, createdPod.Name)
				}
				Expect(createdPods).ToNot(BeEmpty(), "There is no node for migration")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)

				By("Removing our migration killer pods")
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), podName, metav1.DeleteOptions{})

						if err != nil && errors.IsNotFound(err) {
							return nil
						}
						return err
					}, 10*time.Second, 1*time.Second).Should(Succeed(), "Should delete helper pod")

					Eventually(func() error {
						_, err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Get(context.Background(), podName, metav1.GetOptions{})
						return err
					}, 300*time.Second, 1*time.Second).Should(
						SatisfyAll(HaveOccurred(), WithTransform(errors.IsNotFound, BeTrue())),
						"The killer pod should be gone within the given timeout",
					)
				}

				By("Waiting for virt-handler to come back online")
				Eventually(func() error {
					handler, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
					if err != nil {
						return err
					}

					if handler.Status.DesiredNumberScheduled == handler.Status.NumberAvailable {
						return nil
					}
					return fmt.Errorf("waiting for virt-handler pod to come back online")
				}, 120*time.Second, 1*time.Second).Should(Succeed(), "Virt handler should come online")

				By("Starting new migration and waiting for it to succeed")
				migration = tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("old finalized migrations should get garbage collected", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestForceLauncherMigrationFailureAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				for i := 0; i < 10; i++ {
					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration.Name = fmt.Sprintf("%s-iter-%d", vmi.Name, i)
					migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)

					Eventually(func() error {
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())

						if pod.Status.Phase == k8sv1.PodFailed || pod.Status.Phase == k8sv1.PodSucceeded {
							return nil
						}

						return fmt.Errorf("still waiting on target pod to complete, current phase is %s", pod.Status.Phase)
					}, 10*time.Second, time.Second).Should(Succeed(), "Target pod should exit quickly after migration fails.")
				}

				migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(&metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(migrations.Items).To(HaveLen(5))
			})

			It("[test_id:6979]Target pod should exit after failed migration", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestForceLauncherMigrationFailureAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)

				Eventually(func() error {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					pod, err := virtClient.CoreV1().Pods(vmi.Namespace).Get(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if pod.Status.Phase == k8sv1.PodFailed || pod.Status.Phase == k8sv1.PodSucceeded {
						return nil
					}

					return fmt.Errorf("still waiting on target pod to complete, current phase is %s", pod.Status.Phase)
				}, 10*time.Second, time.Second).Should(Succeed(), "Target pod should exit quickly after migration fails.")
			})

			It("[test_id:6980]Migration should fail if target pod fails during target preparation", func() {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("1Gi")

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					phase := migration.Status.Phase
					Expect(phase).NotTo(Equal(v1.MigrationSucceeded))
					return phase
				}, 120, 1*time.Second).Should(Equal(v1.MigrationPreparingTarget))

				By("Killing the target pod and expecting failure")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))

				err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Expecting VMI migration failure")
				Eventually(func() error {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmi.Status.MigrationState).ToNot(BeNil())

					if !vmi.Status.MigrationState.Failed {
						return fmt.Errorf("Waiting on vmi's migration state to be marked as failed")
					}

					// once set to failed, we expect start and end times and completion to be set as well.
					Expect(vmi.Status.MigrationState.StartTimestamp).ToNot(BeNil())
					Expect(vmi.Status.MigrationState.EndTimestamp).ToNot(BeNil())
					Expect(vmi.Status.MigrationState.Completed).To(BeTrue())

					return nil
				}, 120*time.Second, time.Second).Should(Succeed(), "vmi's migration state should be finalized as failed after target pod exits")
			})
			It("Migration should generate empty isos of the right size on the target", func() {
				By("Creating a VMI with cloud-init and config maps")
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				configMapName := "configmap-" + rand.String(5)
				secretName := "secret-" + rand.String(5)
				downwardAPIName := "downwardapi-" + rand.String(5)
				config_data := map[string]string{
					"config1": "value1",
					"config2": "value2",
				}
				secret_data := map[string]string{
					"user":     "admin",
					"password": "community",
				}
				tests.CreateConfigMap(configMapName, vmi.Namespace, config_data)
				tests.CreateSecret(secretName, vmi.Namespace, secret_data)
				tests.AddConfigMapDisk(vmi, configMapName, configMapName)
				tests.AddSecretDisk(vmi, secretName, secretName)
				tests.AddServiceAccountDisk(vmi, "default")
				// In case there are no existing labels add labels to add some data to the downwardAPI disk
				if vmi.ObjectMeta.Labels == nil {
					vmi.ObjectMeta.Labels = map[string]string{downwardTestLabelKey: downwardTestLabelVal}
				}
				tests.AddLabelDownwardAPIVolume(vmi, downwardAPIName)

				// this annotation causes virt launcher to immediately fail a migration
				vmi.Annotations = map[string]string{v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation: ""}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					phase := migration.Status.Phase
					Expect(phase).NotTo(Equal(v1.MigrationSucceeded))
					return phase
				}, 120, 1*time.Second).Should(Equal(v1.MigrationPreparingTarget))

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))

				By("Sanity checking the volume status size and the actual virt-launcher file")
				for _, volume := range vmi.Spec.Volumes {
					for _, volType := range []string{"cloud-init", "configmap-", "default-", "downwardapi-", "secret-"} {
						if strings.HasPrefix(volume.Name, volType) {
							for _, volStatus := range vmi.Status.VolumeStatus {
								if volStatus.Name == volume.Name {
									Expect(volStatus.Size).To(BeNumerically(">", 0), "Size of volume %s is 0", volume.Name)
									volPath, found := virthandler.IsoGuestVolumePath(vmi, &volume)
									if !found {
										continue
									}
									// Wait for the iso to be created
									Eventually(func() error {
										output, err := tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "[[ -f " + volPath + " ]] && echo found || true"})
										if err != nil {
											return err
										}
										if !strings.Contains(output, "found") {
											return fmt.Errorf("%s never appeared", volPath)
										}
										return nil
									}, 30*time.Second, time.Second).Should(Not(HaveOccurred()))
									output, err := tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "/usr/bin/stat --printf=%s " + volPath})
									Expect(err).ToNot(HaveOccurred())
									Expect(strconv.Atoi(output)).To(Equal(int(volStatus.Size)), "ISO file for volume %s is not the right size", volume.Name)
									output, err = tests.RunCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", fmt.Sprintf(`/usr/bin/cmp -n %d %s /dev/zero || true`, volStatus.Size, volPath)})
									Expect(err).ToNot(HaveOccurred())
									Expect(output).ToNot(ContainSubstring("differ"), "ISO file for volume %s is not empty", volume.Name)
								}
							}
						}
					}
				}
			})
		})
		Context("[storage-req]with an Alpine non-shared block volume PVC", decorators.StorageReq, func() {

			It("[test_id:1862][posneg:negative]should reject migrations for a non-migratable vmi", func() {
				// Start the VirtualMachineInstance with the PVC attached

				vmi, _ := tests.NewRandomVirtualMachineInstanceWithBlockDisk(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), testsuite.GetTestNamespace(nil), k8sv1.ReadWriteOnce)
				vmi.Spec.Hostname = string(cd.ContainerDiskAlpine)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(vmi).Should(HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))

				// execute a migration, wait for finalized state
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))
			})
		})

		Context("live migration cancelation", func() {
			type vmiBuilder func() *v1.VirtualMachineInstance

			newVirtualMachineInstanceWithFedoraContainerDisk := func() *v1.VirtualMachineInstance {
				return tests.NewRandomFedoraVMI()
			}

			newVirtualMachineInstanceWithFedoraRWXBlockDisk := func() *v1.VirtualMachineInstance {
				if !libstorage.HasCDI() {
					Skip("Skip DataVolume tests when CDI is not present")
				}

				sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteMany)
				if !foundSC {
					Skip("Skip test when Block storage is not present")
				}

				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling), cdiv1.RegistryPullNode),
					libdv.WithPVC(
						libdv.PVCWithStorageClass(sc),
						libdv.PVCWithVolumeSize(cd.FedoraVolumeSize),
						libdv.PVCWithReadWriteManyAccessMode(),
						libdv.PVCWithBlockVolumeMode(),
					),
				)

				dv, err = virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dv, 600, HaveSucceeded())
				vmi := tests.NewRandomVMIWithDataVolume(dv.Name)
				tests.AddUserData(vmi, "disk1", "#!/bin/bash\n echo hello\n")
				return vmi
			}

			DescribeTable("should be able to cancel a migration", decorators.SigStorage, func(createVMI vmiBuilder, with_virtctl bool) {
				vmi := createVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				migration = libmigration.RunAndCancelMigration(migration, vmi, with_virtctl, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigrationAborted(vmi, string(migration.UID), 180)

				By("Waiting for the migration object to disappear")
				libwait.WaitForMigrationToDisappearWithTimeout(migration, 240)
			},
				Entry("[sig-storage][test_id:2226] with ContainerDisk", newVirtualMachineInstanceWithFedoraContainerDisk, false),
				Entry("[sig-storage][storage-req][test_id:2731] with RWX block disk from block volume PVC", decorators.StorageReq, newVirtualMachineInstanceWithFedoraRWXBlockDisk, false),
				Entry("[sig-storage][test_id:2228] with ContainerDisk and virtctl", newVirtualMachineInstanceWithFedoraContainerDisk, true),
				Entry("[sig-storage][storage-req][test_id:2732] with RWX block disk and virtctl", decorators.StorageReq, newVirtualMachineInstanceWithFedoraRWXBlockDisk, true))

			DescribeTable("Immediate migration cancellation after migration starts running", func(with_virtctl bool) {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
				sourcePod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				migration = libmigration.RunAndImmediatelyCancelMigration(migration, vmi, with_virtctl, 60)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigrationAborted(vmi, string(migration.UID), 60)

				By("Waiting for the target virt-launcher pod to disappear")
				labelSelector, err := labels.Parse(fmt.Sprintf("%s=virt-launcher,%s=%s", v1.AppLabel, v1.CreatedByLabel, string(vmi.GetUID())))
				Expect(err).NotTo(HaveOccurred())

				Eventually(func() error {
					vmiPods, err := virtClient.CoreV1().Pods(vmi.GetNamespace()).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					Expect(err).NotTo(HaveOccurred())
					Expect(len(vmiPods.Items)).To(BeNumerically("<=", 2), "vmi has 3 active pods")

					if len(vmiPods.Items) == 1 {
						return nil
					}

					var targetPodPhase k8sv1.PodPhase
					for _, pod := range vmiPods.Items {
						if pod.Name == sourcePod.Name {
							continue
						}

						targetPodPhase = pod.Status.Phase
					}

					Expect(targetPodPhase).ToNot(BeEmpty())

					if targetPodPhase != k8sv1.PodSucceeded && targetPodPhase != k8sv1.PodFailed {
						return fmt.Errorf("pod phase is not expected to be %v", targetPodPhase)
					}

					return nil
				}, 30*time.Second, 2*time.Second).ShouldNot(HaveOccurred(), "target migration pod is expected to disappear after migration cancellation")

				By("Waiting for the migration object to disappear")
				libwait.WaitForMigrationToDisappearWithTimeout(migration, 20)
			},
				Entry("[sig-compute][test_id:3241]cancel a migration by deleting vmim object", false),
				Entry("[sig-compute][test_id:8583]cancel a migration with virtctl", true),
			)

			DescribeTable("Immediate migration cancellation before migration starts running", func(with_virtctl bool) {
				vmi := tests.NewRandomFedoraVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
				vmiOriginalNode := vmi.Status.NodeName

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				const timeout = 180
				Eventually(func() error {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(migration, &metav1.CreateOptions{})
					return err
				}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

				By("Waiting until the Migration has UID")
				Eventually(func() bool {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return migration.UID != ""
				}, timeout, 1*time.Second).Should(BeTrue())

				By("Cancelling migration")
				libmigration.CancelMigration(migration, vmi.Name, with_virtctl)

				By("Waiting for the migration object to disappear")
				libwait.WaitForMigrationToDisappearWithTimeout(migration, 240)

				By("Retrieving the VMI post migration")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying the VMI's migration state")
				Expect(vmi.Status.MigrationState).To(BeNil())

				By("Verifying the VMI's is in the running state and on original node")
				Expect(vmi.Status.Phase).To(Equal(v1.Running))
				Expect(vmi.Status.NodeName).To(Equal(vmiOriginalNode), "expecting VMI to not migrate")
			},
				Entry("[sig-compute][test_id:8584]cancel a migration by deleting vmim object", false),
				Entry("[sig-compute][test_id:8585]cancel a migration with virtctl", true),
			)

			Context("[Serial]when target pod cannot be scheduled and is suck in Pending phase", Serial, func() {

				var nodesSetUnschedulable []string

				BeforeEach(func() {
					By("Keeping only one schedulable node")
					schedulableNodes := libnode.GetAllSchedulableNodes(virtClient).Items
					Expect(schedulableNodes).NotTo(And(BeEmpty(), HaveLen(1)))

					// Iterate on all schedulable nodes but one
					for _, schedulableNode := range schedulableNodes[:len(schedulableNodes)-1] {
						libnode.SetNodeUnschedulable(schedulableNode.Name, virtClient)
						nodesSetUnschedulable = append(nodesSetUnschedulable, schedulableNode.Name)
					}
				})

				AfterEach(func() {
					By("Restoring nodes to be schedulable")
					for _, schedulableNodeName := range nodesSetUnschedulable {
						libnode.SetNodeSchedulable(schedulableNodeName, virtClient)
					}
				})

				It("should be able to properly abort migration", func() {
					By("Starting a VirtualMachineInstance")
					vmi := tests.NewRandomVMI()
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Trying to migrate VM and expect for the migration to get stuck")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigration(virtClient, migration)
					expectMigrationSchedulingPhase := func() v1.VirtualMachineInstanceMigrationPhase {
						migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
						Expect(err).ShouldNot(HaveOccurred())

						return migration.Status.Phase
					}
					Eventually(expectMigrationSchedulingPhase, 30*time.Second, 1*time.Second).Should(Equal(v1.MigrationScheduling))
					Consistently(expectMigrationSchedulingPhase, 60*time.Second, 5*time.Second).Should(Equal(v1.MigrationScheduling))

					By("Finding VMI's pod and expecting one to be running and the other to be pending")
					labelSelector, err := labels.Parse(fmt.Sprintf(v1.AppLabel + "=virt-launcher," + v1.CreatedByLabel + "=" + string(vmi.GetUID())))
					Expect(err).ShouldNot(HaveOccurred())

					vmiPods, err := virtClient.CoreV1().Pods(vmi.GetNamespace()).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector.String()})
					Expect(err).ShouldNot(HaveOccurred())
					Expect(vmiPods.Items).To(HaveLen(2), "two pods are expected for stuck vmi: source and target pods")

					var sourcePod *k8sv1.Pod
					for _, pod := range vmiPods.Items {

						if pod.Status.Phase == k8sv1.PodRunning {
							sourcePod = pod.DeepCopy()
						} else {
							Expect(pod.Status.Phase).ToNot(Or(Equal(k8sv1.PodSucceeded), Equal(k8sv1.PodFailed), Equal(k8sv1.PodUnknown)),
								"VMI is expected to have exactly 2 pods: one running and one pending")
						}
					}

					By("Aborting the migration")
					err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(migration.Name, &metav1.DeleteOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Expecting migration to be deleted")
					Eventually(func() bool {
						migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
						return errors.IsNotFound(err)
					}, 60*time.Second, 5*time.Second).Should(BeTrue(), `expecting to get "is not found" error`)

					By("Making sure source pod is still running")
					sourcePod, err = virtClient.CoreV1().Pods(sourcePod.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					Expect(sourcePod.Status.Phase).To(Equal(k8sv1.PodRunning))

					By("Making sure the VMI's migration state remains nil")
					Consistently(func() error {
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
						if err != nil {
							return err
						}

						if vmi.Status.MigrationState != nil {
							return fmt.Errorf("migration state is expected to be nil")
						}

						return nil
					}, 30*time.Second, 5*time.Second).ShouldNot(HaveOccurred())
				})

			})

		})

		Context("with a host-model cpu", func() {
			getNodeHostModel := func(node *k8sv1.Node) (hostModel string) {
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelCPULabel) {
						hostModel = strings.TrimPrefix(key, v1.HostModelCPULabel)
						break
					}
				}
				Expect(hostModel).ToNot(BeEmpty(), "must find node's host model")
				return hostModel
			}
			getNodeHostRequiredFeatures := func(node *k8sv1.Node) (features []string) {
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelRequiredFeaturesLabel) {
						features = append(features, strings.TrimPrefix(key, v1.HostModelRequiredFeaturesLabel))
					}
				}
				return features
			}
			isModelSupportedOnNode := func(node *k8sv1.Node, model string) bool {
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.HostModelCPULabel) && strings.Contains(key, model) {
						return true
					}
				}
				return false
			}
			isFeatureSupported := func(node *k8sv1.Node, feature string) bool {
				for key := range node.Labels {
					if strings.HasPrefix(key, v1.CPUFeatureLabel) && strings.Contains(key, feature) {
						return true
					}
				}
				return false
			}
			expectFeatureToBeSupportedOnNode := func(node *k8sv1.Node, features []string) {
				supportedFeatures := make(map[string]bool)
				for _, feature := range features {
					supportedFeatures[feature] = isFeatureSupported(node, feature)
				}

				Expect(supportedFeatures).Should(Not(ContainElement(false)),
					"copy features must be supported on node")
			}

			It("[test_id:6981]should migrate only to nodes supporting right cpu model", func() {
				sourceNode, targetNode, err := libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
				if err != nil {
					Skip(err.Error())
				}

				By("Creating a VMI with default CPU mode to land in source node")
				vmi := libvmi.NewAlpine(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
					libvmi.WithCPUModel(v1.CPUModeHostModel),
				)
				By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
				nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
				Expect(err).ToNot(HaveOccurred())
				vmi.Spec.Affinity = &k8sv1.Affinity{
					NodeAffinity: nodeAffinityRule,
				}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)
				Expect(vmi.Spec.Domain.CPU.Model).To(Equal(v1.CPUModeHostModel))

				By("Fetching original host CPU model & supported CPU features")
				originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				hostModel := getNodeHostModel(originalNode)
				requiredFeatures := getNodeHostRequiredFeatures(originalNode)

				By("Starting the migration and expecting it to end successfully")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				By("Ensuring that target pod has correct nodeSelector label")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(vmiPod.Spec.NodeSelector).To(HaveKey(v1.SupportedHostModelMigrationCPU+hostModel),
					"target pod is expected to have correct nodeSelector label defined")

				By("Ensuring that target node has correct CPU mode & features")
				newNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(isModelSupportedOnNode(newNode, hostModel)).To(BeTrue(), "original host model should be supported on new node")
				expectFeatureToBeSupportedOnNode(newNode, requiredFeatures)
			})

			Context("[Serial]Should trigger event if vmi with host-model start on source node with uniq host-model", Serial, func() {

				var vmi *v1.VirtualMachineInstance
				var node *k8sv1.Node

				const fakeHostModelLabel = v1.HostModelCPULabel + "fake-model"

				BeforeEach(func() {
					By("Creating a VMI with default CPU mode")
					vmi = alpineVMIWithEvictionStrategy()
					vmi.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Saving the original node's state")
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), vmi.Status.NodeName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					node = libinfra.ExpectStoppingNodeLabellerToSucceed(node.Name, virtClient)
				})

				AfterEach(func() {
					By("Resuming node labeller")
					node = libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, virtClient)
					_, doesFakeHostLabelExists := node.Labels[fakeHostModelLabel]
					Expect(doesFakeHostLabelExists).To(BeFalse(), fmt.Sprintf("label %s is expected to disappear from node %s", fakeHostModelLabel, node.Name))
				})

				It("[test_id:7505]when no node is suited for host model", func() {
					By("Changing node labels to support fake host model")
					// Remove all supported host models
					for key := range node.Labels {
						if strings.HasPrefix(key, v1.HostModelCPULabel) {
							libnode.RemoveLabelFromNode(node.Name, key)
						}
					}
					node = libnode.AddLabelToNode(node.Name, fakeHostModelLabel, "true")

					Eventually(func() bool {
						node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
						Expect(err).ShouldNot(HaveOccurred())

						labelValue, ok := node.Labels[v1.HostModelCPULabel+"fake-model"]
						return ok && labelValue == "true"
					}, 10*time.Second, 1*time.Second).Should(BeTrue(), "Node should have fake host model")

					By("Starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					_ = libmigration.RunMigration(virtClient, migration)

					events.ExpectEvent(vmi, k8sv1.EventTypeWarning, watch.NoSuitableNodesForHostModelMigration)
				})

			})

			Context("[Serial]Should trigger event if the nodes doesn't contain MigrationSelectorLabel for the vmi host-model type", Serial, func() {

				var vmi *v1.VirtualMachineInstance
				var nodes []k8sv1.Node

				BeforeEach(func() {
					nodes = libnode.GetAllSchedulableNodes(virtClient).Items
					if len(nodes) == 1 || len(nodes) > 10 {
						Skip("This test can't run with single node and it's too slow to run with more than 10 nodes")
					}

					By("Creating a VMI with default CPU mode")
					vmi = alpineVMIWithEvictionStrategy()
					vmi.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}

					By("Starting the VirtualMachineInstance")
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					for indx, node := range nodes {
						patchedNode := libinfra.ExpectStoppingNodeLabellerToSucceed(node.Name, virtClient)
						Expect(patchedNode).ToNot(BeNil())
						nodes[indx] = *patchedNode
					}
				})

				AfterEach(func() {
					By("Restore node to its original state")
					for _, node := range nodes {
						updatedNode := libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, virtClient)

						supportedHostModelLabelExists := false
						for labelKey := range updatedNode.Labels {
							if strings.HasPrefix(labelKey, v1.SupportedHostModelMigrationCPU) {
								supportedHostModelLabelExists = true
								break
							}
						}
						Expect(supportedHostModelLabelExists).To(BeTrue(), fmt.Sprintf("label with %s prefix is supposed to exist for node %s", v1.SupportedHostModelMigrationCPU, updatedNode.Name))
					}
				})

				It("no node contain suited SupportedHostModelMigrationCPU label", func() {
					By("Changing node labels to support fake host model")
					// Remove all supported host models
					for _, node := range nodes {
						currNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
						Expect(err).ShouldNot(HaveOccurred())
						for key := range currNode.Labels {
							if strings.HasPrefix(key, v1.SupportedHostModelMigrationCPU) {
								libnode.RemoveLabelFromNode(currNode.Name, key)
							}
						}
					}

					By("Starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					_ = libmigration.RunMigration(virtClient, migration)

					events.ExpectEvent(vmi, k8sv1.EventTypeWarning, watch.NoSuitableNodesForHostModelMigration)
				})

			})

		})

		Context("parallel migration threads", func() {
			const numberOfMigrationThreads uint = 4

			newVmi := func() *v1.VirtualMachineInstance {
				return libvmi.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
			}

			Context("should", func() {
				setMigrationParallelismWithAnnotation := func(vmi *v1.VirtualMachineInstance) {
					if vmi.Annotations == nil {
						vmi.Annotations = map[string]string{}
					}
					vmi.Annotations[cmdclient.MultiThreadedQemuMigrationAnnotation] = fmt.Sprintf("%d", numberOfMigrationThreads)
				}

				DescribeTable("run successfully when configured through", func(setParallelMigration func(vmi *v1.VirtualMachineInstance)) {
					vmi := newVmi()

					By("Setting parallel migration")
					setParallelMigration(vmi)

					By("Running vmi %s" + vmi.Name)
					vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

					By("Starting the Migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					_ = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)
				},
					Entry("a VMI annotation", setMigrationParallelismWithAnnotation),
				)
			})
		})

		Context("[Serial] with migration policies", Serial, func() {

			confirmMigrationPolicyName := func(vmi *v1.VirtualMachineInstance, expectedName *string) {
				By("Verifying the VMI's configuration source")
				if expectedName == nil {
					Expect(vmi.Status.MigrationState.MigrationPolicyName).To(BeNil())
				} else {
					Expect(vmi.Status.MigrationState.MigrationPolicyName).ToNot(BeNil())
					Expect(*vmi.Status.MigrationState.MigrationPolicyName).To(Equal(*expectedName))
				}
			}

			DescribeTable("migration policy", func(defineMigrationPolicy bool) {
				By("Updating config to allow auto converge")
				config := getCurrentKvConfig(virtClient)
				config.MigrationConfiguration.AllowAutoConverge = pointer.BoolPtr(true)
				tests.UpdateKubeVirtConfigValueAndWait(config)

				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))

				var expectedPolicyName *string
				if defineMigrationPolicy {
					By("Creating a migration policy that overrides cluster policy")
					policy := GeneratePolicyAndAlignVMI(vmi)
					policy.Spec.AllowAutoConverge = pointer.BoolPtr(false)

					_, err := virtClient.MigrationPolicy().Create(context.Background(), policy, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					expectedPolicyName = &policy.Name
				}

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Starting the Migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Retrieving the VMI post migration")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Status.MigrationState.MigrationConfiguration).ToNot(BeNil())
				confirmMigrationPolicyName(vmi, expectedPolicyName)
			},
				Entry("should override cluster-wide policy if defined", true),
				Entry("should not affect cluster-wide policy if not defined", false),
			)

		})

		Context("[Serial] with freePageReporting", Serial, func() {

			BeforeEach(func() {
				kv := util.GetCurrentKv(virtClient)
				kvConfigurationCopy := kv.Spec.Configuration.DeepCopy()
				kvConfigurationCopy.VirtualMachineOptions = nil
				tests.UpdateKubeVirtConfigValueAndWait(*kvConfigurationCopy)
			})

			It("should be able to migrate", func() {
				vmi := libvmi.NewAlpineWithTestTooling(
					libvmi.WithMasqueradeNetworking()...,
				)

				By("Starting the VirtualMachineInstance")
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				Expect(domSpec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo("on"))

				By("starting the migration")
				migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				domXml, err = tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())
				domSpec = &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				Expect(domSpec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo("on"))
			})

		})
	})

	Context("with sata disks", func() {

		It("[test_id:1853]VM with containerDisk + CloudInit + ServiceAccount + ConfigMap + Secret + DownwardAPI + External Kernel Boot + USB Disk", func() {
			vmi := prepareVMIWithAllVolumeSources(testsuite.GetTestNamespace(nil))

			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(7))
			Expect(vmi.Spec.Domain.Devices.Interfaces).To(HaveLen(1))

			vmi = tests.RunVMIAndExpectLaunch(vmi, 180)

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
		})
	})

	Context("[test_id:8482] Migration Metrics", func() {
		It("exposed to prometheus during VM migration", func() {
			vmi := tests.NewRandomFedoraVMI()
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

			By("Limiting the bandwidth of migrations in the test namespace")
			CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

			By("Starting the VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			// Need to wait for cloud init to finnish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndCollectMigrationMetrics(vmi, migration)
		})
	})

	Context("[Serial] With Huge Pages", Serial, func() {
		var hugepagesVmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			hugepagesVmi = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
		})

		DescribeTable("should consume hugepages ", func(hugepageSize string, memory string) {
			hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + hugepageSize)
			v, err := cluster.GetKubernetesVersion()
			Expect(err).ShouldNot(HaveOccurred())
			if strings.Contains(v, "1.16") {
				hugepagesVmi.Annotations = map[string]string{
					v1.MemfdMemoryBackend: "false",
				}
				log.DefaultLogger().Object(hugepagesVmi).Infof("Fall back to use hugepages source file. Libvirt in the 1.16 provider version doesn't support memfd as memory backend")
			}

			count := 0
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			requestedMemory := resource.MustParse(memory)
			hugepagesVmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = requestedMemory

			for _, node := range nodes.Items {
				// Cmp returns -1, 0, or 1 for less than, equal to, or greater than
				if v, ok := node.Status.Capacity[hugepageType]; ok && v.Cmp(requestedMemory) == 1 {
					count += 1
				}
			}

			if count < 2 {
				Skip(fmt.Sprintf("Not enough nodes with hugepages %s capacity. Need 2, found %d.", hugepageType, count))
			}

			hugepagesVmi.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{PageSize: hugepageSize},
			}

			By("Starting hugepages VMI")
			_, err = virtClient.VirtualMachineInstance(hugepagesVmi.Namespace).Create(context.Background(), hugepagesVmi)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(hugepagesVmi)

			By("starting the migration")
			migration := tests.NewRandomMigration(hugepagesVmi.Name, hugepagesVmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, hugepagesVmi, migration)
		},
			Entry("[test_id:6983]hugepages-2Mi", "2Mi", "64Mi"),
			Entry("[test_id:6984]hugepages-1Gi", "1Gi", "1Gi"),
		)
	})

	Context("[Serial] with CPU pinning and huge pages", Serial, func() {
		It("should not make migrations fail", func() {
			checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(2)
			var err error
			cpuVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
			cpuVMI.Spec.Domain.CPU = &v1.CPU{
				Cores:                 3,
				DedicatedCPUPlacement: true,
			}
			cpuVMI.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{PageSize: "2Mi"},
			}

			By("Starting a VirtualMachineInstance")
			cpuVMI, err = virtClient.VirtualMachineInstance(cpuVMI.Namespace).Create(context.Background(), cpuVMI)
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(cpuVMI)

			By("Performing a migration")
			migration := tests.NewRandomMigration(cpuVMI.Name, cpuVMI.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
		})
		Context("and NUMA passthrough", func() {
			It("should not make migrations fail", func() {
				checks.SkipTestIfNoFeatureGate(virtconfig.NUMAFeatureGate)
				checks.SkipTestIfNotEnoughNodesWithCPUManagerWith2MiHugepages(2)
				var err error
				cpuVMI := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
				cpuVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("128Mi")
				cpuVMI.Spec.Domain.CPU = &v1.CPU{
					Cores:                 3,
					DedicatedCPUPlacement: true,
					NUMA:                  &v1.NUMA{GuestMappingPassthrough: &v1.NUMAGuestMappingPassthrough{}},
				}
				cpuVMI.Spec.Domain.Memory = &v1.Memory{
					Hugepages: &v1.Hugepages{PageSize: "2Mi"},
				}

				By("Starting a VirtualMachineInstance")
				cpuVMI, err = virtClient.VirtualMachineInstance(cpuVMI.Namespace).Create(context.Background(), cpuVMI)
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(cpuVMI)

				By("Performing a migration")
				migration := tests.NewRandomMigration(cpuVMI.Name, cpuVMI.Namespace)
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			})
		})
	})

	It("should replace containerdisk and kernel boot images with their reproducible digest during migration", func() {

		vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
		vmi.Spec.Domain.Firmware = utils.GetVMIKernelBootWithRandName().Spec.Domain.Firmware

		By("Starting a VirtualMachineInstance")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Create(context.Background(), vmi)
		Expect(err).ToNot(HaveOccurred())
		libwait.WaitForSuccessfulVMIStart(vmi)

		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
		By("Verifying that all relevant images are without the digest on the source")
		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			if container.Name == "container-disk-binary" || container.Name == "compute" || container.Name == "guest-console-log" {
				continue
			}
			Expect(container.Image).ToNot(ContainSubstring("@sha256:"), "image:%s should not contain the container digest for container %s", container.Image, container.Name)
		}

		digestRegex := regexp.MustCompile(`sha256:[a-zA-Z0-9]+`)

		By("Collecting digest information from the container statuses")
		imageIDs := map[string]string{}
		for _, status := range append(pod.Status.ContainerStatuses, pod.Status.InitContainerStatuses...) {
			if status.Name == "container-disk-binary" || status.Name == "compute" || status.Name == "guest-console-log" {
				continue
			}
			digest := digestRegex.FindString(status.ImageID)
			Expect(digest).ToNot(BeEmpty())
			imageIDs[status.Name] = digest
		}

		By("Performing a migration")
		migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
		libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

		By("Verifying that all imageIDs are in a reproducible form on the target")
		pod = tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

		for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
			if container.Name == "container-disk-binary" || container.Name == "compute" || container.Name == "guest-console-log" {
				continue
			}
			digest := digestRegex.FindString(container.Image)
			Expect(container.Image).To(ContainSubstring(digest), "image:%s should contain the container digest for container %s", container.Image, container.Name)
			Expect(digest).ToNot(BeEmpty())
			Expect(imageIDs).To(HaveKeyWithValue(container.Name, digest), "expected image:%s for container %s to be the same like on the source pod but got %s", container.Image, container.Name, imageIDs[container.Name])
		}
	})

	Context("[Serial]Testing host-model cpuModel edge cases in the cluster if the cluster is host-model migratable", Serial, func() {

		var sourceNode *k8sv1.Node
		var targetNode *k8sv1.Node

		const fakeRequiredFeature = v1.HostModelRequiredFeaturesLabel + "fakeFeature"
		const fakeHostModel = v1.HostModelCPULabel + "fakeHostModel"

		BeforeEach(func() {
			sourceNode, targetNode, err = libmigration.GetValidSourceNodeAndTargetNodeForHostModelMigration(virtClient)
			if err != nil {
				Skip(err.Error())
			}
			targetNode = libinfra.ExpectStoppingNodeLabellerToSucceed(targetNode.Name, virtClient)
		})

		AfterEach(func() {
			By("Resuming node labeller")
			targetNode = libinfra.ExpectResumingNodeLabellerToSucceed(targetNode.Name, virtClient)

			By("Validating that fake labels are being removed")
			for _, labelKey := range []string{fakeRequiredFeature, fakeHostModel} {
				_, fakeLabelExists := targetNode.Labels[labelKey]
				Expect(fakeLabelExists).To(BeFalse(), fmt.Sprintf("fake feature %s is expected to disappear form node %s", labelKey, targetNode.Name))
			}
		})

		It("Should be able to migrate back to the initial node from target node with host-model even if target is newer than source", func() {
			libnode.AddLabelToNode(targetNode.Name, fakeRequiredFeature, "true")

			vmiToMigrate := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Creating a VMI with default CPU mode to land in source node")
			vmiToMigrate.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			By("Starting the VirtualMachineInstance")
			vmiToMigrate = tests.RunVMIAndExpectLaunch(vmiToMigrate, 240)
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration to target node(with the amazing feature")
			migration := tests.NewRandomMigration(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			vmiToMigrate, err = virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(targetNode.Name))

			labelsBeforeMigration := make(map[string]string)
			labelsAfterMigration := make(map[string]string)
			By("Fetching virt-launcher pod")
			virtLauncherPod := tests.GetRunningPodByVirtualMachineInstance(vmiToMigrate, vmiToMigrate.Namespace)
			for key, value := range virtLauncherPod.Spec.NodeSelector {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) {
					labelsBeforeMigration[key] = value
				}
			}

			By("Starting the Migration to return to the source node")
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			vmiToMigrate, err = virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			By("Fetching virt-launcher pod")
			virtLauncherPod = tests.GetRunningPodByVirtualMachineInstance(vmiToMigrate, vmiToMigrate.Namespace)
			for key, value := range virtLauncherPod.Spec.NodeSelector {
				if strings.HasPrefix(key, v1.CPUFeatureLabel) {
					labelsAfterMigration[key] = value
				}
			}
			Expect(labelsAfterMigration).To(BeEquivalentTo(labelsBeforeMigration))
		})

		It("vmi with host-model should be able to migrate to node that support the initial node's host-model even if this model isn't the target's host-model", func() {
			targetNode, err = virtClient.CoreV1().Nodes().Get(context.Background(), targetNode.Name, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())

			targetHostModel := tests.GetNodeHostModel(targetNode)
			targetNode = libnode.RemoveLabelFromNode(targetNode.Name, v1.HostModelCPULabel+targetHostModel)
			targetNode = libnode.AddLabelToNode(targetNode.Name, fakeHostModel, "true")

			vmiToMigrate := libvmi.NewFedora(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			By("Creating a VMI with default CPU mode to land in source node")
			vmiToMigrate.Spec.Domain.CPU = &v1.CPU{Model: v1.CPUModeHostModel}
			By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
			nodeAffinityRule, err := libmigration.CreateNodeAffinityRuleToMigrateFromSourceToTargetAndBack(sourceNode, targetNode)
			Expect(err).ToNot(HaveOccurred())
			vmiToMigrate.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: nodeAffinityRule,
			}
			By("Starting the VirtualMachineInstance")
			vmiToMigrate = tests.RunVMIAndExpectLaunch(vmiToMigrate, 240)
			Expect(vmiToMigrate.Status.NodeName).To(Equal(sourceNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

			// execute a migration, wait for finalized state
			By("Starting the Migration to target node(with the amazing feature")
			migration := tests.NewRandomMigration(vmiToMigrate.Name, vmiToMigrate.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			vmiToMigrate, err = virtClient.VirtualMachineInstance(vmiToMigrate.Namespace).Get(context.Background(), vmiToMigrate.GetName(), &metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(vmiToMigrate.Status.NodeName).To(Equal(targetNode.Name))
			Expect(console.LoginToFedora(vmiToMigrate)).To(Succeed())

		})
	})

	Context("with dedicated CPUs", func() {
		var (
			virtClient    kubecli.KubevirtClient
			err           error
			nodes         []k8sv1.Node
			migratableVMI *v1.VirtualMachineInstance
			pausePod      *k8sv1.Pod
			workerLabel   = "node-role.kubernetes.io/worker"
			testLabel1    = "kubevirt.io/testlabel1"
			testLabel2    = "kubevirt.io/testlabel2"
			cgroupVersion cgroup.CgroupVersion
		)

		parseVCPUPinOutput := func(vcpuPinOutput string) []int {
			var cpuSet []int
			vcpuPinOutputLines := strings.Split(vcpuPinOutput, "\n")
			cpuLines := vcpuPinOutputLines[2 : len(vcpuPinOutputLines)-2]

			for _, line := range cpuLines {
				lineSplits := strings.Fields(line)
				cpu, err := strconv.Atoi(lineSplits[1])
				Expect(err).ToNot(HaveOccurred(), "cpu id is non string in vcpupin output")

				cpuSet = append(cpuSet, cpu)
			}

			return cpuSet
		}

		getLibvirtDomainCPUSet := func(vmi *v1.VirtualMachineInstance) []int {
			pod, err := tests.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())

			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient,
				pod,
				"compute",
				[]string{"virsh", "vcpupin", fmt.Sprintf("%s_%s", vmi.GetNamespace(), vmi.GetName())})
			Expect(err).ToNot(HaveOccurred())
			Expect(stderr).To(BeEmpty())

			return parseVCPUPinOutput(stdout)
		}

		parseSysCpuSet := func(cpuset string) []int {
			set, err := hardware.ParseCPUSetLine(cpuset, 5000)
			Expect(err).ToNot(HaveOccurred())
			return set
		}

		getPodCPUSet := func(pod *k8sv1.Pod) []int {

			var cpusetPath string
			if cgroupVersion == cgroup.V2 {
				cpusetPath = "/sys/fs/cgroup/cpuset.cpus.effective"
			} else {
				cpusetPath = "/sys/fs/cgroup/cpuset/cpuset.cpus"
			}

			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient,
				pod,
				"compute",
				[]string{"cat", cpusetPath})
			Expect(err).ToNot(HaveOccurred())
			Expect(stderr).To(BeEmpty())

			return parseSysCpuSet(strings.TrimSpace(stdout))
		}

		getVirtLauncherCPUSet := func(vmi *v1.VirtualMachineInstance) []int {
			pod, err := tests.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())

			return getPodCPUSet(pod)
		}

		hasCommonCores := func(vmi *v1.VirtualMachineInstance, pod *k8sv1.Pod) bool {
			set1 := getVirtLauncherCPUSet(vmi)
			set2 := getPodCPUSet(pod)
			for _, corei := range set1 {
				for _, corej := range set2 {
					if corei == corej {
						return true
					}
				}
			}

			return false
		}

		BeforeEach(func() {
			// We will get focused to run on migration test lanes because we contain the word "Migration".
			// However, we need to be sig-something or we'll fail the check, even if we don't run on any sig- lane.
			// So let's be sig-compute and skip ourselves on sig-compute always... (they have only 1 node with CPU manager)
			checks.SkipTestIfNotEnoughNodesWithCPUManager(2)
			virtClient = kubevirt.Client()

			By("getting the list of worker nodes that have cpumanager enabled")
			nodeList, err := virtClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=,%s=%s", workerLabel, "cpumanager", "true"),
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(nodeList).ToNot(BeNil())
			nodes = nodeList.Items
			Expect(len(nodes)).To(BeNumerically(">=", 2), "at least two worker nodes with cpumanager are required for migration")

			By("creating a migratable VMI with 2 dedicated CPU cores")
			migratableVMI = tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			migratableVMI.Spec.Domain.CPU = &v1.CPU{
				Cores:                 uint32(2),
				DedicatedCPUPlacement: true,
			}
			migratableVMI.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("512Mi")

			By("creating a template for a pause pod with 1 dedicated CPU core")
			pausePod = tests.RenderPod("pause-", nil, nil)
			pausePod.Spec.Containers[0].Name = "compute"
			pausePod.Spec.Containers[0].Command = []string{"sleep"}
			pausePod.Spec.Containers[0].Args = []string{"3600"}
			pausePod.Spec.Containers[0].Resources = k8sv1.ResourceRequirements{
				Requests: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("1"),
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
				Limits: k8sv1.ResourceList{
					k8sv1.ResourceCPU:    resource.MustParse("1"),
					k8sv1.ResourceMemory: resource.MustParse("128Mi"),
				},
			}
			pausePod.Spec.Affinity = &k8sv1.Affinity{
				NodeAffinity: &k8sv1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &k8sv1.NodeSelector{
						NodeSelectorTerms: []k8sv1.NodeSelectorTerm{
							{
								MatchExpressions: []k8sv1.NodeSelectorRequirement{
									{Key: testLabel2, Operator: k8sv1.NodeSelectorOpIn, Values: []string{"true"}},
								},
							},
						},
					},
				},
			}
		})

		AfterEach(func() {
			libnode.RemoveLabelFromNode(nodes[0].Name, testLabel1)
			libnode.RemoveLabelFromNode(nodes[1].Name, testLabel2)
			libnode.RemoveLabelFromNode(nodes[1].Name, testLabel1)
		})

		It("should successfully update a VMI's CPU set on migration", func() {
			By("ensuring at least 2 worker nodes have cpumanager")
			Expect(len(nodes)).To(BeNumerically(">=", 2), "at least two worker nodes with cpumanager are required for migration")

			By("starting a VMI on the first node of the list")
			libnode.AddLabelToNode(nodes[0].Name, testLabel1, "true")
			vmi := tests.CreateVmiOnNodeLabeled(migratableVMI, testLabel1, "true")

			By("waiting until the VirtualMachineInstance starts")
			libwait.WaitForSuccessfulVMIStart(vmi,
				libwait.WithTimeout(120),
			)
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("determining cgroups version")
			cgroupVersion = getVMIsCgroupVersion(vmi, virtClient)

			By("ensuring the VMI started on the correct node")
			Expect(vmi.Status.NodeName).To(Equal(nodes[0].Name))

			By("reserving the cores used by the VMI on the second node with a paused pod")
			var pods []*k8sv1.Pod
			var pausedPod *k8sv1.Pod
			libnode.AddLabelToNode(nodes[1].Name, testLabel2, "true")
			for pausedPod = tests.RunPod(pausePod); !hasCommonCores(vmi, pausedPod); pausedPod = tests.RunPod(pausePod) {
				pods = append(pods, pausedPod)
				By("creating another paused pod since last didn't have common cores with the VMI")
			}

			By("deleting the paused pods that don't have cores in common with the VMI")
			for _, pod := range pods {
				err = virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("migrating the VMI from first node to second node")
			libnode.AddLabelToNode(nodes[1].Name, testLabel1, "true")
			cpuSetSource := getVirtLauncherCPUSet(vmi)
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("ensuring the target cpuset is different from the source")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "should have been able to retrieve the VMI instance")
			cpuSetTarget := getVirtLauncherCPUSet(vmi)
			Expect(cpuSetSource).NotTo(Equal(cpuSetTarget), "CPUSet of source launcher should not match targets one")

			By("ensuring the libvirt domain cpuset is equal to the virt-launcher pod cpuset")
			cpuSetTargetLibvirt := getLibvirtDomainCPUSet(vmi)
			Expect(cpuSetTargetLibvirt).To(Equal(cpuSetTarget))

			By("deleting the last paused pod")
			err = virtClient.CoreV1().Pods(pausedPod.Namespace).Delete(context.Background(), pausedPod.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("[Serial] with a dedicated migration network", Serial, func() {
		BeforeEach(func() {
			virtClient = kubevirt.Client()

			By("Creating the Network Attachment Definition")
			nad := libmigration.GenerateMigrationCNINetworkAttachmentDefinition()
			_, err = virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(flags.KubeVirtInstallNamespace).Create(context.TODO(), nad, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to create the Network Attachment Definition")

			By("Setting it as the migration network in the KubeVirt CR")
			libmigration.SetDedicatedMigrationNetwork(nad.Name)
		})
		AfterEach(func() {
			By("Clearing the migration network in the KubeVirt CR")
			libmigration.ClearDedicatedMigrationNetwork()

			By("Deleting the Network Attachment Definition")
			nad := libmigration.GenerateMigrationCNINetworkAttachmentDefinition()
			err = virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(flags.KubeVirtInstallNamespace).Delete(context.TODO(), nad.Name, metav1.DeleteOptions{})
			Expect(err).NotTo(HaveOccurred(), "Failed to delete the Network Attachment Definition")
		})
		It("Should migrate over that network", func() {
			vmi := libvmi.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Starting the migration")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Checking if the migration happened, and over the right network")
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			Expect(vmi.Status.MigrationState.TargetNodeAddress).To(HavePrefix("172.21.42."), "The migration did not appear to go over the dedicated migration network")
		})
	})

	It("should update MigrationState's MigrationConfiguration of VMI status", func() {
		By("Starting a VMI")
		vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
		vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

		By("Starting a Migration")
		migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)
		libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

		By("Ensuring MigrationConfiguration is updated")
		vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi.Status.MigrationState).ToNot(BeNil())
		Expect(vmi.Status.MigrationState.MigrationConfiguration).ToNot(BeNil())
	})

	Context("with a live-migration in flight", func() {
		It("there should always be a single active migration per VMI", func() {
			By("Starting a VMI")
			vmi := libvmi.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Checking that there always is at most one migration running")
			Consistently(func() int {
				vmim := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				// not checking err as the migration creation will be blocked immediately by virt-api's validating webhook
				// if another one is currently running
				vmim, err = virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(vmim, &metav1.CreateOptions{})

				labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationSelectorLabel, vmi.Name))
				Expect(err).ToNot(HaveOccurred())
				listOptions := &metav1.ListOptions{
					LabelSelector: labelSelector.String(),
				}
				migrations, err := virtClient.VirtualMachineInstanceMigration(vmim.Namespace).List(listOptions)
				Expect(err).ToNot(HaveOccurred())

				activeMigrations := 0
				for _, migration := range migrations.Items {
					switch migration.Status.Phase {
					case v1.MigrationScheduled, v1.MigrationPreparingTarget, v1.MigrationTargetReady, v1.MigrationRunning:
						activeMigrations += 1
					}
				}
				return activeMigrations

			}, time.Second*30, time.Second*1).Should(BeNumerically("<=", 1))
		})
	})

	Context("topology hints", decorators.Reenlightenment, decorators.TscFrequencies, func() {

		Context("needs to be set when", func() {

			expectTopologyHintsToBeSet := func(vmi *v1.VirtualMachineInstance) {
				EventuallyWithOffset(1, func() bool {
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return topology.AreTSCFrequencyTopologyHintsDefined(vmi)
				}, 90*time.Second, 3*time.Second).Should(BeTrue(), fmt.Sprintf("tsc frequency topology hints are expected to exist for vmi %s", vmi.Name))
			}

			It("invtsc feature exists", decorators.Invtsc, func() {
				vmi := libvmi.New(
					libvmi.WithResourceMemory("1Mi"),
					libvmi.WithCPUFeature("invtsc", "require"),
				)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				expectTopologyHintsToBeSet(vmi)
			})

			It("HyperV reenlightenment is enabled", func() {
				vmi := libvmi.NewWindows()
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{}
				vmi.Spec.Volumes = []v1.Volume{}
				vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: pointer.Bool(true)}
				vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

				expectTopologyHintsToBeSet(vmi)
			})

		})

	})

	Context("when evacuating fails", func() {
		var vmi *v1.VirtualMachineInstance

		setEvacuationAnnotation := func(migrations ...*v1.VirtualMachineInstanceMigration) {
			for _, m := range migrations {
				m.Annotations = map[string]string{
					v1.EvacuationMigrationAnnotation: m.Name,
				}
			}
		}

		BeforeEach(func() {
			vmi = libvmi.NewCirros(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""),
			)
		})

		It("retrying immediately should be blocked by the migration backoff", func() {
			By("Starting the VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Waiting for the migration to fail")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

			By("Try again")
			migration = tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

			events.ExpectEvent(vmi, k8sv1.EventTypeWarning, watch.MigrationBackoffReason)
		})

		It("after a successful migration backoff should be cleared", func() {
			By("Starting the VirtualMachineInstance")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Waiting for the migration to fail")
			migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigrationAndExpectFailure(migration, libmigration.MigrationWaitTime)

			By("Patch VMI")
			patchBytes := []byte(fmt.Sprintf(`[{"op": "remove", "path": "/metadata/annotations/%s"}]`, patch.EscapeJSONPointer(v1.FuncTestForceLauncherMigrationFailureAnnotation)))
			_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Patch(context.Background(), vmi.Name, types.JSONPatchType, patchBytes, &metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Try again with backoff")
			migration = tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// Intentionally modifying history
			events.DeleteEvents(vmi, k8sv1.EventTypeWarning, watch.MigrationBackoffReason)

			By("There should be no backoff now")
			migration = tests.NewRandomMigration(vmi.Name, vmi.Namespace)
			setEvacuationAnnotation(migration)
			_ = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Checking that no backoff event occurred")
			events.ExpectNoEvent(vmi, k8sv1.EventTypeWarning, watch.MigrationBackoffReason)
			events, err := virtClient.CoreV1().Events(util.NamespaceTestDefault).List(context.Background(), metav1.ListOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, ev := range events.Items {
				Expect(ev.Reason).ToNot(Equal(watch.MigrationBackoffReason))
			}
		})
	})

	Context("ResourceQuota rejection", func() {
		It("Should contain condition when migrating with quota that doesn't have resources for both source and target", func() {
			vmiRequest := resource.MustParse("200Mi")
			vmi := libvmi.NewCirros(
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithResourceMemory(vmiRequest.String()),
			)

			vmiRequest.Add(resource.MustParse("50Mi")) //add 50Mi memoryOverHead to make sure vmi creation won't be blocked
			enoughMemoryToStartVmiButNotEnoughForMigration := services.GetMemoryOverhead(vmi, runtime.GOARCH, nil)
			enoughMemoryToStartVmiButNotEnoughForMigration.Add(vmiRequest)
			resourcesToLimit := k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse(enoughMemoryToStartVmiButNotEnoughForMigration.String()),
			}

			By("Creating ResourceQuota with enough memory for the vmi but not enough for migration")
			resourceQuota := newResourceQuota(resourcesToLimit, testsuite.GetTestNamespace(vmi))
			_ = createResourceQuota(resourceQuota)

			By("Starting the VirtualMachineInstance")
			_ = tests.RunVMIAndExpectLaunch(vmi, 240)

			By("Trying to migrate the VirtualMachineInstance")
			migration := tests.NewRandomMigration(vmi.Name, testsuite.GetTestNamespace(vmi))
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(func() *v1.VirtualMachineInstanceMigration {
				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(migration.Name, &metav1.GetOptions{})
				if err != nil {
					return nil
				}
				return migration
			}, 60*time.Second, 1*time.Second).Should(HaveConditionTrue(v1.VirtualMachineInstanceMigrationRejectedByResourceQuota))
		})
	})
})

func createResourceQuota(resourceQuota *k8sv1.ResourceQuota) *k8sv1.ResourceQuota {
	virtCli := kubevirt.Client()

	var obj *k8sv1.ResourceQuota
	var err error
	obj, err = virtCli.CoreV1().ResourceQuotas(resourceQuota.Namespace).Create(context.Background(), resourceQuota, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	return obj
}

func newResourceQuota(hardResourcesLimitation k8sv1.ResourceList, namespace string) *k8sv1.ResourceQuota {
	return &k8sv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      "test-quota",
		},
		Spec: k8sv1.ResourceQuotaSpec{
			Hard: hardResourcesLimitation,
		},
	}
}

func temporaryTLSConfig() *tls.Config {
	// Generate new certs if secret doesn't already exist
	caKeyPair, _ := triple.NewCA("kubevirt.io", time.Hour)

	clientKeyPair, _ := triple.NewClientKeyPair(caKeyPair,
		"kubevirt.io:system:node:virt-handler",
		nil,
		time.Hour,
	)

	certPEM := cert.EncodeCertPEM(clientKeyPair.Cert)
	keyPEM := cert.EncodePrivateKeyPEM(clientKeyPair.Key)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	Expect(err).ToNot(HaveOccurred())
	return &tls.Config{
		InsecureSkipVerify: true,
		GetClientCertificate: func(info *tls.CertificateRequestInfo) (certificate *tls.Certificate, e error) {
			return &cert, nil
		},
	}
}

func libvirtDomainIsPersistent(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) (bool, error) {
	vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
		virtClient,
		vmiPod,
		tests.GetComputeContainerOfPod(vmiPod).Name,
		[]string{"virsh", "--quiet", "list", "--persistent", "--name"},
	)
	if err != nil {
		return false, fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v: %s", err, stderr)
	}
	return strings.Contains(stdout, vmi.Namespace+"_"+vmi.Name), nil
}

func getVMIsCgroupVersion(vmi *v1.VirtualMachineInstance, virtClient kubecli.KubevirtClient) cgroup.CgroupVersion {
	pod, err := tests.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
	Expect(err).ToNot(HaveOccurred())

	return getPodsCgroupVersion(pod, virtClient)
}

func getPodsCgroupVersion(pod *k8sv1.Pod, virtClient kubecli.KubevirtClient) cgroup.CgroupVersion {
	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient,
		pod,
		"compute",
		[]string{"stat", "/sys/fs/cgroup/", "-f", "-c", "%T"})

	Expect(err).ToNot(HaveOccurred())
	Expect(stderr).To(BeEmpty())

	cgroupFsType := strings.TrimSpace(stdout)

	if cgroupFsType == "cgroup2fs" {
		return cgroup.V2
	} else {
		return cgroup.V1
	}
}

func getCurrentKvConfig(virtClient kubecli.KubevirtClient) v1.KubeVirtConfiguration {
	kvc := util.GetCurrentKv(virtClient)

	if kvc.Spec.Configuration.MigrationConfiguration == nil {
		kvc.Spec.Configuration.MigrationConfiguration = &v1.MigrationConfiguration{}
	}

	if kvc.Spec.Configuration.DeveloperConfiguration == nil {
		kvc.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
	}

	if kvc.Spec.Configuration.NetworkConfiguration == nil {
		kvc.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{}
	}

	return kvc.Spec.Configuration
}

func runStressTest(vmi *v1.VirtualMachineInstance, vmsize string, stressTimeoutSeconds int) {
	By("Run a stress test to dirty some pages and slow down the migration")
	stressCmd := fmt.Sprintf("stress-ng --vm 1 --vm-bytes %s --vm-keep &\n", vmsize)
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: stressCmd},
		&expect.BExp{R: console.PromptExpression},
	}, 15)).To(Succeed(), "should run a stress test")

	// give stress tool some time to trash more memory pages before returning control to next steps
	if stressTimeoutSeconds < 15 {
		time.Sleep(time.Duration(stressTimeoutSeconds) * time.Second)
	} else {
		time.Sleep(15 * time.Second)
	}
}
