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
 * Copyright (C) 2019 Red Hat, Inc.
 *
 */

package libvirtxml

import (
	"encoding/xml"
	"fmt"
)

type NetworkPort struct {
	XMLName     xml.Name                `xml:"networkport"`
	UUID        string                  `xml:"uuid,omitempty"`
	Owner       *NetworkPortOwner       `xml:"owner",`
	MAC         *NetworkPortMAC         `xml:"mac"`
	Group       string                  `xml:"group,omitempty"`
	Bandwidth   *NetworkBandwidth       `xml:"bandwidth"`
	VLAN        *NetworkPortVLAN        `xml:"vlan"`
	PortOptions *NetworkPortPortOptions `xml:"port"`
	VirtualPort *NetworkVirtualPort     `xml:"virtualport"`
	RXFilters   *NetworkPortRXFilters   `xml:"rxfilters"`
	Plug        *NetworkPortPlug        `xml:"plug"`
}

type NetworkPortPortOptions struct {
	Isolated string `xml:"isolated,attr,omitempty"`
}

type NetworkPortVLAN struct {
	Trunk string               `xml:"trunk,attr,omitempty"`
	Tags  []NetworkPortVLANTag `xml:"tag"`
}

type NetworkPortVLANTag struct {
	ID         uint   `xml:"id,attr"`
	NativeMode string `xml:"nativeMode,attr,omitempty"`
}

type NetworkPortOwner struct {
	UUID string `xml:"uuid,omitempty"`
	Name string `xml:"name,omitempty"`
}

type NetworkPortMAC struct {
	Address string `xml:"address,attr"`
}

type NetworkPortRXFilters struct {
	TrustGuest string `xml:"trustGuest,attr"`
}

type NetworkPortPlug struct {
	Bridge     *NetworkPortPlugBridge     `xml:"-"`
	Network    *NetworkPortPlugNetwork    `xml:"-"`
	Direct     *NetworkPortPlugDirect     `xml:"-"`
	HostDevPCI *NetworkPortPlugHostDevPCI `xml:"-"`
}

type NetworkPortPlugBridge struct {
	Bridge          string `xml:"bridge,attr"`
	MacTableManager string `xml:"macTableManager,attr,omitempty"`
}

type NetworkPortPlugNetwork struct {
	Bridge          string `xml:"bridge,attr"`
	MacTableManager string `xml:"macTableManager,attr,omitempty"`
}

type NetworkPortPlugDirect struct {
	Dev  string `xml:"dev,attr"`
	Mode string `xml:"mode,attr"`
}

type NetworkPortPlugHostDevPCI struct {
	Managed string                            `xml:"managed,attr,omitempty"`
	Driver  *NetworkPortPlugHostDevPCIDriver  `xml:"driver"`
	Address *NetworkPortPlugHostDevPCIAddress `xml:"address"`
}

type NetworkPortPlugHostDevPCIDriver struct {
	Name string `xml:"name,attr"`
}

type NetworkPortPlugHostDevPCIAddress struct {
	Domain   *uint `xml:"domain,attr"`
	Bus      *uint `xml:"bus,attr"`
	Slot     *uint `xml:"slot,attr"`
	Function *uint `xml:"function,attr"`
}

func (a *NetworkPortPlugHostDevPCIAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "domain", a.Domain, "0x%04x")
	marshalUintAttr(&start, "bus", a.Bus, "0x%02x")
	marshalUintAttr(&start, "slot", a.Slot, "0x%02x")
	marshalUintAttr(&start, "function", a.Function, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *NetworkPortPlugHostDevPCIAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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

func (p *NetworkPortPlug) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "plug"
	if p.Bridge != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "bridge",
		})
		return e.EncodeElement(p.Bridge, start)
	} else if p.Network != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "network",
		})
		return e.EncodeElement(p.Network, start)
	} else if p.Direct != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "direct",
		})
		return e.EncodeElement(p.Direct, start)
	} else if p.HostDevPCI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "hostdev-pci",
		})
		return e.EncodeElement(p.HostDevPCI, start)
	}
	return nil
}

func (p *NetworkPortPlug) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing type attribute on plug")
	} else if typ == "bridge" {
		var pb NetworkPortPlugBridge
		if err := d.DecodeElement(&pb, &start); err != nil {
			return err
		}
		p.Bridge = &pb
	} else if typ == "network" {
		var pn NetworkPortPlugNetwork
		if err := d.DecodeElement(&pn, &start); err != nil {
			return err
		}
		p.Network = &pn
	} else if typ == "direct" {
		var pd NetworkPortPlugDirect
		if err := d.DecodeElement(&pd, &start); err != nil {
			return err
		}
		p.Direct = &pd
	} else if typ == "hostdev-pci" {
		var ph NetworkPortPlugHostDevPCI
		if err := d.DecodeElement(&ph, &start); err != nil {
			return err
		}
		p.HostDevPCI = &ph
	}
	d.Skip()
	return nil
}

func (s *NetworkPort) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkPort) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
