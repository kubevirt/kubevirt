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
 * Copyright (C) 2017 Lian Duan <blazeblue@gmail.com>
 *
 */

package libvirtxml

import (
	"encoding/xml"
)

type NetworkBridge struct {
	Name            string `xml:"name,attr,omitempty"`
	STP             string `xml:"stp,attr,omitempty"`
	Delay           string `xml:"delay,attr,omitempty"`
	MACTableManager string `xml:"macTableManager,attr,omitempty"`
	Zone            string `xml:"zone,attr,omitempty"`
}

type NetworkVirtualPort struct {
	Params *NetworkVirtualPortParams `xml:"parameters"`
}

type NetworkVirtualPortParams struct {
	Any          *NetworkVirtualPortParamsAny          `xml:"-"`
	VEPA8021QBG  *NetworkVirtualPortParamsVEPA8021QBG  `xml:"-"`
	VNTag8011QBH *NetworkVirtualPortParamsVNTag8021QBH `xml:"-"`
	OpenVSwitch  *NetworkVirtualPortParamsOpenVSwitch  `xml:"-"`
	MidoNet      *NetworkVirtualPortParamsMidoNet      `xml:"-"`
}

type NetworkVirtualPortParamsAny struct {
	ManagerID     *uint  `xml:"managerid,attr"`
	TypeID        *uint  `xml:"typeid,attr"`
	TypeIDVersion *uint  `xml:"typeidversion,attr"`
	InstanceID    string `xml:"instanceid,attr,omitempty"`
	ProfileID     string `xml:"profileid,attr,omitempty"`
	InterfaceID   string `xml:"interfaceid,attr,omitempty"`
}

type NetworkVirtualPortParamsVEPA8021QBG struct {
	ManagerID     *uint  `xml:"managerid,attr"`
	TypeID        *uint  `xml:"typeid,attr"`
	TypeIDVersion *uint  `xml:"typeidversion,attr"`
	InstanceID    string `xml:"instanceid,attr,omitempty"`
}

type NetworkVirtualPortParamsVNTag8021QBH struct {
	ProfileID string `xml:"profileid,attr,omitempty"`
}

type NetworkVirtualPortParamsOpenVSwitch struct {
	InterfaceID string `xml:"interfaceid,attr,omitempty"`
	ProfileID   string `xml:"profileid,attr,omitempty"`
}

type NetworkVirtualPortParamsMidoNet struct {
	InterfaceID string `xml:"interfaceid,attr,omitempty"`
}

type NetworkDomain struct {
	Name      string `xml:"name,attr,omitempty"`
	LocalOnly string `xml:"localOnly,attr,omitempty"`
}

type NetworkForwardNATAddress struct {
	Start string `xml:"start,attr"`
	End   string `xml:"end,attr"`
}

type NetworkForwardNATPort struct {
	Start uint `xml:"start,attr"`
	End   uint `xml:"end,attr"`
}

type NetworkForwardNAT struct {
	IPv6      string                     `xml:"ipv6,attr,omitempty"`
	Addresses []NetworkForwardNATAddress `xml:"address"`
	Ports     []NetworkForwardNATPort    `xml:"port"`
}

type NetworkForward struct {
	Mode       string                    `xml:"mode,attr,omitempty"`
	Dev        string                    `xml:"dev,attr,omitempty"`
	Managed    string                    `xml:"managed,attr,omitempty"`
	Driver     *NetworkForwardDriver     `xml:"driver"`
	PFs        []NetworkForwardPF        `xml:"pf"`
	NAT        *NetworkForwardNAT        `xml:"nat"`
	Interfaces []NetworkForwardInterface `xml:"interface"`
	Addresses  []NetworkForwardAddress   `xml:"address"`
}

type NetworkForwardDriver struct {
	Name  string `xml:"name,attr,omitempty"`
	Model string `xml:"model,attr,omitempty"`
}

type NetworkForwardPF struct {
	Dev string `xml:"dev,attr"`
}

