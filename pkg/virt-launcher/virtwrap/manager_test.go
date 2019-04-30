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
	"os/user"
	"strings"
	"time"

	"github.com/golang/mock/gomock"
	libvirt "github.com/libvirt/libvirt-go"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	cloudinit "kubevirt.io/kubevirt/pkg/cloud-init"
	"kubevirt.io/kubevirt/pkg/log"
	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/network"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/stats"
)

var _ = Describe("Manager", func() {
	var mockConn *cli.MockConnection
	var mockDomain *cli.MockVirDomain
	var ctrl *gomock.Controller
	testVmName := "testvmi"
	testNamespace := "testnamespace"
	testDomainName := fmt.Sprintf("%s_%s", testNamespace, testVmName)

	log.Log.SetIOWriter(GinkgoWriter)

	tmpDir, _ := ioutil.TempDir("", "cloudinittest")
	owner, err := user.Current()
	if err != nil {
		panic(err)
	}
	isoCreationFunc := func(isoOutFile string, inFiles []string) error {
		_, err := os.Create(isoOutFile)
		return err
	}
	BeforeSuite(func() {
		err := cloudinit.SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		cloudinit.SetLocalDataOwner(owner.Username)
		cloudinit.SetIsoCreationFunction(isoCreationFunc)
	})

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockConn = cli.NewMockConnection(ctrl)
		mockDomain = cli.NewMockVirDomain(ctrl)
	})

	expectIsolationDetectionForVMI := func(vmi *v1.VirtualMachineInstance) *api.DomainSpec {
		domain := &api.Domain{}
		c := &api.ConverterContext{
			VirtualMachine: vmi,
			UseEmulation:   true,
		}
		Expect(api.Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c)).To(Succeed())
		api.SetObjectDefaults_Domain(domain)

		return &domain.Spec
	}

	Context("on successful VirtualMachineInstance sync", func() {
		It("should define and start a new VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			StubOutNetworkForTest()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			domainSpec := expectIsolationDetectionForVMI(vmi)

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should define and start a new VirtualMachineInstance with userData", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			StubOutNetworkForTest()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})

			userData := "fake\nuser\ndata\n"
			networkData := ""
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should define and start a new VirtualMachineInstance with userData and networkData", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			StubOutNetworkForTest()
			vmi := newVMI(testNamespace, testVmName)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, libvirt.Error{Code: libvirt.ERR_NO_DOMAIN})
			userData := "fake\nuser\ndata\n"
			networkData := "FakeNetwork"
			addCloudInitDisk(vmi, userData, networkData)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_SHUTDOWN, 1, nil)
			mockDomain.EXPECT().Create().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		It("should leave a defined and started VirtualMachineInstance alone", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
		})
		table.DescribeTable("should try to start a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				vmi := newVMI(testNamespace, testVmName)
				domainSpec := expectIsolationDetectionForVMI(vmi)
				xml, err := xml.Marshal(domainSpec)

				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().GetState().Return(state, 1, nil)
				mockConn.EXPECT().DomainDefineXML(string(xml)).Return(mockDomain, nil)
				mockDomain.EXPECT().Create().Return(nil)
				mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
				manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
				newspec, err := manager.SyncVMI(vmi, true)
				Expect(err).To(BeNil())
				Expect(newspec).ToNot(BeNil())
			},
			table.Entry("crashed", libvirt.DOMAIN_CRASHED),
			table.Entry("shutdown", libvirt.DOMAIN_SHUTDOWN),
			table.Entry("shutoff", libvirt.DOMAIN_SHUTOFF),
			table.Entry("unknown", libvirt.DOMAIN_NOSTATE),
		)
		It("should resume a paused VirtualMachineInstance", func() {
			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			vmi := newVMI(testNamespace, testVmName)
			domainSpec := expectIsolationDetectionForVMI(vmi)
			xml, err := xml.Marshal(domainSpec)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_PAUSED, 1, nil)
			mockDomain.EXPECT().Resume().Return(nil)
			mockDomain.EXPECT().GetXMLDesc(libvirt.DomainXMLFlags(0)).Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			newspec, err := manager.SyncVMI(vmi, true)
			Expect(err).To(BeNil())
			Expect(newspec).ToNot(BeNil())
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
			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			manager := &LibvirtDomainManager{
				virConn:                mockConn,
				virtShareDir:           "fake",
				notifier:               nil,
				lessPVCSpaceToleration: 0,
			}
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob()
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(string(xml), nil)

			liveMigrationMonitor(vmi, mockDomain, manager, options, migrationErrorChan)
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
			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			manager := &LibvirtDomainManager{
				virConn:                mockConn,
				virtShareDir:           "fake",
				notifier:               nil,
				lessPVCSpaceToleration: 0,
			}
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob()
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(string(xml), nil)

			liveMigrationMonitor(vmi, mockDomain, manager, options, migrationErrorChan)
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
			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			mockDomain.EXPECT().AbortJob().MaxTimes(1)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).AnyTimes().Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
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

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)
			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).AnyTimes().Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).AnyTimes().Return(string(xml), nil)
			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			err = manager.CancelVMIMigration(vmi)
			Expect(err).To(BeNil())
		})

	})

	Context("on successful VirtualMachineInstance migrate", func() {
		It("should prepare the target pod", func() {
			updateHostsFile = func(entry string) error {
				return nil
			}
			StubOutNetworkForTest()
			vmi := newVMI(testNamespace, testVmName)
			vmi.Status.MigrationState = &v1.VirtualMachineInstanceMigrationState{
				MigrationUID: "111222333",
				TargetPod:    "fakepod",
			}

			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
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

			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)

			mockConn.EXPECT().LookupDomainByName(testDomainName).AnyTimes().Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().AnyTimes().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())
			mockDomain.EXPECT().GetJobInfo().AnyTimes().Return(fake_jobinfo, nil)
			gomock.InOrder(
				mockConn.EXPECT().DomainDefineXML(gomock.Any()).Return(mockDomain, nil),
				mockConn.EXPECT().DomainDefineXML(gomock.Any()).DoAndReturn(func(xml string) (cli.VirDomain, error) {
					Expect(strings.Contains(xml, "MigrationFailed")).To(BeTrue())
					isMigrationFailedSet <- true
					return mockDomain, nil
				}),
			)
			mockDomain.EXPECT().GetXMLDesc(gomock.Any()).AnyTimes().Return(string(xml), nil)
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

			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)

			mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
			mockDomain.EXPECT().GetState().Return(libvirt.DOMAIN_RUNNING, 1, nil)

			xml, err := xml.Marshal(domainSpec)
			Expect(err).To(BeNil())

			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).Return(string(xml), nil)
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(string(xml), nil)
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

	})

	Context("on successful VirtualMachineInstance kill", func() {
		table.DescribeTable("should try to undefine a VirtualMachineInstance in state",
			func(state libvirt.DomainState) {
				// Make sure that we always free the domain after use
				mockDomain.EXPECT().Free()
				mockConn.EXPECT().LookupDomainByName(testDomainName).Return(mockDomain, nil)
				mockDomain.EXPECT().UndefineFlags(libvirt.DOMAIN_UNDEFINE_NVRAM).Return(nil)
				manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
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
				manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
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
			flags := prepareMigrationFlags(isBlockMigration, isUnsafeMigration)
			expectedMigrateFlags := libvirt.MIGRATE_LIVE | libvirt.MIGRATE_PEER2PEER

			if isBlockMigration {
				expectedMigrateFlags |= libvirt.MIGRATE_NON_SHARED_INC
			} else if migrationType == "unsafe" {
				expectedMigrateFlags |= libvirt.MIGRATE_UNSAFE
			}
			Expect(flags).To(Equal(expectedMigrateFlags))
		},
		table.Entry("with block migration", "block"),
		table.Entry("without block migration", "live"),
		table.Entry("unsafe migration", "unsafe"),
	)

	table.DescribeTable("on successful list all domains",
		func(state libvirt.DomainState, kubevirtState api.LifeCycle, libvirtReason int, kubevirtReason api.StateChangeReason) {

			// Make sure that we always free the domain after use
			mockDomain.EXPECT().Free()
			mockDomain.EXPECT().GetState().Return(state, libvirtReason, nil).AnyTimes()
			mockDomain.EXPECT().GetName().Return("test", nil)
			x, err := xml.Marshal(api.NewMinimalDomainSpec("test"))
			Expect(err).To(BeNil())
			if !cli.IsDown(state) {
				mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_MIGRATABLE)).Return(string(x), nil)
			}
			mockDomain.EXPECT().GetXMLDesc(gomock.Eq(libvirt.DOMAIN_XML_INACTIVE)).Return(string(x), nil)
			mockConn.EXPECT().ListAllDomains(gomock.Eq(libvirt.CONNECT_LIST_DOMAINS_ACTIVE|libvirt.CONNECT_LIST_DOMAINS_INACTIVE)).Return([]cli.VirDomain{mockDomain}, nil)

			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			doms, err := manager.ListAllDomains()

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
				gomock.Eq(libvirt.DOMAIN_STATS_CPU_TOTAL|libvirt.DOMAIN_STATS_VCPU|libvirt.DOMAIN_STATS_INTERFACE|libvirt.DOMAIN_STATS_BLOCK),
				gomock.Eq(libvirt.CONNECT_GET_ALL_DOMAINS_STATS_RUNNING),
			).Return([]*stats.DomainStats{
				&stats.DomainStats{},
			}, nil)

			manager, _ := NewLibvirtDomainManager(mockConn, "fake", nil, 0)
			domStats, err := manager.GetDomainStats()

			Expect(err).To(BeNil())
			Expect(len(domStats)).To(Equal(1))
		})
	})

	// TODO: test error reporting on non successful VirtualMachineInstance syncs and kill attempts

	AfterEach(func() {
		ctrl.Finish()
	})
})

