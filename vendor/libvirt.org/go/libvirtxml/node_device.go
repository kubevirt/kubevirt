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

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type NodeDevice struct {
	XMLName    xml.Name             `xml:"device"`
	Name       string               `xml:"name"`
	Path       string               `xml:"path,omitempty"`
	DevNodes   []NodeDeviceDevNode  `xml:"devnode"`
	Parent     string               `xml:"parent,omitempty"`
	Driver     *NodeDeviceDriver    `xml:"driver"`
	Capability NodeDeviceCapability `xml:"capability"`
}

type NodeDeviceDevNode struct {
	Type string `xml:"type,attr,omitempty"`
	Path string `xml:",chardata"`
}

type NodeDeviceDriver struct {
	Name string `xml:"name"`
}

type NodeDeviceCapability struct {
	System     *NodeDeviceSystemCapability
	PCI        *NodeDevicePCICapability
	USB        *NodeDeviceUSBCapability
	USBDevice  *NodeDeviceUSBDeviceCapability
	Net        *NodeDeviceNetCapability
	SCSIHost   *NodeDeviceSCSIHostCapability
	SCSITarget *NodeDeviceSCSITargetCapability
	SCSI       *NodeDeviceSCSICapability
	Storage    *NodeDeviceStorageCapability
	DRM        *NodeDeviceDRMCapability
	CCW        *NodeDeviceCCWCapability
	MDev       *NodeDeviceMDevCapability
	CSS        *NodeDeviceCSSCapability
	APQueue    *NodeDeviceAPQueueCapability
	APCard     *NodeDeviceAPCardCapability
	APMatrix   *NodeDeviceAPMatrixCapability
}

type NodeDeviceIDName struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type NodeDevicePCIExpress struct {
	Links []NodeDevicePCIExpressLink `xml:"link"`
}

type NodeDevicePCIExpressLink struct {
	Validity string  `xml:"validity,attr,omitempty"`
	Speed    float64 `xml:"speed,attr,omitempty"`
	Port     *uint   `xml:"port,attr"`
	Width    *uint   `xml:"width,attr"`
}

type NodeDeviceIOMMUGroup struct {
	Number  int                    `xml:"number,attr"`
	Address []NodeDevicePCIAddress `xml:"address"`
}

type NodeDeviceNUMA struct {
	Node int `xml:"node,attr"`
}

type NodeDevicePCICapability struct {
	Class        string                       `xml:"class,omitempty"`
	Domain       *uint                        `xml:"domain"`
	Bus          *uint                        `xml:"bus"`
	Slot         *uint                        `xml:"slot"`
	Function     *uint                        `xml:"function"`
	Product      NodeDeviceIDName             `xml:"product,omitempty"`
	Vendor       NodeDeviceIDName             `xml:"vendor,omitempty"`
	IOMMUGroup   *NodeDeviceIOMMUGroup        `xml:"iommuGroup"`
	NUMA         *NodeDeviceNUMA              `xml:"numa"`
	PCIExpress   *NodeDevicePCIExpress        `xml:"pci-express"`
	Capabilities []NodeDevicePCISubCapability `xml:"capability"`
}

type NodeDevicePCIAddress struct {
	Domain   *uint `xml:"domain,attr"`
	Bus      *uint `xml:"bus,attr"`
	Slot     *uint `xml:"slot,attr"`
	Function *uint `xml:"function,attr"`
}

type NodeDevicePCISubCapability struct {
	VirtFunctions *NodeDevicePCIVirtFunctionsCapability
	PhysFunction  *NodeDevicePCIPhysFunctionCapability
	MDevTypes     *NodeDevicePCIMDevTypesCapability
	Bridge        *NodeDevicePCIBridgeCapability
	VPD           *NodeDevicePCIVPDCapability
}

type NodeDevicePCIVirtFunctionsCapability struct {
	Address  []NodeDevicePCIAddress `xml:"address,omitempty"`
	MaxCount int                    `xml:"maxCount,attr,omitempty"`
}

