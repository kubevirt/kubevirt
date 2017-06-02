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

package inspector

import (
	"fmt"

	"github.com/libvirt/libvirt-go-xml"

	apiv1 "kubevirt.io/kubevirt/pkg/api/v1"
)

func applyOSConfig(src *libvirtxml.Domain, dst *apiv1.DomainSpec) error {
	if dst.OS.Type.Arch == "" {
		dst.OS.Type.Arch = src.OS.Type.Arch
	}
	if dst.OS.Type.Machine == "" {
		dst.OS.Type.Machine = src.OS.Type.Machine
	}

	return nil
}

func createAddress(src *libvirtxml.DomainAddress) *apiv1.Address {
	addr := &apiv1.Address{
		Type: src.Type,
	}
	if src.Domain != nil {
		addr.Domain = fmt.Sprintf("%d", *src.Domain)
	}
	if src.Bus != nil {
		addr.Bus = fmt.Sprintf("%d", *src.Bus)
	}
	if src.Slot != nil {
		addr.Slot = fmt.Sprintf("%d", *src.Slot)
	}
	if src.Function != nil {
		addr.Function = fmt.Sprintf("%d", *src.Function)
	}

	return addr
}

func applyDiskConfig(src *libvirtxml.DomainDisk, dst *apiv1.Disk) error {
	if dst.Device == "" {
		dst.Device = src.Device
	}

	if dst.Target.Bus == "" {
		dst.Target.Bus = src.Target.Bus
	}

	if src.Driver != nil {
		if dst.Driver == nil {
			dst.Driver = &apiv1.DiskDriver{}
		}
		if dst.Driver.Name == "" {
			dst.Driver.Name = src.Driver.Name
		}
		if dst.Driver.Type == "" {
			dst.Driver.Type = src.Driver.Type
		}
	}

	/*
		if dst.Address == nil && src.Address != nil {
			dst.Address = createAddress(src.Address)
		}
	*/

	return nil
}

func applyInterfaceConfig(src *libvirtxml.DomainInterface, dst *apiv1.Interface) error {
	if dst.MAC.MAC == "" {
		dst.MAC.MAC = src.MAC.Address
	}

	if dst.Address == nil && src.Address != nil {
		dst.Address = createAddress(src.Address)
	}

	return nil
}

func applyVideoConfig(src *libvirtxml.DomainVideo, dst *apiv1.Video) error {
	if dst.Heads == nil && src.Model.Heads != 0 {
		heads := src.Model.Heads
		dst.Heads = &heads
	}

	/*
		if dst.Address == nil && src.Address != nil {
			dst.Address = createAddress(src.Address)
		}
	*/

	return nil
}

func applyBalloonConfig(src *libvirtxml.DomainMemBalloon, dst *apiv1.Ballooning) error {

	/*
		if dst.Address == nil && src.Address != nil {
			dst.Address = createAddress(src.Address)
		}
	*/

	return nil
}

func applyDeviceConfig(src *libvirtxml.DomainDeviceList, dst *apiv1.Devices) error {

	if len(src.Disks) != len(dst.Disks) {
		return fmt.Errorf("Expected %d disks but got %d",
			len(dst.Disks), len(src.Disks))
	}
	for n := 0; n < len(src.Disks); n++ {
		if err := applyDiskConfig(&src.Disks[n], &dst.Disks[n]); err != nil {
			return err
		}
	}

	if len(src.Interfaces) != len(dst.Interfaces) {
		return fmt.Errorf("Expected %d interfaces but got %d",
			len(dst.Interfaces), len(src.Interfaces))
	}
	for n := 0; n < len(src.Interfaces); n++ {
		if err := applyInterfaceConfig(&src.Interfaces[n], &dst.Interfaces[n]); err != nil {
			return err
		}
	}

	/* Libvirt auto-adds a video device if <graphics> is present and no
	 * <video> is defined
	 */
	if len(dst.Video) == 0 && len(src.Videos) == 1 {
		vid := apiv1.Video{
			Type: src.Videos[0].Model.Type,
		}
		dst.Video = append(dst.Video, vid)
	}

	if len(src.Videos) != len(dst.Video) {
		return fmt.Errorf("Expected %d videos but got %d",
			len(dst.Video), len(src.Videos))
	}
	for n := 0; n < len(src.Videos); n++ {
		if err := applyVideoConfig(&src.Videos[n], &dst.Video[n]); err != nil {
			return err
		}
	}

	/* Libvirt (sometimes) auto-adds a memory balloon device if none
	 *  is present
	 */
	if dst.Ballooning == nil && src.MemBalloon != nil {
		bal := apiv1.Ballooning{
			Model: src.MemBalloon.Model,
		}
		dst.Ballooning = &bal
	}
	if dst.Ballooning != nil {
		if src.MemBalloon == nil {
			return fmt.Errorf("Balloon was missing in config")
		}
		if err := applyBalloonConfig(src.MemBalloon, dst.Ballooning); err != nil {
			return err
		}
	}

	return nil
}

func ApplyConfig(src *libvirtxml.Domain, dst *apiv1.DomainSpec) error {

	if err := applyOSConfig(src, dst); err != nil {
		return err
	}

	if err := applyDeviceConfig(src.Devices, &dst.Devices); err != nil {
		return err
	}

	return nil
}
