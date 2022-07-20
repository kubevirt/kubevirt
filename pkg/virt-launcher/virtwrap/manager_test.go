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
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"libvirt.org/go/libvirt"

	api2 "kubevirt.io/client-go/api"

	v1 "kubevirt.io/api/core/v1"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var (
	expectedThawedOutput = `{"return":"thawed"}`
	expectedFrozenOutput = `{"return":"frozen"}`
	testDumpPath         = "/test/dump/path/vol1.memory.dump"
)

var _ = BeforeSuite(func() {
	tmpDir, err := ioutil.TempDir("", "cloudinittest")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(os.RemoveAll, tmpDir)

	Expect(cloudinit.SetLocalDirectory(tmpDir)).To(Succeed())

	ephemeraldiskutils.MockDefaultOwnershipManager()
	cloudinit.SetIsoCreationFunction(isoCreationFunc)
})

var _ = Describe("Manager", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var mockDirectIOChecker *converter.MockDirectIOChecker
	var ctrl *gomock.Controller
	var testVirtShareDir string
	var testEphemeralDiskDir string
	testVmName := "testvmi"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)
	ephemeralDiskCreatorMock := &fake.MockEphemeralDiskImageCreator{}

	BeforeEach(func() {
		testVirtShareDir = fmt.Sprintf("fake-virt-share-%d", GinkgoRandomSeed())
		testEphemeralDiskDir = fmt.Sprintf("fake-ephemeral-disk-%d", GinkgoRandomSeed())
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
		mockDomain.EXPECT().IsPersistent().AnyTimes().Return(true, nil)
		mockDomain.EXPECT().GetBlockInfo(gomock.Any(), gomock.Any()).AnyTimes().Return(&libvirt.DomainBlockInfo{Capacity: 0}, nil)
		mockDirectIOChecker = converter.NewMockDirectIOChecker(ctrl)
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(true, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(true, nil)
	})

	expectedDomainFor := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
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
			AllowEmulation:   true,
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

			domainSpec := expectedDomainFor(vmi)

			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should define and start a new VirtualMachineInstance with StartStrategy paused", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainSpec := expectedDomainFor(vmi)

			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_START_PAUSED).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
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
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
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
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should leave a defined and started VirtualMachineInstance alone", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		DescribeTable("should try to start a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				vmi := newVMI(testNamespace, testVmName)
				domainSpec := expectedDomainFor(vmi)
				xml, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).NotTo(HaveOccurred())

				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
				newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
				Expect(err).ToNot(HaveOccurred())
				Expect(newspec).ToNot(BeNil())
			},
			Entry("crashed", libvirt.DOMAIN_CRASHED),
			Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should unpause a paused VirtualMachineInstance on SyncVMI, which was not paused by user", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should not unpause a paused VirtualMachineInstance on SyncVMI, which was paused by user", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().Suspend().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			Expect(manager.PauseVMI(vmi)).To(Succeed())

			mockDomain.EXPECT().Free()
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			// no expected call to unpause

			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should freeze a VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-freeze"}`, testDomainName).Return("1", nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		})
		It("should unfreeze a VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-thaw"}`, testDomainName).Return("1", nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		})
		It("should automatically unfreeze after a timeout a frozen VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-freeze"}`, testDomainName).Return("1", nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-thaw"}`, testDomainName).Return("1", nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			var unfreezeTimeout time.Duration = 3 * time.Second
			Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
			// wait for the unfreeze timeout
			time.Sleep(unfreezeTimeout + 2*time.Second)
		})
		It("should freeze and unfreeze a VirtualMachineInstance without a trigger to the unfreeze timeout", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-freeze"}`, testDomainName).Return("1", nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GET_FSFREEZE_STATUS)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockConn.EXPECT().QemuAgentCommand(`{"execute":"guest-fsfreeze-thaw"}`, testDomainName).Return("1", nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			var unfreezeTimeout time.Duration = 3 * time.Second
			Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
			time.Sleep(time.Second)
			Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
			// wait for the unfreeze timeout
			time.Sleep(unfreezeTimeout + 2*time.Second)
		})
		It("should update domain with memory dump info when completed successfully", func() {
			isMemoryDumpCompletedSet := make(chan bool, 1)
			defer close(isMemoryDumpCompletedSet)

			mockDomain.EXPECT().Free().AnyTimes()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).DoAndReturn(
				func(_ libvirt.DomainXMLFlags) (string, error) {
					return domainToXml(domainSpec), nil
				})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).DoAndReturn(
				func(_, _, _ interface{}) (string, error) {
					return domainToMetadataXml(domainSpec), nil
				})
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(domainXml).To(ContainSubstring(filepath.Base(testDumpPath)))
				if domainSpec.Metadata.KubeVirt.MemoryDump == nil {
					domainSpec.Metadata.KubeVirt.MemoryDump = &api.MemoryDumpMetadata{}
				}
				now := metav1.Now()
				domainSpec.Metadata.KubeVirt.MemoryDump.FileName = filepath.Base(testDumpPath)
				domainSpec.Metadata.KubeVirt.MemoryDump.StartTimestamp = &now
				return mockDomain, nil
			})

			mockDomain.EXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(nil)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).DoAndReturn(
				func(_ libvirt.DomainXMLFlags) (string, error) {
					return domainToXml(domainSpec), nil
				})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).DoAndReturn(
				func(_, _, _ interface{}) (string, error) {
					return domainToMetadataXml(domainSpec), nil
				})
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(domainXml).To(ContainSubstring("endTimestamp"))
				Expect(domainXml).To(ContainSubstring("<completed>true</completed>"))
				Expect(domainXml).ToNot(ContainSubstring("failed"))
				isMemoryDumpCompletedSet <- true
				return mockDomain, nil
			})

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			err := manager.MemoryDump(vmi, testDumpPath)
			Expect(err).ToNot(HaveOccurred())
			// Expect extra call to memory dump not to impact
			errInProgress := manager.MemoryDump(vmi, testDumpPath)
			Expect(errInProgress).ToNot(HaveOccurred())
			Eventually(func() bool {
				select {
				case isSet := <-isMemoryDumpCompletedSet:
					return isSet
				default:
				}
				return false
			}, 20*time.Second, 2).Should(BeTrue(), "failed memory dump result wasn't set")
		})
		It("should update domain with memory dump info if memory dump failed", func() {
			isMemoryDumpCompletedSet := make(chan bool, 1)
			defer close(isMemoryDumpCompletedSet)

			mockDomain.EXPECT().Free().AnyTimes()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).DoAndReturn(
				func(_ libvirt.DomainXMLFlags) (string, error) {
					return domainToXml(domainSpec), nil
				})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).DoAndReturn(
				func(_, _, _ interface{}) (string, error) {
					return domainToMetadataXml(domainSpec), nil
				})
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(domainXml).To(ContainSubstring(filepath.Base(testDumpPath)))
				if domainSpec.Metadata.KubeVirt.MemoryDump == nil {
					domainSpec.Metadata.KubeVirt.MemoryDump = &api.MemoryDumpMetadata{}
				}
				now := metav1.Now()
				domainSpec.Metadata.KubeVirt.MemoryDump.FileName = filepath.Base(testDumpPath)
				domainSpec.Metadata.KubeVirt.MemoryDump.StartTimestamp = &now
				return mockDomain, nil
			})

			dumpFailure := fmt.Errorf("Memory dump failed!!")
			mockDomain.EXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(dumpFailure)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).DoAndReturn(
				func(_ libvirt.DomainXMLFlags) (string, error) {
					return domainToXml(domainSpec), nil
				})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).DoAndReturn(
				func(_, _, _ interface{}) (string, error) {
					return domainToMetadataXml(domainSpec), nil
				})
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				reason := fmt.Sprintf("<failureReason>%s: %s</failureReason>", failedDomainMemoryDump, dumpFailure)
				Expect(domainXml).To(ContainSubstring("endTimestamp"))
				Expect(domainXml).To(ContainSubstring("<completed>true</completed>"))
				Expect(domainXml).To(ContainSubstring("<failed>true</failed>"))
				Expect(domainXml).To(ContainSubstring(reason))
				isMemoryDumpCompletedSet <- true
				return mockDomain, nil
			})

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			err := manager.MemoryDump(vmi, testDumpPath)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				select {
				case isSet := <-isMemoryDumpCompletedSet:
					return isSet
				default:
				}
				return false
			}, 20*time.Second, 2).Should(BeTrue(), "failed memory dump result wasn't set")
		})
		It("should pause a VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().Suspend().Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			Expect(manager.PauseVMI(vmi)).To(Succeed())
		})
		It("should not try to pause a paused VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			// no call to suspend

			Expect(manager.PauseVMI(vmi)).To(Succeed())
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
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", "fake", nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			Expect(manager.UnpauseVMI(vmi)).To(Succeed())
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
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			// no call to unpause
			Expect(manager.UnpauseVMI(vmi)).To(Succeed())
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
							Bus: v1.DiskBusVirtio,
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
			domainSpec := expectedDomainFor(vmi)
			domainSpec.Devices.Disks = []api.Disk{
				{
					Device: "disk",
					Type:   "file",
					Source: api.DiskSource{
						File: "/var/run/kubevirt-private/vmi-disks/permvolume1/disk.img",
					},
					Target: api.DiskTarget{
						Bus:    v1.DiskBusVirtio,
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
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := newLibvirtDomainManager(mockConn, "fake", "fake", nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{
				VirtualMachineSMBios: &cmdv1.SMBios{},
				PreallocatedVolumes:  []string{"permvolume1"},
			})
			Expect(err).ToNot(HaveOccurred())
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
							Bus: v1.DiskBusVirtio,
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
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1.img"))
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
						Bus:    v1.DiskBusVirtio,
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
					File: "/var/run/kubevirt/hotplug-disks/hpvolume1.img",
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
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().AttachDeviceFlags(strings.ToLower(string(attachBytes)), affectLiveAndConfigLibvirtFlags)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			manager, _ := newLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
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
							Bus: v1.DiskBusVirtio,
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
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			detachDisk := api.Disk{
				Device: "disk",
				Type:   "file",
				Source: api.DiskSource{
					File: "/var/run/kubevirt/hotplug-disks/hpvolume1.img",
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
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().DetachDeviceFlags(strings.ToLower(string(detachBytes)), affectLiveAndConfigLibvirtFlags)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := newLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
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
							Bus: v1.DiskBusVirtio,
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
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1.img"))
				return true, nil
			}
			mockConn.EXPECT().DomainDefineXML(string(xmlDomain)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := newLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
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
							Bus: v1.DiskBusVirtio,
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
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal("/var/run/kubevirt/hotplug-disks/hpvolume1.img"))
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
						Bus:    v1.DiskBusVirtio,
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
			mockDomain.EXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			manager, _ := newLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
	})
	Context("test marking graceful shutdown", func() {
		It("Should set metadata when calling MarkGracefulShutdown api", func() {
			mockDomain.EXPECT().Free().AnyTimes()

			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)

			oldXML, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())

			t := true
			domainSpec.Metadata.KubeVirt.GracePeriod = &api.GracePeriodMetadata{MarkedForGracefulShutdown: &t}

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
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			manager.MarkGracefulShutdownVMI(vmi)
		})

		It("Should signal graceful shutdown after marked for shutdown", func() {
			mockDomain.EXPECT().Free().AnyTimes()

			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)

			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())

			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(`<kubevirt><graceperiod><deletionGracePeriodSeconds>3600</deletionGracePeriodSeconds><deletionTimestamp>2021-03-11T09:08:20.144606353Z</deletionTimestamp><markedForGracefulShutdown>true</markedForGracefulShutdown></graceperiod></kubevirt>`, nil)

			mockDomain.EXPECT().ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_ACPI_POWER_BTN).Times(1).Return(nil)
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			manager.SignalShutdownVMI(vmi)
		})
	})
	Context("test migration monitor", func() {
		It("migration should be canceled if it's not progressing", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := &libvirt.DomainJobInfo{
				Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
				DataRemaining:    32479827394,
				DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
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
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
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
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().DoAndReturn(func(_ libvirt.DomainXMLFlags) (string, error) {
				xmlOriginal, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).ToNot(HaveOccurred())
				return string(xmlOriginal), nil
			})
			mockDomain.EXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(1).Return(nil)
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				DoAndReturn(func(_ libvirt.DomainMetadataType, _ string, _ libvirt.DomainModificationImpact) (string, error) {
					metadata, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
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

		It("migration should switch to PostCopy eventually", func() {
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
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(migrationData),
					DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Times(2).Return(mockDomain, nil)

			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().DoAndReturn(func(_ libvirt.DomainXMLFlags) (string, error) {
				xmlOriginal, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).ToNot(HaveOccurred())
				return string(xmlOriginal), nil
			})

			counter := 0
			mockDomain.EXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(2).DoAndReturn(func(flag uint32) error {
				if counter == 0 {
					counter += 1
					return libvirt.Error{

						Code:    1,
						Domain:  1,
						Message: "internal error: unable to execute QEMU command 'migrate-start-postcopy': Postcopy must be started after migration has been started",
						Level:   libvirt.ERR_ERROR,
					}
				}
				return nil
			})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				DoAndReturn(func(_ libvirt.DomainMetadataType, _ string, _ libvirt.DomainModificationImpact) (string, error) {
					metadata, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
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
		// This is incomplete as it is not verifying that we abort. Previously it wasn't even testing anything at all
		It("migration should be canceled when requested", func() {
			migrationUid := types.UID("111222333")

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()

			now := metav1.Now()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   migrationUid,
				StartTimestamp: &now,
			}

			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				StartTimestamp: &now,
				UID:            migrationUid,
			}

			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			// expect domainspecwithruntimeinfo todo this can be useful for other tests
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DomainXMLFlags(0))).AnyTimes().DoAndReturn(
				func(_ libvirt.DomainXMLFlags) (string, error) {
					return domainToXml(domainSpec), nil
				})
			mockDomain.EXPECT().GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				DoAndReturn(func(_, _, _ interface{}) (string, error) {
					return domainToMetadataXml(domainSpec), nil
				})

			mockConn.EXPECT().DomainDefineXML(gomock.Any()).Times(1).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(domainXml).To(ContainSubstring(string(v1.MigrationAbortInProgress)))
				return mockDomain, nil
			})

			// These lines do not test anything but needs to be here because otherwise test will panic
			mockDomain.EXPECT().AbortJob().MaxTimes(1)
			migrationInProgress := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(32479827394),
					DataRemainingSet: true,
				}
			}()
			mockDomain.EXPECT().GetJobInfo().MaxTimes(1).Return(migrationInProgress, nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())

		})

		It("shouldn't be able to call cancel migration more than once", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()

			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			secondBefore := metav1.Time{Time: now.Add(-time.Second)}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "111222333",
				StartTimestamp: &now,
			}

			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID:            vmi.Status.MigrationState.MigrationUID,
				AbortStatus:    string(v1.MigrationAbortInProgress),
				StartTimestamp: &secondBefore,
			}

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).AnyTimes().Return(string(domainXml), nil)

			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())
		})
		It("migration cancellation should be finilized even if we missed status update", func() {
			isMigrationAbortSet := make(chan bool, 1)
			defer close(isMigrationAbortSet)

			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_NONE,
					DataRemaining: uint64(0),
				}
			}()
			fake_jobinfo_running := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(32479827777),
					DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{

				UID:         vmi.Status.MigrationState.MigrationUID,
				AbortStatus: string(v1.MigrationAbortInProgress),
			}

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(domainXml), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			gomock.InOrder(
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo_running, nil),
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo, nil),
			)
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(strings.Contains(domainXml, string(v1.MigrationAbortSucceeded))).To(BeTrue())
				isMigrationAbortSet <- true
				return mockDomain, nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
			Eventually(func() bool {
				select {
				case isSet := <-isMigrationAbortSet:
					return isSet
				default:
				}
				return false
			}, 20*time.Second, 2).Should(BeTrue(), "migration cancelled result wasn't set")
		})
		It("migration failure should be finalized even if we missed status update", func() {
			isMigrationFailedSet := make(chan bool, 1)
			defer close(isMigrationFailedSet)

			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:          libvirt.DOMAIN_JOB_NONE,
					DataRemaining: uint64(0),
				}
			}()
			fake_jobinfo_running := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(32479827777),
					DataRemainingSet: true,
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

			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{
				UID: vmi.Status.MigrationState.MigrationUID,
			}

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			metadataXml, err := xml.MarshalIndent(domainSpec.Metadata.KubeVirt, "", "\t")
			Expect(err).NotTo(HaveOccurred())
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(domainXml), nil)
			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return(string(metadataXml), nil)
			manager := &LibvirtDomainManager{
				virConn:      mockConn,
				virtShareDir: testVirtShareDir,
			}
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			gomock.InOrder(
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo_running, nil),
				mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo, nil),
			)
			mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(domainXml string) (cli.VirDomain, error) {
				Expect(strings.Contains(domainXml, "<failed>true</failed>")).To(BeTrue())
				isMigrationFailedSet <- true
				return mockDomain, nil
			})

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
			Eventually(func() bool {
				select {
				case isSet := <-isMigrationFailedSet:
					return isSet
				default:
				}
				return false
			}, 20*time.Second, 2).Should(BeTrue(), "migration failed result wasn't set")
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
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
				TargetPod:    "fakepod",
			}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			Expect(manager.PrepareMigrationTarget(vmi, true, &cmdv1.VirtualMachineOptions{})).To(Succeed())
		})
		It("should verify that migration failure is set in the monitor thread", func() {
			isMigrationFailedSet := make(chan bool, 1)

			defer close(isMigrationFailedSet)

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free().AnyTimes()
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_NONE,
					DataRemaining:    uint64(32479827394),
					DataRemainingSet: true,
				}
			}()

			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockDomain.EXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
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
			Expect(manager.MigrateVMI(vmi, options)).To(Succeed())
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

			domainSpec := expectedDomainFor(vmi)
			domainSpec.Metadata.KubeVirt.Migration = &api.MigrationMetadata{

				UID: vmi.Status.MigrationState.MigrationUID,
			}

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())

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
			Expect(manager.MigrateVMI(vmi, options)).To(Succeed())
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
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
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
		DescribeTable("should try to undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().UndefineFlags(libvirt.DOMAIN_UNDEFINE_NVRAM).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, "fake", "fake", nil, "/usr/share/", ephemeralDiskCreatorMock)
				Expect(manager.DeleteVMI(newVMI(testNamespace, testVmName))).To(Succeed())
			},
			Entry("crashed", libvirt.DOMAIN_CRASHED),
			Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
		)
		DescribeTable("should try to destroy a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockDomain.EXPECT().DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
				Expect(manager.KillVMI(newVMI(testNamespace, testVmName))).To(Succeed())
			},
			Entry("shuttingDown", libvirt.DOMAIN_SHUTDOWN),
			Entry("running", libvirt.DOMAIN_RUNNING),
			Entry("paused", libvirt.DOMAIN_PAUSED),
		)
	})
	DescribeTable("check migration flags",
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
		Entry("with block migration", "block"),
		Entry("without block migration", "live"),
		Entry("unsafe migration", "unsafe"),
		Entry("migration auto converge", "autoConverge"),
		Entry("migration using postcopy", "postCopy"),
		Entry("migration of paused vmi", "paused"),
	)

	DescribeTable("on successful list all domains",
		func(state libvirt.DomainState, kubevirtState api.LifeCycle, libvirtReason int, kubevirtReason api.StateChangeReason) {

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			mockDomain.EXPECT().GetState().Return(state, libvirtReason, nil).AnyTimes()
			mockDomain.EXPECT().GetName().Return("test", nil)
			x, err := xml.MarshalIndent(api.NewMinimalDomainSpec("test"), "", "\t")
			Expect(err).ToNot(HaveOccurred())

			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(x), nil)
			mockConn.EXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]cli.VirDomain{mockDomain}, nil)

			mockDomain.EXPECT().
				GetMetadata(libvirt.DOMAIN_METADATA_ELEMENT, "http://kubevirt.io", libvirt.DOMAIN_AFFECT_CONFIG).
				AnyTimes().
				Return("<kubevirt></kubevirt>", nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			doms, err := manager.ListAllDomains()
			Expect(err).NotTo(HaveOccurred())
			Expect(doms).To(HaveLen(1))

			domain := doms[0]
			domain.Spec.XMLName = xml.Name{}

			Expect(&domain.Spec).To(Equal(api.NewMinimalDomainSpec("test")))
			Expect(domain.Status.Status).To(Equal(kubevirtState))
			Expect(domain.Status.Reason).To(Equal(kubevirtReason))
		},
		Entry("crashed", libvirt.DOMAIN_CRASHED, api.Crashed, int(libvirt.DOMAIN_CRASHED_UNKNOWN), api.ReasonUnknown),
		Entry("shutoff", libvirt.DOMAIN_SHUTOFF, api.Shutoff, int(libvirt.DOMAIN_SHUTOFF_DESTROYED), api.ReasonDestroyed),
		Entry("shutdown", libvirt.DOMAIN_SHUTDOWN, api.Shutdown, int(libvirt.DOMAIN_SHUTDOWN_USER), api.ReasonUser),
		Entry("unknown", libvirt.DOMAIN_NOSTATE, api.NoState, int(libvirt.DOMAIN_NOSTATE_UNKNOWN), api.ReasonUnknown),
		Entry("running", libvirt.DOMAIN_RUNNING, api.Running, int(libvirt.DOMAIN_RUNNING_UNKNOWN), api.ReasonUnknown),
		Entry("paused", libvirt.DOMAIN_PAUSED, api.Paused, int(libvirt.DOMAIN_PAUSED_STARTING_UP), api.ReasonPausedStartingUp),
	)

	Context("on successful GetAllDomainStats", func() {
		It("should return content", func() {
			fake_jobinfo := stats.DomainJobInfo{}
			mockConn.EXPECT().GetDomainStats(
				gomock.Eq(libvirt.DOMAIN_STATS_BALLOON|libvirt.DOMAIN_STATS_CPU_TOTAL|libvirt.DOMAIN_STATS_VCPU|libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BLOCK|libvirt.DOMAIN_STATS_DIRTYRATE),
				&fake_jobinfo,
				gomock.Eq(libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING),
			).Return([]*stats.DomainStats{
				{},
			}, nil)

			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			domStats, err := manager.GetDomainStats()

			Expect(err).ToNot(HaveOccurred())
			Expect(domStats).To(HaveLen(1))
		})
	})

	Context("on failed GetDomainSpecWithRuntimeInfo", func() {
		It("should fall back to returning domain spec without runtime info", func() {
			manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			vmi := newVMI(testNamespace, testVmName)

			domainSpec := expectedDomainFor(vmi)

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

		Context("on call to GetGuestOSInfo", func() {
			var libvirtmanager DomainManager
			var agentStore agentpoller.AsyncAgentStore

			BeforeEach(func() {
				agentStore = agentpoller.NewAsyncAgentStore()
				libvirtmanager, _ = NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			})

			It("should report nil when no OS info exists in the cache", func() {
				Expect(libvirtmanager.GetGuestOSInfo()).To(BeNil())
			})

			It("should report OS info when it exists in the cache", func() {
				fakeInfo := api.GuestOSInfo{
					Name: "TestGuestOSName",
				}
				agentStore.Store(agentpoller.GET_OSINFO, fakeInfo)

				osInfo := libvirtmanager.GetGuestOSInfo()
				Expect(*osInfo).To(Equal(fakeInfo))
			})
		})

		Context("on call to InterfacesStatus", func() {
			var libvirtmanager DomainManager
			var agentStore agentpoller.AsyncAgentStore

			BeforeEach(func() {
				agentStore = agentpoller.NewAsyncAgentStore()
				libvirtmanager, _ = NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)
			})

			It("should return nil when no interfaces exists in the cache", func() {
				Expect(libvirtmanager.InterfacesStatus()).To(BeNil())
			})

			It("should report interfaces info when interfaces exists", func() {
				fakeInterfaces := []api.InterfaceStatus{{
					InterfaceName: "eth1",
					Mac:           "00:00:00:00:00:01",
				}}
				agentStore.Store(agentpoller.GET_INTERFACES, fakeInterfaces)
				interfacesStatus := agentStore.GetInterfaceStatus()

				Expect(interfacesStatus).To(Equal(fakeInterfaces))
			})
		})
	})

	It("executes hotPlugHostDevices", func() {
		os.Setenv("KUBEVIRT_RESOURCE_NAME_test1", "127.0.0.1")
		os.Setenv("PCIDEVICE_127_0_0_1", "05EA:Fc:1d.6")

		defer os.Unsetenv("KUBEVIRT_RESOURCE_NAME_test1")
		defer os.Unsetenv("PCIDEVICE_127_0_0_1")

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		vmi := newVMI(testNamespace, testVmName)
		vmi.Spec.Domain.Devices.Interfaces = append(
			vmi.Spec.Domain.Devices.Interfaces,
			v1.Interface{
				Name: "test1",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
				MacAddress: "de:ad:00:00:be:af",
			},
		)
		vmi.Spec.Networks = append(
			vmi.Spec.Networks,
			v1.Network{Name: "test1",
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: "test1"},
				}},
		)

		domainSpec := expectedDomainFor(vmi)
		xml, err := xml.MarshalIndent(domainSpec, "", "\t")
		Expect(err).NotTo(HaveOccurred())

		mockDomain.EXPECT().Free()
		mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
		mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
		mockDomain.EXPECT().AttachDeviceFlags(`<hostdev type="pci" managed="no"><source><address type="pci" domain="0x05EA" bus="0xFc" slot="0x1d" function="0x6"></address></source><alias name="ua-sriov-test1"></alias></hostdev>`, libvirt.DomainDeviceModifyFlags(3)).Return(nil)

		Expect(libvirtmanager.hotPlugHostDevices(vmi)).To(Succeed())
	})

	It("executes GetGuestInfo", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GET_USERS, []api.User{
			{
				Name:      "test",
				Domain:    "test",
				LoginTime: 0,
			},
		})
		agentStore.Store(agentpoller.GET_FILESYSTEM, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		_, err := libvirtmanager.GetGuestInfo()
		Expect(err).ToNot(HaveOccurred())
	})

	It("executes GetUsers", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GET_USERS, []api.User{
			{
				Name:      "test",
				Domain:    "test",
				LoginTime: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		virtualMachineInstanceGuestAgentInfo, err := libvirtmanager.GetUsers()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtualMachineInstanceGuestAgentInfo).ToNot(BeEmpty())
	})

	It("executes GetFilesystems", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GET_FILESYSTEM, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		virtualMachineInstanceGuestAgentInfo, err := libvirtmanager.GetFilesystems()
		Expect(err).ToNot(HaveOccurred())
		Expect(virtualMachineInstanceGuestAgentInfo).ToNot(BeEmpty())
	})

	It("executes generateCloudInitEmptyISO and succeeds", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GET_FILESYSTEM, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		vmi := newVMI(testNamespace, testVmName)
		vmi.Status.VolumeStatus = make([]v1.VolumeStatus, 1)
		vmi.Status.VolumeStatus[0] = v1.VolumeStatus{
			Name: "test1",
			Size: 42,
		}

		userData := "fake\nuser\ndata\n"
		networkData := "FakeNetwork"
		addCloudInitDisk(vmi, userData, networkData)
		libvirtmanager.cloudInitDataStore = &cloudinit.CloudInitData{
			DataSource: cloudinit.DataSourceNoCloud,
			VolumeName: "test1",
		}

		Expect(libvirtmanager.generateCloudInitEmptyISO(vmi, nil)).To(Succeed())

		isoPath := cloudinit.GetIsoFilePath(libvirtmanager.cloudInitDataStore.DataSource, vmi.Name, vmi.Namespace)
		stats, err := os.Stat(isoPath)
		Expect(err).ToNot(HaveOccurred())
		Expect(stats.Size()).To(Equal(int64(42)))
	})

	It("executes generateCloudInitEmptyISO and fails", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GET_FILESYSTEM, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockConn, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		vmi := newVMI(testNamespace, testVmName)
		vmi.Status.VolumeStatus = make([]v1.VolumeStatus, 1)

		userData := "fake\nuser\ndata\n"
		networkData := "FakeNetwork"
		addCloudInitDisk(vmi, userData, networkData)
		libvirtmanager.cloudInitDataStore = &cloudinit.CloudInitData{
			DataSource: cloudinit.DataSourceNoCloud,
			VolumeName: "test1",
		}

		err := libvirtmanager.generateCloudInitEmptyISO(vmi, nil)
		Expect(err).To(MatchError(ContainSubstring("failed to find the status of volume test1")))
	})

	// TODO: test error reporting on non successful VirtualMachineInstance syncs and kill attempts
})