type NodeDevicePCIPhysFunctionCapability struct {
	Address NodeDevicePCIAddress `xml:"address,omitempty"`
}

type NodeDevicePCIMDevTypesCapability struct {
	Types []NodeDeviceMDevType `xml:"type"`
}

type NodeDeviceMDevType struct {
	ID                 string `xml:"id,attr"`
	Name               string `xml:"name"`
	DeviceAPI          string `xml:"deviceAPI"`
	AvailableInstances uint   `xml:"availableInstances"`
}

type NodeDevicePCIBridgeCapability struct {
}

type NodeDevicePCIVPDCapability struct {
	Name      string                    `xml:"name,omitempty"`
	ReadOnly  *NodeDevicePCIVPDFieldsRO `xml:"-"`
	ReadWrite *NodeDevicePCIVPDFieldsRW `xml:"-"`
}

type NodeDevicePCIVPDFieldsRO struct {
	ChangeLevel   string                        `xml:"change_level,omitempty"`
	ManufactureID string                        `xml:"manufacture_id,omitempty"`
	PartNumber    string                        `xml:"part_number,omitempty"`
	SerialNumber  string                        `xml:"serial_number,omitempty"`
	VendorFields  []NodeDevicePCIVPDCustomField `xml:"vendor_field"`
}

type NodeDevicePCIVPDFieldsRW struct {
	AssetTag     string                        `xml:"asset_tag,omitempty"`
	VendorFields []NodeDevicePCIVPDCustomField `xml:"vendor_field"`
	SystemFields []NodeDevicePCIVPDCustomField `xml:"system_field"`
}

type NodeDevicePCIVPDCustomField struct {
	Index string `xml:"index,attr"`
	Value string `xml:",chardata"`
}

type NodeDeviceSystemHardware struct {
	Vendor  string `xml:"vendor"`
	Version string `xml:"version"`
	Serial  string `xml:"serial"`
	UUID    string `xml:"uuid"`
}

type NodeDeviceSystemFirmware struct {
	Vendor      string `xml:"vendor"`
	Version     string `xml:"version"`
	ReleaseData string `xml:"release_date"`
}

type NodeDeviceSystemCapability struct {
	Product  string                    `xml:"product,omitempty"`
	Hardware *NodeDeviceSystemHardware `xml:"hardware"`
	Firmware *NodeDeviceSystemFirmware `xml:"firmware"`
}

type NodeDeviceUSBDeviceCapability struct {
	Bus     int              `xml:"bus"`
	Device  int              `xml:"device"`
	Product NodeDeviceIDName `xml:"product,omitempty"`
	Vendor  NodeDeviceIDName `xml:"vendor,omitempty"`
}

type NodeDeviceUSBCapability struct {
	Number      int    `xml:"number"`
	Class       int    `xml:"class"`
	Subclass    int    `xml:"subclass"`
	Protocol    int    `xml:"protocol"`
	Description string `xml:"description,omitempty"`
}

type NodeDeviceNetOffloadFeatures struct {
	Name string `xml:"name,attr"`
}

type NodeDeviceNetLink struct {
	State string `xml:"state,attr"`
	Speed string `xml:"speed,attr,omitempty"`
}

type NodeDeviceNetSubCapability struct {
	Wireless80211 *NodeDeviceNet80211Capability
	Ethernet80203 *NodeDeviceNet80203Capability
}

type NodeDeviceNet80211Capability struct {
}

type NodeDeviceNet80203Capability struct {
}

type NodeDeviceNetCapability struct {
	Interface  string                         `xml:"interface"`
	Address    string                         `xml:"address"`
	Link       *NodeDeviceNetLink             `xml:"link"`
	Features   []NodeDeviceNetOffloadFeatures `xml:"feature,omitempty"`
	Capability []NodeDeviceNetSubCapability   `xml:"capability"`
}

type NodeDeviceSCSIVPortOpsCapability struct {
	VPorts    int `xml:"vports"`
	MaxVPorts int `xml:"max_vports"`
}

