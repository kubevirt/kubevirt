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
	"encoding/xml"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	v1 "kubevirt.io/client-go/api/v1"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

var _ = Describe("Converter", func() {

	TestSmbios := &cmdv1.SMBios{}

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
			Convert_v1_Clock_To_api_Clock(clock, &convertClock, &ConverterContext{})
			data, err := xml.MarshalIndent(convertClock, "", "  ")
			Expect(err).ToNot(HaveOccurred())

			expectedClock := `<Clock offset="timezone" timezone="America/New_York"></Clock>`
			Expect(string(data)).To(Equal(expectedClock))
		})
	})

	Context("with v1.Disk", func() {
		It("Should add boot order when provided", func() {
			order := uint(1)
			kubevirtDisk := &v1.Disk{
				Name:      "mydisk",
				BootOrder: &order,
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{
						Bus: "virtio",
					},
				},
			}
			var convertedDisk = `<Disk device="disk" type="" model="virtio-non-transitional">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type=""></driver>
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
						Bus: "virtio",
					},
				},
			}
			var convertedDisk = `<Disk device="disk" type="" model="virtio-non-transitional">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type=""></driver>
  <alias name="ua-mydisk"></alias>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			Expect(xml).To(Equal(convertedDisk))
		})

	})

	Context("with v1.VirtualMachineInstance", func() {

		var vmi *v1.VirtualMachineInstance
		_false := false
		_true := true
		domainType := "kvm"
		if _, err := os.Stat("/dev/kvm"); os.IsNotExist(err) {
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
						Enabled:    &_false,
						TickPolicy: v1.HPETTickPolicyDelay,
					},
					KVM: &v1.KVMTimer{
						Enabled: &_true,
					},
					PIT: &v1.PITTimer{
						Enabled:    &_false,
						TickPolicy: v1.PITTickPolicyDiscard,
					},
					RTC: &v1.RTCTimer{
						Enabled:    &_true,
						TickPolicy: v1.RTCTickPolicyCatchup,
						Track:      v1.TrackGuest,
					},
					Hyperv: &v1.HypervTimer{
						Enabled: &_true,
					},
				},
			}
			vmi.Spec.Domain.Features = &v1.Features{
				APIC: &v1.FeatureAPIC{},
				SMM:  &v1.FeatureState{},
				KVM:  &v1.FeatureKVM{Hidden: true},
				Hyperv: &v1.FeatureHyperv{
					Relaxed:         &v1.FeatureState{Enabled: &_false},
					VAPIC:           &v1.FeatureState{Enabled: &_true},
					Spinlocks:       &v1.FeatureSpinlocks{Enabled: &_true},
					VPIndex:         &v1.FeatureState{Enabled: &_true},
					Runtime:         &v1.FeatureState{Enabled: &_false},
					SyNIC:           &v1.FeatureState{Enabled: &_true},
					SyNICTimer:      &v1.FeatureState{Enabled: &_false},
					Reset:           &v1.FeatureState{Enabled: &_true},
					VendorID:        &v1.FeatureVendorID{Enabled: &_false, VendorID: "myvendor"},
					Frequencies:     &v1.FeatureState{Enabled: &_false},
					Reenlightenment: &v1.FeatureState{Enabled: &_false},
					TLBFlush:        &v1.FeatureState{Enabled: &_true},
					IPI:             &v1.FeatureState{Enabled: &_true},
					EVMCS:           &v1.FeatureState{Enabled: &_false},
				},
			}
			vmi.Spec.Domain.Resources.Limits = make(k8sv1.ResourceList)
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{k8sv1.ResourceMemory: resource.MustParse("8192Ki")}
			vmi.Spec.Domain.Devices.DisableHotplug = true
			vmi.Spec.Domain.Devices.Inputs = []v1.Input{
				{
					Bus:  "virtio",
					Type: "tablet",
					Name: "tablet0",
				},
			}
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					DedicatedIOThread: &_true,
				},
				{
					Name: "nocloud",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
					DedicatedIOThread: &_true,
				},
				{
					Name: "cdrom_tray_unspecified",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							ReadOnly: &_false,
						},
					},
					DedicatedIOThread: &_false,
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
					Name: "floppy_tray_unspecified",
					DiskDevice: v1.DiskDevice{
						Floppy: &v1.FloppyTarget{},
					},
				},
				{
					Name: "floppy_tray_open",
					DiskDevice: v1.DiskDevice{
						Floppy: &v1.FloppyTarget{
							Tray:     v1.TrayStateOpen,
							ReadOnly: true,
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
					Name: "floppy_tray_unspecified",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume2/disk.img",
							Type:     v1.HostDiskExistsOrCreate,
							Capacity: resource.MustParse("1Gi"),
						},
					},
				},
				{
					Name: "floppy_tray_open",
					VolumeSource: v1.VolumeSource{
						HostDisk: &v1.HostDisk{
							Path:     "/var/run/kubevirt-private/vmi-disks/volume3/disk.img",
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
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testblock",
						},
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

		var convertedDomain = fmt.Sprintf(`<domain type="%s" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="b">8388608</memory>
  <os>
    <type arch="x86_64" machine="q35">hvm</type>
    <smbios mode="sysinfo"></smbios>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
      <entry name="serial">e4686d2c-6e8d-4335-b8fd-81bee22f4815</entry>
      <entry name="manufacturer"></entry>
      <entry name="family"></entry>
      <entry name="product"></entry>
      <entry name="sku"></entry>
      <entry name="version"></entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
    <chassis></chassis>
  </sysinfo>
  <devices>
    <interface type="ethernet">
      <source></source>
      <model type="virtio-non-transitional"></model>
      <alias name="ua-default"></alias>
      <rom enabled="no"></rom>
    </interface>
    <channel type="unix">
      <target name="org.qemu.guest_agent.0" type="virtio"></target>
    </channel>
    <controller type="usb" index="0" model="none"></controller>
    <controller type="virtio-serial" index="0" model="virtio-non-transitional"></controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <graphics type="vnc">
      <listen type="socket" socket="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-vnc"></listen>
    </graphics>
    %s
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vda"></target>
      <driver name="qemu" type="raw" iothread="2"></driver>
      <alias name="ua-myvolume"></alias>
    </disk>
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="virtio" dev="vdb"></target>
      <driver name="qemu" type="raw" iothread="3"></driver>
      <alias name="ua-nocloud"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="sata" dev="sda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-cdrom_tray_unspecified"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/cdrom_tray_open/disk.img"></source>
      <target bus="sata" dev="sdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-cdrom_tray_open"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_unspecified/disk.img"></source>
      <target bus="fdc" dev="fda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-floppy_tray_unspecified"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_open/disk.img"></source>
      <target bus="fdc" dev="fdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-floppy_tray_open"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/should_default_to_disk/disk.img"></source>
      <target bus="sata" dev="sdc"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-should_default_to_disk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/kubevirt-ephemeral-disk/ephemeral_pvc/disk.qcow2"></source>
      <target bus="sata" dev="sdd"></target>
      <driver cache="none" name="qemu" type="qcow2" iothread="1"></driver>
      <alias name="ua-ephemeral_pvc"></alias>
      <backingStore type="file">
        <format type="raw"></format>
        <source file="/var/run/kubevirt-private/vmi-disks/ephemeral_pvc/disk.img"></source>
      </backingStore>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/secret-disks/secret_test.iso"></source>
      <target bus="sata" dev="sde"></target>
      <serial>D23YZ9W6WA5DJ487</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-secret_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/config-map-disks/configmap_test.iso"></source>
      <target bus="sata" dev="sdf"></target>
      <serial>CVLY623300HK240D</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-configmap_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/pvc_block_test"></source>
      <target bus="sata" dev="sdg"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-pvc_block_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/dv_block_test"></source>
      <target bus="sata" dev="sdh"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-dv_block_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/service-account-disk/service-account.iso"></source>
      <target bus="sata" dev="sdi"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-serviceaccount_test"></alias>
    </disk>
    <input type="tablet" bus="virtio" model="virtio">
      <alias name="ua-tablet0"></alias>
    </input>
    <serial type="unix">
      <target port="0"></target>
      <source mode="bind" path="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-serial0"></source>
    </serial>
    <console type="pty">
      <target type="serial" port="0"></target>
    </console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="ua-mywatchdog"></alias>
    </watchdog>
    <rng model="virtio-non-transitional">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
  <clock offset="utc" adjustment="reset">
    <timer name="rtc" tickpolicy="catchup" present="yes" track="guest"></timer>
    <timer name="pit" tickpolicy="discard" present="no"></timer>
    <timer name="kvmclock" present="yes"></timer>
    <timer name="hpet" tickpolicy="delay" present="no"></timer>
    <timer name="hypervclock" present="yes"></timer>
  </clock>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>f4686d2c-6e8d-4335-b8fd-81bee22f4814</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>5</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <features>
    <acpi></acpi>
    <apic></apic>
    <hyperv>
      <relaxed state="off"></relaxed>
      <vapic state="on"></vapic>
      <spinlocks state="on" retries="4096"></spinlocks>
      <vpindex state="on"></vpindex>
      <runtime state="off"></runtime>
      <synic state="on"></synic>
      <stimer state="off"></stimer>
      <reset state="on"></reset>
      <vendor_id state="off" value="myvendor"></vendor_id>
      <frequencies state="off"></frequencies>
      <reenlightenment state="off"></reenlightenment>
      <tlbflush state="on"></tlbflush>
      <ipi state="on"></ipi>
      <evmcs state="off"></evmcs>
    </hyperv>
    <smm></smm>
    <kvm>
      <hidden state="on"></hidden>
    </kvm>
  </features>
  <cpu mode="host-model">
    <topology sockets="1" cores="1" threads="1"></topology>
  </cpu>
  <vcpu placement="static">1</vcpu>
  <iothreads>3</iothreads>
</domain>`, domainType, "%s")
		var convertedDomainWith5Period = fmt.Sprintf(convertedDomain,
			`<memballoon model="virtio-non-transitional">
      <stats period="5"></stats>
    </memballoon>`)
		var convertedDomainWith0Period = fmt.Sprintf(convertedDomain,
			`<memballoon model="virtio-non-transitional"></memballoon>`)
		var convertedDomainWithFalseAutoattach = fmt.Sprintf(convertedDomain,
			`<memballoon model="none"></memballoon>`)
		convertedDomain = fmt.Sprintf(convertedDomain,
			`<memballoon model="virtio-non-transitional">
      <stats period="10"></stats>
    </memballoon>`)

		var convertedDomainppc64le = fmt.Sprintf(`<domain type="%s" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="b">8388608</memory>
  <os>
    <type arch="ppc64le" machine="pseries">hvm</type>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
      <entry name="serial">e4686d2c-6e8d-4335-b8fd-81bee22f4815</entry>
      <entry name="manufacturer"></entry>
      <entry name="family"></entry>
      <entry name="product"></entry>
      <entry name="sku"></entry>
      <entry name="version"></entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
    <chassis></chassis>
  </sysinfo>
  <devices>
    <interface type="ethernet">
      <source></source>
      <model type="virtio-non-transitional"></model>
      <alias name="ua-default"></alias>
      <rom enabled="no"></rom>
    </interface>
    <channel type="unix">
      <target name="org.qemu.guest_agent.0" type="virtio"></target>
    </channel>
    <controller type="usb" index="0" model="qemu-xhci"></controller>
    <controller type="virtio-serial" index="0" model="virtio-non-transitional"></controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <graphics type="vnc">
      <listen type="socket" socket="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-vnc"></listen>
    </graphics>
    %s
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vda"></target>
      <driver name="qemu" type="raw" iothread="2"></driver>
      <alias name="ua-myvolume"></alias>
    </disk>
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="virtio" dev="vdb"></target>
      <driver name="qemu" type="raw" iothread="3"></driver>
      <alias name="ua-nocloud"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="sata" dev="sda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-cdrom_tray_unspecified"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/cdrom_tray_open/disk.img"></source>
      <target bus="sata" dev="sdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-cdrom_tray_open"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_unspecified/disk.img"></source>
      <target bus="fdc" dev="fda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-floppy_tray_unspecified"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_open/disk.img"></source>
      <target bus="fdc" dev="fdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-floppy_tray_open"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/should_default_to_disk/disk.img"></source>
      <target bus="sata" dev="sdc"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-should_default_to_disk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/kubevirt-ephemeral-disk/ephemeral_pvc/disk.qcow2"></source>
      <target bus="sata" dev="sdd"></target>
      <driver cache="none" name="qemu" type="qcow2" iothread="1"></driver>
      <alias name="ua-ephemeral_pvc"></alias>
      <backingStore type="file">
        <format type="raw"></format>
        <source file="/var/run/kubevirt-private/vmi-disks/ephemeral_pvc/disk.img"></source>
      </backingStore>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/secret-disks/secret_test.iso"></source>
      <target bus="sata" dev="sde"></target>
      <serial>D23YZ9W6WA5DJ487</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-secret_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/config-map-disks/configmap_test.iso"></source>
      <target bus="sata" dev="sdf"></target>
      <serial>CVLY623300HK240D</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-configmap_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/pvc_block_test"></source>
      <target bus="sata" dev="sdg"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-pvc_block_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/dv_block_test"></source>
      <target bus="sata" dev="sdh"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-dv_block_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/service-account-disk/service-account.iso"></source>
      <target bus="sata" dev="sdi"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-serviceaccount_test"></alias>
    </disk>
    <input type="tablet" bus="virtio" model="virtio">
      <alias name="ua-tablet0"></alias>
    </input>
    <serial type="unix">
      <target port="0"></target>
      <source mode="bind" path="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-serial0"></source>
    </serial>
    <console type="pty">
      <target type="serial" port="0"></target>
    </console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="ua-mywatchdog"></alias>
    </watchdog>
    <rng model="virtio-non-transitional">
      <backend model="random">/dev/urandom</backend>
    </rng>
  </devices>
  <clock offset="utc" adjustment="reset">
    <timer name="rtc" tickpolicy="catchup" present="yes" track="guest"></timer>
    <timer name="pit" tickpolicy="discard" present="no"></timer>
    <timer name="kvmclock" present="yes"></timer>
    <timer name="hpet" tickpolicy="delay" present="no"></timer>
    <timer name="hypervclock" present="yes"></timer>
  </clock>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>f4686d2c-6e8d-4335-b8fd-81bee22f4814</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>5</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <features>
    <acpi></acpi>
    <apic></apic>
    <hyperv>
      <relaxed state="off"></relaxed>
      <vapic state="on"></vapic>
      <spinlocks state="on" retries="4096"></spinlocks>
      <vpindex state="on"></vpindex>
      <runtime state="off"></runtime>
      <synic state="on"></synic>
      <stimer state="off"></stimer>
      <reset state="on"></reset>
      <vendor_id state="off" value="myvendor"></vendor_id>
      <frequencies state="off"></frequencies>
      <reenlightenment state="off"></reenlightenment>
      <tlbflush state="on"></tlbflush>
      <ipi state="on"></ipi>
      <evmcs state="off"></evmcs>
    </hyperv>
    <smm></smm>
    <kvm>
      <hidden state="on"></hidden>
    </kvm>
  </features>
  <cpu mode="host-model">
    <topology sockets="1" cores="1" threads="1"></topology>
  </cpu>
  <vcpu placement="static">1</vcpu>
  <iothreads>3</iothreads>
</domain>`, domainType, "%s")

		var convertedDomainppc64leWith5Period = fmt.Sprintf(convertedDomainppc64le,
			`<memballoon model="virtio-non-transitional">
      <stats period="5"></stats>
    </memballoon>`)
		var convertedDomainppc64leWith0Period = fmt.Sprintf(convertedDomainppc64le,
			`<memballoon model="virtio-non-transitional"></memballoon>`)

		var convertedDomainppc64leWithFalseAutoattach = fmt.Sprintf(convertedDomainppc64le,
			`<memballoon model="none"></memballoon>`)
		convertedDomainppc64le = fmt.Sprintf(convertedDomainppc64le,
			`<memballoon model="virtio-non-transitional">
      <stats period="10"></stats>
    </memballoon>`)

		var convertedDomainWithDevicesOnRootBus = fmt.Sprintf(`<domain type="%s" xmlns:qemu="http://libvirt.org/schemas/domain/qemu/1.0">
  <name>mynamespace_testvmi</name>
  <memory unit="b">8388608</memory>
  <os>
    <type arch="x86_64" machine="q35">hvm</type>
    <smbios mode="sysinfo"></smbios>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
      <entry name="serial">e4686d2c-6e8d-4335-b8fd-81bee22f4815</entry>
      <entry name="manufacturer"></entry>
      <entry name="family"></entry>
      <entry name="product"></entry>
      <entry name="sku"></entry>
      <entry name="version"></entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
    <chassis></chassis>
  </sysinfo>
  <devices>
    <interface type="ethernet">
      <address type="pci" domain="0x0000" bus="0x00" slot="0x02" function="0x0"></address>
      <source></source>
      <model type="virtio-non-transitional"></model>
      <alias name="ua-default"></alias>
      <rom enabled="no"></rom>
    </interface>
    <channel type="unix">
      <target name="org.qemu.guest_agent.0" type="virtio"></target>
    </channel>
    <controller type="usb" index="0" model="none">
      <address type="pci" domain="0x0000" bus="0x00" slot="0x03" function="0x0"></address>
    </controller>
    <controller type="virtio-serial" index="0" model="virtio-non-transitional">
      <address type="pci" domain="0x0000" bus="0x00" slot="0x04" function="0x0"></address>
    </controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <graphics type="vnc">
      <listen type="socket" socket="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-vnc"></listen>
    </graphics>
    <memballoon model="virtio-non-transitional">
      <stats period="10"></stats>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x0a" function="0x0"></address>
    </memballoon>
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vda"></target>
      <driver name="qemu" type="raw" iothread="2"></driver>
      <alias name="ua-myvolume"></alias>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x05" function="0x0"></address>
    </disk>
    <disk device="disk" type="file" model="virtio-non-transitional">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="virtio" dev="vdb"></target>
      <driver name="qemu" type="raw" iothread="3"></driver>
      <alias name="ua-nocloud"></alias>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x06" function="0x0"></address>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="sata" dev="sda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-cdrom_tray_unspecified"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/cdrom_tray_open/disk.img"></source>
      <target bus="sata" dev="sdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-cdrom_tray_open"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_unspecified/disk.img"></source>
      <target bus="fdc" dev="fda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-floppy_tray_unspecified"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/floppy_tray_open/disk.img"></source>
      <target bus="fdc" dev="fdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-floppy_tray_open"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/should_default_to_disk/disk.img"></source>
      <target bus="sata" dev="sdc"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-should_default_to_disk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/kubevirt-ephemeral-disk/ephemeral_pvc/disk.qcow2"></source>
      <target bus="sata" dev="sdd"></target>
      <driver cache="none" name="qemu" type="qcow2" iothread="1"></driver>
      <alias name="ua-ephemeral_pvc"></alias>
      <backingStore type="file">
        <format type="raw"></format>
        <source file="/var/run/kubevirt-private/vmi-disks/ephemeral_pvc/disk.img"></source>
      </backingStore>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/secret-disks/secret_test.iso"></source>
      <target bus="sata" dev="sde"></target>
      <serial>D23YZ9W6WA5DJ487</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-secret_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/config-map-disks/configmap_test.iso"></source>
      <target bus="sata" dev="sdf"></target>
      <serial>CVLY623300HK240D</serial>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-configmap_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/pvc_block_test"></source>
      <target bus="sata" dev="sdg"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-pvc_block_test"></alias>
    </disk>
    <disk device="disk" type="block">
      <source dev="/dev/dv_block_test"></source>
      <target bus="sata" dev="sdh"></target>
      <driver cache="writethrough" name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-dv_block_test"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/service-account-disk/service-account.iso"></source>
      <target bus="sata" dev="sdi"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-serviceaccount_test"></alias>
    </disk>
    <input type="tablet" bus="virtio" model="virtio">
      <alias name="ua-tablet0"></alias>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x07" function="0x0"></address>
    </input>
    <serial type="unix">
      <target port="0"></target>
      <source mode="bind" path="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-serial0"></source>
    </serial>
    <console type="pty">
      <target type="serial" port="0"></target>
    </console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="ua-mywatchdog"></alias>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x08" function="0x0"></address>
    </watchdog>
    <rng model="virtio-non-transitional">
      <backend model="random">/dev/urandom</backend>
      <address type="pci" domain="0x0000" bus="0x00" slot="0x09" function="0x0"></address>
    </rng>
  </devices>
  <clock offset="utc" adjustment="reset">
    <timer name="rtc" tickpolicy="catchup" present="yes" track="guest"></timer>
    <timer name="pit" tickpolicy="discard" present="no"></timer>
    <timer name="kvmclock" present="yes"></timer>
    <timer name="hpet" tickpolicy="delay" present="no"></timer>
    <timer name="hypervclock" present="yes"></timer>
  </clock>
  <metadata>
    <kubevirt xmlns="http://kubevirt.io">
      <uid>f4686d2c-6e8d-4335-b8fd-81bee22f4814</uid>
      <graceperiod>
        <deletionGracePeriodSeconds>5</deletionGracePeriodSeconds>
      </graceperiod>
    </kubevirt>
  </metadata>
  <features>
    <acpi></acpi>
    <apic></apic>
    <hyperv>
      <relaxed state="off"></relaxed>
      <vapic state="on"></vapic>
      <spinlocks state="on" retries="4096"></spinlocks>
      <vpindex state="on"></vpindex>
      <runtime state="off"></runtime>
      <synic state="on"></synic>
      <stimer state="off"></stimer>
      <reset state="on"></reset>
      <vendor_id state="off" value="myvendor"></vendor_id>
      <frequencies state="off"></frequencies>
      <reenlightenment state="off"></reenlightenment>
      <tlbflush state="on"></tlbflush>
      <ipi state="on"></ipi>
      <evmcs state="off"></evmcs>
    </hyperv>
    <smm></smm>
    <kvm>
      <hidden state="on"></hidden>
    </kvm>
  </features>
  <cpu mode="host-model">
    <topology sockets="1" cores="1" threads="1"></topology>
  </cpu>
  <vcpu placement="static">1</vcpu>
  <iothreads>3</iothreads>
</domain>`, domainType)

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
				UseEmulation:          true,
				IsBlockPVC:            isBlockPVCMap,
				IsBlockDV:             isBlockDVMap,
				SRIOVDevices:          map[string][]string{},
				SMBios:                TestSmbios,
				GpuDevices:            []string{},
				MemBalloonStatsPeriod: 10,
			}
		})

		It("should use virtio-transitional models if requested", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.DisableHotplug = false
			c.UseVirtioTransitional = true
			dom := vmiToDomain(vmi, c)
			testutils.ExpectVirtioTransitionalOnly(&dom.Spec)
		})

		It("should handle float memory", func() {
			vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory] = resource.MustParse("2222222200m")
			xml := vmiToDomainXML(vmi, c)
			Expect(strings.Contains(xml, `<memory unit="b">2222222</memory>`)).To(BeTrue(), xml)
		})

		table.DescribeTable("should be converted to a libvirt Domain with vmi defaults set", func(arch string, domain string) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			c.Architecture = arch
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			table.Entry("for amd64", "amd64", convertedDomain),
			table.Entry("for ppc64le", "ppc64le", convertedDomainppc64le),
		)

		table.DescribeTable("should be converted to a libvirt Domain", func(arch string, domain string, period uint) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			c.Architecture = arch
			c.MemBalloonStatsPeriod = period
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			table.Entry("when context define 5 period on memballoon device for amd64", "amd64", convertedDomainWith5Period, uint(5)),
			table.Entry("when context define 5 period on memballoon device for ppc64le", "ppc64le", convertedDomainppc64leWith5Period, uint(5)),
			table.Entry("when context define 0 period on memballoon device for amd64 ", "amd64", convertedDomainWith0Period, uint(0)),
			table.Entry("when context define 0 period on memballoon device for ppc64le", "ppc64le", convertedDomainppc64leWith0Period, uint(0)),
		)

		table.DescribeTable("should be converted to a libvirt Domain", func(arch string, domain string) {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			vmi.Spec.Domain.Devices.AutoattachMemBalloon = &_false
			c.Architecture = arch
			Expect(vmiToDomainXML(vmi, c)).To(Equal(domain))
		},
			table.Entry("when Autoattach memballoon device is false for amd64", "amd64", convertedDomainWithFalseAutoattach),
			table.Entry("when Autoattach memballoon device is false for ppc64le", "ppc64le", convertedDomainppc64leWithFalseAutoattach),
		)

		It("should use kvm if present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			Expect(vmiToDomainXMLToDomainSpec(vmi, c).Type).To(Equal(domainType))
		})

		Context("when all addresses should be places at the root complex", func() {
			It("should be converted to a libvirt Domain with vmi defaults set", func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
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

			table.DescribeTable("should convert CPU model", func(model string) {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.CPU = &v1.CPU{
					Cores: 3,
					Model: model,
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

				Expect(domainSpec.CPU.Mode).To(Equal(model), "Expect mode")
			},
				table.Entry(v1.CPUModeHostPassthrough, v1.CPUModeHostPassthrough),
				table.Entry(v1.CPUModeHostModel, v1.CPUModeHostModel),
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

		It("should set disk pci address when specified", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.PciAddress = "0000:81:01.0"
			test_address := api.Address{
				Type:     "pci",
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
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed())
		})

		It("should add a virtio-scsi controller if a scsci disk is present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Disks[0].Disk.Bus = "scsi"
			dom := &api.Domain{}
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, dom, c)).To(Succeed())
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
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, dom, c)).To(Succeed())
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
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should fail when input device is set to keyboard type", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Type = "keyboard"
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should succeed when input device is set to usb bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "usb"
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &api.Domain{}, c)).To(Succeed(), "Expect success")
		})

		It("should succeed when input device bus is empty", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = ""
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Inputs[0].Bus).To(Equal("usb"), "Expect usb bus")
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
			expectedPCIAddress := api.Address{
				Type:     "pci",
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x01",
				Function: "0x0",
			}

			BeforeEach(func() {
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = "0000:81:01.0"
			})

			It("should be set on the domain spec for a non-SRIOV nic", func() {
				domain := vmiToDomain(vmi, c)
				Expect(*domain.Spec.Devices.Interfaces[0].Address).To(Equal(expectedPCIAddress))

			})
			It("should be set on the domain spec for a SRIOV nic", func() {
				iface := &vmi.Spec.Domain.Devices.Interfaces[0]
				iface.SRIOV = &v1.InterfaceSRIOV{}
				c := &ConverterContext{
					VirtualMachine: vmi,
					UseEmulation:   true,
					SRIOVDevices:   map[string][]string{iface.Name: []string{"0000:81:11.1"}},
				}

				domain := vmiToDomain(vmi, c)
				Expect(*domain.Spec.Devices.HostDevices[0].Address).To(Equal(expectedPCIAddress))
			})

		})

		It("should calculate mebibyte from a quantity", func() {
			mi64, _ := resource.ParseQuantity("2G")
			q, err := QuantityToMebiByte(mi64)
			Expect(err).ToNot(HaveOccurred())
			Expect(q).To(BeNumerically("==", 1907))
		})

		It("should fail calculating mebibyte if the quantity is less than 0", func() {
			mi64, _ := resource.ParseQuantity("-2G")
			_, err := QuantityToMebiByte(mi64)
			Expect(err).To(HaveOccurred())
		})

		It("should calculate memory in bytes", func() {
			By("specifying memory 64M")
			m64, _ := resource.ParseQuantity("64M")
			memory, err := QuantityToByte(m64)
			Expect(memory.Value).To(Equal(uint64(64000000)))
			Expect(memory.Unit).To(Equal("b"))
			Expect(err).ToNot(HaveOccurred())

			By("specifying memory 64Mi")
			mi64, _ := resource.ParseQuantity("64Mi")
			memory, err = QuantityToByte(mi64)
			Expect(memory.Value).To(Equal(uint64(67108864)))
			Expect(err).ToNot(HaveOccurred())

			By("specifying memory 3G")
			g3, _ := resource.ParseQuantity("3G")
			memory, err = QuantityToByte(g3)
			Expect(memory.Value).To(Equal(uint64(3000000000)))
			Expect(err).ToNot(HaveOccurred())

			By("specifying memory 3Gi")
			gi3, _ := resource.ParseQuantity("3Gi")
			memory, err = QuantityToByte(gi3)
			Expect(memory.Value).To(Equal(uint64(3221225472)))
			Expect(err).ToNot(HaveOccurred())

			By("specifying memory 45Gi")
			gi45, _ := resource.ParseQuantity("45Gi")
			memory, err = QuantityToByte(gi45)
			Expect(memory.Value).To(Equal(uint64(48318382080)))
			Expect(err).ToNot(HaveOccurred())

			By("specifying memory 451231 bytes")
			b451231, _ := resource.ParseQuantity("451231")
			memory, err = QuantityToByte(b451231)
			Expect(memory.Value).To(Equal(uint64(451231)))
			Expect(err).ToNot(HaveOccurred())

			By("specyfing negative memory size -45Gi")
			m45gi, _ := resource.ParseQuantity("-45Gi")
			_, err = QuantityToByte(m45gi)
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
				UseEmulation: true,
				SMBios:       TestSmbios,
			}
		})

		It("should fail to convert if non network source are present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name := "otherName"
			iface := v1.DefaultBridgeNetworkInterface()
			net := v1.DefaultPodNetwork()
			iface.Name = name
			net.Name = name
			net.Pod = nil
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, *iface)
			vmi.Spec.Networks = append(vmi.Spec.Networks, *net)
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &api.Domain{}, c)).ToNot(Succeed())
		})

		It("should add tcp if protocol not exist", func() {
			iface := v1.Interface{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{v1.Port{Port: 80}}}
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			qemuArg := api.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

			err := configPortForward(&qemuArg, iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(qemuArg.Value).To(Equal(fmt.Sprintf("user,id=%s,hostfwd=tcp::80-:80", iface.Name)))
		})
		It("should not fail for duplicate port with different protocol configuration", func() {
			iface := v1.Interface{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 80}, {Port: 80, Protocol: "UDP"}}}
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			qemuArg := api.Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

			err := configPortForward(&qemuArg, iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(qemuArg.Value).To(Equal(fmt.Sprintf("user,id=%s,hostfwd=tcp::80-:80,hostfwd=udp::80-:80", iface.Name)))
		})
		It("Should create network configuration for slirp device", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name := "otherName"
			iface := v1.Interface{Name: name, InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 80}, {Port: 80, Protocol: "UDP"}}}
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net := v1.DefaultPodNetwork()
			net.Name = name
			vmi.Spec.Networks = []v1.Network{*net}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
		})
		It("Should create two network configuration for slirp device", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 80}, {Port: 80, Protocol: "UDP"}}}
			iface1.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			name2 := "otherName"
			iface2 := v1.Interface{Name: name2, InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 90}}}
			iface2.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net2 := v1.DefaultPodNetwork()
			net2.Name = name2

			vmi.Spec.Networks = []v1.Network{*net1, *net2}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1, iface2}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(4))
		})
		It("Should create two network configuration one for slirp device and one for bridge device", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.DefaultBridgeNetworkInterface()
			iface1.Name = name1
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			name2 := "otherName"
			iface2 := v1.Interface{Name: name2, InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 90}}}
			iface2.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net2 := v1.DefaultPodNetwork()
			net2.Name = name2

			vmi.Spec.Networks = []v1.Network{*net1, *net2}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*iface1, iface2}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
			Expect(len(domain.Spec.QEMUCmd.QEMUArg)).To(Equal(2))
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(2))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
			Expect(domain.Spec.Devices.Interfaces[0].Model.Type).To(Equal("virtio-non-transitional"))
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("user"))
			Expect(domain.Spec.Devices.Interfaces[1].Model.Type).To(Equal("e1000"))
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
				v1.Network{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
				v1.Network{
					Name: "red2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
				v1.Network{
					Name: "default",
					NetworkSource: v1.NetworkSource{
						Pod: &v1.PodNetwork{},
					},
				},
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
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
				v1.Network{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red", Default: true},
					},
				},
				v1.Network{
					Name: "red2",
					NetworkSource: v1.NetworkSource{
						Multus: &v1.MultusNetwork{NetworkName: "red"},
					},
				},
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
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
			Expect(domain).ToNot(Equal(nil))
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].BootOrder).NotTo(BeNil())
			Expect(domain.Spec.Devices.Interfaces[0].BootOrder.Order).To(Equal(uint(bootOrder)))
			Expect(domain.Spec.Devices.Interfaces[1].BootOrder).To(BeNil())
		})
		It("Should create network configuration for masquerade interface", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			iface1.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			vmi.Spec.Networks = []v1.Network{*net1}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(1))
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("ethernet"))
		})
		It("Should create network configuration for masquerade interface and the pod network and a secondary network using multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			iface1.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
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
			Expect(domain).ToNot(Equal(nil))
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
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c)).To(HaveOccurred(), "conversion should fail because a macvtap interface requires a multus network attachment")
		})
	})

	Context("graphics and video device", func() {

		table.DescribeTable("should check autoAttachGraphicsDevices", func(autoAttach *bool, devices int) {

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
			domain := vmiToDomain(&vmi, &ConverterContext{UseEmulation: true})
			Expect(domain.Spec.Devices.Video).To(HaveLen(devices))
			Expect(domain.Spec.Devices.Graphics).To(HaveLen(devices))

		},
			table.Entry("and add the graphics and video device if it is not set", nil, 1),
			table.Entry("and add the graphics and video device if it is set to true", True(), 1),
			table.Entry("and not add the graphics and video device if it is set to false", False(), 0),
		)
	})

	Context("serial console", func() {

		table.DescribeTable("should check autoAttachSerialConsole", func(autoAttach *bool, devices int) {

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
			domain := vmiToDomain(&vmi, &ConverterContext{UseEmulation: true})
			Expect(domain.Spec.Devices.Serials).To(HaveLen(devices))
			Expect(domain.Spec.Devices.Consoles).To(HaveLen(devices))

		},
			table.Entry("and add the serial console if it is not set", nil, 1),
			table.Entry("and add the serial console if it is set to true", True(), 1),
			table.Entry("and not add the serial console if it is set to false", False(), 0),
		)
	})

	Context("IOThreads", func() {
		_false := false
		_true := true

		table.DescribeTable("Should use correct IOThreads policies", func(policy v1.IOThreadsPolicy, cpuCores int, threadCount int, threadIDs []int) {
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
											Bus: "virtio",
										},
									},
									DedicatedIOThread: &_true,
								},
								{
									Name: "shared",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
									DedicatedIOThread: &_false,
								},
								{
									Name: "omitted1",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name: "omitted2",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name: "omitted3",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name: "omitted4",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
										},
									},
								},
								{
									Name: "omitted5",
									DiskDevice: v1.DiskDevice{
										Disk: &v1.DiskTarget{
											Bus: "virtio",
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

			domain := vmiToDomain(&vmi, &ConverterContext{UseEmulation: true})
			Expect(domain.Spec.IOThreads).ToNot(BeNil())
			Expect(int(domain.Spec.IOThreads.IOThreads)).To(Equal(threadCount))
			for idx, disk := range domain.Spec.Devices.Disks {
				Expect(disk.Driver.IOThread).ToNot(BeNil())
				Expect(int(*disk.Driver.IOThread)).To(Equal(threadIDs[idx]))
			}
		},
			table.Entry("using a shared policy with 1 CPU", v1.IOThreadsPolicyShared, 1, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			table.Entry("using a shared policy with 2 CPUs", v1.IOThreadsPolicyShared, 2, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			table.Entry("using a shared policy with 3 CPUs", v1.IOThreadsPolicyShared, 2, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			table.Entry("using an auto policy with 1 CPU", v1.IOThreadsPolicyAuto, 1, 2, []int{2, 1, 1, 1, 1, 1, 1}),
			table.Entry("using an auto policy with 2 CPUs", v1.IOThreadsPolicyAuto, 2, 4, []int{4, 1, 2, 3, 1, 2, 3}),
			table.Entry("using an auto policy with 3 CPUs", v1.IOThreadsPolicyAuto, 3, 6, []int{6, 1, 2, 3, 4, 5, 1}),
			table.Entry("using an auto policy with 4 CPUs", v1.IOThreadsPolicyAuto, 4, 7, []int{7, 1, 2, 3, 4, 5, 6}),
			table.Entry("using an auto policy with 5 CPUs", v1.IOThreadsPolicyAuto, 5, 7, []int{7, 1, 2, 3, 4, 5, 6}),
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
							Bus: "virtio",
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
					Disk: &v1.DiskTarget{Bus: "virtio"},
				},
			}
			apiDisk := api.Disk{}
			devicePerBus := map[string]deviceNamer{}
			numQueues := uint(2)
			Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, &numQueues)
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
			err := Convert_v1_Disk_To_api_Disk(context, &v1Disk, &apiDisk, devicePerBus, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
			Expect(apiDisk.Driver.Queues).To(BeNil(), "expected no queues to be requested")
		})

		It("should honor multiQueue setting", func() {
			var expectedQueues uint = 2
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores: 2,
			}

			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true, SMBios: &cmdv1.SMBios{}})
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
				UseEmulation: true,
				SMBios:       &cmdv1.SMBios{},
			}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &api.IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			err := formatDomainIOThreadPin(vmi, domain, c)
			Expect(err).ToNot(HaveOccurred())
			expectedLayout := []api.CPUTuneIOThreadPin{
				api.CPUTuneIOThreadPin{IOThread: 1, CPUSet: "5,6,7"},
				api.CPUTuneIOThreadPin{IOThread: 2, CPUSet: "8,9,10"},
				api.CPUTuneIOThreadPin{IOThread: 3, CPUSet: "11,12,13"},
				api.CPUTuneIOThreadPin{IOThread: 4, CPUSet: "14,15,16"},
				api.CPUTuneIOThreadPin{IOThread: 5, CPUSet: "17,18"},
				api.CPUTuneIOThreadPin{IOThread: 6, CPUSet: "19,20"},
			}
			isExpectedThreadsLayout := reflect.DeepEqual(expectedLayout, domain.Spec.CPUTune.IOThreadPin)
			Expect(isExpectedThreadsLayout).To(BeTrue())

		})
		It("should pack iothreads equally on available vcpus, if there are more iothreads than vcpus", func() {
			vmi.Spec.Domain.CPU.Cores = 2
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			c := &ConverterContext{CPUSet: []int{5, 6}, UseEmulation: true}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &api.IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			err := formatDomainIOThreadPin(vmi, domain, c)
			Expect(err).ToNot(HaveOccurred())
			expectedLayout := []api.CPUTuneIOThreadPin{
				api.CPUTuneIOThreadPin{IOThread: 1, CPUSet: "6"},
				api.CPUTuneIOThreadPin{IOThread: 2, CPUSet: "5"},
				api.CPUTuneIOThreadPin{IOThread: 3, CPUSet: "6"},
				api.CPUTuneIOThreadPin{IOThread: 4, CPUSet: "5"},
				api.CPUTuneIOThreadPin{IOThread: 5, CPUSet: "6"},
				api.CPUTuneIOThreadPin{IOThread: 6, CPUSet: "5"},
			}
			isExpectedThreadsLayout := reflect.DeepEqual(expectedLayout, domain.Spec.CPUTune.IOThreadPin)
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

			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
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
			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested vCPUs")
		})

		It("should not assign queues to a non-virtio devices", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			Expect(domain.Spec.Devices.Interfaces[0].Driver).To(BeNil(),
				"queues should not be set for models other than virtio")
		})

		It("should cap the maximum number of queues", func() {
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores:   512,
				Sockets: 1,
				Threads: 2,
			}
			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			expectedNumberQueues := uint(multiQueueMaxQueues)
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedNumberQueues),
				"should be capped to the maximum number of queues on tap devices")
		})
	})

	Context("sriov", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "testvmi",
				Namespace: "mynamespace",
			},
		}
		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
		vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultBridgeNetworkInterface()}

		sriovInterface := v1.Interface{
			Name: "sriov",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				SRIOV: &v1.InterfaceSRIOV{},
			},
		}
		vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, sriovInterface)
		sriovNetwork := v1.Network{
			Name: "sriov",
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{NetworkName: "sriov"},
			},
		}
		vmi.Spec.Networks = append(vmi.Spec.Networks, sriovNetwork)

		sriovInterface2 := v1.Interface{
			Name: "sriov2",
			InterfaceBindingMethod: v1.InterfaceBindingMethod{
				SRIOV: &v1.InterfaceSRIOV{},
			},
		}
		vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, sriovInterface2)
		sriovNetwork2 := v1.Network{
			Name: "sriov2",
			NetworkSource: v1.NetworkSource{
				Multus: &v1.MultusNetwork{NetworkName: "sriov2"},
			},
		}
		vmi.Spec.Networks = append(vmi.Spec.Networks, sriovNetwork2)

		It("should convert sriov interfaces into host devices", func() {
			c := &ConverterContext{
				UseEmulation: true,
				SRIOVDevices: map[string][]string{
					"sriov":  []string{"0000:81:11.1"},
					"sriov2": []string{"0000:81:11.2"},
				},
			}
			domain := vmiToDomain(vmi, c)

			// check that new sriov interfaces are *not* represented in xml domain as interfaces
			Expect(len(domain.Spec.Devices.Interfaces)).To(Equal(1))

			// check that the sriov interfaces are represented as PCI host devices
			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x0000"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x81"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x11"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x1"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Domain).To(Equal("0x0000"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Bus).To(Equal("0x81"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Slot).To(Equal("0x11"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Function).To(Equal("0x2"))
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
				UseEmulation:   true,
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

			It("should configure the EFI bootloader if EFI insecure option", func() {

				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{
							SecureBoot: False(),
						},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader.ReadOnly).To(Equal("yes"))
				Expect(domainSpec.OS.BootLoader.Type).To(Equal("pflash"))
				Expect(domainSpec.OS.BootLoader.Secure).To(Equal("no"))
				Expect(path.Base(domainSpec.OS.BootLoader.Path)).To(Equal(EFICode))
				Expect(path.Base(domainSpec.OS.NVRam.Template)).To(Equal(EFIVars))
				Expect(domainSpec.OS.NVRam.NVRam).To(Equal("/tmp/mynamespace_testvmi"))
			})

			It("should configure the EFI bootloader if EFI secure option", func() {
				vmi.Spec.Domain.Firmware = &v1.Firmware{
					Bootloader: &v1.Bootloader{
						EFI: &v1.EFI{},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader.ReadOnly).To(Equal("yes"))
				Expect(domainSpec.OS.BootLoader.Type).To(Equal("pflash"))
				Expect(domainSpec.OS.BootLoader.Secure).To(Equal("yes"))
				Expect(path.Base(domainSpec.OS.BootLoader.Path)).To(Equal(EFICodeSecureBoot))
				Expect(path.Base(domainSpec.OS.NVRam.Template)).To(Equal(EFIVarsSecureBoot))
				Expect(domainSpec.OS.NVRam.NVRam).To(Equal("/tmp/mynamespace_testvmi"))
			})
		})
	})

	Context("Legacy GPU resource request", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "testvmi",
				Namespace: "mynamespace",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							v1.GPU{
								Name: "vendor.com/gpu_name",
							},
						},
					},
				},
			},
		}

		v1.SetObjectDefaults_VirtualMachineInstance(vmi)

		It("should convert GPU resource request into host devices", func() {
			c := &ConverterContext{
				UseEmulation: true,
				GpuDevices:   []string{"2609:19:90.0", "2609:19:90.1"},
			}

			domain := vmiToDomain(vmi, c)

			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x0"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Function).To(Equal("0x1"))

		})

		It("should convert GPU resource request into host devices for VGPU", func() {
			c := &ConverterContext{
				UseEmulation: true,
				VgpuDevices:  []string{"aa618089-8b16-4d01-a136-25a0f3c73123", "aa618089-8b16-4d01-a136-25a0f3c73124"},
			}

			domain := vmiToDomain(vmi, c)

			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("mdev"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.UUID).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73123"))
			Expect(domain.Spec.Devices.HostDevices[0].Mode).To(Equal("subsystem"))
			Expect(domain.Spec.Devices.HostDevices[0].Model).To(Equal("vfio-pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("mdev"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.UUID).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73124"))
			Expect(domain.Spec.Devices.HostDevices[1].Mode).To(Equal("subsystem"))
			Expect(domain.Spec.Devices.HostDevices[1].Model).To(Equal("vfio-pci"))

		})
	})
	Context("GPU resource request", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "testvmi",
				Namespace: "mynamespace",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						GPUs: []v1.GPU{
							v1.GPU{
								DeviceName: "vendor.com/gpu_name",
								Name:       "gpu_name",
							},
						},
					},
				},
			},
		}

		v1.SetObjectDefaults_VirtualMachineInstance(vmi)

		It("should convert GPU resource request into host devices", func() {
			c := &ConverterContext{
				UseEmulation: true,
				HostDevices: map[string]HostDevicesList{
					"vendor.com/gpu_name": HostDevicesList{
						Type:     HostDevicePCI,
						AddrList: []string{"2609:19:90.0", "2609:19:90.1"},
					},
				},
			}

			domain := vmiToDomain(vmi, c)

			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(1))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x0"))
			Expect(domain.Spec.Devices.HostDevices[0].Alias.Name).To(Equal("gpu_name"))

		})
		It("should convert 2 GPU resource request into host devices", func() {
			c := &ConverterContext{
				UseEmulation: true,
				HostDevices: map[string]HostDevicesList{
					"vendor.com/gpu_name": HostDevicesList{
						Type:     HostDevicePCI,
						AddrList: []string{"2609:19:90.0", "2609:19:90.1"},
					},
				},
			}
			newGpu := v1.GPU{
				DeviceName: "vendor.com/gpu_name",
				Name:       "gpu_name1",
			}
			vmi.Spec.Domain.Devices.GPUs = append(vmi.Spec.Domain.Devices.GPUs, newGpu)

			domain := vmiToDomain(vmi, c)
			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x0"))
			Expect(domain.Spec.Devices.HostDevices[0].Alias.Name).To(Equal("gpu_name"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.Function).To(Equal("0x1"))
			Expect(domain.Spec.Devices.HostDevices[1].Alias.Name).To(Equal("gpu_name1"))

		})

		It("should convert GPU resource request into host devices for VGPU", func() {
			c := &ConverterContext{
				UseEmulation: true,
				HostDevices: map[string]HostDevicesList{
					"vendor.com/gpu_name": HostDevicesList{
						Type:     HostDevicePCI,
						AddrList: []string{"2609:19:90.0", "2609:19:90.1"},
					},
					"vendor.com/vgpu_name": HostDevicesList{
						Type:     HostDeviceMDEV,
						AddrList: []string{"aa618089-8b16-4d01-a136-25a0f3c73123", "aa618089-8b16-4d01-a136-25a0f3c73124"},
					},
				},
			}
			gpus := []v1.GPU{
				v1.GPU{
					DeviceName: "vendor.com/gpu_name",
					Name:       "gpu_name",
				},
				v1.GPU{
					DeviceName: "vendor.com/vgpu_name",
					Name:       "vgpu_name1",
				},
			}
			vmi.Spec.Domain.Devices.GPUs = gpus

			domain := vmiToDomain(vmi, c)

			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x0"))
			Expect(domain.Spec.Devices.HostDevices[0].Alias.Name).To(Equal("gpu_name"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("mdev"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.UUID).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73123"))
			Expect(domain.Spec.Devices.HostDevices[1].Mode).To(Equal("subsystem"))
			Expect(domain.Spec.Devices.HostDevices[1].Model).To(Equal("vfio-pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Alias.Name).To(Equal("vgpu_name1"))
		})
	})

	Context("HostDevices resource request", func() {
		vmi := &v1.VirtualMachineInstance{
			ObjectMeta: k8smeta.ObjectMeta{
				Name:      "testvmi",
				Namespace: "mynamespace",
				UID:       "1234",
			},
			Spec: v1.VirtualMachineInstanceSpec{
				Domain: v1.DomainSpec{
					Devices: v1.Devices{
						HostDevices: []v1.HostDevice{
							v1.HostDevice{
								DeviceName: "vendor.com/pci_name",
								Name:       "pci_name",
							},
							v1.HostDevice{
								DeviceName: "vendor.com/mdev_name",
								Name:       "mdev_name",
							},
						},
					},
				},
			},
		}

		v1.SetObjectDefaults_VirtualMachineInstance(vmi)
		It("should convert HostDevices resources request into host devices for MDEV and PCI", func() {
			c := &ConverterContext{
				UseEmulation: true,
				HostDevices: map[string]HostDevicesList{
					"vendor.com/pci_name": HostDevicesList{
						Type:     HostDevicePCI,
						AddrList: []string{"2609:19:90.0", "2609:19:90.1"},
					},
					"vendor.com/mdev_name": HostDevicesList{
						Type:     HostDeviceMDEV,
						AddrList: []string{"aa618089-8b16-4d01-a136-25a0f3c73123", "aa618089-8b16-4d01-a136-25a0f3c73124"},
					},
				},
			}
			domain := vmiToDomain(vmi, c)

			Expect(len(domain.Spec.Devices.HostDevices)).To(Equal(2))
			Expect(domain.Spec.Devices.HostDevices[0].Type).To(Equal("pci"))
			Expect(domain.Spec.Devices.HostDevices[0].Managed).To(Equal("yes"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Domain).To(Equal("0x2609"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Bus).To(Equal("0x19"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Slot).To(Equal("0x90"))
			Expect(domain.Spec.Devices.HostDevices[0].Source.Address.Function).To(Equal("0x0"))
			Expect(domain.Spec.Devices.HostDevices[0].Alias.Name).To(Equal("pci_name"))
			Expect(domain.Spec.Devices.HostDevices[1].Type).To(Equal("mdev"))
			Expect(domain.Spec.Devices.HostDevices[1].Source.Address.UUID).To(Equal("aa618089-8b16-4d01-a136-25a0f3c73123"))
			Expect(domain.Spec.Devices.HostDevices[1].Mode).To(Equal("subsystem"))
			Expect(domain.Spec.Devices.HostDevices[1].Model).To(Equal("vfio-pci"))
			Expect(domain.Spec.Devices.HostDevices[1].Alias.Name).To(Equal("mdev_name"))
		})
	})

	Context("hotplug", func() {
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
				UseEmulation:   true,
			}
		})

		It("should automatically add virtio-scsi controller", func() {
			domain := vmiToDomain(vmi, c)
			Expect(len(domain.Spec.Devices.Controllers)).To(Equal(3))
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
			Expect(len(domain.Spec.Devices.Controllers)).To(Equal(2))
		})
	})

})

var _ = Describe("popSRIOVPCIAddress", func() {
	It("fails on empty map", func() {
		_, _, err := popSRIOVPCIAddress("testnet", map[string][]string{})
		Expect(err).To(HaveOccurred())
	})
	It("fails on empty map entry", func() {
		_, _, err := popSRIOVPCIAddress("testnet", map[string][]string{"testnet": []string{}})
		Expect(err).To(HaveOccurred())
	})
	It("pops the next address from a non-empty slice", func() {
		addrsMap := map[string][]string{"testnet": []string{"0000:81:11.1", "0001:02:00.0"}}
		addr, rest, err := popSRIOVPCIAddress("testnet", addrsMap)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr).To(Equal("0000:81:11.1"))
		Expect(len(rest["testnet"])).To(Equal(1))
		Expect(rest["testnet"][0]).To(Equal("0001:02:00.0"))
	})
	It("pops the next address from all tracked networks", func() {
		addrsMap := map[string][]string{
			"testnet1": []string{"0000:81:11.1", "0001:02:00.0"},
			"testnet2": []string{"0000:81:11.1", "0001:02:00.0"},
		}
		addr, rest, err := popSRIOVPCIAddress("testnet1", addrsMap)
		Expect(err).ToNot(HaveOccurred())
		Expect(addr).To(Equal("0000:81:11.1"))
		Expect(len(rest["testnet1"])).To(Equal(1))
		Expect(rest["testnet1"][0]).To(Equal("0001:02:00.0"))
		Expect(len(rest["testnet2"])).To(Equal(1))
		Expect(rest["testnet2"][0]).To(Equal("0001:02:00.0"))
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
		res, index := makeDeviceName("test1", "virtio", prefixMap)
		Expect(res).To(Equal("vda"))
		Expect(index).To(Equal(0))
		for i := 2; i < 10; i++ {
			makeDeviceName(fmt.Sprintf("test%d", i), "virtio", prefixMap)
		}
		prefix := getPrefixFromBus("virtio")
		delete(prefixMap[prefix].usedDeviceMap, "vdd")
		By("Verifying next value is vdd")
		res, index = makeDeviceName("something", "virtio", prefixMap)
		Expect(index).To(Equal(3))
		Expect(res).To(Equal("vdd"))
		res, index = makeDeviceName("something_else", "virtio", prefixMap)
		Expect(res).To(Equal("vdj"))
		Expect(index).To(Equal(9))
		By("verifying existing returns correct value")
		res, index = makeDeviceName("something", "virtio", prefixMap)
		Expect(res).To(Equal("vdd"))
		Expect(index).To(Equal(3))
		By("Verifying a new bus returns from start")
		res, index = makeDeviceName("something", "scsi", prefixMap)
		Expect(res).To(Equal("sda"))
		Expect(index).To(Equal(0))
	})
})

func diskToDiskXML(disk *v1.Disk) string {
	devicePerBus := make(map[string]deviceNamer)
	libvirtDisk := &api.Disk{}
	Expect(Convert_v1_Disk_To_api_Disk(&ConverterContext{UseVirtioTransitional: false}, disk, libvirtDisk, devicePerBus, nil)).To(Succeed())
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
	Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c)).To(Succeed())
	api.NewDefaulter(c.Architecture).SetObjectDefaults_Domain(domain)
	return domain
}

func xmlToDomainSpec(data string) *api.DomainSpec {
	newDomain := &api.DomainSpec{}
	err := xml.Unmarshal([]byte(data), newDomain)
	newDomain.XMLName.Local = ""
	newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	Expect(err).To(BeNil())
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
