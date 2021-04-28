package api

import "encoding/xml"

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
	Name      string `xml:"name,attr"`
	Frequency string `xml:"frequency,attr"`
	Scaling   string `xml:"scaling,attr"`
}
