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

package designer

import (
	"fmt"
	"net"
	"strconv"

	"github.com/libvirt/libvirt-go-xml"

	kubev1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	apiv1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/logging"
)

type DomainDesigner struct {
	Client    cache.Getter
	Namespace string
	Domain    *libvirtxml.Domain
}

func NewDomainDesigner(client cache.Getter, namespace string) *DomainDesigner {
	return &DomainDesigner{
		Client:    client,
		Namespace: namespace,
		Domain:    &libvirtxml.Domain{},
	}
}

func (d *DomainDesigner) ApplyResourcePartition(partition string) {
	d.Domain.Resource = &libvirtxml.DomainResource{
		Partition: partition,
	}
}

func (d *DomainDesigner) ApplySpec(src *apiv1.VM) error {
	d.Domain.UUID = string(src.GetObjectMeta().GetUID())
	d.Domain.Name = src.GetObjectMeta().GetName()
	d.Domain.Type = src.Spec.Domain.Type

	if err := d.applyOSConfig(src.Spec.Domain); err != nil {
		return err
	}

	if err := d.applySysInfo(src.Spec.Domain.SysInfo); err != nil {
		return err
	}

	if err := d.applyMemoryConfig(src.Spec.Domain); err != nil {
		return err
	}

	if err := d.applyDeviceConfig(&src.Spec.Domain.Devices); err != nil {
		return err
	}

	return nil
}

func (d *DomainDesigner) applyOSConfig(src *apiv1.DomainSpec) error {
	d.Domain.OS = &libvirtxml.DomainOS{
		Type: &libvirtxml.DomainOSType{
			Type:    src.OS.Type.OS,
			Arch:    src.OS.Type.Arch,
			Machine: src.OS.Type.Machine,
		},
	}

	if src.OS.SMBios != nil {
		d.Domain.OS.SMBios = &libvirtxml.DomainSMBios{
			Mode: src.OS.SMBios.Mode,
		}
	}

	if src.OS.BootMenu != nil {
		d.Domain.OS.BootMenu = &libvirtxml.DomainBootMenu{}
		if src.OS.BootMenu.Enabled {
			d.Domain.OS.BootMenu.Enabled = "yes"
		}
		if src.OS.BootMenu.Timeout != nil {
			d.Domain.OS.BootMenu.Timeout = fmt.Sprintf("%d", *src.OS.BootMenu.Timeout)
		}
	}

	return nil
}

func copySysInfoEntries(src []apiv1.Entry) []libvirtxml.DomainSysInfoEntry {
	dst := make([]libvirtxml.DomainSysInfoEntry, len(src))

	for i := 0; i < len(src); i++ {
		dst[i] = libvirtxml.DomainSysInfoEntry{
			Name:  src[i].Name,
			Value: src[i].Value,
		}
	}

	return dst
}

func (d *DomainDesigner) applySysInfo(src *apiv1.SysInfo) error {
	if src == nil {
		return nil
	}

	d.Domain.SysInfo = &libvirtxml.DomainSysInfo{
		Type: src.Type,
	}

	d.Domain.SysInfo.System = copySysInfoEntries(src.System)
	d.Domain.SysInfo.BIOS = copySysInfoEntries(src.BIOS)
	d.Domain.SysInfo.BaseBoard = copySysInfoEntries(src.BaseBoard)

	return nil
}

func (d *DomainDesigner) applyMemoryConfig(src *apiv1.DomainSpec) error {
	d.Domain.Memory = &libvirtxml.DomainMemory{
		Value: src.Memory.Value,
		Unit:  "MiB",
	}

	return nil
}

func (d *DomainDesigner) applyDeviceConfig(src *apiv1.Devices) error {
	d.Domain.Devices = &libvirtxml.DomainDeviceList{}

	for _, iface := range src.Interfaces {
		if err := d.applyInterfaceConfig(&iface); err != nil {
			return err
		}
	}

	for _, channel := range src.Channels {
		if err := d.applyChannelConfig(&channel); err != nil {
			return err
		}
	}

	for _, video := range src.Video {
		if err := d.applyVideoConfig(&video); err != nil {
			return err
		}
	}

	for _, graphics := range src.Graphics {
		if err := d.applyGraphicsConfig(&graphics); err != nil {
			return err
		}
	}

	for _, disk := range src.Disks {
		if err := d.applyDiskConfig(&disk); err != nil {
			return err
		}
	}

	for _, serial := range src.Serials {
		if err := d.applySerialConfig(&serial); err != nil {
			return err
		}
	}

	for _, console := range src.Consoles {
		if err := d.applyConsoleConfig(&console); err != nil {
			return err
		}
	}

	if src.Ballooning != nil {
		if err := d.applyBalloonConfig(src.Ballooning); err != nil {
			return err
		}
	}

	return nil
}

