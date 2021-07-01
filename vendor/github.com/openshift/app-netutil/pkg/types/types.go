package types

import (
	nettypes "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
)

const (
	INTERFACE_TYPE_HOST    = "host"
	INTERFACE_TYPE_SRIOV   = "sr-iov"
	INTERFACE_TYPE_UNKNOWN = "unknown"
	INTERFACE_TYPE_INVALID = "invalid"
)

type CPUResponse struct {
	CPUSet string `json:"cpuset,omitempty"`
}

type HugepagesResponse struct {
	MyContainerName string
	Hugepages       []*HugepagesData
}

type HugepagesData struct {
	ContainerName string
	Request       int64
	Limit         int64
	Request1G     int64
	Limit1G       int64
	Request2M     int64
	Limit2M       int64
}

type InterfaceResponse struct {
	Interface []*InterfaceData
}

type InterfaceData struct {
	// DeviceType is similar to NetworkStatus.DeviceInfo.Type except:
	// - Don't need to check for "NetworkStatus.DeviceInfo != nil" before using
	// - Internally could be "unknown" or "invalid" while data is being processed
	// - Not all DPs/CNIs support Device-Info-Spec, so NetworkStatus.DeviceInfo may be nil
	// - For NetworkStatus.DeviceInfo.Type of "pci", DeviceType may be "host" or "sriov"
	DeviceType    string                 `json:"device-type,omitempty"`
	NetworkStatus nettypes.NetworkStatus `json:"network-status,omitempty"`
}