type NodeDeviceSCSIFCHostCapability struct {
	WWNN      string `xml:"wwnn,omitempty"`
	WWPN      string `xml:"wwpn,omitempty"`
	FabricWWN string `xml:"fabric_wwn,omitempty"`
}

type NodeDeviceSCSIHostSubCapability struct {
	VPortOps *NodeDeviceSCSIVPortOpsCapability
	FCHost   *NodeDeviceSCSIFCHostCapability
}

type NodeDeviceSCSIHostCapability struct {
	Host       uint                              `xml:"host"`
	UniqueID   *uint                             `xml:"unique_id"`
	Capability []NodeDeviceSCSIHostSubCapability `xml:"capability"`
}

type NodeDeviceSCSITargetCapability struct {
	Target     string                              `xml:"target"`
	Capability []NodeDeviceSCSITargetSubCapability `xml:"capability"`
}

type NodeDeviceSCSITargetSubCapability struct {
	FCRemotePort *NodeDeviceSCSIFCRemotePortCapability
}

type NodeDeviceSCSIFCRemotePortCapability struct {
	RPort string `xml:"rport"`
	WWPN  string `xml:"wwpn"`
}

type NodeDeviceSCSICapability struct {
	Host   int    `xml:"host"`
	Bus    int    `xml:"bus"`
	Target int    `xml:"target"`
	Lun    int    `xml:"lun"`
	Type   string `xml:"type"`
}

type NodeDeviceStorageSubCapability struct {
	Removable *NodeDeviceStorageRemovableCapability
}

type NodeDeviceStorageRemovableCapability struct {
	MediaAvailable   *uint  `xml:"media_available"`
	MediaSize        *uint  `xml:"media_size"`
	MediaLabel       string `xml:"media_label,omitempty"`
	LogicalBlockSize *uint  `xml:"logical_block_size"`
	NumBlocks        *uint  `xml:"num_blocks"`
}

type NodeDeviceStorageCapability struct {
	Block            string                           `xml:"block,omitempty"`
	Bus              string                           `xml:"bus,omitempty"`
	DriverType       string                           `xml:"drive_type,omitempty"`
	Model            string                           `xml:"model,omitempty"`
	Vendor           string                           `xml:"vendor,omitempty"`
	Serial           string                           `xml:"serial,omitempty"`
	Size             *uint                            `xml:"size"`
	LogicalBlockSize *uint                            `xml:"logical_block_size"`
	NumBlocks        *uint                            `xml:"num_blocks"`
	Capability       []NodeDeviceStorageSubCapability `xml:"capability"`
}

type NodeDeviceDRMCapability struct {
	Type string `xml:"type"`
}

type NodeDeviceCCWCapability struct {
	CSSID *uint `xml:"cssid"`
	SSID  *uint `xml:"ssid"`
	DevNo *uint `xml:"devno"`
}

type NodeDeviceMDevCapability struct {
	Type       *NodeDeviceMDevCapabilityType   `xml:"type"`
	IOMMUGroup *NodeDeviceIOMMUGroup           `xml:"iommuGroup"`
	UUID       string                          `xml:"uuid,omitempty"`
	ParentAddr string                          `xml:"parent_addr,omitempty"`
	Attrs      []NodeDeviceMDevCapabilityAttrs `xml:"attr,omitempty"`
}

type NodeDeviceMDevCapabilityType struct {
	ID string `xml:"id,attr"`
}

type NodeDeviceMDevCapabilityAttrs struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type NodeDeviceCSSCapability struct {
	CSSID          *uint                        `xml:"cssid"`
	SSID           *uint                        `xml:"ssid"`
	DevNo          *uint                        `xml:"devno"`
	ChannelDevAddr *NodeDeviceCSSChannelDevAddr `xml:"channel_dev_addr"`
	Capabilities   []NodeDeviceCSSSubCapability `xml:"capability"`
}

type NodeDeviceCSSChannelDevAddr struct {
	CSSID *uint `xml:"cssid"`
	SSID  *uint `xml:"ssid"`
	DevNo *uint `xml:"devno"`
}

