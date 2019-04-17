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

package api

import (
	"encoding/xml"
	"fmt"
	"os"
	"reflect"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Converter", func() {

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
			var convertedDisk = `<Disk device="disk" type="">
  <source></source>
  <target bus="virtio" dev="vda"></target>
  <driver name="qemu" type=""></driver>
  <alias name="ua-mydisk"></alias>
  <boot order="1"></boot>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			Expect(xml).To(Equal(convertedDisk))
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
			var convertedDisk = `<Disk device="disk" type="">
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
					Name: "serviceaccount_test",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: "testaccount",
						},
					},
				},
			}

			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}

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
  <memory unit="B">8388608</memory>
  <os>
    <type arch="x86_64" machine="q35">hvm</type>
  </os>
  <sysinfo type="smbios">
    <system>
      <entry name="uuid">e4686d2c-6e8d-4335-b8fd-81bee22f4814</entry>
      <entry name="serial">e4686d2c-6e8d-4335-b8fd-81bee22f4815</entry>
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
  </sysinfo>
  <devices>
    <interface type="bridge">
      <source bridge="k6t-eth0"></source>
      <model type="virtio"></model>
      <alias name="ua-default"></alias>
    </interface>
    <channel type="unix">
      <target name="org.qemu.guest_agent.0" type="virtio"></target>
    </channel>
    <controller type="usb" index="0" model="none"></controller>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <graphics type="vnc">
      <listen type="socket" socket="/var/run/kubevirt-private/f4686d2c-6e8d-4335-b8fd-81bee22f4814/virt-vnc"></listen>
    </graphics>
    <memballoon model="none"></memballoon>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vda"></target>
      <driver name="qemu" type="raw" iothread="2"></driver>
      <alias name="ua-myvolume"></alias>
    </disk>
    <disk device="disk" type="file">
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
      <source file="/var/run/kubevirt-private/vmi-disks/volume1/disk.img"></source>
      <target bus="sata" dev="sdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-cdrom_tray_open"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume2/disk.img"></source>
      <target bus="fdc" dev="fda" tray="closed"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-floppy_tray_unspecified"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume3/disk.img"></source>
      <target bus="fdc" dev="fdb" tray="open"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <readonly></readonly>
      <alias name="ua-floppy_tray_open"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume4/disk.img"></source>
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
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/service-account-disk/service-account.iso"></source>
      <target bus="sata" dev="sdh"></target>
      <driver name="qemu" type="raw" iothread="1"></driver>
      <alias name="ua-serviceaccount_test"></alias>
    </disk>
    <input type="tablet" bus="virtio">
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
    <rng model="virtio">
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
				UseEmulation: true,
				IsBlockPVC:   isBlockPVCMap,
				SRIOVDevices: map[string][]string{},
			}
		})

		It("should be converted to a libvirt Domain with vmi defaults set", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Rng = &v1.Rng{}
			Expect(vmiToDomainXML(vmi, c)).To(Equal(convertedDomain))
		})

		It("should use kvm if present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			Expect(vmiToDomainXMLToDomainSpec(vmi, c).Type).To(Equal(domainType))
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
			test_address := Address{
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
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &Domain{}, c)).ToNot(Succeed())
		})

		It("should not disable usb controller when usb device is present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "usb"
			domain := vmiToDomain(vmi, c)
			disabled := false
			for _, controller := range domain.Spec.Devices.Controllers {
				if controller.Type == "usb" && controller.Model == "none" {
					disabled = !disabled
				}
			}

			Expect(disabled).To(Equal(false), "Expect controller not to be disabled")
		})

		It("should fail when input device is set to ps2 bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "ps2"
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should fail when input device is set to keyboard type", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Type = "keyboard"
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &Domain{}, c)).ToNot(Succeed(), "Expect error")
		})

		It("should succeed when input device is set to usb bus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Inputs[0].Bus = "usb"
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &Domain{}, c)).To(Succeed(), "Expect success")
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

		It("should set nic pci address when specified", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces[0].PciAddress = "0000:81:01.0"
			test_address := Address{
				Type:     "pci",
				Domain:   "0x0000",
				Bus:      "0x81",
				Slot:     "0x01",
				Function: "0x0",
			}
			domain := vmiToDomain(vmi, c)
			Expect(*domain.Spec.Devices.Interfaces[0].Address).To(Equal(test_address))
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
			Expect(memory.Unit).To(Equal("B"))
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

			Expect(domainSpec.Memory.Value).To(Equal(uint64(8388608)))
			Expect(domainSpec.Memory.Unit).To(Equal("B"))
		})

		It("should use guest memory instead of requested memory if present", func() {
			guestMemory := resource.MustParse("123Mi")
			vmi.Spec.Domain.Memory = &v1.Memory{
				Guest: &guestMemory,
			}
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)

			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

			Expect(domainSpec.Memory.Value).To(Equal(uint64(128974848)))
			Expect(domainSpec.Memory.Unit).To(Equal("B"))
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
			}
		})

		It("should fail to convert if non network source are present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name := "otherName"
			iface := v1.DefaultNetworkInterface()
			net := v1.DefaultPodNetwork()
			iface.Name = name
			net.Name = name
			net.Pod = nil
			vmi.Spec.Domain.Devices.Interfaces = append(vmi.Spec.Domain.Devices.Interfaces, *iface)
			vmi.Spec.Networks = append(vmi.Spec.Networks, *net)
			Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, &Domain{}, c)).ToNot(Succeed())
		})

		It("should add tcp if protocol not exist", func() {
			iface := v1.Interface{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{v1.Port{Port: 80}}}
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			qemuArg := Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

			err := configPortForward(&qemuArg, iface)
			Expect(err).ToNot(HaveOccurred())
			Expect(qemuArg.Value).To(Equal(fmt.Sprintf("user,id=%s,hostfwd=tcp::80-:80", iface.Name)))
		})
		It("should not fail for duplicate port with different protocol configuration", func() {
			iface := v1.Interface{Name: "test", InterfaceBindingMethod: v1.InterfaceBindingMethod{}, Ports: []v1.Port{{Port: 80}, {Port: 80, Protocol: "UDP"}}}
			iface.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			qemuArg := Arg{Value: fmt.Sprintf("user,id=%s", iface.Name)}

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

			iface1 := v1.DefaultNetworkInterface()
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
			Expect(domain.Spec.Devices.Interfaces[0].Type).To(Equal("bridge"))
			Expect(domain.Spec.Devices.Interfaces[0].Model.Type).To(Equal("virtio"))
			Expect(domain.Spec.Devices.Interfaces[1].Type).To(Equal("user"))
			Expect(domain.Spec.Devices.Interfaces[1].Model.Type).To(Equal("e1000"))
		})
		It("Should set domain interface source correctly for multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
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
			Expect(domain.Spec.Devices.Interfaces[0].Source.Bridge).To(Equal("k6t-net1"))
			Expect(domain.Spec.Devices.Interfaces[1].Source.Bridge).To(Equal("k6t-net2"))
			Expect(domain.Spec.Devices.Interfaces[2].Source.Bridge).To(Equal("k6t-eth0"))
		})
		It("Should set domain interface source correctly for default multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
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
			Expect(domain.Spec.Devices.Interfaces[0].Source.Bridge).To(Equal("k6t-eth0"))
			Expect(domain.Spec.Devices.Interfaces[1].Source.Bridge).To(Equal("k6t-net1"))
		})
		It("Should set domain interface source correctly for genie", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{
				*v1.DefaultNetworkInterface(),
				*v1.DefaultNetworkInterface(),
			}
			vmi.Spec.Domain.Devices.Interfaces[0].Name = "red1"
			vmi.Spec.Domain.Devices.Interfaces[1].Name = "red2"
			vmi.Spec.Networks = []v1.Network{
				v1.Network{
					Name: "red1",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.GenieNetwork{NetworkName: "red"},
					},
				},
				v1.Network{
					Name: "red2",
					NetworkSource: v1.NetworkSource{
						Genie: &v1.GenieNetwork{NetworkName: "red"},
					},
				},
			}

			domain := vmiToDomain(vmi, c)
			Expect(domain).ToNot(Equal(nil))
			Expect(domain.Spec.Devices.Interfaces).To(HaveLen(2))
			Expect(domain.Spec.Devices.Interfaces[0].Source.Bridge).To(Equal("k6t-eth0"))
			Expect(domain.Spec.Devices.Interfaces[1].Source.Bridge).To(Equal("k6t-eth1"))
		})
		It("should allow setting boot order", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name1"
			name2 := "Name2"
			iface1 := v1.DefaultNetworkInterface()
			iface2 := v1.DefaultNetworkInterface()
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
			Expect(domain.Spec.Devices.Interfaces[0].Source.Bridge).To(Equal("k6t-eth0"))
		})
		It("Should create network configuration for masquerade interface and the pod network and a secondary network using multus", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			name1 := "Name"

			iface1 := v1.Interface{Name: name1, InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}}
			iface1.InterfaceBindingMethod.Slirp = &v1.InterfaceSlirp{}
			net1 := v1.DefaultPodNetwork()
			net1.Name = name1

			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{iface1, *v1.DefaultNetworkInterface()}
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
			Expect(domain.Spec.Devices.Interfaces[0].Source.Bridge).To(Equal("k6t-eth0"))
			Expect(domain.Spec.Devices.Interfaces[1].Source.Bridge).To(Equal("k6t-net1"))
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

		BeforeEach(func() {
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
					Disk: &v1.DiskTarget{},
				},
			}
			apiDisk := Disk{}
			devicePerBus := map[string]int{}
			numQueues := uint(2)
			Convert_v1_Disk_To_api_Disk(&v1Disk, &apiDisk, devicePerBus, &numQueues)
			Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
			Expect(*(apiDisk.Driver.Queues)).To(Equal(expectedQueues), "expected queues to be 2")
		})

		It("should not assign queues to a device if omitted", func() {
			v1Disk := v1.Disk{
				DiskDevice: v1.DiskDevice{
					Disk: &v1.DiskTarget{},
				},
			}
			apiDisk := Disk{}
			devicePerBus := map[string]int{}
			Convert_v1_Disk_To_api_Disk(&v1Disk, &apiDisk, devicePerBus, nil)
			Expect(apiDisk.Device).To(Equal("disk"), "expected disk device to be defined")
			Expect(apiDisk.Driver.Queues).To(BeNil(), "expected no queues to be requested")
		})

		It("should honor multiQueue setting", func() {
			var expectedQueues uint = 2

			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			Expect(*(domain.Spec.Devices.Disks[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested CPUs")
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
			vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceCPU] = resource.MustParse("16")
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			c := &ConverterContext{CPUSet: []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
				UseEmulation: true,
			}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			err := formatDomainIOThreadPin(vmi, domain, c)
			Expect(err).ToNot(HaveOccurred())
			expectedLayout := []CPUTuneIOThreadPin{
				CPUTuneIOThreadPin{IOThread: 1, CPUSet: "5,6,7"},
				CPUTuneIOThreadPin{IOThread: 2, CPUSet: "8,9,10"},
				CPUTuneIOThreadPin{IOThread: 3, CPUSet: "11,12,13"},
				CPUTuneIOThreadPin{IOThread: 4, CPUSet: "14,15,16"},
				CPUTuneIOThreadPin{IOThread: 5, CPUSet: "17,18"},
				CPUTuneIOThreadPin{IOThread: 6, CPUSet: "19,20"},
			}
			isExpectedThreadsLayout := reflect.DeepEqual(expectedLayout, domain.Spec.CPUTune.IOThreadPin)
			Expect(isExpectedThreadsLayout).To(BeTrue())

		})
		It("should pack iothreads equally on available vcpus, if there are more iothreads than vcpus", func() {
			vmi.Spec.Domain.CPU.Cores = 2
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			c := &ConverterContext{CPUSet: []int{5, 6}, UseEmulation: true}
			domain := vmiToDomain(vmi, c)
			domain.Spec.IOThreads = &IOThreads{}
			domain.Spec.IOThreads.IOThreads = uint(6)

			err := formatDomainIOThreadPin(vmi, domain, c)
			Expect(err).ToNot(HaveOccurred())
			expectedLayout := []CPUTuneIOThreadPin{
				CPUTuneIOThreadPin{IOThread: 1, CPUSet: "6"},
				CPUTuneIOThreadPin{IOThread: 2, CPUSet: "5"},
				CPUTuneIOThreadPin{IOThread: 3, CPUSet: "6"},
				CPUTuneIOThreadPin{IOThread: 4, CPUSet: "5"},
				CPUTuneIOThreadPin{IOThread: 5, CPUSet: "6"},
				CPUTuneIOThreadPin{IOThread: 6, CPUSet: "5"},
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

			vmi.Spec.Domain.Devices.NetworkInterfaceMultiQueue = True()
			vmi.Spec.Domain.Resources.Requests = k8sv1.ResourceList{
				k8sv1.ResourceMemory: resource.MustParse("8192Ki"),
				k8sv1.ResourceCPU:    resource.MustParse("2"),
			}
		})

		It("should assign queues to a device if requested", func() {
			var expectedQueues uint = 2

			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			Expect(*(domain.Spec.Devices.Interfaces[0].Driver.Queues)).To(Equal(expectedQueues),
				"expected number of queues to equal number of requested CPUs")
		})

		It("should not assign queues to a non-virtio devices", func() {
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
			domain := vmiToDomain(vmi, &ConverterContext{UseEmulation: true})
			Expect(domain.Spec.Devices.Interfaces[0].Driver).To(BeNil(),
				"queues should not be set for models other than virtio")
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
						EFI: &v1.EFI{},
					},
				}
				domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)
				Expect(domainSpec.OS.BootLoader.ReadOnly).To(Equal("yes"))
				Expect(domainSpec.OS.BootLoader.Type).To(Equal("pflash"))
				Expect(domainSpec.OS.BootLoader.Path).To(Equal(EFIPath))
				Expect(domainSpec.OS.NVRam.Template).To(Equal(EFIVarsPath))
				Expect(domainSpec.OS.NVRam.NVRam).To(Equal("/tmp/mynamespace_testvmi"))
			})
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

func diskToDiskXML(disk *v1.Disk) string {
	devicePerBus := make(map[string]int)
	libvirtDisk := &Disk{}
	Expect(Convert_v1_Disk_To_api_Disk(disk, libvirtDisk, devicePerBus, nil)).To(Succeed())
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

func vmiToDomain(vmi *v1.VirtualMachineInstance, c *ConverterContext) *Domain {
	domain := &Domain{}
	Expect(Convert_v1_VirtualMachine_To_api_Domain(vmi, domain, c)).To(Succeed())
	SetObjectDefaults_Domain(domain)
	return domain
}

func xmlToDomainSpec(data string) *DomainSpec {
	newDomain := &DomainSpec{}
	err := xml.Unmarshal([]byte(data), newDomain)
	newDomain.XMLName.Local = ""
	newDomain.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	Expect(err).To(BeNil())
	return newDomain
}

func vmiToDomainXMLToDomainSpec(vmi *v1.VirtualMachineInstance, c *ConverterContext) *DomainSpec {
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
