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

package virtwrap

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"libvirt.org/go/libvirt"
	"libvirt.org/go/libvirtxml"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	api2 "kubevirt.io/client-go/api"

	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	ephemeraldiskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	hostdisk "kubevirt.io/kubevirt/pkg/host-disk"
	"kubevirt.io/kubevirt/pkg/liveupdate/memory"
	virtpointer "kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/util/net/ip"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/metadata"
	agentpoller "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/agent-poller"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/efi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/testing"
)

var (
	expectedThawedOutput = `{"return":"thawed"}`
	expectedFrozenOutput = `{"return":"frozen"}`
	testDumpPath         = "/test/dump/path/vol1.memory.dump"
	clusterConfig        *virtconfig.ClusterConfig

	//go:embed testdata/migration_domain.xml
	embedMigrationDomain string

	fakeCpuSetGetter = func() ([]int, error) {
		return []int{0, 1, 2, 3, 4}, nil
	}
)

var _ = BeforeSuite(func() {
	tmpDir, err := os.MkdirTemp("", "cloudinittest")
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(os.RemoveAll, tmpDir)

	Expect(cloudinit.SetLocalDirectory(tmpDir)).To(Succeed())

	ephemeraldiskutils.MockDefaultOwnershipManager()
	cloudinit.SetIsoCreationFunction(isoCreationFunc)
})

