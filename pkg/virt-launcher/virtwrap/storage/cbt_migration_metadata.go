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

package storage

import (
	"encoding/xml"
	"fmt"
	"math"

	"libvirt.org/go/libvirtxml"
)

// migrationDiskSizesMetadata is carried in DestXML <metadata> under the
// kubevirt namespace so the target premigration hook can size CBT overlays to
// match the source virtual size. Libvirt accepts a single namespaced root
// element under <metadata>; DestXML does not need source-only kubevirt fields.
type migrationDiskSizesMetadata struct {
	XMLName xml.Name                `xml:"http://kubevirt.io kubevirt"`
	Sizes   *migrationDiskSizesList `xml:"migrationDiskSizes,omitempty"`
}

type migrationDiskSizesList struct {
	Disks []migrationDiskSizeEntry `xml:"disk"`
}

type migrationDiskSizeEntry struct {
	Name string `xml:"name,attr"`
	Size uint64 `xml:"size,attr"`
}

// SetMigrationDiskSizes writes per-volume source virtual sizes into domain
// metadata for the migration DestXML path.
func SetMigrationDiskSizes(dom *libvirtxml.Domain, sizes map[string]int64) error {
	if dom == nil || len(sizes) == 0 {
		return nil
	}

	meta := &migrationDiskSizesMetadata{
		Sizes: &migrationDiskSizesList{},
	}
	for name, size := range sizes {
		if name == "" || size <= 0 {
			continue
		}
		meta.Sizes.Disks = append(meta.Sizes.Disks, migrationDiskSizeEntry{
			Name: name,
			Size: uint64(size),
		})
	}
	if len(meta.Sizes.Disks) == 0 {
		return nil
	}

	raw, err := xml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal CBT migration metadata: %w", err)
	}
	dom.Metadata = &libvirtxml.DomainMetadata{XML: string(raw)}
	return nil
}

// MigrationDiskSize returns the source virtual size for volumeName from DestXML
// metadata, if present.
func MigrationDiskSize(dom *libvirtxml.Domain, volumeName string) (int64, bool) {
	meta, ok := parseMigrationDiskSizesMetadata(dom)
	if !ok || meta.Sizes == nil {
		return 0, false
	}
	for _, disk := range meta.Sizes.Disks {
		if disk.Name != volumeName || disk.Size == 0 || disk.Size > math.MaxInt64 {
			continue
		}
		return int64(disk.Size), true
	}
	return 0, false
}

// ClearMigrationDiskSizes removes migration disk size metadata from the domain.
func ClearMigrationDiskSizes(dom *libvirtxml.Domain) {
	if dom == nil || dom.Metadata == nil {
		return
	}
	meta, ok := parseMigrationDiskSizesMetadata(dom)
	if !ok || meta.Sizes == nil {
		return
	}
	// DestXML metadata we set contains only migrationDiskSizes; drop it entirely.
	dom.Metadata = nil
}

func parseMigrationDiskSizesMetadata(dom *libvirtxml.Domain) (*migrationDiskSizesMetadata, bool) {
	if dom == nil || dom.Metadata == nil || dom.Metadata.XML == "" {
		return nil, false
	}
	var meta migrationDiskSizesMetadata
	// xml.Unmarshal enforces the http://kubevirt.io namespace via the struct tag.
	if err := xml.Unmarshal([]byte(dom.Metadata.XML), &meta); err != nil {
		return nil, false
	}
	return &meta, true
}
