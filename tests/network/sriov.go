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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package network

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/network/vmispec"
	"kubevirt.io/kubevirt/pkg/util/hardware"
	"kubevirt.io/kubevirt/pkg/util/net/dns"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/testsuite"
	"kubevirt.io/kubevirt/tests/util"
)

const (
	sriovLinkEnableConfNAD = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
	sriovVlanConfNAD       = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"link_state\": \"enable\", \"vlan\": 200, \"ipam\":{}}"}}`
	sriovConfNAD           = `{"apiVersion":"k8s.cni.cncf.io/v1","kind":"NetworkAttachmentDefinition","metadata":{"name":"%s","namespace":"%s","annotations":{"k8s.v1.cni.cncf.io/resourceName":"%s"}},"spec":{"config":"{ \"cniVersion\": \"0.3.1\", \"name\": \"sriov\", \"type\": \"sriov\", \"vlan\": 0, \"ipam\": { \"type\": \"host-local\", \"subnet\": \"10.1.1.0/24\" } }"}}`
)

const (
	sriovnet1           = "sriov"
	sriovnet2           = "sriov2"
	sriovnetLinkEnabled = "sriov-linked"
)

var _ = Describe("[Serial]SRIOV", func() {

	var err error
	var virtClient kubecli.KubevirtClient

	sriovResourceName := os.Getenv("SRIOV_RESOURCE_NAME")

	if sriovResourceName == "" {
		sriovResourceName = "kubevirt.io/sriov_net"
	}

	createSriovNetworkAttachmentDefinition := func(networkName string, namespace string, networkAttachmentDefinition string) error {
		sriovNad := fmt.Sprintf(networkAttachmentDefinition, networkName, namespace, sriovResourceName)
		return createNetworkAttachmentDefinition(virtClient, networkName, namespace, sriovNad)
	}

	BeforeEach(func() {
		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)

		// Check if the hardware supports SRIOV
		if err := validateSRIOVSetup(virtClient, sriovResourceName, 1); err != nil {
			Skip("Sriov is not enabled in this environment. Skip these tests using - export FUNC_TEST_ARGS='--skip=SRIOV'")
		}
	})

	Context("VirtualMachineInstance with sriov plugin interface", func() {

		getSriovVmi := func(networks []string, cloudInitNetworkData string) *v1.VirtualMachineInstance {
			withVmiOptions := []libvmi.Option{
				libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
				libvmi.WithNetwork(v1.DefaultPodNetwork()),
			}
			if cloudInitNetworkData != "" {
				cloudinitOption := libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkData, false)
				withVmiOptions = append(withVmiOptions, cloudinitOption)
			}
			// sriov network interfaces
			for _, name := range networks {
				withVmiOptions = append(withVmiOptions,
					libvmi.WithInterface(libvmi.InterfaceDeviceWithSRIOVBinding(name)),
					libvmi.WithNetwork(libvmi.MultusNetwork(name, name)),
				)
			}
			return libvmi.NewFedora(withVmiOptions...)
		}

		startVmi := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Create(vmi)
			Expect(err).ToNot(HaveOccurred())
			return vmi
		}

		waitVmi := func(vmi *v1.VirtualMachineInstance) *v1.VirtualMachineInstance {
			// Need to wait for cloud init to finish and start the agent inside the vmi.
			vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			// Running multi sriov jobs with Kind, DinD is resource extensive, causing DeadlineExceeded transient warning
			// Kubevirt re-enqueue the request once it happens, so its safe to ignore this warning.
			// see https://github.com/kubevirt/kubevirt/issues/5027
			warningsIgnoreList := []string{"unknown error encountered sending command SyncVMI: rpc error: code = DeadlineExceeded desc = context deadline exceeded"}
			tests.WaitUntilVMIReadyIgnoreSelectedWarnings(vmi, console.LoginToFedora, warningsIgnoreList)
			tests.WaitAgentConnected(virtClient, vmi)
			return vmi
		}

		checkDefaultInterfaceInPod := func(vmi *v1.VirtualMachineInstance) {
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

			By("checking default interface is present")
			_, err = tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				"compute",
				[]string{"ip", "address", "show", "eth0"},
			)
			Expect(err).ToNot(HaveOccurred())

			By("checking default interface is attached to VMI")
			_, err = tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				"compute",
				[]string{"ip", "address", "show", "k6t-eth0"},
			)
			Expect(err).ToNot(HaveOccurred())
		}

		checkInterfacesInGuest := func(vmi *v1.VirtualMachineInstance, interfaces []string) {
			for _, iface := range interfaces {
				Expect(checkInterface(vmi, iface)).To(Succeed())
			}
		}

		// createSriovVMs instantiates two VMs on the same node connected through SR-IOV.
		createSriovVMs := func(networkNameA, networkNameB, cidrA, cidrB string) (*v1.VirtualMachineInstance, *v1.VirtualMachineInstance) {
			// Explicitly choose different random mac addresses instead of relying on kubemacpool to do it:
			// 1) we don't at the moment deploy kubemacpool in kind providers
			// 2) even if we would do, it's probably a good idea to have the suite not depend on this fact
			//
			// This step is needed to guarantee that no VFs on the PF carry a duplicate MAC address that may affect
			// ability of VMIs to send and receive ICMP packets on their ports.
			mac1, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())

			mac2, err := GenerateRandomMac()
			Expect(err).ToNot(HaveOccurred())

			// start peer machines with sriov interfaces from the same resource pool
			// manually configure IP/link on sriov interfaces because there is
			// no DHCP server to serve the address to the guest
			vmi1 := getSriovVmi([]string{networkNameA}, cloudInitNetworkDataWithStaticIPsByMac(networkNameA, mac1.String(), cidrA))
			vmi2 := getSriovVmi([]string{networkNameB}, cloudInitNetworkDataWithStaticIPsByMac(networkNameB, mac2.String(), cidrB))

			vmi1.Spec.Domain.Devices.Interfaces[1].MacAddress = mac1.String()
			vmi2.Spec.Domain.Devices.Interfaces[1].MacAddress = mac2.String()

			// schedule both VM's on the same node to prevent test from being affected by how the SR-IOV card port's are connected
			sriovNodes := getNodesWithAllocatedResource(virtClient, sriovResourceName)
			Expect(sriovNodes).ToNot(BeEmpty())
			sriovNode := sriovNodes[0].Name
			vmi1 = tests.CreateVmiOnNode(vmi1, sriovNode)
			vmi2 = tests.CreateVmiOnNode(vmi2, sriovNode)

			vmi1 = waitVmi(vmi1)
			vmi2 = waitVmi(vmi2)

			vmi1, err = virtClient.VirtualMachineInstance(vmi1.Namespace).Get(vmi1.Name, &k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			vmi2, err = virtClient.VirtualMachineInstance(vmi2.Namespace).Get(vmi2.Name, &k8smetav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())

			return vmi1, vmi2
		}

		Context("Connected to single SRIOV network", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).
					To(Succeed(), shouldCreateNetwork)
			})

			It("should block migration for SR-IOV VMI's when LiveMigration feature-gate is on but SRIOVLiveMigration is off", func() {
				tests.EnableFeatureGate(virtconfig.LiveMigrationGate)
				defer tests.UpdateKubeVirtConfigValueAndWait(testsuite.KubeVirtDefaultConfig)

				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				vmim := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
				Eventually(func() error {
					_, err = virtClient.VirtualMachineInstanceMigration(vmim.Namespace).Create(vmim, &k8smetav1.CreateOptions{})
					return err
				}, 1*time.Minute, 20*time.Second).ShouldNot(Succeed())
			})
			It("should have cloud-init meta_data with aligned cpus to sriov interface numa node for VMIs with dedicatedCPUs", func() {
				checks.SkipTestIfNoCPUManager()
				noCloudInitNetworkData := ""
				vmi := getSriovVmi([]string{sriovnet1}, noCloudInitNetworkData)
				tests.AddCloudInitConfigDriveData(vmi, "disk1", "", defaultCloudInitNetworkData(), false)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 4,
					DedicatedCPUPlacement: true,
				}

				for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
					if iface.Name == sriovnet1 {
						iface.Tag = "specialNet"
						vmi.Spec.Domain.Devices.Interfaces[idx] = iface
					}
				}
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				nic := domSpec.Devices.HostDevices[0]
				// find the SRIOV interface
				for _, iface := range domSpec.Devices.HostDevices {
					if iface.Alias.GetName() == sriovnet1 {
						nic = iface
					}
				}
				address := nic.Address
				pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
				srcAddr := nic.Source.Address
				sourcePCIAddress := fmt.Sprintf("%s:%s:%s.%s", srcAddr.Domain[2:], srcAddr.Bus[2:], srcAddr.Slot[2:], srcAddr.Function[2:])
				alignedCPUsInt, err := hardware.LookupDeviceVCPUAffinity(sourcePCIAddress, domSpec)
				Expect(err).ToNot(HaveOccurred())
				deviceData := []cloudinit.DeviceData{
					{
						Type:        cloudinit.NICMetadataType,
						Bus:         nic.Address.Type,
						Address:     pciAddrStr,
						Tags:        []string{"specialNet"},
						AlignedCPUs: alignedCPUsInt,
					},
				}
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID: fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					Hostname:   dns.SanitizeHostname(vmi),
					UUID:       string(vmi.Spec.Domain.Firmware.UUID),
					Devices:    &deviceData,
				}

				buf, err := json.Marshal(metadataStruct)
				Expect(err).To(BeNil())
				By("mouting cloudinit iso")
				mountCloudInitConfigDrive := tests.MountCloudInitFunc("config-2")
				mountCloudInitConfigDrive(vmi)

				By("checking cloudinit meta-data")
				tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))

			})

			It("should have cloud-init meta_data with tagged sriov nics", func() {
				noCloudInitNetworkData := ""
				vmi := getSriovVmi([]string{sriovnet1}, noCloudInitNetworkData)
				testInstancetype := "testInstancetype"
				if vmi.Annotations == nil {
					vmi.Annotations = make(map[string]string)
				}
				vmi.Annotations[v1.InstancetypeAnnotation] = testInstancetype
				tests.AddCloudInitConfigDriveData(vmi, "disk1", "", defaultCloudInitNetworkData(), false)

				for idx, iface := range vmi.Spec.Domain.Devices.Interfaces {
					if iface.Name == sriovnet1 {
						iface.Tag = "specialNet"
						vmi.Spec.Domain.Devices.Interfaces[idx] = iface
					}
				}
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				domXml, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
				Expect(err).ToNot(HaveOccurred())

				domSpec := &api.DomainSpec{}
				Expect(xml.Unmarshal([]byte(domXml), domSpec)).To(Succeed())
				nic := domSpec.Devices.HostDevices[0]
				// find the SRIOV interface
				for _, iface := range domSpec.Devices.HostDevices {
					if iface.Alias.GetName() == sriovnet1 {
						nic = iface
					}
				}
				address := nic.Address
				pciAddrStr := fmt.Sprintf("%s:%s:%s.%s", address.Domain[2:], address.Bus[2:], address.Slot[2:], address.Function[2:])
				deviceData := []cloudinit.DeviceData{
					{
						Type:    cloudinit.NICMetadataType,
						Bus:     nic.Address.Type,
						Address: pciAddrStr,
						Tags:    []string{"specialNet"},
					},
				}
				vmi, err = virtClient.VirtualMachineInstance(util.NamespaceTestDefault).Get(vmi.Name, &k8smetav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				metadataStruct := cloudinit.ConfigDriveMetadata{
					InstanceID:   fmt.Sprintf("%s.%s", vmi.Name, vmi.Namespace),
					InstanceType: testInstancetype,
					Hostname:     dns.SanitizeHostname(vmi),
					UUID:         string(vmi.Spec.Domain.Firmware.UUID),
					Devices:      &deviceData,
				}

				buf, err := json.Marshal(metadataStruct)
				Expect(err).To(BeNil())
				By("mouting cloudinit iso")
				mountCloudInitConfigDrive := tests.MountCloudInitFunc("config-2")
				mountCloudInitConfigDrive(vmi)

				By("checking cloudinit meta-data")
				tests.CheckCloudInitMetaData(vmi, "openstack/latest/meta_data.json", string(buf))
			})

			It("[test_id:1754]should create a virtual machine with sriov interface", func() {
				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

				checkDefaultInterfaceInPod(vmi)

				By("checking virtual machine instance has two interfaces")
				checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

				// there is little we can do beyond just checking two devices are present: PCI slots are different inside
				// the guest, and DP doesn't pass information about vendor IDs of allocated devices into the pod, so
				// it's hard to match them.
			})

			It("[test_id:1754]should create a virtual machine with sriov interface with all pci devices on the root bus", func() {
				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi.Annotations = map[string]string{
					v1.PlacePCIDevicesOnRootComplex: "true",
				}
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

				checkDefaultInterfaceInPod(vmi)

				By("checking virtual machine instance has two interfaces")
				checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})

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

			It("[test_id:3959]should create a virtual machine with sriov interface and dedicatedCPUs", func() {
				checks.SkipTestIfNoCPUManager()
				// In addition to verifying that we can start a VMI with CPU pinning
				// this also tests if we've correctly calculated the overhead for VFIO devices.
				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:                 2,
					DedicatedCPUPlacement: true,
				}
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variable is defined in pod")
				Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, sriovnet1, sriovResourceName)).To(Succeed())

				checkDefaultInterfaceInPod(vmi)

				By("checking virtual machine instance has two interfaces")
				checkInterfacesInGuest(vmi, []string{"eth0", "eth1"})
			})
			It("[test_id:3985]should create a virtual machine with sriov interface with custom MAC address", func() {
				const mac = "de:ad:00:00:be:ef"
				vmi := getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
				vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking virtual machine instance has an interface with the requested MAC address")
				ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 140*time.Second)
				Expect(err).NotTo(HaveOccurred())
				vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				Expect(checkMacAddress(vmi, ifaceName, mac)).To(Succeed())

				By("checking virtual machine instance reports the expected network name")
				Expect(getInterfaceNetworkNameByMAC(vmi, mac)).To(Equal(sriovnet1))
				By("checking virtual machine instance reports the expected info source")
				networkInterface := vmispec.LookupInterfaceStatusByMac(vmi.Status.Interfaces, mac)
				Expect(networkInterface).NotTo(BeNil(), "interface not found")
				Expect(networkInterface.InfoSource).To(Equal(vmispec.InfoSourceDomainAndGA))
			})

			Context("migration", func() {

				BeforeEach(func() {
					if err := validateSRIOVSetup(virtClient, sriovResourceName, 2); err != nil {
						Skip("Migration tests require at least 2 nodes: " + err.Error())
					}
				})

				BeforeEach(func() {
					tests.EnableFeatureGate(virtconfig.SRIOVLiveMigrationGate)
				})

				AfterEach(func() {
					tests.DisableFeatureGate(virtconfig.SRIOVLiveMigrationGate)
				})

				var vmi *v1.VirtualMachineInstance

				const mac = "de:ad:00:00:be:ef"

				BeforeEach(func() {
					// The SR-IOV VF MAC should be preserved on migration, therefore explicitly specify it.
					vmi = getSriovVmi([]string{sriovnet1}, defaultCloudInitNetworkData())
					vmi.Spec.Domain.Devices.Interfaces[1].MacAddress = mac

					vmi = startVmi(vmi)
					vmi = waitVmi(vmi)

					ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
					Expect(err).NotTo(HaveOccurred())
					vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(checkMacAddress(vmi, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest")
				})

				It("should be successful with a running VMI on the target", func() {
					By("starting the migration")
					migration := tests.NewRandomMigration(vmi.Name, vmi.Namespace)
					migrationUID := tests.RunMigrationAndExpectCompletion(virtClient, migration, tests.MigrationWaitTime)
					tests.ConfirmVMIPostMigration(virtClient, vmi, migrationUID)

					// It may take some time for the VMI interface status to be updated with the information reported by
					// the guest-agent.
					ifaceName, err := findIfaceByMAC(virtClient, vmi, mac, 30*time.Second)
					Expect(err).NotTo(HaveOccurred())
					updatedVMI, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(checkMacAddress(updatedVMI, ifaceName, mac)).To(Succeed(), "SR-IOV VF is expected to exist in the guest after migration")
				})
			})
		})

		Context("Connected to two SRIOV networks", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnet1, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), shouldCreateNetwork)
				Expect(createSriovNetworkAttachmentDefinition(sriovnet2, util.NamespaceTestDefault, sriovConfNAD)).To(Succeed(), shouldCreateNetwork)
			})

			It("[test_id:1755]should create a virtual machine with two sriov interfaces referring the same resource", func() {
				sriovNetworks := []string{sriovnet1, sriovnet2}
				vmi := getSriovVmi(sriovNetworks, defaultCloudInitNetworkData())
				vmi.Spec.Domain.Devices.Interfaces[1].PciAddress = "0000:06:00.0"
				vmi.Spec.Domain.Devices.Interfaces[2].PciAddress = "0000:07:00.0"
				vmi = startVmi(vmi)
				vmi = waitVmi(vmi)

				By("checking KUBEVIRT_RESOURCE_NAME_<networkName> variables are defined in pod")
				for _, name := range sriovNetworks {
					Expect(validatePodKubevirtResourceNameByVMI(virtClient, vmi, name, sriovResourceName)).To(Succeed())
				}

				checkDefaultInterfaceInPod(vmi)

				By("checking virtual machine instance has three interfaces")
				checkInterfacesInGuest(vmi, []string{"eth0", "eth1", "eth2"})

				Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[1].PciAddress)).To(Succeed())
				Expect(pciAddressExistsInGuest(vmi, vmi.Spec.Domain.Devices.Interfaces[2].PciAddress)).To(Succeed())
			})
		})

		Context("Connected to link-enabled SRIOV network", func() {
			BeforeEach(func() {
				Expect(createSriovNetworkAttachmentDefinition(sriovnetLinkEnabled, util.NamespaceTestDefault, sriovLinkEnableConfNAD)).
					To(Succeed(), shouldCreateNetwork)
			})

			It("[test_id:3956]should connect to another machine with sriov interface over IPv4", func() {
				cidrA := "192.168.1.1/24"
				cidrB := "192.168.1.2/24"
				ipA, err := libnet.CidrToIP(cidrA)
				Expect(err).ToNot(HaveOccurred())
				ipB, err := libnet.CidrToIP(cidrB)
				Expect(err).ToNot(HaveOccurred())

				//create two vms on the same sriov network
				vmi1, vmi2 := createSriovVMs(sriovnetLinkEnabled, sriovnetLinkEnabled, cidrA, cidrB)

				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, ipB)
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, ipA)
				}, 15*time.Second, time.Second).Should(Succeed())
			})

			It("[test_id:3957]should connect to another machine with sriov interface over IPv6", func() {
				vmi1CIDR := "fc00::1/64"
				vmi2CIDR := "fc00::2/64"
				vmi1IP, err := libnet.CidrToIP(vmi1CIDR)
				Expect(err).ToNot(HaveOccurred())
				vmi2IP, err := libnet.CidrToIP(vmi2CIDR)
				Expect(err).ToNot(HaveOccurred())

				//create two vms on the same sriov network
				vmi1, vmi2 := createSriovVMs(sriovnetLinkEnabled, sriovnetLinkEnabled, vmi1CIDR, vmi2CIDR)

				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi1, vmi2IP)
				}, 15*time.Second, time.Second).Should(Succeed())
				Eventually(func() error {
					return libnet.PingFromVMConsole(vmi2, vmi1IP)
				}, 15*time.Second, time.Second).Should(Succeed())
			})

			Context("With VLAN", func() {
				const (
					cidrVlaned1     = "192.168.0.1/24"
					sriovnetVlanned = "sriov-vlan"
				)
				var ipVlaned1 string

				BeforeEach(func() {
					var err error
					ipVlaned1, err = libnet.CidrToIP(cidrVlaned1)
					Expect(err).ToNot(HaveOccurred())
					Expect(createSriovNetworkAttachmentDefinition(sriovnetVlanned, util.NamespaceTestDefault, sriovVlanConfNAD)).To(Succeed())
				})

				It("should be able to ping between two VMIs with the same VLAN over SRIOV network", func() {
					_, vlanedVMI2 := createSriovVMs(sriovnetVlanned, sriovnetVlanned, cidrVlaned1, "192.168.0.2/24")

					By("pinging from vlanedVMI2 and the anonymous vmi over vlan")
					Eventually(func() error {
						return libnet.PingFromVMConsole(vlanedVMI2, ipVlaned1)
					}, 15*time.Second, time.Second).ShouldNot(HaveOccurred())
				})

				It("should NOT be able to ping between Vlaned VMI and a non Vlaned VMI", func() {
					_, nonVlanedVMI := createSriovVMs(sriovnetVlanned, sriovnetLinkEnabled, cidrVlaned1, "192.168.0.3/24")

					By("pinging between nonVlanedVMIand the anonymous vmi")
					Eventually(func() error {
						return libnet.PingFromVMConsole(nonVlanedVMI, ipVlaned1)
					}, 15*time.Second, time.Second).Should(HaveOccurred())
				})
			})
		})
	})
})

func pciAddressExistsInGuest(vmi *v1.VirtualMachineInstance, pciAddress string) error {
	command := fmt.Sprintf("grep -q %s /sys/class/net/*/device/uevent\n", pciAddress)
	return console.RunCommand(vmi, command, 15*time.Second)
}

func getInterfaceNameByMAC(vmi *v1.VirtualMachineInstance, mac string) (string, error) {
	for _, iface := range vmi.Status.Interfaces {
		if iface.MAC == mac {
			return iface.InterfaceName, nil
		}
	}

	return "", fmt.Errorf("could not get sriov interface by MAC: no interface on VMI %s with MAC %s", vmi.Name, mac)
}

func getInterfaceNetworkNameByMAC(vmi *v1.VirtualMachineInstance, macAddress string) string {
	for _, iface := range vmi.Status.Interfaces {
		if iface.MAC == macAddress {
			return iface.Name
		}
	}

	return ""
}

func validateSRIOVSetup(virtClient kubecli.KubevirtClient, sriovResourceName string, minRequiredNodes int) error {
	sriovNodes := getNodesWithAllocatedResource(virtClient, sriovResourceName)
	if len(sriovNodes) < minRequiredNodes {
		return fmt.Errorf("not enough compute nodes with SR-IOV support detected")
	}
	return nil
}

func getNodesWithAllocatedResource(virtClient kubecli.KubevirtClient, resourceName string) []k8sv1.Node {
	nodes := libnode.GetAllSchedulableNodes(virtClient)
	filteredNodes := []k8sv1.Node{}
	for _, node := range nodes.Items {
		resourceList := node.Status.Allocatable
		for k, v := range resourceList {
			if string(k) == resourceName {
				if v.Value() > 0 {
					filteredNodes = append(filteredNodes, node)
					break
				}
			}
		}
	}

	return filteredNodes
}

func validatePodKubevirtResourceNameByVMI(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, networkName, sriovResourceName string) error {
	out, err := tests.ExecuteCommandOnPod(
		virtClient,
		tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace),
		"compute",
		[]string{"sh", "-c", fmt.Sprintf("echo $KUBEVIRT_RESOURCE_NAME_%s", networkName)},
	)
	if err != nil {
		return err
	}

	out = strings.TrimSuffix(out, "\n")
	if out != sriovResourceName {
		return fmt.Errorf("env settings %s didnt match %s", out, sriovResourceName)
	}

	return nil
}

func defaultCloudInitNetworkData() string {
	networkData := libnet.CreateDefaultCloudInitNetworkData()
	return networkData
}

func findIfaceByMAC(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, mac string, timeout time.Duration) (string, error) {
	var ifaceName string
	err := wait.Poll(timeout, 5*time.Second, func() (done bool, err error) {
		vmi, err := virtClient.VirtualMachineInstance(vmi.GetNamespace()).Get(vmi.GetName(), &k8smetav1.GetOptions{})
		if err != nil {
			return false, err
		}

		ifaceName, err = getInterfaceNameByMAC(vmi, mac)
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("could not find interface with MAC %q on VMI %q: %v", mac, vmi.Name, err)
	}
	return ifaceName, nil
}