func createDeviceAddress(src *apiv1.Address) (*libvirtxml.DomainAddress, error) {
	domain64, err := strconv.ParseInt(src.Domain, 10, 0)
	if err != nil {
		return nil, err
	}
	bus64, err := strconv.ParseInt(src.Bus, 10, 0)
	if err != nil {
		return nil, err
	}
	slot64, err := strconv.ParseInt(src.Slot, 10, 0)
	if err != nil {
		return nil, err
	}
	function64, err := strconv.ParseInt(src.Function, 10, 0)
	if err != nil {
		return nil, err
	}
	domain := uint(domain64)
	bus := uint(bus64)
	slot := uint(slot64)
	function := uint(function64)
	return &libvirtxml.DomainAddress{
		Type:     src.Type,
		Domain:   &domain,
		Bus:      &bus,
		Slot:     &slot,
		Function: &function,
	}, nil
}

func (d *DomainDesigner) applyInterfaceConfig(src *apiv1.Interface) error {
	// XXX make source configurable...
	dst := libvirtxml.DomainInterface{
		Type: "network",
		Source: &libvirtxml.DomainInterfaceSource{
			Network: "default",
		},
	}

	if src.MAC != nil {
		dst.MAC = &libvirtxml.DomainInterfaceMAC{
			Address: src.MAC.MAC,
		}
	}

	if src.Model != nil {
		dst.Model = &libvirtxml.DomainInterfaceModel{
			Type: src.Model.Type,
		}
	}

	if src.BootOrder != nil {
		dst.Boot = &libvirtxml.DomainDeviceBoot{
			Order: src.BootOrder.Order,
		}
	}

	if src.LinkState != nil {
		dst.Link = &libvirtxml.DomainInterfaceLink{
			State: src.LinkState.State,
		}
	}

	if src.Address != nil {
		addr, err := createDeviceAddress(src.Address)
		if err != nil {
			return err
		}
		dst.Address = addr
	}

	d.Domain.Devices.Interfaces = append(d.Domain.Devices.Interfaces, dst)

	return nil
}

func (d *DomainDesigner) applyChannelConfig(src *apiv1.Channel) error {
	dst := libvirtxml.DomainChannel{}
	if src.Target.Name == "com.redhat.spice.0" {
		dst.Type = "spicevmc"
	} else {
		dst.Type = "unix"
	}

	dst.Target = &libvirtxml.DomainChannelTarget{
		Type: src.Target.Type,
		Name: src.Target.Name,
	}

	d.Domain.Devices.Channels = append(d.Domain.Devices.Channels, dst)

	return nil
}

func (d *DomainDesigner) applyVideoConfig(src *apiv1.Video) error {
	dst := libvirtxml.DomainVideo{
		Model: libvirtxml.DomainVideoModel{
			Type: src.Type,
		},
	}

	if src.Heads != nil {
		dst.Model.Heads = *src.Heads
	}
	if src.Ram != nil {
		dst.Model.Ram = *src.Ram
	}
	if src.VRam != nil {
		dst.Model.VRam = *src.VRam
	}
	if src.VGAMem != nil {
		dst.Model.VGAMem = *src.VGAMem
	}

	d.Domain.Devices.Videos = append(d.Domain.Devices.Videos, dst)

	return nil
}

func (d *DomainDesigner) applyGraphicsConfig(src *apiv1.Graphics) error {
	if src.Type != "spice" && src.Type != "vnc" {
		return fmt.Errorf("Unsupported graphics type '%s', required 'spice' or 'vnc'",
			src.Type)
	}

	dst := libvirtxml.DomainGraphic{
		Type:     src.Type,
		AutoPort: "yes",
		Listen:   "0.0.0.0",
	}

	d.Domain.Devices.Graphics = append(d.Domain.Devices.Graphics, dst)

	return nil
}

func (d *DomainDesigner) buildDiskConfigPVCISCSI(src *kubev1.ISCSIVolumeSource) (*libvirtxml.DomainDisk, error) {
	logging.DefaultLogger().Info().Msg("Mapping iSCSI PVC")

	host, port, err := net.SplitHostPort(src.TargetPortal)
	if err != nil {
		return nil, err
	}

	return &libvirtxml.DomainDisk{
		Type: "network",
		Source: &libvirtxml.DomainDiskSource{
			Protocol: "iscsi",
			Name:     fmt.Sprintf("%s/%d", src.IQN, src.Lun),
			Hosts: []libvirtxml.DomainDiskSourceHost{
				libvirtxml.DomainDiskSourceHost{
					Transport: "tcp",
					Name:      host,
					Port:      port,
				},
			},
		},
	}, nil
}

