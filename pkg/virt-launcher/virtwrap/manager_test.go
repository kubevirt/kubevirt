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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package virtwrap

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	libvirt "libvirt.org/libvirt-go"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"

	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/util/net/ip"

	v1 "kubevirt.io/client-go/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("Manager", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var ctrl *gomock.Controller
	var testVirtShareDir string
	testVmName := "testvmi"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)

	tmpDir, _ := ioutil.TempDir("", "cloudinittest")
	isoCreationFunc := func(isoOutFile, volumeID string, inDir string) error {
		_, err := os.Create(isoOutFile)
		return err
	}
	BeforeSuite(func() {
		err := cloudinit.SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		defer os.RemoveAll(tmpDir)
		ephemeraldiskutils.MockDefaultOwnershipManager()
		cloudinit.SetIsoCreationFunction(isoCreationFunc)
		setOSChosenMigrationProxyPort(true)
	})

	AfterSuite(func() {
		setOSChosenMigrationProxyPort(false)
	})

	BeforeEach(func() {
		testVirtShareDir = fmt.Sprintf("fake-%d", GinkgoRandomSeed())
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		mockDomain.EXPECT().IsPersistent().AnyTimes().Return(true, nil)
	})

	expectIsolationDetectionForVMI := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
		domain := &api.Domain{}
		hotplugVolumes := make(map[string]v1.VolumeStatus)
		permanentVolumes := make(map[string]v1.VolumeStatus)
		for _, status := range vmi.Status.VolumeStatus {
			if status.HotplugVolume != nil {
				hotplugVolumes[status.Name] = status
			} else {
				permanentVolumes[status.Name] = status
			}
		}

		c := &converter.ConverterContext{
			Architecture:     runtime.GOARCH,
			VirtualMachine:   vmi,
			UseEmulation:     true,
			SMBios:           &cmdv1.SMBios{},
			HotplugVolumes:   hotplugVolumes,
			PermanentVolumes: permanentVolumes,
		}
		Expect(converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	Context("on successful VirtualMachineInstance sync", func() {
		It("should define and start a new VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainSpec := expectIsolationDetectionForVMI(vmi)

			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should define and start a new VirtualMachineInstance with userData", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			userData := "fake\nuser\ndata\n"
			networkData := ""
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should define and start a new VirtualMachineInstance with userData and networkData", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should leave a defined and started VirtualMachineInstance alone", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		table.DescribeTable("should try to start a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				vmi := newVMI(testNamespace, testVmName)
				domainSpec := expectIsolationDetectionForVMI(vmi)
				xml, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).NotTo(HaveOccurred())

				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().Create().Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
				newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
				Expect(err).To(BeNil())
				Expect(newspec).ToNot(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should unpause a paused VirtualMachineInstance on SyncVMI, which was not paused by user", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should not unpause a paused VirtualMachineInstance on SyncVMI, which was paused by user", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().Suspend().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			err = manager.PauseVMI(vmi)
			Expect(err).To(BeNil())

			mockDomain.EXPECT().Free()
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			// no expected call to unpause

			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should pause a VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().Suspend().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			err := manager.PauseVMI(vmi)
			Expect(err).To(BeNil())
		})
		It("should not try to pause a paused VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			// no call to suspend

			err := manager.PauseVMI(vmi)
			Expect(err).To(BeNil())
		})
		It("should unpause a VirtualMachineInstance", func() {
			isSetTimeCalled := make(chan bool, 1)
			defer close(isSetTimeCalled)

			// Make sure that we always free the domain after use
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).MaxTimes(2).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().SetTime(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(interface{}, interface{}, interface{}) {
				isSetTimeCalled <- true
			})
			mockDomain.EXPECT().Free()
			isFreeCalled := make(chan bool, 1)
			defer close(isFreeCalled)
			mockDomain.EXPECT().Free().Do(
				func() {
					isFreeCalled <- true
				})
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0, nil, "/usr/share/OVMF")

			err := manager.UnpauseVMI(vmi)
			Expect(err).To(BeNil())
			Eventually(func() bool {
				select {
				case isCalled := <-isSetTimeCalled:
					return isCalled
				default:
				}
				return false
			}, 20*time.Second, 1).Should(BeTrue(), "SetTime wasn't called")
			Eventually(func() bool {
				select {
				case isCalled := <-isFreeCalled:
					return isCalled
				default:
				}
				return false
			}, 20*time.Second, 1).Should(BeTrue(), "Free wasn't called")
		})
		It("should not try to unpause a running VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			// no call to unpause
			err := manager.UnpauseVMI(vmi)
			Expect(err).To(BeNil())

		})
		It("should not add discard=unmap if a disk is preallocated", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "permvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Cache: "none",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "permvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
			}
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectIsolationDetectionForVMI(vmi)
			domainSpec.Devices.Disks = []api.Disk{
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    "virtio",
						Device: "vda",
					},
					Driver: &api.DiskDriver{
						Cache:       "none",
						Name:        "qemu",
						Type:        "raw",
						ErrorPolicy: "stop",
					},
					Alias: api.NewUserDefinedAlias("permvolume1"),
				},
			}
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(xml string) (cli.VirDomain, error) {
				By(fmt.Sprintf("%s\n", xml))
				Expect(strings.Contains(xml, "discard=\"unmap\"")).To(BeFalse())
				return mockDomain, nil
			})
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{
				VirtualMachineSMBios: &cmdv1.SMBios{},
				PreallocatedVolumes:  []string{"permvolume1"},
			})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should hotplug a disk if a volume was hotplugged", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "permvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Cache: "none",
				},
				{
					Name: "hpvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "scsi",
						},
					},
					Cache: "none",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "permvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
				{
					Name: "hpvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv2",
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
				{
					Name:  "hpvolume1",
					Phase: v1.VolumeReady,
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod1",
						AttachPodUID:  "abcd",
					},
				},
			}
			isBlockDeviceVolume = func(volumeName string) (bool, error) {
				if volumeName == "dv1" {
					return true, nil
				}
				return false, nil
			}
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1/disk.img"))
				return true, nil
			}
			domainSpec.Devices.Disks = []api.Disk{
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    "virtio",
						Device: "vda",
					},
					Driver: &api.DiskDriver{
						Cache:       "none",
						Name:        "qemu",
						Type:        "raw",
						ErrorPolicy: "stop",
					},
					Alias: api.NewUserDefinedAlias("permvolume1"),
				},
			}
			attachDisk := api.Disk{
				Device: "disk",
				Type:   "file",
				Source: api.DiskSource{
					File: "/var/run/kubevirt/hotplug-disks/hpvolume1/disk.img",
				},
				Target: api.DiskTarget{
					Bus:    "scsi",
					Device: "sda",
				},
				Driver: &api.DiskDriver{
					Cache:       "none",
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: "stop",
					Discard:     "unmap",
				},
				Alias: api.NewUserDefinedAlias("hpvolume1"),
				Address: &api.Address{
					Type:       "drive",
					Bus:        "0",
					Controller: "0",
					Unit:       "0",
				},
			}
			xmlDomain2, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			attachBytes, err := xml.Marshal(attachDisk)
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xmlDomain)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().AttachDevice(strings.ToLower(string(attachBytes)))
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should unplug a disk if a volume was unplugged", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "permvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Cache: "none",
				},
				{
					Name: "hpvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "scsi",
						},
					},
					Cache: "none",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "permvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
				{
					Name: "hpvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv2",
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
				{
					Name:  "hpvolume1",
					Phase: v1.VolumeReady,
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod1",
						AttachPodUID:  "abcd",
					},
				},
			}
			isBlockDeviceVolume = func(volumeName string) (bool, error) {
				if volumeName == "dv1" {
					return true, nil
				}
				return false, nil
			}
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			detachDisk := api.Disk{
				Device: "disk",
				Type:   "file",
				Source: api.DiskSource{
					File: "/var/run/kubevirt/hotplug-disks/hpvolume1/disk.img",
				},
				Target: api.DiskTarget{
					Bus:    "scsi",
					Device: "sda",
				},
				Driver: &api.DiskDriver{
					Cache:       "none",
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: "stop",
					Discard:     "unmap",
				},
				Alias: api.NewUserDefinedAlias("hpvolume1"),
				Address: &api.Address{
					Type:       "drive",
					Bus:        "0",
					Controller: "0",
					Unit:       "0",
				},
			}
			detachBytes, err := xml.Marshal(detachDisk)
			Expect(err).ToNot(HaveOccurred())
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
			}

			mockConn.EXPECT().DomainDefineXML(gomock.Any()).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().DetachDevice(strings.ToLower(string(detachBytes)))
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should not plug/unplug a disk if nothing changed", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "permvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Cache: "none",
				},
				{
					Name: "hpvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "scsi",
						},
					},
					Cache: "none",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "permvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
				{
					Name: "hpvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv2",
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
				{
					Name:  "hpvolume1",
					Phase: v1.VolumeReady,
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod1",
						AttachPodUID:  "abcd",
					},
				},
			}
			isBlockDeviceVolume = func(volumeName string) (bool, error) {
				if volumeName == "dv1" {
					return true, nil
				}
				return false, nil
			}
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1/disk.img"))
				return true, nil
			}
			mockConn.EXPECT().DomainDefineXML(string(xmlDomain)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should not hotplug a disk if a volume was hotplugged, but the disk is not ready yet", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "permvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					Cache: "none",
				},
				{
					Name: "hpvolume1",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "scsi",
						},
					},
					Cache: "none",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "permvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv1",
						},
					},
				},
				{
					Name: "hpvolume1",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv2",
						},
					},
				},
			}
			vmi.Status.VolumeStatus = []v1.VolumeStatus{
				{
					Name:  "permvolume1",
					Phase: v1.VolumeReady,
				},
				{
					Name:  "hpvolume1",
					Phase: v1.VolumeReady,
					HotplugVolume: &v1.HotplugVolumeStatus{
						AttachPodName: "testpod1",
						AttachPodUID:  "abcd",
					},
				},
			}
			isBlockDeviceVolume = func(volumeName string) (bool, error) {
				if volumeName == "dv1" {
					return true, nil
				}
				return false, nil
			}
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1/disk.img"))
				return false, nil
			}
			domainSpec.Devices.Disks = []api.Disk{
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    "virtio",
						Device: "vda",
					},
					Driver: &api.DiskDriver{
						Cache:       "none",
						Name:        "qemu",
						Type:        "raw",
						ErrorPolicy: "stop",
					},
					Alias: api.NewUserDefinedAlias("permvolume1"),
				},
			}
			xmlDomain2, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xmlDomain)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
	})
	Context("test marking graceful shutdown", func() {
		It("Should set metadata when calling MarkGracefulShutdown api", func() {
			mockDomain.EXPECT().Free().AnyTimes()

			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)

			oldXML, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())

			t := true
			domainSpec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{MarkedForGracefulShutdown: &t}

			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().Return(string(oldXML), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return("<kubevirt></kubevirt>", nil)
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(xml string) (cli.VirDomain, error) {
				Expect(strings.Contains(xml, "<markedForGracefulShutdown>true</markedForGracefulShutdown>")).To(BeTrue())
				return mockDomain, nil
			})
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			manager.MarkGracefulShutdownVMI(vmi)
		})
	})
	Context("test migration monitor", func() {
		It("migration should be canceled if it's not progressing", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := &libvirt.DomainJobInfo{
				Type:          libvirt.DOMAIN_JOB_UNBOUNDED,
				DataRemaining: 32479827394,
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         2,
				CompletionTimeoutPerGiB: 300,
			}

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			manager := &LibvirtDomainManager{
				virConn:                mockConn,
				virtShareDir:           testVirtShareDir,
				notifier:               nil,
				lessPVCSpaceToleration: 0,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob()
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return("<kubevirt></kubevirt>", nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should be canceled if timeout has been reached", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			var migrationData = 32479827394
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining: uint64(migrationData),
				}
			}()

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 150,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			manager := &LibvirtDomainManager{
				virConn:                mockConn,
				virtShareDir:           testVirtShareDir,
				notifier:               nil,
				lessPVCSpaceToleration: 0,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob()
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return("<kubevirt></kubevirt>", nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should switch to PostCopy", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			var migrationData = 32479827394
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and send a different event otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					return &libvirt.DomainJobInfo{
						Type: libvirt.DOMAIN_JOB_COMPLETED,
					}
				}

				migrationData -= 125
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining: uint64(migrationData),
				}
			}

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         3,
				CompletionTimeoutPerGiB: 1,
				AllowPostCopy:           true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			manager := &LibvirtDomainManager{
				virConn:                mockConn,
				virtShareDir:           testVirtShareDir,
				notifier:               nil,
				lessPVCSpaceToleration: 0,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().DoAndReturn(func() (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().DoAndReturn(func(_ libvirt.DomainXMLFlags) (string, error) {
				xmlOriginal, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).To(BeNil())
				return string(xmlOriginal), nil
			})
			mockDomain.EXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).AnyTimes().Return(nil)
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				DoAndReturn(func(_ libvirt.DomainMetadataType, _ string, _ libvirt.DomainModificationImpact) (string, error) {
					metadata, err := xml.MarshalIndent(domainSpec.Metadata, "", "\t")
					Expect(err).ShouldNot(HaveOccurred())
					return string(metadata), nil
				}).AnyTimes()
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).AnyTimes().DoAndReturn(func(xml string) (cli.VirDomain, error) {
				Expect(strings.Contains(xml, "<mode>PostCopy</mode>")).To(BeTrue())

				if domainSpec.Metadata.KubeVirt.Migration == nil {
					domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}
				}
				domainSpec.Metadata.KubeVirt.Migration.Mode = v1.MigrationPostCopy

				return mockDomain, nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should be canceled when requested", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining: uint64(32479827394),
				}
			}()

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob().MaxTimes(1)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).AnyTimes().Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			manager.CancelVMIMigration(vmi)

		})
		It("shouldn't be able to call cancel migration more than once", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()

			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "111222333",
				StartTimestamp: &now,
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{

				UID:         vmi.Status.MigrationState.MigrationUID,
				AbortStatus: string(v1.MigrationAbortInProgress),
			}

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).AnyTimes().Return(string(domainXml), nil)

			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			err = manager.CancelVMIMigration(vmi)
			Expect(err).To(BeNil())
		})

	})

	Context("on successful VirtualMachineInstance migrate", func() {
		funcPreviousValue := ip.GetLoopbackAddress

		BeforeEach(func() {
			ip.GetLoopbackAddress = func() string {
				return "127.0.0.1"
			}
		})

		It("should prepare the target pod", func() {
			updateHostsFile = func(entry string) error {
				return nil
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
				TargetPod:    "fakepod",
			}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			err := manager.PrepareMigrationTarget(vmi, true)
			Expect(err).To(BeNil())
		})
		It("should verify that migration failure is set in the monitor thread", func() {
			isMigrationFailedSet := make(chan bool, 1)

			defer close(isMigrationFailedSet)

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_NONE,
					DataRemaining: uint64(32479827394),
				}
			}()

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			gomock.InOrder(
				mockConn.EXPECT().DomainDefineXML(gomock.Any()).Return(mockDomain, nil),
				mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
					Expect(strings.Contains(domainXml, "MigrationFailed")).To(BeTrue())
					isMigrationFailedSet <- true
					return mockDomain, nil
				}),
			)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(domainXml), nil)

			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)

			mockDomain.EXPECT().MigrateToURI3(gomock.Any(), gomock.Any(), gomock.Any()).Return(fmt.Errorf("MigrationFailed"))
			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 300,
			}
			err = manager.MigrateVMI(vmi, options)
			Expect(err).To(BeNil())
			Eventually(func() bool {
				select {
				case isSet := <-isMigrationFailedSet:
					return isSet
				default:
				}
				return false
			}, 20*time.Second, 2).Should(BeTrue(), "failed migration result wasn't set")
		})

		It("should detect inprogress migration job", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			domainSpec := expectIsolationDetectionForVMI(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{

				UID: vmi.Status.MigrationState.MigrationUID,
			}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).To(BeNil())

			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).Return(string(domainXml), nil)

			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 300,
			}
			err = manager.MigrateVMI(vmi, options)
			Expect(err).To(BeNil())
		})
		It("should correctly collect a list of disks for migration", func() {
			_true := true
			var convertedDomain = `<domain type="kvm" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <devices>
    <disk device="disk" type="block">
      <source dev="/dev/pvc_block_test"></source>
      <target bus="virtio" dev="vda"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-myvolume"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/kubevirt-ephemeral-disk/ephemeral_pvc/disk.qcow2"></source>
      <target bus="virtio" dev="vdb"></target>
      <driver cache="none" name="qemu" type="qcow2" iothread="1"></driver>
      <alias name="ua-myvolume1"></alias>
      <backingStore type="file">
        <format type="raw"></format>
        <source file="/var/run/kubevirt-private/vmi-disks/ephemeral_pvc/disk.img"></source>
      </backingStore>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vdc"></target>
      <driver name="qemu" type="raw" iothread="2"></driver>
      <alias name="ua-myvolumehost"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="virtio" dev="vdd"></target>
      <driver name="qemu" type="raw" iothread="3"></driver>
      <alias name="ua-cloudinit"></alias>
	  <readonly/>
    </disk>
  </devices>
</domain>`
			vmi := newVMI(testNamespace, testVmName)
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
					},
				},
				{
					Name: "myvolume1",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
				{
					Name: "myvolumehost",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
							Shared:   &_true,
						},
					},
				},
			}
			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)

			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(convertedDomain), nil)

			copyDisks := getDiskTargetsForMigration(mockDomain, vmi)
			Expect(copyDisks).Should(ConsistOf("vdb", "vdd"))
		})
		AfterEach(func() {
			ip.GetLoopbackAddress = funcPreviousValue
		})
	})

	Context("on successful VirtualMachineInstance kill", func() {
		table.DescribeTable("should try to undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().UndefineFlags(libvirt.DOMAIN_UNDEFINE_NVRAM).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0, nil, "/usr/share/OVMF")
				err := manager.DeleteVMI(newVMI(testNamespace, testVmName))
				Expect(err).To(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
		)
		table.DescribeTable("should try to destroy a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
				err := manager.KillVMI(newVMI(testNamespace, testVmName))
				Expect(err).To(BeNil())
			},
			table.Entry("shuttingDown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("running", libvirt.DOMAIN_RUNNING),
			table.Entry("paused", libvirt.DOMAIN_PAUSED),
		)
	})
	table.DescribeTable("check migration flags",
		func(migrationType string) {
			isBlockMigration := migrationType == "block"
			isUnsafeMigration := migrationType == "unsafe"
			allowAutoConverge := migrationType == "autoConverge"
			migrationMode := migrationType == "postCopy"
			isVmiPaused := migrationType == "paused"

			flags := generateMigrationFlags(isBlockMigration, isUnsafeMigration, allowAutoConverge, migrationMode, isVmiPaused)
			expectedMigrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

			if isBlockMigration {
				expectedMigrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
			} else if migrationType == "unsafe" {
				expectedMigrateFlags |= libvirt.MIGRATE_UNSAFE
			}
			if allowAutoConverge {
				expectedMigrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
			}
			if migrationType == "postCopy" {
				expectedMigrateFlags |= libvirt.MIGRATE_POSTCOPY
			}
			if migrationType == "paused" {
				expectedMigrateFlags |= libvirt.MIGRATE_PAUSED
			}
			Expect(flags).To(Equal(expectedMigrateFlags))
		},
		table.Entry("with block migration", "block"),
		table.Entry("without block migration", "live"),
		table.Entry("unsafe migration", "unsafe"),
		table.Entry("migration auto converge", "autoConverge"),
		table.Entry("migration using postcopy", "postCopy"),
		table.Entry("migration of paused vmi", "paused"),
	)

	table.DescribeTable("on successful list all domains",
		func(state libvirt.DomainState, kubevirtState api.LifeCycle, libvirtReason int, kubevirtReason api.StateChangeReason) {

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			mockDomain.EXPECT().GetState().Return(state, libvirtReason, nil).AnyTimes()
			mockDomain.EXPECT().GetName().Return("test", nil)
			x, err := xml.MarshalIndent(api.NewMinimalDomainSpec("test"), "", "\t")
			Expect(err).To(BeNil())

			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(x), nil)
			mockConn.EXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]cli.VirDomain{mockDomain}, nil)

			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return("<kubevirt></kubevirt>", nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			doms, err := manager.ListAllDomains()
			Expect(err).NotTo(HaveOccurred())

			Expect(len(doms)).To(Equal(1))

			domain := doms[0]
			domain.Spec.XMLName = xml.Name{}

			Expect(&domain.Spec).To(Equal(api.NewMinimalDomainSpec("test")))
			Expect(domain.Status.Status).To(Equal(kubevirtState))
			Expect(domain.Status.Reason).To(Equal(kubevirtReason))
		},
		table.Entry("crashed", libvirt.DOMAIN_CRASHED, api.Crashed, int(libvirt.DOMAIN_CRASHED_UNKNOWN), api.ReasonUnknown),
		table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF, api.Shutoff, int(libvirt.DOMAIN_SHUTOFF_DESTROYED), api.ReasonDestroyed),
		table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN, api.Shutdown, int(libvirt.DOMAIN_SHUTDOWN_USER), api.ReasonUser),
		table.Entry("unknown", libvirt.DOMAIN_NOSTATE, api.NoState, int(libvirt.DOMAIN_NOSTATE_UNKNOWN), api.ReasonUnknown),
		table.Entry("running", libvirt.DOMAIN_RUNNING, api.Running, int(libvirt.DOMAIN_RUNNING_UNKNOWN), api.ReasonUnknown),
		table.Entry("paused", libvirt.DOMAIN_PAUSED, api.Paused, int(libvirt.DOMAIN_PAUSED_STARTING_UP), api.ReasonPausedStartingUp),
	)

	Context("on successful GetAllDomainStats", func() {
		It("should return content", func() {
			mockConn.EXPECT().GetDomainStats(
				gomock.Eq(libvirt.DOMAIN_STATS_BALLOON|libvirt.DOMAIN_STATS_CPU_TOTAL|libvirt.DOMAIN_STATS_VCPU|libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BLOCK),
				gomock.Eq(libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING),
			).Return([]*stats.DomainStats{
				{},
			}, nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")
			domStats, err := manager.GetDomainStats()

			Expect(err).To(BeNil())
			Expect(len(domStats)).To(Equal(1))
		})
	})

	Context("on failed GetDomainSpecWithRuntimeInfo", func() {
		It("should fall back to returning domain spec without runtime info", func() {
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, nil, 0, nil, "/usr/share/OVMF")

			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			vmi := newVMI(testNamespace, testVmName)

			domainSpec := expectIsolationDetectionForVMI(vmi)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())

			gomock.InOrder(
				// First call is via GetDomainSpecWithRuntimeInfo. Force an error
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return("", libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}),
				// Subsequent calls are via GetDomainSpec
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(libvirt.DOMAIN_XML_INACTIVE)).MaxTimes(2).Return(string(domainXml), nil),
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(libvirt.DOMAIN_XML_MIGRATABLE)).MaxTimes(2).Return(string(domainXml), nil),
			)

			// we need the non-typecast object to make the function we want to test available
			libvirtmanager := manager.(*LibvirtDomainManager)

			domSpec, err := libvirtmanager.getDomainSpec(mockDomain)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec).ToNot(BeNil())
		})
	})

	// TODO: test error reporting on non successful VirtualMachineInstance syncs and kill attempts

	AfterEach(func() {
		ctrl.Finish()
	})
})