type NodeDeviceCSSSubCapability struct {
	MDevTypes *NodeDeviceCSSMDevTypesCapability
}

type NodeDeviceCSSMDevTypesCapability struct {
	Types []NodeDeviceMDevType `xml:"type"`
}

type NodeDeviceAPQueueCapability struct {
	APAdapter string `xml:"ap-adapter"`
	APDomain  string `xml:"ap-domain"`
}

type NodeDeviceAPCardCapability struct {
	APAdapter string `xml:"ap-adapter"`
}

type NodeDeviceAPMatrixCapability struct {
	Capabilities []NodeDeviceAPMatrixSubCapability `xml:"capability"`
}

type NodeDeviceAPMatrixSubCapability struct {
	MDevTypes *NodeDeviceAPMatrixMDevTypesCapability
}

type NodeDeviceAPMatrixMDevTypesCapability struct {
	Types []NodeDeviceMDevType `xml:"type"`
}

func (a *NodeDevicePCIAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "domain", a.Domain, "0x%04x")
	marshalUintAttr(&start, "bus", a.Bus, "0x%02x")
	marshalUintAttr(&start, "slot", a.Slot, "0x%02x")
	marshalUintAttr(&start, "function", a.Function, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *NodeDevicePCIAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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

func (c *NodeDeviceCSSSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "mdev_types":
		var mdevTypesCaps NodeDeviceCSSMDevTypesCapability
		if err := d.DecodeElement(&mdevTypesCaps, &start); err != nil {
			return err
		}
		c.MDevTypes = &mdevTypesCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceCSSSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.MDevTypes != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "mdev_types",
		})
		return e.EncodeElement(c.MDevTypes, start)
	}
	return nil
}

func (c *NodeDeviceCCWCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	if c.CSSID != nil {
		cssid := xml.StartElement{
			Name: xml.Name{Local: "cssid"},
		}
		e.EncodeToken(cssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.CSSID)))
		e.EncodeToken(cssid.End())
	}
	if c.SSID != nil {
		ssid := xml.StartElement{
			Name: xml.Name{Local: "ssid"},
		}
		e.EncodeToken(ssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.SSID)))
		e.EncodeToken(ssid.End())
	}
	if c.DevNo != nil {
		devno := xml.StartElement{
			Name: xml.Name{Local: "devno"},
		}
		e.EncodeToken(devno)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%04x", *c.DevNo)))
		e.EncodeToken(devno.End())
	}
	e.EncodeToken(start.End())
	return nil
}

func (c *NodeDeviceCCWCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			cdata, err := d.Token()
			if err != nil {
				return err
			}

			if tok.Name.Local != "cssid" &&
				tok.Name.Local != "ssid" &&
				tok.Name.Local != "devno" {
				continue
			}

			chardata, ok := cdata.(xml.CharData)
			if !ok {
				return fmt.Errorf("Expected text for CCW '%s'", tok.Name.Local)
			}

			valstr := strings.TrimPrefix(string(chardata), "0x")
			val, err := strconv.ParseUint(valstr, 16, 64)
			if err != nil {
				return err
			}

			vali := uint(val)
			if tok.Name.Local == "cssid" {
				c.CSSID = &vali
			} else if tok.Name.Local == "ssid" {
				c.SSID = &vali
			} else if tok.Name.Local == "devno" {
				c.DevNo = &vali
			}
		}
	}
	return nil
}

