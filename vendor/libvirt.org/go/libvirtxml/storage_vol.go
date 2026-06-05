/*
 * This file is part of the libvirt-go-xml-module project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (C) 2017 Red Hat, Inc.
 *
 */

package libvirtxml

import "encoding/xml"

type StorageVolumeSize struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value uint64 `xml:",chardata"`
}

type StorageVolumeTargetPermissions struct {
	Owner string `xml:"owner,omitempty"`
	Group string `xml:"group,omitempty"`
	Mode  string `xml:"mode,omitempty"`
	Label string `xml:"label,omitempty"`
}

type StorageVolumeTargetFeature struct {
	LazyRefcounts *struct{} `xml:"lazy_refcounts"`
	ExtendedL2    *struct{} `xml:"extended_l2"`
}

type StorageVolumeTargetFormat struct {
	Type string `xml:"type,attr"`
}

type StorageVolumeTargetTimestamps struct {
	Atime string `xml:"atime"`
	Mtime string `xml:"mtime"`
	Ctime string `xml:"ctime"`
}

type StorageVolumeTargetClusterSize struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value uint64 `xml:",chardata"`
}

type StorageVolumeTarget struct {
	Path        string                          `xml:"path,omitempty"`
	Format      *StorageVolumeTargetFormat      `xml:"format"`
	Permissions *StorageVolumeTargetPermissions `xml:"permissions"`
	Timestamps  *StorageVolumeTargetTimestamps  `xml:"timestamps"`
	Compat      string                          `xml:"compat,omitempty"`
	ClusterSize *StorageVolumeTargetClusterSize `xml:"clusterSize"`
	NoCOW       *struct{}                       `xml:"nocow"`
	Features    []StorageVolumeTargetFeature    `xml:"features"`
	Encryption  *StorageEncryption              `xml:"encryption"`
}

type StorageVolumeBackingStore struct {
	Path        string                          `xml:"path"`
	Format      *StorageVolumeTargetFormat      `xml:"format"`
	Permissions *StorageVolumeTargetPermissions `xml:"permissions"`
}

type StorageVolume struct {
	XMLName      xml.Name                   `xml:"volume"`
	Type         string                     `xml:"type,attr,omitempty"`
	Name         string                     `xml:"name"`
	Key          string                     `xml:"key,omitempty"`
	Allocation   *StorageVolumeSize         `xml:"allocation"`
	Capacity     *StorageVolumeSize         `xml:"capacity"`
	Physical     *StorageVolumeSize         `xml:"physical"`
	Target       *StorageVolumeTarget       `xml:"target"`
	BackingStore *StorageVolumeBackingStore `xml:"backingStore"`
}

func (s *StorageVolume) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *StorageVolume) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