var _ = Describe("getEnvAddressListByPrefix with gpu prefix", func() {
	It("returns empty if PCI address is not set", func() {
		Expect(len(getEnvAddressListByPrefix(gpuEnvPrefix))).To(Equal(0))
	})

	It("returns single PCI address ", func() {
		os.Setenv("GPU_PASSTHROUGH_DEVICES_SOME_VENDOR", "2609:19:90.0,")
		addrs := getEnvAddressListByPrefix(gpuEnvPrefix)
		Expect(len(addrs)).To(Equal(1))
		Expect(addrs[0]).To(Equal("2609:19:90.0"))
	})

	It("returns multiple PCI addresses", func() {
		os.Setenv("GPU_PASSTHROUGH_DEVICES_SOME_VENDOR", "2609:19:90.0,2609:19:90.1")
		addrs := getEnvAddressListByPrefix(gpuEnvPrefix)
		Expect(len(addrs)).To(Equal(2))
		Expect(addrs[0]).To(Equal("2609:19:90.0"))
		Expect(addrs[1]).To(Equal("2609:19:90.1"))
	})
})

var _ = Describe("getEnvAddressListByPrefix with vgpu prefix", func() {
	It("returns empty if Mdev Uuid is not set", func() {
		Expect(len(getEnvAddressListByPrefix(vgpuEnvPrefix))).To(Equal(0))
	})

	It("returns single  Mdev Uuid ", func() {
		os.Setenv("VGPU_PASSTHROUGH_DEVICES_SOME_VENDOR", "aa618089-8b16-4d01-a136-25a0f3c73123,")
		addrs := getEnvAddressListByPrefix(vgpuEnvPrefix)
		Expect(len(addrs)).To(Equal(1))
		Expect(addrs[0]).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73123"))
	})

	It("returns multiple  Mdev Uuid", func() {
		os.Setenv("VGPU_PASSTHROUGH_DEVICES_SOME_VENDOR", "aa618089-8b16-4d01-a136-25a0f3c73123,aa618089-8b16-4d01-a136-25a0f3c73124")
		addrs := getEnvAddressListByPrefix(vgpuEnvPrefix)
		Expect(len(addrs)).To(Equal(2))
		Expect(addrs[0]).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73123"))
		Expect(addrs[1]).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73124"))
	})

})