func (c *NodeDeviceCSSCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	if c.CSSID != nil {
		cssid := xml.StartElement{
			Name: xml.Name{Local: "cssid"},
		}
		e.EncodeToken(cssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.CSSID)))
		e.EncodeToken(cssid.End())
	}
	if c.SSID != nil {
		ssid := xml.StartElement{
			Name: xml.Name{Local: "ssid"},
		}
		e.EncodeToken(ssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.SSID)))
		e.EncodeToken(ssid.End())
	}
	if c.DevNo != nil {
		devno := xml.StartElement{
			Name: xml.Name{Local: "devno"},
		}
		e.EncodeToken(devno)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%04x", *c.DevNo)))
		e.EncodeToken(devno.End())
	}
	if c.ChannelDevAddr != nil {
		start := xml.StartElement{
			Name: xml.Name{Local: "channel_dev_addr"},
		}
		e.EncodeElement(c.ChannelDevAddr, start)
	}
	if c.Capabilities != nil {
		for _, subcap := range c.Capabilities {
			start := xml.StartElement{
				Name: xml.Name{Local: "capability"},
			}
			e.EncodeElement(&subcap, start)
		}
	}
	e.EncodeToken(start.End())
	return nil
}

func (c *NodeDeviceCSSCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			cdata, err := d.Token()
			if err != nil {
				return err
			}

			if tok.Name.Local == "capability" {
				subcap := &NodeDeviceCSSSubCapability{}
				err := d.DecodeElement(subcap, &tok)
				if err != nil {
					return err
				}
				c.Capabilities = append(c.Capabilities, *subcap)
				continue
			} else if tok.Name.Local == "channel_dev_addr" {
				chandev := &NodeDeviceCSSChannelDevAddr{}
				err := d.DecodeElement(chandev, &tok)
				if err != nil {
					return err
				}
				c.ChannelDevAddr = chandev
				continue
			}

			if tok.Name.Local != "cssid" &&
				tok.Name.Local != "ssid" &&
				tok.Name.Local != "devno" {
				continue
			}

			chardata, ok := cdata.(xml.CharData)
			if !ok {
				return fmt.Errorf("Expected text for CSS '%s'", tok.Name.Local)
			}

			valstr := strings.TrimPrefix(string(chardata), "0x")
			val, err := strconv.ParseUint(valstr, 16, 64)
			if err != nil {
				return err
			}

			vali := uint(val)
			if tok.Name.Local == "cssid" {
				c.CSSID = &vali
			} else if tok.Name.Local == "ssid" {
				c.SSID = &vali
			} else if tok.Name.Local == "devno" {
				c.DevNo = &vali
			}
		}
	}
	return nil
}

func (c *NodeDeviceCSSChannelDevAddr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	if c.CSSID != nil {
		cssid := xml.StartElement{
			Name: xml.Name{Local: "cssid"},
		}
		e.EncodeToken(cssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.CSSID)))
		e.EncodeToken(cssid.End())
	}
	if c.SSID != nil {
		ssid := xml.StartElement{
			Name: xml.Name{Local: "ssid"},
		}
		e.EncodeToken(ssid)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%x", *c.SSID)))
		e.EncodeToken(ssid.End())
	}
	if c.DevNo != nil {
		devno := xml.StartElement{
			Name: xml.Name{Local: "devno"},
		}
		e.EncodeToken(devno)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%04x", *c.DevNo)))
		e.EncodeToken(devno.End())
	}
	e.EncodeToken(start.End())
	return nil
}

func (c *NodeDeviceCSSChannelDevAddr) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			cdata, err := d.Token()
			if err != nil {
				return err
			}

			if tok.Name.Local != "cssid" &&
				tok.Name.Local != "ssid" &&
				tok.Name.Local != "devno" {
				continue
			}

			chardata, ok := cdata.(xml.CharData)
			if !ok {
				return fmt.Errorf("Expected text for CSS '%s'", tok.Name.Local)
			}

			valstr := strings.TrimPrefix(string(chardata), "0x")
			val, err := strconv.ParseUint(valstr, 16, 64)
			if err != nil {
				return err
			}

			vali := uint(val)
			if tok.Name.Local == "cssid" {
				c.CSSID = &vali
			} else if tok.Name.Local == "ssid" {
				c.SSID = &vali
			} else if tok.Name.Local == "devno" {
				c.DevNo = &vali
			}
		}
	}
	return nil
}