func (d *DomainDesigner) buildDiskConfigPVC(src *apiv1.DiskSourcePersistentVolumeClaim) (*libvirtxml.DomainDisk, error) {
	logging.DefaultLogger().V(3).Info().Msgf("Mapping PersistentVolumeClaim: %s", src.ClaimName)

	// Look up existing persistent volume
	obj, err := d.Client.Get().Namespace(d.Namespace).Resource("persistentvolumeclaims").Name(src.ClaimName).Do().Get()
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msgf("unable to look up persistent volume claim %s", src.ClaimName)
		return nil, fmt.Errorf("unable to look up persistent volume claim %s: %v", src.ClaimName, err)
	}

	pvc := obj.(*kubev1.PersistentVolumeClaim)
	if pvc.Status.Phase != kubev1.ClaimBound {
		logging.DefaultLogger().Error().Msgf("attempted use of unbound persistent volume %s", pvc.Name)
		return nil, fmt.Errorf("attempted use of unbound persistent volume claim: %s", pvc.Name)
	}

	// Look up the PersistentVolume this PVC is bound to
	// Note: This call is not namespaced!
	obj, err = d.Client.Get().Resource("persistentvolumes").Name(pvc.Spec.VolumeName).Do().Get()
	if err != nil {
		logging.DefaultLogger().Error().Reason(err).Msgf("unable to access persistent volume record %s", pvc.Spec.VolumeName)
		return nil, fmt.Errorf("unable to access persistent volume record %s: %v", pvc.Spec.VolumeName, err)
	}

	pv := obj.(*kubev1.PersistentVolume)

	if pv.Spec.ISCSI != nil {
		return d.buildDiskConfigPVCISCSI(pv.Spec.ISCSI)
	} else {
		logging.DefaultLogger().Error().Msg(fmt.Sprintf("Referenced PV %v is backed by an unsupported storage type", pvc.Spec.VolumeName))
		return nil, fmt.Errorf("Referenced PV %v is backed by an unsupported storage type", pvc.Spec.VolumeName)
	}
}

func (d *DomainDesigner) buildDiskConfigISCSI(src *apiv1.DiskSourceISCSI) (*libvirtxml.DomainDisk, error) {
	logging.DefaultLogger().Info().Msg("Mapping iSCSI")

	host, port, err := net.SplitHostPort(src.TargetPortal)
	if err != nil {
		return nil, err
	}

	return &libvirtxml.DomainDisk{
		Type: "network",
		Source: &libvirtxml.DomainDiskSource{
			Protocol: "iscsi",
			Name:     fmt.Sprintf("%s/%d", src.IQN, src.Lun),
			Hosts: []libvirtxml.DomainDiskSourceHost{
				libvirtxml.DomainDiskSourceHost{
					Transport: "tcp",
					Name:      host,
					Port:      port,
				},
			},
		},
	}, nil
}

func (d *DomainDesigner) buildDiskConfig(src *apiv1.Disk) (*libvirtxml.DomainDisk, error) {
	if src.Source.PersistentVolumeClaim != nil {
		return d.buildDiskConfigPVC(src.Source.PersistentVolumeClaim)
	} else if src.Source.ISCSI != nil {
		return d.buildDiskConfigISCSI(src.Source.ISCSI)
	} else {
		return nil, fmt.Errorf("No disk source provided")
	}
}

func (d *DomainDesigner) applyDiskConfig(src *apiv1.Disk) error {
	dst, err := d.buildDiskConfig(src)
	if err != nil {
		return err
	}

	dst.Device = src.Device
	dst.Snapshot = src.Snapshot
	dst.Serial = src.Serial

	if src.Driver != nil {
		dst.Driver = &libvirtxml.DomainDiskDriver{
			Name:        src.Driver.Name,
			Type:        src.Driver.Type,
			IO:          src.Driver.IO,
			Cache:       src.Driver.Cache,
			ErrorPolicy: src.Driver.ErrorPolicy,
		}
	}

	if src.ReadOnly != nil {
		dst.ReadOnly = &libvirtxml.DomainDiskReadOnly{}
	}

	dst.Target = &libvirtxml.DomainDiskTarget{
		Dev: src.Target.Device,
		Bus: src.Target.Bus,
	}

	d.Domain.Devices.Disks = append(d.Domain.Devices.Disks, *dst)

	return nil
}

func (d *DomainDesigner) applySerialConfig(src *apiv1.Serial) error {
	dst := libvirtxml.DomainSerial{
		Type: "pty",
	}

	if src.Target != nil {
		dst.Target = &libvirtxml.DomainSerialTarget{}

		if src.Target.Port != nil {
			port := *src.Target.Port
			dst.Target.Port = &port
		}
	}

	d.Domain.Devices.Serials = append(d.Domain.Devices.Serials, dst)

	return nil
}

func (d *DomainDesigner) applyConsoleConfig(src *apiv1.Console) error {
	dst := libvirtxml.DomainConsole{
		Type: "pty",
	}

	if src.Target != nil {
		dst.Target = &libvirtxml.DomainConsoleTarget{}

		if src.Target.Type != nil {
			dst.Target.Type = *src.Target.Type
		}
		if src.Target.Port != nil {
			port := *src.Target.Port
			dst.Target.Port = &port
		}
	}

	d.Domain.Devices.Consoles = append(d.Domain.Devices.Consoles, dst)

	return nil
}

func (d *DomainDesigner) applyBalloonConfig(src *apiv1.Ballooning) error {

	dst := &libvirtxml.DomainMemBalloon{
		Model: src.Model,
	}

	d.Domain.Devices.MemBalloon = dst

	return nil
}
