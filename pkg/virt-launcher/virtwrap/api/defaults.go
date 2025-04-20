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
 * Copyright The KubeVirt Authors.
 */

package api

const (
	DefaultProtocol   = "TCP"
	DefaultVMCIDR     = "10.0.2.0/24"
	DefaultVMIpv6CIDR = "fd10:0:2::/120"
	DefaultBridgeName = "k6t-eth0"
)

func NewDefaulter(arch string) *Defaulter {
	return &Defaulter{Architecture: arch}
}

type Defaulter struct {
	Architecture string
}

func (d *Defaulter) isPPC64() bool {
	return d.Architecture == "ppc64le"
}

func (d *Defaulter) isARM64() bool {
	return d.Architecture == "arm64"
}

func (d *Defaulter) isS390X() bool {
	return d.Architecture == "s390x"
}

func (d *Defaulter) setDefaults_OSType(ostype *OSType) {
	ostype.OS = "hvm"

	if ostype.Arch == "" {
		switch {
		case d.isPPC64():
			ostype.Arch = "ppc64le"
		case d.isARM64():
			ostype.Arch = "aarch64"
		case d.isS390X():
			ostype.Arch = "s390x"
		default:
			ostype.Arch = "x86_64"
		}
	}

	// q35 is an alias of the newest q35 machine type.
	// TODO: we probably want to select concrete type in the future for "future-backwards" compatibility.
	if ostype.Machine == "" {
		switch {
		case d.isPPC64():
			ostype.Machine = "pseries"
		case d.isARM64():
			ostype.Machine = "virt"
		case d.isS390X():
			ostype.Machine = "s390-ccw-virtio"
		default:
			ostype.Machine = "q35"
		}
	}
}

func (d *Defaulter) setDefaults_DomainSpec(spec *DomainSpec) {
	spec.XmlNS = "http://libvirt.org/schemas/domain/qemu/1.0"
	if spec.Type == "" {
		spec.Type = "kvm"
	}
}

func (d *Defaulter) setDefaults_SysInfo(sysinfo *SysInfo) {
	if sysinfo.Type == "" {
		sysinfo.Type = "smbios"
	}
}

func (d *Defaulter) setDefaults_Features(spec *DomainSpec) {
	if spec.Features == nil {
		spec.Features = &Features{}
	}
}

func (d *Defaulter) SetObjectDefaults_Domain(in *Domain) {
	d.setDefaults_DomainSpec(&in.Spec)
	d.setDefaults_OSType(&in.Spec.OS.Type)
	if in.Spec.SysInfo != nil {
		d.setDefaults_SysInfo(in.Spec.SysInfo)
	}
	d.setDefaults_Features(&in.Spec)
}