type NetworkForwardAddress struct {
	PCI *NetworkForwardAddressPCI `xml:"-"`
}

type NetworkForwardAddressPCI struct {
	Domain   *uint `xml:"domain,attr"`
	Bus      *uint `xml:"bus,attr"`
	Slot     *uint `xml:"slot,attr"`
	Function *uint `xml:"function,attr"`
}

type NetworkForwardInterface struct {
	XMLName xml.Name `xml:"interface"`
	Dev     string   `xml:"dev,attr,omitempty"`
}

type NetworkMAC struct {
	Address string `xml:"address,attr,omitempty"`
}

type NetworkDHCPRange struct {
	XMLName xml.Name          `xml:"range"`
	Start   string            `xml:"start,attr,omitempty"`
	End     string            `xml:"end,attr,omitempty"`
	Lease   *NetworkDHCPLease `xml:"lease"`
}

type NetworkDHCPLease struct {
	Expiry uint   `xml:"expiry,attr"`
	Unit   string `xml:"unit,attr,omitempty"`
}

type NetworkDHCPHost struct {
	XMLName xml.Name          `xml:"host"`
	ID      string            `xml:"id,attr,omitempty"`
	MAC     string            `xml:"mac,attr,omitempty"`
	Name    string            `xml:"name,attr,omitempty"`
	IP      string            `xml:"ip,attr,omitempty"`
	Lease   *NetworkDHCPLease `xml:"lease"`
}

type NetworkBootp struct {
	File   string `xml:"file,attr,omitempty"`
	Server string `xml:"server,attr,omitempty"`
}

type NetworkDHCP struct {
	Ranges []NetworkDHCPRange `xml:"range"`
	Hosts  []NetworkDHCPHost  `xml:"host"`
	Bootp  []NetworkBootp     `xml:"bootp"`
}

type NetworkIP struct {
	Address  string       `xml:"address,attr,omitempty"`
	Family   string       `xml:"family,attr,omitempty"`
	Netmask  string       `xml:"netmask,attr,omitempty"`
	Prefix   uint         `xml:"prefix,attr,omitempty"`
	LocalPtr string       `xml:"localPtr,attr,omitempty"`
	DHCP     *NetworkDHCP `xml:"dhcp"`
	TFTP     *NetworkTFTP `xml:"tftp"`
}

type NetworkTFTP struct {
	Root string `xml:"root,attr,omitempty"`
}

type NetworkRoute struct {
	Family  string `xml:"family,attr,omitempty"`
	Address string `xml:"address,attr,omitempty"`
	Netmask string `xml:"netmask,attr,omitempty"`
	Prefix  uint   `xml:"prefix,attr,omitempty"`
	Gateway string `xml:"gateway,attr,omitempty"`
	Metric  string `xml:"metric,attr,omitempty"`
}

type NetworkDNSForwarder struct {
	Domain string `xml:"domain,attr,omitempty"`
	Addr   string `xml:"addr,attr,omitempty"`
}

type NetworkDNSTXT struct {
	XMLName xml.Name `xml:"txt"`
	Name    string   `xml:"name,attr"`
	Value   string   `xml:"value,attr"`
}

type NetworkDNSHostHostname struct {
	Hostname string `xml:",chardata"`
}

type NetworkDNSHost struct {
	XMLName   xml.Name                 `xml:"host"`
	IP        string                   `xml:"ip,attr"`
	Hostnames []NetworkDNSHostHostname `xml:"hostname"`
}

type NetworkDNSSRV struct {
	XMLName  xml.Name `xml:"srv"`
	Service  string   `xml:"service,attr,omitempty"`
	Protocol string   `xml:"protocol,attr,omitempty"`
	Target   string   `xml:"target,attr,omitempty"`
	Port     uint     `xml:"port,attr,omitempty"`
	Priority uint     `xml:"priority,attr,omitempty"`
	Weight   uint     `xml:"weight,attr,omitempty"`
	Domain   string   `xml:"domain,attr,omitempty"`
}

