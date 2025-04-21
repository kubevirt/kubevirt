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
	"fmt"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	gomegatypes "github.com/onsi/gomega/types"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"kubevirt.io/api/migrations/v1alpha1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	cdiv1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libdv"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libvmici "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
	virthandler "kubevirt.io/kubevirt/pkg/virt-handler"
	"kubevirt.io/kubevirt/pkg/virt-handler/cgroup"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnet/job"
	"kubevirt.io/kubevirt/tests/libnet/service"
	"kubevirt.io/kubevirt/tests/libnet/vmnetserver"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libstorage"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/watcher"
	"kubevirt.io/kubevirt/tools/vms-generator/utils"
)

const (
	fedoraVMSize               = "256M"
	secretDiskSerial           = "D23YZ9W6WA5DJ487"
	stressDefaultVMSize        = "100M"
	stressLargeVMSize          = "400M"
	stressDefaultSleepDuration = 15
)

var _ = SIGMigrationDescribe("VM Live Migration", decorators.RequiresTwoSchedulableNodes, func() {
	var (
		virtClient              kubecli.KubevirtClient
		migrationBandwidthLimit resource.Quantity
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
		cm := libconfigmap.New(name, data)
		cm, err := virtClient.CoreV1().ConfigMaps(namespace).Create(context.Background(), cm, metav1.CreateOptions{})
		Expect(err).ToNot(HaveOccurred())
		return name
	}

	prepareVMIWithAllVolumeSources := func(namespace string, kernelBootEnabled bool) *v1.VirtualMachineInstance {
		name := "secret-" + rand.String(5)
		secret := libsecret.New(name, libsecret.DataString{"user": "admin", "password": "redhat"})
		_, err := kubevirt.Client().CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
		if !errors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}

		configMapName := createConfigMap(namespace)
		opts := []libvmi.Option{
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithLabel(downwardTestLabelKey, downwardTestLabelVal),
			libvmi.WithDownwardAPIDisk("downwardapi-" + rand.String(5)),
			libvmi.WithServiceAccountDisk("default"),
			libvmi.WithSecretDisk(secret.Name, secret.Name),
			libvmi.WithConfigMapDisk(configMapName, configMapName),
			libvmi.WithEmptyDisk("usb-disk", v1.DiskBusUSB, resource.MustParse("64Mi")),
			libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData("#!/bin/bash\necho 'hello'\n")),
		}
		if kernelBootEnabled {
			opts = append(opts, withKernelBoot())
		}

		return libvmifact.NewFedora(opts...)
	}

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		migrationBandwidthLimit = resource.MustParse("1Ki")
	})

	Context("with Headless service", func() {
		const subdomain = "mysub"

		AfterEach(func() {
			err := virtClient.CoreV1().Services(testsuite.NamespaceTestDefault).Delete(context.Background(), subdomain, metav1.DeleteOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).NotTo(HaveOccurred())
			}
		})

		It("should remain to able resolve the VM IP", decorators.Conformance, func() {
			const hostname = "alpine"
			const port int = 1500
			const labelKey = "subdomain"
			const labelValue = "mysub"

			vmi := libvmifact.NewCirros(
				libvmi.WithHostname(hostname),
				withSubdomain(subdomain),
				libvmi.WithLabel(labelKey, labelValue),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Starting hello world in the VM")
			vmnetserver.StartTCPServer(vmi, port, console.LoginToCirros)

			By("Exposing headless service matching subdomain")
			service := service.BuildHeadlessSpec(subdomain, port, port, labelKey, labelValue)
			_, err := virtClient.CoreV1().Services(vmi.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			assertConnectivityToService := func(msg string) {
				By(msg)
				tcpJob := job.NewHelloWorldJobTCP(fmt.Sprintf("%s.%s", hostname, subdomain), strconv.FormatInt(int64(port), 10))
				tcpJob.Spec.BackoffLimit = pointer.P(int32(3))
				tcpJob, err := virtClient.BatchV1().Jobs(vmi.Namespace).Create(context.Background(), tcpJob, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				err = job.WaitForJobToSucceed(tcpJob, 90*time.Second)
				Expect(err).ToNot(HaveOccurred(), msg)
			}

			assertConnectivityToService("Asserting connectivity through service before migration")

			By("Executing a migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			assertConnectivityToService("Asserting connectivity through service after migration")
		})
	})

	Describe("Starting a VirtualMachineInstance ", func() {
		Context("with bandwidth limitations", func() {

			updateMigrationPolicyBandwidth := func(migrationPolicy *v1alpha1.MigrationPolicy, bandwidth resource.Quantity) {
				var err error
				migrationPolicy, err = virtClient.MigrationPolicy().Get(context.Background(), migrationPolicy.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				if policyBandwidth := migrationPolicy.Spec.BandwidthPerMigration; policyBandwidth != nil && policyBandwidth.Equal(bandwidth) {
					return
				}

				patchPayload, err := patch.New(
					patch.WithAdd("/spec/bandwidthPerMigration", pointer.P(bandwidth)),
				).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())

				migrationPolicy, err = virtClient.MigrationPolicy().Patch(context.Background(), migrationPolicy.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			repeatedlyMigrateWithBandwidthLimitation := func(vmi *v1.VirtualMachineInstance, migrationPolicy *v1alpha1.MigrationPolicy, bandwidth string, repeat int) time.Duration {
				var migrationDurationTotal time.Duration
				updateMigrationPolicyBandwidth(migrationPolicy, resource.MustParse(bandwidth))

				for x := 0; x < repeat; x++ {
					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())

					By("starting the migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

					vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					migrationDuration := vmi.Status.MigrationState.EndTimestamp.Sub(vmi.Status.MigrationState.StartTimestamp.Time)
					log.DefaultLogger().Infof("Migration with bandwidth %v took: %v", bandwidth, migrationDuration)
					migrationDurationTotal += migrationDuration
				}
				return migrationDurationTotal
			}

			It("[test_id:6968]should apply them and result in different migration durations", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
				migrationPolicy := CreateMigrationPolicy(virtClient, GeneratePolicyAndAlignVMI(vmi))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				durationLowBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, migrationPolicy, "1Mi", 3)
				durationHighBandwidth := repeatedlyMigrateWithBandwidthLimitation(vmi, migrationPolicy, "128Mi", 3)
				Expect(durationHighBandwidth.Seconds() * 2).To(BeNumerically("<", durationLowBandwidth.Seconds()))
			})
		})
		Context("with a Alpine disk", func() {
			It("[test_id:6969]should be successfully migrate with a tablet device", decorators.Conformance, func() {
				vmi := libvmifact.NewAlpineWithTestTooling(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithTablet("tablet0", v1.InputBusUSB),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
			It("should be successfully migrate with a WriteBack disk cache", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
				vmi.Spec.Domain.Devices.Disks[0].Cache = v1.CacheWriteBack

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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

			It("[test_id:6970]should migrate vmi with cdroms on various bus types", decorators.Conformance, func() {
				vmi := libvmifact.NewAlpineWithTestTooling(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithEphemeralCDRom("cdrom-0", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
					libvmi.WithEphemeralCDRom("cdrom-1", v1.DiskBusSCSI, cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("should migrate vmi with LiveMigrateIfPossible eviction strategy", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrateIfPossible),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("should migrate vmi and use Live Migration method with read-only disks", func() {
				By("Defining a VMI with PVC disk and read-only CDRoms")
				if !libstorage.HasCDI() {
					Fail("Fail DataVolume tests when CDI is not present")
				}
				sc, exists := libstorage.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}
				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize(cd.CirrosVolumeSize),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteMany),
						libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
					),
				)

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(dv)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dv, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))
				vmi := libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume("disk0", dv.Name),
					libvmi.WithResourceMemory("1Gi"),
					libvmi.WithEphemeralCDRom("cdrom-0", v1.DiskBusSATA, cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
					libvmi.WithEphemeralCDRom("cdrom-1", v1.DiskBusSCSI, cd.ContainerDiskFor(cd.ContainerDiskAlpine)),
				)
				vmi.Spec.Hostname = string(cd.ContainerDiskAlpine)
				By("Starting the VirtualMachineInstance")
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(240))

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Ensuring migration is using Live Migration method")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ShouldNot(HaveOccurred())
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.LiveMigration), "migration method is expected to be Live Migration")
			})

			DescribeTable("should migrate with a downwardMetrics", func(via libvmi.Option, metricsGetter libinfra.MetricsGetter) {
				vmi := libvmifact.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					via,
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(libinfra.GetHostnameFromMetrics(metrics)).To(Equal(vmi.Status.NodeName))

			},
				Entry("[test_id:6971]disk", libvmi.WithDownwardMetricsVolume("vhostmd"), libinfra.GetDownwardMetricsDisk),
				Entry("channel", libvmi.WithDownwardMetricsChannel(), libinfra.GetDownwardMetricsVirtio),
			)

			It("[test_id:6842]should migrate with TSC frequency set", decorators.Invtsc, decorators.TscFrequencies, func() {
				vmi := libvmifact.NewAlpine(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithCPUFeature("invtsc", "require"),
					libvmi.WithEvictionStrategy(v1.EvictionStrategyLiveMigrate),
				)

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)
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
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())
				vmi.Annotations = map[string]string{
					v1.PlacePCIDevicesOnRootComplex: "true",
				}

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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

			It("[test_id:9795]should migrate vmi with a usb disk", func() {

				vmi := libvmifact.NewAlpineWithTestTooling(
					libvmi.WithEmptyDisk("uniqueusbdisk", v1.DiskBusUSB, resource.MustParse("128Mi")),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:1783]should be successfully migrated multiple times with cloud-init disk", decorators.Conformance, func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				num := 4

				for i := 0; i < num; i++ {
					// execute a migration, wait for finalized state
					By(fmt.Sprintf("Starting the Migration for iteration %d", i))
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
					libmigration.ConfirmMigrationDataIsStored(virtClient, migration, vmi)

					By("Check if Migrated VMI has updated IP and IPs fields")
					Eventually(func() error {
						newvmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred(), "Should successfully get new VMI")
						vmiPod, err := libpod.GetPodByVirtualMachineInstance(newvmi, vmi.Namespace)
						Expect(err).NotTo(HaveOccurred())
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
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: v1.CreatedByLabel + "=" + string(vmi.GetUID()),
				})
				Expect(err).ToNot(HaveOccurred(), "Should list pods successfully")
				Expect(pods.Items).To(HaveLen(1), "There should be only one VMI pod")

				// find virtqemud pid
				pid, err := getVirtqemudPid(&pods.Items[0])
				Expect(err).ToNot(HaveOccurred())

				// kill virtqemud
				By(fmt.Sprintf("Killing virtqemud with pid %s", pid))
				stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(&pods.Items[0], "compute",
					[]string{
						"kill",
						"-9",
						pid,
					})
				errorMessageFormat := "failed after running `kill -9 %v` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(errorMessageFormat, pid, stdout, stderr, err))

				// ensure new pid comes online
				Eventually(getVirtqemudPid).WithArguments(&pods.Items[0]).
					WithPolling(1*time.Second).
					WithTimeout(30*time.Second).
					Should(Not(Equal(pid)), "expected virtqemud to be cycled")

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:6972]should migrate to a persistent (non-transient) libvirt domain.", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				By(fmt.Sprintf("Starting the Migration"))
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				// ensure the libvirt domain is persistent
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(isLibvirtDomainPersistent(vmi)).To(BeTrue(), "The VMI was not found in the list of libvirt persistent domains")
				libmigration.EnsureNoMigrationMetadataInPersistentXML(vmi)
			})
			It("[test_id:6973]should be able to successfully migrate with a paused vmi", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Pausing the VirtualMachineInstance")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Pause(context.Background(), vmi.Name, &v1.PauseOptions{})).To(Succeed())
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstancePaused))

				By("verifying that the vmi is still paused before migration")
				Expect(isLibvirtDomainPaused(vmi)).To(BeTrue(), "The VMI should be paused before migration, but it is not.")

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("verifying that the vmi is still paused after migration")
				Expect(isLibvirtDomainPaused(vmi)).To(BeTrue(), "The VMI should be paused before migration, but it is not.")

				By("verify that VMI can be unpaused after migration")
				Expect(virtClient.VirtualMachineInstance(vmi.Namespace).Unpause(context.Background(), vmi.Name, &v1.UnpauseOptions{})).To(Succeed(), "should successfully unpause the vmi")
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, 2*time.Second).Should(matcher.HaveConditionMissingOrFalse(v1.VirtualMachineInstancePaused))

				By("verifying that the vmi is running")
				Expect(isLibvirtDomainPaused(vmi)).To(BeFalse(), "The VMI should be running, but it is not.")
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

			It("should automatically cancel unschedulable migration after a timeout period", decorators.Conformance, func() {
				// Add node affinity to ensure VMI affinity rules block target pod from being created
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithNodeAffinityFor(nodes.Items[0].Name),
					libvmi.WithResourceMemory(fedoraVMSize),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationUnschedulablePodTimeoutSecondsAnnotation: "130"}

				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred(), "migration creation should succeed")

				By("Should receive warning event that target pod is currently unschedulable")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				watcher.New(migration).
					Timeout(60*time.Second).
					SinceWatchedObjectResourceVersion().
					WaitFor(ctx, watcher.WarningEvent, "migrationTargetPodUnschedulable")

				By("Migration should observe a timeout period before canceling unschedulable target pod")
				Consistently(matcher.ThisMigration(migration)).WithPolling(10 * time.Second).WithTimeout(1 * time.Minute).Should(Not(matcher.BeInPhase(v1.MigrationFailed)))

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(matcher.ThisMigration(migration)).WithPolling(5 * time.Second).WithTimeout(2 * time.Minute).Should(matcher.BeInPhase(v1.MigrationFailed))
			})

			It("should automatically cancel pending target pod after a catch all timeout period", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration that is expected to fail
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration.Annotations = map[string]string{v1.MigrationPendingPodTimeoutSecondsAnnotation: "130"}

				// Add a fake container image to the target pod to force a image pull failure which
				// keeps the target pod in pending state
				// Make sure to actually use an image repository we own here so no one
				// can somehow figure out a way to execute custom logic in our func tests.
				migration.Annotations[v1.FuncTestMigrationTargetImageOverrideAnnotation] = "quay.io/kubevirtci/some-fake-image:" + rand.String(12)

				By("Starting a Migration")
				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Migration should observe a timeout period before canceling pending target pod")
				Consistently(matcher.ThisMigration(migration)).WithPolling(10 * time.Second).WithTimeout(1 * time.Minute).Should(Not(matcher.BeInPhase(v1.MigrationFailed)))

				By("Migration should fail eventually due to pending target pod timeout")
				Eventually(matcher.ThisMigration(migration)).WithPolling(5 * time.Second).WithTimeout(2 * time.Minute).Should(matcher.BeInPhase(v1.MigrationFailed))
			})
		})
		Context(" with auto converge enabled", Serial, func() {
			BeforeEach(func() {

				// set autoconverge flag
				config := getCurrentKvConfig(virtClient)
				config.MigrationConfiguration.AllowAutoConverge = pointer.P(true)
				kvconfig.UpdateKubeVirtConfigValueAndWait(config)
			})

			It("[test_id:3237]should complete a migration", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finnish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				runStressTest(vmi, stressDefaultVMSize)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
		})
		Context("with setting guest time", func() {
			It("[test_id:4114]should set an updated time after a migration", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize), libvmi.WithRng())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

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
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the migrated VirtualMachineInstance has an updated time")
				if !console.OnPrivilegedPrompt(vmi, 60) {
					Expect(console.LoginToFedora(vmi)).To(Succeed())
				}

				By("Waiting for the agent to set the right time")
				Eventually(func() error {
					// get current time on the node
					output := libpod.RunCommandOnVmiPod(vmi, []string{"date", "+%H:%M"})
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
					Fail("Fail DataVolume tests when CDI is not present")
				}
			})

			It("[test_id:3239]should reject a migration of a vmi with a non-shared data volume", func() {
				sc, foundSC := libstorage.GetRWOFileSystemStorageClass()
				if !foundSC {
					Skip("Skip test when Filesystem storage is not present")
				}

				dataVolume := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine), cdiv1.RegistryPullNode),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(sc),
						libdv.StorageWithAccessMode(k8sv1.ReadWriteOnce),
					),
				)

				vmi := libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume("disk0", dataVolume.Name),
					libvmi.WithResourceMemory("1Gi"),
				)

				dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				libstorage.EventuallyDV(dataVolume, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// Verify console on last iteration to verify the VirtualMachineInstance is still booting properly
				// after being restarted multiple times
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(vmi).Should(matcher.HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))

				// execute a migration, wait for finalized state
				migration := libmigration.New(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))
			})
			It("[test_id:1479][storage-req] should migrate a vmi with a shared block disk", decorators.StorageReq, func() {
				sc, exists := libstorage.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}

				By("Starting the VirtualMachineInstance")
				vmi := newVMIWithDataVolumeForMigration(cd.ContainerDiskAlpine, k8sv1.ReadWriteMany, sc)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 300)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:6974]should reject additional migrations on the same VMI if the first one is not finished", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// Need to wait for cloud init to finish and start the agent inside the vmi.
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				By("Stressing the VMI")
				runStressTest(vmi, stressDefaultVMSize)

				By("Starting a first migration")
				migration1 := libmigration.New(vmi.Name, vmi.Namespace)
				migration1, err := virtClient.VirtualMachineInstanceMigration(migration1.Namespace).Create(context.Background(), migration1, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				// Successfully tested with 40, but requests start getting throttled above 10, which is better to avoid to prevent flakyness
				By("Starting 10 more migrations expecting all to fail to create")
				var wg sync.WaitGroup
				for n := 0; n < 10; n++ {
					wg.Add(1)
					go func(n int) {
						defer GinkgoRecover()
						defer wg.Done()
						migration := libmigration.New(vmi.Name, vmi.Namespace)
						_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
						Expect(err).To(HaveOccurred(), fmt.Sprintf("Extra migration %d should have failed to create", n))
						Expect(err.Error()).To(ContainSubstring(`admission webhook "migration-create-validator.kubevirt.io" denied the request: in-flight migration detected.`))
					}(n)
				}
				wg.Wait()

				libmigration.ExpectMigrationToSucceedWithDefaultTimeout(virtClient, migration1)
			})
		})
		Context("[storage-req]with an Alpine shared block volume PVC", decorators.StorageReq, func() {
			var sc string
			var exists bool

			BeforeEach(func() {
				sc, exists = libstorage.GetRWXBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}
			})

			It("[test_id:1854]should migrate a VMI with shared and non-shared disks", func() {
				// Start the VirtualMachineInstance with PVC and Ephemeral Disks
				image := cd.ContainerDiskFor(cd.ContainerDiskAlpine)
				vmi := newVMIWithDataVolumeForMigration(cd.ContainerDiskAlpine, k8sv1.ReadWriteMany, sc, libvmi.WithContainerDisk("myephemeral", image))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				By("Starting a Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})
			It("[release-blocker][test_id:1377]should be successfully migrated multiple times", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := libvmops.RunVMIAndExpectLaunch(newVMIWithDataVolumeForMigration(cd.ContainerDiskAlpine, k8sv1.ReadWriteMany, sc), 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				// execute a migration, wait for finalized state
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("[test_id:3240]should be successfully with a cloud init", func() {
				// Start the VirtualMachineInstance with the PVC attached
				vmi := newVMIWithDataVolumeForMigration(cd.ContainerDiskCirros, k8sv1.ReadWriteMany, sc,
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
					libvmi.WithHostname(fmt.Sprintf("%s", cd.ContainerDiskCirros)),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToCirros(vmi)).To(Succeed())

				By("Checking that MigrationMethod is set to BlockMigration")
				Expect(vmi.Status.MigrationMethod).To(Equal(v1.BlockMigration))

				// execute a migration, wait for finalized state
				By("Starting the Migration for iteration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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
					libdv.WithStorage(
						libdv.StorageWithStorageClass(storageClass),
						libdv.StorageWithVolumeSize(cd.FedoraVolumeSize),
						libdv.StorageWithReadWriteManyAccessMode(),
					),
				)

				var err error
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

			It("[test_id:2653] should be migrated successfully, using guest agent on VM with default migration configuration", func() {
				By("Creating the DV")
				createDV(testsuite.NamespacePrivileged)
				VMIMigrationWithGuestAgent(virtClient, dv.Name, fedoraVMSize, nil)
			})

			It("[test_id:6975] should have guest agent functional after migration", func() {
				By("Creating the DV")
				createDV(testsuite.GetTestNamespace(nil))
				By("Creating the VMI")
				vmi = libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithPersistentVolumeClaim("disk0", dv.Name),
					libvmi.WithResourceMemory(fedoraVMSize),
					libvmi.WithRng(),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)

				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 180)

				By("Checking guest agent")
				Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

				By("Starting the Migration for iteration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
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
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeSize(size),
					libdv.StorageWithReadWriteManyAccessMode(),
				),
				libdv.WithForceBindAnnotation(),
			)

			dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(namespace).Create(context.Background(), dv, metav1.CreateOptions{})
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
			libstorage.EventuallyDV(dv, 180, matcher.HaveSucceeded())
			libstorage.ChangeImgFilePermissionsToNonQEMU(pvc)
			return dv
		}

		Context(" migration to nonroot", Serial, func() {
			var dv *cdiv1.DataVolume
			size := "256Mi"
			var clusterIsRoot bool

			BeforeEach(func() {
				clusterIsRoot = checks.HasFeature(featuregate.Root)
				if !clusterIsRoot {
					kvconfig.EnableFeatureGate(featuregate.Root)
				}
			})
			AfterEach(func() {
				if !clusterIsRoot {
					kvconfig.DisableFeatureGate(featuregate.Root)
				} else {
					kvconfig.EnableFeatureGate(featuregate.Root)
				}
			})

			DescribeTable("should migrate root implementation to nonroot", func(createVMI func() *v1.VirtualMachineInstance, loginFunc console.LoginToFunction) {
				By("Create a VMI that will run root(default)")
				vmi := createVMI()

				By("Starting the VirtualMachineInstance")
				// Resizing takes too long and therefor a warning is thrown
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				By("Checking that the launcher is running as root")
				Expect(getIdOfLauncher(vmi)).To(Equal("0"))

				kvconfig.DisableFeatureGate(featuregate.Root)

				By("Starting new migration and waiting for it to succeed")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Checking that the launcher is running as qemu")
				Expect(getIdOfLauncher(vmi)).To(Equal("107"))
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				vmi, err := matcher.ThisVMI(vmi)()
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).To(HaveKey(v1.DeprecatedNonRootVMIAnnotation))
			},
				Entry("[test_id:8609] with simple VMI", func() *v1.VirtualMachineInstance {
					return libvmifact.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()))
				}, console.LoginToAlpine),

				Entry("[test_id:8610] with DataVolume", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the DataVolume
					return libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithDataVolume("disk0", dv.Name),
						libvmi.WithResourceMemory("1Gi"),
					)
				}, console.LoginToAlpine),

				Entry("[test_id:8611] with CD + CloudInit + SA + ConfigMap + Secret + DownwardAPI", func() *v1.VirtualMachineInstance {
					return prepareVMIWithAllVolumeSources(testsuite.NamespacePrivileged, false)
				}, console.LoginToFedora),

				Entry("[test_id:8612] with PVC", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the Underlying PVC
					return libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithPersistentVolumeClaim("disk0", dv.Name),
						libvmi.WithResourceMemory("128Mi"),
					)
				}, console.LoginToAlpine),
			)
		})
		Context(" migration to root", Serial, func() {
			var dv *cdiv1.DataVolume
			var clusterIsRoot bool
			size := "256Mi"

			BeforeEach(func() {
				clusterIsRoot = checks.HasFeature(featuregate.Root)
				if clusterIsRoot {
					kvconfig.DisableFeatureGate(featuregate.Root)
				}
			})
			AfterEach(func() {
				if clusterIsRoot {
					kvconfig.EnableFeatureGate(featuregate.Root)
				} else {
					kvconfig.DisableFeatureGate(featuregate.Root)
				}
			})

			DescribeTable("should migrate nonroot implementation to root", func(createVMI func() *v1.VirtualMachineInstance, loginFunc console.LoginToFunction) {
				By("Create a VMI that will run root(default)")
				vmi := createVMI()
				// force VMI on privileged namespace since we will be migrating to a root VMI
				vmi.Namespace = testsuite.NamespacePrivileged

				By("Starting the VirtualMachineInstance")
				// Resizing takes too long and therefor a warning is thrown
				vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				By("Checking that the launcher is running as root")
				Expect(getIdOfLauncher(vmi)).To(Equal("107"))

				kvconfig.EnableFeatureGate(featuregate.Root)

				By("Starting new migration and waiting for it to succeed")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Checking that the launcher is running as qemu")
				Expect(getIdOfLauncher(vmi)).To(Equal("0"))
				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(loginFunc(vmi)).To(Succeed())

				vmi, err := matcher.ThisVMI(vmi)()
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Annotations).ToNot(HaveKey(v1.DeprecatedNonRootVMIAnnotation))
			},
				Entry("with simple VMI", func() *v1.VirtualMachineInstance {
					return libvmifact.NewAlpine(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()))
				}, console.LoginToAlpine),

				Entry("with DataVolume", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the DataVolume
					return libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithDataVolume("disk0", dv.Name),
						libvmi.WithResourceMemory("1Gi"),
					)
				}, console.LoginToAlpine),

				Entry("with CD + CloudInit + SA + ConfigMap + Secret + DownwardAPI + Kernel Boot", func() *v1.VirtualMachineInstance {
					return prepareVMIWithAllVolumeSources(testsuite.NamespacePrivileged, false)
				}, console.LoginToFedora),

				Entry("with PVC", func() *v1.VirtualMachineInstance {
					dv = createDataVolumePVCAndChangeDiskImgPermissions(testsuite.NamespacePrivileged, size)
					// Use the underlying PVC
					return libvmi.New(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
						libvmi.WithPersistentVolumeClaim("disk0", dv.Name),
						libvmi.WithResourceMemory("128Mi"),
					)

				}, console.LoginToAlpine),
			)
		})
		Context("migration security", func() {
			Context(" with TLS disabled", Serial, func() {
				It("[test_id:6976] should be successfully migrated", func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration.DisableTLS = pointer.P(true)
					kvconfig.UpdateKubeVirtConfigValueAndWait(cfg)

					vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToAlpine(vmi)).To(Succeed())

					By("starting the migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

					// check VMI, confirm migration state
					libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
				})

				It("[test_id:6977]should not secure migrations with TLS", func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration.BandwidthPerMigration = resource.NewQuantity(1, resource.BinarySI)
					cfg.MigrationConfiguration.DisableTLS = pointer.P(true)
					kvconfig.UpdateKubeVirtConfigValueAndWait(cfg)
					vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					runStressTest(vmi, stressDefaultVMSize)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})

					By("Waiting for the proxy connection details to appear")
					Eventually(matcher.ThisVMI(vmi)).WithPolling(1 * time.Second).WithTimeout(60 * time.Second).Should(haveMigrationState(
						gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"TargetNodeAddress":              Not(Equal("")),
							"TargetDirectMigrationNodePorts": Not(BeEmpty()),
						})),
					))
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

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
							Expect(libpod.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
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
					vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

					By("Limiting the bandwidth of migrations in the test namespace")
					CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					// Run
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					By("Waiting for the proxy connection details to appear")
					Eventually(func() bool {
						migratingVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if migratingVMI.Status.MigrationState == nil {
							return false
						}

						if migratingVMI.Status.MigrationState.TargetNodeAddress == "" || len(migratingVMI.Status.MigrationState.TargetDirectMigrationNodePorts) == 0 {
							return false
						}
						vmi = migratingVMI
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue(), "Timed out waiting for migration state to include TargetNodeAddress and TargetDirectMigrationNodePorts")

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
							Expect(libpod.ForwardPorts(handler, []string{fmt.Sprintf("4321%d:%d", i, port)}, stopChan, 10*time.Second)).To(Succeed())
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
						if strings.Contains(err.Error(), "remote error: tls:") {
							tlsErrorFound = true
						}
						Expect(err.Error()).To(Or(ContainSubstring("remote error: tls: unknown certificate authority"), Or(ContainSubstring("remote error: tls: bad certificate")), ContainSubstring("EOF")))
					}

					Expect(tlsErrorFound).To(BeTrue())
				})
			})
		})

		Context(" migration monitor", Serial, func() {
			var createdPods []string
			AfterEach(func() {
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), podName, metav1.DeleteOptions{})
						return err
					}, 10*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Should delete helper pod")
				}
			})
			BeforeEach(func() {
				createdPods = []string{}
				cfg := getCurrentKvConfig(virtClient)
				cfg.MigrationConfiguration = &v1.MigrationConfiguration{
					CompletionTimeoutPerGiB: pointer.P(int64(5)),
				}
				kvconfig.UpdateKubeVirtConfigValueAndWait(cfg)
			})
			Context("without progress", func() {

				BeforeEach(func() {
					cfg := getCurrentKvConfig(virtClient)
					cfg.MigrationConfiguration = &v1.MigrationConfiguration{
						ProgressTimeout:         pointer.P(int64(5)),
						CompletionTimeoutPerGiB: pointer.P(int64(5)),
						BandwidthPerMigration:   resource.NewQuantity(1, resource.BinarySI),
					}
					kvconfig.UpdateKubeVirtConfigValueAndWait(cfg)
				})

				It("[test_id:2227] should abort a vmi migration", func() {
					vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory("1Gi"))

					By("Starting the VirtualMachineInstance")
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

					By("Checking that the VirtualMachineInstance console has expected output")
					Expect(console.LoginToFedora(vmi)).To(Succeed())

					// Need to wait for cloud init to finish and start the agent inside the vmi.
					Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

					runStressTest(vmi, stressLargeVMSize)

					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

					// check VMI, confirm migration state
					vmi = libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)
					Expect(vmi.Status.MigrationState.FailureReason).To(ContainSubstring("has been aborted"))
				})

			})
			It("[test_id:6978] Should detect a failed migration", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory("1Gi"))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

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
					pod := libpod.RenderPrivilegedPod(podName, []string{"/bin/bash", "-c"}, []string{fmt.Sprintf("while true; do ps aux | grep -v \"defunct\" | grep -v \"D\" | grep \"%s\" && pkill -9 virt-handler && sleep 5; done", emulator)})

					pod.Spec.NodeName = entry.Name
					createdPod, err := virtClient.CoreV1().Pods(pod.Namespace).Create(context.Background(), pod, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred(), "Should create helper pod")
					createdPods = append(createdPods, createdPod.Name)
				}
				Expect(createdPods).ToNot(BeEmpty(), "There is no node for migration")

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				vmi = libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)
				// Not sure how consistent the entire error is, so just making sure the failure happened in libvirt. Example string:
				// Live migration failed error encountered during MigrateToURI3 libvirt api call: virError(Code=1, Domain=7, Message='internal error: client socket is closed'
				Expect(vmi.Status.MigrationState.FailureReason).To(ContainSubstring("libvirt api call"))

				By("Removing our migration killer pods")
				for _, podName := range createdPods {
					Eventually(func() error {
						err := virtClient.CoreV1().Pods(testsuite.NamespacePrivileged).Delete(context.Background(), podName, metav1.DeleteOptions{})
						return err
					}, 10*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "Should delete helper pod")

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
				migration = libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 340)

				By("Verifying Second Migration Succeeeds")
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			})

			It("old finalized migrations should get garbage collected", func() {
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithResourceMemory("1Gi"),
					// this annotation causes virt launcher to immediately fail a migration
					libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				for i := 0; i < 10; i++ {
					// execute a migration, wait for finalized state
					By("Starting the Migration")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration.Name = fmt.Sprintf("%s-iter-%d", vmi.Name, i)
					migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

					// check VMI, confirm migration state
					vmi = libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)
					Expect(vmi.Status.MigrationState.FailureReason).To(ContainSubstring("Failed migration to satisfy functional test condition"))
					Eventually(matcher.ThisPodWith(vmi.Namespace, vmi.Status.MigrationState.TargetPod)).WithPolling(time.Second).WithTimeout(10*time.Second).
						Should(
							Or(
								matcher.BeInPhase(k8sv1.PodFailed),
								matcher.BeInPhase(k8sv1.PodSucceeded),
							), "Target pod should exit quickly after migration fails.")
				}

				migrations, err := virtClient.VirtualMachineInstanceMigration(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(migrations.Items).To(HaveLen(5))
			})

			It("[test_id:6979]Target pod should exit after failed migration", func() {
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithResourceMemory("1Gi"),
					// this annotation causes virt launcher to immediately fail a migration
					libvmi.WithAnnotation(v1.FuncTestForceLauncherMigrationFailureAnnotation, ""),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migrationUID := libmigration.RunMigrationAndExpectFailure(migration, 180)

				// check VMI, confirm migration state
				vmi = libmigration.ConfirmVMIPostMigrationFailed(vmi, migrationUID)
				Expect(vmi.Status.MigrationState.FailureReason).To(ContainSubstring("Failed migration to satisfy functional test condition"))
				Eventually(matcher.ThisPodWith(vmi.Namespace, vmi.Status.MigrationState.TargetPod)).WithPolling(time.Second).WithTimeout(10*time.Second).
					Should(Or(matcher.BeInPhase(k8sv1.PodFailed), matcher.BeInPhase(k8sv1.PodSucceeded)), "Target pod should exit quickly after migration fails.")
			})

			It("[test_id:6980][QUARANTINE]Migration should fail if target pod fails during target preparation", decorators.Quarantine, func() {
				vmi := libvmifact.NewFedora(
					libnet.WithMasqueradeNetworking(),
					libvmi.WithResourceMemory("1Gi"),
					// this annotation prevents virt launcher from finishing the target pod preparation.
					libvmi.WithAnnotation(v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation, ""),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(func() v1.VirtualMachineInstanceMigrationPhase {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					phase := migration.Status.Phase
					Expect(phase).NotTo(Equal(v1.MigrationSucceeded))
					return phase
				}, 120, 1*time.Second).Should(Equal(v1.MigrationPreparingTarget))

				By("Killing the target pod and expecting failure")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Status.MigrationState).ToNot(BeNil())
				Expect(vmi.Status.MigrationState.TargetPod).ToNot(Equal(""))

				err = virtClient.CoreV1().Pods(vmi.Namespace).Delete(context.Background(), vmi.Status.MigrationState.TargetPod, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Expecting VMI migration failure")
				Eventually(matcher.ThisVMI(vmi)).WithPolling(time.Second).WithTimeout(120*time.Second).Should(haveMigrationState(
					gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
						"Failed":         BeTrue(),
						"StartTimestamp": Not(BeNil()),
						"EndTimestamp":   Not(BeNil()),
						"Completed":      BeTrue(),
					})),
				), "vmi's migration state should be finalized as failed after target pod exits")
			})
			It("Migration should generate empty isos of the right size on the target", func() {
				By("Creating a VMI with cloud-init and config maps")
				configMapName := "configmap-" + rand.String(5)
				secretName := "secret-" + rand.String(5)
				downwardAPIName := "downwardapi-" + rand.String(5)
				config_data := map[string]string{
					"config1": "value1",
					"config2": "value2",
				}
				cm := libconfigmap.New(configMapName, config_data)
				cm, err := virtClient.CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				secret := libsecret.New(secretName, libsecret.DataString{"user": "admin", "password": "community"})
				secret, err = kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
				if !errors.IsAlreadyExists(err) {
					Expect(err).ToNot(HaveOccurred())
				}

				vmi := libvmifact.NewAlpine(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithConfigMapDisk(configMapName, configMapName),
					libvmi.WithSecretDisk(secretName, secretName),
					libvmi.WithServiceAccountDisk("default"),
					libvmi.WithDownwardAPIDisk(downwardAPIName),
					// this annotation causes virt launcher to immediately fail a migration
					libvmi.WithAnnotation(v1.FuncTestBlockLauncherPrepareMigrationTargetAnnotation, ""),
					libvmi.WithLabel(downwardTestLabelKey, downwardTestLabelVal),
				)

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				// execute a migration
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting for Migration to reach Preparing Target Phase")
				Eventually(matcher.ThisMigration(migration)).WithPolling(1 * time.Second).WithTimeout(120 * time.Second).Should(matcher.BeInPhase(v1.MigrationPreparingTarget))

				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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
									volPath := virthandler.IsoGuestVolumePath(vmi.Namespace, vmi.Name, &volume)
									if volPath == "" {
										continue
									}
									// Wait for the iso to be created
									Eventually(func() error {
										output, err := runCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "[[ -f " + volPath + " ]] && echo found || true"})
										if err != nil {
											return err
										}
										if !strings.Contains(output, "found") {
											return fmt.Errorf("%s never appeared", volPath)
										}
										return nil
									}, 30*time.Second, time.Second).Should(Not(HaveOccurred()))
									output, err := runCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", "/usr/bin/stat --printf=%s " + volPath})
									Expect(err).ToNot(HaveOccurred())
									Expect(strconv.Atoi(output)).To(Equal(int(volStatus.Size)), "ISO file for volume %s is not the right size", volume.Name)
									output, err = runCommandOnVmiTargetPod(vmi, []string{"/bin/bash", "-c", fmt.Sprintf(`/usr/bin/cmp -n %d %s /dev/zero || true`, volStatus.Size, volPath)})
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
				sc, exists := libstorage.GetRWOBlockStorageClass()
				if !exists {
					Skip("Skip test when Block storage is not present")
				}

				// Start the VirtualMachineInstance with the PVC attached
				vmi := newVMIWithDataVolumeForMigration(cd.ContainerDiskAlpine, k8sv1.ReadWriteOnce, sc, libvmi.WithHostname(string(cd.ContainerDiskAlpine)))
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				Expect(vmi).Should(matcher.HaveConditionFalse(v1.VirtualMachineInstanceIsMigratable))

				// execute a migration, wait for finalized state
				migration := libmigration.New(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				_, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("DisksNotLiveMigratable"))
			})
		})

		Context("live migration cancellation", func() {
			type vmiBuilder func() *v1.VirtualMachineInstance

			newVirtualMachineInstanceWithFedoraContainerDisk := func() *v1.VirtualMachineInstance {
				return libvmifact.NewFedora(libnet.WithMasqueradeNetworking())
			}

			newVirtualMachineInstanceWithFedoraRWXBlockDisk := func() *v1.VirtualMachineInstance {
				if !libstorage.HasCDI() {
					Fail("Fail DataVolume tests when CDI is not present")
				}

				sc, foundSC := libstorage.GetBlockStorageClass(k8sv1.ReadWriteMany)
				if !foundSC {
					Skip("Skip test when Block storage is not present")
				}

				dv := libdv.NewDataVolume(
					libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskFedoraTestTooling), cdiv1.RegistryPullNode),
					libdv.WithStorage(
						libdv.StorageWithStorageClass(sc),
						libdv.StorageWithVolumeSize(cd.FedoraVolumeSize),
						libdv.StorageWithReadWriteManyAccessMode(),
						libdv.StorageWithBlockVolumeMode(),
					),
				)

				dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				libstorage.EventuallyDV(dv, 600, Or(matcher.HaveSucceeded(), matcher.PendingPopulation()))
				vmi := libvmi.New(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithDataVolume("disk0", dv.Name),
					libvmi.WithResourceMemory("1Gi"),
					libvmi.WithCloudInitNoCloud(libvmifact.WithDummyCloudForFastBoot()),
				)
				return vmi
			}

			DescribeTable("should be able to cancel a migration", decorators.SigStorage, func(createVMI vmiBuilder) {
				vmi := createVMI()
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse(fedoraVMSize)

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				const timeout = 180
				Eventually(func() error {
					var err error
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
					return err
				}, timeout, 1*time.Second).ShouldNot(HaveOccurred())

				By("Waiting until the Migration is Running")
				Eventually(func() bool {
					migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					Expect(migration.Status.Phase).ToNot(Equal(v1.MigrationFailed))
					if migration.Status.Phase == v1.MigrationRunning {
						vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						if vmi.Status.MigrationState.Completed != true {
							return true
						}
					}
					return false

				}, timeout, 1*time.Second).Should(BeTrue())

				By("Cancelling a Migration")
				Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})).To(Succeed())

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigrationAborted(vmi, string(migration.UID), 180)

				By("Waiting for the migration object to disappear")
				libwait.WaitForMigrationToDisappearWithTimeout(migration, 240)
			},
				Entry("[sig-storage][test_id:2226] with ContainerDisk", newVirtualMachineInstanceWithFedoraContainerDisk),
				Entry("[sig-storage][storage-req][test_id:2731] with RWX block disk from block volume PVC", decorators.StorageReq, newVirtualMachineInstanceWithFedoraRWXBlockDisk),
			)

			It("[sig-compute][test_id:3241]Immediate migration cancellation after migration starts running cancel a migration by deleting vmim object", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)
				sourcePod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the Migration is Running")
				Eventually(matcher.ThisMigration(migration)).WithPolling(1 * time.Second).WithTimeout(60 * time.Second).Should(matcher.BeRunning())

				By("Cancelling a Migration")
				Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})).To(Succeed())

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
			})

			It("[sig-compute][test_id:8584]Immediate migration cancellation before migration starts running cancel a migration by deleting vmim object", func() {
				vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

				By("Limiting the bandwidth of migrations in the test namespace")
				CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)
				vmiOriginalNode := vmi.Status.NodeName

				// execute a migration, wait for finalized state
				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)

				By("Starting a Migration")
				migration, err := virtClient.VirtualMachineInstanceMigration(migration.Namespace).Create(context.Background(), migration, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Waiting until the Migration has UID")
				Eventually(func() (types.UID, error) {
					migration, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
					return migration.UID, err
				}, 180, 1*time.Second).ShouldNot(Equal(types.UID("")))

				By("Cancelling a Migration")
				Expect(virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})).To(Succeed())

				By("Waiting for the migration object to disappear")
				libwait.WaitForMigrationToDisappearWithTimeout(migration, 240)

				By("Verifying the VMI's phase, migration state and on node")
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi).To(
					And(
						matcher.BeRunning(),
						haveMigrationState(BeNil()),
						gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
							"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
								"NodeName": Equal(vmiOriginalNode),
							}),
						})),
					),
				)
			})

			Context("when target pod cannot be scheduled and is suck in Pending phase", Serial, func() {

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
					vmi := libvmifact.NewGuestless(
						libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
						libvmi.WithNetwork(v1.DefaultPodNetwork()),
					)
					vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

					By("Trying to migrate VM and expect for the migration to get stuck")
					migration := libmigration.New(vmi.Name, vmi.Namespace)
					migration = libmigration.RunMigration(virtClient, migration)
					Eventually(matcher.ThisMigration(migration), 30*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.MigrationScheduling))
					Consistently(matcher.ThisMigration(migration), 60*time.Second, 5*time.Second).Should(matcher.BeInPhase(v1.MigrationScheduling))

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
							Expect(pod).To(matcher.BeInPhase(k8sv1.PodPending),
								"VMI is expected to have exactly 2 pods: one running and one pending")
						}
					}

					By("Aborting the migration")
					err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Delete(context.Background(), migration.Name, metav1.DeleteOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					By("Expecting migration to be deleted")
					Eventually(func() error {
						_, err = virtClient.VirtualMachineInstanceMigration(migration.Namespace).Get(context.Background(), migration.Name, metav1.GetOptions{})
						return err
					}, 60*time.Second, 5*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

					By("Making sure source pod is still running")
					sourcePod, err = virtClient.CoreV1().Pods(sourcePod.Namespace).Get(context.Background(), sourcePod.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())
					Expect(sourcePod).To(matcher.BeInPhase(k8sv1.PodRunning))
					By("Making sure the VMI's migration state remains nil")
					Consistently(matcher.ThisVMI(vmi)).WithPolling(5 * time.Second).WithTimeout(30 * time.Second).Should(haveMigrationState(BeNil()))
				})

			})

		})

		Context(" with migration policies", Serial, func() {
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
				config.MigrationConfiguration.AllowAutoConverge = pointer.P(true)
				kvconfig.UpdateKubeVirtConfigValueAndWait(config)

				vmi := libvmifact.NewAlpine(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)

				var expectedPolicyName *string
				if defineMigrationPolicy {
					By("Creating a migration policy that overrides cluster policy")
					policy := GeneratePolicyAndAlignVMI(vmi)
					policy.Spec.AllowAutoConverge = pointer.P(false)

					_, err := virtClient.MigrationPolicy().Create(context.Background(), policy, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					expectedPolicyName = &policy.Name
				}

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Starting the Migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)

				// check VMI, confirm migration state
				libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				By("Retrieving the VMI post migration")
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Status.MigrationState.MigrationConfiguration).ToNot(BeNil())
				confirmMigrationPolicyName(vmi, expectedPolicyName)
			},
				Entry("should override cluster-wide policy if defined", true),
				Entry("should not affect cluster-wide policy if not defined", false),
			)

		})

		Context(" with freePageReporting", Serial, func() {

			BeforeEach(func() {
				kv := libkubevirt.GetCurrentKv(virtClient)
				kvConfigurationCopy := kv.Spec.Configuration.DeepCopy()
				kvConfigurationCopy.VirtualMachineOptions = nil
				kvconfig.UpdateKubeVirtConfigValueAndWait(*kvConfigurationCopy)
			})

			It("should be able to migrate", func() {
				vmi := libvmifact.NewAlpineWithTestTooling(libnet.WithMasqueradeNetworking())

				By("Starting the VirtualMachineInstance")
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				By("Checking that the VirtualMachineInstance console has expected output")
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				domSpec, err := tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domSpec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo("on"))

				By("starting the migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

				vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

				domSpec, err = tests.GetRunningVMIDomainSpec(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(domSpec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo("on"))
			})

		})

		Context("with unsupported machine type", Serial, func() {

			It("should prevent migration scheduling", func() {
				vmi := libvmifact.NewGuestless(libnet.WithMasqueradeNetworking())
				vmi.Namespace = testsuite.GetTestNamespace(vmi)

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 60)
				machineType := vmi.Status.Machine.Type
				Expect(machineType).ToNot(BeEmpty(), "VMI should have a valid machine type in its status")

				By("Fetching all nodes in the cluster")
				nodeList, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(nodeList.Items).ToNot(BeEmpty())

				By("Patching all nodes to remove the supported machine type label")
				for _, node := range nodeList.Items {
					nodeName := node.Name
					libinfra.ExpectStoppingNodeLabellerToSucceed(nodeName, virtClient)

					// we remove the vmi specific machine type from all the labels on all nodes to render it unsupported for scheduling
					libnode.RemoveLabelFromNode(node.Name, v1.SupportedMachineTypeLabel+machineType)
				}

				DeferCleanup(func() {
					By("Restoring the machine type label for all nodes")
					for _, node := range nodeList.Items {
						libinfra.ExpectResumingNodeLabellerToSucceed(node.Name, virtClient)
					}
				})

				By("Initiating a migration")
				migration := libmigration.New(vmi.Name, vmi.Namespace)
				createdMigration := libmigration.RunMigration(virtClient, migration)

				Eventually(ThisMigration(createdMigration), 30*time.Second, 1*time.Second).Should(BeInPhase(v1.MigrationScheduling))

				targetPod, err := libpod.GetTargetPodForMigration(createdMigration)
				Expect(err).ToNot(HaveOccurred())
				Expect(targetPod.Spec.NodeSelector).To(HaveKey(v1.SupportedMachineTypeLabel + machineType))

				var scheduledCond *k8sv1.PodCondition
				Eventually(func() *k8sv1.PodCondition {
					scheduledCond = controller.NewPodConditionManager().GetCondition(targetPod, k8sv1.PodScheduled)
					return scheduledCond
				}, 30*time.Second, 1*time.Second).ShouldNot(BeNil(), "PodScheduled condition should not be nil")

				Expect(scheduledCond.Status).To(BeEquivalentTo(k8sv1.ConditionFalse), "PodScheduled status should be False")
				Expect(scheduledCond.Reason).To(BeEquivalentTo(k8sv1.PodReasonUnschedulable), "PodScheduled reason should be Unschedulable")
				Expect(scheduledCond.Message).To(ContainSubstring("node(s) didn't match Pod's node affinity/selector"), "PodScheduled message mismatch")
			})
		})
	})

	Context("with sata disks", func() {

		It("[test_id:1853]VM with containerDisk + CloudInit + ServiceAccount + ConfigMap + Secret + DownwardAPI + External Kernel Boot + USB Disk", decorators.Conformance, func() {
			vmi := prepareVMIWithAllVolumeSources(testsuite.GetTestNamespace(nil), true)

			Expect(vmi.Spec.Domain.Devices.Disks).To(HaveLen(7))
			Expect(vmi.Spec.Domain.Devices.Interfaces).To(HaveLen(1))

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
		})
	})

	Context("[test_id:8482] Migration Metrics", func() {
		It("exposed to prometheus during VM migration", func() {
			vmi := libvmifact.NewFedora(libnet.WithMasqueradeNetworking(), libvmi.WithResourceMemory(fedoraVMSize))

			By("Limiting the bandwidth of migrations in the test namespace")
			CreateMigrationPolicy(virtClient, PreparePolicyAndVMIWithBandwidthLimitation(vmi, migrationBandwidthLimit))

			By("Starting the VirtualMachineInstance")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Checking that the VirtualMachineInstance console has expected output")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			// Need to wait for cloud init to finnish and start the agent inside the vmi.
			Eventually(matcher.ThisVMI(vmi), 12*time.Minute, 2*time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceAgentConnected))

			// execute a migration, wait for finalized state
			By("Starting the Migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			libmigration.RunMigrationAndCollectMigrationMetrics(vmi, migration)
		})
	})

	Context(" With Huge Pages", Serial, func() {
		DescribeTable("should consume hugepages ", func(hugepageSize string, memory string) {
			hugepageType := k8sv1.ResourceName(k8sv1.ResourceHugePagesPrefix + hugepageSize)

			count := 0
			nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			for _, node := range nodes.Items {
				// Cmp returns -1, 0, or 1 for less than, equal to, or greater than
				if v, ok := node.Status.Capacity[hugepageType]; ok && v.Cmp(resource.MustParse(memory)) == 1 {
					count += 1
				}
			}

			if count < 2 {
				Skip(fmt.Sprintf("Not enough nodes with hugepages %s capacity. Need 2, found %d.", hugepageType, count))
			}

			hugepagesVmi := libvmifact.NewAlpine(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithResourceMemory(memory),
				libvmi.WithHugepages(hugepageSize),
			)

			By("Starting hugepages VMI")
			libvmops.RunVMIAndExpectLaunch(hugepagesVmi, 360)

			By("starting the migration")
			migration := libmigration.New(hugepagesVmi.Name, hugepagesVmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			// check VMI, confirm migration state
			libmigration.ConfirmVMIPostMigration(virtClient, hugepagesVmi, migration)
		},
			Entry("[test_id:6983]hugepages-2Mi", "2Mi", "64Mi"),
			Entry("[test_id:6984]hugepages-1Gi", "1Gi", "1Gi"),
		)
	})

	Context(" with CPU pinning and huge pages", Serial, decorators.RequiresTwoWorkerNodesWithCPUManager, decorators.RequiresHugepages2Mi, func() {
		It("should not make migrations fail", func() {
			var err error
			cpuVMI := libvmifact.NewAlpine(
				libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithResourceMemory("128Mi"),
				libvmi.WithDedicatedCPUPlacement(),
				libvmi.WithCPUCount(3, 1, 1),
				libvmi.WithHugepages("2Mi"),
			)

			By("Starting a VirtualMachineInstance")
			cpuVMI, err = virtClient.VirtualMachineInstance(cpuVMI.Namespace).Create(context.Background(), cpuVMI, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(cpuVMI)

			By("Performing a migration")
			migration := libmigration.New(cpuVMI.Name, cpuVMI.Namespace)
			libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
		})
		Context("and NUMA passthrough", decorators.RequiresTwoWorkerNodesWithCPUManager, decorators.RequiresHugepages2Mi, func() {
			It("should not make migrations fail", func() {
				cpuVMI := libvmifact.NewAlpine(
					libvmi.WithNamespace(testsuite.GetTestNamespace(nil)),
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithResourceMemory("128Mi"),
					libvmi.WithDedicatedCPUPlacement(),
					libvmi.WithCPUCount(3, 1, 1),
					libvmi.WithNUMAGuestMappingPassthrough(),
					libvmi.WithHugepages("2Mi"),
				)

				By("Starting a VirtualMachineInstance")
				libvmops.RunVMIAndExpectLaunch(cpuVMI, 360)

				By("Performing a migration")
				migration := libmigration.New(cpuVMI.Name, cpuVMI.Namespace)
				libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			})
		})
	})

	Context("with dedicated CPUs", decorators.RequiresTwoWorkerNodesWithCPUManager, func() {
		var (
			nodes         []k8sv1.Node
			pausePod      *k8sv1.Pod
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
				// Range will be found when there are disabled/not plugged cpus
				if strings.Contains(lineSplits[1], "-") {
					continue
				}
				cpu, err := strconv.Atoi(lineSplits[1])
				Expect(err).ToNot(HaveOccurred(), "cpu id is non string in vcpupin output", vcpuPinOutput)

				cpuSet = append(cpuSet, cpu)
			}

			return cpuSet
		}

		getLibvirtDomainCPUSet := func(vmi *v1.VirtualMachineInstance) []int {
			pod, err := libpod.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())

			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
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

			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
				pod,
				"compute",
				[]string{"cat", cpusetPath})
			Expect(err).ToNot(HaveOccurred())
			Expect(stderr).To(BeEmpty())

			return parseSysCpuSet(strings.TrimSpace(stdout))
		}

		getVirtLauncherCPUSet := func(vmi *v1.VirtualMachineInstance) []int {
			pod, err := libpod.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
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
			nodes = libnode.GetWorkerNodesWithCPUManagerEnabled(virtClient)

			By("creating a template for a pause pod with 1 dedicated CPU core")
			pausePod = libpod.RenderPod("pause-", nil, nil)
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

			By("starting a VMI on the first node of the list")
			libnode.AddLabelToNode(nodes[0].Name, testLabel1, "true")

			By("creating a migratable VMI with 2 dedicated CPU cores")
			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithCPUCount(2, 1, 1),
				libvmi.WithDedicatedCPUPlacement(),
				libvmi.WithResourceMemory("512Mi"),
				libvmi.WithNodeAffinityForLabel(testLabel1, "true"),
			)
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the VirtualMachineInstance starts")
			vmi = libwait.WaitForSuccessfulVMIStart(vmi, libwait.WithTimeout(120))

			By("determining cgroups version")
			cgroupVersion = getVMIsCgroupVersion(vmi)

			By("ensuring the VMI started on the correct node")
			Expect(vmi.Status.NodeName).To(Equal(nodes[0].Name))

			By("reserving the cores used by the VMI on the second node with a paused pod")
			libnode.AddLabelToNode(nodes[1].Name, testLabel2, "true")

			var pods []*k8sv1.Pod
			ns := testsuite.GetTestNamespace(pausePod)
			pausedPod, err := libpod.Run(pausePod, ns)
			Expect(err).ToNot(HaveOccurred())
			for !hasCommonCores(vmi, pausedPod) {
				By("creating another paused pod since last didn't have common cores with the VMI")
				pods = append(pods, pausedPod)
				pausedPod, err = libpod.Run(pausePod, ns)
				Expect(err).ToNot(HaveOccurred())
			}

			By("deleting the paused pods that don't have cores in common with the VMI")
			for _, pod := range pods {
				err = virtClient.CoreV1().Pods(pod.Namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
				Expect(err).ToNot(HaveOccurred())
			}

			By("migrating the VMI from first node to second node")
			libnode.AddLabelToNode(nodes[1].Name, testLabel1, "true")
			cpuSetSource := getVirtLauncherCPUSet(vmi)
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)
			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("ensuring the target cpuset is different from the source")
			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
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

	Context("with a dedicated migration network", Serial, func() {
		var nadName string

		BeforeEach(func() {
			virtClient = kubevirt.Client()

			if flags.MigrationNetworkName != "" {
				By(fmt.Sprintf("Using the provided Network Attachment Definition: %s", flags.MigrationNetworkName))
				nadName = flags.MigrationNetworkName
			} else {
				By("Creating the Network Attachment Definition")
				nad := libmigration.GenerateMigrationCNINetworkAttachmentDefinition()
				nadName = nad.Name
				_, err := virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(flags.KubeVirtInstallNamespace).Create(context.Background(), nad, metav1.CreateOptions{})
				Expect(err).NotTo(HaveOccurred(), "Failed to create the Network Attachment Definition")
				DeferCleanup(func() {
					By("Deleting the Network Attachment Definition")
					Expect(virtClient.NetworkClient().K8sCniCncfIoV1().NetworkAttachmentDefinitions(flags.KubeVirtInstallNamespace).Delete(context.Background(), nadName, metav1.DeleteOptions{})).To(Succeed(), "Failed to delete the Network Attachment Definition")
				})
			}

			By("Setting it as the migration network in the KubeVirt CR")
			libmigration.SetDedicatedMigrationNetwork(nadName)
		})

		AfterEach(func() {
			By("Clearing the migration network in the KubeVirt CR")
			libmigration.ClearDedicatedMigrationNetwork()
		})

		It("Should migrate over that network", func() {
			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)

			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Starting the migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Checking if the migration happened, and over the right network")
			vmi = libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)
			targetHandler, err := libnode.GetVirtHandlerPod(kubevirt.Client(), vmi.Status.NodeName)
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Status.MigrationState.TargetNodeAddress).ToNot(Equal(targetHandler.Status.PodIP), "The migration did not appear to go over the dedicated migration network")
		})
	})
	It("should update MigrationState's MigrationConfiguration of VMI status", func() {
		By("Starting a VMI")
		vmi := libvmifact.NewAlpine(
			libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
			libvmi.WithNetwork(v1.DefaultPodNetwork()),
		)

		vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

		By("Starting a Migration")
		migration := libmigration.New(vmi.Name, vmi.Namespace)
		migration = libmigration.RunMigrationAndExpectToComplete(virtClient, migration, 180)
		libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

		By("Ensuring MigrationConfiguration is updated")
		vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(vmi).To(haveMigrationState(
			gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
				"MigrationConfiguration": Not(BeNil()),
			})),
		))
	})

	Context("with a live-migration in flight", func() {
		It("there should always be a single active migration per VMI", decorators.Conformance, func() {
			By("Starting a VMI")
			vmi := libvmifact.NewGuestless(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			)
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Checking that there always is at most one migration running")
			labelSelector, err := labels.Parse(fmt.Sprintf("%s in (%s)", v1.MigrationSelectorLabel, vmi.Name))
			Expect(err).ToNot(HaveOccurred())
			listOptions := metav1.ListOptions{
				LabelSelector: labelSelector.String(),
			}
			Consistently(func() int {
				vmim := libmigration.New(vmi.Name, vmi.Namespace)
				// not checking err as the migration creation will be blocked immediately by virt-api's validating webhook
				// if another one is currently running
				vmim, _ = virtClient.VirtualMachineInstanceMigration(vmi.Namespace).Create(context.Background(), vmim, metav1.CreateOptions{})

				migrations, err := virtClient.VirtualMachineInstanceMigration(vmim.Namespace).List(context.Background(), listOptions)
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
					var err error
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return topology.AreTSCFrequencyTopologyHintsDefined(vmi)
				}, 90*time.Second, 3*time.Second).Should(BeTrue(), fmt.Sprintf("tsc frequency topology hints are expected to exist for vmi %s", vmi.Name))
			}

			It("invtsc feature exists", decorators.Invtsc, func() {
				vmi := libvmi.New(
					libvmi.WithResourceMemory("1Mi"),
					libvmi.WithCPUFeature("invtsc", "require"),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				expectTopologyHintsToBeSet(vmi)
			})

			It("HyperV reenlightenment is enabled", func() {
				vmi := libvmifact.NewWindows()
				vmi.Spec.Domain.Devices.Disks = []v1.Disk{}
				vmi.Spec.Volumes = []v1.Volume{}
				vmi.Spec.Domain.Features.Hyperv.Reenlightenment = &v1.FeatureState{Enabled: pointer.P(true)}
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 240)

				expectTopologyHintsToBeSet(vmi)
			})

		})

	})

	Context("ResourceQuota rejection", func() {
		It("Should contain condition when migrating with quota that doesn't have resources for both source and target", decorators.Conformance, func() {
			vmiRequest := resource.MustParse("200Mi")
			vmi := libvmifact.NewCirros(
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
			_ = libvmops.RunVMIAndExpectLaunch(vmi, 240)

			By("Trying to migrate the VirtualMachineInstance")
			migration := libmigration.New(vmi.Name, testsuite.GetTestNamespace(vmi))
			migration = libmigration.RunMigration(virtClient, migration)
			Eventually(matcher.ThisMigration(migration)).WithPolling(1 * time.Second).WithTimeout(60 * time.Second).Should(matcher.HaveConditionTrue(v1.VirtualMachineInstanceMigrationRejectedByResourceQuota))
		})
	})

	Context("Virtiofs", decorators.VirtioFS, func() {
		It("should migrate with a shared ConfigMap", func() {
			configMapName := "configmap-" + uuid.NewString()[:6]
			data := map[string]string{
				"option1": "value1",
				"option2": "value2",
				"option3": "value3",
			}
			testCommand := "cat /mnt/option1 /mnt/option2 /mnt/option3\n"
			expectedOutput := "value1value2value3"

			By("Creating ConfigMap")
			cm := libconfigmap.New(configMapName, data)
			cm, err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			vmi := libvmifact.NewAlpine(
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithConfigMapFs(configMapName, configMapName),
			)

			By("Running VMI")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

			By("Logging into the VMI")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Checking mounted ConfigVolume")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt \n", configMapName)},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: testCommand},
				&expect.BExp{R: expectedOutput},
			}, 200)).To(Succeed())

			By("Starting the migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Checking the ConfigVolume is still mounted and data integrity")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: testCommand},
				&expect.BExp{R: expectedOutput},
			}, 200)).To(Succeed())
		})

		It("should migrate with a shared DV", func() {
			sc, foundSC := libstorage.GetRWXFileSystemStorageClass()

			if !foundSC {
				Skip("Skip test when Filesystem RWX storage is not present")
			}

			dataVolume := libdv.NewDataVolume(
				libdv.WithRegistryURLSource(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine)),
				libdv.WithStorage(
					libdv.StorageWithStorageClass(sc),
					libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeFilesystem),
					libdv.StorageWithAccessMode(k8sv1.ReadWriteMany),
					libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(cd.ContainerDiskAlpine))),
				),
				libdv.WithNamespace(testsuite.GetTestNamespace(nil)),
			)

			dataVolume, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dataVolume, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libstorage.EventuallyDV(dataVolume, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

			mountVirtiofsCommand := fmt.Sprintf(`#!/bin/bash
                                      mount -t virtiofs %s /mnt
									  touch /mnt/test_file
								`, dataVolume.Name)

			vmi := libvmifact.NewFedora(
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountVirtiofsCommand)),
				libvmi.WithFilesystemPVC(dataVolume.Name),
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
				libvmi.WithCloudInitNoCloud(libvmici.WithNoCloudEncodedUserData(mountVirtiofsCommand)),
			)

			By("Starting VMI")
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 180)

			By("Logging into the VMI")
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			By("Starting the migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			libmigration.ConfirmVMIPostMigration(virtClient, vmi, migration)

			By("Checking the DV is still mounted and testing file is accessible")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "test -f /mnt/test_file \n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
			}, 200)).To(Succeed())
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