func (c *NodeDevicePCISubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "virt_functions":
		var virtFuncCaps NodeDevicePCIVirtFunctionsCapability
		if err := d.DecodeElement(&virtFuncCaps, &start); err != nil {
			return err
		}
		c.VirtFunctions = &virtFuncCaps
	case "phys_function":
		var physFuncCaps NodeDevicePCIPhysFunctionCapability
		if err := d.DecodeElement(&physFuncCaps, &start); err != nil {
			return err
		}
		c.PhysFunction = &physFuncCaps
	case "mdev_types":
		var mdevTypeCaps NodeDevicePCIMDevTypesCapability
		if err := d.DecodeElement(&mdevTypeCaps, &start); err != nil {
			return err
		}
		c.MDevTypes = &mdevTypeCaps
	case "pci-bridge":
		var bridgeCaps NodeDevicePCIBridgeCapability
		if err := d.DecodeElement(&bridgeCaps, &start); err != nil {
			return err
		}
		c.Bridge = &bridgeCaps
	case "vpd":
		var vpdCaps NodeDevicePCIVPDCapability
		if err := d.DecodeElement(&vpdCaps, &start); err != nil {
			return err
		}
		c.VPD = &vpdCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDevicePCISubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.VirtFunctions != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "virt_functions",
		})
		return e.EncodeElement(c.VirtFunctions, start)
	} else if c.PhysFunction != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "phys_function",
		})
		return e.EncodeElement(c.PhysFunction, start)
	} else if c.MDevTypes != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "mdev_types",
		})
		return e.EncodeElement(c.MDevTypes, start)
	} else if c.Bridge != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci-bridge",
		})
		return e.EncodeElement(c.Bridge, start)
	} else if c.VPD != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "vpd",
		})
		return e.EncodeElement(c.VPD, start)
	}
	return nil
}

func (c *NodeDeviceSCSITargetSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "fc_remote_port":
		var fcCaps NodeDeviceSCSIFCRemotePortCapability
		if err := d.DecodeElement(&fcCaps, &start); err != nil {
			return err
		}
		c.FCRemotePort = &fcCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceSCSITargetSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.FCRemotePort != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "fc_remote_port",
		})
		return e.EncodeElement(c.FCRemotePort, start)
	}
	return nil
}

func (c *NodeDeviceSCSIHostSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "fc_host":
		var fcCaps NodeDeviceSCSIFCHostCapability
		if err := d.DecodeElement(&fcCaps, &start); err != nil {
			return err
		}
		c.FCHost = &fcCaps
	case "vport_ops":
		var vportCaps NodeDeviceSCSIVPortOpsCapability
		if err := d.DecodeElement(&vportCaps, &start); err != nil {
			return err
		}
		c.VPortOps = &vportCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceSCSIHostSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.FCHost != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "fc_host",
		})
		return e.EncodeElement(c.FCHost, start)
	} else if c.VPortOps != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "vport_ops",
		})
		return e.EncodeElement(c.VPortOps, start)
	}
	return nil
}

func (c *NodeDeviceStorageSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "removable":
		var removeCaps NodeDeviceStorageRemovableCapability
		if err := d.DecodeElement(&removeCaps, &start); err != nil {
			return err
		}
		c.Removable = &removeCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceStorageSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.Removable != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "removable",
		})
		return e.EncodeElement(c.Removable, start)
	}
	return nil
}

func (c *NodeDeviceNetSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "80211":
		var wlanCaps NodeDeviceNet80211Capability
		if err := d.DecodeElement(&wlanCaps, &start); err != nil {
			return err
		}
		c.Wireless80211 = &wlanCaps
	case "80203":
		var ethCaps NodeDeviceNet80203Capability
		if err := d.DecodeElement(&ethCaps, &start); err != nil {
			return err
		}
		c.Ethernet80203 = &ethCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceNetSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.Wireless80211 != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "80211",
		})
		return e.EncodeElement(c.Wireless80211, start)
	} else if c.Ethernet80203 != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "80203",
		})
		return e.EncodeElement(c.Ethernet80203, start)
	}
	return nil
}

func (c *NodeDeviceAPMatrixSubCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "mdev_types":
		var mdevTypeCaps NodeDeviceAPMatrixMDevTypesCapability
		if err := d.DecodeElement(&mdevTypeCaps, &start); err != nil {
			return err
		}
		c.MDevTypes = &mdevTypeCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceAPMatrixSubCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.MDevTypes != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "mdev_types",
		})
		return e.EncodeElement(c.MDevTypes, start)
	}
	return nil
}

type nodeDevicePCIVPDFields struct {
	ReadOnly  *NodeDevicePCIVPDFieldsRO
	ReadWrite *NodeDevicePCIVPDFieldsRW
}

type nodeDevicePCIVPDCapability struct {
	Name   string                   `xml:"name,omitempty"`
	Fields []nodeDevicePCIVPDFields `xml:"fields"`
}

func (c *nodeDevicePCIVPDFields) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	acc, ok := getAttr(start.Attr, "access")
	if !ok {
		return fmt.Errorf("Missing node device PCI VPD capability access")
	}

	switch acc {
	case "readonly":
		var ro NodeDevicePCIVPDFieldsRO
		if err := d.DecodeElement(&ro, &start); err != nil {
			return err
		}
		c.ReadOnly = &ro
	case "readwrite":
		var rw NodeDevicePCIVPDFieldsRW
		if err := d.DecodeElement(&rw, &start); err != nil {
			return err
		}
		c.ReadWrite = &rw
	}
	d.Skip()
	return nil
}

func (c *NodeDevicePCIVPDCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	var ccopy nodeDevicePCIVPDCapability
	ccopy.Name = c.Name
	if c.ReadOnly != nil {
		ccopy.Fields = append(ccopy.Fields, nodeDevicePCIVPDFields{
			ReadOnly: c.ReadOnly,
		})
	}
	if c.ReadWrite != nil {
		ccopy.Fields = append(ccopy.Fields, nodeDevicePCIVPDFields{
			ReadWrite: c.ReadWrite,
		})
	}
	e.EncodeElement(&ccopy, start)
	return nil
}

func (c *NodeDevicePCIVPDCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var ccopy nodeDevicePCIVPDCapability
	if err := d.DecodeElement(&ccopy, &start); err != nil {
		return err
	}
	c.Name = ccopy.Name
	for _, field := range ccopy.Fields {
		if field.ReadOnly != nil {
			c.ReadOnly = field.ReadOnly
		} else if field.ReadWrite != nil {
			c.ReadWrite = field.ReadWrite
		}
	}
	return nil
}

func (c *nodeDevicePCIVPDFields) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.ReadOnly != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "access"}, "readonly",
		})
		return e.EncodeElement(c.ReadOnly, start)
	} else if c.ReadWrite != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "access"}, "readwrite",
		})
		return e.EncodeElement(c.ReadWrite, start)
	}
	return nil
}