var _ = Describe("getAttachedDisks", func() {
	DescribeTable("should return the correct values", func(oldDisks, newDisks, expected []api.Disk) {
		res := getAttachedDisks(oldDisks, newDisks)
		Expect(res).To(Equal(expected))
	},
		Entry("be empty with empty old and new",
			[]api.Disk{},
			[]api.Disk{},
			[]api.Disk{}),
		Entry("be empty with empty old and new being identical",
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
		Entry("contain a new disk with empty having a new disk compared to old",
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
	DescribeTable("should return the correct values", func(oldDisks, newDisks, expected []api.Disk) {
		res := getDetachedDisks(oldDisks, newDisks)
		Expect(res).To(Equal(expected))
	},
		Entry("be empty with empty old and new",
			[]api.Disk{},
			[]api.Disk{},
			[]api.Disk{}),
		Entry("be empty with empty old and new being identical",
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
		Entry("contains something if new has less than old",
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
	It("should remove only the kubevirt migration metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <metadata>
         <kubevirt>
            <migration>nested</migration>
         </kubevirt>
      </metadata>
      <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
      <migration>
         <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
         <failed>false</failed>
      </migration>
      <graceperiod>
        <deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
    <othermetadata>
      <kubevirt>
         <migration>42</migration>
      </kubevirt>
    </othermetadata>
  </metadata>
  <kubevirt><migration>this should stay</migration></kubevirt>
</domain>`
		// migratableDomXML() removes the migration block but not its ident, which is its own token, hence the blank line below
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <metadata>
         <kubevirt>
            <migration>nested</migration>
         </kubevirt>
      </metadata>
      <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
      
      <graceperiod>
        <deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
    <othermetadata>
      <kubevirt>
         <migration>42</migration>
      </kubevirt>
    </othermetadata>
  </metadata>
  <kubevirt><migration>this should stay</migration></kubevirt>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		mockDomain.EXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domain := &api.Domain{}
		Expect(xml.Unmarshal([]byte(domXML), domain)).To(Succeed())
		newXML, err := migratableDomXML(mockDomain, vmi, &domain.Spec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
	It("should change CPU pinning according to migration metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
      <migration>
         <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
         <failed>false</failed>
      </migration>
      <graceperiod>
        <deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="4"></vcpupin>
    <vcpupin vcpu="1" cpuset="5"></vcpupin>
  </cputune>
</domain>`
		// migratableDomXML() removes the migration block but not its ident, which is its own token, hence the blank line below
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>d38cac9c-435b-42d5-960e-06e8d41146e8</uid>
      
      <graceperiod>
        <deletionGracePeriodSeconds>0</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="6"></vcpupin>
    <vcpupin vcpu="1" cpuset="7"></vcpupin>
  </cputune>
</domain>`

		By("creating a VMI with dedicated CPU cores")
		vmi := newVMI("testns", "kubevirt")
		vmi.Spec.Domain.CPU = &v1.CPU{
			Cores:                 2,
			DedicatedCPUPlacement: true,
		}

		By("making up a target topology")
		topology := &cmdv1.Topology{NumaCells: []*cmdv1.Cell{{
			Id: 0,
			Cpus: []*cmdv1.CPU{
				{
					Id:       6,
					Siblings: []uint32{6},
				},
				{
					Id:       7,
					Siblings: []uint32{7},
				},
			},
		}}}
		targetNodeTopology, err := json.Marshal(topology)
		Expect(err).NotTo(HaveOccurred(), "failed to marshall the topology")

		By("saving that topology in the migration state of the VMI")
		vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
			TargetCPUSet:       []int{6, 7},
			TargetNodeTopology: string(targetNodeTopology),
		}

		By("generated the domain XML for a migration to that target")
		mockDomain.EXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domain := &api.Domain{}
		Expect(xml.Unmarshal([]byte(domXML), domain)).To(Succeed())
		newXML, err := migratableDomXML(mockDomain, vmi, &domain.Spec)
		Expect(err).ToNot(HaveOccurred(), "failed to generate target domain XML")

		By("ensuring the generated XML is accurate")
		Expect(newXML).To(Equal(expectedXML), "the target XML is not as expected")
	})
})

var _ = Describe("Manager helper functions", func() {

	Context("getVMIEphemeralDisksTotalSize", func() {

		var tmpDir string
		var zeroQuantity resource.Quantity

		BeforeEach(func() {
			var err error
			tmpDir, err = os.MkdirTemp("", "tempdir")
			Expect(err).ToNot(HaveOccurred())

			zeroQuantity = *resource.NewScaledQuantity(0, 0)
		})

		AfterEach(func() {
			_ = os.RemoveAll(tmpDir)
		})

		expectNonZeroQuantity := func(ephemeralDiskDir string) {
			By("Expecting quantity larger than zero")
			quantity := getVMIEphemeralDisksTotalSize(ephemeralDiskDir)

			Expect(quantity).ToNot(BeNil())
			Expect(quantity).ToNot(Equal(zeroQuantity))
			quantityValue := quantity.Value()
			Expect(quantityValue).To(BeNumerically(">", 0))
		}

		expectZeroQuantity := func(ephemeralDiskDir string) {
			By("Expecting zero quantity")
			quantity := getVMIEphemeralDisksTotalSize(ephemeralDiskDir)

			Expect(quantity).ToNot(BeNil())
			Expect(*quantity).To(Equal(zeroQuantity))
			quantityValue := quantity.Value()
			Expect(quantityValue).To(BeNumerically("==", 0))
		}

		It("successful run with non-zero size", func() {
			By("Creating a file with non-zero size")
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "testfile"), []byte("file contents"), 0666)).To(Succeed())

			expectNonZeroQuantity(tmpDir)
		})

		It("successful run with zero size", func() {
			By("Creating a file with non-zero size")
			Expect(ioutil.WriteFile(filepath.Join(tmpDir, "testfile"), []byte("file contents"), 0666)).To(Succeed())

			expectNonZeroQuantity(tmpDir)
		})

		It("expect zero quantity when path does not exist", func() {
			expectZeroQuantity("path_that_doesnt_exist")
		})

		It("expect zero quantity in an empty directory", func() {
			expectZeroQuantity(tmpDir)
		})

	})

	Context("possibleGuestSize", func() {

		var properDisk api.Disk
		var fakePercentFloat float64

		BeforeEach(func() {
			fakePercentFloat = 0.7648
			fakePercent := cdiv1beta1.Percent(fmt.Sprint(fakePercentFloat))
			fakeCapacity := int64(2345 * 3456) // We need (1-0.7648)*fakeCapacity to be > 1MiB and misaligned

			properDisk = api.Disk{
				FilesystemOverhead: &fakePercent,
				Capacity:           &fakeCapacity,
			}
		})

		It("should return correct value", func() {
			size, ok := possibleGuestSize(properDisk)
			Expect(ok).To(BeTrue())
			capacity := properDisk.Capacity
			Expect(capacity).ToNot(BeNil())

			expectedSize := int64((1 - fakePercentFloat) * float64(*capacity))
			// The size is expected to be 1MiB-aligned
			expectedSize = expectedSize - expectedSize%(1024*1024)

			Expect(size).To(Equal(expectedSize))
		})

		DescribeTable("should return error when", func(createDisk func() api.Disk) {

		},
			Entry("disk capacity is nil", func() api.Disk {
				disk := properDisk
				disk.Capacity = nil
				return disk
			}),
			Entry("filesystem overhead is nil", func() api.Disk {
				disk := properDisk
				disk.FilesystemOverhead = nil
				return disk
			}),
			Entry("filesystem overhead is non-float", func() api.Disk {
				disk := properDisk
				fakePercent := cdiv1beta1.Percent(fmt.Sprint("abcdefg"))
				disk.FilesystemOverhead = &fakePercent
				return disk
			}),
		)

	})

})

func newVMI(namespace, name string) *v1.VirtualMachineInstance {
	vmi := api2.NewMinimalVMIWithNS(namespace, name)
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
				Bus: v1.DiskBusVirtio,
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

func isoCreationFunc(isoOutFile, volumeID string, inDir string) error {
	_, err := os.Create(isoOutFile)
	return err
}

func domainToXml(domain *api.DomainSpec) string {
	xml, err := xml.MarshalIndent(domain, "", "\t")
	Expect(err).ToNot(HaveOccurred())
	return string(xml)
}

func domainToMetadataXml(domain *api.DomainSpec) string {
	xml, err := xml.MarshalIndent(domain.Metadata.KubeVirt, "", "\t")
	Expect(err).ToNot(HaveOccurred())
	return string(xml)
}