var _ = Describe("resourceNameToEnvvar", func() {
	It("handles resource name with dots and slashes", func() {
		Expect(resourceNameToEnvvar("intel.com/sriov_test")).To(Equal("PCIDEVICE_INTEL_COM_SRIOV_TEST"))
	})
})

var _ = Describe("getSRIOVPCIAddresses", func() {
	getSRIOVInterfaceList := func() []v1.Interface {
		return []v1.Interface{
			v1.Interface{
				Name: "testnet",
				InterfaceBindingMethod: v1.InterfaceBindingMethod{
					SRIOV: &v1.InterfaceSRIOV{},
				},
			},
		}
	}

	It("returns empty map when empty interfaces", func() {
		Expect(len(getSRIOVPCIAddresses([]v1.Interface{}))).To(Equal(0))
	})
	It("returns empty map when interface is not sriov", func() {
		ifaces := []v1.Interface{v1.Interface{Name: "testnet"}}
		Expect(len(getSRIOVPCIAddresses(ifaces))).To(Equal(0))
	})
	It("returns map with empty device id list when variables are not set", func() {
		addrs := getSRIOVPCIAddresses(getSRIOVInterfaceList())
		Expect(len(addrs)).To(Equal(1))
		Expect(len(addrs["testnet"])).To(Equal(0))
	})
	It("gracefully handles a single address value", func() {
		os.Setenv("PCIDEVICE_INTEL_COM_TESTNET_POOL", "0000:81:11.1")
		os.Setenv("KUBEVIRT_RESOURCE_NAME_testnet", "intel.com/testnet_pool")
		addrs := getSRIOVPCIAddresses(getSRIOVInterfaceList())
		Expect(len(addrs)).To(Equal(1))
		Expect(addrs["testnet"][0]).To(Equal("0000:81:11.1"))
	})
	It("returns multiple PCI addresses", func() {
		os.Setenv("PCIDEVICE_INTEL_COM_TESTNET_POOL", "0000:81:11.1,0001:02:00.0")
		os.Setenv("KUBEVIRT_RESOURCE_NAME_testnet", "intel.com/testnet_pool")
		addrs := getSRIOVPCIAddresses(getSRIOVInterfaceList())
		Expect(len(addrs["testnet"])).To(Equal(2))
		Expect(addrs["testnet"][0]).To(Equal("0000:81:11.1"))
		Expect(addrs["testnet"][1]).To(Equal("0001:02:00.0"))
	})
})

func newVMI(namespace string, name string) *v1.VirtualMachineInstance {
	vmi := &v1.VirtualMachineInstance{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       v1.VirtualMachineInstanceSpec{Domain: v1.NewMinimalDomainSpec()},
	}
	v1.SetObjectDefaults_VirtualMachineInstance(vmi)
	return vmi
}

func StubOutNetworkForTest() {
	network.SetupPodNetwork = func(vm *v1.VirtualMachineInstance, domain *api.Domain) error { return nil }
}

func addCloudInitDisk(vmi *v1.VirtualMachineInstance, userData string, networkData string) {
	vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
		Name:  "cloudinit",
		Cache: v1.CacheNone,
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