var _ = Describe("Manager", func() {
	var mockLibvirt *testing.Libvirt
	var mockDirectIOChecker *converter.MockDirectIOChecker
	var ctrl *gomock.Controller
	var testVirtShareDir string
	var testEphemeralDiskDir string
	var metadataCache *metadata.Cache
	var topology *cmdv1.Topology
	testVmName := "testvmi"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)
	ephemeralDiskCreatorMock := &fake.MockEphemeralDiskImageCreator{}
	newLibvirtDomainManagerDefault := func() (DomainManager, error) {
		return NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
	}

	BeforeEach(func() {
		testVirtShareDir = fmt.Sprintf("fake-virt-share-%d", GinkgoRandomSeed())
		testEphemeralDiskDir = fmt.Sprintf("fake-ephemeral-disk-%d", GinkgoRandomSeed())
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
		metadataCache = metadata.NewCache()
		mockLibvirt.DomainEXPECT().GetBlockInfo(gomock.Any(), gomock.Any()).AnyTimes().Return(&libvirt.DomainBlockInfo{Capacity: 0}, nil)
		mockDirectIOChecker = converter.NewMockDirectIOChecker(ctrl)
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(true, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(true, nil)
		topology = &cmdv1.Topology{
			NumaCells: []*cmdv1.Cell{
				{
					Id: uint32(0),
					Memory: &cmdv1.Memory{
						Amount: 1289144,
						Unit:   "KiB",
					},
					Pages: []*cmdv1.Pages{
						{
							Count: 314094,
							Unit:  "KiB",
							Size:  4,
						},
						{
							Count: 16,
							Unit:  "KiB",
							Size:  2048,
						},
						{
							Count: 0,
							Unit:  "KiB",
							Size:  1048576,
						},
					},
					Distances: []*cmdv1.Sibling{
						{
							Id:    0,
							Value: 10,
						},
						{
							Id:    1,
							Value: 10,
						},
						{
							Id:    2,
							Value: 10,
						},
						{
							Id:    3,
							Value: 10,
						},
					},
					Cpus: []*cmdv1.CPU{
						{
							Id:       0,
							Siblings: []uint32{0},
						},
						{
							Id:       1,
							Siblings: []uint32{1},
						},
						{
							Id:       2,
							Siblings: []uint32{2},
						},
						{
							Id:       3,
							Siblings: []uint32{3},
						},
						{
							Id:       4,
							Siblings: []uint32{4},
						},
						{
							Id:       5,
							Siblings: []uint32{5},
						},
					},
				},
				{
					Id: uint32(2),
					Memory: &cmdv1.Memory{
						Amount: 1223960,
						Unit:   "KiB",
					},
					Pages: []*cmdv1.Pages{
						{
							Count: 297798,
							Unit:  "KiB",
							Size:  4,
						},
						{
							Count: 16,
							Unit:  "KiB",
							Size:  2048,
						},
						{
							Count: 0,
							Unit:  "KiB",
							Size:  1048576,
						},
					},
					Distances: []*cmdv1.Sibling{
						{
							Id:    0,
							Value: 10,
						},
						{
							Id:    1,
							Value: 10,
						},
						{
							Id:    2,
							Value: 10,
						},
						{
							Id:    3,
							Value: 10,
						},
					},
					Cpus: []*cmdv1.CPU{
						{
							Id:       0,
							Siblings: []uint32{0},
						},
						{
							Id:       1,
							Siblings: []uint32{1},
						},
						{
							Id:       2,
							Siblings: []uint32{2},
						},
						{
							Id:       3,
							Siblings: []uint32{3},
						},
						{
							Id:       4,
							Siblings: []uint32{4},
						},
						{
							Id:       5,
							Siblings: []uint32{5},
						},
					},
				},
				{
					Id: uint32(3),
					Memory: &cmdv1.Memory{
						Amount: 1251752,
						Unit:   "KiB",
					},
					Pages: []*cmdv1.Pages{
						{
							Count: 304746,
							Unit:  "KiB",
							Size:  4,
						},
						{
							Count: 16,
							Unit:  "KiB",
							Size:  2048,
						},
						{
							Count: 0,
							Unit:  "KiB",
							Size:  1048576,
						},
					},
					Distances: []*cmdv1.Sibling{
						{
							Id:    0,
							Value: 10,
						},
						{
							Id:    1,
							Value: 10,
						},
						{
							Id:    2,
							Value: 10,
						},
						{
							Id:    3,
							Value: 10,
						},
					},
					Cpus: []*cmdv1.CPU{
						{
							Id:       0,
							Siblings: []uint32{0},
						},
						{
							Id:       1,
							Siblings: []uint32{1},
						},
						{
							Id:       2,
							Siblings: []uint32{2},
						},
						{
							Id:       3,
							Siblings: []uint32{3},
						},
						{
							Id:       4,
							Siblings: []uint32{4},
						},
						{
							Id:       5,
							Siblings: []uint32{5},
						},
					},
				},
				{
					Id: uint32(4),
					Memory: &cmdv1.Memory{
						Amount: 1289404,
						Unit:   "KiB",
					},
					Pages: []*cmdv1.Pages{
						{
							Count: 314159,
							Unit:  "KiB",
							Size:  4,
						},
						{
							Count: 16,
							Unit:  "KiB",
							Size:  2048,
						},
						{
							Count: 0,
							Unit:  "KiB",
							Size:  1048576,
						},
					},
					Distances: []*cmdv1.Sibling{
						{
							Id:    0,
							Value: 10,
						},
						{
							Id:    1,
							Value: 10,
						},
						{
							Id:    2,
							Value: 10,
						},
						{
							Id:    3,
							Value: 10,
						},
					},
					Cpus: []*cmdv1.CPU{
						{
							Id:       0,
							Siblings: []uint32{0},
						},
						{
							Id:       1,
							Siblings: []uint32{1},
						},
						{
							Id:       2,
							Siblings: []uint32{2},
						},
						{
							Id:       3,
							Siblings: []uint32{3},
						},
						{
							Id:       4,
							Siblings: []uint32{4},
						},
						{
							Id:       5,
							Siblings: []uint32{5},
						},
					},
				},
			},
		}
		clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{})
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

		freePageReportingDisabled := clusterConfig.IsFreePageReportingDisabled()
		serialConsoleLogDisabled := clusterConfig.IsSerialConsoleLogDisabled()

		c := &converter.ConverterContext{
			Architecture:      arch.NewConverter(runtime.GOARCH),
			VirtualMachine:    vmi,
			AllowEmulation:    true,
			SMBios:            &cmdv1.SMBios{},
			HotplugVolumes:    hotplugVolumes,
			PermanentVolumes:  permanentVolumes,
			FreePageReporting: isFreePageReportingEnabled(freePageReportingDisabled, vmi),
			SerialConsoleLog:  isSerialConsoleLogEnabled(serialConsoleLogDisabled, vmi),
			CPUSet:            []int{0, 1, 2, 3, 4, 5},
			Topology:          topology,
		}
		Expect(converter.Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.NewDefaulter(runtime.GOARCH).SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	mockDomainWithFreeExpectation := func(_ string) (cli.VirDomain, error) {
		// Make sure that we always free the domain after use
		mockLibvirt.DomainEXPECT().Free()
		return mockLibvirt.VirtDomain, nil
	}

	Context("on successful VirtualMachineInstance sync", func() {

		addPlaceHolderInterfaces := func(vmi *v1.VirtualMachineInstance, domainSpec *api.DomainSpec) *api.DomainSpec {
			count, err := calculateHotplugPortCount(vmi, domainSpec)
			Expect(err).ToNot(HaveOccurred())
			return appendPlaceholderInterfacesToTheDomain(vmi, domainSpec, count)
		}

		setDomainExpectations := func(vmi *v1.VirtualMachineInstance) {
			domainSpec := expectedDomainFor(vmi)
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXML, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXML)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXMLWithInterfaces)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(domainXML), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
		}

		It("should define and start a new VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			setDomainExpectations(vmi)

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should define and start a new VirtualMachineInstance with StartStrategy paused", func() {
			vmi := newVMI(testNamespace, testVmName)
			strategy := v1.StartStrategyPaused
			vmi.Spec.StartStrategy = &strategy
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			setDomainExpectations(vmi)

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_START_PAUSED).Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should define and start a new VirtualMachineInstance with userData", func() {
			vmi := newVMI(testNamespace, testVmName)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			userData := "fake\nuser\ndata\n"
			networkData := ""
			addCloudInitDisk(vmi, userData, networkData)

			setDomainExpectations(vmi)

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should define and start a new VirtualMachineInstance with userData and networkData", func() {
			vmi := newVMI(testNamespace, testVmName)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)

			setDomainExpectations(vmi)

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should leave a defined and started VirtualMachineInstance alone", func() {
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		DescribeTable("should try to start a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				vmi := newVMI(testNamespace, testVmName)
				domainSpec := expectedDomainFor(vmi)
				xml, err := xml.MarshalIndent(domainSpec, "", "\t")
				Expect(err).NotTo(HaveOccurred())

				mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
				mockLibvirt.DomainEXPECT().GetState().Return(state, 1, nil)
				mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xml), nil)
				manager, _ := newLibvirtDomainManagerDefault()
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
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockLibvirt.DomainEXPECT().Resume().Return(nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should not unpause a paused VirtualMachineInstance on SyncVMI, which was paused by user", func() {
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectedDomainFor(vmi)
			xml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).NotTo(HaveOccurred())

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().Suspend().Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()

			Expect(manager.PauseVMI(vmi)).To(Succeed())

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			// no expected call to unpause

			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should freeze a VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil).Times(1)
			mockLibvirt.DomainEXPECT().Free().Times(1)
			mockLibvirt.DomainEXPECT().FSFreeze(nil, uint32(0)).Times(1)

			manager, _ := newLibvirtDomainManagerDefault()

			Expect(manager.FreezeVMI(vmi, 0)).To(Succeed())
		})
		It("should fail freeze a VirtualMachineInstance during migration", func() {
			vmi := newVMI(testNamespace, testVmName)
			now := metav1.Now()
			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			metadataCache.Migration.Store(migrationMetadata)

			manager, _ := newLibvirtDomainManagerDefault()

			Expect(manager.FreezeVMI(vmi, 0)).To(MatchError(ContainSubstring("VMI is currently during migration")))
		})
		It("should unfreeze a VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil).Times(1)
			mockLibvirt.DomainEXPECT().Free().Times(1)
			mockLibvirt.DomainEXPECT().FSThaw(nil, uint32(0)).Times(1)

			manager, _ := newLibvirtDomainManagerDefault()

			Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
		})
		It("should automatically unfreeze after a timeout a frozen VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil).Times(2)
			mockLibvirt.DomainEXPECT().Free().Times(2)
			mockLibvirt.DomainEXPECT().FSFreeze(nil, uint32(0)).Times(1)
			mockLibvirt.DomainEXPECT().FSThaw(nil, uint32(0)).Times(1)

			manager, _ := newLibvirtDomainManagerDefault()

			var unfreezeTimeout time.Duration = 3 * time.Second
			Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
			// wait for the unfreeze timeout
			time.Sleep(unfreezeTimeout + 2*time.Second)
		})
		It("should freeze and unfreeze a VirtualMachineInstance without a trigger to the unfreeze timeout", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedThawedOutput, nil)
			mockLibvirt.ConnectionEXPECT().QemuAgentCommand(`{"execute":"`+string(agentpoller.GetFSFreezeStatus)+`"}`, testDomainName).Return(expectedFrozenOutput, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil).Times(2)
			mockLibvirt.DomainEXPECT().Free().Times(2)
			mockLibvirt.DomainEXPECT().FSFreeze(nil, uint32(0)).Times(1)
			mockLibvirt.DomainEXPECT().FSThaw(nil, uint32(0)).Times(1)

			manager, _ := newLibvirtDomainManagerDefault()

			var unfreezeTimeout time.Duration = 3 * time.Second
			Expect(manager.FreezeVMI(vmi, int32(unfreezeTimeout.Seconds()))).To(Succeed())
			time.Sleep(time.Second)
			Expect(manager.UnfreezeVMI(vmi)).To(Succeed())
			// wait for the unfreeze timeout
			time.Sleep(unfreezeTimeout + 2*time.Second)
		})
		It("should update domain with memory dump info when completed successfully", func() {
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(nil)

			manager, _ := newLibvirtDomainManagerDefault()

			vmi := newVMI(testNamespace, testVmName)
			Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
			// Expect extra call to memory dump not to impact
			Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())

			Eventually(func() bool {
				memoryDump, _ := metadataCache.MemoryDump.Load()
				return memoryDump.Completed
			}, 5*time.Second, 2).Should(BeTrue())
		})
		It("should skip memory dump if the same dump command already completed successfully", func() {
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Times(1).Return(nil)

			manager, _ := newLibvirtDomainManagerDefault()

			vmi := newVMI(testNamespace, testVmName)
			Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
			// Expect extra call to memory dump not to impact
			Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())

			Eventually(func() bool {
				memoryDump, _ := metadataCache.MemoryDump.Load()
				return memoryDump.Completed
			}, 5*time.Second, 2).Should(BeTrue())
			// Expect extra call to memory dump after completion
			// not to call core dump command again
			Expect(manager.MemoryDump(vmi, testDumpPath)).To(Succeed())
		})
		It("should update domain with memory dump info if memory dump failed", func() {
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			dumpFailure := fmt.Errorf("Memory dump failed!!")
			mockLibvirt.DomainEXPECT().CoreDumpWithFormat(testDumpPath, libvirt.DOMAIN_CORE_DUMP_FORMAT_RAW, libvirt.DUMP_MEMORY_ONLY).Return(dumpFailure)

			manager, _ := newLibvirtDomainManagerDefault()

			vmi := newVMI(testNamespace, testVmName)
			err := manager.MemoryDump(vmi, testDumpPath)
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() bool {
				memoryDump, _ := metadataCache.MemoryDump.Load()
				return memoryDump.Failed
			}, 5*time.Second).Should(BeTrue(), "failed memory dump result wasn't set")
		})
		It("should pause a VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().Suspend().Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()

			Expect(manager.PauseVMI(vmi)).To(Succeed())
		})
		It("should not try to pause a paused VirtualMachineInstance", func() {
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			manager, _ := newLibvirtDomainManagerDefault()
			// no call to suspend

			Expect(manager.PauseVMI(vmi)).To(Succeed())
		})
		It("should unpause a VirtualMachineInstance", func() {
			isSetTimeCalled := make(chan bool, 1)
			defer close(isSetTimeCalled)

			// Make sure that we always free the domain after use
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).MaxTimes(2).Return(mockLibvirt.VirtDomain, nil)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockLibvirt.DomainEXPECT().Resume().Return(nil)
			mockLibvirt.DomainEXPECT().SetTime(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Do(func(interface{}, interface{}, interface{}) {
				isSetTimeCalled <- true
			})
			mockLibvirt.DomainEXPECT().Free()
			isFreeCalled := make(chan bool, 1)
			defer close(isFreeCalled)
			mockLibvirt.DomainEXPECT().Free().Do(
				func() {
					isFreeCalled <- true
				})
			manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, "fake", "fake", nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
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
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			manager, _ := newLibvirtDomainManagerDefault()
			// no call to unpause
			Expect(manager.UnpauseVMI(vmi)).To(Succeed())
		})

		It("should not add discard=unmap if a disk is preallocated", func() {
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
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
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(gomock.Any()).MaxTimes(2).DoAndReturn(func(xml string) (cli.VirDomain, error) {
				By(fmt.Sprintf("%s\n", xml))
				Expect(strings.Contains(xml, "discard=\"unmap\"")).To(BeFalse())
				return mockDomainWithFreeExpectation(xml)
			})
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, "fake", "fake", nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{
				VirtualMachineSMBios: &cmdv1.SMBios{},
				PreallocatedVolumes:  []string{"permvolume1"},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should hotplug a disk if a volume was hotplugged", func() {
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal(filepath.Join(v1.HotplugDiskDir, "/hpvolume1.img")))
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
					File: filepath.Join(v1.HotplugDiskDir, "hpvolume1.img"),
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
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(xmlDomain)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXMLWithInterfaces)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().AttachDeviceFlags(strings.ToLower(string(attachBytes)), affectDeviceLiveAndConfigLibvirtFlags)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should unplug a disk if a volume was unplugged", func() {
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			detachDisk := api.Disk{
				Device: "disk",
				Type:   "file",
				Source: api.DiskSource{
					File: filepath.Join(v1.HotplugDiskDir, "hpvolume1.img"),
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

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().DetachDeviceFlags(strings.ToLower(string(detachBytes)), affectDeviceLiveAndConfigLibvirtFlags)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should not plug/unplug a disk if nothing changed", func() {
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			setDomainExpectations(vmi)
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal(filepath.Join(v1.HotplugDiskDir, "hpvolume1.img")))
				return true, nil
			}
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})

		It("should not hotplug a disk if a volume was hotplugged, but the disk is not ready yet", func() {
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal(filepath.Join(v1.HotplugDiskDir, "hpvolume1.img")))
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
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(xmlDomain)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXMLWithInterfaces)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should inject a cd-rom", func() {
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
					Name: "cdrom-volume",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Bus: v1.DiskBusSATA,
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
					Name: "cdrom-volume",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name:         "dv2",
							Hotpluggable: true,
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
					Name:  "cdrom-volume",
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal(filepath.Join(v1.HotplugDiskDir, "/cdrom-volume.img")))
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
				{
					Device: "cdrom",
					Type:   "file",
					Target: api.DiskTarget{
						Bus:    v1.DiskBusSATA,
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Cache:       "none",
						Name:        "qemu",
						Type:        "raw",
						ErrorPolicy: "stop",
						Discard:     "unmap",
					},
					Alias: api.NewUserDefinedAlias("cdrom-volume"),
				},
			}
			updateDisk := api.Disk{
				Device: "cdrom",
				Type:   "file",
				Source: api.DiskSource{
					File: filepath.Join(v1.HotplugDiskDir, "cdrom-volume.img"),
				},
				Target: api.DiskTarget{
					Bus:    v1.DiskBusSATA,
					Device: "sda",
				},
				Driver: &api.DiskDriver{
					Cache:       "none",
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: "stop",
					Discard:     "unmap",
				},
				Alias: api.NewUserDefinedAlias("cdrom-volume"),
			}
			xmlDomain2, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			updateBytes, err := xml.Marshal(updateDisk)
			Expect(err).ToNot(HaveOccurred())
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(xmlDomain)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXMLWithInterfaces)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().UpdateDeviceFlags(strings.ToLower(string(updateBytes)), affectDeviceLiveAndConfigLibvirtFlags)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		It("should eject a cd-rom", func() {
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
					Name: "cdrom-volume",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Bus: v1.DiskBusSATA,
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
				{
					Name:  "cdrom-volume",
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
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			domainSpec := expectedDomainFor(vmi)
			xmlDomain, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			domainSpecWithPlaceholderInterfaces := addPlaceHolderInterfaces(vmi, domainSpec)
			domainXMLWithInterfaces, err := xml.MarshalIndent(domainSpecWithPlaceholderInterfaces, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			checkIfDiskReadyToUse = func(filename string) (bool, error) {
				Expect(filename).To(Equal(filepath.Join(v1.HotplugDiskDir, "/cdrom-volume.img")))
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
				{
					Device: "cdrom",
					Type:   "file",
					Source: api.DiskSource{
						File: filepath.Join(v1.HotplugDiskDir, "cdrom-volume.img"),
					},
					Target: api.DiskTarget{
						Bus:    v1.DiskBusSATA,
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Cache:       "none",
						Name:        "qemu",
						Type:        "raw",
						ErrorPolicy: "stop",
						Discard:     "unmap",
					},
					Alias: api.NewUserDefinedAlias("cdrom-volume"),
				},
			}
			updateDisk := api.Disk{
				Device: "cdrom",
				Type:   "block",
				Target: api.DiskTarget{
					Bus:    v1.DiskBusSATA,
					Device: "sda",
				},
				Driver: &api.DiskDriver{
					Cache:       "none",
					Name:        "qemu",
					Type:        "raw",
					ErrorPolicy: "stop",
					Discard:     "unmap",
				},
				Alias: api.NewUserDefinedAlias("cdrom-volume"),
			}
			xmlDomain2, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())
			updateBytes, err := xml.Marshal(updateDisk)
			Expect(err).ToNot(HaveOccurred())
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(xmlDomain)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.ConnectionEXPECT().DomainDefineXML(string(domainXMLWithInterfaces)).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			mockLibvirt.DomainEXPECT().UpdateDeviceFlags(strings.ToLower(string(updateBytes)), affectDeviceLiveAndConfigLibvirtFlags)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return(string(xmlDomain2), nil)
			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(2)).MaxTimes(1).Return(string(domainXMLWithInterfaces), nil)
			manager, _ := newLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, "/usr/share/OVMF", ephemeralDiskCreatorMock, mockDirectIOChecker, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
		})
		DescribeTable("should set freePageReporting", func(memory *v1.Memory, clusterFreePageReportingDisabled bool, cpu *v1.CPU, annotationValue, expectedFreePageReportingValue string) {
			vmi := newVMI(testNamespace, testVmName)
			if vmi.Annotations == nil {
				vmi.Annotations = make(map[string]string)
			}

			vmi.Annotations[v1.FreePageReportingDisabledAnnotation] = annotationValue
			vmi.Spec.Domain.Memory = memory
			vmi.Spec.Domain.CPU = cpu
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(nil, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			if clusterFreePageReportingDisabled {
				clusterConfig, _, _ = testutils.NewFakeClusterConfigUsingKVConfig(&v1.KubeVirtConfiguration{
					VirtualMachineOptions: &v1.VirtualMachineOptions{
						DisableFreePageReporting: &v1.DisableFreePageReporting{},
					},
				})
			}
			setDomainExpectations(vmi)
			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockLibvirt.DomainEXPECT().CreateWithFlags(libvirt.DOMAIN_NONE).Return(nil)
			manager, _ := newLibvirtDomainManagerDefault()
			newspec, err := manager.SyncVMI(vmi, true, &cmdv1.VirtualMachineOptions{VirtualMachineSMBios: &cmdv1.SMBios{}, Topology: topology, ClusterConfig: &cmdv1.ClusterConfig{FreePageReportingDisabled: clusterFreePageReportingDisabled}})
			Expect(err).ToNot(HaveOccurred())
			Expect(newspec).ToNot(BeNil())
			Expect(newspec.Devices.Ballooning.FreePageReporting).To(Equal(expectedFreePageReportingValue))
		},
			Entry("disabled if free page reporting is disabled at cluster level", nil, true, nil, "false", "off"),
			Entry("enabled if vmi is not requesting any high performance components", nil, false, nil, "false", "on"),
			Entry("disabled if vmi is requesting Hugepages", &v1.Memory{Hugepages: &v1.Hugepages{PageSize: "1Gi"}}, false, nil, "false", "off"),
			Entry("disabled if vmi is requesting Realtime", nil, false, &v1.CPU{Realtime: &v1.Realtime{}}, "false", "off"),
			Entry("disabled if vmi is requesting DedicatedCPU", nil, false, &v1.CPU{
				DedicatedCPUPlacement: true}, "false", "off"),
			Entry("disabled if vmi has the disable free page reporting annotation", nil, false, nil, "true", "off"),
		)

		It("should return SEV platform info", func() {
			sevNodeParameters := &api.SEVNodeParameters{
				PDH:       "AAABBBCCC",
				CertChain: "DDDEEEFFF",
			}

			mockLibvirt.ConnectionEXPECT().GetSEVInfo().Return(sevNodeParameters, nil)

			manager, _ := newLibvirtDomainManagerDefault()
			sevPlatfomrInfo, err := manager.GetSEVInfo()
			Expect(err).ToNot(HaveOccurred())
			Expect(sevPlatfomrInfo.PDH).To(Equal(sevNodeParameters.PDH))
			Expect(sevPlatfomrInfo.CertChain).To(Equal(sevNodeParameters.CertChain))
		})

		It("should return a VirtualMachineInstance launch measurement", func() {
			if runtime.GOARCH == "s390x" {
				Skip("Test is specific to amd64 architecture")
			}

			domainLaunchSecurityParameters := &libvirt.DomainLaunchSecurityParameters{
				SEVMeasurementSet: true,
				SEVMeasurement:    "AAABBBCCC",
			}
			loaderBytes := []byte("OVMF binary with SEV support")
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil)
			// Make sure that we always free the domain after use
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetLaunchSecurityInfo(uint32(0)).Return(domainLaunchSecurityParameters, nil)

			ovmfDir, err := os.MkdirTemp("", "ovmfdir")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(ovmfDir)
			err = os.WriteFile(filepath.Join(ovmfDir, efi.EFICodeSEV), loaderBytes, 0644)
			Expect(err).ToNot(HaveOccurred())

			manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, nil, ovmfDir, ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			sevMeasurementInfo, err := manager.GetLaunchMeasurement(vmi)
			if runtime.GOARCH == "amd64" {
				Expect(err).ToNot(HaveOccurred())
				Expect(sevMeasurementInfo.Measurement).To(Equal(domainLaunchSecurityParameters.SEVMeasurement))
				Expect(sevMeasurementInfo.LoaderSHA).To(Equal(fmt.Sprintf("%x", sha256.Sum256(loaderBytes))))
			} else {
				Expect(err).To(HaveOccurred())
			}
		})

		It("should inject a secret into a VirtualMachineInstance", func() {
			sevSecretOptions := &v1.SEVSecretOptions{
				Header: "AAABBB",
				Secret: "CCCDDD",
			}
			domainLaunchSecurityStateParameters := &libvirt.DomainLaunchSecurityStateParameters{
				SEVSecret:          sevSecretOptions.Secret,
				SEVSecretSet:       true,
				SEVSecretHeader:    sevSecretOptions.Header,
				SEVSecretHeaderSet: true,
			}
			vmi := newVMI(testNamespace, testVmName)

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).Return(mockLibvirt.VirtDomain, nil)
			// Make sure that we always free the domain after use
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().SetLaunchSecurityState(domainLaunchSecurityStateParameters, uint32(0)).Return(nil)

			manager, _ := newLibvirtDomainManagerDefault()
			err := manager.InjectLaunchSecret(vmi, sevSecretOptions)
			Expect(err).ToNot(HaveOccurred())
		})

		Context("Memory hotplug", func() {
			var vmi *v1.VirtualMachineInstance
			var manager *LibvirtDomainManager
			var domainSpec *api.DomainSpec

			BeforeEach(func() {
				vmi = newVMI(testNamespace, testVmName)

				guestMemory := resource.MustParse("128Mi")
				maxGuestMemory := resource.MustParse("256Mi")
				vmi.Spec.Domain.Memory = &v1.Memory{
					Guest:    &maxGuestMemory,
					MaxGuest: &maxGuestMemory,
				}
				vmi.Status.Memory = &v1.MemoryStatus{
					GuestCurrent:   &guestMemory,
					GuestAtBoot:    &guestMemory,
					GuestRequested: &guestMemory,
				}

				manager = &LibvirtDomainManager{
					virConn:       mockLibvirt.VirtConnection,
					virtShareDir:  testVirtShareDir,
					metadataCache: metadataCache,
					cpuSetGetter:  fakeCpuSetGetter,
				}
			})

			It("should attach a virtio-mem device when memory hotplug has been requested", func() {
				mockLibvirt.ConnectionEXPECT().LookupDomainByName(api.VMINamespaceKeyFunc(vmi)).Return(mockLibvirt.VirtDomain, nil)

				domainSpec = &api.DomainSpec{Devices: api.Devices{}}
				domainSpecXML, err := xml.Marshal(domainSpec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(domainSpecXML), nil)

				memoryDevice, err := memory.BuildMemoryDevice(vmi)
				Expect(err).ToNot(HaveOccurred())
				memoryDeviceXML, err := xml.Marshal(memoryDevice)
				Expect(err).ToNot(HaveOccurred())

				attachFlags := libvirt.DOMAIN_DEVICE_MODIFY_LIVE | libvirt.DOMAIN_DEVICE_MODIFY_CONFIG
				mockLibvirt.DomainEXPECT().AttachDeviceFlags(strings.ToLower(string(memoryDeviceXML)), attachFlags).Return(nil)

				mockLibvirt.DomainEXPECT().Free()

				err = manager.UpdateGuestMemory(vmi)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should update the virtio-mem device if it already exists", func() {
				mockLibvirt.ConnectionEXPECT().LookupDomainByName(api.VMINamespaceKeyFunc(vmi)).Return(mockLibvirt.VirtDomain, nil)

				size, err := vcpu.QuantityToByte(resource.MustParse("128Mi"))
				Expect(err).ToNot(HaveOccurred())
				requested, err := vcpu.QuantityToByte(resource.MustParse("64Mi"))
				Expect(err).ToNot(HaveOccurred())
				block, err := vcpu.QuantityToByte(resource.MustParse("2Mi"))
				Expect(err).ToNot(HaveOccurred())

				domainSpec = &api.DomainSpec{
					Devices: api.Devices{
						Memory: &api.MemoryDevice{
							Model: "virtio-mem",
							Alias: api.NewUserDefinedAlias("virtio-mem"),
							Address: &api.Address{
								Type:     "pci",
								Domain:   "0x0000",
								Bus:      "0x02",
								Slot:     "0x00",
								Function: "0x0",
							},
							Target: &api.MemoryTarget{
								Node:      "0",
								Address:   &api.MemoryAddress{Base: "0x100000000"},
								Size:      size,
								Requested: requested,
								Block:     block,
							},
						},
					},
				}
				domainSpecXML, err := xml.Marshal(domainSpec)
				Expect(err).ToNot(HaveOccurred())

				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(domainSpecXML), nil)

				// hotplug to MaxGuest
				vmi.Spec.Domain.Memory.Guest = virtpointer.P(resource.MustParse("256Mi"))

				memoryDevice, err := memory.BuildMemoryDevice(vmi)
				Expect(err).ToNot(HaveOccurred())

				domainSpec.Devices.Memory.Target.Requested = memoryDevice.Target.Requested

				memoryDeviceXML, err := xml.Marshal(domainSpec.Devices.Memory)
				Expect(err).ToNot(HaveOccurred())

				attachFlags := libvirt.DOMAIN_DEVICE_MODIFY_LIVE
				mockLibvirt.DomainEXPECT().UpdateDeviceFlags(strings.ToLower(string(memoryDeviceXML)), attachFlags).Return(nil)

				mockLibvirt.DomainEXPECT().Free()

				err = manager.UpdateGuestMemory(vmi)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	Context("test marking graceful shutdown", func() {
		It("Should set metadata when calling MarkGracefulShutdown api", func() {
			manager, _ := newLibvirtDomainManagerDefault()
			manager.MarkGracefulShutdownVMI()

			gracePeriod, _ := metadataCache.GracePeriod.Load()
			Expect(gracePeriod.MarkedForGracefulShutdown).To(Equal(virtpointer.P(true)))
		})

		It("Should signal graceful shutdown after marked for shutdown", func() {
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).AnyTimes().DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().ShutdownFlags(libvirt.DOMAIN_SHUTDOWN_DEFAULT).Return(nil)

			manager, _ := newLibvirtDomainManagerDefault()

			vmi := newVMI(testNamespace, testVmName)
			manager.SignalShutdownVMI(vmi)

			gracePeriod, _ := metadataCache.GracePeriod.Load()
			Expect(gracePeriod.DeletionTimestamp).NotTo(BeNil())
		})
	})
	Context("test migration monitor", func() {
		It("migration should be canceled if it's not progressing", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
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

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
			mockLibvirt.DomainEXPECT().AbortJob()

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should be canceled if timeout has been reached", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			var migrationData = 32479827394
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

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().Return(fake_jobinfo, nil)
			mockLibvirt.DomainEXPECT().AbortJob()

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should switch to PostCopy", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and send a different event otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					return &libvirt.DomainJobInfo{
						Type: libvirt.DOMAIN_JOB_CANCELLED,
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
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(1).Return(nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})

		It("migration should switch to PostCopy eventually", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and send a different event otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					return &libvirt.DomainJobInfo{
						Type: libvirt.DOMAIN_JOB_CANCELLED,
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
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})

			counter := 0
			mockLibvirt.DomainEXPECT().MigrateStartPostCopy(gomock.Eq(uint32(0))).Times(2).DoAndReturn(func(flag uint32) error {
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

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should switch to Paused if AllowWorkloadDisruption is allowed and PostCopy is not", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
				// stop decreasing data and send a different event otherwise this
				// job will run indefinitely until timeout
				if migrationData <= 32479826519 {
					return &libvirt.DomainJobInfo{
						Type: libvirt.DOMAIN_JOB_CANCELLED,
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
				AllowPostCopy:           false,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				paused: pausedVMIs{
					paused: make(map[types.UID]bool),
				},
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().Suspend().Times(1).Return(nil)

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		It("migration should be canceled if Paused workload didn't migrate until timeout was reached", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
			var migrationData = 32479827394
			fake_jobinfo := func() *libvirt.DomainJobInfo {
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
				AllowPostCopy:           false,
				AllowWorkloadDisruption: true,
			}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			manager := &LibvirtDomainManager{
				paused: pausedVMIs{
					paused: make(map[types.UID]bool),
				},
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).AnyTimes().DoAndReturn(func(flag libvirt.DomainGetJobStatsFlags) (*libvirt.DomainJobInfo, error) {
				return fake_jobinfo(), nil
			})
			mockLibvirt.DomainEXPECT().Suspend().Times(1).Return(nil)
			mockLibvirt.DomainEXPECT().AbortJob()

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
		})
		// This is incomplete as it is not verifying that we abort. Previously it wasn't even testing anything at all
		It("migration should be canceled when requested", func() {
			migrationUid := types.UID("111222333")

			now := metav1.Now()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   migrationUid,
				StartTimestamp: &now,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			// These lines do not test anything but needs to be here because otherwise test will panic
			mockLibvirt.DomainEXPECT().AbortJob().MaxTimes(1)
			migrationInProgress := func() *libvirt.DomainJobInfo {
				return &libvirt.DomainJobInfo{
					Type:             libvirt.DOMAIN_JOB_UNBOUNDED,
					DataRemaining:    uint64(32479827394),
					DataRemainingSet: true,
				}
			}()
			mockLibvirt.DomainEXPECT().GetJobInfo().MaxTimes(1).Return(migrationInProgress, nil)

			manager, _ := newLibvirtDomainManagerDefault()

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.StartTimestamp = &now
			migrationMetadata.UID = migrationUid
			metadataCache.Migration.Store(migrationMetadata)

			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())

			// Allow the aync-abort (goroutine) to be processed before finishing.
			// This is required in order to allow the expected calls to occur.
			time.Sleep(2 * time.Second)

			migration, _ := metadataCache.Migration.Load()
			Expect(migration.AbortStatus).To(Equal(string(v1.MigrationAbortSucceeded)))
		})

		It("shouldn't be able to call cancel migration more than once", func() {
			now := metav1.Time{Time: time.Unix(time.Now().UTC().Unix(), 0)}
			secondBefore := metav1.Time{Time: now.Add(-time.Second)}
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID:   "111222333",
				StartTimestamp: &now,
			}

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.UID = vmi.Status.MigrationState.MigrationUID
			migrationMetadata.AbortStatus = string(v1.MigrationAbortInProgress)
			migrationMetadata.StartTimestamp = &secondBefore
			metadataCache.Migration.Store(migrationMetadata)

			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			manager, _ := newLibvirtDomainManagerDefault()
			Expect(manager.CancelVMIMigration(vmi)).To(Succeed())
		})
		It("migration cancellation should be finilized even if we missed status update", func() {
			migrationErrorChan := make(chan error)
			defer close(migrationErrorChan)
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

			migrationMetadata, _ := metadataCache.Migration.Load()
			migrationMetadata.UID = vmi.Status.MigrationState.MigrationUID
			migrationMetadata.AbortStatus = string(v1.MigrationAbortInProgress)
			metadataCache.Migration.Store(migrationMetadata)

			manager := &LibvirtDomainManager{
				virConn:       mockLibvirt.VirtConnection,
				virtShareDir:  testVirtShareDir,
				metadataCache: metadataCache,
				cpuSetGetter:  fakeCpuSetGetter,
			}

			mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
			mockLibvirt.DomainEXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			gomock.InOrder(
				mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo_running, nil),
				mockLibvirt.DomainEXPECT().GetJobStats(libvirt.DomainGetJobStatsFlags(0)).Return(fake_jobinfo, nil),
			)

			monitor := newMigrationMonitor(vmi, manager, options, migrationErrorChan)
			monitor.startMonitor()
			Eventually(func() string {
				migration, _ := metadataCache.Migration.Load()
				return migration.AbortStatus
			}, 5*time.Second, 2).Should(Equal(string(v1.MigrationAbortSucceeded)))
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

			manager, _ := newLibvirtDomainManagerDefault()
			Expect(manager.PrepareMigrationTarget(vmi, true, &cmdv1.VirtualMachineOptions{})).To(Succeed())
		})

		It("should detect inprogress migration job", func() {
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
			}

			startupMigrationMetadata, _ := metadataCache.Migration.Load()
			startupMigrationMetadata.UID = vmi.Status.MigrationState.MigrationUID
			t := metav1.Now()
			startupMigrationMetadata.StartTimestamp = &t
			metadataCache.Migration.Store(startupMigrationMetadata)
			manager, _ := newLibvirtDomainManagerDefault()

			options := &cmdclient.MigrationOptions{
				Bandwidth:               resource.MustParse("64Mi"),
				ProgressTimeout:         150,
				CompletionTimeoutPerGiB: 300,
			}
			Expect(manager.MigrateVMI(vmi, options)).To(Succeed())
			migration, _ := metadataCache.Migration.Load()
			Expect(migration).To(Equal(startupMigrationMetadata))
		})
		It("should correctly collect a list of disks for migration", func() {
			_true := true
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

			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(embedMigrationDomain, nil)

			copyDisks := getDiskTargetsForMigration(mockLibvirt.VirtDomain, vmi)
			Expect(copyDisks).Should(ConsistOf("vdb", "vdd"))
		})
		AfterEach(func() {
			ip.GetLoopbackAddress = funcPreviousValue
		})
	})

	Context("on successful VirtualMachineInstance kill", func() {
		DescribeTable("should try to undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
				mockLibvirt.DomainEXPECT().UndefineFlags(libvirt.DOMAIN_UNDEFINE_KEEP_NVRAM).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, "fake", "fake", nil, "/usr/share/", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
				Expect(manager.DeleteVMI(newVMI(testNamespace, testVmName))).To(Succeed())
			},
			Entry("crashed", libvirt.DOMAIN_CRASHED),
			Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
		)
		DescribeTable("should try to destroy a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				mockLibvirt.ConnectionEXPECT().LookupDomainByName(testDomainName).DoAndReturn(mockDomainWithFreeExpectation)
				mockLibvirt.DomainEXPECT().GetState().Return(state, 1, nil)
				mockLibvirt.DomainEXPECT().DestroyFlags(libvirt.DOMAIN_DESTROY_GRACEFUL).Return(nil)
				manager, _ := newLibvirtDomainManagerDefault()
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
			isVmiPaused := migrationType == "paused"

			options := &cmdclient.MigrationOptions{
				UnsafeMigration:   migrationType == "unsafe",
				AllowAutoConverge: migrationType == "autoConverge",
				AllowPostCopy:     migrationType == "postCopy",
			}

			shouldConfigureParallel, parallelMigrationThreads := shouldConfigureParallelMigration(options)
			if shouldConfigureParallel {
				options.ParallelMigrationThreads = virtpointer.P(uint(parallelMigrationThreads))
			}

			flags := generateMigrationFlags(isBlockMigration, isVmiPaused, options)
			expectedMigrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER | libvirt.MIGRATE_PERSIST_DEST

			if isBlockMigration {
				expectedMigrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
			} else if migrationType == "unsafe" {
				expectedMigrateFlags |= libvirt.MIGRATE_UNSAFE
			}
			if options.AllowAutoConverge {
				expectedMigrateFlags |= libvirt.MIGRATE_AUTO_CONVERGE
			}
			if migrationType == "postCopy" {
				expectedMigrateFlags |= libvirt.MIGRATE_POSTCOPY
			}
			if migrationType == "paused" {
				expectedMigrateFlags |= libvirt.MIGRATE_PAUSED
			}
			if shouldConfigureParallel {
				expectedMigrateFlags |= libvirt.MIGRATE_PARALLEL
			}
			Expect(flags).To(Equal(expectedMigrateFlags), "libvirt migration flags are not set as expected")
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
			mockLibvirt.DomainEXPECT().Free()
			mockLibvirt.DomainEXPECT().GetState().Return(state, libvirtReason, nil).AnyTimes()
			mockLibvirt.DomainEXPECT().GetName().Return("test", nil)
			x, err := xml.MarshalIndent(api.NewMinimalDomainSpec("test"), "", "\t")
			Expect(err).ToNot(HaveOccurred())

			mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(x), nil)
			mockLibvirt.ConnectionEXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]cli.VirDomain{mockLibvirt.VirtDomain}, nil)

			manager, _ := newLibvirtDomainManagerDefault()
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
			const (
				domainStats = libvirt.DOMAIN_STATS_BALLOON |
					libvirt.DOMAIN_STATS_CPU_TOTAL |
					libvirt.DOMAIN_STATS_VCPU |
					libvirt.DOMAIN_STATS_INTERFACE |
					libvirt.DOMAIN_STATS_BLOCK |
					libvirt.DOMAIN_STATS_DIRTYRATE
				flags = libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING | libvirt.CONNECT_GET_ALL_DOMAINS_STATS_PAUSED
			)
			fakeDomainStats := []*stats.DomainStats{
				{},
			}

			mockLibvirt.ConnectionEXPECT().GetDomainStats(domainStats, gomock.Any(), flags).Return(fakeDomainStats, nil)

			manager, _ := newLibvirtDomainManagerDefault()
			domStats, err := manager.GetDomainStats()

			Expect(err).ToNot(HaveOccurred())
			Expect(domStats).ToNot(BeNil())
		})
	})

	Context("on failed GetDomainSpecWithRuntimeInfo", func() {
		It("should fall back to returning domain spec without runtime info", func() {
			manager, _ := newLibvirtDomainManagerDefault()

			mockLibvirt.DomainEXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			vmi := newVMI(testNamespace, testVmName)

			domainSpec := expectedDomainFor(vmi)

			domainXml, err := xml.MarshalIndent(domainSpec, "", "\t")
			Expect(err).ToNot(HaveOccurred())

			gomock.InOrder(
				// First call is via GetDomainSpecWithRuntimeInfo. Force an error
				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).MaxTimes(2).Return("", libvirt.Error{Code: libvirt.ERR_NO_DOMAIN}),
				// Subsequent calls are via GetDomainSpec
				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(libvirt.DOMAIN_XML_INACTIVE)).MaxTimes(2).Return(string(domainXml), nil),
				mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DomainXMLFlags(libvirt.DOMAIN_XML_MIGRATABLE)).MaxTimes(2).Return(string(domainXml), nil),
			)

			// we need the non-typecast object to make the function we want to test available
			libvirtmanager := manager.(*LibvirtDomainManager)

			domSpec, err := libvirtmanager.getDomainSpec(mockLibvirt.VirtDomain)
			Expect(err).ToNot(HaveOccurred())
			Expect(domSpec).ToNot(BeNil())
		})

		Context("on call to GetGuestOSInfo", func() {
			var libvirtmanager DomainManager
			var agentStore agentpoller.AsyncAgentStore

			BeforeEach(func() {
				agentStore = agentpoller.NewAsyncAgentStore()
				libvirtmanager, _ = NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			})

			It("should report nil when no OS info exists in the cache", func() {
				Expect(libvirtmanager.GetGuestOSInfo()).To(BeNil())
			})

			It("should report OS info when it exists in the cache", func() {
				fakeInfo := api.GuestOSInfo{
					Name: "TestGuestOSName",
				}
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_OS, fakeInfo)

				osInfo := libvirtmanager.GetGuestOSInfo()
				Expect(*osInfo).To(Equal(fakeInfo))
			})
		})

		Context("on call to InterfacesStatus", func() {
			var libvirtmanager DomainManager
			var agentStore agentpoller.AsyncAgentStore

			BeforeEach(func() {
				agentStore = agentpoller.NewAsyncAgentStore()
				libvirtmanager, _ = NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)
			})

			It("should return nil when no interfaces exists in the cache", func() {
				Expect(libvirtmanager.InterfacesStatus()).To(BeNil())
			})

			It("should report interfaces info when interfaces exists", func() {
				fakeInterfaces := []api.InterfaceStatus{{
					InterfaceName: "eth1",
					Mac:           "00:00:00:00:00:01",
				}}
				agentStore.Store(libvirt.DOMAIN_GUEST_INFO_INTERFACES, fakeInterfaces)
				interfacesStatus := agentStore.GetInterfaceStatus()

				Expect(interfacesStatus).To(Equal(fakeInterfaces))
			})
		})
	})

	It("executes GetGuestInfo", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(libvirt.DOMAIN_GUEST_INFO_USERS, []api.User{
			{
				Name:      "test",
				Domain:    "test",
				LoginTime: 0,
			},
		})
		agentStore.Store(agentpoller.GetFilesystem, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
				Disk: []api.FSDisk{
					{
						BusType: "scsi",
						Serial:  "testserial-1234",
					},
				},
			},
		})

		manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		vmi := newVMI(testNamespace, testVmName)
		guestInfo, _ := libvirtmanager.GetGuestInfo(vmi, []string{})
		Expect(guestInfo.UserList).To(ConsistOf(v1.VirtualMachineInstanceGuestOSUser{
			UserName:  "test",
			Domain:    "test",
			LoginTime: 0,
		}))
		Expect(guestInfo.FSInfo.Filesystems).To(ConsistOf(v1.VirtualMachineInstanceFileSystem{
			DiskName:       "test",
			MountPoint:     "/mnt/whatever",
			FileSystemType: "fs",
			UsedBytes:      0,
			TotalBytes:     0,
			Disk: []v1.VirtualMachineInstanceFileSystemDisk{
				{
					BusType: "scsi",
					Serial:  "testserial-1234",
				},
			},
		}))
	})

	It("executes GetUsers", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(libvirt.DOMAIN_GUEST_INFO_USERS, []api.User{
			{
				Name:      "test",
				Domain:    "test",
				LoginTime: 0,
			},
		})

		manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		virtualMachineInstanceGuestAgentInfo := libvirtmanager.GetUsers()
		Expect(virtualMachineInstanceGuestAgentInfo).ToNot(BeEmpty())
	})

	It("executes GetFilesystems", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GetFilesystem, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
				Disk: []api.FSDisk{
					{
						BusType: "scsi",
						Serial:  "testserial-1234",
					},
				},
			},
		})

		manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)

		// we need the non-typecast object to make the function we want to test available
		libvirtmanager := manager.(*LibvirtDomainManager)

		virtualMachineInstanceGuestAgentInfo := libvirtmanager.GetFilesystems()
		Expect(virtualMachineInstanceGuestAgentInfo).ToNot(BeEmpty())
	})

	It("executes generateCloudInitEmptyISO and succeeds", func() {
		agentStore := agentpoller.NewAsyncAgentStore()
		agentStore.Store(agentpoller.GetFilesystem, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
				Disk: []api.FSDisk{
					{
						BusType: "scsi",
						Serial:  "testserial-1234",
					},
				},
			},
		})

		manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)

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
		agentStore.Store(agentpoller.GetFilesystem, []api.Filesystem{
			{
				Name:       "test",
				Mountpoint: "/mnt/whatever",
				Type:       "fs",
				UsedBytes:  0,
				TotalBytes: 0,
				Disk: []api.FSDisk{
					{
						BusType: "scsi",
						Serial:  "testserial-1234",
					},
				},
			},
		})

		manager, _ := NewLibvirtDomainManager(mockLibvirt.VirtConnection, testVirtShareDir, testEphemeralDiskDir, &agentStore, "/usr/share/OVMF", ephemeralDiskCreatorMock, metadataCache, nil, virtconfig.DefaultDiskVerificationMemoryLimitBytes, fakeCpuSetGetter, false)

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

	Context("Guest Agent Compatibility", func() {
		var vmi *v1.VirtualMachineInstance
		var vmiWithPassword *v1.VirtualMachineInstance
		var vmiWithSSH *v1.VirtualMachineInstance
		var basicCommands []v1.GuestAgentCommandInfo
		var sshCommands []v1.GuestAgentCommandInfo
		var oldSshCommands []v1.GuestAgentCommandInfo
		var passwordCommands []v1.GuestAgentCommandInfo
		const agentSupported = "This guest agent is supported"

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{}
			vmiWithPassword = &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					AccessCredentials: []v1.AccessCredential{
						{
							UserPassword: &v1.UserPasswordAccessCredential{
								PropagationMethod: v1.UserPasswordAccessCredentialPropagationMethod{
									QemuGuestAgent: &v1.QemuGuestAgentUserPasswordAccessCredentialPropagation{},
								},
							},
						},
					},
				},
			}
			vmiWithSSH = &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					AccessCredentials: []v1.AccessCredential{
						{
							SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
								PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
									QemuGuestAgent: &v1.QemuGuestAgentSSHPublicKeyAccessCredentialPropagation{},
								},
							},
						},
					},
				},
			}

			basicCommands = []v1.GuestAgentCommandInfo{}
			for _, cmdName := range requiredGuestAgentCommands {
				basicCommands = append(basicCommands, v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				})
			}

			sshCommands = []v1.GuestAgentCommandInfo{}
			for _, cmdName := range sshRelatedGuestAgentCommands {
				sshCommands = append(sshCommands, v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				})
			}

			oldSshCommands = []v1.GuestAgentCommandInfo{}
			for _, cmdName := range oldSSHRelatedGuestAgentCommands {
				oldSshCommands = append(oldSshCommands, v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				})
			}

			passwordCommands = []v1.GuestAgentCommandInfo{}
			for _, cmdName := range passwordRelatedGuestAgentCommands {
				passwordCommands = append(passwordCommands, v1.GuestAgentCommandInfo{
					Name:    cmdName,
					Enabled: true,
				})
			}
		})

		It("should succeed with empty VMI and basic commands", func() {
			result, reason := isGuestAgentSupported(vmi, basicCommands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should succeed with empty VMI and all commands", func() {
			var commands []v1.GuestAgentCommandInfo
			commands = append(commands, basicCommands...)
			commands = append(commands, sshCommands...)
			commands = append(commands, passwordCommands...)

			result, reason := isGuestAgentSupported(vmi, commands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should fail with password and basic commands", func() {
			result, reason := isGuestAgentSupported(vmiWithPassword, basicCommands)
			Expect(result).To(BeFalse())
			Expect(reason).To(Equal("This guest agent doesn't support required password commands"))
		})

		It("should succeed with password and required commands", func() {
			var commands []v1.GuestAgentCommandInfo
			commands = append(commands, basicCommands...)
			commands = append(commands, passwordCommands...)

			result, reason := isGuestAgentSupported(vmiWithPassword, commands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should fail with SSH and basic commands", func() {
			result, reason := isGuestAgentSupported(vmiWithSSH, basicCommands)
			Expect(result).To(BeFalse())
			Expect(reason).To(Equal("This guest agent doesn't support required public key commands"))
		})

		It("should succeed with SSH and required commands", func() {
			var commands []v1.GuestAgentCommandInfo
			commands = append(commands, basicCommands...)
			commands = append(commands, sshCommands...)

			result, reason := isGuestAgentSupported(vmiWithSSH, commands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})

		It("should succeed with SSH and old required commands", func() {
			var commands []v1.GuestAgentCommandInfo
			commands = append(commands, basicCommands...)
			commands = append(commands, oldSshCommands...)

			result, reason := isGuestAgentSupported(vmiWithSSH, commands)
			Expect(result).To(BeTrue())
			Expect(reason).To(Equal(agentSupported))
		})
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
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
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
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			}),
		Entry("be empty if non-hotplug disk is added",
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			},
			[]api.Disk{}),
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
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
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
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			}),
		Entry("be empty if non-hotplug disk changed",
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file",
					},
				},
			},
			[]api.Disk{
				{
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file-changed",
					},
				},
			},
			[]api.Disk{}),
	)
})