func isLibvirtDomainPersistent(vmi *v1.VirtualMachineInstance) (bool, error) {
	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
		vmiPod,
		libpod.LookupComputeContainer(vmiPod).Name,
		[]string{"virsh", "--quiet", "list", "--persistent", "--name"},
	)
	if err != nil {
		return false, fmt.Errorf("could not dump libvirt domxml (remotely on pod): %v: %s", err, stderr)
	}
	return strings.Contains(stdout, vmi.Namespace+"_"+vmi.Name), nil
}

func isLibvirtDomainPaused(vmi *v1.VirtualMachineInstance) (bool, error) {
	namespace := testsuite.GetTestNamespace(vmi)
	vmi, err := kubevirt.Client().VirtualMachineInstance(namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	vmiPod, err := libpod.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, namespace, vmi.Status.NodeName)
	if err != nil {
		return false, err
	}

	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
		vmiPod,
		libpod.LookupComputeContainer(vmiPod).Name,
		[]string{"virsh", "--quiet", "domstate", vmi.Namespace + "_" + vmi.Name},
	)
	if err != nil {
		return false, fmt.Errorf("could not get libvirt domstate (remotely on pod): %v: %s", err, stderr)
	}
	return strings.Contains(stdout, "paused"), nil
}

func getVMIsCgroupVersion(vmi *v1.VirtualMachineInstance) cgroup.CgroupVersion {
	pod, err := libpod.GetRunningPodByLabel(string(vmi.GetUID()), v1.CreatedByLabel, vmi.Namespace, vmi.Status.NodeName)
	Expect(err).ToNot(HaveOccurred())

	return getPodsCgroupVersion(pod)
}

