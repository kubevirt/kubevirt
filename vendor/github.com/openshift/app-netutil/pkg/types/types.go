package types

import (
	cnitypes "github.com/containernetworking/cni/pkg/types"
)

const (
	INTERFACE_TYPE_ALL = "all"
	INTERFACE_TYPE_KERNEL = "kernel"
	INTERFACE_TYPE_SRIOV = "sr-iov"
	INTERFACE_TYPE_VHOST = "vhost"
	INTERFACE_TYPE_MEMIF = "memif"
	INTERFACE_TYPE_VDPA = "vDPA"

	INTERFACE_TYPE_UNKNOWN = "unknown"
	INTERFACE_TYPE_INVALID = "invalid"
)

type CPUResponse struct {
	CPUSet	string	`json:"cpuset,omitempty"`
}

type InterfaceResponse struct {
	Interface  []*InterfaceData
}

type InterfaceData struct {
	IfName  string        `json:"ifName,omitempty"`  // IfName, from CNIArgs, if available
	Name    string        `json:"name,omitempty"`    // Name from Network-Attachment-Definition, if available
	Type    string        `json:"type,omitempty"`    // Of Type INTERFACE_TYPE_xxx
	Network *NetworkData  `json:"network,omitempty"`

	// Per Interface Type Data
	Sriov   *SriovData    `json:"sriov,omitempty"`
	Memif   *MemifData    `json:"memif,omitempty"`
	Vhost   *VhostData    `json:"vhost,omitempty"`
	Vdpa    *VdpaData     `json:"vDPA,omitempty"`
}

type NetworkData struct {
	IPs       []string  `json:"ips,omitempty"`
	Mac       string    `json:"mac,omitempty"`
	Default   bool      `json:"default,omitempty"`
	DNS       cnitypes.DNS `json:"dns,omitempty"`
}

type SriovData struct {
	PciAddress  string  `json:"pciAddress,omitempty"`
}

type VhostData struct {
	Socketpath  string  `json:"socketpath,omitempty"`  // Unix Socketfile for control neg.
	Mode        string  `json:"mode,omitempty"`        // Mode: client|server
}

type MemifData struct {
	Socketpath  string  `json:"socketpath,omitempty"`  // Unix Socketfile for control neg.
	Role        string  `json:"role,omitempty"`        // Role of memif: master|slave
	Mode        string  `json:"mode,omitempty"`        // Mode of memif: ip|ethernet|inject-punt
}

type VdpaData struct {
	// TBD
}