var _ = Describe("getUpdatedDisks", func() {
	DescribeTable("should return the correct values", func(oldDisks, newDisks, expected []api.Disk) {
		res := getUpdatedDisks(oldDisks, newDisks)
		Expect(res).To(Equal(expected))
	},
		Entry("be empty with empty old and new",
			[]api.Disk{},
			[]api.Disk{},
			nil),
		Entry("be empty with empty old and new being identical",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			nil),
		Entry("be empty with new disk being added",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			nil),
		Entry("be empty if disk removed",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sdb",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			nil),
		Entry("be empty not cd-roms",
			[]api.Disk{
				{
					Device: "disk",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "disk",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			nil),
		Entry("be empty if not hotplug",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test",
						File: "file1",
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: "file2",
					},
				},
			},
			nil),
		Entry("cd-rom inject",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test1",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Type:   "file",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Type: "raw",
					},
					Source: api.DiskSource{
						Name: "test1",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			}),
		Entry("cd-rom eject",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test1",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Type:   "block",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Type: "raw",
					},
				},
			}),
		Entry("cd-rom swap",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test1",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Type:   "file",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Type: "raw",
					},
					Source: api.DiskSource{
						Name: "test2",
						File: filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			}),
		Entry("cd-rom swap block",
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test1",
						File: filepath.Join(v1.HotplugDiskDir, "file1"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Source: api.DiskSource{
						Name: "test2",
						Dev:  filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			},
			[]api.Disk{
				{
					Device: "cdrom",
					Type:   "block",
					Target: api.DiskTarget{
						Device: "sda",
					},
					Driver: &api.DiskDriver{
						Type: "raw",
					},
					Source: api.DiskSource{
						Name: "test2",
						Dev:  filepath.Join(v1.HotplugDiskDir, "file2"),
					},
				},
			}),
	)
})

