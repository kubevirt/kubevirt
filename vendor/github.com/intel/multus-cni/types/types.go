// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package types

import (
	"net"

	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/current"
	v1 "k8s.io/api/core/v1"
)

// NetConf for cni config file written in json
type NetConf struct {
	types.NetConf

	// support chaining for master interface and IP decisions
	// occurring prior to running ipvlan plugin
	RawPrevResult *map[string]interface{} `json:"prevResult"`
	PrevResult    *current.Result         `json:"-"`

	ConfDir string `json:"confDir"`
	CNIDir  string `json:"cniDir"`
	BinDir  string `json:"binDir"`
	// RawDelegates is private to the NetConf class; use Delegates instead
	RawDelegates    []map[string]interface{} `json:"delegates"`
	Delegates       []*DelegateNetConf       `json:"-"`
	NetStatus       []*NetworkStatus         `json:"-"`
	Kubeconfig      string                   `json:"kubeconfig"`
	ClusterNetwork  string                   `json:"clusterNetwork"`
	DefaultNetworks []string                 `json:"defaultNetworks"`
	LogFile         string                   `json:"logFile"`
	LogLevel        string                   `json:"logLevel"`
	RuntimeConfig   *RuntimeConfig           `json:"runtimeConfig,omitempty"`
	// Default network readiness options
	ReadinessIndicatorFile string `json:"readinessindicatorfile"`
	// Option to isolate the usage of CR's to the namespace in which a pod resides.
	NamespaceIsolation bool `json:"namespaceIsolation"`
	// Option to set system namespaces (to avoid to add defaultNetworks)
	SystemNamespaces []string `json:"systemNamespaces"`
	// Option to set the namespace that multus-cni uses (clusterNetwork/defaultNetworks)
	MultusNamespace string `json:"multusNamespace"`
}

// RuntimeConfig specifies CNI RuntimeConfig
type RuntimeConfig struct {
	PortMaps  []*PortMapEntry `json:"portMappings,omitempty"`
	Bandwidth *BandwidthEntry `json:"bandwidth,omitempty"`
	IPs       []string        `json:"ips,omitempty"`
	Mac       string          `json:"mac,omitempty"`
}

// PortMapEntry for CNI PortMapEntry
type PortMapEntry struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol,omitempty"`
	HostIP        string `json:"hostIP,omitempty"`
}

// BandwidthEntry for CNI BandwidthEntry
type BandwidthEntry struct {
	IngressRate  int `json:"ingressRate"`
	IngressBurst int `json:"ingressBurst"`

	EgressRate  int `json:"egressRate"`
	EgressBurst int `json:"egressBurst"`
}

// NetworkStatus is for network status annotation for pod
type NetworkStatus struct {
	Name      string    `json:"name"`
	Interface string    `json:"interface,omitempty"`
	IPs       []string  `json:"ips,omitempty"`
	Mac       string    `json:"mac,omitempty"`
	DNS       types.DNS `json:"dns,omitempty"`
	Gateway   []net.IP  `json:"default-route,omitempty"`
}

// DelegateNetConf for net-attach-def for pod
type DelegateNetConf struct {
	Conf                types.NetConf
	ConfList            types.NetConfList
	Name                string
	IfnameRequest       string          `json:"ifnameRequest,omitempty"`
	MacRequest          string          `json:"macRequest,omitempty"`
	IPRequest           []string        `json:"ipRequest,omitempty"`
	PortMappingsRequest []*PortMapEntry `json:"-"`
	BandwidthRequest    *BandwidthEntry `json:"-"`
	GatewayRequest      []net.IP        `json:"default-route,omitempty"`
	IsFilterGateway     bool
	// MasterPlugin is only used internal housekeeping
	MasterPlugin bool `json:"-"`
	// Conflist plugin is only used internal housekeeping
	ConfListPlugin bool `json:"-"`

	// Raw JSON
	Bytes []byte
}

// NetworkSelectionElement represents one element of the JSON format
// Network Attachment Selection Annotation as described in section 4.1.2
// of the CRD specification.
type NetworkSelectionElement struct {
	// Name contains the name of the Network object this element selects
	Name string `json:"name"`
	// Namespace contains the optional namespace that the network referenced
	// by Name exists in
	Namespace string `json:"namespace,omitempty"`
	// IPRequest contains an optional requested IP address for this network
	// attachment
	IPRequest []string `json:"ips,omitempty"`
	// MacRequest contains an optional requested MAC address for this
	// network attachment
	MacRequest string `json:"mac,omitempty"`
	// InterfaceRequest contains an optional requested name for the
	// network interface this attachment will create in the container
	InterfaceRequest string `json:"interface,omitempty"`
	// DeprecatedInterfaceRequest is obsolated parameter at pre 3.2.
	// This will be removed in 4.0 release.
	DeprecatedInterfaceRequest string `json:"interfaceRequest,omitempty"`
	// PortMappingsRequest contains an optional requested port mapping
	// for the network
	PortMappingsRequest []*PortMapEntry `json:"portMappings,omitempty"`
	// BandwidthRequest contains an optional requested bandwidth for
	// the network
	BandwidthRequest *BandwidthEntry `json:"bandwidth,omitempty"`
	// CNIArgs contains additional CNI arguments for the network interface
	CNIArgs *map[string]interface{} `json:"cni-args"`
	// GatewayRequest contains default route IP address for the pod
	GatewayRequest []net.IP `json:"default-route,omitempty"`
}

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

// ResourceInfo is struct to hold Pod device allocation information
type ResourceInfo struct {
	Index     int
	DeviceIDs []string
}

// ResourceClient provides a kubelet Pod resource handle
type ResourceClient interface {
	// GetPodResourceMap returns an instance of a map of Pod ResourceInfo given a (Pod name, namespace) tuple
	GetPodResourceMap(*v1.Pod) (map[string]*ResourceInfo, error)
}
