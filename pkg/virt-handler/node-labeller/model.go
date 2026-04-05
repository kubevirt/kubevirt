/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package nodelabeller

type cpuFeatures map[string]bool

type supportedFeatures struct {
	items []string
}

type supportedModels struct {
	usableModels []string
	knownModels  []string
}

type hostCPUModel struct {
	Name             string
	fallback         string
	requiredFeatures cpuFeatures
}

// HostDomCapabilities represents structure for parsing output of virsh capabilities
type HostDomCapabilities struct {
	CPU             CPU                          `xml:"cpu"`
	SEV             SEVConfiguration             `xml:"features>sev"`
	SecureExecution SecureExecutionConfiguration `xml:"features>s390-pv"`
	TDX             TDXConfiguration             `xml:"features>tdx"`
	LaunchSecurity  LaunchSecurityConfiguration  `xml:"features>launchSecurity"`
}

// CPU represents slice of cpu modes
type CPU struct {
	Mode []Mode `xml:"mode"`
}

// Mode represents slice of cpu models
type Mode struct {
	Name      string        `xml:"name,attr"`
	Supported string        `xml:"supported,attr"`
	Vendor    Vendor        `xml:"vendor"`
	Feature   []HostFeature `xml:"feature"`
	Model     []Model       `xml:"model"`
}

type SupportedHostFeature struct {
	Feature []HostFeature `xml:"feature"`
}

type HostFeature struct {
	Policy string `xml:"policy,attr"`
	Name   string `xml:"name,attr"`
}

// Vendor represents vendor of host CPU
type Vendor struct {
	Name string `xml:",chardata"`
}

// Model represents cpu model
type Model struct {
	Name     string `xml:",chardata"`
	Usable   string `xml:"usable,attr"`
	Fallback string `xml:"fallback,attr"`
}

// Structures needed to parse cpu features
type FeatureModel struct {
	Model Features `xml:"model"`
}

type Features struct {
	Features []Feature `xml:"feature"`
}

type Feature struct {
	Name string `xml:"name,attr"`
}

type SEVConfiguration struct {
	Supported       string `xml:"supported,attr"`
	CBitPos         uint   `xml:"cbitpos"`
	ReducedPhysBits uint   `xml:"reducedPhysBits"`
	MaxGuests       uint   `xml:"maxGuests"`
	MaxESGuests     uint   `xml:"maxESGuests"`
	SupportedES     string `xml:"-"`
	SupportedSNP    string `xml:"-"`
}
type SecureExecutionConfiguration struct {
	Supported string `xml:"supported,attr"`
}

type TDXConfiguration struct {
	Supported string `xml:"supported,attr"`
}

type LaunchSecurityConfiguration struct {
	Supported string      `xml:"supported,attr"`
	SecTypes  SecTypeEnum `xml:"enum"`
}

type SecTypeEnum struct {
	Name   string   `xml:"name,attr"`
	Values []string `xml:"value"`
}