var _ = Describe("migratableDomXML", func() {
	var ctrl *gomock.Controller
	var mockLibvirt *testing.Libvirt
	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockLibvirt = testing.NewLibvirt(ctrl)
	})
	It("should parse the XML with the metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
    </kubevirt>
   </metadata>
</domain>`
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
    </kubevirt>
   </metadata>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
	It("should change CPU pinning according to migration metadata", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="4"></vcpupin>
    <vcpupin vcpu="1" cpuset="5"></vcpupin>
  </cputune>
</domain>`
		// migratableDomXML() removes the migration block but not its ident, which is its own token, hence the blank line below
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <vcpu placement="static">2</vcpu>
  <cputune>
    <vcpupin vcpu="0" cpuset="6"></vcpupin>
    <vcpupin vcpu="1" cpuset="7"></vcpupin>
  </cputune>
  <cpu>
    <topology sockets="1" cores="2" threads="1"></topology>
  </cpu>
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
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		Expect(domSpec.VCPU).NotTo(BeNil())
		Expect(domSpec.CPUTune).NotTo(BeNil())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred(), "failed to generate target domain XML")

		By("ensuring the generated XML is accurate")
		Expect(newXML).To(Equal(expectedXML), "the target XML is not as expected")
	})
	DescribeTable("slices section", func(domXML string) {
		retDiskSize := func(disk *libvirtxml.DomainDisk) (int64, error) {
			return 2028994560, nil
		}
		getDiskVirtualSizeFunc = retDiskSize
		const (
			volName       = "datavolumedisk1"
			sourcePvcName = "src-pvc"
			destPvcName   = "dst-pvc"
		)
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type="file" device="disk" model="virtio-non-transitional">
      <driver name="qemu" type="raw" cache="none" error_policy="stop" discard="unmap"></driver>
      <source file="/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img" index="1">
        <slices>
          <slice type="storage" offset="0" size="2028994560"></slice>
        </slices>
      </source>
      <backingStore></backingStore>
      <target dev="vda" bus="virtio"></target>
      <alias name="ua-datavolumedisk1"></alias>
      <address type="pci" domain="0x0000" bus="0x07" slot="0x00" function="0x0"></address>
    </disk>
  </devices>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		vmi.Spec.Volumes = append(vmi.Spec.Volumes,
			v1.Volume{
				Name: volName,
				VolumeSource: v1.VolumeSource{
					DataVolume: &v1.DataVolumeSource{
						Name: sourcePvcName,
					},
				},
			})
		vmi.Status.MigratedVolumes = []v1.StorageMigratedVolumeInfo{
			{
				VolumeName: volName,
				SourcePVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  sourcePvcName,
					VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
				},
				DestinationPVCInfo: &v1.PersistentVolumeClaimInfo{
					ClaimName:  destPvcName,
					VolumeMode: virtpointer.P(k8sv1.PersistentVolumeFilesystem),
				},
			},
		}
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	},
		Entry("add slices section", `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img' index='1'/>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-datavolumedisk1'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`),
		Entry("slices section already set", `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-private/vmi-disks/datavolumedisk1/disk.img' index='1'>
        <slices>
          <slice type='storage' offset='0' size='2028994560'></slice>
        </slices>
      </source>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-datavolumedisk1'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`),
	)
	It("should generate correct xml for user data for copied disks during the migration", func() {
		domXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type='file' device='disk' model='virtio-non-transitional'>
      <driver name='qemu' type='raw' cache='none' error_policy='stop' discard='unmap'/>
      <source file='/var/run/kubevirt-ephemeral-disks/cloud-init-data/default/vm-dv/noCloud.iso' index='1'/>
      <backingStore/>
      <target dev='vda' bus='virtio'/>
      <alias name='ua-cloudinitdisk'/>
      <address type='pci' domain='0x0000' bus='0x07' slot='0x00' function='0x0'/>
    </disk>
  </devices>
</domain>`
		expectedXML := `<domain type="kvm" id="1">
  <name>kubevirt</name>
  <devices>
    <disk type="file" device="disk" model="virtio-non-transitional">
      <driver name="qemu" type="raw" cache="none" error_policy="stop" discard="unmap"></driver>
      <source file="/var/run/kubevirt-ephemeral-disks/cloud-init-data/default/vm-dv/noCloud.iso" index="1"></source>
      <backingStore></backingStore>
      <target dev="vda" bus="virtio"></target>
      <alias name="ua-cloudinitdisk"></alias>
      <address type="pci" domain="0x0000" bus="0x07" slot="0x00" function="0x0"></address>
    </disk>
  </devices>
</domain>`
		vmi := newVMI("testns", "kubevirt")
		userData := "fake\nuser\ndata\n"
		networkData := "FakeNetwork"
		addCloudInitDisk(vmi, userData, networkData)
		mockLibvirt.DomainEXPECT().GetXMLDesc(libvirt.DOMAIN_XML_MIGRATABLE).MaxTimes(1).Return(domXML, nil)
		domSpec := &api.DomainSpec{}
		Expect(xml.Unmarshal([]byte(domXML), domSpec)).To(Succeed())
		newXML, err := migratableDomXML(mockLibvirt.VirtDomain, vmi, domSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(newXML).To(Equal(expectedXML))
	})
})

var _ = Describe("Manager helper functions", func() {

	Context("getVMIEphemeralDisksTotalSize", func() {

		var tmpDir string
		var zeroQuantity resource.Quantity

		BeforeEach(func() {
			tmpDir = GinkgoT().TempDir()
			zeroQuantity = *resource.NewScaledQuantity(0, 0)
		})

		expectNonZeroQuantity := func(ephemeralDiskDir string) {
			By("Expecting quantity larger than zero")
			quantity := getVMIEphemeralDisksTotalSize(ephemeralDiskDir)

			Expect(quantity).ToNot(BeNil())
			Expect(quantity).ToNot(HaveValue(Equal(zeroQuantity)))
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
			Expect(os.WriteFile(filepath.Join(tmpDir, "testfile"), []byte("file contents"), 0666)).To(Succeed())

			expectNonZeroQuantity(tmpDir)
		})

		It("successful run with zero size", func() {
			By("Creating a file with non-zero size")
			Expect(os.WriteFile(filepath.Join(tmpDir, "testfile"), []byte("file contents"), 0666)).To(Succeed())

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
			fakePercent := v1.Percent(fmt.Sprint(fakePercentFloat))
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
			_, ok := possibleGuestSize(createDisk())
			Expect(ok).To(BeFalse())
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
			Entry("filesystem overhead is invalid float", func() api.Disk {
				disk := properDisk
				badPercent := v1.Percent("3.14") // Must be between 0 and 1
				disk.FilesystemOverhead = &badPercent
				return disk
			}),
			Entry("filesystem overhead is non-float", func() api.Disk {
				disk := properDisk
				fakePercent := v1.Percent("abcdefg")
				disk.FilesystemOverhead = &fakePercent
				return disk
			}),
		)

	})

	Context("configureLocalDiskToMigrate", func() {
		const (
			testvol = "test"
			src     = "src"
			dst     = "dst"
		)

		fsMode := k8sv1.PersistentVolumeFilesystem
		blockMode := k8sv1.PersistentVolumeBlock
		infoFs := v1.PersistentVolumeClaimInfo{
			ClaimName:  src,
			VolumeMode: &fsMode,
		}
		infoBlock := v1.PersistentVolumeClaimInfo{
			ClaimName:  src,
			VolumeMode: &blockMode,
		}

		getFsImagePath := func(name string) string {
			return filepath.Join(hostdisk.GetMountedHostDiskDir(name), "disk.img")
		}

		getBlockPath := func(name string) string {
			return filepath.Join(string(filepath.Separator), "dev", name)
		}

		createDomWithFsImage := func(name string) *libvirtxml.Domain {
			return &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								File: &libvirtxml.DomainDiskSourceFile{
									File: getFsImagePath(name),
								},
							},
							Alias: &libvirtxml.DomainAlias{
								Name: fmt.Sprintf("ua-%s", name),
							},
						},
					},
				},
			}
		}
		createDomWithBlock := func(name string) *libvirtxml.Domain {
			return &libvirtxml.Domain{
				Devices: &libvirtxml.DomainDeviceList{
					Disks: []libvirtxml.DomainDisk{
						{
							Source: &libvirtxml.DomainDiskSource{
								Block: &libvirtxml.DomainDiskSourceBlock{
									Dev: getBlockPath(name),
								},
							},
							Alias: &libvirtxml.DomainAlias{
								Name: fmt.Sprintf("ua-%s", name),
							},
						},
					},
				},
			}
		}
		volPVC := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
						ClaimName: src,
					},
				},
			},
		}
		volDV := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				DataVolume: &v1.DataVolumeSource{
					Name: src,
				},
			},
		}
		volHostDisk := v1.Volume{
			Name: testvol,
			VolumeSource: v1.VolumeSource{
				HostDisk: &v1.HostDisk{
					Path: getFsImagePath(testvol),
				},
			},
		}

		DescribeTable("replace filesystem and block migrated volumes", func(isSrcBlock, isDstBlock bool, vol v1.Volume) {
			retDiskSize := func(disk *libvirtxml.DomainDisk) (int64, error) {
				return 2028994560, nil
			}
			getDiskVirtualSizeFunc = retDiskSize
			var dom *libvirtxml.Domain
			vmi := &v1.VirtualMachineInstance{
				Spec: v1.VirtualMachineInstanceSpec{
					Volumes: []v1.Volume{vol},
				},
				Status: v1.VirtualMachineInstanceStatus{
					MigratedVolumes: []v1.StorageMigratedVolumeInfo{
						{
							VolumeName: testvol,
						},
					},
					VolumeStatus: []v1.VolumeStatus{
						{
							Name: testvol,
							PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
								ClaimName: src,
							},
						},
					},
				},
			}
			if isSrcBlock {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoBlock
				dom = createDomWithBlock(testvol)
			} else {
				vmi.Status.MigratedVolumes[0].SourcePVCInfo = &infoFs
				dom = createDomWithFsImage(testvol)
			}
			if isDstBlock {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoBlock
			} else {
				vmi.Status.MigratedVolumes[0].DestinationPVCInfo = &infoFs
			}

			err := configureLocalDiskToMigrate(dom, vmi)
			Expect(err).ToNot(HaveOccurred())

			if isDstBlock {
				Expect(dom.Devices.Disks[0].Source.File).To(BeNil())
				Expect(dom.Devices.Disks[0].Source.Block).NotTo(BeNil())
				Expect(dom.Devices.Disks[0].Source.Block.Dev).To(Equal(getBlockPath(testvol)))

			} else {
				Expect(dom.Devices.Disks[0].Source.Block).To(BeNil())
				Expect(dom.Devices.Disks[0].Source.File).NotTo(BeNil())
				Expect(dom.Devices.Disks[0].Source.File.File).To(Equal(getFsImagePath(testvol)))
			}
		},
			Entry("filesystem source and destination", false, false, volPVC),
			Entry("filesystem source and block destination", false, true, volPVC),
			Entry("block source and filesystem destination", true, false, volPVC),
			Entry("block source and destination", true, true, volPVC),
			Entry("filesystem source and block destination with DV", false, true, volDV),
			Entry("block source and filesystem destination with DV", true, false, volDV),
			Entry("filesystem source and block destination with hostdisks", false, true, volHostDisk),
			Entry("block source and filesystem destination with hostdisks", true, false, volHostDisk),
		)
	})

	Context("shouldConfigureParallelMigration", func() {
		DescribeTable("should not configure parallel migration", func(options *cmdclient.MigrationOptions) {
			shouldConfigure, _ := shouldConfigureParallelMigration(options)
			Expect(shouldConfigure).To(BeFalse())
		},
			Entry("with nil options", nil),
			Entry("with nil migration threads", &cmdclient.MigrationOptions{ParallelMigrationThreads: nil}),
			Entry("with nil migration threads and post-copy allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: nil, AllowPostCopy: true}),
			Entry("with non-nil migration threads and post-copy allowed", &cmdclient.MigrationOptions{ParallelMigrationThreads: virtpointer.P(uint(3)), AllowPostCopy: true}),
		)

		It("should configure parallel migration with non-nil migration threads and post-copy not allowed", func() {
			options := &cmdclient.MigrationOptions{
				ParallelMigrationThreads: virtpointer.P(uint(3)),
				AllowPostCopy:            false,
			}
			shouldConfigure, _ := shouldConfigureParallelMigration(options)
			Expect(shouldConfigure).To(BeTrue())
		})
	})
})

var _ = Describe("calculateHotplugPortCount", func() {
	const gb = 1024 * 1024 * 1024

	domainWithDevices := func(num int) *api.DomainSpec {
		dom := &api.DomainSpec{}
		for i := 0; i < num; i++ {
			dom.Devices.Disks = append(dom.Devices.Disks, api.Disk{
				Target: api.DiskTarget{
					Bus: v1.DiskBusVirtio,
				},
			})
		}
		return dom
	}

	It("should return 0 when PlacePCIDevicesOnRootComplex is true", func() {
		vmi := newVMI("testns", "kubevirt")
		vmi.Annotations = map[string]string{
			v1.PlacePCIDevicesOnRootComplex: "true",
		}

		count, err := calculateHotplugPortCount(vmi, nil)
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(0))
	})

	DescribeTable("should return the correct port count", func(mem uint64, portsInUse, expectedResult int) {
		vmi := newVMI("testns", "kubevirt")
		domainSpec := domainWithDevices(portsInUse)
		domainSpec.Memory.Value = mem
		count, err := calculateHotplugPortCount(vmi, domainSpec)
		Expect(err).ToNot(HaveOccurred())
		Expect(count).To(Equal(expectedResult))
	},
		Entry("with 1G memory and no ports in use", uint64(1*gb), 0, 8),
		Entry("with 1G memory and 2 ports in use", uint64(1*gb), 2, 6),
		Entry("with 1G memory and 4 ports in use", uint64(1*gb), 4, 4),
		Entry("with 1G memory and 5 ports in use", uint64(1*gb), 5, 3),
		Entry("with 1G memory and 6 ports in use", uint64(1*gb), 6, 3),
		Entry("with 1G memory and 8 ports in use", uint64(1*gb), 8, 3),
		Entry("with 2G memory and 2 ports in use", uint64(2*gb), 2, 6),
		Entry("with 2G memory and 8 ports in use", uint64(2*gb), 8, 3),
		Entry("with 2G+ memory and 2 ports in use", uint64(2*gb+1), 2, 14),
		Entry("with 2G+ memory and 8 ports in use", uint64(2*gb+1), 8, 8),
		Entry("with 3G memory and no ports in use", uint64(3*gb), 0, 16),
		Entry("with 3G memory and 4 ports in use", uint64(3*gb), 4, 12),
		Entry("with 3G memory and 8 ports in use", uint64(3*gb), 8, 8),
		Entry("with 3G memory and 10 ports in use", uint64(3*gb), 10, 6),
		Entry("with 3G memory and 12 ports in use", uint64(3*gb), 12, 6),
		Entry("with 3G memory and 16 ports in use", uint64(3*gb), 16, 6),
	)
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