func getPodsCgroupVersion(pod *k8sv1.Pod) cgroup.CgroupVersion {
	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(
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
	kvc := libkubevirt.GetCurrentKv(virtClient)

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

func runStressTest(vmi *v1.VirtualMachineInstance, vmsize string) {
	By("Run a stress test to dirty some pages and slow down the migration")
	stressCmd := fmt.Sprintf("stress-ng --vm 1 --vm-bytes %s --vm-keep &\n", vmsize)
	Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: stressCmd},
		&expect.BExp{R: console.PromptExpression},
	}, 15)).To(Succeed(), "should run a stress test")

	// give stress tool some time to trash more memory pages before returning control to next steps
	time.Sleep(stressDefaultSleepDuration * time.Second)
}

func getIdOfLauncher(vmi *v1.VirtualMachineInstance) string {
	vmi, err := kubevirt.Client().VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())

	vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).ToNot(HaveOccurred())

	podOutput, err := exec.ExecuteCommandOnPod(
		vmiPod,
		vmiPod.Spec.Containers[0].Name,
		[]string{"id", "-u"},
	)
	Expect(err).NotTo(HaveOccurred())

	return strings.TrimSpace(podOutput)
}

func getVirtqemudPid(pod *k8sv1.Pod) (string, error) {
	stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(pod, "compute",
		[]string{
			"pidof",
			"virtqemud",
		})
	errorMessageFormat := "failed after running `pidof virtqemud` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
	if err != nil {
		return "", fmt.Errorf(errorMessageFormat, stdout, stderr, err)
	}

	return strings.TrimSuffix(stdout, "\n"), nil

}

