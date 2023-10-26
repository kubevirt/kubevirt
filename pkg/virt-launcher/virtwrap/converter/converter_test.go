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
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package converter

import (
	_ "embed"
	"encoding/xml"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/vcpu"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"kubevirt.io/kubevirt/pkg/virt-api/webhooks"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"

	"kubevirt.io/kubevirt/pkg/downwardmetrics"
	"kubevirt.io/kubevirt/pkg/ephemeral-disk/fake"
	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/api/core/v1"
	kvapi "kubevirt.io/client-go/api"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	kubevirtpointer "kubevirt.io/kubevirt/pkg/pointer"
	sev "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/launchsecurity"
)

var (
	//go:embed testdata/domain_x86_64.xml.tmpl
	embedDomainTemplateX86_64 string
	//go:embed testdata/domain_ppc64le.xml.tmpl
	embedDomainTemplatePPC64le string
	//go:embed testdata/domain_arm64.xml.tmpl
	embedDomainTemplateARM64 string
	//go:embed testdata/domain_x86_64_root.xml.tmpl
	embedDomainTemplateRootBus string
)

const (
	argNoMemBalloon      = `<memballoon model="none"></memballoon>`
	argMemBalloon0period = `<memballoon model="virtio-non-transitional" freePageReporting="on"></memballoon>`
	argMemBalloon5period = `<memballoon model="virtio-non-transitional" freePageReporting="on">
      <stats period="5"></stats>
    </memballoon>`
	argMemBalloon10period = `<memballoon model="virtio-non-transitional" freePageReporting="on">
      <stats period="10"></stats>
    </memballoon>`
)

