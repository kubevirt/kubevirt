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
 *
*/

package types

import (
	k8sv1 "k8s.io/api/core/v1"
	v1 "kubevirt.io/api/core/v1"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"

	ephemeraldisk "kubevirt.io/kubevirt/pkg/ephemeral-disk"
	"kubevirt.io/kubevirt/pkg/os/disk"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
)

type EFIConfiguration struct {
	EFICode      string
	EFIVars      string
	SecureLoader bool
}

type ConverterContext struct {
	Architecture                    arch.Converter
	AllowEmulation                  bool
	KvmAvailable                    bool
	Secrets                         map[string]*k8sv1.Secret
	VirtualMachine                  *v1.VirtualMachineInstance
	CPUSet                          []int
	IsBlockPVC                      map[string]bool
	IsBlockDV                       map[string]bool
	ApplyCBT                        map[string]string
	HotplugVolumes                  map[string]v1.VolumeStatus
	PermanentVolumes                map[string]v1.VolumeStatus
	MigratedVolumes                 map[string]string
	DisksInfo                       map[string]*disk.DiskInfo
	SMBios                          *cmdv1.SMBios
	SRIOVDevices                    []api.HostDevice
	GenericHostDevices              []api.HostDevice
	GPUHostDevices                  []api.HostDevice
	EFIConfiguration                *EFIConfiguration
	MemBalloonStatsPeriod           uint
	UseVirtioTransitional           bool
	EphemeraldiskCreator            ephemeraldisk.EphemeralDiskCreatorInterface
	VolumesDiscardIgnore            []string
	Topology                        *cmdv1.Topology
	UseLaunchSecuritySEV            bool // For AMD SEV/ES/SNP
	UseLaunchSecurityTDX            bool // For Intel TDX
	UseLaunchSecurityPV             bool // For IBM SE(s390-pv)
	FreePageReporting               bool
	BochsForEFIGuests               bool
	SerialConsoleLog                bool
	DomainAttachmentByInterfaceName map[string]string
}
