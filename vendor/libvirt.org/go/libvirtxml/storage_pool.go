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

type StoragePoolSize struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value uint64 `xml:",chardata"`
}

type StoragePoolTargetPermissions struct {
	Owner string `xml:"owner,omitempty"`
	Group string `xml:"group,omitempty"`
	Mode  string `xml:"mode,omitempty"`
	Label string `xml:"label,omitempty"`
}

type StoragePoolTargetTimestamps struct {
	Atime string `xml:"atime"`
	Mtime string `xml:"mtime"`
	Ctime string `xml:"ctime"`
}

type StoragePoolTarget struct {
	Path        string                        `xml:"path,omitempty"`
	Permissions *StoragePoolTargetPermissions `xml:"permissions"`
	Timestamps  *StoragePoolTargetTimestamps  `xml:"timestamps"`
	Encryption  *StorageEncryption            `xml:"encryption"`
}

type StoragePoolSourceFormat struct {
	Type string `xml:"type,attr"`
}

type StoragePoolSourceProtocol struct {
	Version string `xml:"ver,attr"`
}

type StoragePoolSourceHost struct {
	Name string `xml:"name,attr"`
	Port string `xml:"port,attr,omitempty"`
}

type StoragePoolSourceDevice struct {
	Path          string                              `xml:"path,attr"`
	PartSeparator string                              `xml:"part_separator,attr,omitempty"`
	FreeExtents   []StoragePoolSourceDeviceFreeExtent `xml:"freeExtent"`
}

type StoragePoolSourceDeviceFreeExtent struct {
	Start uint64 `xml:"start,attr"`
	End   uint64 `xml:"end,attr"`
}

type StoragePoolSourceAuthSecret struct {
	Usage string `xml:"usage,attr,omitempty"`
	UUID  string `xml:"uuid,attr,omitempty"`
}

type StoragePoolSourceAuth struct {
	Type     string                       `xml:"type,attr"`
	Username string                       `xml:"username,attr"`
	Secret   *StoragePoolSourceAuthSecret `xml:"secret"`
}

type StoragePoolSourceVendor struct {
	Name string `xml:"name,attr"`
}

type StoragePoolSourceProduct struct {
	Name string `xml:"name,attr"`
}

type StoragePoolPCIAddress struct {
	Domain   *uint `xml:"domain,attr"`
	Bus      *uint `xml:"bus,attr"`
	Slot     *uint `xml:"slot,attr"`
	Function *uint `xml:"function,attr"`
}

type StoragePoolSourceAdapterParentAddr struct {
	UniqueID uint64                 `xml:"unique_id,attr"`
	Address  *StoragePoolPCIAddress `xml:"address"`
}

type StoragePoolSourceAdapter struct {
	Type       string                              `xml:"type,attr,omitempty"`
	Name       string                              `xml:"name,attr,omitempty"`
	Parent     string                              `xml:"parent,attr,omitempty"`
	Managed    string                              `xml:"managed,attr,omitempty"`
	WWNN       string                              `xml:"wwnn,attr,omitempty"`
	WWPN       string                              `xml:"wwpn,attr,omitempty"`
	ParentAddr *StoragePoolSourceAdapterParentAddr `xml:"parentaddr"`
}

type StoragePoolSourceDir struct {
	Path string `xml:"path,attr"`
}

type StoragePoolSourceInitiator struct {
	IQN StoragePoolSourceInitiatorIQN `xml:"iqn"`
}

type StoragePoolSourceInitiatorIQN struct {
	Name string `xml:"name,attr,omitempty"`
}

type StoragePoolSource struct {
	Name      string                      `xml:"name,omitempty"`
	Dir       *StoragePoolSourceDir       `xml:"dir"`
	Host      []StoragePoolSourceHost     `xml:"host"`
	Device    []StoragePoolSourceDevice   `xml:"device"`
	Auth      *StoragePoolSourceAuth      `xml:"auth"`
	Vendor    *StoragePoolSourceVendor    `xml:"vendor"`
	Product   *StoragePoolSourceProduct   `xml:"product"`
	Format    *StoragePoolSourceFormat    `xml:"format"`
	Protocol  *StoragePoolSourceProtocol  `xml:"protocol"`
	Adapter   *StoragePoolSourceAdapter   `xml:"adapter"`
	Initiator *StoragePoolSourceInitiator `xml:"initiator"`
}

type StoragePoolRefreshVol struct {
	Allocation string `xml:"allocation,attr"`
}

type StoragePoolRefresh struct {
	Volume StoragePoolRefreshVol `xml:"volume"`
}

type StoragePoolFeatures struct {
	COW StoragePoolFeatureCOW `xml:"cow"`
}

type StoragePoolFeatureCOW struct {
	State string `xml:"state,attr"`
}

type StoragePool struct {
	XMLName    xml.Name             `xml:"pool"`
	Type       string               `xml:"type,attr"`
	Name       string               `xml:"name,omitempty"`
	UUID       string               `xml:"uuid,omitempty"`
	Allocation *StoragePoolSize     `xml:"allocation"`
	Capacity   *StoragePoolSize     `xml:"capacity"`
	Available  *StoragePoolSize     `xml:"available"`
	Features   *StoragePoolFeatures `xml:"features"`
	Target     *StoragePoolTarget   `xml:"target"`
	Source     *StoragePoolSource   `xml:"source"`
	Refresh    *StoragePoolRefresh  `xml:"refresh"`

	/* Pool backend namespcaes must be last */
	FSCommandline  *StoragePoolFSCommandline
	RBDCommandline *StoragePoolRBDCommandline
}

type StoragePoolFSCommandlineOption struct {
	Name string `xml:"name,attr"`
}

type StoragePoolFSCommandline struct {
	XMLName xml.Name                         `xml:"http://libvirt.org/schemas/storagepool/fs/1.0 mount_opts"`
	Options []StoragePoolFSCommandlineOption `xml:"option"`
}

type StoragePoolRBDCommandlineOption struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type StoragePoolRBDCommandline struct {
	XMLName xml.Name                          `xml:"http://libvirt.org/schemas/storagepool/rbd/1.0 config_opts"`
	Options []StoragePoolRBDCommandlineOption `xml:"option"`
}

func (a *StoragePoolPCIAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "domain", a.Domain, "0x%04x")
	marshalUintAttr(&start, "bus", a.Bus, "0x%02x")
	marshalUintAttr(&start, "slot", a.Slot, "0x%02x")
	marshalUintAttr(&start, "function", a.Function, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *StoragePoolPCIAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "domain" {
			if err := unmarshalUintAttr(attr.Value, &a.Domain, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "bus" {
			if err := unmarshalUintAttr(attr.Value, &a.Bus, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "slot" {
			if err := unmarshalUintAttr(attr.Value, &a.Slot, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "function" {
			if err := unmarshalUintAttr(attr.Value, &a.Function, 0); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (s *StoragePool) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *StoragePool) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