var _ = Describe("getOptimalBlockIO", func() {

	It("Should detect disk block sizes for a file DiskSource", func() {
		disk := &api.Disk{
			Source: api.DiskSource{
				File: "/",
			},
		}
		blockIO, err := getOptimalBlockIO(disk)
		Expect(err).ToNot(HaveOccurred())
		Expect(blockIO.LogicalBlockSize).To(Equal(blockIO.PhysicalBlockSize))
		// The default for most filesystems nowadays is 4096 but it can be changed.
		// As such, relying on a specific value is flakey unless
		// we create a disk image and filesystem just for this test.
		// For now, as long as we have a value, the exact value doesn't matter.
		Expect(blockIO.LogicalBlockSize).ToNot(BeZero())
	})

	It("Should fail for non-file or non-block devices", func() {
		disk := &api.Disk{
			Source: api.DiskSource{},
		}
		_, err := getOptimalBlockIO(disk)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Converter", func() {

	TestSmbios := &cmdv1.SMBios{}
	EphemeralDiskImageCreator := &fake.MockEphemeralDiskImageCreator{BaseDir: "/var/run/libvirt/kubevirt-ephemeral-disk/"}

	Context("with timezone", func() {
		It("Should set timezone attribute", func() {
			timezone := v1.ClockOffsetTimezone("America/New_York")
			clock := &v1.Clock{
				ClockOffset: v1.ClockOffset{
					Timezone: &timezone,
				},
				Timer: &v1.Timer{},
			}

			var convertClock api.Clock
			Convert_v1_Clock_To_api_Clock(clock, &convertClock)
			data, err := xml.MarshalIndent(convertClock, "", "  ")
			Expect(err).ToNot(HaveOccurred())

			expectedClock := `<Clock offset="timezone" timezone="America/New_York"></Clock>`
			Expect(string(data)).To(Equal(expectedClock))
		})
	})

	Context("with v1.Disk", func() {
		DescribeTable("Should define disk capacity as the minimum of capacity and request", func(requests, capacity int64) {
			context := &ConverterContext{}
			v1Disk := v1.Disk{
				Name: "myvolume",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{Bus: v1.VirtIO},
				},
			}
			apiDisk := api.Disk{}
			devicePerBus := map[string]deviceNamer{}
			numQueues := uint(2)
			volumeStatusMap := make(map[string]v1.VolumeStatus)
			volumeStatusMap["myvolume"] = v1.VolumeStatus{
				PersistentVolumeClaimInfo: &v1.PersistentVolumeClaimInfo{
					Capacity: k8sv1.ResourceList{
						k8sv1.ResourceStorage: *resource.NewQuantity(capacity, resource.DecimalSI),
					},
					Requests: k8sv1.ResourceList{
						k8sv1.ResourceStorage: *resource.NewQuantity(requests, resource.DecimalSI),
					},
				},
			}
			Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues, volumeStatusMap)
			Expect(apiDisk.Capacity).ToNot(BeNil())
			Expect(*apiDisk.Capacity).To(Equal(min(capacity, requests)))
		},
			Entry("Higher request than capacity", int64(9999), int64(1111)),
			Entry("Lower request than capacity", int64(1111), int64(9999)),
		)

		DescribeTable("Should assign scsi controller to", func(diskDevice v1.DiskDevice) {
			context := &ConverterContext{}
			v1Disk := v1.Disk{
				Name:       "myvolume",
				DiskDevice: diskDevice,
			}
			apiDisk := api.Disk{}
			devicePerBus := map[string]deviceNamer{}
			numQueues := uint(2)
			volumeStatusMap := make(map[string]v1.VolumeStatus)
			volumeStatusMap["myvolume"] = v1.VolumeStatus{}
			Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues, volumeStatusMap)
			Expect(apiDisk.Address).ToNot(BeNil())
			Expect(apiDisk.Address.Bus).To(Equal("0"))
			Expect(apiDisk.Address.Controller).To(Equal("0"))
			Expect(apiDisk.Address.Type).To(Equal("drive"))
			Expect(apiDisk.Address.Unit).To(Equal("0"))
		},
			Entry("LUN-type disk", v1.DiskDevice{
				LUN: &v1.LunTarget{Bus: "scsi"},
			}),
			Entry("Disk-type disk", v1.DiskDevice{
				Disk: &v1.DiskTarget{Bus: "scsi"},
			}),
		)

		It("Should add boot order when provided", func() {
			order := uint(1)
			kubevirtDisk := &v1.Disk{
				Name:      "mydisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.VirtIO,
					},
				},
			}
			var convertedDisk = `<Disk device="disk" type="" model="virtio-non-transitional">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
  <boot order="1"></boot>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			Expect(xml).To(Equal(convertedDisk))
		})

		It("should set disk I/O mode if requested", func() {
			v1Disk := &v1.Disk{
				IO: "native",
			}
			xml := diskToDiskXML(v1Disk)
			expectedXML := `<Disk device="" type="">
  <source></source>
  <target></target>
  <driver io="native" name="qemu" type=""></driver>
  <alias name="ua-"></alias>
</Disk>`
			Expect(xml).To(Equal(expectedXML))
		})

		It("should not set disk I/O mode if not requested", func() {
			v1Disk := &v1.Disk{}
			xml := diskToDiskXML(v1Disk)
			expectedXML := `<Disk device="" type="">
  <source></source>
  <target></target>
  <driver name="qemu" type=""></driver>
  <alias name="ua-"></alias>
</Disk>`
			Expect(xml).To(Equal(expectedXML))
		})

		It("Should omit boot order when not provided", func() {
			kubevirtDisk := &v1.Disk{
				Name: "mydisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.VirtIO,
					},
				},
			}
			var convertedDisk = `<Disk device="disk" type="" model="virtio-non-transitional">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			Expect(xml).To(Equal(convertedDisk))
		})

		It("Should add blockio fields when custom sizes are provided", func() {
			kubevirtDisk := &v1.Disk{
				BlockSize: &v1.BlockSize{
					Custom: &v1.CustomBlockSize{
						Logical:  1234,
						Physical: 1234,
					},
				},
			}
			expectedXML := `<Disk device="" type="">
  <source></source>
  <target></target>
  <blockio logical_block_size="1234" physical_block_size="1234"></blockio>
</Disk>`
			libvirtDisk := &api.Disk{}
			Expect(Convert_v1_BlockSize_To_api_BlockIO(kubevirtDisk, libvirtDisk)).To(Succeed())
			data, err := xml.MarshalIndent(libvirtDisk, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			xml := string(data)
			Expect(xml).To(Equal(expectedXML))
		})
		It("should set sharable and the cache if requested", func() {
			v1Disk := &v1.Disk{
				Name: "mydisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.VirtIO,
					},
				},
				Shareable: True(),
			}
			var expectedXML = `<Disk device="disk" type="" model="virtio-non-transitional">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver cache="none" name="qemu" type="" discard="unmap"></driver>
  <alias name="ua-mydisk"></alias>
  <shareable></shareable>
</Disk>`
			xml := diskToDiskXML(v1Disk)
			Expect(xml).To(Equal(expectedXML))
		})
	})

	Context("with v1.VirtualMachineInstance", func() {

		var vmi *v1.VirtualMachineInstance
		domainType := "kvm"
		if _, err := os.Stat("/dev/kvm"); errors.Is(err, os.ErrNotExist) {
			domainType = "qemu"
		}

		BeforeEach(func() {

			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Watchdog = &v1.Watchdog{
				Name: "mywatchdog",
				WatchdogDevice: v1.WatchdogDevice{
					I6300ESB: &v1.I6300ESBWatchdog{
						Action: v1.WatchdogActionPoweroff,
					},
				},
			}
			vmi.Spec.Domain.Clock = &v1.Clock{
				ClockOffset: v1.ClockOffset{
					UTC: &v1.ClockOffsetUTC{},
				},
				Timer: &v1.Timer{
					HPET: &v1.HPETTimer{
						Enabled:    False(),
						TickPolicy: v1.HPETTickPolicyDelay,
					},
					KVM: &v1.KVMTimer{
						Enabled: True(),
					},
					PIT: &v1.PITTimer{
						Enabled:    False(),
						TickPolicy: v1.PITTickPolicyDiscard,
					},
					RTC: &v1.RTCTimer{
						Enabled:    True(),
						TickPolicy: v1.RTCTickPolicyCatchup,
						Track:      v1.TrackGuest,
					},
					Hyperv: &v1.HypervTimer{
						Enabled: True(),
					},
				},
			}
			vmi.Spec.Domain.Features = &v1.Features{
				APIC:       &v1.FeatureAPIC{},
				SMM:        &v1.FeatureState{},
				KVM:        &v1.FeatureKVM{Hidden: true},
				Pvspinlock: &v1.FeatureState{Enabled: False()},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:         &v1.FeatureState{Enabled: False()},
					VAPIC:           &v1.FeatureState{Enabled: True()},
					Spinlocks:       &v1.FeatureSpinlocks{Enabled: True()},
					VPIndex:         &v1.FeatureState{Enabled: True()},
					Runtime:         &v1.FeatureState{Enabled: False()},
					SyNIC:           &v1.FeatureState{Enabled: True()},
					SyNICTimer:      &v1.SyNICTimer{Enabled: True(), Direct: &v1.FeatureState{Enabled: True()}},
					Reset:           &v1.FeatureState{Enabled: True()},
					VendorID:        &v1.FeatureVendorID{Enabled: False(), VendorID: "myvendor"},
					Frequencies:     &v1.FeatureState{Enabled: False()},
					Reenlightenment: &v1.FeatureState{Enabled: False()},
					TLBFlush:        &v1.FeatureState{Enabled: True()},
					IPI:             &v1.FeatureState{Enabled: True()},
					EVMCS:           &v1.FeatureState{Enabled: False()},
				},
			}
			vmi.Spec.Domain.Resources.Limits = make(k8sv1.ResourceList)
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{k8sv1.ResourceMemory: resource.MustParse("8192Ki")}
			vmi.Spec.Domain.Devices.DisableHotplug = true
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{
				{
					Bus:  v1.VirtIO,
					Type: "tablet",
					Name: "tablet0",
				},
			}
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.VirtIO,
						},
					},
					DedicatedIOThread: True(),
				},
				{
					Name: "nocloud",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.VirtIO,
						},
					},
					DedicatedIOThread: True(),
				},
				{
					Name: "cdrom_tray_unspecified",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							ReadOnly: False(),
						},
					},
					DedicatedIOThread: False(),
				},
				{
					Name: "cdrom_tray_open",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Tray: v1.TrayStateOpen,
						},
					},
				},
				{
					Name: "should_default_to_disk",
				},
				{
					Name:  "ephemeral_pvc",
					Cache: "none",
				},
				{
					Name:   "secret_test",
					Serial: "D23YZ9W6WA5DJ487",
				},
				{
					Name:   "configmap_test",
					Serial: "CVLY623300HK240D",
				},
				{
					Name:  "pvc_block_test",
					Cache: "writethrough",
				},
				{
					Name:  "dv_block_test",
					Cache: "writethrough",
				},
				{
					Name: "serviceaccount_test",
				},
				{
					Name: "sysprep",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							ReadOnly: False(),
						},
					},
					DedicatedIOThread: False(),
				},
				{
					Name: "sysprep_secret",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							ReadOnly: False(),
						},
					},
					DedicatedIOThread: False(),
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/myvolume/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "nocloud",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserDataBase64:    "1234",
							NetworkDataBase64: "1234",
						},
					},
				},
				{
					Name: "cdrom_tray_unspecified",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserDataBase64:    "1234",
							NetworkDataBase64: "1234",
						},
					},
				},
				{
					Name: "cdrom_tray_open",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume1/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "should_default_to_disk",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume4/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "ephemeral_pvc",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
				{
					Name: "secret_test",
					VolumeSource: v1.VolumeSource{
						Secret: &v1.SecretVolumeSource{
							SecretName: "testsecret",
						},
					},
				},
				{
					Name: "configmap_test",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: k8sv1.LocalObjectReference{
								Name: "testconfig",
							},
						},
					},
				},
				{
					Name: "pvc_block_test",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{PersistentVolumeClaimVolumeSource: k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						}},
					},
				},
				{
					Name: "dv_block_test",
					VolumeSource: v1.VolumeSource{
						DataVolume: &v1.DataVolumeSource{
							Name: "dv_block_test",
						},
					},
				},
				{
					Name: "serviceaccount_test",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: "testaccount",
						},
					},
				},
				{
					Name: "sysprep",
					VolumeSource: v1.VolumeSource{
						Sysprep: &v1.SysprepSource{
							ConfigMap: &k8sv1.LocalObjectReference{
								Name: "testconfig",
							},
						},
					},
				},
				{
					Name: "sysprep_secret",
					VolumeSource: v1.VolumeSource{
						Sysprep: &v1.SysprepSource{
							Secret: &k8sv1.LocalObjectReference{
								Name: "testsecret",
							},
						},
					},
				},
			}

			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				UUID:   "e4686d2c-6e8d-4335-b8fd-81bee22f4814",
				Serial: "e4686d2c-6e8d-4335-b8fd-81bee22f4815",
			}

			gracePerod := int64(5)
			vmi.Spec.TerminationGracePeriodSeconds = &gracePerod

			vmi.ObjectMeta.UID = "f4686d2c-6e8d-4335-b8fd-81bee22f4814"
		})

		var convertedDomain = strings.TrimSpace(fmt.Sprintf(embedDomainTemplateX86_64, domainType, "%s"))
		var convertedDomainWith5Period = fmt.Sprintf(convertedDomain, argMemBalloon5period)
		var convertedDomainWith0Period = fmt.Sprintf(convertedDomain, argMemBalloon0period)
		var convertedDomainWithFalseAutoattach = fmt.Sprintf(convertedDomain, argNoMemBalloon)

		convertedDomain = fmt.Sprintf(convertedDomain, argMemBalloon10period)

		var convertedDomainppc64le = strings.TrimSpace(fmt.Sprintf(embedDomainTemplatePPC64le, domainType, "%s"))
		var convertedDomainppc64leWith5Period = fmt.Sprintf(convertedDomainppc64le, argMemBalloon5period)
		var convertedDomainppc64leWith0Period = fmt.Sprintf(convertedDomainppc64le, argMemBalloon0period)
		var convertedDomainppc64leWithFalseAutoattach = fmt.Sprintf(convertedDomainppc64le, argNoMemBalloon)

		convertedDomainppc64le = fmt.Sprintf(convertedDomainppc64le, argMemBalloon10period)

		var convertedDomainarm64 = strings.TrimSpace(fmt.Sprintf(embedDomainTemplateARM64, domainType, "%s"))
		var convertedDomainarm64With5Period = fmt.Sprintf(convertedDomainarm64, argMemBalloon5period)
		var convertedDomainarm64With0Period = fmt.Sprintf(convertedDomainarm64, argMemBalloon0period)
		var convertedDomainarm64WithFalseAutoattach = fmt.Sprintf(convertedDomainarm64, argNoMemBalloon)

		convertedDomainarm64 = fmt.Sprintf(convertedDomainarm64, argMemBalloon10period)

		var convertedDomainWithDevicesOnRootBus = strings.TrimSpace(fmt.Sprintf(embedDomainTemplateRootBus, domainType))

		var c *ConverterContext

		isBlockPVCMap := make(map[string]bool)
		isBlockPVCMap["pvc_block_test"] = true
		isBlockDVMap := make(map[string]bool)
		isBlockDVMap["dv_block_test"] = true

		BeforeEach(func() {
			c = &ConverterContext{
				VirtualMachine: vmi,
				Secrets: map[string]*k8sv1.Secret{
					"mysecret": {
						Data: map[string][]byte{
							"node.session.auth.username": []byte("admin"),
						},
					},
				},
				AllowEmulation:        true,
				IsBlockPVC:            isBlockPVCMap,
				IsBlockDV:             isBlockDVMap,
				SMBios:                TestSmbios,
				MemBalloonStatsPeriod: 10,
				EphemeraldiskCreator:  EphemeralDiskImageCreator,
				FreePageReporting:     true,
				SerialConsoleLog:      true,
			}
		})

		It("should use virtio-transitional models if requested", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.DisableHotplug = false
			c.UseVirtioTransitional = true
			vmi.Spec.Domain.Devices.UseVirtioTransitional = &c.UseVirtioTransitional
			dom := vmiToDomain(vmi, c)
			testutils.ExpectVirtioTransitionalOnly(&dom.Spec)
		})

		It("should handle float memory", func() {
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory] = resource.MustParse("2222222200m")
			xml := vmiToDomainXML(vmi, c)
			Expect(strings.Contains(xml, `<memory unit="b">2222222</memory>`)).To(BeTrue(), xml)
		})

		DescribeTable("should be converted to a libvirt Domain with vmi defaults set", func(arch string, domain string) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			c.Architecture = arch
			vmiArchMutate(arch, vmi, c)
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			Entry("for amd64", "amd64", convertedDomain),
			Entry("for ppc64le", "ppc64le", convertedDomainppc64le),
			Entry("for arm64", "arm64", convertedDomainarm64),
		)

		DescribeTable("should be converted to a libvirt Domain", func(arch string, domain string, period uint) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			c.Architecture = arch
			vmiArchMutate(arch, vmi, c)
			c.MemBalloonStatsPeriod = period
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			Entry("when context define 5 period on memballoon device for amd64", "amd64", convertedDomainWith5Period, uint(5)),
			Entry("when context define 5 period on memballoon device for ppc64le", "ppc64le", convertedDomainppc64leWith5Period, uint(5)),
			Entry("when context define 5 period on memballoon device for arm64", "arm64", convertedDomainarm64With5Period, uint(5)),
			Entry("when context define 0 period on memballoon device for amd64 ", "amd64", convertedDomainWith0Period, uint(0)),
			Entry("when context define 0 period on memballoon device for ppc64le", "ppc64le", convertedDomainppc64leWith0Period, uint(0)),
			Entry("when context define 0 period on memballoon device for arm64", "arm64", convertedDomainarm64With0Period, uint(0)),
		)

		DescribeTable("should be converted to a libvirt Domain", func(arch string, domain string) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = False()
			c.Architecture = arch
			vmiArchMutate(arch, vmi, c)
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			Entry("when Autoattach memballoon device is false for amd64", "amd64", convertedDomainWithFalseAutoattach),
			Entry("when Autoattach memballoon device is false for ppc64le", "ppc64le", convertedDomainppc64leWithFalseAutoattach),
			Entry("when Autoattach memballoon device is false for arm64", "arm64", convertedDomainarm64WithFalseAutoattach),
		)

		It("should use kvm if present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			Expect(vmiToDomainXMLToDomainSpec(vmi, c).Type).To(Equal(domainType))
		})

		Context("when all addresses should be places at the root complex", func() {
			It("should be converted to a libvirt Domain with vmi defaults set", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
				c.Architecture = "amd64"
				vmiArchMutate("amd64", vmi, c)
				spec := vmiToDomain(vmi, c).Spec.DeepCopy()
				Expect(PlacePCIDevicesOnRootComplex(spec)).To(Succeed())
				data, err := xml.MarshalIndent(spec, "", "  ")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(data)).To(Equal(convertedDomainWithDevicesOnRootBus))
			})
		})

		Context("when CPU spec defined", func() {
			It("should convert CPU cores, model and features", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:   3,
					Sockets: 2,
					Threads: 2,
					Model:   "Conroe",
					Features: []v1.CPUFeature{
						{
							Name:   "lahf_lm",
							Policy: "require",
						},
						{
							Name:   "mmx",
							Policy: "disable",
						},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(3)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(2)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(2)), "Expect threads")
				Expect(domainSpec.CPU.Mode).To(Equal("custom"), "Expect cpu mode")
				Expect(domainSpec.CPU.Model).To(Equal("Conroe"), "Expect cpu model")
				Expect(domainSpec.CPU.Features[0].Name).To(Equal("lahf_lm"), "Expect cpu feature name")
				Expect(domainSpec.CPU.Features[0].Policy).To(Equal("require"), "Expect cpu feature policy")
				Expect(domainSpec.CPU.Features[1].Name).To(Equal("mmx"), "Expect cpu feature name")
				Expect(domainSpec.CPU.Features[1].Policy).To(Equal("disable"), "Expect cpu feature policy")
				Expect(domainSpec.VCPU.Placement).To(Equal("static"), "Expect vcpu placement")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(12)), "Expect vcpus")
			})

			It("should convert CPU cores", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(3)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(1)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should convert CPU sockets", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 3,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should convert CPU threads", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Threads: 3,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(1)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(3)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should convert CPU requests to sockets", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("2200m")
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should convert CPU limits to sockets", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = nil
				vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("2.3")
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should prefer CPU spec instead of CPU requests", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 3,
				}
				vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("400m")
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should prefer CPU spec instead of CPU limits", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Sockets: 3,
				}
				vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceCPU] = resource.MustParse("400m")
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(1)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)), "Expect vcpus")
			})

			It("should define hotplugable default topology", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores:      2,
					MaxSockets: 3,
					Sockets:    2,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(2)), "Expect cores")
				Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(3)), "Expect sockets")
				Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)), "Expect threads")
				Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(6)), "Expect vcpus")
				Expect(domainSpec.VCPUs).ToNot(BeNil(), "Expecting topology for hotplug")
				Expect(domainSpec.VCPUs.VCPU).To(HaveLen(6), "Expecting topology for hotplug")
				Expect(domainSpec.VCPUs.VCPU[0].Hotpluggable).To(Equal("no"), "Expecting the 1st vcpu to be stable")
				Expect(domainSpec.VCPUs.VCPU[1].Hotpluggable).To(Equal("no"), "Expecting the 2nd vcpu to be stable")
				Expect(domainSpec.VCPUs.VCPU[2].Hotpluggable).To(Equal("yes"), "Expecting the 3rd vcpu to be Hotpluggable")
				Expect(domainSpec.VCPUs.VCPU[3].Hotpluggable).To(Equal("yes"), "Expecting the 4th vcpu to be Hotpluggable")
				Expect(domainSpec.VCPUs.VCPU[4].Hotpluggable).To(Equal("yes"), "Expecting the 5th vcpu to be Hotpluggable")
				Expect(domainSpec.VCPUs.VCPU[5].Hotpluggable).To(Equal("yes"), "Expecting the 6th vcpu to be Hotpluggable")
				Expect(domainSpec.VCPUs.VCPU[0].Enabled).To(Equal("yes"), "Expecting the 1st vcpu to be enabled")
				Expect(domainSpec.VCPUs.VCPU[1].Enabled).To(Equal("yes"), "Expecting the 2nd vcpu to be enabled")
				Expect(domainSpec.VCPUs.VCPU[2].Enabled).To(Equal("yes"), "Expecting the 3rd vcpu to be enabled")
				Expect(domainSpec.VCPUs.VCPU[3].Enabled).To(Equal("yes"), "Expecting the 4th vcpu to be enabled")
				Expect(domainSpec.VCPUs.VCPU[4].Enabled).To(Equal("no"), "Expecting the 5th vcpu to be disabled")
				Expect(domainSpec.VCPUs.VCPU[5].Enabled).To(Equal("no"), "Expecting the 6th vcpu to be disabled")
			})

			DescribeTable("should convert CPU model", func(model string) {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
					Model: model,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Mode).To(Equal(model), "Expect mode")
			},
				Entry(v1.CPUModeHostPassthrough, v1.CPUModeHostPassthrough),
				Entry(v1.CPUModeHostModel, v1.CPUModeHostModel),
			)
		})

		Context("when CPU spec defined and model not", func() {
			It("should set host-model CPU mode", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Mode).To(Equal("host-model"))
			})
		})

		Context("when CPU spec not defined", func() {
			It("should set host-model CPU mode", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Mode).To(Equal("host-model"))
			})
		})

		Context("when downwardMetrics are exposed via virtio-serial", func() {
			It("should set socket options", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.DownwardMetrics = &v1.DownwardMetrics{}
				domain := vmiToDomain(vmi, c)

				Expect(domain.Spec.Devices.Channels).To(ContainElement(
					api.Channel{
						Type: "unix",
						Source: &api.ChannelSource{
							Mode: "bind",
							Path: downwardmetrics.DownwardMetricsChannelSocket,
						},
						Target: &api.ChannelTarget{
							Type: v1.VirtIO,
							Name: downwardmetrics.DownwardMetricsSerialDeviceName,
						},
					}))
			})
		})

		It("should set disk pci address when specified", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:81:01.0"
			test_address := api.Address{
				Type:     api.AddressPCI,
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x01",
				Function: "0x0",
			}
			domain := vmiToDomain(vmi, c)
			Expect(*domain.Spec.Devices.Disks[0].Address).To(Equal(test_address))
		})

		It("should fail disk config pci address is set with a non virtio bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:81:01.0"
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "scsi"
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed())
		})

		It("should succeed with SCSI reservation", func() {
			name := "scsi-reservation"
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: name,
					DiskDevice: v1.DiskDevice{
						Disk: nil,
						LUN: &v1.LunTarget{
							Bus:         "scsi",
							Reservation: true,
						},
					},
				}}
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: name,
				VolumeSource: v1.VolumeSource{
					Ephemeral: &v1.EphemeralVolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: name,
						},
					},
				},
			})
			c.DisksInfo = make(map[string]*cmdv1.DiskInfo)
			c.DisksInfo[name] = &cmdv1.DiskInfo{}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			reserv := domainSpec.Devices.Disks[0].Source.Reservations
			Expect(reserv.Managed).To(Equal("no"))
			Expect(reserv.SourceReservations.Type).To(Equal("unix"))
			Expect(reserv.SourceReservations.Path).To(Equal("/var/run/kubevirt/daemons/pr/pr-helper.sock"))
			Expect(reserv.SourceReservations.Mode).To(Equal("client"))
		})

		It("should add a virtio-scsi controller if a scsci disk is present and iothreads set", func() {
			one := uint(1)
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "scsi"
			dom := &api.Domain{}
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, dom, c)).To(Succeed())
			Expect(dom.Spec.Devices.Controllers).To(ContainElement(api.Controller{
				Type:  "scsi",
				Index: "0",
				Model: "virtio-non-transitional",
				Driver: &api.ControllerDriver{
					IOThread: &one,
					Queues:   &one,
				},
			}))
		})

		It("should add a virtio-scsi controller if a scsci disk is present and iothreads NOT set", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "scsi"
			vmi.Spec.Domain.IOThreadsPolicy = nil
			for i, _ := range vmi.Spec.Domain.Devices.Disks {
				vmi.Spec.Domain.Devices.Disks[i].DedicatedIOThread = nil
			}
			dom := &api.Domain{}
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, dom, c)).To(Succeed())
			Expect(dom.Spec.Devices.Controllers).To(ContainElement(api.Controller{
				Type:  "scsi",
				Index: "0",
				Model: "virtio-non-transitional",
			}))
		})

		It("should not add a virtio-scsi controller if no scsi disk is present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "sata"
			dom := &api.Domain{}
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, dom, c)).To(Succeed())
			Expect(dom.Spec.Devices.Controllers).ToNot(ContainElement(api.Controller{
				Type:  "scsi",
				Index: "0",
				Model: "virtio-non-transitional",
			}))
		})

		It("should not disable usb controller when usb device is present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "usb"
			domain := vmiToDomain(vmi, c)
			disabled := false
			for _, controller := range domain.Spec.Devices.Controllers {
				if controller.Type == "usb" && controller.Model == "none" {
					disabled = true
				}
			}

			Expect(disabled).To(BeFalse(), "Expect controller not to be disabled")
		})

		It("should not disable usb controller when device with no bus is present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = ""
			domain := vmiToDomain(vmi, c)
			disabled := false
			for _, controller := range domain.Spec.Devices.Controllers {
				if controller.Type == "usb" && controller.Model == "none" {
					disabled = true
				}
			}

			Expect(disabled).To(BeFalse(), "Expect controller not to be disabled")
		})

		It("should fail when input device is set to ps2 bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "ps2"
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should fail when input device is set to keyboard type", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Type = "keyboard"
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should succeed when input device is set to usb bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "usb"
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, &api.Domain{}, c)).To(Succeed(), "Expect success")
		})

		It("should succeed when input device bus is empty", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = ""
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Inputs[0].Bus).To(Equal(v1.InputBusUSB), "Expect usb bus")
		})

		It("should not enable sound cards emulation by default", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Sound = nil
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.SoundCards).To(BeEmpty())
		})

		It("should enable default sound card with existing but empty sound devices", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name := "audio-default-ich9"
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name: name,
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.SoundCards).To(HaveLen(1))
			Expect(domain.Spec.Devices.SoundCards).To(ContainElement(api.SoundCard{
				Alias: api.NewUserDefinedAlias(name),
				Model: "ich9",
			}))
		})

		It("should enable ac97 sound card ", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name := "audio-ac97"
			vmi.Spec.Domain.Devices.Sound = &v1.SoundDevice{
				Name:  name,
				Model: "ac97",
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.SoundCards).To(HaveLen(1))
			Expect(domain.Spec.Devices.SoundCards).To(ContainElement(api.SoundCard{
				Alias: api.NewUserDefinedAlias(name),
				Model: "ac97",
			}))
		})

		It("should enable usb redirection when number of USB client devices > 0", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.ClientPassthrough = &v1.ClientPassthroughDevices{}
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Redirs).To(HaveLen(4))
			Expect(domain.Spec.Devices.Controllers).To(ContainElement(api.Controller{
				Type:  "usb",
				Index: "0",
				Model: "qemu-xhci",
			}))
		})

		It("should not enable usb redirection when numberOfDevices == 0", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.ClientPassthrough = nil
			c.Architecture = "amd64"
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Redirs).To(BeNil())
			Expect(domain.Spec.Devices.Controllers).ToNot(ContainElement(api.Controller{
				Type:  "usb",
				Index: "0",
				Model: "qemu-xhci",
			}))
		})

		It("should select explicitly chosen network model", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Interfaces[0].Model.Type).To(Equal("e1000"))
		})

		It("should set rom to off when no boot order is specified", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces[0].BootOrder = nil
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Interfaces[0].Rom.Enabled).To(Equal("no"))
		})

		When("NIC PCI address is specified on VMI", func() {
			const pciAddress = "0000:81:01.0"
			expectedPCIAddress := api.Address{
				Type:     api.AddressPCI,
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x01",
				Function: "0x0",
			}

			BeforeEach(func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			})

			It("should be set on the domain spec for a non-SRIOV nic", func() {
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = pciAddress
				domain := vmiToDomain(vmi, c)
				Expect(*domain.Spec.Devices.Interfaces[0].Address).To(Equal(expectedPCIAddress))

			})
		})

		DescribeTable("should calculate mebibyte from a quantity", func(quantity string, mebibyte int) {
			mi64, _ := resource.ParseQuantity(quantity)
			Expect(vcpu.QuantityToMebiByte(mi64)).To(BeNumerically("==", mebibyte))
		},
			Entry("when 0M is given", "0M", 0),
			Entry("when 0 is given", "0", 0),
			Entry("when 1 is given", "1", 1),
			Entry("when 1M is given", "1M", 1),
			Entry("when 3M is given", "3M", 3),
			Entry("when 100M is given", "100M", 95),
			Entry("when 1Mi is given", "1Mi", 1),
			Entry("when 2G are given", "2G", 1907),
			Entry("when 2Gi are given", "2Gi", 2*1024),
			Entry("when 2780Gi are given", "2780Gi", 2780*1024),
		)

		It("should fail calculating mebibyte if the quantity is less than 0", func() {
			mi64, _ := resource.ParseQuantity("-2G")
			_, err := vcpu.QuantityToMebiByte(mi64)
			Expect(err).To(HaveOccurred())
		})

		DescribeTable("should calculate memory in bytes", func(quantity string, bytes int) {
			m64, _ := resource.ParseQuantity(quantity)
			memory, err := vcpu.QuantityToByte(m64)
			Expect(memory.Value).To(BeNumerically("==", bytes))
			Expect(memory.Unit).To(Equal("b"))
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("specifying memory 64M", "64M", 64*1000*1000),
			Entry("specifying memory 64Mi", "64Mi", 64*1024*1024),
			Entry("specifying memory 3G", "3G", 3*1000*1000*1000),
			Entry("specifying memory 3Gi", "3Gi", 3*1024*1024*1024),
			Entry("specifying memory 45Gi", "45Gi", 45*1024*1024*1024),
			Entry("specifying memory 2780Gi", "2780Gi", 2780*1024*1024*1024),
			Entry("specifying memory 451231 bytes", "451231", 451231),
		)
		It("should calculate memory in bytes", func() {
			By("specyfing negative memory size -45Gi")
			m45gi, _ := resource.ParseQuantity("-45Gi")
			_, err := vcpu.QuantityToByte(m45gi)
			Expect(err).To(HaveOccurred())
		})

		It("should convert hugepages", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Memory = &v1.Memory{
				Hugepages: &v1.Hugepages{},
			}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.MemoryBacking.HugePages).ToNot(BeNil())
			Expect(domainSpec.MemoryBacking.Source).ToNot(BeNil())
			Expect(domainSpec.MemoryBacking.Source.Type).To(Equal("memfd"))

			Expect(domainSpec.Memory.Value).To(Equal(uint64(8388608)))
			Expect(domainSpec.Memory.Unit).To(Equal("b"))
		})

		It("should use guest memory instead of requested memory if present", func() {
			guestMemory := resource.MustParse("123Mi")
			vmi.Spec.Domain.Memory = &v1.Memory{
				Guest: &guestMemory,
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)

			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

			Expect(domainSpec.Memory.Value).To(Equal(uint64(128974848)))
			Expect(domainSpec.Memory.Unit).To(Equal("b"))
		})

		It("should not add RNG when not present", func() {
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.Devices.Rng).To(BeNil())
		})

		It("should add RNG when present", func() {
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.Devices.Rng).ToNot(BeNil())
		})

		DescribeTable("Validate that QEMU SeaBios debug logs are ",
			func(toDefineVerbosityEnvVariable bool, virtLauncherLogVerbosity int, shouldEnableDebugLogs bool) {

				if toDefineVerbosityEnvVariable {
					Expect(os.Setenv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY, strconv.Itoa(virtLauncherLogVerbosity))).
						To(Succeed())
					defer func() {
						Expect(os.Unsetenv(services.ENV_VAR_VIRT_LAUNCHER_LOG_VERBOSITY)).To(Succeed())
					}()
				}

				domain := api.Domain{}

				Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, &domain, c)).To(Succeed())

				if domain.Spec.QEMUCmd == nil || (domain.Spec.QEMUCmd.QEMUArg == nil) {
					return
				}

				if shouldEnableDebugLogs {
					Expect(domain.Spec.QEMUCmd.QEMUArg).Should(ContainElements(
						api.Arg{Value: "-chardev"},
						api.Arg{Value: "file,id=firmwarelog,path=/tmp/qemu-firmware.log"},
						api.Arg{Value: "-device"},
						api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"},
					))
				} else {
					Expect(domain.Spec.QEMUCmd.QEMUArg).ShouldNot(Or(
						ContainElements(api.Arg{Value: "-chardev"}),
						ContainElements(api.Arg{Value: "file,id=firmwarelog,path=/tmp/qemu-firmware.log"}),
						ContainElements(api.Arg{Value: "-device"}),
						ContainElements(api.Arg{Value: "isa-debugcon,iobase=0x402,chardev=firmwarelog"}),
					))
				}

			},
			Entry("disabled - virtLauncherLogVerbosity does not exceed verbosity threshold", true, 0, false),
			Entry("enabled - virtLaucherLogVerbosity exceeds verbosity threshold", true, 1, true),
			Entry("disabled - virtLauncherLogVerbosity variable is not defined", false, -1, false),
		)

		DescribeTable("should add VSOCK section when present",
			func(useVirtioTransitional bool) {
				cid := uint32(100)
				vmi.Status.VSOCKCID = &cid
				vmi.Spec.Domain.Devices.AutoattachVSOCK = pointer.Bool(true)
				c.UseVirtioTransitional = useVirtioTransitional
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.Devices.VSOCK).ToNot(BeNil())
				Expect(domainSpec.Devices.VSOCK.Model).To(Equal("virtio-non-transitional"))
				Expect(domainSpec.Devices.VSOCK.CID.Auto).To(Equal("no"))
				Expect(domainSpec.Devices.VSOCK.CID.Address).To(BeNumerically("==", 100))
			},
			Entry("use virtio transitional", true),
			Entry("use virtio non-transitional", false),
		)
		DescribeTable("Should set the error policy", func(epolicy *v1.DiskErrorPolicy, expected string) {
			vmi.Spec.Domain.Devices.Disks[0] = v1.Disk{
				Name: "mydisk",
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: v1.VirtIO,
					},
				},
				ErrorPolicy: epolicy,
			}
			vmi.Spec.Volumes[0] = v1.Volume{
				Name: "mydisk",
				VolumeSource: v1.VolumeSource{
					Ephemeral: &v1.EphemeralVolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
			}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(string(domainSpec.Devices.Disks[0].Driver.ErrorPolicy)).To(Equal(expected))
		},
			Entry("ErrorPolicy not specified", nil, "stop"),
			Entry("ErrorPolicy equal to stop", kubevirtpointer.P(v1.DiskErrorPolicyStop), "stop"),
			Entry("ErrorPolicy equal to ignore", kubevirtpointer.P(v1.DiskErrorPolicyIgnore), "ignore"),
			Entry("ErrorPolicy equal to report", kubevirtpointer.P(v1.DiskErrorPolicyReport), "report"),
			Entry("ErrorPolicy equal to enospace", kubevirtpointer.P(v1.DiskErrorPolicyEnospace), "enospace"),
		)

	})
	Context("Network convert", func() {
		var vmi *v1.VirtualMachineInstance
		var c *ConverterContext

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}

			c = &ConverterContext{
				VirtualMachine: vmi,
				Secrets: map[string]*k8sv1.Secret{
					"mysecret": {
						Data: map[string][]byte{
							"node.session.auth.username": []byte("admin"),
						},
					},
				},
				AllowEmulation: true,
				SMBios:         TestSmbios,
			}
		})

		It("Should set domain interface source correctly for multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
			}
			vmi.Spec.Domain.Devices.Interfaces[0].Name = "red1"
			vmi.Spec.Domain.Devices.Interfaces[1].Name = "red2"
			// 3rd network is the default pod network, name is "default"
			vmi.Spec.Networks = []v1.Network{
				{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
				{
					Name: "red2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
				{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(3))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("ethernet"))
			Expect(domain.Spec.Devices.Interfaces[2].Type).To(Equal("ethernet"))
		})
		It("Should set domain interface source correctly for default multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(),
				*v1.DefaultBridgeNetworkInterface(),
			}
			vmi.Spec.Domain.Devices.Interfaces[0].Name = "red1"
			vmi.Spec.Domain.Devices.Interfaces[1].Name = "red2"
			vmi.Spec.Networks = []v1.Network{
				{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red", Default: true},
					},
				},
				{
					Name: "red2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("ethernet"))
		})
		It("should allow setting boot order", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name1"
			name2 := "Name2"
			iface1 := v1.DefaultBridgeNetworkInterface()
			iface2 := v1.DefaultBridgeNetworkInterface()
			net1 := v1.DefaultPodNetwork()
			net2 := v1.DefaultPodNetwork()
			iface1.Name = name1
			iface2.Name = name2
			bootOrder := uint(1)
			iface1.BootOrder = &bootOrder
			net1.Name = name1
			net2.Name = name2
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*iface1, *iface2}
			vmi.Spec.Networks = []v1.Network{*net1, *net2}
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].BootOrder).NotTo(BeNil())
			Expect(domain.Spec.Devices.Interfaces[0].BootOrder.Order).To(Equal(uint(bootOrder)))
			Expect(domain.Spec.Devices.Interfaces[1].BootOrder).To(BeNil())
		})
		It("Should create network configuration for masquerade interface", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			vmi.Spec.Networks = []v1.Network{*net1}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
		})
		It("Should create network configuration for masquerade interface and the pod network and a secondary network using multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1, *v1.DefaultBridgeNetworkInterface()}
			vmi.Spec.Domain.Devices.Interfaces[1].Name = "red1"

			vmi.Spec.Networks = []v1.Network{*net1,
				{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				}}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("ethernet"))
		})
		It("Should create network configuration for macvtap interface and a multus network", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			multusNetworkName := "multusNet"
			networkName := "net1"

			iface1 := v1.Interface{Name: networkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}}

			multusNetwork := v1.Network{
				Name: networkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: multusNetworkName},
				},
			}
			vmi.Spec.Networks = []v1.Network{multusNetwork}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1}

			domain := vmiToDomain(vmi, c)
			Expect(domain).NotTo(BeNil(), "domain should not be nil")
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1), "should have a single interface")
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"), "Macvtap interfaces must be of type `ethernet`")
		})
		It("Should create network configuration for the default pod network plus a secondary macvtap network interface using multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			secondaryNetworkName := "net1"

			iface1 := v1.Interface{Name: secondaryNetworkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}}

			defaultPodNetwork := v1.DefaultPodNetwork()
			multusNetwork := v1.Network{
				Name: secondaryNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: secondaryNetworkName},
				},
			}
			vmi.Spec.Networks = []v1.Network{*defaultPodNetwork, multusNetwork}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface(), iface1}

			domain := vmiToDomain(vmi, c)
			Expect(domain).NotTo(BeNil(), "domain should not be nil")
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2), "the VMI spec should feature 2 interfaces")
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("ethernet"), "Macvtap interfaces must be of type `ethernet`")
		})
		It("Macvtap interfaces should allow setting boot order", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			firstMacvtapNetworkName := "net1"
			secondMacvtapNetworkName := "net2"

			firstToBoot := uint(1)
			lastToBoot := uint(2)
			iface1 := v1.Interface{Name: firstMacvtapNetworkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}, BootOrder: &lastToBoot}
			iface2 := v1.Interface{Name: secondMacvtapNetworkName, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}, BootOrder: &firstToBoot}

			firstMacvtapNetwork := v1.Network{
				Name: firstMacvtapNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: firstMacvtapNetworkName},
				},
			}
			secondMacvtapNetwork := v1.Network{
				Name: secondMacvtapNetworkName,
				NetworkSource: v1.NetworkSource{
					Multus: &v1.MultusNetwork{NetworkName: secondMacvtapNetworkName},
				},
			}
			vmi.Spec.Networks = []v1.Network{firstMacvtapNetwork, secondMacvtapNetwork}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1, iface2}

			domain := vmiToDomain(vmi, c)
			Expect(domain).NotTo(BeNil(), "domain should not be nil")
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2), "the VMI spec should feature 2 interfaces")
			Expect(domain.Spec.Devices.Interfaces[0].BootOrder.Order).To(Equal(lastToBoot), "the interface whose boot order is higher should be the last to boot")
			Expect(domain.Spec.Devices.Interfaces[1].BootOrder.Order).To(Equal(firstToBoot), "the interface whose boot order is lower should be the first to boot")
		})
		Specify("macvtap interface binding must be used on a multus network", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "net1"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Macvtap: &v1.InterfaceMacvtap{}}}

			podNetwork := v1.Network{
				Name: name1,
				NetworkSource: v1.NetworkSource{
					Pod: &v1.PodNetwork{},
				},
			}
			vmi.Spec.Networks = []v1.Network{podNetwork}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1}

			domain := &api.Domain{}
			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(HaveOccurred(), "conversion should fail because a macvtap interface requires a multus network attachment")
		})
		It("creates SRIOV hostdev", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			domain := &api.Domain{}

			const identifyDevice = "sriov-test"
			c.SRIOVDevices = append(c.SRIOVDevices, api.HostDevice{Type: identifyDevice})

			Expect(Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
			Expect(domain.Spec.Devices.HostDevices).To(Equal([]api.HostDevice{{Type: identifyDevice}}))
		})
	})

	Context("graphics and video device", func() {

		DescribeTable("should check autoAttachGraphicsDevices", func(autoAttach *bool, devices int, arch string) {

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{Cores: 3},
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceCPU:    resource.MustParse("1m"),
								k8sv1.ResourceMemory: resource.MustParse("64M"),
							},
						},
					},
				},
			}
			vmi.Spec.Domain.Devices = v1.Devices{
				AutoattachGraphicsDevice: autoAttach,
			}
			domain := vmiToDomain(&vmi, &ConverterContext{AllowEmulation: true, Architecture: arch})
			Expect(domain.Spec.Devices.Video).To(HaveLen(devices))
			Expect(domain.Spec.Devices.Graphics).To(HaveLen(devices))

			if isARM64(arch) && (autoAttach == nil || *autoAttach) {
				Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal(v1.VirtIO))
				Expect(domain.Spec.Devices.Inputs[0].Type).To(Equal(v1.InputTypeTablet))
				Expect(domain.Spec.Devices.Inputs[1].Type).To(Equal(v1.InputTypeKeyboard))
			}
			if isAMD64(arch) && (autoAttach == nil || *autoAttach) {
				Expect(domain.Spec.Devices.Video[0].Model.Type).To(Equal("vga"))
			}
		},
			Entry("and add the graphics and video device if it is not set on amd64", nil, 1, "amd64"),
			Entry("and add the graphics and video device if it is set to true on amd64", True(), 1, "amd64"),
			Entry("and not add the graphics and video device if it is set to false on amd64", False(), 0, "amd64"),
			Entry("and add the graphics and video device if it is not set on arm64", nil, 1, "arm64"),
			Entry("and add the graphics and video device if it is set to true on arm64", True(), 1, "arm64"),
			Entry("and not add the graphics and video device if it is set to false on arm64", False(), 0, "arm64"),
		)
	})

	Context("HyperV features", func() {
		DescribeTable("should convert hyperv features", func(hyperV *v1.FeatureHyperv, result *api.FeatureHyperv) {
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						Features: &v1.Features{
							Hyperv: hyperV,
						},
					},
				},
			}

			domain := vmiToDomain(&vmi, &ConverterContext{AllowEmulation: true})
			Expect(domain.Spec.Features.Hyperv).To(Equal(result))

		},
			Entry("and add the vapic feature", &v1.FeatureHyperv{VAPIC: &v1.FeatureState{}}, &api.FeatureHyperv{VAPIC: &api.FeatureState{State: "on"}}),
			Entry("and add the stimer direct feature", &v1.FeatureHyperv{
				SyNICTimer: &v1.SyNICTimer{
					Direct: &v1.FeatureState{},
				},
			}, &api.FeatureHyperv{
				SyNICTimer: &api.SyNICTimer{
					State:  "on",
					Direct: &api.FeatureState{State: "on"},
				},
			}),
			Entry("and add the stimer feature without direct", &v1.FeatureHyperv{
				SyNICTimer: &v1.SyNICTimer{},
			}, &api.FeatureHyperv{
				SyNICTimer: &api.SyNICTimer{
					State: "on",
				},
			}),
			Entry("and add the vapic and the stimer direct feature", &v1.FeatureHyperv{
				SyNICTimer: &v1.SyNICTimer{
					Direct: &v1.FeatureState{},
				},
				VAPIC: &v1.FeatureState{},
			}, &api.FeatureHyperv{
				SyNICTimer: &api.SyNICTimer{
					State:  "on",
					Direct: &api.FeatureState{State: "on"},
				},
				VAPIC: &api.FeatureState{State: "on"},
			}),
		)
	})

	Context("serial console", func() {

		DescribeTable("should check autoAttachSerialConsole", func(autoAttach *bool, devices int) {

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{Cores: 3},
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceCPU:    resource.MustParse("1m"),
								k8sv1.ResourceMemory: resource.MustParse("64M"),
							},
						},
					},
				},
			}
			vmi.Spec.Domain.Devices = v1.Devices{
				AutoattachSerialConsole: autoAttach,
			}
			domain := vmiToDomain(&vmi, &ConverterContext{AllowEmulation: true})
			Expect(domain.Spec.Devices.Serials).To(HaveLen(devices))
			Expect(domain.Spec.Devices.Consoles).To(HaveLen(devices))

		},
			Entry("and add the serial console if it is not set", nil, 1),
			Entry("and add the serial console if it is set to true", True(), 1),
			Entry("and not add the serial console if it is set to false", False(), 0),
		)
	})

	Context("IOThreads", func() {

		DescribeTable("Should use correct IOThreads policies", func(policy v1.IOThreadsPolicy, cpuCores int, threadCount int, threadIDs []int) {
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						IOThreadsPolicy: &policy,
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceCPU: resource.MustParse(fmt.Sprintf("%d", cpuCores)),
							},
						},
						Devices: v1.Devices{
							Disks: []v1.Disk{
								{
									Name: "dedicated",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
									DedicatedIOThread: True(),
								},
								{
									Name: "shared",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
									DedicatedIOThread: False(),
								},
								{
									Name: "omitted1",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
								},
								{
									Name: "omitted2",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
								},
								{
									Name: "omitted3",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
								},
								{
									Name: "omitted4",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
								},
								{
									Name: "omitted5",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: v1.VirtIO,
										},
									},
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "dedicated",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "shared",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "omitted1",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "omitted2",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "omitted3",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "omitted4",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
						{
							Name: "omitted5",
							VolumeSource: v1.VolumeSource{
								Ephemeral: &v1.EphemeralVolumeSource{
									PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
										ClaimName: "testclaim",
									},
								},
							},
						},
					},
				},
			}

			domain := vmiToDomain(&vmi, &ConverterContext{AllowEmulation: true, EphemeraldiskCreator: EphemeralDiskImageCreator})
			Expect(domain.Spec.IOThreads).ToNot(BeNil())
			Expect(int(domain.Spec.IOThreads.IOThreads)).To(Equal(threadCount))
			for idx, disk := range domain.Spec.Devices.Disks {
				Expect(disk.Driver.IOThread).ToNot(BeNil())
				Expect(int(*disk.Driver.IOThread)).To(Equal(threadIDs[idx]))
			}
		},
			Entry("using a shared policy with 1 CPU", v1.IOThreadsPolicyShared, 1, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			Entry("using a shared policy with 2 CPUs", v1.IOThreadsPolicyShared, 2, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			Entry("using a shared policy with 3 CPUs", v1.IOThreadsPolicyShared, 2, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			Entry("using an auto policy with 1 CPU", v1.IOThreadsPolicyAuto, 1, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			Entry("using an auto policy with 2 CPUs", v1.IOThreadsPolicyAuto, 2, 4, []int{4, 1, 2, 3, 1, 2, 3}),
			Entry("using an auto policy with 3 CPUs", v1.IOThreadsPolicyAuto, 3, 6, []int{6, 1, 2, 3, 4, 5, 1}),
			Entry("using an auto policy with 4 CPUs", v1.IOThreadsPolicyAuto, 4, 7, []int{7, 1, 2, 3, 4, 5, 6}),
			Entry("using an auto policy with 5 CPUs", v1.IOThreadsPolicyAuto, 5, 7, []int{7, 1, 2, 3, 4, 5, 6}),
		)

	})

	Context("virtio block multi-queue", func() {
		var vmi *v1.VirtualMachineInstance
		var context *ConverterContext

		BeforeEach(func() {
			context = &ConverterContext{UseVirtioTransitional: false}
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "mydisk",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.VirtIO,
						},
					},
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "mydisk",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/myvolume/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
						},
					},
				},
			}

			vmi.Spec.Domain.Devices.BlockMultiQueue = True()
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("8192Ki"),
				k8sv1.ResourceCPU:    resource.MustParse("2"),
			}
		})

		It("should assign queues to a device if requested", func() {
			expectedQueues := uint(2)

			v1Disk := v1.Disk{
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{Bus: v1.VirtIO},
				},
			}
			apiDisk := api.Disk{}
			devicePerBus := map[string]deviceNamer{}
			numQueues := uint(2)
			Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues, make(map[string]v1.VolumeStatus))
			Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
			Expect(*(apiDisk.Driver.Queues)).To(Equal(expectedQueues), "expected queues to be 2")
		})

		It("should not assign queues to a device if omitted", func() {
			v1Disk := v1.Disk{
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			}
			apiDisk := api.Disk{}
			devicePerBus := map[string]deviceNamer{}
			Expect(Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, nil, make(map[string]v1.VolumeStatus))).
				To(Succeed())
			Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
			Expect(apiDisk.Driver.Queues).To(BeNil(), "expected no queues to be requested")
		})

		It("should honor multiQueue setting", func() {
			var expectedQueues uint = 2
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores: 2,
			}

			domain := vmiToDomain(vmi, &ConverterContext{AllowEmulation: true, SMBios: &cmdv1.SMBios{}})
			Expect(*(domain.Spec.Devices.Disks[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested vCPUs")
		})
	})
	Context("Correctly handle iothreads with dedicated cpus", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "default",
					UID:       "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{
					Domain: v1.DomainSpec{
						CPU: &v1.CPU{DedicatedCPUPlacement: true},
						Resources: v1.ResourceRequirements{
							Requests: k8sv1.ResourceList{
								k8sv1.ResourceMemory: resource.MustParse("64M"),
							},
						},
					},
				},
			}
		})
		It("assigns a set of cpus per iothread, if there are more vcpus than iothreads", func() {
			vmi.Spec.Domain.CPU.Cores = 16
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("16")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			c := &ConverterContext{CPUSet: []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
				AllowEmulation: true,
				SMBios:         &cmdv1.SMBios{},
				Topology: &cmdv1.Topology{
					NumaCells: []*cmdv1.Cell{{
						Cpus: []*cmdv1.CPU{
							{Id: 5},
							{Id: 6},
							{Id: 7},
							{Id: 8},
							{Id: 9},
							{Id: 10},
							{Id: 11},
							{Id: 12},
							{Id: 13},
							{Id: 14},
							{Id: 15},
							{Id: 16},
							{Id: 17},
							{Id: 18},
							{Id: 19},
							{Id: 20},
						},
					}},
				},
			}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &api.IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			Expect(vcpu.FormatDomainIOThreadPin(vmi, domain, 0, c.CPUSet)).To(Succeed())
			expectedLayout := []api.CPUTuneIOThreadPin{
				{IOThread: 1, CPUSet: "5,6,7"},
				{IOThread: 2, CPUSet: "8,9,10"},
				{IOThread: 3, CPUSet: "11,12,13"},
				{IOThread: 4, CPUSet: "14,15,16"},
				{IOThread: 5, CPUSet: "17,18"},
				{IOThread: 6, CPUSet: "19,20"},
			}
			isExpectedThreadsLayout := equality.Semantic.DeepEqual(expectedLayout, domain.Spec.CPUTune.IOThreadPin)
			Expect(isExpectedThreadsLayout).To(BeTrue())

		})
		It("should pack iothreads equally on available vcpus, if there are more iothreads than vcpus", func() {
			vmi.Spec.Domain.CPU.Cores = 2
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			c := &ConverterContext{
				CPUSet:         []int{5, 6},
				AllowEmulation: true,
				Topology: &cmdv1.Topology{
					NumaCells: []*cmdv1.Cell{{
						Cpus: []*cmdv1.CPU{
							{Id: 5},
							{Id: 6},
						},
					}},
				},
			}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &api.IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			Expect(vcpu.FormatDomainIOThreadPin(vmi, domain, 0, c.CPUSet)).To(Succeed())
			expectedLayout := []api.CPUTuneIOThreadPin{
				{IOThread: 1, CPUSet: "6"},
				{IOThread: 2, CPUSet: "5"},
				{IOThread: 3, CPUSet: "6"},
				{IOThread: 4, CPUSet: "5"},
				{IOThread: 5, CPUSet: "6"},
				{IOThread: 6, CPUSet: "5"},
			}
			isExpectedThreadsLayout := equality.Semantic.DeepEqual(expectedLayout, domain.Spec.CPUTune.IOThreadPin)
			Expect(isExpectedThreadsLayout).To(BeTrue())
		})
	})
	Context("virtio-net multi-queue", func() {
		var vmi *v1.VirtualMachineInstance

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}

			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = True()
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("8192Ki"),
				k8sv1.ResourceCPU:    resource.MustParse("2"),
			}
		})

		It("should assign queues to a device if requested", func() {
			var expectedQueues uint = 2
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores: 2,
			}

			domain := vmiToDomain(vmi, &ConverterContext{AllowEmulation: true})
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested vCPUs")
		})
		It("should assign queues to a device if requested based on vcpus", func() {
			var expectedQueues uint = 4

			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:   2,
				Sockets: 1,
				Threads: 2,
			}
			domain := vmiToDomain(vmi, &ConverterContext{AllowEmulation: true})
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested vCPUs")
		})

		It("should not assign queues to a non-virtio devices", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
			domain := vmiToDomain(vmi, &ConverterContext{AllowEmulation: true})
			Expect(domain.Spec.Devices.Interfaces[0].Driver).To(BeNil(),
				"queues should not be set for models other than virtio")
		})

		It("should cap the maximum number of queues", func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:   512,
				Sockets: 1,
				Threads: 2,
			}
			domain := vmiToDomain(vmi, &ConverterContext{AllowEmulation: true})
			expectedNumberQueues := uint(multiQueueMaxQueues)
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedNumberQueues),
				"should be capped to the maximum number of queues on tap devices")
		})

	})
	Context("Realtime", func() {
		var vmi *v1.VirtualMachineInstance
		var rtContext *ConverterContext
		BeforeEach(func() {
			rtContext = &ConverterContext{
				AllowEmulation: true,
				CPUSet:         []int{0, 1, 2, 3, 4},
				Topology: &cmdv1.Topology{
					NumaCells: []*cmdv1.Cell{
						{Id: 0,
							Memory:    &cmdv1.Memory{Amount: 10737418240, Unit: "G"},
							Pages:     []*cmdv1.Pages{{Count: 5, Unit: "G", Size: 1073741824}},
							Distances: []*cmdv1.Sibling{{Id: 0, Value: 1}},
							Cpus:      []*cmdv1.CPU{{Id: 0}, {Id: 1}, {Id: 2}}}}},
			}

			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:                 2,
				Sockets:               1,
				Threads:               1,
				Realtime:              &v1.Realtime{},
				DedicatedCPUPlacement: true,
			}
		})
		It("should configure the VCPU scheduler information utilizing all pinned vcpus when realtime is enabled", func() {
			domain := vmiToDomain(vmi, rtContext)
			Expect(domain.Spec.Features.PMU).To(Equal(&api.FeatureState{State: "off"}))
		})
	})

	Context("Bootloader", func() {
		var vmi *v1.VirtualMachineInstance
		var c *ConverterContext

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{
				ObjectMeta: k8smeta.ObjectMeta{
					Name:      "testvmi",
					Namespace: "mynamespace",
				},
			}

			v1.SetObjectDefaults_VirtualMachineInstance(vmi)

			c = &ConverterContext{
				VirtualMachine: vmi,
				AllowEmulation: true,
			}
		})

		Context("when bootloader is not set", func() {
			It("should configure the BIOS bootloader", func() {
				vmi.Spec.Domain.Firmware = &v1.Firmware{}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader).To(BeNil())
				Expect(domainSpec.OS.NVRam).To(BeNil())
			})
		})

		Context("when bootloader is set", func() {
			It("should configure the BIOS bootloader if no BIOS or EFI option", func() {
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader).To(BeNil())
				Expect(domainSpec.OS.NVRam).To(BeNil())
			})

			It("should configure the BIOS bootloader if BIOS", func() {
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						BIOS: &v1.BIOS{},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader).To(BeNil())
				Expect(domainSpec.OS.NVRam).To(BeNil())
			})
		})

		DescribeTable("EFI bootloader", func(secureBoot *bool, efiCode, efiVars string) {
			c.EFIConfiguration = &EFIConfiguration{
				EFICode:      efiCode,
				EFIVars:      efiVars,
				SecureLoader: secureBoot == nil || *secureBoot,
			}

			secureLoader := "yes"
			if secureBoot != nil && !*secureBoot {
				secureLoader = "no"
			}

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: secureBoot,
					},
				},
			}
			vmi.Status.RuntimeUser = 107
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.OS.BootLoader.ReadOnly).To(Equal("yes"))
			Expect(domainSpec.OS.BootLoader.Type).To(Equal("pflash"))
			Expect(domainSpec.OS.BootLoader.Secure).To(Equal(secureLoader))
			Expect(path.Base(domainSpec.OS.BootLoader.Path)).To(Equal(efiCode))
			Expect(path.Base(domainSpec.OS.NVRam.Template)).To(Equal(efiVars))
			Expect(domainSpec.OS.NVRam.NVRam).To(Equal("/var/run/kubevirt-private/libvirt/qemu/nvram/testvmi_VARS.fd"))
		},
			Entry("should use SecureBoot", True(), "OVMF_CODE.secboot.fd", "OVMF_VARS.secboot.fd"),
			Entry("should use SecureBoot when SB not defined", nil, "OVMF_CODE.secboot.fd", "OVMF_VARS.secboot.fd"),
			Entry("should not use SecureBoot", False(), "OVMF_CODE.fd", "OVMF_VARS.fd"),
			Entry("should not use SecureBoot when OVMF_CODE.fd not present", True(), "OVMF_CODE.secboot.fd", "OVMF_VARS.fd"),
		)

		It("EFI vars should be in the right place when running as root", func() {
			c.EFIConfiguration = &EFIConfiguration{
				EFICode:      "OVMF_CODE.fd",
				EFIVars:      "OVMF_VARS.fd",
				SecureLoader: false,
			}

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: pointer.BoolPtr(false),
					},
				},
			}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.OS.BootLoader.ReadOnly).To(Equal("yes"))
			Expect(domainSpec.OS.BootLoader.Type).To(Equal("pflash"))
			Expect(domainSpec.OS.BootLoader.Secure).To(Equal("no"))
			Expect(path.Base(domainSpec.OS.BootLoader.Path)).To(Equal(c.EFIConfiguration.EFICode))
			Expect(path.Base(domainSpec.OS.NVRam.Template)).To(Equal(c.EFIConfiguration.EFIVars))
			Expect(domainSpec.OS.NVRam.NVRam).To(Equal("/var/lib/libvirt/qemu/nvram/testvmi_VARS.fd"))
		})

		DescribeTable("display device should be set to", func(bootloader v1.Bootloader, enableFG bool, expectedDevice string) {
			vmi.Spec.Domain.Firmware = &v1.Firmware{Bootloader: &bootloader}
			c = &ConverterContext{
				BochsForEFIGuests: enableFG,
				VirtualMachine:    vmi,
				AllowEmulation:    true,
				EFIConfiguration:  &EFIConfiguration{},
			}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
			Expect(domainSpec.Devices.Video).To(HaveLen(1))
			Expect(domainSpec.Devices.Video[0].Model.Type).To(Equal(expectedDevice))
			if expectedDevice == "bochs" {
				// Bochs doesn't support the vram option
				Expect(domainSpec.Devices.Video[0].Model.VRam).To(BeNil())
			}
		},
			Entry("VGA with BIOS and BochsDisplayForEFIGuests unset", v1.Bootloader{BIOS: &v1.BIOS{}}, false, "vga"),
			Entry("VGA with BIOS and BochsDisplayForEFIGuests set", v1.Bootloader{BIOS: &v1.BIOS{}}, true, "vga"),
			Entry("VGA with EFI and BochsDisplayForEFIGuests unset", v1.Bootloader{EFI: &v1.EFI{}}, false, "vga"),
			Entry("Bochs with EFI and BochsDisplayForEFIGuests set", v1.Bootloader{EFI: &v1.EFI{}}, true, "bochs"),
		)
	})

	Context("Kernel Boot", func() {
		var vmi *v1.VirtualMachineInstance
		var c *ConverterContext

		BeforeEach(func() {
			vmi = &v1.VirtualMachineInstance{}

			v1.SetObjectDefaults_VirtualMachineInstance(vmi)

			c = &ConverterContext{
				VirtualMachine: vmi,
				AllowEmulation: true,
			}
		})

		Context("when kernel boot is set", func() {
			DescribeTable("should configure the kernel, initrd and Cmdline arguments correctly", func(kernelPath string, initrdPath string, kernelArgs string) {
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					KernelBoot: &v1.KernelBoot{
						KernelArgs: kernelArgs,
						Container: &v1.KernelBootContainer{
							KernelPath: kernelPath,
							InitrdPath: initrdPath,
						},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				if kernelPath == "" {
					Expect(domainSpec.OS.Kernel).To(BeEmpty())
				} else {
					Expect(domainSpec.OS.Kernel).To(ContainSubstring(kernelPath))
				}
				if initrdPath == "" {
					Expect(domainSpec.OS.Initrd).To(BeEmpty())
				} else {
					Expect(domainSpec.OS.Initrd).To(ContainSubstring(initrdPath))
				}

				Expect(domainSpec.OS.KernelArgs).To(Equal(kernelArgs))
			},
				Entry("when kernel, initrd and Cmdline are provided", "fully specified path to kernel", "fully specified path to initrd", "some cmdline arguments"),
				Entry("when only kernel and Cmdline are provided", "fully specified path to kernel", "", "some cmdline arguments"),
				Entry("when only kernel and initrd are provided", "fully specified path to kernel", "fully specified path to initrd", ""),
				Entry("when only kernel is provided", "fully specified path to kernel", "", ""),
				Entry("when only initrd and Cmdline are provided", "", "fully specified path to initrd", "some cmdline arguments"),
				Entry("when only Cmdline is provided", "", "", "some cmdline arguments"),
				Entry("when only initrd is provided", "", "fully specified path to initrd", ""),
				Entry("when no arguments provided", "", "", ""),
			)
		})
	})

	Context("hotplug", func() {
		var vmi *v1.VirtualMachineInstance
		var c *ConverterContext

		Context("disk", func() {

			type ConverterFunc = func(name string, disk *api.Disk, c *ConverterContext) error

			BeforeEach(func() {
				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: k8smeta.ObjectMeta{
						Name:      "testvmi",
						Namespace: "mynamespace",
					},
				}

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)

				c = &ConverterContext{
					VirtualMachine: vmi,
					AllowEmulation: true,
					IsBlockPVC: map[string]bool{
						"test-block-pvc": true,
					},
					IsBlockDV: map[string]bool{
						"test-block-dv": true,
					},
					VolumesDiscardIgnore: []string{
						"test-discard-ignore",
					},
				}
			})

			It("should automatically add virtio-scsi controller", func() {
				domain := vmiToDomain(vmi, c)
				Expect(domain.Spec.Devices.Controllers).To(HaveLen(3))
				foundScsiController := false
				for _, controller := range domain.Spec.Devices.Controllers {
					if controller.Type == "scsi" {
						foundScsiController = true
						Expect(controller.Model).To(Equal("virtio-non-transitional"))

					}
				}
				Expect(foundScsiController).To(BeTrue(), "did not find SCSI controller when expected")
			})

			It("should not automatically add virtio-scsi controller, if hotplug disabled", func() {
				vmi.Spec.Domain.Devices.DisableHotplug = true
				domain := vmiToDomain(vmi, c)
				Expect(domain.Spec.Devices.Controllers).To(HaveLen(2))
			})

			DescribeTable("should convert",
				func(converterFunc ConverterFunc, volumeName string, isBlockMode bool, ignoreDiscard bool) {
					expectedDisk := &api.Disk{}
					expectedDisk.Driver = &api.DiskDriver{}
					expectedDisk.Driver.Type = "raw"
					expectedDisk.Driver.ErrorPolicy = "stop"
					if isBlockMode {
						expectedDisk.Type = "block"
						expectedDisk.Source.Dev = filepath.Join(v1.HotplugDiskDir, volumeName)
					} else {
						expectedDisk.Type = "file"
						expectedDisk.Source.File = fmt.Sprintf("%s.img", filepath.Join(v1.HotplugDiskDir, volumeName))
					}
					if !ignoreDiscard {
						expectedDisk.Driver.Discard = "unmap"
					}

					disk := &api.Disk{
						Driver: &api.DiskDriver{},
					}
					Expect(converterFunc(volumeName, disk, c)).To(Succeed())
					Expect(disk).To(Equal(expectedDisk))
				},
				Entry("filesystem PVC", Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-fs-pvc", false, false),
				Entry("block mode PVC", Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-block-pvc", true, false),
				Entry("'discard ignore' PVC", Convert_v1_Hotplug_PersistentVolumeClaim_To_api_Disk, "test-discard-ignore", false, true),
				Entry("filesystem DV", Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-fs-dv", false, false),
				Entry("block mode DV", Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-block-dv", true, false),
				Entry("'discard ignore' DV", Convert_v1_Hotplug_DataVolume_To_api_Disk, "test-discard-ignore", false, true),
			)
		})

		Context("memory", func() {
			var domain *api.Domain
			var guestMemory resource.Quantity
			var maxGuestMemory resource.Quantity

			BeforeEach(func() {
				guestMemory = resource.MustParse("32Mi")
				maxGuestMemory = resource.MustParse("128Mi")

				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: k8smeta.ObjectMeta{
						Name:      "testvmi",
						Namespace: "mynamespace",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Memory: &v1.Memory{
								Guest:    &guestMemory,
								MaxGuest: &maxGuestMemory,
							},
						},
					},
					Status: v1.VirtualMachineInstanceStatus{
						Memory: &v1.MemoryStatus{
							GuestAtBoot:  &guestMemory,
							GuestCurrent: &guestMemory,
						},
					},
				}

				domain = &api.Domain{
					Spec: api.DomainSpec{
						VCPU: &api.VCPU{
							CPUs: 2,
						},
					},
				}

				v1.SetObjectDefaults_VirtualMachineInstance(vmi)

				c = &ConverterContext{
					VirtualMachine: vmi,
					AllowEmulation: true,
				}
			})

			It("should not setup hotplug when maxGuest is missing", func() {
				vmi.Spec.Domain.Memory.MaxGuest = nil
				err := setupDomainMemory(vmi, domain)
				Expect(err).ToNot(HaveOccurred())
				Expect(domain.Spec.MaxMemory).To(BeNil())
			})

			It("should not setup hotplug when maxGuest equals guest memory", func() {
				vmi.Spec.Domain.Memory.MaxGuest = &guestMemory
				err := setupDomainMemory(vmi, domain)
				Expect(err).ToNot(HaveOccurred())
				Expect(domain.Spec.MaxMemory).To(BeNil())
			})

			It("should setup hotplug when maxGuest is set", func() {
				err := setupDomainMemory(vmi, domain)
				Expect(err).ToNot(HaveOccurred())

				Expect(domain.Spec.MaxMemory).ToNot(BeNil())
				Expect(domain.Spec.MaxMemory.Unit).To(Equal("b"))
				Expect(domain.Spec.MaxMemory.Value).To(Equal(uint64(maxGuestMemory.Value())))

				Expect(domain.Spec.Memory).ToNot(BeNil())
				Expect(domain.Spec.Memory.Unit).To(Equal("b"))
				Expect(domain.Spec.Memory.Value).To(Equal(uint64(maxGuestMemory.Value())))

				Expect(domain.Spec.CPU.NUMA).ToNot(BeNil())
				Expect(domain.Spec.CPU.NUMA.Cells).To(HaveLen(1))
				Expect(domain.Spec.CPU.NUMA.Cells[0].Unit).To(Equal("b"))
				Expect(domain.Spec.CPU.NUMA.Cells[0].Memory).To(Equal(uint64(guestMemory.Value())))

				pluggableMemory := uint64(maxGuestMemory.Value() - guestMemory.Value())

				Expect(domain.Spec.Devices.Memory).ToNot(BeNil())
				Expect(domain.Spec.Devices.Memory.Model).To(Equal("virtio-mem"))
				Expect(domain.Spec.Devices.Memory.Target).ToNot(BeNil())
				Expect(domain.Spec.Devices.Memory.Target.Node).To(Equal("0"))
				Expect(domain.Spec.Devices.Memory.Target.Size.Value).To(Equal(pluggableMemory))
				Expect(domain.Spec.Devices.Memory.Target.Size.Unit).To(Equal("b"))
				Expect(domain.Spec.Devices.Memory.Target.Block.Value).To(Equal(uint64(MemoryHotplugBlockAlignmentBytes)))
				Expect(domain.Spec.Devices.Memory.Target.Block.Unit).To(Equal("b"))
			})
		})
	})

	Context("with AMD SEV LaunchSecurity", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *ConverterContext
		)

		BeforeEach(func() {
			vmi = kvapi.NewMinimalVMI("testvmi")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = pointer.BoolPtr(true)
			nonVirtioIface := v1.Interface{Name: "red", Model: "e1000"}
			secondaryNetwork := v1.Network{Name: "red"}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultBridgeNetworkInterface(), nonVirtioIface,
			}
			vmi.Spec.Networks = []v1.Network{
				*v1.DefaultPodNetwork(), secondaryNetwork,
			}
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{},
			}
			vmi.Spec.Domain.Features = &v1.Features{
				SMM: &v1.FeatureState{
					Enabled: pointer.BoolPtr(false),
				},
			}
			vmi.Spec.Domain.Firmware = &v1.Firmware{
				Bootloader: &v1.Bootloader{
					EFI: &v1.EFI{
						SecureBoot: pointer.BoolPtr(false),
					},
				},
			}
			c = &ConverterContext{
				AllowEmulation:    true,
				EFIConfiguration:  &EFIConfiguration{},
				UseLaunchSecurity: true,
			}
		})

		It("should set LaunchSecurity domain element with 'sev' type and 'NoDebug' policy", func() {
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.LaunchSecurity).ToNot(BeNil())
			Expect(domain.Spec.LaunchSecurity.Type).To(Equal("sev"))
			Expect(domain.Spec.LaunchSecurity.Policy).To(Equal("0x" + strconv.FormatUint(uint64(sev.SEVPolicyNoDebug), 16)))
		})

		It("should set LaunchSecurity domain element with 'sev' type with 'NoDebug' and 'EncryptedState' policy bits", func() {
			// VMI with SEV-ES
			vmi.Spec.Domain.LaunchSecurity = &v1.LaunchSecurity{
				SEV: &v1.SEV{
					Policy: &v1.SEVPolicy{
						EncryptedState: pointer.Bool(true),
					},
				},
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.LaunchSecurity).ToNot(BeNil())
			Expect(domain.Spec.LaunchSecurity.Type).To(Equal("sev"))
			Expect(domain.Spec.LaunchSecurity.Policy).To(Equal("0x" + strconv.FormatUint(uint64(sev.SEVPolicyNoDebug|sev.SEVPolicyEncryptedState), 16)))
		})

		It("should set IOMMU attribute of the RngDriver", func() {
			rng := &api.Rng{}
			Expect(Convert_v1_Rng_To_api_Rng(&v1.Rng{}, rng, c)).To(Succeed())
			Expect(rng.Driver).ToNot(BeNil())
			Expect(rng.Driver.IOMMU).To(Equal("on"))

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Rng).ToNot(BeNil())
			Expect(domain.Spec.Devices.Rng.Driver).ToNot(BeNil())
			Expect(domain.Spec.Devices.Rng.Driver.IOMMU).To(Equal("on"))
		})

		It("should set IOMMU attribute of the MemBalloonDriver", func() {
			memBaloon := &api.MemBalloon{}
			ConvertV1ToAPIBalloning(&v1.Devices{}, memBaloon, c)
			Expect(memBaloon.Driver).ToNot(BeNil())
			Expect(memBaloon.Driver.IOMMU).To(Equal("on"))

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning.Driver).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning.Driver.IOMMU).To(Equal("on"))
		})

		It("should set IOMMU attribute of the virtio-net driver", func() {
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].Driver).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces[0].Driver.IOMMU).To(Equal("on"))
			Expect(domain.Spec.Devices.Interfaces[1].Driver).To(BeNil())
		})

		It("should disable the iPXE option ROM", func() {
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].Rom).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces[0].Rom.Enabled).To(Equal("no"))
			Expect(domain.Spec.Devices.Interfaces[1].Rom).ToNot(BeNil())
			Expect(domain.Spec.Devices.Interfaces[1].Rom.Enabled).To(Equal("no"))
		})
	})

	Context("when TSC Frequency", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *ConverterContext
		)

		const fakeFrequency = 12345

		BeforeEach(func() {
			vmi = kvapi.NewMinimalVMI("testvmi")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Status.TopologyHints = &v1.TopologyHints{TSCFrequency: pointer.Int64(fakeFrequency)}
			c = &ConverterContext{
				AllowEmulation: true,
			}
		})

		expectTsc := func(domain *api.Domain, expectExists bool) {
			Expect(domain).ToNot(BeNil())
			if !expectExists && domain.Spec.Clock == nil {
				return
			}

			Expect(domain.Spec.Clock).ToNot(BeNil())

			found := false
			for _, timer := range domain.Spec.Clock.Timer {
				if timer.Name == "tsc" {
					actualFrequency, err := strconv.Atoi(timer.Frequency)
					Expect(err).ToNot(HaveOccurred(), "frequency cannot be converted into a number")
					Expect(actualFrequency).To(Equal(fakeFrequency), "set frequency is incorrect")

					found = true
					break
				}
			}

			expectationStr := "exist"
			if !expectExists {
				expectationStr = "not " + expectationStr
			}
			Expect(found).To(Equal(expectExists), fmt.Sprintf("domain TSC frequency is expected to %s", expectationStr))
		}

		Context("is required because VMI is using", func() {
			It("hyperV reenlightenment", func() {
				vmi.Spec.Domain.Features = &v1.Features{
					Hyperv: &v1.FeatureHyperv{
						Reenlightenment: &v1.FeatureState{Enabled: pointer.Bool(true)},
					},
				}

				domain := vmiToDomain(vmi, c)
				expectTsc(domain, true)
			})

			It("invtsc CPU feature", func() {
				vmi.Spec.Domain.CPU = &v1.CPU{
					Features: []v1.CPUFeature{
						{Name: "invtsc", Policy: "require"},
					},
				}

				domain := vmiToDomain(vmi, c)
				expectTsc(domain, true)
			})
		})

		It("is not required", func() {
			domain := vmiToDomain(vmi, c)
			expectTsc(domain, false)
		})
	})

	Context("with FreePageReporting", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *ConverterContext
		)

		BeforeEach(func() {
			vmi = kvapi.NewMinimalVMI("testvmi")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		})

		DescribeTable("should set freePageReporting attribute of memballooning device, accordingly to the context value", func(freePageReporting bool, expectedValue string) {
			c = &ConverterContext{
				FreePageReporting: freePageReporting,
				AllowEmulation:    true,
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())

			Expect(domain.Spec.Devices).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning).ToNot(BeNil())
			Expect(domain.Spec.Devices.Ballooning.FreePageReporting).To(BeEquivalentTo(expectedValue))
		},
			Entry("when true", true, "on"),
			Entry("when false", false, "off"),
		)
	})

	Context("with Paused strategy", func() {
		var (
			vmi *v1.VirtualMachineInstance
			c   *ConverterContext
		)

		BeforeEach(func() {
			vmi = kvapi.NewMinimalVMI("testvmi")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		})

		DescribeTable("bootmenu should be", func(startPaused bool) {
			c = &ConverterContext{
				AllowEmulation: true,
			}

			if startPaused {
				strategy := v1.StartStrategyPaused
				vmi.Spec.StartStrategy = &strategy
			}
			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(BeNil())

			if startPaused {
				Expect(domain.Spec.OS.BootMenu).ToNot(BeNil())
				Expect(domain.Spec.OS.BootMenu.Enable).To(Equal("yes"))
				Expect(*domain.Spec.OS.BootMenu.Timeout).To(Equal(BootMenuTimeoutMS))
			} else {
				Expect(domain.Spec.OS.BootMenu).To(BeNil())
			}

		},
			Entry("enabled when set", true),
			Entry("disabled when not set", false),
		)
	})
})