// runCommandOnVmiTargetPod runs specified command on the target virt-launcher pod of a migration
func runCommandOnVmiTargetPod(vmi *v1.VirtualMachineInstance, command []string) (string, error) {
	virtClient := kubevirt.Client()

	pods, err := virtClient.CoreV1().Pods(vmi.Namespace).List(context.Background(), metav1.ListOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, pods.Items).NotTo(BeEmpty())
	var vmiPod *k8sv1.Pod
	for _, pod := range pods.Items {
		if pod.Name == vmi.Status.MigrationState.TargetPod {
			vmiPod = &pod
			break
		}
	}
	if vmiPod == nil {
		return "", fmt.Errorf("failed to find migration target pod")
	}

	output, err := exec.ExecuteCommandOnPod(
		vmiPod,
		"compute",
		command,
	)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return output, nil
}

func newVMIWithDataVolumeForMigration(containerDisk cd.ContainerDisk, accessMode k8sv1.PersistentVolumeAccessMode, storageClass string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	virtClient := kubevirt.Client()

	dv := libdv.NewDataVolume(
		libdv.WithRegistryURLSourceAndPullMethod(cd.DataVolumeImportUrlForContainerDisk(containerDisk), cdiv1.RegistryPullNode),
		libdv.WithStorage(
			libdv.StorageWithStorageClass(storageClass),
			libdv.StorageWithVolumeSize(cd.ContainerDiskSizeBySourceURL(cd.DataVolumeImportUrlForContainerDisk(containerDisk))),
			libdv.StorageWithAccessMode(accessMode),
			libdv.StorageWithVolumeMode(k8sv1.PersistentVolumeBlock),
		),
	)

	dv, err := virtClient.CdiClient().CdiV1beta1().DataVolumes(testsuite.GetTestNamespace(nil)).Create(context.Background(), dv, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())
	libstorage.EventuallyDV(dv, 240, Or(matcher.HaveSucceeded(), matcher.WaitForFirstConsumer()))

	return libstorage.RenderVMIWithDataVolume(dv.Name, dv.Namespace, opts...)
}

func haveMigrationState(matcher gomegatypes.GomegaMatcher) gomegatypes.GomegaMatcher {
	return gstruct.PointTo(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
		"Status": gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
			"MigrationState": matcher,
		}),
	}))
}

func withKernelBoot() libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		kernelBootFirmware := utils.GetVMIKernelBootWithRandName().Spec.Domain.Firmware
		if vmi.Spec.Domain.Firmware == nil {
			vmi.Spec.Domain.Firmware = kernelBootFirmware
		} else {
			vmi.Spec.Domain.Firmware.KernelBoot = kernelBootFirmware.KernelBoot
		}
	}
}

func withSubdomain(subdomain string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Subdomain = subdomain
	}
}
