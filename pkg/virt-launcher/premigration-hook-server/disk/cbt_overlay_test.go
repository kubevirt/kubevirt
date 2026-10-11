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
 */

package disk_test

import (
	"errors"

	"libvirt.org/go/libvirtxml"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/virt-launcher/premigration-hook-server/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/storage"
)

var _ = Describe("CBTOverlayHook", func() {
	const (
		overlayPath = "/var/lib/libvirt/qemu/cbt/disk0.qcow2"
		fsDataPath  = "/var/run/kubevirt-private/vmi-disks/disk0/disk.img"
		blkDataPath = "/dev/disk0"
	)

	type overlayCall struct{ overlay, data string }

	var (
		calls          []overlayCall
		createErr      error
		originalCreate = storage.CreateQCOW2Overlay
	)

	BeforeEach(func() {
		calls = nil
		createErr = nil
		storage.CreateQCOW2Overlay = func(overlay, data string) error {
			calls = append(calls, overlayCall{overlay, data})
			return createErr
		}
		DeferCleanup(func() { storage.CreateQCOW2Overlay = originalCreate })
	})

	cbtDisk := func(dataStore libvirtxml.DomainDiskSource) libvirtxml.DomainDisk {
		return libvirtxml.DomainDisk{
			Alias: &libvirtxml.DomainAlias{Name: "ua-disk0"},
			Source: &libvirtxml.DomainDiskSource{
				File:      &libvirtxml.DomainDiskSourceFile{File: overlayPath},
				DataStore: &libvirtxml.DomainDiskDataStore{Source: &dataStore},
			},
		}
	}

	domainWith := func(disks ...libvirtxml.DomainDisk) *libvirtxml.Domain {
		return &libvirtxml.Domain{Devices: &libvirtxml.DomainDeviceList{Disks: disks}}
	}

	DescribeTable("should create the overlay the incoming domain references", func(dataStore libvirtxml.DomainDiskSource, dataPath string) {
		domain := domainWith(cbtDisk(dataStore))

		Expect(disk.CBTOverlayHook(nil, libvmi.New(), domain)).To(Succeed())
		Expect(calls).To(ConsistOf(overlayCall{overlayPath, dataPath}))
		Expect(domain.Devices.Disks[0].Source.File.File).To(Equal(overlayPath))
	},
		Entry("with a filesystem data store", libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: fsDataPath}}, fsDataPath),
		Entry("with a block data store", libvirtxml.DomainDiskSource{Block: &libvirtxml.DomainDiskSourceBlock{Dev: blkDataPath}}, blkDataPath),
	)

	It("should ignore disks without a data store", func() {
		domain := domainWith(libvirtxml.DomainDisk{
			Alias:  &libvirtxml.DomainAlias{Name: "ua-disk0"},
			Source: &libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: fsDataPath}},
		})

		Expect(disk.CBTOverlayHook(nil, libvmi.New(), domain)).To(Succeed())
		Expect(calls).To(BeEmpty())
	})

	It("should fail when the overlay cannot be created", func() {
		createErr = errors.New("boom")
		domain := domainWith(cbtDisk(libvirtxml.DomainDiskSource{File: &libvirtxml.DomainDiskSourceFile{File: fsDataPath}}))

		Expect(disk.CBTOverlayHook(nil, libvmi.New(), domain)).To(MatchError(ContainSubstring("boom")))
	})
})