var _ = Describe("disk device naming", func() {
	It("format device name should return correct value", func() {
		res := FormatDeviceName("sd", 0)
		Expect(res).To(Equal("sda"))
		res = FormatDeviceName("sd", 1)
		Expect(res).To(Equal("sdb"))
		// 25 is z 26 starting at 0
		res = FormatDeviceName("sd", 25)
		Expect(res).To(Equal("sdz"))
		res = FormatDeviceName("sd", 26*2-1)
		Expect(res).To(Equal("sdaz"))
		res = FormatDeviceName("sd", 26*26-1)
		Expect(res).To(Equal("sdyz"))
	})

	It("makeDeviceName should generate proper name", func() {
		prefixMap := make(map[string]deviceNamer)
		res, index := makeDeviceName("test1", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vda"))
		Expect(index).To(Equal(0))
		for i := 2; i < 10; i++ {
			makeDeviceName(fmt.Sprintf("test%d", i), v1.VirtIO, prefixMap)
		}
		prefix := getPrefixFromBus(v1.VirtIO)
		delete(prefixMap[prefix].usedDeviceMap, "vdd")
		By("Verifying next value is vdd")
		res, index = makeDeviceName("something", v1.VirtIO, prefixMap)
		Expect(index).To(Equal(3))
		Expect(res).To(Equal("vdd"))
		res, index = makeDeviceName("something_else", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vdj"))
		Expect(index).To(Equal(9))
		By("verifying existing returns correct value")
		res, index = makeDeviceName("something", v1.VirtIO, prefixMap)
		Expect(res).To(Equal("vdd"))
		Expect(index).To(Equal(3))
		By("Verifying a new bus returns from start")
		res, index = makeDeviceName("something", "scsi", prefixMap)
		Expect(res).To(Equal("sda"))
		Expect(index).To(Equal(0))
	})
})

var _ = Describe("direct IO checker", func() {
	var directIOChecker DirectIOChecker
	var tmpDir string
	var existingFile string
	var nonExistingFile string
	var err error

	BeforeEach(func() {
		directIOChecker = NewDirectIOChecker()
		tmpDir, err = os.MkdirTemp("", "direct-io-checker")
		Expect(err).ToNot(HaveOccurred())
		existingFile = filepath.Join(tmpDir, "disk.img")
		Expect(os.WriteFile(existingFile, []byte("test"), 0644)).To(Succeed())
		nonExistingFile = filepath.Join(tmpDir, "non-existing-file")
	})

	AfterEach(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	It("should not fail when file/device exists", func() {
		_, err = directIOChecker.CheckFile(existingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = directIOChecker.CheckBlockDevice(existingFile)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should not fail when file does not exist", func() {
		_, err := directIOChecker.CheckFile(nonExistingFile)
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when device does not exist", func() {
		_, err := directIOChecker.CheckBlockDevice(nonExistingFile)
		Expect(err).To(HaveOccurred())
		_, err = os.Stat(nonExistingFile)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})

	It("should fail when the path does not exist", func() {
		nonExistingPath := "/non/existing/path/disk.img"
		_, err = directIOChecker.CheckFile(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = directIOChecker.CheckBlockDevice(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
		_, err = os.Stat(nonExistingPath)
		Expect(err).To(MatchError(fs.ErrNotExist))
	})
})

var _ = Describe("SetDriverCacheMode", func() {
	var ctrl *gomock.Controller
	var mockDirectIOChecker *MockDirectIOChecker

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockDirectIOChecker = NewMockDirectIOChecker(ctrl)
	})

	expectCheckTrue := func() {
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(true, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(true, nil)
	}

	expectCheckFalse := func() {
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(false, nil)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(false, nil)
	}

	expectCheckError := func() {
		checkerError := fmt.Errorf("DirectIOChecker error")
		mockDirectIOChecker.EXPECT().CheckBlockDevice(gomock.Any()).AnyTimes().Return(false, checkerError)
		mockDirectIOChecker.EXPECT().CheckFile(gomock.Any()).AnyTimes().Return(false, checkerError)
	}

	DescribeTable("should correctly set driver cache mode", func(cache, expectedCache string, setExpectations func()) {
		disk := &api.Disk{
			Driver: &api.DiskDriver{
				Cache: cache,
			},
			Source: api.DiskSource{
				File: "file",
			},
		}
		setExpectations()
		err := SetDriverCacheMode(disk, mockDirectIOChecker)
		if expectedCache == "" {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).ToNot(HaveOccurred())
			Expect(disk.Driver.Cache).To(Equal(expectedCache))
		}
	},
		Entry("detect 'none' with direct io", string(""), string(v1.CacheNone), expectCheckTrue),
		Entry("detect 'writethrough' without direct io", string(""), string(v1.CacheWriteThrough), expectCheckFalse),
		Entry("fallback to 'writethrough' on error", string(""), string(v1.CacheWriteThrough), expectCheckError),
		Entry("keep 'none' with direct io", string(v1.CacheNone), string(v1.CacheNone), expectCheckTrue),
		Entry("return error without direct io", string(v1.CacheNone), string(""), expectCheckFalse),
		Entry("return error on error", string(v1.CacheNone), string(""), expectCheckError),
		Entry("'writethrough' with direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckTrue),
		Entry("'writethrough' without direct io", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckFalse),
		Entry("'writethrough' on error", string(v1.CacheWriteThrough), string(v1.CacheWriteThrough), expectCheckError),
	)
})

func diskToDiskXML(disk *v1.Disk) string {
	devicePerBus := make(map[string]deviceNamer)
	libvirtDisk := &api.Disk{}
	Expect(Convert_v1_Disk_To_api_Disk(&ConverterContext{UseVirtioTransitional: false}, disk, libvirtDisk, devicePerBus, nil, make(map[string]v1.VolumeStatus))).To(Succeed())
	data, err := xml.MarshalIndent(libvirtDisk, "", "  ")
	Expect(err).ToNot(HaveOccurred())
	return string(data)
}

func vmiToDomainXML(vmi *v1.VirtualMachineInstance, c *ConverterContext) string {
	domain := vmiToDomain(vmi, c)
	data, err := xml.MarshalIndent(domain.Spec, "", "  ")
	Expect(err).ToNot(HaveOccurred())
	return string(data)
}

func vmiToDomain(vmi *v1.VirtualMachineInstance, c *ConverterContext) *api.Domain {
	domain := &api.Domain{}
	ExpectWithOffset(1, Convert_v1_VirtualMachineInstance_To_api_Domain(vmi, domain, c)).To(Succeed())
	api.NewDefaulter(c.Architecture).SetObjectDefaults_Domain(domain)
	return domain
}

func xmlToDomainSpec(data string) *api.DomainSpec {
	newDomain := &api.DomainSpec{}
	ExpectWithOffset(1, xml.Unmarshal([]byte(data), newDomain)).To(Succeed())
	newDomain.XMLName.Local = ""
	newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	return newDomain
}

func vmiToDomainXMLToDomainSpec(vmi *v1.VirtualMachineInstance, c *ConverterContext) *api.DomainSpec {
	return xmlToDomainSpec(vmiToDomainXML(vmi, c))
}

func True() *bool {
	b := true
	return &b
}

func False() *bool {
	b := false
	return &b
}

// As the arch specific default disk is set in the mutating webhook, so in some tests,
// it needs to run the mutate function before verifying converter
func vmiArchMutate(arch string, vmi *v1.VirtualMachineInstance, c *ConverterContext) {
	if arch == "arm64" {
		webhooks.SetArm64Defaults(&vmi.Spec)
		// bootloader has been initialized in webhooks.SetArm64Defaults,
		// c.EFIConfiguration.SecureLoader is needed in the converter.Convert_v1_VirtualMachineInstance_To_api_Domain.
		c.EFIConfiguration = &EFIConfiguration{
			SecureLoader: false,
		}

	} else {
		webhooks.SetAmd64Defaults(&vmi.Spec)
	}
}