var _ = Describe("getAttachedDisks", func() {
	table.DescribeTable("should return the correct values", func(oldDisks, newDisks, expected []api.Disk) {
		res := getAttachedDisks(oldDisks, newDisks)
		Expect(res).To(Equal(expected))
	},
		table.Entry("be empty with empty old and new",
			[]api.Disk{},
			[]api.Disk{},
			[]api.Disk{}),
		table.Entry("be empty with empty old and new being identical",
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{}),
		table.Entry("contain a new disk with empty having a new disk compared to old",
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
				{
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			}),
	)
})

var _ = Describe("getDetachedDisks", func() {
	table.DescribeTable("should return the correct values", func(oldDisks, newDisks, expected []api.Disk) {
		res := getDetachedDisks(oldDisks, newDisks)
		Expect(res).To(Equal(expected))
	},
		table.Entry("be empty with empty old and new",
			[]api.Disk{},
			[]api.Disk{},
			[]api.Disk{}),
		table.Entry("be empty with empty old and new being identical",
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{}),
		table.Entry("contains something if new has less than old",
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
				{
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			}),
	)
})

var _ = Describe("migratableDomXML", func() {
	var ctrl *gomock.Controller
	var mockDomain *cli.MockVirDomain
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockDomain = cli.NewMockVirDomain(ctrl)
	})
	It("should remove only the kubevirt metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <metadata>
         <kubevirt>nested</kubevirt>
      </metadata>
      <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
    <othermetadata>
      <kubevirt>
         <keepme>42</keepme>
      </kubevirt>
    </othermetadata>
  </metadata>
  <kubevirt>this should stay</kubevirt>
</domain>`
		// migratableDomXML() removes the kubevirt block but not its ident, which is its own token, hence the blank line below
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    
    <othermetadata>
      <kubevirt>
         <keepme>42</keepme>
      </kubevirt>
    </othermetadata>
  </metadata>
  <kubevirt>this should stay</kubevirt>
</domain>`
		mockDomain.EXPECT().Free()
		vmi := newVMI("testns", "kubevirt")
		mockDomain.EXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(string(domXML), nil)
		newXML, err := migratableDomXML(mockDomain, vmi)
		Expect(err).To(BeNil())
		Expect(newXML).To(Equal(expectedXML))
	})
})

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := v1.NewMinimalVMIWithNS(namespace, name)
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func addCloudInitDisk(vmi *v1.VirtualMachineInstance, userData string, networkData string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:  "cloudinit",
		Cache: v1.CacheWriteThrough,
		IO:    v1.IONative,
		DiskDevice: v1.DiskDevice{
			Disk: &v1.DiskTarget{
				Bus: "virtio",
			},
		},
	})
	vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
		Name: "cloudinit",
		VolumeSource: v1.VolumeSource{
			CloudInitNoCloud: &v1.CloudInitNoCloudSource{
				UserDataBase64:    base64.StdEncoding.EncodeToString([]byte(userData)),
				NetworkDataBase64: base64.StdEncoding.EncodeToString([]byte(networkData)),
			},
		},
	})
}