type NetworkDNS struct {
	Enable            string                `xml:"enable,attr,omitempty"`
	ForwardPlainNames string                `xml:"forwardPlainNames,attr,omitempty"`
	Forwarders        []NetworkDNSForwarder `xml:"forwarder"`
	TXTs              []NetworkDNSTXT       `xml:"txt"`
	Host              []NetworkDNSHost      `xml:"host"`
	SRVs              []NetworkDNSSRV       `xml:"srv"`
}

type NetworkMetadata struct {
	XML string `xml:",innerxml"`
}

type NetworkMTU struct {
	Size uint `xml:"size,attr"`
}

type Network struct {
	XMLName             xml.Name            `xml:"network"`
	IPv6                string              `xml:"ipv6,attr,omitempty"`
	TrustGuestRxFilters string              `xml:"trustGuestRxFilters,attr,omitempty"`
	Name                string              `xml:"name,omitempty"`
	UUID                string              `xml:"uuid,omitempty"`
	Metadata            *NetworkMetadata    `xml:"metadata"`
	Forward             *NetworkForward     `xml:"forward"`
	Bridge              *NetworkBridge      `xml:"bridge"`
	MTU                 *NetworkMTU         `xml:"mtu"`
	MAC                 *NetworkMAC         `xml:"mac"`
	Domain              *NetworkDomain      `xml:"domain"`
	DNS                 *NetworkDNS         `xml:"dns"`
	VLAN                *NetworkVLAN        `xml:"vlan"`
	Bandwidth           *NetworkBandwidth   `xml:"bandwidth"`
	PortOptions         *NetworkPortOptions `xml:"port"`
	IPs                 []NetworkIP         `xml:"ip"`
	Routes              []NetworkRoute      `xml:"route"`
	VirtualPort         *NetworkVirtualPort `xml:"virtualport"`
	PortGroups          []NetworkPortGroup  `xml:"portgroup"`

	DnsmasqOptions *NetworkDnsmasqOptions
}

type NetworkPortOptions struct {
	Isolated string `xml:"isolated,attr,omitempty"`
}

type NetworkPortGroup struct {
	XMLName             xml.Name            `xml:"portgroup"`
	Name                string              `xml:"name,attr,omitempty"`
	Default             string              `xml:"default,attr,omitempty"`
	TrustGuestRxFilters string              `xml:"trustGuestRxFilters,attr,omitempty"`
	VLAN                *NetworkVLAN        `xml:"vlan"`
	VirtualPort         *NetworkVirtualPort `xml:"virtualport"`
}

type NetworkVLAN struct {
	Trunk string           `xml:"trunk,attr,omitempty"`
	Tags  []NetworkVLANTag `xml:"tag"`
}

type NetworkVLANTag struct {
	ID         uint   `xml:"id,attr"`
	NativeMode string `xml:"nativeMode,attr,omitempty"`
}

type NetworkBandwidthParams struct {
	Average *uint `xml:"average,attr"`
	Peak    *uint `xml:"peak,attr"`
	Burst   *uint `xml:"burst,attr"`
	Floor   *uint `xml:"floor,attr"`
}

type NetworkBandwidth struct {
	ClassID  uint                    `xml:"classID,attr,omitempty"`
	Inbound  *NetworkBandwidthParams `xml:"inbound"`
	Outbound *NetworkBandwidthParams `xml:"outbound"`
}

type NetworkDnsmasqOptions struct {
	XMLName xml.Name               `xml:"http://libvirt.org/schemas/network/dnsmasq/1.0 options"`
	Option  []NetworkDnsmasqOption `xml:"option"`
}

type NetworkDnsmasqOption struct {
	Value string `xml:"value,attr"`
}

