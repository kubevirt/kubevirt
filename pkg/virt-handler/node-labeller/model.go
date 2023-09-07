/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2021 Red Hat, Inc.
 */

package nodelabeller

type cpuFeatures map[string]bool

type supportedFeatures struct {
	items []string
}

type hostCPUModel struct {
	Name             string
	fallback         string
	requiredFeatures cpuFeatures
}

// hostCapabilities holds informations which provides libvirt,
// so we don't have to call libvirt at every request
type cpuInfo struct {
	usableModels map[string]cpuFeatures
}

// HostDomCapabilities represents structure for parsing output of virsh capabilities
type HostDomCapabilities struct {
	CPU CPU              `xml:"cpu"`
	SEV SEVConfiguration `xml:"features>sev"`
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
}
