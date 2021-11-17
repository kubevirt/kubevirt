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
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"
	kubev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	sshAuthorizedKey = "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEA6NF8iallvQVp22WDkT test-ssh-key"
	fedoraPassword   = "fedora"
	expectedUserData = "printed from cloud-init userdata"
	testNetworkData  = "#Test networkData"
	testUserData     = "#cloud-config"
)

var _ = Describe("[rfe_id:151][crit:high][vendor:cnv-qe@redhat.com][level:component][sig-compute]CloudInit UserData", func() {

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

	tests.BeforeAll(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		// from default virt-launcher flag: do we need to make this configurable in some cases?
		cloudinit.SetLocalDirectoryOnly("/var/run/kubevirt-ephemeral-disks/cloud-init-data")

		LaunchVMI = func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			By("Starting a VirtualMachineInstance")
			obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(util.NamespaceTestDefault).Body(vmi).Do(context.Background()).Get()
			Expect(err).To(BeNil())

			By("Waiting the VirtualMachineInstance start")
			vmi, ok := obj.(*v1.VirtualMachineInstance)
			Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
			Expect(tests.WaitForSuccessfulVMIStart(obj)).ToNot(BeEmpty())
			return vmi
		}

		VerifyUserDataVMI = func(vmi *v1.VirtualMachineInstance, commands []expect.Batcher, timeout time.Duration) {
			By("Checking that the VirtualMachineInstance serial console output equals to expected one")
			Expect(console.SafeExpectBatch(vmi, commands, int(timeout.Seconds()))).To(Succeed())
		}

		MountCloudInitNoCloud = tests.MountCloudInitFunc("cidata")
		MountCloudInitConfigDrive = tests.MountCloudInitFunc("config-2")

		CheckCloudInitFile = func(vmi *v1.VirtualMachineInstance, testFile, testData string) {
			cmdCheck := "cat /mnt/" + testFile + "\n"
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

			out, err := tests.ExecuteCommandOnPod(virtClient, pod, "compute", cmdCheck)
			Expect(err).NotTo(HaveOccurred())
			size, err := strconv.Atoi(strings.Trim(out, "'"))
			Expect(err).NotTo(HaveOccurred())
			Expect(size % 4096).To(Equal(0))
		}
	})

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Describe("[rfe_id:151][crit:medium][vendor:cnv-qe@redhat.com][level:component]A new VirtualMachineInstance", func() {
		Context("with cloudInitNoCloud userDataBase64 source", func() {
			It("[test_id:1615]should have cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)
				LaunchVMI(vmi)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)
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
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)

				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)
				LaunchVMI(vmi)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)
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
		})

		Context("with cloudInitNoCloud userData source", func() {
			It("[test_id:1617]should process provided cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

				vmi = LaunchVMI(vmi)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)
				By("executing a user-data script")
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)

				By("applying the hostname from meta-data")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "hostname\n"},
					&expect.BExp{R: dns.SanitizeHostname(vmi)},
				}, 10)).To(Succeed())
			})
		})

		Context("with cloudInitConfigDrive userData source", func() {
			It("[test_id:3180]should process provided cloud-init data", func() {
				userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdata(cd.ContainerDiskFor(cd.ContainerDiskCirros), userData)

				vmi = LaunchVMI(vmi)
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)
				By("executing a user-data script")
				VerifyUserDataVMI(vmi, []expect.Batcher{
					&expect.BSnd{S: "\n"},
					&expect.BExp{R: expectedUserData},
				}, time.Second*120)

				By("applying the hostname from meta-data")
				Expect(libnet.WithIPv6(console.LoginToCirros)(vmi)).To(Succeed())

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "hostname\n"},
					&expect.BExp{R: dns.SanitizeHostname(vmi)},
				}, 10)).To(Succeed())
			})
		})

		It("[test_id:1618]should take user-data from k8s secret", func() {
			userData := fmt.Sprintf("#!/bin/sh\n\necho '%s'\n", expectedUserData)
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
							tests.SecretLabel: secretID,
						},
					},
					Type: "Opaque",
					Data: map[string][]byte{
						"userdata": []byte(userData), // The client encrypts the secret for us
					},
				}
				_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
				Expect(err).To(BeNil())
				break
			}
			LaunchVMI(vmi)
			CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)
			VerifyUserDataVMI(vmi, []expect.Batcher{
				&expect.BSnd{S: "\n"},
				&expect.BExp{R: expectedUserData},
			}, time.Second*120)

			// Expect that the secret is not present on the vmi itself
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserData).To(BeEmpty())
			Expect(vmi.Spec.Volumes[idx].CloudInitNoCloud.UserDataBase64).To(BeEmpty())
		})

		Context("with cloudInitNoCloud networkData", func() {
			It("[test_id:3181]should have cloud-init network-config with NetworkData source", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, false)
				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

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
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

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
								tests.SecretLabel: secretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							"networkdata": []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
					Expect(err).To(BeNil())

					break
				}

				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceNoCloud)

				By("mouting cloudinit iso")
				MountCloudInitNoCloud(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "network-config", testNetworkData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
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
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)
			})
			It("[test_id:4622]should have cloud-init meta_data with tagged devices", func() {
				vmi := tests.NewRandomVMIWithEphemeralDiskAndConfigDriveUserdataNetworkData(
					cd.ContainerDiskFor(cd.ContainerDiskCirros), "", testNetworkData, false)
				vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{{Name: "default", Tag: "specialNet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}}
				vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))
				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				nic := domSpec.Devices.Interfaces[0]
				address := nic.Address
				pciAddrStr := fmt.Sprintf("%s:%s:%s:%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
				deviceData := []cloudinit.DeviceData{
					{
						Type:    cloudinit.NICMetadataType,
						Bus:     nic.Address.Type,
						Address: pciAddrStr,
						MAC:     nic.MAC.MAC,
						Tags:    []string{"specialNet"},
					},
				}
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID: fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					Hostname:   dns.SanitizeHostname(vmi),
					UUID:       string(vmi.UID),
					Devices:    &deviceData,
				}

				buf, err := json.Marshal(metadataStruct)
				Expect(err).To(BeNil())
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
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

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
								tests.SecretLabel: secretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							"networkdata": []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &secret, metav1.CreateOptions{})
					Expect(err).To(BeNil())

					break
				}

				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mouting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
			})

			table.DescribeTable("[test_id:3187]should have cloud-init userdata and network-config from separate k8s secrets", func(userDataLabel string, networkDataLabel string) {
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
								tests.SecretLabel: uSecretID,
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
								tests.SecretLabel: nSecretID,
							},
						},
						Type: "Opaque",
						Data: map[string][]byte{
							// The client encrypts the secret for us
							networkDataLabel: []byte(testNetworkData),
						},
					}
					_, err := virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &uSecret, metav1.CreateOptions{})
					Expect(err).To(BeNil())

					_, err = virtClient.CoreV1().Secrets(vmi.Namespace).Create(context.Background(), &nSecret, metav1.CreateOptions{})
					Expect(err).To(BeNil())

					break
				}

				LaunchVMI(vmi)
				tests.WaitUntilVMIReady(vmi, libnet.WithIPv6(console.LoginToCirros))

				CheckCloudInitIsoSize(vmi, cloudinit.DataSourceConfigDrive)

				By("mounting cloudinit iso")
				MountCloudInitConfigDrive(vmi)

				By("checking cloudinit network-config")
				CheckCloudInitFile(vmi, "openstack/latest/network_data.json", testNetworkData)

				By("checking cloudinit user-data")
				CheckCloudInitFile(vmi, "openstack/latest/user_data", testUserData)

				// Expect that the secret is not present on the vmi itself
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.UserData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.UserDataBase64).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkData).To(BeEmpty())
				Expect(vmi.Spec.Volumes[idx].CloudInitConfigDrive.NetworkDataBase64).To(BeEmpty())
			},
				table.Entry("with lowercase labels", "userdata", "networkdata"),
				table.Entry("with camelCase labels", "userData", "networkData"),
			)
		})

	})
})