func (a *NetworkVirtualPortParams) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "parameters"
	if a.Any != nil {
		return e.EncodeElement(a.Any, start)
	} else if a.VEPA8021QBG != nil {
		return e.EncodeElement(a.VEPA8021QBG, start)
	} else if a.VNTag8011QBH != nil {
		return e.EncodeElement(a.VNTag8011QBH, start)
	} else if a.OpenVSwitch != nil {
		return e.EncodeElement(a.OpenVSwitch, start)
	} else if a.MidoNet != nil {
		return e.EncodeElement(a.MidoNet, start)
	}
	return nil
}

func (a *NetworkVirtualPortParams) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.Any != nil {
		return d.DecodeElement(a.Any, &start)
	} else if a.VEPA8021QBG != nil {
		return d.DecodeElement(a.VEPA8021QBG, &start)
	} else if a.VNTag8011QBH != nil {
		return d.DecodeElement(a.VNTag8011QBH, &start)
	} else if a.OpenVSwitch != nil {
		return d.DecodeElement(a.OpenVSwitch, &start)
	} else if a.MidoNet != nil {
		return d.DecodeElement(a.MidoNet, &start)
	}
	return nil
}

type networkVirtualPort NetworkVirtualPort

func (a *NetworkVirtualPort) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "virtualport"
	if a.Params != nil {
		if a.Params.Any != nil {
			/* no type attr wanted */
		} else if a.Params.VEPA8021QBG != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "802.1Qbg",
			})
		} else if a.Params.VNTag8011QBH != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "802.1Qbh",
			})
		} else if a.Params.OpenVSwitch != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "openvswitch",
			})
		} else if a.Params.MidoNet != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "midonet",
			})
		}
	}
	vp := networkVirtualPort(*a)
	return e.EncodeElement(&vp, start)
}

func (a *NetworkVirtualPort) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	a.Params = &NetworkVirtualPortParams{}
	if !ok {
		var any NetworkVirtualPortParamsAny
		a.Params.Any = &any
	} else if typ == "802.1Qbg" {
		var vepa NetworkVirtualPortParamsVEPA8021QBG
		a.Params.VEPA8021QBG = &vepa
	} else if typ == "802.1Qbh" {
		var vntag NetworkVirtualPortParamsVNTag8021QBH
		a.Params.VNTag8011QBH = &vntag
	} else if typ == "openvswitch" {
		var ovs NetworkVirtualPortParamsOpenVSwitch
		a.Params.OpenVSwitch = &ovs
	} else if typ == "midonet" {
		var mido NetworkVirtualPortParamsMidoNet
		a.Params.MidoNet = &mido
	}

	vp := networkVirtualPort(*a)
	err := d.DecodeElement(&vp, &start)
	if err != nil {
		return err
	}
	*a = NetworkVirtualPort(vp)
	return nil
}

func (a *NetworkForwardAddressPCI) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "domain", a.Domain, "0x%04x")
	marshalUintAttr(&start, "bus", a.Bus, "0x%02x")
	marshalUintAttr(&start, "slot", a.Slot, "0x%02x")
	marshalUintAttr(&start, "function", a.Function, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *NetworkForwardAddressPCI) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
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

func (a *NetworkForwardAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.PCI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci",
		})
		return e.EncodeElement(a.PCI, start)
	} else {
		return nil
	}
}

func (a *NetworkForwardAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var typ string
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			typ = attr.Value
			break
		}
	}
	if typ == "" {
		d.Skip()
		return nil
	}

	if typ == "pci" {
		a.PCI = &NetworkForwardAddressPCI{}
		return d.DecodeElement(a.PCI, &start)
	}

	return nil
}

func (s *NetworkDHCPHost) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkDHCPHost) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkDNSHost) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkDNSHost) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkPortGroup) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkPortGroup) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkDNSTXT) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkDNSTXT) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkDNSSRV) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkDNSSRV) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkDHCPRange) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkDHCPRange) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *NetworkForwardInterface) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *NetworkForwardInterface) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (s *Network) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *Network) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
