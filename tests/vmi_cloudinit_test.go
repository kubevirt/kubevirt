/*
 * This file is part of the kubevirt project
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
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/libvmi"
	libcloudinit "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/util/net/dns"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

const (
	sshAuthorizedKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	fedoraPassword       = "fedora"
	expectedUserDataFile = "cloud-init-userdata-executed"
	testNetworkData      = "#Test networkData"
	testUserData         = "#cloud-config"

	dataSourceNoCloudVolumeID     = "cidata"
	dataSourceConfigDriveVolumeID = "config-2"
	startupTime                   = 30
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]CloudInit UserData", decorators.SigCompute, func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		// from default virt-launcher flag: do we need to make this configurable in some cases?
		cloudinit.SetLocalDirectoryOnly("/var/run/kubevirt-ephemeral-disks/cloud-init-data")
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with cloudInitNoCloud", func() {
			It("[test_id:1618]should take user-data from k8s secret", decorators.Conformance, func() {
				userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
				secretID := fmt.Sprintf("%s-test-secret", uuid.NewString())

				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudUserDataSecretName(secretID)),
				)

				// Store userdata as k8s secret
				By("Creating a user-data secret")
				secret := libsecret.New(secretID, libsecret.DataString{"userdata": userData})
				_, err := virtClient.CoreV1().Secrets(testsuite.GetTestNamespace(vmi)).Create(context.Background(), secret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				runningVMI := libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
				runningVMI = libwait.WaitUntilVMIReady(runningVMI, console.LoginToCirros)

				checkCloudInitIsoSize(runningVMI, cloudinit.DataSourceNoCloud)

				By("Checking whether the user-data script had created the file")
				Expect(console.RunCommand(runningVMI, fmt.Sprintf("cat /%s\n", expectedUserDataFile), time.Second*120)).To(Succeed())

				// Expect that the secret is not present on the vmi itself
				runningVMI, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(runningVMI)).Get(context.Background(), runningVMI.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				runningCloudInitVolume := lookupCloudInitNoCloudVolume(runningVMI.Spec.Volumes)
				origCloudInitVolume := lookupCloudInitNoCloudVolume(vmi.Spec.Volumes)

				Expect(origCloudInitVolume).To(Equal(runningCloudInitVolume), "volume must not be changed when running the vmi, to prevent secret leaking")
			})

			It("[test_id:1615]should have cloud-init data from userDataBase64 source", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudEncodedUserData(userData)),
				)

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				checkCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("Checking whether the user-data script had created the file")
				Expect(console.RunCommand(vmi, fmt.Sprintf("cat /%s\n", expectedUserDataFile), time.Second*120)).To(Succeed())
			})

			Context("with injected ssh-key", func() {
				It("[test_id:1616]should have ssh-key under authorized keys", func() {
					userData := fmt.Sprintf(
						"#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_authorized_keys:\n  - %s",
						fedoraPassword,
						sshAuthorizedKey,
					)
					vmi := libvmifact.NewFedora(libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudUserData(userData)))

					vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
					checkCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

					verifyUserDataVMI(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "fedora\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: fedoraPassword + "\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
						&expect.BExp{R: "test-ssh-key"},
					}, time.Second*300)
				})
			})
		})

		Context("with cloudInitConfigDrive", func() {
			It("[test_id:3178]should have cloud-init data from userDataBase64 source", decorators.Conformance, func() {
				userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
				vmi := libvmifact.NewCirros(libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveUserData(userData)))

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, libvmops.StartupTimeoutSecondsSmall)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("Checking whether the user-data script had created the file")
				Expect(console.RunCommand(vmi, fmt.Sprintf("cat /%s\n", expectedUserDataFile), time.Second*120)).To(Succeed())
			})

			Context("with injected ssh-key", func() {
				It("[test_id:3178]should have ssh-key under authorized keys", func() {
					userData := fmt.Sprintf(
						"#cloud-config\npassword: %s\nchpasswd: { expire: False }\nssh_authorized_keys:\n  - %s",
						fedoraPassword,
						sshAuthorizedKey,
					)
					vmi := libvmifact.NewFedora(
						libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveUserData(userData)),
					)

					vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
					checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

					verifyUserDataVMI(vmi, []expect.Batcher{
						&expect.BSnd{S: "\n"},
						&expect.BExp{R: "login:"},
						&expect.BSnd{S: "fedora\n"},
						&expect.BExp{R: "Password:"},
						&expect.BSnd{S: fedoraPassword + "\n"},
						&expect.BExp{R: console.PromptExpression},
						&expect.BSnd{S: "cat /home/fedora/.ssh/authorized_keys\n"},
						&expect.BExp{R: "test-ssh-key"},
					}, time.Second*300)
				})
			})

			It("cloud-init instance-id should be stable", func() {
				getInstanceId := func(vmi *v1.VirtualMachineInstance) (string, error) {
					cmd := "cat /var/lib/cloud/data/instance-id"
					return console.RunCommandAndStoreOutput(vmi, cmd, time.Second*30)
				}

				userData := fmt.Sprintf(
					"#cloud-config\npassword: %s\nchpasswd: { expire: False }",
					fedoraPassword,
				)
				vm := libvmi.NewVirtualMachine(
					libvmifact.NewFedora(libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveUserData(userData))),
					libvmi.WithRunStrategy(v1.RunStrategyAlways),
				)

				By("Start VM")
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vm)).Create(context.Background(), vm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Get VM cloud-init instance-id")
				Eventually(matcher.ThisVMIWith(vm.Namespace, vm.Name)).WithTimeout(10 * time.Second).WithPolling(time.Second).Should(matcher.Exist())
				vmi, err := virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
				instanceId, err := getInstanceId(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(instanceId).ToNot(BeEmpty())

				By("Restart VM")
				vm = libvmops.StopVirtualMachine(vm)
				vm = libvmops.StartVirtualMachine(vm)

				By("Get VM cloud-init instance-id after restart")
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)
				newInstanceId, err := getInstanceId(vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Make sure the instance-ids match")
				Expect(instanceId).To(Equal(newInstanceId))
			})
		})

		Context("with cloudInitNoCloud networkData", func() {
			It("[test_id:3181]should have cloud-init network-config with NetworkData source", func() {
				vmi := libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudNetworkData(testNetworkData)),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceNoCloudVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "network-config", testNetworkData)

			})
			It("[test_id:3182]should have cloud-init network-config with NetworkDataBase64 source", func() {
				vmi := libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudEncodedNetworkData(testNetworkData)),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceNoCloudVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "network-config", testNetworkData)

			})
			It("[test_id:3183]should have cloud-init network-config from k8s secret", func() {
				secretID := fmt.Sprintf("%s-test-secret", uuid.NewString())

				vmi := libvmifact.NewCirros(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithCloudInitNoCloud(libcloudinit.WithNoCloudNetworkDataSecretName(secretID)),
				)

				By("Creating a secret with network data")
				secret := libsecret.New(secretID, libsecret.DataString{"networkdata": testNetworkData})
				_, err := virtClient.CoreV1().Secrets(testsuite.GetTestNamespace(vmi)).Create(context.Background(), secret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceNoCloudVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "network-config", testNetworkData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				testVolume := lookupCloudInitNoCloudVolume(vmi.Spec.Volumes)
				Expect(testVolume).ToNot(BeNil(), "should find cloud-init volume in vmi spec")
				Expect(testVolume.CloudInitNoCloud.UserData).To(BeEmpty())
				Expect(testVolume.CloudInitNoCloud.NetworkData).To(BeEmpty())
				Expect(testVolume.CloudInitNoCloud.NetworkDataBase64).To(BeEmpty())
			})

		})

		Context("with cloudInitConfigDrive networkData", func() {
			It("[test_id:3184]should have cloud-init network-config with NetworkData source", func() {
				vmi := libvmifact.NewCirros(libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveNetworkData(testNetworkData)))
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceConfigDriveVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)
			})
			It("[test_id:4622]should have cloud-init meta_data with tagged devices", func() {
				const (
					pciAddress = "0000:01:00.0"
					macAddress = "9a:50:e8:6f:f3:fe"
					tag        = "specialNet"
				)
				testInstancetype := "testInstancetype"
				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveNetworkData(testNetworkData)),
					libvmi.WithInterface(v1.Interface{
						Name: "default",
						Tag:  tag,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{
							Masquerade: &v1.InterfaceMasquerade{},
						},
						PciAddress: pciAddress,
						MacAddress: macAddress,
					}),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
					libvmi.WithAnnotation(v1.InstancetypeAnnotation, testInstancetype),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID:   fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					InstanceType: testInstancetype,
					Hostname:     dns.SanitizeHostname(vmi),
					UUID:         string(vmi.Spec.Domain.Firmware.UUID),
					Devices: &[]cloudinit.DeviceData{
						{
							Type:    cloudinit.NICMetadataType,
							Bus:     "pci",
							Address: pciAddress,
							MAC:     macAddress,
							Tags:    []string{tag},
						},
					},
				}
				buf, err := json.Marshal(metadataStruct)
				Expect(err).ToNot(HaveOccurred())
				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceConfigDriveVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				By("checking cloudinit meta-data")
				const consoleCmd = `cat /mnt/openstack/latest/meta_data.json; printf "@@"`
				res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
					&expect.BSnd{S: consoleCmd + console.CRLF},
					&expect.BExp{R: `(.*)@@`},
				}, 15)
				Expect(err).ToNot(HaveOccurred())
				rawOutput := res[len(res)-1].Output
				Expect(rawOutput).To(ContainSubstring(string(buf)))
			})
			It("[test_id:3185]should have cloud-init network-config with NetworkDataBase64 source", func() {
				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitConfigDrive(libcloudinit.WithConfigDriveEncodedNetworkData(testNetworkData)),
				)
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)
				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceConfigDriveVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

			})
			It("[test_id:3186]should have cloud-init network-config from k8s secret", func() {
				secretID := fmt.Sprintf("%s-test-secret", uuid.NewString())
				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitConfigDrive(
						libcloudinit.WithConfigDriveUserDataSecretName(secretID),
						libcloudinit.WithConfigDriveNetworkDataSecretName(secretID),
					),
				)

				// Store cloudinit data as k8s secret
				By("Creating a secret with user and network data")
				secret := libsecret.New(secretID, libsecret.DataString{
					"userdata":    testUserData,
					"networkdata": testNetworkData,
				})

				_, err := virtClient.CoreV1().Secrets(testsuite.GetTestNamespace(vmi)).Create(context.Background(), secret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceConfigDriveVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)
				checkCloudInitFile(vmi, "openstack/latest/user_data", testUserData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, volume := range vmi.Spec.Volumes {
					if volume.CloudInitConfigDrive != nil {
						Expect(volume.CloudInitConfigDrive.UserData).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.UserDataBase64).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.UserDataSecretRef).ToNot(BeNil())

						Expect(volume.CloudInitConfigDrive.NetworkData).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.NetworkDataSecretRef).ToNot(BeNil())
						break
					}
				}
			})

			DescribeTable("[test_id:3187]should have cloud-init userdata and network-config from separate k8s secrets", func(userDataLabel string, networkDataLabel string) {
				uSecretID := fmt.Sprintf("%s-test-secret", uuid.NewString())
				nSecretID := fmt.Sprintf("%s-test-secret", uuid.NewString())

				vmi := libvmifact.NewCirros(
					libvmi.WithCloudInitConfigDrive(
						libcloudinit.WithConfigDriveUserDataSecretName(uSecretID),
						libcloudinit.WithConfigDriveNetworkDataSecretName(nSecretID),
					),
				)

				ns := testsuite.GetTestNamespace(vmi)

				By("Creating a secret with userdata")
				uSecret := libsecret.New(uSecretID, libsecret.DataString{userDataLabel: testUserData})

				_, err := virtClient.CoreV1().Secrets(ns).Create(context.Background(), uSecret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Creating a secret with network data")
				nSecret := libsecret.New(nSecretID, libsecret.DataString{networkDataLabel: testNetworkData})

				_, err = virtClient.CoreV1().Secrets(ns).Create(context.Background(), nSecret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, startupTime)
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				checkCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mounting cloudinit iso")
				Expect(mountGuestDevice(vmi, dataSourceConfigDriveVolumeID)).To(Succeed())

				By("checking cloudinit network-config")
				checkCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				By("checking cloudinit user-data")
				checkCloudInitFile(vmi, "openstack/latest/user_data", testUserData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, volume := range vmi.Spec.Volumes {
					if volume.CloudInitConfigDrive != nil {
						Expect(volume.CloudInitConfigDrive.UserData).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.UserDataBase64).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.UserDataSecretRef).ToNot(BeNil())

						Expect(volume.CloudInitConfigDrive.NetworkData).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
						Expect(volume.CloudInitConfigDrive.NetworkDataSecretRef).ToNot(BeNil())
						break
					}
				}
			},
				Entry("with lowercase labels", "userdata", "networkdata"),
				Entry("with camelCase labels", "userData", "networkData"),
			)
		})

	})
})

func lookupCloudInitNoCloudVolume(volumes []v1.Volume) *v1.Volume {
	for i, volume := range volumes {
		if volume.CloudInitNoCloud != nil {
			return &volumes[i]
		}
	}
	return nil
}

func mountGuestDevice(vmi *v1.VirtualMachineInstance, devName string) error {
	cmdCheck := fmt.Sprintf("mount $(blkid  -L %s) /mnt/\n", devName)
	return console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "sudo su -\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: ""},
		&expect.BSnd{S: console.EchoLastReturnValue},
		&expect.BExp{R: console.RetValue("0")},
	}, 15)
}

func verifyUserDataVMI(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
	By("Checking that the VirtualMachineInstance serial console output equals to expected one")
	Expect(console.ExpectBatch(vmi, commands, timeout)).To(Succeed())
}

func checkCloudInitFile(vmi *v1.VirtualMachineInstance, testFile, testData string) {
	cmdCheck := "cat " + filepath.Join("/mnt", testFile) + "\n"
	err := console.SafeExpectBatch(vmi, []expect.Batcher{
		&expect.BSnd{S: "sudo su -\n"},
		&expect.BExp{R: ""},
		&expect.BSnd{S: cmdCheck},
		&expect.BExp{R: testData},
	}, 15)
	Expect(err).ToNot(HaveOccurred())
}

func checkCloudInitIsoSize(vmi *v1.VirtualMachineInstance, source cloudinit.DataSourceType) {
	pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
	Expect(err).NotTo(HaveOccurred())

	path := cloudinit.GetIsoFilePath(source, vmi.Name, vmi.Namespace)

	By(fmt.Sprintf("Checking cloud init ISO at '%s' is 4k-block fs compatible", path))
	cmdCheck := []string{"stat", "--printf='%s'", path}

	out, err := exec.ExecuteCommandOnPod(pod, "compute", cmdCheck)
	Expect(err).NotTo(HaveOccurred())
	size, err := strconv.Atoi(strings.Trim(out, "'"))
	Expect(err).NotTo(HaveOccurred())
	Expect(size % 4096).To(Equal(0))
}
