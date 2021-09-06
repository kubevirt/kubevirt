package api

import (
	"encoding/xml"
	"fmt"

	hwutil "kubevirt.io/kubevirt/pkg/util/hardware"
)

type yesnobool bool

type CPUSiblings []uint32

type Capabilities struct {
	XMLName xml.Name `xml:"capabilities"`
	Host    Host     `xml:"host"`
}

type Host struct {
	UUID     string   `xml:"uuid"`
	CPU      HostCPU  `xml:"cpu"`
	Topology Topology `xml:"topology"`
}

type HostCPU struct {
	Arch    string    `xml:"arch"`
	Model   string    `xml:"model"`
	Vendor  string    `xml:"vendor"`
	Counter []Counter `xml:"counter"`
}

type Counter struct {
	Name      string    `xml:"name,attr"`
	Frequency int64     `xml:"frequency,attr"`
	Scaling   yesnobool `xml:"scaling,attr"`
}

type CPU struct {
	ID       uint32      `xml:"id,attr"`
	SocketID uint32      `xml:"socket_id,attr"`
	DieID    uint32      `xml:"die_id,attr"`
	CoreID   uint32      `xml:"core_id,attr"`
	Siblings CPUSiblings `xml:"siblings,attr"`
}

type CPUs struct {
	Num int   `xml:"num,attr"`
	CPU []CPU `xml:"cpu"`
}

type Distances struct {
	Sibling []Sibling `xml:"sibling"`
}

type Sibling struct {
	ID    uint32 `xml:"id,attr"`
	Value uint64 `xml:"value,attr"`
}

type Pages struct {
	Count uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr"`
	Size  uint32 `xml:"size,attr"`
}

type Memory struct {
	Amount uint64 `xml:",chardata"`
	Unit   string `xml:"unit,attr"`
}

type Cell struct {
	ID        uint32    `xml:"id,attr"`
	Memory    Memory    `xml:"memory"`
	Pages     []Pages   `xml:"pages"`
	Distances Distances `xml:"distances"`
	Cpus      CPUs      `xml:"cpus"`
}

type Cells struct {
	Num  uint32 `xml:"num,attr"`
	Cell []Cell `xml:"cell"`
}

type Topology struct {
	Cells Cells `xml:"cells"`
}

func (c *Capabilities) GetTSCCounter() (*Counter, error) {
	for _, c := range c.Host.CPU.Counter {
		if c.Name == "tsc" {
			return &c, nil
		}
	}
	return nil, nil
}

func (b *yesnobool) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value == "yes" {
		*b = true
		return nil
	} else if attr.Value == "no" {
		*b = false
		return nil
	}
	return fmt.Errorf("value %v of %v is not (yes|no)", attr.Value, attr.Name)
}

func (b *CPUSiblings) UnmarshalXMLAttr(attr xml.Attr) error {
	if attr.Value != "" {
		if list, err := hwutil.ParseCPUSetLine(attr.Value, 100); err == nil {
			for _, cpu := range list {
				*b = append(*b, uint32(cpu))
			}
		} else {
			return fmt.Errorf("failed to parse %v to ints: %v", attr.Value, err)

		}
	}
	return nil
}
