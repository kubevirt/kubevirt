package api

import (
	"encoding/xml"
	"fmt"
)

type yesnobool bool

type Capabilities struct {
	XMLName xml.Name `xml:"capabilities"`
	Host    Host     `xml:"host"`
}

type Host struct {
	UUID string  `xml:"uuid"`
	CPU  HostCPU `xml:"cpu"`
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
