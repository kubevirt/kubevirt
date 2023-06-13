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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package tests_test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

const (
	sshAuthorizedKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	fedoraPassword       = "fedora"
	expectedUserDataFile = "cloud-init-userdata-executed"
	testNetworkData      = "#Test networkData"
	testUserData         = "#cloud-config"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]CloudInit UserData", decorators.SigCompute, func() {

	var err error
	var virtClient kubecli.KubevirtClient

	var (
		LaunchVMI                 func(*v1.VirtualMachineInstance) *v1.VirtualMachineInstance
		VerifyUserDataVMI         func(*v1.VirtualMachineInstance, []expect.Batcher, time.Duration)
		MountCloudInitNoCloud     func(*v1.VirtualMachineInstance)
		MountCloudInitConfigDrive func(*v1.VirtualMachineInstance)
		CheckCloudInitFile        func(*v1.VirtualMachineInstance, string, string)
		CheckCloudInitIsoSize     func(vmi *v1.VirtualMachineInstance, source cloudinit.DataSourceType)
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()

		// from default virt-launcher flag: do we need to make this configurable in some cases?
		cloudinit.SetLocalDirectoryOnly("/var/run/kubevirt-ephemeral-disks/cloud-init-data")
		MountCloudInitNoCloud = tests.MountCloudInitFunc("cidata")
		MountCloudInitConfigDrive = tests.MountCloudInitFunc("config-2")
	})

	LaunchVMI = func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
		By("Starting a VirtualMachineInstance")
		obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
		Expect(err).ToNot(HaveOccurred())

		By("Waiting the VirtualMachineInstance start")
		vmi, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
		Expect(libwait.WaitForSuccessfulVMIStart(vmi).Status.NodeName).ToNot(BeEmpty())
		return vmi
	}

	VerifyUserDataVMI = func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
		By("Checking that the VirtualMachineInstance serial console output equals to expected one")
		Expect(console.SafeExpectBatch(vmi, commands, int(timeout.Seconds()))).To(Succeed())
	}

	CheckCloudInitFile = func(vmi *v1.VirtualMachineInstance, testFile, testData string) {
		cmdCheck := "cat " + filepath.Join("/mnt", testFile) + "\n"
		err := console.SafeExpectBatch(vmi, []expect.Batcher{
			&expect.BSnd{S: "sudo su -\n"},
			&expect.BExp{R: console.PromptExpression},
			&expect.BSnd{S: cmdCheck},
			&expect.BExp{R: testData},
		}, 15)
		Expect(err).ToNot(HaveOccurred())
	}

	CheckCloudInitIsoSize = func(vmi *v1.VirtualMachineInstance, source cloudinit.DataSourceType) {
		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)
		path := cloudinit.GetIsoFilePath(source, vmi.Name, vmi.Namespace)

		By(fmt.Sprintf("Checking cloud init ISO at '%s' is 4k-block fs compatible", path))
		cmdCheck := []string{"stat", "--printf='%s'", path}

		out, err := exec.ExecuteCommandOnPod(virtClient, pod, "compute", cmdCheck)
		Expect(err).NotTo(HaveOccurred())
		size, err := strconv.Atoi(strings.Trim(out, "'"))
		Expect(err).NotTo(HaveOccurred())
		Expect(size % 4096).To(Equal(0))
	}

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with cloudInitNoCloud userDataBase64 source", func() {
			It("[test_id:1615]should have cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

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
					vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling), userData)

					LaunchVMI(vmi)
					CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

					VerifyUserDataVMI(vmi, []expect.Batcher{
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

		Context("with cloudInitConfigDrive userDataBase64 source", func() {
			It("[test_id:3178]should have cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

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
					vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataHighMemory(cd.ContainerDiskFor(cd.ContainerDiskFedoraTestTooling), userData)

					LaunchVMI(vmi)
					CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

					VerifyUserDataVMI(vmi, []expect.Batcher{
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
					instanceId, err := console.RunCommandAndStoreOutput(vmi, cmd, time.Second*30)
					return instanceId, err
				}

				userData := fmt.Sprintf(
					"#cloud-config\npassword: %s\nchpasswd: { expire: False }",
					fedoraPassword,
				)
				vmi := libvmi.NewFedora(libvmi.WithCloudInitConfigDriveData(userData, false))
				// runStrategy := v1.RunStrategyManual
				vm := &v1.VirtualMachine{
					ObjectMeta: vmi.ObjectMeta,
					Spec: v1.VirtualMachineSpec{
						Running: tests.NewBool(false),
						Template: &v1.VirtualMachineInstanceTemplateSpec{
							Spec: vmi.Spec,
						},
					},
				}

				By("Start VM")
				vm, err := virtClient.VirtualMachine(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vm)
				Expect(vm.Namespace).ToNot(BeEmpty())
				Expect(err).ToNot(HaveOccurred())
				vm = tests.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				By("Get VM cloud-init instance-id")
				instanceId, err := getInstanceId(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(instanceId).ToNot(BeEmpty())

				By("Restart VM")
				vm = tests.StopVirtualMachine(vm)
				tests.StartVirtualMachine(vm)
				vmi, err = virtClient.VirtualMachineInstance(vm.Namespace).Get(context.Background(), vm.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				vmi = libwait.WaitUntilVMIReady(vmi, console.LoginToFedora)

				By("Get VM cloud-init instance-id after restart")
				newInstanceId, err := getInstanceId(vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Make sure the instance-ids match")
				Expect(instanceId).To(Equal(newInstanceId))
			})
		})

		Context("should process provided cloud-init data", func() {
			userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)

			runTest := func(vmi *v1.VirtualMachineInstance, dsType cloudinit.DataSourceType) {
				vmi = tests.RunVMIAndExpectLaunch(vmi, 60)

				By("waiting until login appears")
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				By("validating cloud-init disk is 4k aligned")
				CheckCloudInitIsoSize(vmi, dsType)

				By("Checking whether the user-data script had created the file")
				Expect(console.RunCommand(vmi, fmt.Sprintf("cat /%s\n", expectedUserDataFile), time.Second*120)).To(Succeed())

				By("validating the hostname matches meta-data")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "hostname\n"},
					&expect.BExp{R: dns.SanitizeHostname(vmi)},
				}, 10)).To(Succeed())
			}

			It("[test_id:1617] with cloudInitNoCloud userData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(
					cd.ContainerDiskFor(cd.ContainerDiskCirros),
					userData)
				runTest(vmi, cloudinit.DataSourceNoCloud)
			})
			It("[test_id:3180] with cloudInitConfigDrive userData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(
					cd.ContainerDiskFor(cd.ContainerDiskCirros),
					userData)
				runTest(vmi, cloudinit.DataSourceConfigDrive)
			})
		})

		It("[test_id:1618]should take user-data from k8s secret", func() {
			userData := fmt.Sprintf("#!/bin/sh\n\ntouch /%s\n", expectedUserDataFile)
			vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), "")

			idx := 0
			for i, volume := range vmi.Spec.Volumes {
				if volume.CloudInitNoCloud == nil {
					continue
				}
				idx = i

				secretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
				spec := volume.CloudInitNoCloud
				spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

				// Store userdata as k8s secret
				By("Creating a user-data secret")
				secret := kubev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretID,
						Namespace: vmi.Namespace,
						Labels: map[string]string{
							util.SecretLabel: secretID,
						},
					},
					Type: "Opaque",
					Data: map[string][]byte{
						"userdata": []byte(userData), // The client encrypts the secret for us
					},
				}
				_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				break
			}

			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

			CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

			By("Checking whether the user-data script had created the file")
			Expect(console.RunCommand(vmi, fmt.Sprintf("cat /%s\n", expectedUserDataFile), time.Second*120)).To(Succeed())

			// Expect that the secret is not present on the vmi itself
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())
		})

		Context("with cloudInitNoCloud networkData", func() {
			It("[test_id:3181]should have cloud-init network-config with NetworkData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, false)
				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				MountCloudInitNoCloud(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "network-config", testNetworkData)

			})
			It("[test_id:3182]should have cloud-init network-config with NetworkDataBase64 source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, true)
				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				MountCloudInitNoCloud(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "network-config", testNetworkData)

			})
			It("[test_id:3183]should have cloud-init network-config from k8s secret", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", "", false)

				idx := 0
				for i, volume := range vmi.Spec.Volumes {
					if volume.CloudInitNoCloud == nil {
						continue
					}
					idx = i

					secretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
					spec := volume.CloudInitNoCloud
					spec.NetworkDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

					// Store cloudinit data as k8s secret
					By("Creating a secret with network data")
					secret := kubev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretID,
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								util.SecretLabel: secretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							"networkdata": []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					break
				}

				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				MountCloudInitNoCloud(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "network-config", testNetworkData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.NetworkDataBase64).To(BeEmpty())
			})

		})

		Context("with cloudInitConfigDrive networkData", func() {
			It("[test_id:3184]should have cloud-init network-config with NetworkData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, false)

				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)
			})
			It("[test_id:4622]should have cloud-init meta_data with tagged devices", func() {
				testInstancetype := "testInstancetype"
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, false)
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Tag: "specialNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[v1.InstancetypeAnnotation] = testInstancetype
				vmi = LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				nic := domSpec.Devices.Interfaces[0]
				address := nic.Address
				pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
				deviceData := []cloudinit.DeviceData{
					{
						Type:    cloudinit.NICMetadataType,
						Bus:     nic.Address.Type,
						Address: pciAddrStr,
						MAC:     nic.MAC.MAC,
						Tags:    []string{"specialNet"},
					},
				}
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID:   fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					InstanceType: testInstancetype,
					Hostname:     dns.SanitizeHostname(vmi),
					UUID:         string(vmi.Spec.Domain.Firmware.UUID),
					Devices:      &deviceData,
				}

				buf, err := json.Marshal(metadataStruct)
				Expect(err).ToNot(HaveOccurred())
				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				By("checking cloudinit meta-data")
				tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))
			})
			It("[test_id:3185]should have cloud-init network-config with NetworkDataBase64 source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, true)
				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)
				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

			})
			It("[test_id:3186]should have cloud-init network-config from k8s secret", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", "", false)

				idx := 0
				for i, volume := range vmi.Spec.Volumes {
					if volume.CloudInitConfigDrive == nil {
						continue
					}
					idx = i

					secretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
					spec := volume.CloudInitConfigDrive
					spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}
					spec.NetworkDataSecretRef = &kubev1.LocalObjectReference{Name: secretID}

					// Store cloudinit data as k8s secret
					By("Creating a secret with user and network data")
					secret := kubev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      secretID,
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								util.SecretLabel: secretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							"networkdata": []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					break
				}

				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
			})

			DescribeTable("[test_id:3187]should have cloud-init userdata and network-config from separate k8s secrets", func(userDataLabel string, networkDataLabel string) {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", "", false)

				idx := 0
				for i, volume := range vmi.Spec.Volumes {
					if volume.CloudInitConfigDrive == nil {
						continue
					}
					idx = i

					uSecretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
					spec := volume.CloudInitConfigDrive
					spec.UserDataSecretRef = &kubev1.LocalObjectReference{Name: uSecretID}

					nSecretID := fmt.Sprintf("%s-test-secret", uuid.NewRandom().String())
					spec.NetworkDataSecretRef = &kubev1.LocalObjectReference{Name: nSecretID}

					// Store cloudinit data as k8s secret
					By("Creating a secret with userdata")
					uSecret := kubev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      uSecretID,
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								util.SecretLabel: uSecretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							userDataLabel: []byte(testUserData),
						},
					}
					By("Creating a secret with network data")
					nSecret := kubev1.Secret{
						ObjectMeta: metav1.ObjectMeta{
							Name:      nSecretID,
							Namespace: vmi.Namespace,
							Labels: map[string]string{
								util.SecretLabel: nSecretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							networkDataLabel: []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &uSecret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &nSecret, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())

					break
				}

				LaunchVMI(vmi)
				libwait.WaitUntilVMIReady(vmi, console.LoginToCirros)

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mounting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "openstack/latest/user_data", testUserData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.UserData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.UserDataBase64).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
			},
				Entry("with lowercase labels", "userdata", "networkdata"),
				Entry("with camelCase labels", "userData", "networkData"),
			)
		})

	})
})