func (c *NodeDeviceCapability) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing node device capability type")
	}

	switch typ {
	case "pci":
		var pciCaps NodeDevicePCICapability
		if err := d.DecodeElement(&pciCaps, &start); err != nil {
			return err
		}
		c.PCI = &pciCaps
	case "system":
		var systemCaps NodeDeviceSystemCapability
		if err := d.DecodeElement(&systemCaps, &start); err != nil {
			return err
		}
		c.System = &systemCaps
	case "usb_device":
		var usbdevCaps NodeDeviceUSBDeviceCapability
		if err := d.DecodeElement(&usbdevCaps, &start); err != nil {
			return err
		}
		c.USBDevice = &usbdevCaps
	case "usb":
		var usbCaps NodeDeviceUSBCapability
		if err := d.DecodeElement(&usbCaps, &start); err != nil {
			return err
		}
		c.USB = &usbCaps
	case "net":
		var netCaps NodeDeviceNetCapability
		if err := d.DecodeElement(&netCaps, &start); err != nil {
			return err
		}
		c.Net = &netCaps
	case "scsi_host":
		var scsiHostCaps NodeDeviceSCSIHostCapability
		if err := d.DecodeElement(&scsiHostCaps, &start); err != nil {
			return err
		}
		c.SCSIHost = &scsiHostCaps
	case "scsi_target":
		var scsiTargetCaps NodeDeviceSCSITargetCapability
		if err := d.DecodeElement(&scsiTargetCaps, &start); err != nil {
			return err
		}
		c.SCSITarget = &scsiTargetCaps
	case "scsi":
		var scsiCaps NodeDeviceSCSICapability
		if err := d.DecodeElement(&scsiCaps, &start); err != nil {
			return err
		}
		c.SCSI = &scsiCaps
	case "storage":
		var storageCaps NodeDeviceStorageCapability
		if err := d.DecodeElement(&storageCaps, &start); err != nil {
			return err
		}
		c.Storage = &storageCaps
	case "drm":
		var drmCaps NodeDeviceDRMCapability
		if err := d.DecodeElement(&drmCaps, &start); err != nil {
			return err
		}
		c.DRM = &drmCaps
	case "ccw":
		var ccwCaps NodeDeviceCCWCapability
		if err := d.DecodeElement(&ccwCaps, &start); err != nil {
			return err
		}
		c.CCW = &ccwCaps
	case "mdev":
		var mdevCaps NodeDeviceMDevCapability
		if err := d.DecodeElement(&mdevCaps, &start); err != nil {
			return err
		}
		c.MDev = &mdevCaps
	case "css":
		var cssCaps NodeDeviceCSSCapability
		if err := d.DecodeElement(&cssCaps, &start); err != nil {
			return err
		}
		c.CSS = &cssCaps
	case "ap_queue":
		var apCaps NodeDeviceAPQueueCapability
		if err := d.DecodeElement(&apCaps, &start); err != nil {
			return err
		}
		c.APQueue = &apCaps
	case "ap_matrix":
		var apCaps NodeDeviceAPMatrixCapability
		if err := d.DecodeElement(&apCaps, &start); err != nil {
			return err
		}
		c.APMatrix = &apCaps
	case "ap_card":
		var apCaps NodeDeviceAPCardCapability
		if err := d.DecodeElement(&apCaps, &start); err != nil {
			return err
		}
		c.APCard = &apCaps
	}
	d.Skip()
	return nil
}

func (c *NodeDeviceCapability) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if c.PCI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci",
		})
		return e.EncodeElement(c.PCI, start)
	} else if c.System != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "system",
		})
		return e.EncodeElement(c.System, start)
	} else if c.USB != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "usb",
		})
		return e.EncodeElement(c.USB, start)
	} else if c.USBDevice != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "usb_device",
		})
		return e.EncodeElement(c.USBDevice, start)
	} else if c.Net != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "net",
		})
		return e.EncodeElement(c.Net, start)
	} else if c.SCSI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "scsi",
		})
		return e.EncodeElement(c.SCSI, start)
	} else if c.SCSIHost != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "scsi_host",
		})
		return e.EncodeElement(c.SCSIHost, start)
	} else if c.SCSITarget != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "scsi_target",
		})
		return e.EncodeElement(c.SCSITarget, start)
	} else if c.Storage != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "storage",
		})
		return e.EncodeElement(c.Storage, start)
	} else if c.DRM != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "drm",
		})
		return e.EncodeElement(c.DRM, start)
	} else if c.CCW != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ccw",
		})
		return e.EncodeElement(c.CCW, start)
	} else if c.MDev != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "mdev",
		})
		return e.EncodeElement(c.MDev, start)
	} else if c.CSS != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "css",
		})
		return e.EncodeElement(c.CSS, start)
	} else if c.APQueue != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ap_queue",
		})
		return e.EncodeElement(c.APQueue, start)
	} else if c.APCard != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ap_card",
		})
		return e.EncodeElement(c.APCard, start)
	} else if c.APMatrix != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ap_matrix",
		})
		return e.EncodeElement(c.APMatrix, start)
	}
	return nil
}

func (c *NodeDevice) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), c)
}

func (c *NodeDevice) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
