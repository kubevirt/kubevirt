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
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"

	"fmt"
	"os"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var _ = Describe("Converter", func() {

	Context("with v1.Disk", func() {
		It("Should add boot order when provided", func() {
			order := uint(1)
			kubevirtDisk := &v1.Disk{
				Name:       "mydisk",
				BootOrder:  &order,
				VolumeName: "myvolume",
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
  <alias name="mydisk"></alias>
  <boot order="1"></boot>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			fmt.Println(xml)
			Expect(xml).To(Equal(convertedDisk))
		})

		It("Should omit boot order when not provided", func() {
			kubevirtDisk := &v1.Disk{
				Name:       "mydisk",
				VolumeName: "myvolume",
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
  <alias name="mydisk"></alias>
</Disk>`
			xml := diskToDiskXML(kubevirtDisk)
			fmt.Println(xml)
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
				Hyperv: &v1.FeatureHyperv{
					Relaxed:    &v1.FeatureState{Enabled: &_false},
					VAPIC:      &v1.FeatureState{Enabled: &_true},
					Spinlocks:  &v1.FeatureSpinlocks{Enabled: &_true},
					VPIndex:    &v1.FeatureState{Enabled: &_true},
					Runtime:    &v1.FeatureState{Enabled: &_false},
					SyNIC:      &v1.FeatureState{Enabled: &_true},
					SyNICTimer: &v1.FeatureState{Enabled: &_false},
					Reset:      &v1.FeatureState{Enabled: &_true},
					VendorID:   &v1.FeatureVendorID{Enabled: &_false, VendorID: "myvendor"},
				},
			}
			vmi.Spec.Domain.Devices.Disks = []v1.Disk{
				{
					Name:       "mydisk",
					VolumeName: "myvolume",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name:       "mydisk1",
					VolumeName: "nocloud",
					DiskDevice: v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: "virtio",
						},
					},
				},
				{
					Name:       "cdrom_tray_unspecified",
					VolumeName: "volume0",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							ReadOnly: &_false,
						},
					},
				},
				{
					Name:       "cdrom_tray_open",
					VolumeName: "volume1",
					DiskDevice: v1.DiskDevice{
						CDRom: &v1.CDRomTarget{
							Tray: v1.TrayStateOpen,
						},
					},
				},
				{
					Name:       "floppy_tray_unspecified",
					VolumeName: "volume2",
					DiskDevice: v1.DiskDevice{
						Floppy: &v1.FloppyTarget{},
					},
				},
				{
					Name:       "floppy_tray_open",
					VolumeName: "volume3",
					DiskDevice: v1.DiskDevice{
						Floppy: &v1.FloppyTarget{
							Tray:     v1.TrayStateOpen,
							ReadOnly: true,
						},
					},
				},
				{
					Name:       "should_default_to_disk",
					VolumeName: "volume4",
				},
				{
					Name:       "ephemeral_pvc",
					VolumeName: "volume5",
				},
			}
			vmi.Spec.Volumes = []v1.Volume{
				{
					Name: "myvolume",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
				{
					Name: "nocloud",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserDataBase64: "1234",
						},
					},
				},
				{
					Name: "volume0",
					VolumeSource: v1.VolumeSource{
						CloudInitNoCloud: &v1.CloudInitNoCloudSource{
							UserDataBase64: "1234",
						},
					},
				},
				{
					Name: "volume1",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
				{
					Name: "volume2",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
				{
					Name: "volume3",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
				{
					Name: "volume4",
					VolumeSource: v1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: "testclaim",
						},
					},
				},
				{
					Name: "volume5",
					VolumeSource: v1.VolumeSource{
						Ephemeral: &v1.EphemeralVolumeSource{
							PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
								ClaimName: "testclaim",
							},
						},
					},
				},
			}

			vmi.Spec.Networks = []v1.Network{*v1.DefaultPodNetwork()}
			vmi.Spec.Domain.Devices.Interfaces = []v1.Interface{*v1.DefaultNetworkInterface()}

			vmi.Spec.Domain.Firmware = &v1.Firmware{
				UUID: "e4686d2c-6e8d-4335-b8fd-81bee22f4814",
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
    </system>
    <bios></bios>
    <baseBoard></baseBoard>
  </sysinfo>
  <devices>
    <interface type="bridge">
      <source bridge="br1"></source>
      <model type="virtio"></model>
      <alias name="default"></alias>
    </interface>
    <video>
      <model type="vga" heads="1" vram="16384"></model>
    </video>
    <graphics type="vnc">
      <listen type="socket" socket="/var/run/kubevirt-private/mynamespace/testvmi/virt-vnc"></listen>
    </graphics>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/myvolume/disk.img"></source>
      <target bus="virtio" dev="vda"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="mydisk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="virtio" dev="vdb"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="mydisk1"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/libvirt/cloud-init-dir/mynamespace/testvmi/noCloud.iso"></source>
      <target bus="sata" dev="sda" tray="closed"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="cdrom_tray_unspecified"></alias>
    </disk>
    <disk device="cdrom" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume1/disk.img"></source>
      <target bus="sata" dev="sdb" tray="open"></target>
      <driver name="qemu" type="raw"></driver>
      <readonly></readonly>
      <alias name="cdrom_tray_open"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume2/disk.img"></source>
      <target bus="fdc" dev="fda" tray="closed"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="floppy_tray_unspecified"></alias>
    </disk>
    <disk device="floppy" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume3/disk.img"></source>
      <target bus="fdc" dev="fdb" tray="open"></target>
      <driver name="qemu" type="raw"></driver>
      <readonly></readonly>
      <alias name="floppy_tray_open"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/kubevirt-private/vmi-disks/volume4/disk.img"></source>
      <target bus="sata" dev="sdc"></target>
      <driver name="qemu" type="raw"></driver>
      <alias name="should_default_to_disk"></alias>
    </disk>
    <disk device="disk" type="file">
      <source file="/var/run/libvirt/kubevirt-ephemeral-disk/volume5/disk.qcow2"></source>
      <target bus="sata" dev="sdd"></target>
      <driver name="qemu" type="qcow2"></driver>
      <alias name="ephemeral_pvc"></alias>
      <backingStore type="file">
        <format type="raw"></format>
        <source file="/var/run/kubevirt-private/vmi-disks/volume5/disk.img"></source>
      </backingStore>
    </disk>
    <serial type="unix">
      <target port="0"></target>
      <source mode="bind" path="/var/run/kubevirt-private/mynamespace/testvmi/virt-serial0"></source>
    </serial>
    <console type="pty">
      <target type="serial" port="0"></target>
    </console>
    <watchdog model="i6300esb" action="poweroff">
      <alias name="mywatchdog"></alias>
    </watchdog>
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
    </hyperv>
  </features>
  <cpu mode="host-model"></cpu>
</domain>`, domainType)

		var c *ConverterContext

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
			}
		})

		It("should be converted to a libvirt Domain with vmi defaults set", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			Expect(vmiToDomainXML(vmi, c)).To(Equal(convertedDomain))
		})

		It("should use kvm if present", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			Expect(vmiToDomainXMLToDomainSpec(vmi, c).Type).To(Equal(domainType))
		})

		It("should convert CPU cores and model", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.CPU = &v1.CPU{
				Cores: 3,
				Model: "Conroe",
			}
			domainSpec := vmiToDomainXMLToDomainSpec(vmi, c)

			Expect(domainSpec.CPU.Topology.Cores).To(Equal(uint32(3)))
			Expect(domainSpec.CPU.Topology.Sockets).To(Equal(uint32(1)))
			Expect(domainSpec.CPU.Topology.Threads).To(Equal(uint32(1)))
			Expect(domainSpec.CPU.Mode).To(Equal("custom"))
			Expect(domainSpec.CPU.Model).To(Equal("Conroe"))
			Expect(domainSpec.VCPU.Placement).To(Equal("static"))
			Expect(domainSpec.VCPU.CPUs).To(Equal(uint32(3)))
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

		It("should select explicitly chosen network model", func() {
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			vmi.Spec.Domain.Devices.Interfaces[0].Model = "e1000"
			domain := vmiToDomain(vmi, c)
			Expect(domain.Spec.Devices.Interfaces[0].Model.Type).To(Equal("e1000"))
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

		It("should fail to convert if non-pod interfaces are present", func() {
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
	})
	Context("Function ParseNameservers()", func() {
		It("should return a byte array of nameservers", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "nameserver 8.8.8.8\nnameserver 8.8.4.4\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should ignore non-nameserver lines and malformed nameserver lines", func() {
			ns1, ns2 := []uint8{8, 8, 8, 8}, []uint8{8, 8, 4, 4}
			resolvConf := "search example.com\nnameserver 8.8.8.8\nnameserver 8.8.4.4\nnameserver mynameserver\n"
			nameservers, err := ParseNameservers(resolvConf)
			Expect(nameservers).To(Equal([][]uint8{ns1, ns2}))
			Expect(err).To(BeNil())
		})

		It("should return a default nameserver if none is parsed", func() {
			nameservers, err := ParseNameservers("")
			expectedDNS := net.ParseIP(defaultDNS).To4()
			Expect(nameservers).To(Equal([][]uint8{expectedDNS}))
			Expect(err).To(BeNil())
		})
	})

	Context("Function ParseSearchDomains()", func() {
		It("should return a string of search domains", func() {
			resolvConf := "search cluster.local svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle multi-line search domains", func() {
			resolvConf := "search cluster.local\nsearch svc.cluster.local example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should clean up extra whitespace between search domains", func() {
			resolvConf := "search cluster.local\tsvc.cluster.local    example.com\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"cluster.local", "svc.cluster.local", "example.com"}))
			Expect(err).To(BeNil())
		})

		It("should handle non-presence of search domains by returning default search domain", func() {
			resolvConf := fmt.Sprintf("nameserver %s\n", defaultDNS)
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{defaultSearchDomain}))
			Expect(err).To(BeNil())
		})

		It("should allow partial search domains", func() {
			resolvConf := "search local\nnameserver 8.8.8.8\n"
			searchDomains, err := ParseSearchDomains(resolvConf)
			Expect(searchDomains).To(Equal([]string{"local"}))
			Expect(err).To(BeNil())
		})
	})
})

func diskToDiskXML(disk *v1.Disk) string {
	devicePerBus := make(map[string]int)
	libvirtDisk := &Disk{}
	Expect(Convert_v1_Disk_To_api_Disk(disk, libvirtDisk, devicePerBus)).To(Succeed())
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
