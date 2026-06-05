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

type DomainSnapshotDisk struct {
	Name     string            `xml:"name,attr"`
	Snapshot string            `xml:"snapshot,attr,omitempty"`
	Driver   *DomainDiskDriver `xml:"driver"`
	Source   *DomainDiskSource `xml:"source"`
}

type DomainSnapshotDisks struct {
	Disks []DomainSnapshotDisk `xml:"disk"`
}

type DomainSnapshotMemory struct {
	Snapshot string `xml:"snapshot,attr"`
	File     string `xml:"file,attr,omitempty"`
}

type DomainSnapshotParent struct {
	Name string `xml:"name"`
}

type DomainSnapshotInactiveDomain struct {
	XMLName xml.Name `xml:"inactiveDomain"`
	Domain
}

type DomainSnapshotCookie struct {
	XML string `xml:",innerxml"`
}

type DomainSnapshot struct {
	XMLName        xml.Name                      `xml:"domainsnapshot"`
	Name           string                        `xml:"name,omitempty"`
	Description    string                        `xml:"description,omitempty"`
	State          string                        `xml:"state,omitempty"`
	CreationTime   string                        `xml:"creationTime,omitempty"`
	Parent         *DomainSnapshotParent         `xml:"parent"`
	Memory         *DomainSnapshotMemory         `xml:"memory"`
	Disks          *DomainSnapshotDisks          `xml:"disks"`
	Domain         *Domain                       `xml:"domain"`
	InactiveDomain *DomainSnapshotInactiveDomain `xml:"inactiveDomain"`
	Active         *uint                         `xml:"active"`
	Cookie         *DomainSnapshotCookie         `xml:"cookie"`
}

type domainSnapshotDisk DomainSnapshotDisk

func (a *DomainSnapshotDisk) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "disk"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		}
	}
	disk := domainSnapshotDisk(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainSnapshotDisk) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "file"
	}
	a.Source = &DomainDiskSource{}
	if typ == "file" {
		a.Source.File = &DomainDiskSourceFile{}
	} else if typ == "block" {
		a.Source.Block = &DomainDiskSourceBlock{}
	} else if typ == "network" {
		a.Source.Network = &DomainDiskSourceNetwork{}
	} else if typ == "dir" {
		a.Source.Dir = &DomainDiskSourceDir{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainDiskSourceVolume{}
	}
	disk := domainSnapshotDisk(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainSnapshotDisk(disk)
	return nil
}

func (s *DomainSnapshot) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *DomainSnapshot) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
