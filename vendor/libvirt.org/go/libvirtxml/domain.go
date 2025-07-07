/*
 * This file is part of the libvirt-go-xml-module project
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * Copyright (C) 2016 Red Hat, Inc.
 *
 */

package libvirtxml

import (
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type DomainControllerPCIHole64 struct {
	Size uint64 `xml:",chardata"`
	Unit string `xml:"unit,attr,omitempty"`
}

type DomainControllerPCIModel struct {
	Name string `xml:"name,attr"`
}

type DomainControllerPCITarget struct {
	ChassisNr  *uint
	Chassis    *uint
	Port       *uint
	BusNr      *uint
	Index      *uint
	NUMANode   *uint
	Hotplug    string
	MemReserve *uint64
}

type DomainControllerPCI struct {
	Model  *DomainControllerPCIModel  `xml:"model"`
	Target *DomainControllerPCITarget `xml:"target"`
	Hole64 *DomainControllerPCIHole64 `xml:"pcihole64"`
}

type DomainControllerUSBMaster struct {
	StartPort uint `xml:"startport,attr"`
}

type DomainControllerUSB struct {
	Port   *uint                      `xml:"ports,attr"`
	Master *DomainControllerUSBMaster `xml:"master"`
}

type DomainControllerVirtIOSerial struct {
	Ports   *uint `xml:"ports,attr"`
	Vectors *uint `xml:"vectors,attr"`
}

type DomainControllerXenBus struct {
	MaxGrantFrames   uint `xml:"maxGrantFrames,attr,omitempty"`
	MaxEventChannels uint `xml:"maxEventChannels,attr,omitempty"`
}

type DomainControllerDriver struct {
	Queues     *uint  `xml:"queues,attr"`
	CmdPerLUN  *uint  `xml:"cmd_per_lun,attr"`
	MaxSectors *uint  `xml:"max_sectors,attr"`
	IOEventFD  string `xml:"ioeventfd,attr,omitempty"`
	IOThread   uint   `xml:"iothread,attr,omitempty"`
	IOMMU      string `xml:"iommu,attr,omitempty"`
	ATS        string `xml:"ats,attr,omitempty"`
	Packed     string `xml:"packed,attr,omitempty"`
	PagePerVQ  string `xml:"page_per_vq,attr,omitempty"`
}

type DomainController struct {
	XMLName      xml.Name                      `xml:"controller"`
	Type         string                        `xml:"type,attr"`
	Index        *uint                         `xml:"index,attr"`
	Model        string                        `xml:"model,attr,omitempty"`
	Driver       *DomainControllerDriver       `xml:"driver"`
	PCI          *DomainControllerPCI          `xml:"-"`
	USB          *DomainControllerUSB          `xml:"-"`
	VirtIOSerial *DomainControllerVirtIOSerial `xml:"-"`
	XenBus       *DomainControllerXenBus       `xml:"-"`
	ACPI         *DomainDeviceACPI             `xml:"acpi"`
	Alias        *DomainAlias                  `xml:"alias"`
	Address      *DomainAddress                `xml:"address"`
}

type DomainDiskSecret struct {
	Type  string `xml:"type,attr,omitempty"`
	Usage string `xml:"usage,attr,omitempty"`
	UUID  string `xml:"uuid,attr,omitempty"`
}

type DomainDiskAuth struct {
	Username string            `xml:"username,attr,omitempty"`
	Secret   *DomainDiskSecret `xml:"secret"`
}

type DomainDiskSourceHost struct {
	Transport string `xml:"transport,attr,omitempty"`
	Name      string `xml:"name,attr,omitempty"`
	Port      string `xml:"port,attr,omitempty"`
	Socket    string `xml:"socket,attr,omitempty"`
}

type DomainDiskSourceSSL struct {
	Verify string `xml:"verify,attr"`
}

type DomainDiskCookie struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type DomainDiskCookies struct {
	Cookies []DomainDiskCookie `xml:"cookie"`
}

type DomainDiskSourceReadahead struct {
	Size string `xml:"size,attr"`
}

type DomainDiskSourceTimeout struct {
	Seconds string `xml:"seconds,attr"`
}

type DomainDiskReservationsSource DomainChardevSource

type DomainDiskReservations struct {
	Enabled string                        `xml:"enabled,attr,omitempty"`
	Managed string                        `xml:"managed,attr,omitempty"`
	Source  *DomainDiskReservationsSource `xml:"source"`
}

type DomainDiskSource struct {
	File          *DomainDiskSourceFile      `xml:"-"`
	Block         *DomainDiskSourceBlock     `xml:"-"`
	Dir           *DomainDiskSourceDir       `xml:"-"`
	Network       *DomainDiskSourceNetwork   `xml:"-"`
	Volume        *DomainDiskSourceVolume    `xml:"-"`
	NVME          *DomainDiskSourceNVME      `xml:"-"`
	VHostUser     *DomainDiskSourceVHostUser `xml:"-"`
	VHostVDPA     *DomainDiskSourceVHostVDPA `xml:"-"`
	StartupPolicy string                     `xml:"startupPolicy,attr,omitempty"`
	Index         uint                       `xml:"index,attr,omitempty"`
	Encryption    *DomainDiskEncryption      `xml:"encryption"`
	Reservations  *DomainDiskReservations    `xml:"reservations"`
	Slices        *DomainDiskSlices          `xml:"slices"`
	SSL           *DomainDiskSourceSSL       `xml:"ssl"`
	Cookies       *DomainDiskCookies         `xml:"cookies"`
	Readahead     *DomainDiskSourceReadahead `xml:"readahead"`
	Timeout       *DomainDiskSourceTimeout   `xml:"timeout"`
	DataStore     *DomainDiskDataStore       `xml:"dataStore"`
}

type DomainDiskDataStore struct {
	Format *DomainDiskFormat `xml:"format"`
	Source *DomainDiskSource `xml:"source"`
}

type DomainDiskSlices struct {
	Slices []DomainDiskSlice `xml:"slice"`
}

type DomainDiskSlice struct {
	Type   string `xml:"type,attr"`
	Offset uint   `xml:"offset,attr"`
	Size   uint   `xml:"size,attr"`
}

type DomainDiskSourceFile struct {
	File     string                 `xml:"file,attr,omitempty"`
	FDGroup  string                 `xml:"fdgroup,attr,omitempty"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainDiskSourceNVME struct {
	PCI *DomainDiskSourceNVMEPCI
}

type DomainDiskSourceNVMEPCI struct {
	Managed   string            `xml:"managed,attr,omitempty"`
	Namespace uint64            `xml:"namespace,attr,omitempty"`
	Address   *DomainAddressPCI `xml:"address"`
}

type DomainDiskSourceBlock struct {
	Dev      string                 `xml:"dev,attr,omitempty"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainDiskSourceDir struct {
	Dir string `xml:"dir,attr,omitempty"`
}

type DomainDiskSourceNetwork struct {
	Protocol    string                             `xml:"protocol,attr,omitempty"`
	Name        string                             `xml:"name,attr,omitempty"`
	Query       string                             `xml:"query,attr,omitempty"`
	TLS         string                             `xml:"tls,attr,omitempty"`
	TLSHostname string                             `xml:"tlsHostname,attr,omitempty"`
	Hosts       []DomainDiskSourceHost             `xml:"host"`
	Identity    *DomainDiskSourceNetworkIdentity   `xml:"identity"`
	KnownHosts  *DomainDiskSourceNetworkKnownHosts `xml:"knownHosts"`
	Initiator   *DomainDiskSourceNetworkInitiator  `xml:"initiator"`
	Snapshot    *DomainDiskSourceNetworkSnapshot   `xml:"snapshot"`
	Config      *DomainDiskSourceNetworkConfig     `xml:"config"`
	Reconnect   *DomainDiskSourceNetworkReconnect  `xml:"reconnect"`
	Auth        *DomainDiskAuth                    `xml:"auth"`
}

type DomainDiskSourceNetworkKnownHosts struct {
	Path string `xml:"path,attr"`
}

type DomainDiskSourceNetworkIdentity struct {
	User      string `xml:"user,attr,omitempty"`
	Group     string `xml:"group,attr,omitempty"`
	UserName  string `xml:"username,attr,omitempty"`
	Keyfile   string `xml:"keyfile,attr,omitempty"`
	AgentSock string `xml:"agentsock,attr,omitempty"`
}

type DomainDiskSourceNetworkInitiator struct {
	IQN *DomainDiskSourceNetworkIQN `xml:"iqn"`
}

type DomainDiskSourceNetworkIQN struct {
	Name string `xml:"name,attr,omitempty"`
}

type DomainDiskSourceNetworkSnapshot struct {
	Name string `xml:"name,attr"`
}

type DomainDiskSourceNetworkConfig struct {
	File string `xml:"file,attr"`
}

type DomainDiskSourceNetworkReconnect struct {
	Delay string `xml:"delay,attr"`
}

type DomainDiskSourceVolume struct {
	Pool     string                 `xml:"pool,attr,omitempty"`
	Volume   string                 `xml:"volume,attr,omitempty"`
	Mode     string                 `xml:"mode,attr,omitempty"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainDiskSourceVHostUser DomainChardevSource

type DomainDiskSourceVHostVDPA struct {
	Dev string `xml:"dev,attr"`
}

type DomainDiskMetadataCache struct {
	MaxSize *DomainDiskMetadataCacheSize `xml:"max_size"`
}

type DomainDiskMetadataCacheSize struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value int    `xml:",cdata"`
}

type DomainDiskIOThreads struct {
	IOThread []DomainDiskIOThread `xml:"iothread"`
}

type DomainDiskIOThread struct {
	ID     uint                      `xml:"id,attr"`
	Queues []DomainDiskIOThreadQueue `xml:"queue"`
}

type DomainDiskIOThreadQueue struct {
	ID uint `xml:"id,attr"`
}

type DomainDiskDriver struct {
	Name           string                   `xml:"name,attr,omitempty"`
	Type           string                   `xml:"type,attr,omitempty"`
	Cache          string                   `xml:"cache,attr,omitempty"`
	ErrorPolicy    string                   `xml:"error_policy,attr,omitempty"`
	RErrorPolicy   string                   `xml:"rerror_policy,attr,omitempty"`
	IO             string                   `xml:"io,attr,omitempty"`
	IOEventFD      string                   `xml:"ioeventfd,attr,omitempty"`
	EventIDX       string                   `xml:"event_idx,attr,omitempty"`
	CopyOnRead     string                   `xml:"copy_on_read,attr,omitempty"`
	Discard        string                   `xml:"discard,attr,omitempty"`
	DiscardNoUnref string                   `xml:"discard_no_unref,attr,omitempty"`
	IOThread       *uint                    `xml:"iothread,attr"`
	IOThreads      *DomainDiskIOThreads     `xml:"iothreads"`
	DetectZeros    string                   `xml:"detect_zeroes,attr,omitempty"`
	Queues         *uint                    `xml:"queues,attr"`
	QueueSize      *uint                    `xml:"queue_size,attr"`
	IOMMU          string                   `xml:"iommu,attr,omitempty"`
	ATS            string                   `xml:"ats,attr,omitempty"`
	Packed         string                   `xml:"packed,attr,omitempty"`
	PagePerVQ      string                   `xml:"page_per_vq,attr,omitempty"`
	MetadataCache  *DomainDiskMetadataCache `xml:"metadata_cache"`
}

type DomainDiskTarget struct {
	Dev          string `xml:"dev,attr,omitempty"`
	Bus          string `xml:"bus,attr,omitempty"`
	Tray         string `xml:"tray,attr,omitempty"`
	Removable    string `xml:"removable,attr,omitempty"`
	RotationRate uint   `xml:"rotation_rate,attr,omitempty"`
}

type DomainDiskEncryption struct {
	Format  string             `xml:"format,attr,omitempty"`
	Engine  string             `xml:"engine,attr,omitempty"`
	Secrets []DomainDiskSecret `xml:"secret"`
}

type DomainDiskReadOnly struct {
}

type DomainDiskShareable struct {
}

type DomainDiskTransient struct {
	ShareBacking string `xml:"shareBacking,attr,omitempty"`
}

type DomainDiskIOTune struct {
	TotalBytesSec          uint64 `xml:"total_bytes_sec,omitempty"`
	ReadBytesSec           uint64 `xml:"read_bytes_sec,omitempty"`
	WriteBytesSec          uint64 `xml:"write_bytes_sec,omitempty"`
	TotalIopsSec           uint64 `xml:"total_iops_sec,omitempty"`
	ReadIopsSec            uint64 `xml:"read_iops_sec,omitempty"`
	WriteIopsSec           uint64 `xml:"write_iops_sec,omitempty"`
	TotalBytesSecMax       uint64 `xml:"total_bytes_sec_max,omitempty"`
	ReadBytesSecMax        uint64 `xml:"read_bytes_sec_max,omitempty"`
	WriteBytesSecMax       uint64 `xml:"write_bytes_sec_max,omitempty"`
	TotalIopsSecMax        uint64 `xml:"total_iops_sec_max,omitempty"`
	ReadIopsSecMax         uint64 `xml:"read_iops_sec_max,omitempty"`
	WriteIopsSecMax        uint64 `xml:"write_iops_sec_max,omitempty"`
	TotalBytesSecMaxLength uint64 `xml:"total_bytes_sec_max_length,omitempty"`
	ReadBytesSecMaxLength  uint64 `xml:"read_bytes_sec_max_length,omitempty"`
	WriteBytesSecMaxLength uint64 `xml:"write_bytes_sec_max_length,omitempty"`
	TotalIopsSecMaxLength  uint64 `xml:"total_iops_sec_max_length,omitempty"`
	ReadIopsSecMaxLength   uint64 `xml:"read_iops_sec_max_length,omitempty"`
	WriteIopsSecMaxLength  uint64 `xml:"write_iops_sec_max_length,omitempty"`
	SizeIopsSec            uint64 `xml:"size_iops_sec,omitempty"`
	GroupName              string `xml:"group_name,omitempty"`
}

type DomainDiskGeometry struct {
	Cylinders uint   `xml:"cyls,attr"`
	Headers   uint   `xml:"heads,attr"`
	Sectors   uint   `xml:"secs,attr"`
	Trans     string `xml:"trans,attr,omitempty"`
}

type DomainDiskBlockIO struct {
	LogicalBlockSize   uint `xml:"logical_block_size,attr,omitempty"`
	PhysicalBlockSize  uint `xml:"physical_block_size,attr,omitempty"`
	DiscardGranularity uint `xml:"discard_granularity,attr,omitempty"`
}

type DomainDiskFormat struct {
	Type          string                   `xml:"type,attr"`
	MetadataCache *DomainDiskMetadataCache `xml:"metadata_cache"`
}

type DomainDiskBackingStore struct {
	Index        uint                    `xml:"index,attr,omitempty"`
	Format       *DomainDiskFormat       `xml:"format"`
	Source       *DomainDiskSource       `xml:"source"`
	BackingStore *DomainDiskBackingStore `xml:"backingStore"`
}

type DomainDiskMirror struct {
	Job          string                  `xml:"job,attr,omitempty"`
	Ready        string                  `xml:"ready,attr,omitempty"`
	Format       *DomainDiskFormat       `xml:"format"`
	Source       *DomainDiskSource       `xml:"source"`
	BackingStore *DomainDiskBackingStore `xml:"backingStore"`
}

type DomainBackendDomain struct {
	Name string `xml:"name,attr"`
}

type DomainDisk struct {
	XMLName       xml.Name                `xml:"disk"`
	Device        string                  `xml:"device,attr,omitempty"`
	RawIO         string                  `xml:"rawio,attr,omitempty"`
	SGIO          string                  `xml:"sgio,attr,omitempty"`
	Snapshot      string                  `xml:"snapshot,attr,omitempty"`
	Model         string                  `xml:"model,attr,omitempty"`
	Driver        *DomainDiskDriver       `xml:"driver"`
	Auth          *DomainDiskAuth         `xml:"auth"`
	Source        *DomainDiskSource       `xml:"source"`
	BackingStore  *DomainDiskBackingStore `xml:"backingStore"`
	BackendDomain *DomainBackendDomain    `xml:"backenddomain"`
	Geometry      *DomainDiskGeometry     `xml:"geometry"`
	BlockIO       *DomainDiskBlockIO      `xml:"blockio"`
	Mirror        *DomainDiskMirror       `xml:"mirror"`
	Target        *DomainDiskTarget       `xml:"target"`
	IOTune        *DomainDiskIOTune       `xml:"iotune"`
	ReadOnly      *DomainDiskReadOnly     `xml:"readonly"`
	Shareable     *DomainDiskShareable    `xml:"shareable"`
	Transient     *DomainDiskTransient    `xml:"transient"`
	Serial        string                  `xml:"serial,omitempty"`
	WWN           string                  `xml:"wwn,omitempty"`
	Vendor        string                  `xml:"vendor,omitempty"`
	Product       string                  `xml:"product,omitempty"`
	Encryption    *DomainDiskEncryption   `xml:"encryption"`
	Boot          *DomainDeviceBoot       `xml:"boot"`
	ACPI          *DomainDeviceACPI       `xml:"acpi"`
	Alias         *DomainAlias            `xml:"alias"`
	Address       *DomainAddress          `xml:"address"`
}

type DomainFilesystemDriver struct {
	Type      string `xml:"type,attr,omitempty"`
	Format    string `xml:"format,attr,omitempty"`
	Name      string `xml:"name,attr,omitempty"`
	WRPolicy  string `xml:"wrpolicy,attr,omitempty"`
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
	Queue     uint   `xml:"queue,attr,omitempty"`
}

type DomainFilesystemSource struct {
	Mount    *DomainFilesystemSourceMount    `xml:"-"`
	Block    *DomainFilesystemSourceBlock    `xml:"-"`
	File     *DomainFilesystemSourceFile     `xml:"-"`
	Template *DomainFilesystemSourceTemplate `xml:"-"`
	RAM      *DomainFilesystemSourceRAM      `xml:"-"`
	Bind     *DomainFilesystemSourceBind     `xml:"-"`
	Volume   *DomainFilesystemSourceVolume   `xml:"-"`
}

type DomainFilesystemSourceMount struct {
	Dir    string `xml:"dir,attr,omitempty"`
	Socket string `xml:"socket,attr,omitempty"`
}

type DomainFilesystemSourceBlock struct {
	Dev string `xml:"dev,attr"`
}

type DomainFilesystemSourceFile struct {
	File string `xml:"file,attr"`
}

type DomainFilesystemSourceTemplate struct {
	Name string `xml:"name,attr"`
}

type DomainFilesystemSourceRAM struct {
	Usage uint   `xml:"usage,attr"`
	Units string `xml:"units,attr,omitempty"`
}

type DomainFilesystemSourceBind struct {
	Dir string `xml:"dir,attr"`
}

type DomainFilesystemSourceVolume struct {
	Pool   string `xml:"pool,attr"`
	Volume string `xml:"volume,attr"`
}

type DomainFilesystemTarget struct {
	Dir string `xml:"dir,attr"`
}

type DomainFilesystemReadOnly struct {
}

type DomainFilesystemSpaceHardLimit struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainFilesystemSpaceSoftLimit struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainFilesystemBinaryCache struct {
	Mode string `xml:"mode,attr"`
}

type DomainFilesystemBinarySandbox struct {
	Mode string `xml:"mode,attr"`
}

type DomainFilesystemBinaryLock struct {
	POSIX string `xml:"posix,attr,omitempty"`
	Flock string `xml:"flock,attr,omitempty"`
}

type DomainFilesystemBinaryThreadPool struct {
	Size uint `xml:"size,attr,omitempty"`
}

type DomainFilesystemBinaryOpenFiles struct {
	Max uint `xml:"max,attr,"`
}

type DomainFilesystemBinary struct {
	Path       string                            `xml:"path,attr,omitempty"`
	XAttr      string                            `xml:"xattr,attr,omitempty"`
	Cache      *DomainFilesystemBinaryCache      `xml:"cache"`
	Sandbox    *DomainFilesystemBinarySandbox    `xml:"sandbox"`
	Lock       *DomainFilesystemBinaryLock       `xml:"lock"`
	ThreadPool *DomainFilesystemBinaryThreadPool `xml:"thread_pool"`
	OpenFiles  *DomainFilesystemBinaryOpenFiles  `xml:"openfiles"`
}

type DomainFilesystemIDMapEntry struct {
	Start  uint `xml:"start,attr"`
	Target uint `xml:"target,attr"`
	Count  uint `xml:"count,attr"`
}

type DomainFilesystemIDMap struct {
	UID []DomainFilesystemIDMapEntry `xml:"uid"`
	GID []DomainFilesystemIDMapEntry `xml:"gid"`
}

type DomainFilesystem struct {
	XMLName        xml.Name                        `xml:"filesystem"`
	AccessMode     string                          `xml:"accessmode,attr,omitempty"`
	Model          string                          `xml:"model,attr,omitempty"`
	MultiDevs      string                          `xml:"multidevs,attr,omitempty"`
	FMode          string                          `xml:"fmode,attr,omitempty"`
	DMode          string                          `xml:"dmode,attr,omitempty"`
	Driver         *DomainFilesystemDriver         `xml:"driver"`
	Binary         *DomainFilesystemBinary         `xml:"binary"`
	IDMap          *DomainFilesystemIDMap          `xml:"idmap"`
	Source         *DomainFilesystemSource         `xml:"source"`
	Target         *DomainFilesystemTarget         `xml:"target"`
	ReadOnly       *DomainFilesystemReadOnly       `xml:"readonly"`
	SpaceHardLimit *DomainFilesystemSpaceHardLimit `xml:"space_hard_limit"`
	SpaceSoftLimit *DomainFilesystemSpaceSoftLimit `xml:"space_soft_limit"`
	Boot           *DomainDeviceBoot               `xml:"boot"`
	ACPI           *DomainDeviceACPI               `xml:"acpi"`
	Alias          *DomainAlias                    `xml:"alias"`
	Address        *DomainAddress                  `xml:"address"`
}

type DomainInterfaceMAC struct {
	Address string `xml:"address,attr"`
	Type    string `xml:"type,attr,omitempty"`
	Check   string `xml:"check,attr,omitempty"`
}

type DomainInterfaceModel struct {
	Type string `xml:"type,attr"`
}

type DomainInterfaceSource struct {
	User      *DomainInterfaceSourceUser     `xml:"-"`
	Ethernet  *DomainInterfaceSourceEthernet `xml:"-"`
	VHostUser *DomainChardevSource           `xml:"-"`
	Server    *DomainInterfaceSourceServer   `xml:"-"`
	Client    *DomainInterfaceSourceClient   `xml:"-"`
	MCast     *DomainInterfaceSourceMCast    `xml:"-"`
	Network   *DomainInterfaceSourceNetwork  `xml:"-"`
	Bridge    *DomainInterfaceSourceBridge   `xml:"-"`
	Internal  *DomainInterfaceSourceInternal `xml:"-"`
	Direct    *DomainInterfaceSourceDirect   `xml:"-"`
	Hostdev   *DomainInterfaceSourceHostdev  `xml:"-"`
	UDP       *DomainInterfaceSourceUDP      `xml:"-"`
	VDPA      *DomainInterfaceSourceVDPA     `xml:"-"`
	Null      *DomainInterfaceSourceNull     `xml:"-"`
	VDS       *DomainInterfaceSourceVDS      `xml:"-"`
}

type DomainInterfaceSourceUser struct {
	Dev string `xml:"dev,attr,omitempty"`
}

type DomainInterfaceSourcePortForward struct {
	Proto   string                                  `xml:"proto,attr"`
	Address string                                  `xml:"address,attr,omitempty"`
	Dev     string                                  `xml:"dev,attr,omitempty"`
	Ranges  []DomainInterfaceSourcePortForwardRange `xml:"range"`
}

type DomainInterfaceSourcePortForwardRange struct {
	Start   uint   `xml:"start,attr"`
	End     uint   `xml:"end,attr,omitempty"`
	To      uint   `xml:"to,attr,omitempty"`
	Exclude string `xml:"exclude,attr,omitempty"`
}

type DomainInterfaceSourceEthernet struct {
	IP    []DomainInterfaceIP    `xml:"ip"`
	Route []DomainInterfaceRoute `xml:"route"`
}

type DomainInterfaceSourceServer struct {
	Address string                      `xml:"address,attr,omitempty"`
	Port    uint                        `xml:"port,attr,omitempty"`
	Local   *DomainInterfaceSourceLocal `xml:"local"`
}

type DomainInterfaceSourceClient struct {
	Address string                      `xml:"address,attr,omitempty"`
	Port    uint                        `xml:"port,attr,omitempty"`
	Local   *DomainInterfaceSourceLocal `xml:"local"`
}

type DomainInterfaceSourceMCast struct {
	Address string                      `xml:"address,attr,omitempty"`
	Port    uint                        `xml:"port,attr,omitempty"`
	Local   *DomainInterfaceSourceLocal `xml:"local"`
}

type DomainInterfaceSourceNetwork struct {
	Network   string `xml:"network,attr,omitempty"`
	PortGroup string `xml:"portgroup,attr,omitempty"`
	Bridge    string `xml:"bridge,attr,omitempty"`
	PortID    string `xml:"portid,attr,omitempty"`
}

type DomainInterfaceSourceBridge struct {
	Bridge string `xml:"bridge,attr"`
}

type DomainInterfaceSourceInternal struct {
	Name string `xml:"name,attr,omitempty"`
}

type DomainInterfaceSourceDirect struct {
	Dev  string `xml:"dev,attr,omitempty"`
	Mode string `xml:"mode,attr,omitempty"`
}

type DomainInterfaceSourceHostdev struct {
	PCI *DomainHostdevSubsysPCISource `xml:"-"`
	USB *DomainHostdevSubsysUSBSource `xml:"-"`
}

type DomainInterfaceSourceUDP struct {
	Address string                      `xml:"address,attr,omitempty"`
	Port    uint                        `xml:"port,attr,omitempty"`
	Local   *DomainInterfaceSourceLocal `xml:"local"`
}

type DomainInterfaceSourceVDPA struct {
	Device string `xml:"dev,attr,omitempty"`
}

type DomainInterfaceSourceNull struct {
}

type DomainInterfaceSourceVDS struct {
	SwitchID     string `xml:"switchid,attr"`
	PortID       int    `xml:"portid,attr,omitempty"`
	PortGroupID  string `xml:"portgroupid,attr,omitempty"`
	ConnectionID int    `xml:"connectionid,attr,omitempty"`
}

type DomainInterfaceSourceLocal struct {
	Address string `xml:"address,attr,omitempty"`
	Port    uint   `xml:"port,attr,omitempty"`
}

type DomainInterfaceTarget struct {
	Dev     string `xml:"dev,attr"`
	Managed string `xml:"managed,attr,omitempty"`
}

type DomainInterfaceLink struct {
	State string `xml:"state,attr"`
}

type DomainDeviceBoot struct {
	Order    uint   `xml:"order,attr"`
	LoadParm string `xml:"loadparm,attr,omitempty"`
}

type DomainInterfaceScript struct {
	Path string `xml:"path,attr"`
}

type DomainInterfaceDriver struct {
	Name          string                      `xml:"name,attr,omitempty"`
	TXMode        string                      `xml:"txmode,attr,omitempty"`
	IOEventFD     string                      `xml:"ioeventfd,attr,omitempty"`
	EventIDX      string                      `xml:"event_idx,attr,omitempty"`
	Queues        uint                        `xml:"queues,attr,omitempty"`
	RXQueueSize   uint                        `xml:"rx_queue_size,attr,omitempty"`
	TXQueueSize   uint                        `xml:"tx_queue_size,attr,omitempty"`
	IOMMU         string                      `xml:"iommu,attr,omitempty"`
	ATS           string                      `xml:"ats,attr,omitempty"`
	Packed        string                      `xml:"packed,attr,omitempty"`
	PagePerVQ     string                      `xml:"page_per_vq,attr,omitempty"`
	RSS           string                      `xml:"rss,attr,omitempty"`
	RSSHashReport string                      `xml:"rss_hash_report,attr,omitempty"`
	Host          *DomainInterfaceDriverHost  `xml:"host"`
	Guest         *DomainInterfaceDriverGuest `xml:"guest"`
}

type DomainInterfaceDriverHost struct {
	CSum     string `xml:"csum,attr,omitempty"`
	GSO      string `xml:"gso,attr,omitempty"`
	TSO4     string `xml:"tso4,attr,omitempty"`
	TSO6     string `xml:"tso6,attr,omitempty"`
	ECN      string `xml:"ecn,attr,omitempty"`
	UFO      string `xml:"ufo,attr,omitempty"`
	MrgRXBuf string `xml:"mrg_rxbuf,attr,omitempty"`
}

type DomainInterfaceDriverGuest struct {
	CSum string `xml:"csum,attr,omitempty"`
	TSO4 string `xml:"tso4,attr,omitempty"`
	TSO6 string `xml:"tso6,attr,omitempty"`
	ECN  string `xml:"ecn,attr,omitempty"`
	UFO  string `xml:"ufo,attr,omitempty"`
}

type DomainInterfaceVirtualPort struct {
	Params *DomainInterfaceVirtualPortParams `xml:"parameters"`
}

type DomainInterfaceVirtualPortParams struct {
	Any          *DomainInterfaceVirtualPortParamsAny          `xml:"-"`
	VEPA8021QBG  *DomainInterfaceVirtualPortParamsVEPA8021QBG  `xml:"-"`
	VNTag8011QBH *DomainInterfaceVirtualPortParamsVNTag8021QBH `xml:"-"`
	OpenVSwitch  *DomainInterfaceVirtualPortParamsOpenVSwitch  `xml:"-"`
	MidoNet      *DomainInterfaceVirtualPortParamsMidoNet      `xml:"-"`
}

type DomainInterfaceVirtualPortParamsAny struct {
	ManagerID     *uint  `xml:"managerid,attr"`
	TypeID        *uint  `xml:"typeid,attr"`
	TypeIDVersion *uint  `xml:"typeidversion,attr"`
	InstanceID    string `xml:"instanceid,attr,omitempty"`
	ProfileID     string `xml:"profileid,attr,omitempty"`
	InterfaceID   string `xml:"interfaceid,attr,omitempty"`
}

type DomainInterfaceVirtualPortParamsVEPA8021QBG struct {
	ManagerID     *uint  `xml:"managerid,attr"`
	TypeID        *uint  `xml:"typeid,attr"`
	TypeIDVersion *uint  `xml:"typeidversion,attr"`
	InstanceID    string `xml:"instanceid,attr,omitempty"`
}

type DomainInterfaceVirtualPortParamsVNTag8021QBH struct {
	ProfileID string `xml:"profileid,attr,omitempty"`
}

type DomainInterfaceVirtualPortParamsOpenVSwitch struct {
	InterfaceID string `xml:"interfaceid,attr,omitempty"`
	ProfileID   string `xml:"profileid,attr,omitempty"`
}

type DomainInterfaceVirtualPortParamsMidoNet struct {
	InterfaceID string `xml:"interfaceid,attr,omitempty"`
}

type DomainInterfaceBandwidthParams struct {
	Average *int `xml:"average,attr"`
	Peak    *int `xml:"peak,attr"`
	Burst   *int `xml:"burst,attr"`
	Floor   *int `xml:"floor,attr"`
}

type DomainInterfaceBandwidth struct {
	Inbound  *DomainInterfaceBandwidthParams `xml:"inbound"`
	Outbound *DomainInterfaceBandwidthParams `xml:"outbound"`
}

type DomainInterfaceVLan struct {
	Trunk string                   `xml:"trunk,attr,omitempty"`
	Tags  []DomainInterfaceVLanTag `xml:"tag"`
}

type DomainInterfaceVLanTag struct {
	ID         uint   `xml:"id,attr"`
	NativeMode string `xml:"nativeMode,attr,omitempty"`
}

type DomainInterfaceGuest struct {
	Dev    string `xml:"dev,attr,omitempty"`
	Actual string `xml:"actual,attr,omitempty"`
}

type DomainInterfaceFilterRef struct {
	Filter     string                       `xml:"filter,attr"`
	Parameters []DomainInterfaceFilterParam `xml:"parameter"`
}

type DomainInterfaceFilterParam struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

type DomainInterfaceBackend struct {
	Type    string `xml:"type,attr,omitempty"`
	Tap     string `xml:"tap,attr,omitempty"`
	VHost   string `xml:"vhost,attr,omitempty"`
	LogFile string `xml:"logFile,attr,omitempty"`
}

type DomainInterfaceTune struct {
	SndBuf uint `xml:"sndbuf"`
}

type DomainInterfaceMTU struct {
	Size uint `xml:"size,attr"`
}

type DomainInterfaceCoalesce struct {
	RX *DomainInterfaceCoalesceRX `xml:"rx"`
}

type DomainInterfaceCoalesceRX struct {
	Frames *DomainInterfaceCoalesceRXFrames `xml:"frames"`
}

type DomainInterfaceCoalesceRXFrames struct {
	Max *uint `xml:"max,attr"`
}

type DomainROM struct {
	Bar     string  `xml:"bar,attr,omitempty"`
	File    *string `xml:"file,attr"`
	Enabled string  `xml:"enabled,attr,omitempty"`
}

type DomainInterfaceIP struct {
	Address string `xml:"address,attr"`
	Family  string `xml:"family,attr,omitempty"`
	Prefix  uint   `xml:"prefix,attr,omitempty"`
	Peer    string `xml:"peer,attr,omitempty"`
}

type DomainInterfaceRoute struct {
	Family  string `xml:"family,attr,omitempty"`
	Address string `xml:"address,attr"`
	Netmask string `xml:"netmask,attr,omitempty"`
	Prefix  uint   `xml:"prefix,attr,omitempty"`
	Gateway string `xml:"gateway,attr"`
	Metric  uint   `xml:"metric,attr,omitempty"`
}

type DomainInterfaceTeaming struct {
	Type       string `xml:"type,attr"`
	Persistent string `xml:"persistent,attr,omitempty"`
}

type DomainInterfacePortOptions struct {
	Isolated string `xml:"isolated,attr,omitempty"`
}

type DomainInterface struct {
	XMLName             xml.Name                           `xml:"interface"`
	Managed             string                             `xml:"managed,attr,omitempty"`
	TrustGuestRXFilters string                             `xml:"trustGuestRxFilters,attr,omitempty"`
	MAC                 *DomainInterfaceMAC                `xml:"mac"`
	Source              *DomainInterfaceSource             `xml:"source"`
	Boot                *DomainDeviceBoot                  `xml:"boot"`
	VLan                *DomainInterfaceVLan               `xml:"vlan"`
	VirtualPort         *DomainInterfaceVirtualPort        `xml:"virtualport"`
	IP                  []DomainInterfaceIP                `xml:"ip"`
	Route               []DomainInterfaceRoute             `xml:"route"`
	PortForward         []DomainInterfaceSourcePortForward `xml:"portForward"`
	Script              *DomainInterfaceScript             `xml:"script"`
	DownScript          *DomainInterfaceScript             `xml:"downscript"`
	BackendDomain       *DomainBackendDomain               `xml:"backenddomain"`
	Target              *DomainInterfaceTarget             `xml:"target"`
	Guest               *DomainInterfaceGuest              `xml:"guest"`
	Model               *DomainInterfaceModel              `xml:"model"`
	Driver              *DomainInterfaceDriver             `xml:"driver"`
	Backend             *DomainInterfaceBackend            `xml:"backend"`
	FilterRef           *DomainInterfaceFilterRef          `xml:"filterref"`
	Tune                *DomainInterfaceTune               `xml:"tune"`
	Teaming             *DomainInterfaceTeaming            `xml:"teaming"`
	Link                *DomainInterfaceLink               `xml:"link"`
	MTU                 *DomainInterfaceMTU                `xml:"mtu"`
	Bandwidth           *DomainInterfaceBandwidth          `xml:"bandwidth"`
	PortOptions         *DomainInterfacePortOptions        `xml:"port"`
	Coalesce            *DomainInterfaceCoalesce           `xml:"coalesce"`
	ROM                 *DomainROM                         `xml:"rom"`
	ACPI                *DomainDeviceACPI                  `xml:"acpi"`
	Alias               *DomainAlias                       `xml:"alias"`
	Address             *DomainAddress                     `xml:"address"`
}

type DomainChardevSource struct {
	Null        *DomainChardevSourceNull        `xml:"-"`
	VC          *DomainChardevSourceVC          `xml:"-"`
	Pty         *DomainChardevSourcePty         `xml:"-"`
	Dev         *DomainChardevSourceDev         `xml:"-"`
	File        *DomainChardevSourceFile        `xml:"-"`
	Pipe        *DomainChardevSourcePipe        `xml:"-"`
	StdIO       *DomainChardevSourceStdIO       `xml:"-"`
	UDP         *DomainChardevSourceUDP         `xml:"-"`
	TCP         *DomainChardevSourceTCP         `xml:"-"`
	UNIX        *DomainChardevSourceUNIX        `xml:"-"`
	SpiceVMC    *DomainChardevSourceSpiceVMC    `xml:"-"`
	SpicePort   *DomainChardevSourceSpicePort   `xml:"-"`
	NMDM        *DomainChardevSourceNMDM        `xml:"-"`
	QEMUVDAgent *DomainChardevSourceQEMUVDAgent `xml:"-"`
	DBus        *DomainChardevSourceDBus        `xml:"-"`
}

type DomainChardevSourceNull struct {
}

type DomainChardevSourceVC struct {
}

type DomainChardevSourcePty struct {
	Path     string                 `xml:"path,attr"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainChardevSourceDev struct {
	Path     string                 `xml:"path,attr"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainChardevSourceFile struct {
	Path     string                 `xml:"path,attr"`
	Append   string                 `xml:"append,attr,omitempty"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainChardevSourcePipe struct {
	Path     string                 `xml:"path,attr"`
	SecLabel []DomainDeviceSecLabel `xml:"seclabel"`
}

type DomainChardevSourceStdIO struct {
}

type DomainChardevSourceUDP struct {
	BindHost       string `xml:"-"`
	BindService    string `xml:"-"`
	ConnectHost    string `xml:"-"`
	ConnectService string `xml:"-"`
}

type DomainChardevSourceReconnect struct {
	Enabled string `xml:"enabled,attr"`
	Timeout *uint  `xml:"timeout,attr"`
}

type DomainChardevSourceTCP struct {
	Mode      string                        `xml:"mode,attr,omitempty"`
	Host      string                        `xml:"host,attr,omitempty"`
	Service   string                        `xml:"service,attr,omitempty"`
	TLS       string                        `xml:"tls,attr,omitempty"`
	Reconnect *DomainChardevSourceReconnect `xml:"reconnect"`
}

type DomainChardevSourceUNIX struct {
	Mode      string                        `xml:"mode,attr,omitempty"`
	Path      string                        `xml:"path,attr,omitempty"`
	Reconnect *DomainChardevSourceReconnect `xml:"reconnect"`
	SecLabel  []DomainDeviceSecLabel        `xml:"seclabel"`
}

type DomainChardevSourceSpiceVMC struct {
}

type DomainChardevSourceSpicePort struct {
	Channel string `xml:"channel,attr"`
}

type DomainChardevSourceNMDM struct {
	Master string `xml:"master,attr"`
	Slave  string `xml:"slave,attr"`
}

type DomainChardevSourceQEMUVDAgentMouse struct {
	Mode string `xml:"mode,attr"`
}

type DomainChardevSourceQEMUVDAgentClipBoard struct {
	CopyPaste string `xml:"copypaste,attr"`
}

type DomainChardevSourceQEMUVDAgent struct {
	Mouse     *DomainChardevSourceQEMUVDAgentMouse     `xml:"mouse"`
	ClipBoard *DomainChardevSourceQEMUVDAgentClipBoard `xml:"clipboard"`
}

type DomainChardevSourceDBus struct {
	Channel string `xml:"channel,attr,omitempty"`
}

type DomainChardevTarget struct {
	Type  string `xml:"type,attr,omitempty"`
	Name  string `xml:"name,attr,omitempty"`
	State string `xml:"state,attr,omitempty"` // is guest agent connected?
	Port  *uint  `xml:"port,attr"`
}

type DomainConsoleTarget struct {
	Type string `xml:"type,attr,omitempty"`
	Port *uint  `xml:"port,attr"`
}

type DomainSerialTarget struct {
	Type  string                   `xml:"type,attr,omitempty"`
	Port  *uint                    `xml:"port,attr"`
	Model *DomainSerialTargetModel `xml:"model"`
}

type DomainSerialTargetModel struct {
	Name string `xml:"name,attr,omitempty"`
}

type DomainParallelTarget struct {
	Type string `xml:"type,attr,omitempty"`
	Port *uint  `xml:"port,attr"`
}

type DomainChannelTarget struct {
	VirtIO   *DomainChannelTargetVirtIO   `xml:"-"`
	Xen      *DomainChannelTargetXen      `xml:"-"`
	GuestFWD *DomainChannelTargetGuestFWD `xml:"-"`
}

type DomainChannelTargetVirtIO struct {
	Name  string `xml:"name,attr,omitempty"`
	State string `xml:"state,attr,omitempty"` // is guest agent connected?
}

type DomainChannelTargetXen struct {
	Name  string `xml:"name,attr,omitempty"`
	State string `xml:"state,attr,omitempty"` // is guest agent connected?
}

type DomainChannelTargetGuestFWD struct {
	Address string `xml:"address,attr,omitempty"`
	Port    string `xml:"port,attr,omitempty"`
}

type DomainAlias struct {
	Name string `xml:"name,attr"`
}

type DomainDeviceACPI struct {
	Index uint `xml:"index,attr,omitempty"`
}

type DomainAddressPCI struct {
	Domain        *uint              `xml:"domain,attr"`
	Bus           *uint              `xml:"bus,attr"`
	Slot          *uint              `xml:"slot,attr"`
	Function      *uint              `xml:"function,attr"`
	MultiFunction string             `xml:"multifunction,attr,omitempty"`
	ZPCI          *DomainAddressZPCI `xml:"zpci"`
}

type DomainAddressZPCI struct {
	UID *uint `xml:"uid,attr,omitempty"`
	FID *uint `xml:"fid,attr,omitempty"`
}

type DomainAddressUSB struct {
	Bus    *uint  `xml:"bus,attr"`
	Port   string `xml:"port,attr,omitempty"`
	Device *uint  `xml:"device,attr"`
}

type DomainAddressDrive struct {
	Controller *uint `xml:"controller,attr"`
	Bus        *uint `xml:"bus,attr"`
	Target     *uint `xml:"target,attr"`
	Unit       *uint `xml:"unit,attr"`
}

type DomainAddressDIMM struct {
	Slot *uint   `xml:"slot,attr"`
	Base *uint64 `xml:"base,attr"`
}

type DomainAddressISA struct {
	IOBase *uint `xml:"iobase,attr"`
	IRQ    *uint `xml:"irq,attr"`
}

type DomainAddressVirtioMMIO struct {
}

type DomainAddressCCW struct {
	CSSID *uint `xml:"cssid,attr"`
	SSID  *uint `xml:"ssid,attr"`
	DevNo *uint `xml:"devno,attr"`
}

type DomainAddressVirtioSerial struct {
	Controller *uint `xml:"controller,attr"`
	Bus        *uint `xml:"bus,attr"`
	Port       *uint `xml:"port,attr"`
}

type DomainAddressSpaprVIO struct {
	Reg *uint64 `xml:"reg,attr"`
}

type DomainAddressCCID struct {
	Controller *uint `xml:"controller,attr"`
	Slot       *uint `xml:"slot,attr"`
}

type DomainAddressVirtioS390 struct {
}

type DomainAddressUnassigned struct {
}

type DomainAddress struct {
	PCI          *DomainAddressPCI
	Drive        *DomainAddressDrive
	VirtioSerial *DomainAddressVirtioSerial
	CCID         *DomainAddressCCID
	USB          *DomainAddressUSB
	SpaprVIO     *DomainAddressSpaprVIO
	VirtioS390   *DomainAddressVirtioS390
	CCW          *DomainAddressCCW
	VirtioMMIO   *DomainAddressVirtioMMIO
	ISA          *DomainAddressISA
	DIMM         *DomainAddressDIMM
	Unassigned   *DomainAddressUnassigned
}

type DomainChardevLog struct {
	File   string `xml:"file,attr"`
	Append string `xml:"append,attr,omitempty"`
}

type DomainConsole struct {
	XMLName  xml.Name               `xml:"console"`
	TTY      string                 `xml:"tty,attr,omitempty"`
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
	Target   *DomainConsoleTarget   `xml:"target"`
	Log      *DomainChardevLog      `xml:"log"`
	ACPI     *DomainDeviceACPI      `xml:"acpi"`
	Alias    *DomainAlias           `xml:"alias"`
	Address  *DomainAddress         `xml:"address"`
}

type DomainSerial struct {
	XMLName  xml.Name               `xml:"serial"`
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
	Target   *DomainSerialTarget    `xml:"target"`
	Log      *DomainChardevLog      `xml:"log"`
	ACPI     *DomainDeviceACPI      `xml:"acpi"`
	Alias    *DomainAlias           `xml:"alias"`
	Address  *DomainAddress         `xml:"address"`
}

type DomainParallel struct {
	XMLName  xml.Name               `xml:"parallel"`
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
	Target   *DomainParallelTarget  `xml:"target"`
	Log      *DomainChardevLog      `xml:"log"`
	ACPI     *DomainDeviceACPI      `xml:"acpi"`
	Alias    *DomainAlias           `xml:"alias"`
	Address  *DomainAddress         `xml:"address"`
}

type DomainChardevProtocol struct {
	Type string `xml:"type,attr"`
}

type DomainChannel struct {
	XMLName  xml.Name               `xml:"channel"`
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
	Target   *DomainChannelTarget   `xml:"target"`
	Log      *DomainChardevLog      `xml:"log"`
	ACPI     *DomainDeviceACPI      `xml:"acpi"`
	Alias    *DomainAlias           `xml:"alias"`
	Address  *DomainAddress         `xml:"address"`
}

type DomainRedirDev struct {
	XMLName  xml.Name               `xml:"redirdev"`
	Bus      string                 `xml:"bus,attr,omitempty"`
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
	Boot     *DomainDeviceBoot      `xml:"boot"`
	ACPI     *DomainDeviceACPI      `xml:"acpi"`
	Alias    *DomainAlias           `xml:"alias"`
	Address  *DomainAddress         `xml:"address"`
}

type DomainRedirFilter struct {
	USB []DomainRedirFilterUSB `xml:"usbdev"`
}

type DomainRedirFilterUSB struct {
	Class   *uint  `xml:"class,attr"`
	Vendor  *uint  `xml:"vendor,attr"`
	Product *uint  `xml:"product,attr"`
	Version string `xml:"version,attr,omitempty"`
	Allow   string `xml:"allow,attr"`
}

type DomainInput struct {
	XMLName xml.Name           `xml:"input"`
	Type    string             `xml:"type,attr"`
	Bus     string             `xml:"bus,attr,omitempty"`
	Model   string             `xml:"model,attr,omitempty"`
	Driver  *DomainInputDriver `xml:"driver"`
	Source  *DomainInputSource `xml:"source"`
	ACPI    *DomainDeviceACPI  `xml:"acpi"`
	Alias   *DomainAlias       `xml:"alias"`
	Address *DomainAddress     `xml:"address"`
}

type DomainInputDriver struct {
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainInputSource struct {
	Passthrough *DomainInputSourcePassthrough `xml:"-"`
	EVDev       *DomainInputSourceEVDev       `xml:"-"`
}

type DomainInputSourcePassthrough struct {
	EVDev string `xml:"evdev,attr"`
}

type DomainInputSourceEVDev struct {
	Dev        string `xml:"dev,attr"`
	Grab       string `xml:"grab,attr,omitempty"`
	GrabToggle string `xml:"grabToggle,attr,omitempty"`
	Repeat     string `xml:"repeat,attr,omitempty"`
}

type DomainGraphicListenerAddress struct {
	Address string `xml:"address,attr,omitempty"`
}

type DomainGraphicListenerNetwork struct {
	Address string `xml:"address,attr,omitempty"`
	Network string `xml:"network,attr,omitempty"`
}

type DomainGraphicListenerSocket struct {
	Socket string `xml:"socket,attr,omitempty"`
}

type DomainGraphicListener struct {
	Address *DomainGraphicListenerAddress `xml:"-"`
	Network *DomainGraphicListenerNetwork `xml:"-"`
	Socket  *DomainGraphicListenerSocket  `xml:"-"`
}

type DomainGraphicChannel struct {
	Name string `xml:"name,attr,omitempty"`
	Mode string `xml:"mode,attr,omitempty"`
}

type DomainGraphicFileTransfer struct {
	Enable string `xml:"enable,attr,omitempty"`
}

type DomainGraphicsSDLGL struct {
	Enable string `xml:"enable,attr,omitempty"`
}

type DomainGraphicSDL struct {
	Display    string               `xml:"display,attr,omitempty"`
	XAuth      string               `xml:"xauth,attr,omitempty"`
	FullScreen string               `xml:"fullscreen,attr,omitempty"`
	GL         *DomainGraphicsSDLGL `xml:"gl"`
}

type DomainGraphicVNC struct {
	Socket        string                  `xml:"socket,attr,omitempty"`
	Port          int                     `xml:"port,attr,omitempty"`
	AutoPort      string                  `xml:"autoport,attr,omitempty"`
	WebSocket     int                     `xml:"websocket,attr,omitempty"`
	Keymap        string                  `xml:"keymap,attr,omitempty"`
	SharePolicy   string                  `xml:"sharePolicy,attr,omitempty"`
	Passwd        string                  `xml:"passwd,attr,omitempty"`
	PasswdValidTo string                  `xml:"passwdValidTo,attr,omitempty"`
	Connected     string                  `xml:"connected,attr,omitempty"`
	PowerControl  string                  `xml:"powerControl,attr,omitempty"`
	Listen        string                  `xml:"listen,attr,omitempty"`
	Listeners     []DomainGraphicListener `xml:"listen"`
}

type DomainGraphicRDP struct {
	Port        int                     `xml:"port,attr,omitempty"`
	AutoPort    string                  `xml:"autoport,attr,omitempty"`
	ReplaceUser string                  `xml:"replaceUser,attr,omitempty"`
	MultiUser   string                  `xml:"multiUser,attr,omitempty"`
	Listen      string                  `xml:"listen,attr,omitempty"`
	Listeners   []DomainGraphicListener `xml:"listen"`
}

type DomainGraphicDesktop struct {
	Display    string `xml:"display,attr,omitempty"`
	FullScreen string `xml:"fullscreen,attr,omitempty"`
}

type DomainGraphicSpiceChannel struct {
	Name string `xml:"name,attr"`
	Mode string `xml:"mode,attr"`
}

type DomainGraphicSpiceImage struct {
	Compression string `xml:"compression,attr"`
}

type DomainGraphicSpiceJPEG struct {
	Compression string `xml:"compression,attr"`
}

type DomainGraphicSpiceZLib struct {
	Compression string `xml:"compression,attr"`
}

type DomainGraphicSpicePlayback struct {
	Compression string `xml:"compression,attr"`
}

type DomainGraphicSpiceStreaming struct {
	Mode string `xml:"mode,attr"`
}

type DomainGraphicSpiceMouse struct {
	Mode string `xml:"mode,attr"`
}

type DomainGraphicSpiceClipBoard struct {
	CopyPaste string `xml:"copypaste,attr"`
}

type DomainGraphicSpiceFileTransfer struct {
	Enable string `xml:"enable,attr"`
}

type DomainGraphicSpiceGL struct {
	Enable     string `xml:"enable,attr,omitempty"`
	RenderNode string `xml:"rendernode,attr,omitempty"`
}

type DomainGraphicSpice struct {
	Port          int                             `xml:"port,attr,omitempty"`
	TLSPort       int                             `xml:"tlsPort,attr,omitempty"`
	AutoPort      string                          `xml:"autoport,attr,omitempty"`
	Listen        string                          `xml:"listen,attr,omitempty"`
	Keymap        string                          `xml:"keymap,attr,omitempty"`
	DefaultMode   string                          `xml:"defaultMode,attr,omitempty"`
	Passwd        string                          `xml:"passwd,attr,omitempty"`
	PasswdValidTo string                          `xml:"passwdValidTo,attr,omitempty"`
	Connected     string                          `xml:"connected,attr,omitempty"`
	Listeners     []DomainGraphicListener         `xml:"listen"`
	Channel       []DomainGraphicSpiceChannel     `xml:"channel"`
	Image         *DomainGraphicSpiceImage        `xml:"image"`
	JPEG          *DomainGraphicSpiceJPEG         `xml:"jpeg"`
	ZLib          *DomainGraphicSpiceZLib         `xml:"zlib"`
	Playback      *DomainGraphicSpicePlayback     `xml:"playback"`
	Streaming     *DomainGraphicSpiceStreaming    `xml:"streaming"`
	Mouse         *DomainGraphicSpiceMouse        `xml:"mouse"`
	ClipBoard     *DomainGraphicSpiceClipBoard    `xml:"clipboard"`
	FileTransfer  *DomainGraphicSpiceFileTransfer `xml:"filetransfer"`
	GL            *DomainGraphicSpiceGL           `xml:"gl"`
}

type DomainGraphicEGLHeadlessGL struct {
	RenderNode string `xml:"rendernode,attr,omitempty"`
}

type DomainGraphicEGLHeadless struct {
	GL *DomainGraphicEGLHeadlessGL `xml:"gl"`
}

type DomainGraphicDBusGL struct {
	Enable     string `xml:"enable,attr,omitempty"`
	RenderNode string `xml:"rendernode,attr,omitempty"`
}

type DomainGraphicDBus struct {
	Address string               `xml:"address,attr,omitempty"`
	P2P     string               `xml:"p2p,attr,omitempty"`
	GL      *DomainGraphicDBusGL `xml:"gl"`
}

type DomainGraphicAudio struct {
	ID uint `xml:"id,attr,omitempty"`
}

type DomainGraphic struct {
	XMLName     xml.Name                  `xml:"graphics"`
	SDL         *DomainGraphicSDL         `xml:"-"`
	VNC         *DomainGraphicVNC         `xml:"-"`
	RDP         *DomainGraphicRDP         `xml:"-"`
	Desktop     *DomainGraphicDesktop     `xml:"-"`
	Spice       *DomainGraphicSpice       `xml:"-"`
	EGLHeadless *DomainGraphicEGLHeadless `xml:"-"`
	DBus        *DomainGraphicDBus        `xml:"-"`
	Audio       *DomainGraphicAudio       `xml:"audio"`
}

type DomainVideoAccel struct {
	Accel3D    string `xml:"accel3d,attr,omitempty"`
	Accel2D    string `xml:"accel2d,attr,omitempty"`
	RenderNode string `xml:"rendernode,attr,omitempty"`
}

type DomainVideoResolution struct {
	X uint `xml:"x,attr"`
	Y uint `xml:"y,attr"`
}

type DomainVideoModel struct {
	Type       string                 `xml:"type,attr"`
	Heads      uint                   `xml:"heads,attr,omitempty"`
	Ram        uint                   `xml:"ram,attr,omitempty"`
	VRam       uint                   `xml:"vram,attr,omitempty"`
	VRam64     uint                   `xml:"vram64,attr,omitempty"`
	VGAMem     uint                   `xml:"vgamem,attr,omitempty"`
	Primary    string                 `xml:"primary,attr,omitempty"`
	Blob       string                 `xml:"blob,attr,omitempty"`
	Accel      *DomainVideoAccel      `xml:"acceleration"`
	Resolution *DomainVideoResolution `xml:"resolution"`
}

type DomainVideo struct {
	XMLName xml.Name           `xml:"video"`
	Model   DomainVideoModel   `xml:"model"`
	Driver  *DomainVideoDriver `xml:"driver"`
	ACPI    *DomainDeviceACPI  `xml:"acpi"`
	Alias   *DomainAlias       `xml:"alias"`
	Address *DomainAddress     `xml:"address"`
}

type DomainVideoDriver struct {
	Name      string `xml:"name,attr,omitempty"`
	VGAConf   string `xml:"vgaconf,attr,omitempty"`
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainMemBalloonStats struct {
	Period uint `xml:"period,attr"`
}

type DomainMemBalloon struct {
	XMLName           xml.Name                `xml:"memballoon"`
	Model             string                  `xml:"model,attr"`
	AutoDeflate       string                  `xml:"autodeflate,attr,omitempty"`
	FreePageReporting string                  `xml:"freePageReporting,attr,omitempty"`
	Driver            *DomainMemBalloonDriver `xml:"driver"`
	Stats             *DomainMemBalloonStats  `xml:"stats"`
	ACPI              *DomainDeviceACPI       `xml:"acpi"`
	Alias             *DomainAlias            `xml:"alias"`
	Address           *DomainAddress          `xml:"address"`
}

type DomainVSockCID struct {
	Auto    string `xml:"auto,attr,omitempty"`
	Address string `xml:"address,attr,omitempty"`
}

type DomainVSockDriver struct {
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainVSock struct {
	XMLName xml.Name           `xml:"vsock"`
	Model   string             `xml:"model,attr,omitempty"`
	CID     *DomainVSockCID    `xml:"cid"`
	Driver  *DomainVSockDriver `xml:"driver"`
	ACPI    *DomainDeviceACPI  `xml:"acpi"`
	Alias   *DomainAlias       `xml:"alias"`
	Address *DomainAddress     `xml:"address"`
}

type DomainMemBalloonDriver struct {
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainPanic struct {
	XMLName xml.Name          `xml:"panic"`
	Model   string            `xml:"model,attr,omitempty"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainSoundDriver struct {
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainSoundCodec struct {
	Type string `xml:"type,attr"`
}

type DomainSound struct {
	XMLName      xml.Name           `xml:"sound"`
	Model        string             `xml:"model,attr"`
	MultiChannel string             `xml:"multichannel,attr,omitempty"`
	Streams      uint               `xml:"streams,attr,omitempty"`
	Codec        []DomainSoundCodec `xml:"codec"`
	Audio        *DomainSoundAudio  `xml:"audio"`
	ACPI         *DomainDeviceACPI  `xml:"acpi"`
	Alias        *DomainAlias       `xml:"alias"`
	Driver       *DomainSoundDriver `xml:"driver"`
	Address      *DomainAddress     `xml:"address"`
}

type DomainSoundAudio struct {
	ID uint `xml:"id,attr"`
}

type DomainAudio struct {
	XMLName     xml.Name               `xml:"audio"`
	ID          int                    `xml:"id,attr"`
	TimerPeriod uint                   `xml:"timerPeriod,attr,omitempty"`
	None        *DomainAudioNone       `xml:"-"`
	ALSA        *DomainAudioALSA       `xml:"-"`
	CoreAudio   *DomainAudioCoreAudio  `xml:"-"`
	Jack        *DomainAudioJack       `xml:"-"`
	OSS         *DomainAudioOSS        `xml:"-"`
	PulseAudio  *DomainAudioPulseAudio `xml:"-"`
	SDL         *DomainAudioSDL        `xml:"-"`
	SPICE       *DomainAudioSPICE      `xml:"-"`
	File        *DomainAudioFile       `xml:"-"`
	DBus        *DomainAudioDBus       `xml:"-"`
	PipeWire    *DomainAudioPipeWire   `xml:"-"`
}

type DomainAudioChannel struct {
	MixingEngine  string                      `xml:"mixingEngine,attr,omitempty"`
	FixedSettings string                      `xml:"fixedSettings,attr,omitempty"`
	Voices        uint                        `xml:"voices,attr,omitempty"`
	Settings      *DomainAudioChannelSettings `xml:"settings"`
	BufferLength  uint                        `xml:"bufferLength,attr,omitempty"`
}

type DomainAudioChannelSettings struct {
	Frequency uint   `xml:"frequency,attr,omitempty"`
	Channels  uint   `xml:"channels,attr,omitempty"`
	Format    string `xml:"format,attr,omitempty"`
}

type DomainAudioNone struct {
	Input  *DomainAudioNoneChannel `xml:"input"`
	Output *DomainAudioNoneChannel `xml:"output"`
}

type DomainAudioNoneChannel struct {
	DomainAudioChannel
}

type DomainAudioALSA struct {
	Input  *DomainAudioALSAChannel `xml:"input"`
	Output *DomainAudioALSAChannel `xml:"output"`
}

type DomainAudioALSAChannel struct {
	DomainAudioChannel
	Dev string `xml:"dev,attr,omitempty"`
}

type DomainAudioCoreAudio struct {
	Input  *DomainAudioCoreAudioChannel `xml:"input"`
	Output *DomainAudioCoreAudioChannel `xml:"output"`
}

type DomainAudioCoreAudioChannel struct {
	DomainAudioChannel
	BufferCount uint `xml:"bufferCount,attr,omitempty"`
}

type DomainAudioJack struct {
	Input  *DomainAudioJackChannel `xml:"input"`
	Output *DomainAudioJackChannel `xml:"output"`
}

type DomainAudioJackChannel struct {
	DomainAudioChannel
	ServerName   string `xml:"serverName,attr,omitempty"`
	ClientName   string `xml:"clientName,attr,omitempty"`
	ConnectPorts string `xml:"connectPorts,attr,omitempty"`
	ExactName    string `xml:"exactName,attr,omitempty"`
}

type DomainAudioOSS struct {
	TryMMap   string `xml:"tryMMap,attr,omitempty"`
	Exclusive string `xml:"exclusive,attr,omitempty"`
	DSPPolicy *int   `xml:"dspPolicy,attr"`

	Input  *DomainAudioOSSChannel `xml:"input"`
	Output *DomainAudioOSSChannel `xml:"output"`
}

type DomainAudioOSSChannel struct {
	DomainAudioChannel
	Dev         string `xml:"dev,attr,omitempty"`
	BufferCount uint   `xml:"bufferCount,attr,omitempty"`
	TryPoll     string `xml:"tryPoll,attr,omitempty"`
}

type DomainAudioPulseAudio struct {
	ServerName string                        `xml:"serverName,attr,omitempty"`
	Input      *DomainAudioPulseAudioChannel `xml:"input"`
	Output     *DomainAudioPulseAudioChannel `xml:"output"`
}

type DomainAudioPulseAudioChannel struct {
	DomainAudioChannel
	Name       string `xml:"name,attr,omitempty"`
	StreamName string `xml:"streamName,attr,omitempty"`
	Latency    uint   `xml:"latency,attr,omitempty"`
}

type DomainAudioPipeWire struct {
	RuntimeDir string                        `xml:"runtimeDir,attr,omitempty"`
	Input      *DomainAudioPulseAudioChannel `xml:"input"`
	Output     *DomainAudioPulseAudioChannel `xml:"output"`
}

type DomainAudioPipeWireChannel struct {
	DomainAudioChannel
	Name       string `xml:"name,attr,omitempty"`
	StreamName string `xml:"streamName,attr,omitempty"`
	Latency    uint   `xml:"latency,attr,omitempty"`
}

type DomainAudioSDL struct {
	Driver string                 `xml:"driver,attr,omitempty"`
	Input  *DomainAudioSDLChannel `xml:"input"`
	Output *DomainAudioSDLChannel `xml:"output"`
}

type DomainAudioSDLChannel struct {
	DomainAudioChannel
	BufferCount uint `xml:"bufferCount,attr,omitempty"`
}

type DomainAudioSPICE struct {
	Input  *DomainAudioSPICEChannel `xml:"input"`
	Output *DomainAudioSPICEChannel `xml:"output"`
}

type DomainAudioSPICEChannel struct {
	DomainAudioChannel
}

type DomainAudioFile struct {
	Path   string                  `xml:"path,attr,omitempty"`
	Input  *DomainAudioFileChannel `xml:"input"`
	Output *DomainAudioFileChannel `xml:"output"`
}

type DomainAudioFileChannel struct {
	DomainAudioChannel
}

type DomainAudioDBus struct {
	Input  *DomainAudioDBusChannel `xml:"input"`
	Output *DomainAudioDBusChannel `xml:"output"`
}

type DomainAudioDBusChannel struct {
	DomainAudioChannel
}

type DomainRNGRate struct {
	Bytes  uint `xml:"bytes,attr"`
	Period uint `xml:"period,attr,omitempty"`
}

type DomainRNGBackend struct {
	Random  *DomainRNGBackendRandom  `xml:"-"`
	EGD     *DomainRNGBackendEGD     `xml:"-"`
	BuiltIn *DomainRNGBackendBuiltIn `xml:"-"`
}

type DomainRNGBackendEGD struct {
	Source   *DomainChardevSource   `xml:"source"`
	Protocol *DomainChardevProtocol `xml:"protocol"`
}

type DomainRNGBackendRandom struct {
	Device string `xml:",chardata"`
}

type DomainRNGBackendBuiltIn struct {
}

type DomainRNG struct {
	XMLName xml.Name          `xml:"rng"`
	Model   string            `xml:"model,attr"`
	Driver  *DomainRNGDriver  `xml:"driver"`
	Rate    *DomainRNGRate    `xml:"rate"`
	Backend *DomainRNGBackend `xml:"backend"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainRNGDriver struct {
	IOMMU     string `xml:"iommu,attr,omitempty"`
	ATS       string `xml:"ats,attr,omitempty"`
	Packed    string `xml:"packed,attr,omitempty"`
	PagePerVQ string `xml:"page_per_vq,attr,omitempty"`
}

type DomainHostdevSubsysUSB struct {
	Source *DomainHostdevSubsysUSBSource `xml:"source"`
}

type DomainHostdevSubsysUSBSource struct {
	GuestReset string            `xml:"guestReset,attr,omitempty"`
	Address    *DomainAddressUSB `xml:"address"`
}

type DomainHostdevSubsysSCSI struct {
	SGIO      string                         `xml:"sgio,attr,omitempty"`
	RawIO     string                         `xml:"rawio,attr,omitempty"`
	Source    *DomainHostdevSubsysSCSISource `xml:"source"`
	ReadOnly  *DomainDiskReadOnly            `xml:"readonly"`
	Shareable *DomainDiskShareable           `xml:"shareable"`
}

type DomainHostdevSubsysSCSISource struct {
	Host  *DomainHostdevSubsysSCSISourceHost  `xml:"-"`
	ISCSI *DomainHostdevSubsysSCSISourceISCSI `xml:"-"`
}

type DomainHostdevSubsysSCSIAdapter struct {
	Name string `xml:"name,attr"`
}

type DomainHostdevSubsysSCSISourceHost struct {
	Adapter *DomainHostdevSubsysSCSIAdapter `xml:"adapter"`
	Address *DomainAddressDrive             `xml:"address"`
}

type DomainHostdevSubsysSCSISourceISCSI struct {
	Name      string                                  `xml:"name,attr"`
	Host      []DomainDiskSourceHost                  `xml:"host"`
	Auth      *DomainDiskAuth                         `xml:"auth"`
	Initiator *DomainHostdevSubsysSCSISourceInitiator `xml:"initiator"`
}

type DomainHostdevSubsysSCSISourceInitiator struct {
	IQN DomainHostdevSubsysSCSISourceIQN `xml:"iqn"`
}

type DomainHostdevSubsysSCSISourceIQN struct {
	Name string `xml:"name,attr"`
}

type DomainHostdevSubsysSCSIHost struct {
	Model  string                             `xml:"model,attr,omitempty"`
	Source *DomainHostdevSubsysSCSIHostSource `xml:"source"`
}

type DomainHostdevSubsysSCSIHostSource struct {
	Protocol string `xml:"protocol,attr,omitempty"`
	WWPN     string `xml:"wwpn,attr,omitempty"`
}

type DomainHostdevSubsysPCISource struct {
	WriteFiltering string            `xml:"writeFiltering,attr,omitempty"`
	Address        *DomainAddressPCI `xml:"address"`
}

type DomainHostdevSubsysPCIDriver struct {
	Name  string `xml:"name,attr,omitempty"`
	Model string `xml:"model,attr,omitempty"`
}

type DomainHostdevSubsysPCI struct {
	Display string                        `xml:"display,attr,omitempty"`
	RamFB   string                        `xml:"ramfb,attr,omitempty"`
	Driver  *DomainHostdevSubsysPCIDriver `xml:"driver"`
	Source  *DomainHostdevSubsysPCISource `xml:"source"`
	Teaming *DomainInterfaceTeaming       `xml:"teaming"`
}

type DomainAddressMDev struct {
	UUID string `xml:"uuid,attr"`
}

type DomainHostdevSubsysMDevSource struct {
	Address *DomainAddressMDev `xml:"address"`
}

type DomainHostdevSubsysMDev struct {
	Model   string                         `xml:"model,attr,omitempty"`
	Display string                         `xml:"display,attr,omitempty"`
	RamFB   string                         `xml:"ramfb,attr,omitempty"`
	Source  *DomainHostdevSubsysMDevSource `xml:"source"`
}

type DomainHostdevCapsStorage struct {
	Source *DomainHostdevCapsStorageSource `xml:"source"`
}

type DomainHostdevCapsStorageSource struct {
	Block string `xml:"block"`
}

type DomainHostdevCapsMisc struct {
	Source *DomainHostdevCapsMiscSource `xml:"source"`
}

type DomainHostdevCapsMiscSource struct {
	Char string `xml:"char"`
}

type DomainIP struct {
	Address string `xml:"address,attr,omitempty"`
	Family  string `xml:"family,attr,omitempty"`
	Prefix  *uint  `xml:"prefix,attr"`
}

type DomainRoute struct {
	Family  string `xml:"family,attr,omitempty"`
	Address string `xml:"address,attr,omitempty"`
	Gateway string `xml:"gateway,attr,omitempty"`
}

type DomainHostdevCapsNet struct {
	Source *DomainHostdevCapsNetSource `xml:"source"`
	IP     []DomainIP                  `xml:"ip"`
	Route  []DomainRoute               `xml:"route"`
}

type DomainHostdevCapsNetSource struct {
	Interface string `xml:"interface"`
}

type DomainHostdev struct {
	Managed        string                       `xml:"managed,attr,omitempty"`
	SubsysUSB      *DomainHostdevSubsysUSB      `xml:"-"`
	SubsysSCSI     *DomainHostdevSubsysSCSI     `xml:"-"`
	SubsysSCSIHost *DomainHostdevSubsysSCSIHost `xml:"-"`
	SubsysPCI      *DomainHostdevSubsysPCI      `xml:"-"`
	SubsysMDev     *DomainHostdevSubsysMDev     `xml:"-"`
	CapsStorage    *DomainHostdevCapsStorage    `xml:"-"`
	CapsMisc       *DomainHostdevCapsMisc       `xml:"-"`
	CapsNet        *DomainHostdevCapsNet        `xml:"-"`
	Boot           *DomainDeviceBoot            `xml:"boot"`
	ROM            *DomainROM                   `xml:"rom"`
	ACPI           *DomainDeviceACPI            `xml:"acpi"`
	Alias          *DomainAlias                 `xml:"alias"`
	Address        *DomainAddress               `xml:"address"`
}

type DomainMemorydevSource struct {
	NodeMask  string                          `xml:"nodemask,omitempty"`
	PageSize  *DomainMemorydevSourcePagesize  `xml:"pagesize"`
	Path      string                          `xml:"path,omitempty"`
	AlignSize *DomainMemorydevSourceAlignsize `xml:"alignsize"`
	PMem      *DomainMemorydevSourcePMem      `xml:"pmem"`
}

type DomainMemorydevSourcePMem struct {
}

type DomainMemorydevSourcePagesize struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemorydevSourceAlignsize struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemorydevTargetNode struct {
	Value uint `xml:",chardata"`
}

type DomainMemorydevTargetReadOnly struct {
}

type DomainMemorydevTargetSize struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemorydevTargetBlock struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemorydevTargetRequested struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemorydevTargetLabel struct {
	Size *DomainMemorydevTargetSize `xml:"size"`
}

type DomainMemorydevTargetAddress struct {
	Base *uint `xml:"base,attr"`
}

type DomainMemorydevTarget struct {
	DynamicMemslots string                          `xml:"dynamicMemslots,attr,omitempty"`
	Size            *DomainMemorydevTargetSize      `xml:"size"`
	Node            *DomainMemorydevTargetNode      `xml:"node"`
	Label           *DomainMemorydevTargetLabel     `xml:"label"`
	Block           *DomainMemorydevTargetBlock     `xml:"block"`
	Requested       *DomainMemorydevTargetRequested `xml:"requested"`
	ReadOnly        *DomainMemorydevTargetReadOnly  `xml:"readonly"`
	Address         *DomainMemorydevTargetAddress   `xml:"address"`
}

type DomainMemorydev struct {
	XMLName xml.Name               `xml:"memory"`
	Model   string                 `xml:"model,attr"`
	Access  string                 `xml:"access,attr,omitempty"`
	Discard string                 `xml:"discard,attr,omitempty"`
	UUID    string                 `xml:"uuid,omitempty"`
	Source  *DomainMemorydevSource `xml:"source"`
	Target  *DomainMemorydevTarget `xml:"target"`
	ACPI    *DomainDeviceACPI      `xml:"acpi"`
	Alias   *DomainAlias           `xml:"alias"`
	Address *DomainAddress         `xml:"address"`
}

type DomainWatchdog struct {
	XMLName xml.Name          `xml:"watchdog"`
	Model   string            `xml:"model,attr"`
	Action  string            `xml:"action,attr,omitempty"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainHub struct {
	Type    string            `xml:"type,attr"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainIOMMU struct {
	Model   string             `xml:"model,attr"`
	Driver  *DomainIOMMUDriver `xml:"driver"`
	ACPI    *DomainDeviceACPI  `xml:"acpi"`
	Alias   *DomainAlias       `xml:"alias"`
	Address *DomainAddress     `xml:"address"`
}

type DomainIOMMUDriver struct {
	IntRemap       string `xml:"intremap,attr,omitempty"`
	CachingMode    string `xml:"caching_mode,attr,omitempty"`
	EIM            string `xml:"eim,attr,omitempty"`
	IOTLB          string `xml:"iotlb,attr,omitempty"`
	AWBits         uint   `xml:"aw_bits,attr,omitempty"`
	DMATranslation string `xml:"dma_translation,attr,omitempty"`
}

type DomainNVRAM struct {
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainLease struct {
	Lockspace string             `xml:"lockspace"`
	Key       string             `xml:"key"`
	Target    *DomainLeaseTarget `xml:"target"`
}

type DomainLeaseTarget struct {
	Path   string `xml:"path,attr"`
	Offset uint64 `xml:"offset,attr,omitempty"`
}

type DomainSmartcard struct {
	XMLName     xml.Name                  `xml:"smartcard"`
	Passthrough *DomainChardevSource      `xml:"source"`
	Protocol    *DomainChardevProtocol    `xml:"protocol"`
	Host        *DomainSmartcardHost      `xml:"-"`
	HostCerts   []DomainSmartcardHostCert `xml:"certificate"`
	Database    string                    `xml:"database,omitempty"`
	ACPI        *DomainDeviceACPI         `xml:"acpi"`
	Alias       *DomainAlias              `xml:"alias"`
	Address     *DomainAddress            `xml:"address"`
}

type DomainSmartcardHost struct {
}

type DomainSmartcardHostCert struct {
	File string `xml:",chardata"`
}

type DomainTPM struct {
	XMLName xml.Name          `xml:"tpm"`
	Model   string            `xml:"model,attr,omitempty"`
	Backend *DomainTPMBackend `xml:"backend"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainTPMBackend struct {
	Passthrough *DomainTPMBackendPassthrough `xml:"-"`
	Emulator    *DomainTPMBackendEmulator    `xml:"-"`
	External    *DomainTPMBackendExternal    `xml:"-"`
}

type DomainTPMBackendPassthrough struct {
	Device *DomainTPMBackendDevice `xml:"device"`
}

type DomainTPMBackendEmulator struct {
	Version         string                      `xml:"version,attr,omitempty"`
	Encryption      *DomainTPMBackendEncryption `xml:"encryption"`
	PersistentState string                      `xml:"persistent_state,attr,omitempty"`
	Debug           uint                        `xml:"debug,attr,omitempty"`
	ActivePCRBanks  *DomainTPMBackendPCRBanks   `xml:"active_pcr_banks"`
	Source          *DomainTPMBackendSource     `xml:"source"`
	Profile         *DomainTPMBackendProfile    `xml:"profile"`
}

type DomainTPMBackendProfile struct {
	Source         string `xml:"source,attr,omitempty"`
	RemoveDisabled string `xml:"removeDisabled,attr,omitempty"`
	Name           string `xml:"name,attr,omitempty"`
}

type DomainTPMBackendSource struct {
	File *DomainTPMBackendSourceFile `xml:"-"`
	Dir  *DomainTPMBackendSourceDir  `xml:"-"`
}

type DomainTPMBackendSourceFile struct {
	Path string `xml:"path,attr,omitempty"`
}

type DomainTPMBackendSourceDir struct {
	Path string `xml:"path,attr,omitempty"`
}

type DomainTPMBackendPCRBanks struct {
	SHA1   *DomainTPMBackendPCRBank `xml:"sha1"`
	SHA256 *DomainTPMBackendPCRBank `xml:"sha256"`
	SHA384 *DomainTPMBackendPCRBank `xml:"sha384"`
	SHA512 *DomainTPMBackendPCRBank `xml:"sha512"`
}

type DomainTPMBackendPCRBank struct {
}

type DomainTPMBackendEncryption struct {
	Secret string `xml:"secret,attr"`
}

type DomainTPMBackendDevice struct {
	Path string `xml:"path,attr"`
}

type DomainTPMBackendExternalSource DomainChardevSource

type DomainTPMBackendExternal struct {
	Source *DomainTPMBackendExternalSource `xml:"source"`
}

type DomainShmem struct {
	XMLName xml.Name           `xml:"shmem"`
	Name    string             `xml:"name,attr"`
	Role    string             `xml:"role,attr,omitempty"`
	Size    *DomainShmemSize   `xml:"size"`
	Model   *DomainShmemModel  `xml:"model"`
	Server  *DomainShmemServer `xml:"server"`
	MSI     *DomainShmemMSI    `xml:"msi"`
	ACPI    *DomainDeviceACPI  `xml:"acpi"`
	Alias   *DomainAlias       `xml:"alias"`
	Address *DomainAddress     `xml:"address"`
}

type DomainShmemSize struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainShmemModel struct {
	Type string `xml:"type,attr"`
}

type DomainShmemServer struct {
	Path string `xml:"path,attr,omitempty"`
}

type DomainShmemMSI struct {
	Enabled   string `xml:"enabled,attr,omitempty"`
	Vectors   uint   `xml:"vectors,attr,omitempty"`
	IOEventFD string `xml:"ioeventfd,attr,omitempty"`
}

type DomainCrypto struct {
	Model   string               `xml:"model,attr,omitempty"`
	Type    string               `xml:"type,attr,omitempty"`
	Backend *DomainCryptoBackend `xml:"backend"`
	Alias   *DomainAlias         `xml:"alias"`
	Address *DomainAddress       `xml:"address"`
}

type DomainCryptoBackend struct {
	BuiltIn *DomainCryptoBackendBuiltIn `xml:"-"`
	LKCF    *DomainCryptoBackendLKCF    `xml:"-"`
	Queues  uint                        `xml:"queues,attr,omitempty"`
}

type DomainCryptoBackendBuiltIn struct {
}

type DomainCryptoBackendLKCF struct {
}

type DomainPStore struct {
	Backend string            `xml:"backend,attr"`
	Path    string            `xml:"path"`
	Size    DomainPStoreSize  `xml:"size"`
	ACPI    *DomainDeviceACPI `xml:"acpi"`
	Alias   *DomainAlias      `xml:"alias"`
	Address *DomainAddress    `xml:"address"`
}

type DomainPStoreSize struct {
	Size uint64 `xml:",chardata"`
	Unit string `xml:"unit,attr"`
}

type DomainDeviceList struct {
	Emulator     string              `xml:"emulator,omitempty"`
	Disks        []DomainDisk        `xml:"disk"`
	Controllers  []DomainController  `xml:"controller"`
	Leases       []DomainLease       `xml:"lease"`
	Filesystems  []DomainFilesystem  `xml:"filesystem"`
	Interfaces   []DomainInterface   `xml:"interface"`
	Smartcards   []DomainSmartcard   `xml:"smartcard"`
	Serials      []DomainSerial      `xml:"serial"`
	Parallels    []DomainParallel    `xml:"parallel"`
	Consoles     []DomainConsole     `xml:"console"`
	Channels     []DomainChannel     `xml:"channel"`
	Inputs       []DomainInput       `xml:"input"`
	TPMs         []DomainTPM         `xml:"tpm"`
	Graphics     []DomainGraphic     `xml:"graphics"`
	Sounds       []DomainSound       `xml:"sound"`
	Audios       []DomainAudio       `xml:"audio"`
	Videos       []DomainVideo       `xml:"video"`
	Hostdevs     []DomainHostdev     `xml:"hostdev"`
	RedirDevs    []DomainRedirDev    `xml:"redirdev"`
	RedirFilters []DomainRedirFilter `xml:"redirfilter"`
	Hubs         []DomainHub         `xml:"hub"`
	Watchdogs    []DomainWatchdog    `xml:"watchdog"`
	MemBalloon   *DomainMemBalloon   `xml:"memballoon"`
	RNGs         []DomainRNG         `xml:"rng"`
	NVRAM        *DomainNVRAM        `xml:"nvram"`
	Panics       []DomainPanic       `xml:"panic"`
	Shmems       []DomainShmem       `xml:"shmem"`
	Memorydevs   []DomainMemorydev   `xml:"memory"`
	IOMMU        *DomainIOMMU        `xml:"iommu"`
	VSock        *DomainVSock        `xml:"vsock"`
	Crypto       []DomainCrypto      `xml:"crypto"`
	PStore       *DomainPStore       `xml:"pstore"`
}

type DomainMemory struct {
	Value    uint   `xml:",chardata"`
	Unit     string `xml:"unit,attr,omitempty"`
	DumpCore string `xml:"dumpCore,attr,omitempty"`
}

type DomainCurrentMemory struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMaxMemory struct {
	Value uint   `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
	Slots uint   `xml:"slots,attr,omitempty"`
}

type DomainMemoryHugepage struct {
	Size    uint   `xml:"size,attr"`
	Unit    string `xml:"unit,attr,omitempty"`
	Nodeset string `xml:"nodeset,attr,omitempty"`
}

type DomainMemoryHugepages struct {
	Hugepages []DomainMemoryHugepage `xml:"page"`
}

type DomainMemoryNosharepages struct {
}

type DomainMemoryLocked struct {
}

type DomainMemorySource struct {
	Type string `xml:"type,attr,omitempty"`
}

type DomainMemoryAccess struct {
	Mode string `xml:"mode,attr,omitempty"`
}

type DomainMemoryAllocation struct {
	Mode    string `xml:"mode,attr,omitempty"`
	Threads uint   `xml:"threads,attr,omitempty"`
}

type DomainMemoryDiscard struct {
}

type DomainMemoryBacking struct {
	MemoryHugePages    *DomainMemoryHugepages    `xml:"hugepages"`
	MemoryNosharepages *DomainMemoryNosharepages `xml:"nosharepages"`
	MemoryLocked       *DomainMemoryLocked       `xml:"locked"`
	MemorySource       *DomainMemorySource       `xml:"source"`
	MemoryAccess       *DomainMemoryAccess       `xml:"access"`
	MemoryAllocation   *DomainMemoryAllocation   `xml:"allocation"`
	MemoryDiscard      *DomainMemoryDiscard      `xml:"discard"`
}

type DomainOSType struct {
	Arch    string `xml:"arch,attr,omitempty"`
	Machine string `xml:"machine,attr,omitempty"`
	Type    string `xml:",chardata"`
}

type DomainSMBios struct {
	Mode string `xml:"mode,attr"`
}

type DomainNVRam struct {
	NVRam          string            `xml:",chardata"`
	Source         *DomainDiskSource `xml:"source"`
	Template       string            `xml:"template,attr,omitempty"`
	Format         string            `xml:"format,attr,omitempty"`
	TemplateFormat string            `xml:"templateFormat,attr,omitempty"`
}

type DomainBootDevice struct {
	Dev string `xml:"dev,attr"`
}

type DomainBootMenu struct {
	Enable  string `xml:"enable,attr,omitempty"`
	Timeout string `xml:"timeout,attr,omitempty"`
}

type DomainSysInfoBIOS struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoSystem struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoBaseBoard struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoProcessor struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoMemory struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoChassis struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfoOEMStrings struct {
	Entry []string `xml:"entry"`
}

type DomainSysInfoSMBIOS struct {
	BIOS       *DomainSysInfoBIOS       `xml:"bios"`
	System     *DomainSysInfoSystem     `xml:"system"`
	BaseBoard  []DomainSysInfoBaseBoard `xml:"baseBoard"`
	Chassis    *DomainSysInfoChassis    `xml:"chassis"`
	Processor  []DomainSysInfoProcessor `xml:"processor"`
	Memory     []DomainSysInfoMemory    `xml:"memory"`
	OEMStrings *DomainSysInfoOEMStrings `xml:"oemStrings"`
}

type DomainSysInfoFWCfg struct {
	Entry []DomainSysInfoEntry `xml:"entry"`
}

type DomainSysInfo struct {
	SMBIOS *DomainSysInfoSMBIOS `xml:"-"`
	FWCfg  *DomainSysInfoFWCfg  `xml:"-"`
}

type DomainSysInfoEntry struct {
	Name  string `xml:"name,attr"`
	File  string `xml:"file,attr,omitempty"`
	Value string `xml:",chardata"`
}

type DomainBIOS struct {
	UseSerial     string `xml:"useserial,attr,omitempty"`
	RebootTimeout *int   `xml:"rebootTimeout,attr"`
}

type DomainLoader struct {
	Path      string `xml:",chardata"`
	Readonly  string `xml:"readonly,attr,omitempty"`
	Secure    string `xml:"secure,attr,omitempty"`
	Stateless string `xml:"stateless,attr,omitempty"`
	Type      string `xml:"type,attr,omitempty"`
	Format    string `xml:"format,attr,omitempty"`
}

type DomainACPI struct {
	Tables []DomainACPITable `xml:"table"`
}

type DomainACPITable struct {
	Type string `xml:"type,attr"`
	Path string `xml:",chardata"`
}

type DomainOSInitEnv struct {
	Name  string `xml:"name,attr"`
	Value string `xml:",chardata"`
}

type DomainOSFirmwareInfo struct {
	Features []DomainOSFirmwareFeature `xml:"feature"`
}

type DomainOSFirmwareFeature struct {
	Enabled string `xml:"enabled,attr,omitempty"`
	Name    string `xml:"name,attr,omitempty"`
}

type DomainOS struct {
	Type         *DomainOSType         `xml:"type"`
	Firmware     string                `xml:"firmware,attr,omitempty"`
	FirmwareInfo *DomainOSFirmwareInfo `xml:"firmware"`
	Init         string                `xml:"init,omitempty"`
	InitArgs     []string              `xml:"initarg"`
	InitEnv      []DomainOSInitEnv     `xml:"initenv"`
	InitDir      string                `xml:"initdir,omitempty"`
	InitUser     string                `xml:"inituser,omitempty"`
	InitGroup    string                `xml:"initgroup,omitempty"`
	Loader       *DomainLoader         `xml:"loader"`
	NVRam        *DomainNVRam          `xml:"nvram"`
	Kernel       string                `xml:"kernel,omitempty"`
	Initrd       string                `xml:"initrd,omitempty"`
	Cmdline      string                `xml:"cmdline,omitempty"`
	DTB          string                `xml:"dtb,omitempty"`
	ACPI         *DomainACPI           `xml:"acpi"`
	BootDevices  []DomainBootDevice    `xml:"boot"`
	BootMenu     *DomainBootMenu       `xml:"bootmenu"`
	BIOS         *DomainBIOS           `xml:"bios"`
	SMBios       *DomainSMBios         `xml:"smbios"`
}

type DomainResource struct {
	Partition    string                      `xml:"partition,omitempty"`
	FibreChannel *DomainResourceFibreChannel `xml:"fibrechannel"`
}

type DomainResourceFibreChannel struct {
	AppID string `xml:"appid,attr"`
}

type DomainVCPU struct {
	Placement string `xml:"placement,attr,omitempty"`
	CPUSet    string `xml:"cpuset,attr,omitempty"`
	Current   uint   `xml:"current,attr,omitempty"`
	Value     uint   `xml:",chardata"`
}

type DomainVCPUsVCPU struct {
	Id           *uint  `xml:"id,attr"`
	Enabled      string `xml:"enabled,attr,omitempty"`
	Hotpluggable string `xml:"hotpluggable,attr,omitempty"`
	Order        *uint  `xml:"order,attr"`
}

type DomainVCPUs struct {
	VCPU []DomainVCPUsVCPU `xml:"vcpu"`
}

type DomainCPUModel struct {
	Fallback string `xml:"fallback,attr,omitempty"`
	Value    string `xml:",chardata"`
	VendorID string `xml:"vendor_id,attr,omitempty"`
}

type DomainCPUTopology struct {
	Sockets  int `xml:"sockets,attr,omitempty"`
	Dies     int `xml:"dies,attr,omitempty"`
	Clusters int `xml:"clusters,attr,omitempty"`
	Cores    int `xml:"cores,attr,omitempty"`
	Threads  int `xml:"threads,attr,omitempty"`
}

type DomainCPUFeature struct {
	Policy string `xml:"policy,attr,omitempty"`
	Name   string `xml:"name,attr,omitempty"`
}

type DomainCPUCache struct {
	Level uint   `xml:"level,attr,omitempty"`
	Mode  string `xml:"mode,attr"`
}

type DomainCPUMaxPhysAddr struct {
	Mode  string `xml:"mode,attr"`
	Bits  uint   `xml:"bits,attr,omitempty"`
	Limit uint   `xml:"limit,attr,omitempty"`
}

type DomainCPU struct {
	XMLName            xml.Name              `xml:"cpu"`
	Match              string                `xml:"match,attr,omitempty"`
	Mode               string                `xml:"mode,attr,omitempty"`
	Check              string                `xml:"check,attr,omitempty"`
	Migratable         string                `xml:"migratable,attr,omitempty"`
	DeprecatedFeatures string                `xml:"deprecated_features,attr,omitempty"`
	Model              *DomainCPUModel       `xml:"model"`
	Vendor             string                `xml:"vendor,omitempty"`
	Topology           *DomainCPUTopology    `xml:"topology"`
	Cache              *DomainCPUCache       `xml:"cache"`
	MaxPhysAddr        *DomainCPUMaxPhysAddr `xml:"maxphysaddr"`
	Features           []DomainCPUFeature    `xml:"feature"`
	Numa               *DomainNuma           `xml:"numa"`
}

type DomainNuma struct {
	Cell          []DomainCell             `xml:"cell"`
	Interconnects *DomainNUMAInterconnects `xml:"interconnects"`
}

type DomainCell struct {
	ID        *uint                `xml:"id,attr"`
	CPUs      string               `xml:"cpus,attr,omitempty"`
	Memory    uint                 `xml:"memory,attr"`
	Unit      string               `xml:"unit,attr,omitempty"`
	MemAccess string               `xml:"memAccess,attr,omitempty"`
	Discard   string               `xml:"discard,attr,omitempty"`
	Distances *DomainCellDistances `xml:"distances"`
	Caches    []DomainCellCache    `xml:"cache"`
}

type DomainCellDistances struct {
	Siblings []DomainCellSibling `xml:"sibling"`
}

type DomainCellSibling struct {
	ID    uint `xml:"id,attr"`
	Value uint `xml:"value,attr"`
}

type DomainCellCache struct {
	Level         uint                `xml:"level,attr"`
	Associativity string              `xml:"associativity,attr"`
	Policy        string              `xml:"policy,attr"`
	Size          DomainCellCacheSize `xml:"size"`
	Line          DomainCellCacheLine `xml:"line"`
}

type DomainCellCacheSize struct {
	Value string `xml:"value,attr"`
	Unit  string `xml:"unit,attr"`
}

type DomainCellCacheLine struct {
	Value string `xml:"value,attr"`
	Unit  string `xml:"unit,attr"`
}

type DomainNUMAInterconnects struct {
	Latencies  []DomainNUMAInterconnectLatency   `xml:"latency"`
	Bandwidths []DomainNUMAInterconnectBandwidth `xml:"bandwidth"`
}

type DomainNUMAInterconnectLatency struct {
	Initiator uint   `xml:"initiator,attr"`
	Target    uint   `xml:"target,attr"`
	Cache     uint   `xml:"cache,attr,omitempty"`
	Type      string `xml:"type,attr"`
	Value     uint   `xml:"value,attr"`
}

type DomainNUMAInterconnectBandwidth struct {
	Initiator uint   `xml:"initiator,attr"`
	Target    uint   `xml:"target,attr"`
	Cache     uint   `xml:"cache,attr,omitempty"`
	Type      string `xml:"type,attr"`
	Value     uint   `xml:"value,attr"`
	Unit      string `xml:"unit,attr"`
}

type DomainClock struct {
	Offset     string        `xml:"offset,attr,omitempty"`
	Basis      string        `xml:"basis,attr,omitempty"`
	Adjustment string        `xml:"adjustment,attr,omitempty"`
	TimeZone   string        `xml:"timezone,attr,omitempty"`
	Start      uint          `xml:"start,attr,omitempty"`
	Timer      []DomainTimer `xml:"timer"`
}

type DomainTimer struct {
	Name       string              `xml:"name,attr"`
	Track      string              `xml:"track,attr,omitempty"`
	TickPolicy string              `xml:"tickpolicy,attr,omitempty"`
	CatchUp    *DomainTimerCatchUp `xml:"catchup"`
	Frequency  uint64              `xml:"frequency,attr,omitempty"`
	Mode       string              `xml:"mode,attr,omitempty"`
	Present    string              `xml:"present,attr,omitempty"`
}

type DomainTimerCatchUp struct {
	Threshold uint `xml:"threshold,attr,omitempty"`
	Slew      uint `xml:"slew,attr,omitempty"`
	Limit     uint `xml:"limit,attr,omitempty"`
}

type DomainFeature struct {
}

type DomainFeatureState struct {
	State string `xml:"state,attr,omitempty"`
}

type DomainFeatureAPIC struct {
	EOI string `xml:"eoi,attr,omitempty"`
}

type DomainFeatureHyperVVendorId struct {
	DomainFeatureState
	Value string `xml:"value,attr,omitempty"`
}

type DomainFeatureHyperVSpinlocks struct {
	DomainFeatureState
	Retries uint `xml:"retries,attr,omitempty"`
}

type DomainFeatureHyperVSTimer struct {
	DomainFeatureState
	Direct *DomainFeatureState `xml:"direct"`
}

type DomainFeatureHyperVTLBFlush struct {
	DomainFeatureState
	Direct   *DomainFeatureState `xml:"direct"`
	Extended *DomainFeatureState `xml:"extended"`
}

type DomainFeatureHyperV struct {
	DomainFeature
	Mode            string                        `xml:"mode,attr,omitempty"`
	Relaxed         *DomainFeatureState           `xml:"relaxed"`
	VAPIC           *DomainFeatureState           `xml:"vapic"`
	Spinlocks       *DomainFeatureHyperVSpinlocks `xml:"spinlocks"`
	VPIndex         *DomainFeatureState           `xml:"vpindex"`
	Runtime         *DomainFeatureState           `xml:"runtime"`
	Synic           *DomainFeatureState           `xml:"synic"`
	STimer          *DomainFeatureHyperVSTimer    `xml:"stimer"`
	Reset           *DomainFeatureState           `xml:"reset"`
	VendorId        *DomainFeatureHyperVVendorId  `xml:"vendor_id"`
	Frequencies     *DomainFeatureState           `xml:"frequencies"`
	ReEnlightenment *DomainFeatureState           `xml:"reenlightenment"`
	TLBFlush        *DomainFeatureHyperVTLBFlush  `xml:"tlbflush"`
	IPI             *DomainFeatureState           `xml:"ipi"`
	EVMCS           *DomainFeatureState           `xml:"evmcs"`
	AVIC            *DomainFeatureState           `xml:"avic"`
	EMSRBitmap      *DomainFeatureState           `xml:"emsr_bitmap"`
	XMMInput        *DomainFeatureState           `xml:"xmm_input"`
}

type DomainFeatureKVMDirtyRing struct {
	DomainFeatureState
	Size uint `xml:"size,attr,omitempty"`
}

type DomainFeatureKVM struct {
	Hidden        *DomainFeatureState        `xml:"hidden"`
	HintDedicated *DomainFeatureState        `xml:"hint-dedicated"`
	PollControl   *DomainFeatureState        `xml:"poll-control"`
	PVIPI         *DomainFeatureState        `xml:"pv-ipi"`
	DirtyRing     *DomainFeatureKVMDirtyRing `xml:"dirty-ring"`
}

type DomainFeatureTCGTBCache struct {
	Unit string `xml:"unit,attr,omitempty"`
	Size uint   `xml:",chardata"`
}

type DomainFeatureTCG struct {
	TBCache *DomainFeatureTCGTBCache `xml:"tb-cache"`
}

type DomainFeatureXenPassthrough struct {
	State string `xml:"state,attr,omitempty"`
	Mode  string `xml:"mode,attr,omitempty"`
}

type DomainFeatureXenE820Host struct {
	State string `xml:"state,attr"`
}

type DomainFeatureXen struct {
	E820Host    *DomainFeatureXenE820Host    `xml:"e820_host"`
	Passthrough *DomainFeatureXenPassthrough `xml:"passthrough"`
}

type DomainFeatureGIC struct {
	Version string `xml:"version,attr,omitempty"`
}

type DomainFeatureIOAPIC struct {
	Driver string `xml:"driver,attr,omitempty"`
}

type DomainFeatureHPT struct {
	Resizing    string                    `xml:"resizing,attr,omitempty"`
	MaxPageSize *DomainFeatureHPTPageSize `xml:"maxpagesize"`
}

type DomainFeatureHPTPageSize struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value string `xml:",chardata"`
}

type DomainFeatureSMM struct {
	State string                `xml:"state,attr,omitempty"`
	TSeg  *DomainFeatureSMMTSeg `xml:"tseg"`
}

type DomainFeatureSMMTSeg struct {
	Unit  string `xml:"unit,attr,omitempty"`
	Value uint   `xml:",chardata"`
}

type DomainFeatureCapability struct {
	State string `xml:"state,attr,omitempty"`
}

type DomainLaunchSecurity struct {
	SEV    *DomainLaunchSecuritySEV    `xml:"-"`
	SEVSNP *DomainLaunchSecuritySEVSNP `xml:"-"`
	S390PV *DomainLaunchSecurityS390PV `xml:"-"`
}

type DomainLaunchSecuritySEV struct {
	KernelHashes    string `xml:"kernelHashes,attr,omitempty"`
	CBitPos         *uint  `xml:"cbitpos"`
	ReducedPhysBits *uint  `xml:"reducedPhysBits"`
	Policy          *uint  `xml:"policy"`
	DHCert          string `xml:"dhCert"`
	Session         string `xml:"sesion"`
}

type DomainLaunchSecuritySEVSNP struct {
	KernelHashes            string  `xml:"kernelHashes,attr,omitempty"`
	AuthorKey               string  `xml:"authorKey,attr,omitempty"`
	VCEK                    string  `xml:"vcek,attr,omitempty"`
	CBitPos                 *uint   `xml:"cbitpos"`
	ReducedPhysBits         *uint   `xml:"reducedPhysBits"`
	Policy                  *uint64 `xml:"policy"`
	GuestVisibleWorkarounds string  `xml:"guestVisibleWorkarounds,omitempty"`
	IDBlock                 string  `xml:"idBlock,omitempty"`
	IDAuth                  string  `xml:"idAuth,omitempty"`
	HostData                string  `xml:"hostData,omitempty"`
}

type DomainLaunchSecurityS390PV struct {
}

type DomainFeatureCapabilities struct {
	Policy         string                   `xml:"policy,attr,omitempty"`
	AuditControl   *DomainFeatureCapability `xml:"audit_control"`
	AuditWrite     *DomainFeatureCapability `xml:"audit_write"`
	BlockSuspend   *DomainFeatureCapability `xml:"block_suspend"`
	Chown          *DomainFeatureCapability `xml:"chown"`
	DACOverride    *DomainFeatureCapability `xml:"dac_override"`
	DACReadSearch  *DomainFeatureCapability `xml:"dac_read_Search"`
	FOwner         *DomainFeatureCapability `xml:"fowner"`
	FSetID         *DomainFeatureCapability `xml:"fsetid"`
	IPCLock        *DomainFeatureCapability `xml:"ipc_lock"`
	IPCOwner       *DomainFeatureCapability `xml:"ipc_owner"`
	Kill           *DomainFeatureCapability `xml:"kill"`
	Lease          *DomainFeatureCapability `xml:"lease"`
	LinuxImmutable *DomainFeatureCapability `xml:"linux_immutable"`
	MACAdmin       *DomainFeatureCapability `xml:"mac_admin"`
	MACOverride    *DomainFeatureCapability `xml:"mac_override"`
	MkNod          *DomainFeatureCapability `xml:"mknod"`
	NetAdmin       *DomainFeatureCapability `xml:"net_admin"`
	NetBindService *DomainFeatureCapability `xml:"net_bind_service"`
	NetBroadcast   *DomainFeatureCapability `xml:"net_broadcast"`
	NetRaw         *DomainFeatureCapability `xml:"net_raw"`
	SetGID         *DomainFeatureCapability `xml:"setgid"`
	SetFCap        *DomainFeatureCapability `xml:"setfcap"`
	SetPCap        *DomainFeatureCapability `xml:"setpcap"`
	SetUID         *DomainFeatureCapability `xml:"setuid"`
	SysAdmin       *DomainFeatureCapability `xml:"sys_admin"`
	SysBoot        *DomainFeatureCapability `xml:"sys_boot"`
	SysChRoot      *DomainFeatureCapability `xml:"sys_chroot"`
	SysModule      *DomainFeatureCapability `xml:"sys_module"`
	SysNice        *DomainFeatureCapability `xml:"sys_nice"`
	SysPAcct       *DomainFeatureCapability `xml:"sys_pacct"`
	SysPTrace      *DomainFeatureCapability `xml:"sys_ptrace"`
	SysRawIO       *DomainFeatureCapability `xml:"sys_rawio"`
	SysResource    *DomainFeatureCapability `xml:"sys_resource"`
	SysTime        *DomainFeatureCapability `xml:"sys_time"`
	SysTTYCnofig   *DomainFeatureCapability `xml:"sys_tty_config"`
	SysLog         *DomainFeatureCapability `xml:"syslog"`
	WakeAlarm      *DomainFeatureCapability `xml:"wake_alarm"`
}

type DomainFeatureMSRS struct {
	Unknown string `xml:"unknown,attr"`
}

type DomainFeatureCFPC struct {
	Value string `xml:"value,attr"`
}

type DomainFeatureSBBC struct {
	Value string `xml:"value,attr"`
}

type DomainFeatureIBS struct {
	Value string `xml:"value,attr"`
}

type DomainFeatureAsyncTeardown struct {
	Enabled string `xml:"enabled,attr,omitempty"`
}

type DomainFeatureList struct {
	PAE           *DomainFeature              `xml:"pae"`
	ACPI          *DomainFeature              `xml:"acpi"`
	APIC          *DomainFeatureAPIC          `xml:"apic"`
	HAP           *DomainFeatureState         `xml:"hap"`
	Viridian      *DomainFeature              `xml:"viridian"`
	PrivNet       *DomainFeature              `xml:"privnet"`
	HyperV        *DomainFeatureHyperV        `xml:"hyperv"`
	KVM           *DomainFeatureKVM           `xml:"kvm"`
	Xen           *DomainFeatureXen           `xml:"xen"`
	PVSpinlock    *DomainFeatureState         `xml:"pvspinlock"`
	PMU           *DomainFeatureState         `xml:"pmu"`
	VMPort        *DomainFeatureState         `xml:"vmport"`
	GIC           *DomainFeatureGIC           `xml:"gic"`
	SMM           *DomainFeatureSMM           `xml:"smm"`
	IOAPIC        *DomainFeatureIOAPIC        `xml:"ioapic"`
	HPT           *DomainFeatureHPT           `xml:"hpt"`
	HTM           *DomainFeatureState         `xml:"htm"`
	NestedHV      *DomainFeatureState         `xml:"nested-hv"`
	Capabilities  *DomainFeatureCapabilities  `xml:"capabilities"`
	VMCoreInfo    *DomainFeatureState         `xml:"vmcoreinfo"`
	MSRS          *DomainFeatureMSRS          `xml:"msrs"`
	CCFAssist     *DomainFeatureState         `xml:"ccf-assist"`
	CFPC          *DomainFeatureCFPC          `xml:"cfpc"`
	SBBC          *DomainFeatureSBBC          `xml:"sbbc"`
	IBS           *DomainFeatureIBS           `xml:"ibs"`
	TCG           *DomainFeatureTCG           `xml:"tcg"`
	AsyncTeardown *DomainFeatureAsyncTeardown `xml:"async-teardown"`
	RAS           *DomainFeatureState         `xml:"ras"`
	PS2           *DomainFeatureState         `xml:"ps2"`
}

type DomainCPUTuneShares struct {
	Value uint `xml:",chardata"`
}

type DomainCPUTunePeriod struct {
	Value uint64 `xml:",chardata"`
}

type DomainCPUTuneQuota struct {
	Value int64 `xml:",chardata"`
}

type DomainCPUTuneVCPUPin struct {
	VCPU   uint   `xml:"vcpu,attr"`
	CPUSet string `xml:"cpuset,attr"`
}

type DomainCPUTuneEmulatorPin struct {
	CPUSet string `xml:"cpuset,attr"`
}

type DomainCPUTuneIOThreadPin struct {
	IOThread uint   `xml:"iothread,attr"`
	CPUSet   string `xml:"cpuset,attr"`
}

type DomainCPUTuneVCPUSched struct {
	VCPUs     string `xml:"vcpus,attr"`
	Scheduler string `xml:"scheduler,attr,omitempty"`
	Priority  *int   `xml:"priority,attr"`
}

type DomainCPUTuneIOThreadSched struct {
	IOThreads string `xml:"iothreads,attr"`
	Scheduler string `xml:"scheduler,attr,omitempty"`
	Priority  *int   `xml:"priority,attr"`
}

type DomainCPUTuneEmulatorSched struct {
	Scheduler string `xml:"scheduler,attr,omitempty"`
	Priority  *int   `xml:"priority,attr"`
}

type DomainCPUCacheTune struct {
	VCPUs   string                      `xml:"vcpus,attr,omitempty"`
	ID      string                      `xml:"id,attr,omitempty"`
	Cache   []DomainCPUCacheTuneCache   `xml:"cache"`
	Monitor []DomainCPUCacheTuneMonitor `xml:"monitor"`
}

type DomainCPUCacheTuneCache struct {
	ID    uint   `xml:"id,attr"`
	Level uint   `xml:"level,attr"`
	Type  string `xml:"type,attr"`
	Size  uint   `xml:"size,attr"`
	Unit  string `xml:"unit,attr"`
}

type DomainCPUCacheTuneMonitor struct {
	Level uint   `xml:"level,attr,omitempty"`
	VCPUs string `xml:"vcpus,attr,omitempty"`
}

type DomainCPUMemoryTune struct {
	VCPUs   string                       `xml:"vcpus,attr"`
	Nodes   []DomainCPUMemoryTuneNode    `xml:"node"`
	Monitor []DomainCPUMemoryTuneMonitor `xml:"monitor"`
}

type DomainCPUMemoryTuneNode struct {
	ID        uint `xml:"id,attr"`
	Bandwidth uint `xml:"bandwidth,attr"`
}

type DomainCPUMemoryTuneMonitor struct {
	Level uint   `xml:"level,attr,omitempty"`
	VCPUs string `xml:"vcpus,attr,omitempty"`
}

type DomainCPUTune struct {
	Shares         *DomainCPUTuneShares         `xml:"shares"`
	Period         *DomainCPUTunePeriod         `xml:"period"`
	Quota          *DomainCPUTuneQuota          `xml:"quota"`
	GlobalPeriod   *DomainCPUTunePeriod         `xml:"global_period"`
	GlobalQuota    *DomainCPUTuneQuota          `xml:"global_quota"`
	EmulatorPeriod *DomainCPUTunePeriod         `xml:"emulator_period"`
	EmulatorQuota  *DomainCPUTuneQuota          `xml:"emulator_quota"`
	IOThreadPeriod *DomainCPUTunePeriod         `xml:"iothread_period"`
	IOThreadQuota  *DomainCPUTuneQuota          `xml:"iothread_quota"`
	VCPUPin        []DomainCPUTuneVCPUPin       `xml:"vcpupin"`
	EmulatorPin    *DomainCPUTuneEmulatorPin    `xml:"emulatorpin"`
	IOThreadPin    []DomainCPUTuneIOThreadPin   `xml:"iothreadpin"`
	VCPUSched      []DomainCPUTuneVCPUSched     `xml:"vcpusched"`
	EmulatorSched  *DomainCPUTuneEmulatorSched  `xml:"emulatorsched"`
	IOThreadSched  []DomainCPUTuneIOThreadSched `xml:"iothreadsched"`
	CacheTune      []DomainCPUCacheTune         `xml:"cachetune"`
	MemoryTune     []DomainCPUMemoryTune        `xml:"memorytune"`
}

type DomainQEMUCommandlineArg struct {
	Value string `xml:"value,attr"`
}

type DomainQEMUCommandlineEnv struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr,omitempty"`
}

type DomainQEMUCommandline struct {
	XMLName xml.Name                   `xml:"http://libvirt.org/schemas/domain/qemu/1.0 commandline"`
	Args    []DomainQEMUCommandlineArg `xml:"arg"`
	Envs    []DomainQEMUCommandlineEnv `xml:"env"`
}

type DomainQEMUCapabilitiesEntry struct {
	Name string `xml:"capability,attr"`
}
type DomainQEMUCapabilities struct {
	XMLName xml.Name                      `xml:"http://libvirt.org/schemas/domain/qemu/1.0 capabilities"`
	Add     []DomainQEMUCapabilitiesEntry `xml:"add"`
	Del     []DomainQEMUCapabilitiesEntry `xml:"del"`
}

type DomainQEMUDeprecation struct {
	XMLName  xml.Name `xml:"http://libvirt.org/schemas/domain/qemu/1.0 deprecation"`
	Behavior string   `xml:"behavior,attr,omitempty"`
}

type DomainQEMUOverride struct {
	XMLName xml.Name                   `xml:"http://libvirt.org/schemas/domain/qemu/1.0 override"`
	Devices []DomainQEMUOverrideDevice `xml:"device"`
}

type DomainQEMUOverrideDevice struct {
	Alias    string                     `xml:"alias,attr"`
	Frontend DomainQEMUOverrideFrontend `xml:"frontend"`
}

type DomainQEMUOverrideFrontend struct {
	Properties []DomainQEMUOverrideProperty `xml:"property"`
}

type DomainQEMUOverrideProperty struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr,omitempty"`
	Value string `xml:"value,attr,omitempty"`
}

type DomainLXCNamespace struct {
	XMLName  xml.Name               `xml:"http://libvirt.org/schemas/domain/lxc/1.0 namespace"`
	ShareNet *DomainLXCNamespaceMap `xml:"sharenet"`
	ShareIPC *DomainLXCNamespaceMap `xml:"shareipc"`
	ShareUTS *DomainLXCNamespaceMap `xml:"shareuts"`
}

type DomainLXCNamespaceMap struct {
	Type  string `xml:"type,attr"`
	Value string `xml:"value,attr"`
}

type DomainBHyveCommandlineArg struct {
	Value string `xml:"value,attr"`
}

type DomainBHyveCommandlineEnv struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr,omitempty"`
}

type DomainBHyveCommandline struct {
	XMLName xml.Name                    `xml:"http://libvirt.org/schemas/domain/bhyve/1.0 commandline"`
	Args    []DomainBHyveCommandlineArg `xml:"arg"`
	Envs    []DomainBHyveCommandlineEnv `xml:"env"`
}

type DomainXenCommandlineArg struct {
	Value string `xml:"value,attr"`
}

type DomainXenCommandline struct {
	XMLName xml.Name                  `xml:"http://libvirt.org/schemas/domain/xen/1.0 commandline"`
	Args    []DomainXenCommandlineArg `xml:"arg"`
}

type DomainBlockIOTune struct {
	Weight uint                      `xml:"weight,omitempty"`
	Device []DomainBlockIOTuneDevice `xml:"device"`
}

type DomainBlockIOTuneDevice struct {
	Path          string `xml:"path"`
	Weight        uint   `xml:"weight,omitempty"`
	ReadIopsSec   uint   `xml:"read_iops_sec,omitempty"`
	WriteIopsSec  uint   `xml:"write_iops_sec,omitempty"`
	ReadBytesSec  uint   `xml:"read_bytes_sec,omitempty"`
	WriteBytesSec uint   `xml:"write_bytes_sec,omitempty"`
}

type DomainPM struct {
	SuspendToMem  *DomainPMPolicy `xml:"suspend-to-mem"`
	SuspendToDisk *DomainPMPolicy `xml:"suspend-to-disk"`
}

type DomainPMPolicy struct {
	Enabled string `xml:"enabled,attr"`
}

type DomainSecLabel struct {
	Type       string `xml:"type,attr,omitempty"`
	Model      string `xml:"model,attr,omitempty"`
	Relabel    string `xml:"relabel,attr,omitempty"`
	Label      string `xml:"label,omitempty"`
	ImageLabel string `xml:"imagelabel,omitempty"`
	BaseLabel  string `xml:"baselabel,omitempty"`
}

type DomainDeviceSecLabel struct {
	Model     string `xml:"model,attr,omitempty"`
	LabelSkip string `xml:"labelskip,attr,omitempty"`
	Relabel   string `xml:"relabel,attr,omitempty"`
	Label     string `xml:"label,omitempty"`
}

type DomainNUMATune struct {
	Memory   *DomainNUMATuneMemory   `xml:"memory"`
	MemNodes []DomainNUMATuneMemNode `xml:"memnode"`
}

type DomainNUMATuneMemory struct {
	Mode      string `xml:"mode,attr,omitempty"`
	Nodeset   string `xml:"nodeset,attr,omitempty"`
	Placement string `xml:"placement,attr,omitempty"`
}

type DomainNUMATuneMemNode struct {
	CellID  uint   `xml:"cellid,attr"`
	Mode    string `xml:"mode,attr"`
	Nodeset string `xml:"nodeset,attr"`
}

type DomainIOThreadIDs struct {
	IOThreads []DomainIOThread `xml:"iothread"`
}

type DomainIOThreadPoll struct {
	Max    *uint `xml:"max,attr"`
	Grow   *uint `xml:"grow,attr"`
	Shrink *uint `xml:"shrink,attr"`
}

type DomainIOThread struct {
	ID      uint                `xml:"id,attr"`
	PoolMin *uint               `xml:"thread_pool_min,attr"`
	PoolMax *uint               `xml:"thread_pool_max,attr"`
	Poll    *DomainIOThreadPoll `xml:"poll"`
}

type DomainDefaultIOThread struct {
	PoolMin *uint `xml:"thread_pool_min,attr"`
	PoolMax *uint `xml:"thread_pool_max,attr"`
}

type DomainKeyWrap struct {
	Ciphers []DomainKeyWrapCipher `xml:"cipher"`
}

type DomainKeyWrapCipher struct {
	Name  string `xml:"name,attr"`
	State string `xml:"state,attr"`
}

type DomainIDMap struct {
	UIDs []DomainIDMapRange `xml:"uid"`
	GIDs []DomainIDMapRange `xml:"gid"`
}

type DomainIDMapRange struct {
	Start  uint `xml:"start,attr"`
	Target uint `xml:"target,attr"`
	Count  uint `xml:"count,attr"`
}

type DomainMemoryTuneLimit struct {
	Value uint64 `xml:",chardata"`
	Unit  string `xml:"unit,attr,omitempty"`
}

type DomainMemoryTune struct {
	HardLimit     *DomainMemoryTuneLimit `xml:"hard_limit"`
	SoftLimit     *DomainMemoryTuneLimit `xml:"soft_limit"`
	MinGuarantee  *DomainMemoryTuneLimit `xml:"min_guarantee"`
	SwapHardLimit *DomainMemoryTuneLimit `xml:"swap_hard_limit"`
}

type DomainMetadata struct {
	XML string `xml:",innerxml"`
}

type DomainVMWareDataCenterPath struct {
	XMLName xml.Name `xml:"http://libvirt.org/schemas/domain/vmware/1.0 datacenterpath"`
	Value   string   `xml:",chardata"`
}

type DomainPerf struct {
	Events []DomainPerfEvent `xml:"event"`
}

type DomainPerfEvent struct {
	Name    string `xml:"name,attr"`
	Enabled string `xml:"enabled,attr"`
}

type DomainGenID struct {
	Value string `xml:",chardata"`
}

// NB, try to keep the order of fields in this struct
// matching the order of XML elements that libvirt
// will generate when dumping XML.
type Domain struct {
	XMLName         xml.Name               `xml:"domain"`
	Type            string                 `xml:"type,attr,omitempty"`
	ID              *int                   `xml:"id,attr"`
	Name            string                 `xml:"name,omitempty"`
	UUID            string                 `xml:"uuid,omitempty"`
	GenID           *DomainGenID           `xml:"genid"`
	Title           string                 `xml:"title,omitempty"`
	Description     string                 `xml:"description,omitempty"`
	Metadata        *DomainMetadata        `xml:"metadata"`
	MaximumMemory   *DomainMaxMemory       `xml:"maxMemory"`
	Memory          *DomainMemory          `xml:"memory"`
	CurrentMemory   *DomainCurrentMemory   `xml:"currentMemory"`
	BlockIOTune     *DomainBlockIOTune     `xml:"blkiotune"`
	MemoryTune      *DomainMemoryTune      `xml:"memtune"`
	MemoryBacking   *DomainMemoryBacking   `xml:"memoryBacking"`
	VCPU            *DomainVCPU            `xml:"vcpu"`
	VCPUs           *DomainVCPUs           `xml:"vcpus"`
	IOThreads       uint                   `xml:"iothreads,omitempty"`
	IOThreadIDs     *DomainIOThreadIDs     `xml:"iothreadids"`
	DefaultIOThread *DomainDefaultIOThread `xml:"defaultiothread"`
	CPUTune         *DomainCPUTune         `xml:"cputune"`
	NUMATune        *DomainNUMATune        `xml:"numatune"`
	Resource        *DomainResource        `xml:"resource"`
	SysInfo         []DomainSysInfo        `xml:"sysinfo"`
	Bootloader      string                 `xml:"bootloader,omitempty"`
	BootloaderArgs  string                 `xml:"bootloader_args,omitempty"`
	OS              *DomainOS              `xml:"os"`
	IDMap           *DomainIDMap           `xml:"idmap"`
	Features        *DomainFeatureList     `xml:"features"`
	CPU             *DomainCPU             `xml:"cpu"`
	Clock           *DomainClock           `xml:"clock"`
	OnPoweroff      string                 `xml:"on_poweroff,omitempty"`
	OnReboot        string                 `xml:"on_reboot,omitempty"`
	OnCrash         string                 `xml:"on_crash,omitempty"`
	PM              *DomainPM              `xml:"pm"`
	Perf            *DomainPerf            `xml:"perf"`
	Devices         *DomainDeviceList      `xml:"devices"`
	SecLabel        []DomainSecLabel       `xml:"seclabel"`
	KeyWrap         *DomainKeyWrap         `xml:"keywrap"`
	LaunchSecurity  *DomainLaunchSecurity  `xml:"launchSecurity"`

	/* Hypervisor namespaces must all be last */
	QEMUCommandline      *DomainQEMUCommandline
	QEMUCapabilities     *DomainQEMUCapabilities
	QEMUOverride         *DomainQEMUOverride
	QEMUDeprecation      *DomainQEMUDeprecation
	LXCNamespace         *DomainLXCNamespace
	BHyveCommandline     *DomainBHyveCommandline
	VMWareDataCenterPath *DomainVMWareDataCenterPath
	XenCommandline       *DomainXenCommandline
}

func (d *Domain) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *Domain) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainController DomainController

type domainControllerPCI struct {
	DomainControllerPCI
	domainController
}

type domainControllerUSB struct {
	DomainControllerUSB
	domainController
}

type domainControllerVirtIOSerial struct {
	DomainControllerVirtIOSerial
	domainController
}

type domainControllerXenBus struct {
	DomainControllerXenBus
	domainController
}

func (a *DomainControllerPCITarget) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "chassisNr", a.ChassisNr, "%d")
	marshalUintAttr(&start, "chassis", a.Chassis, "%d")
	marshalUintAttr(&start, "port", a.Port, "%d")
	marshalUintAttr(&start, "busNr", a.BusNr, "%d")
	marshalUintAttr(&start, "index", a.Index, "%d")
	marshalUint64Attr(&start, "memReserve", a.MemReserve, "%d")
	if a.Hotplug != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "hotplug"}, a.Hotplug,
		})
	}
	e.EncodeToken(start)
	if a.NUMANode != nil {
		node := xml.StartElement{
			Name: xml.Name{Local: "node"},
		}
		e.EncodeToken(node)
		e.EncodeToken(xml.CharData(fmt.Sprintf("%d", *a.NUMANode)))
		e.EncodeToken(node.End())
	}
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainControllerPCITarget) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "chassisNr" {
			if err := unmarshalUintAttr(attr.Value, &a.ChassisNr, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "chassis" {
			if err := unmarshalUintAttr(attr.Value, &a.Chassis, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "port" {
			if err := unmarshalUintAttr(attr.Value, &a.Port, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "busNr" {
			if err := unmarshalUintAttr(attr.Value, &a.BusNr, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "index" {
			if err := unmarshalUintAttr(attr.Value, &a.Index, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "memReserve" {
			if err := unmarshalUint64Attr(attr.Value, &a.MemReserve, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "hotplug" {
			a.Hotplug = attr.Value
		}
	}
	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "node" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					val, err := strconv.ParseUint(string(data), 10, 64)
					if err != nil {
						return err
					}
					vali := uint(val)
					a.NUMANode = &vali
				}
			}
		}
	}
	return nil
}

func (a *DomainController) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "controller"
	if a.Type == "pci" {
		pci := domainControllerPCI{}
		pci.domainController = domainController(*a)
		if a.PCI != nil {
			pci.DomainControllerPCI = *a.PCI
		}
		return e.EncodeElement(pci, start)
	} else if a.Type == "usb" {
		usb := domainControllerUSB{}
		usb.domainController = domainController(*a)
		if a.USB != nil {
			usb.DomainControllerUSB = *a.USB
		}
		return e.EncodeElement(usb, start)
	} else if a.Type == "virtio-serial" {
		vioserial := domainControllerVirtIOSerial{}
		vioserial.domainController = domainController(*a)
		if a.VirtIOSerial != nil {
			vioserial.DomainControllerVirtIOSerial = *a.VirtIOSerial
		}
		return e.EncodeElement(vioserial, start)
	} else if a.Type == "xenbus" {
		xenbus := domainControllerXenBus{}
		xenbus.domainController = domainController(*a)
		if a.XenBus != nil {
			xenbus.DomainControllerXenBus = *a.XenBus
		}
		return e.EncodeElement(xenbus, start)
	} else {
		gen := domainController(*a)
		return e.EncodeElement(gen, start)
	}
}

func getAttr(attrs []xml.Attr, name string) (string, bool) {
	for _, attr := range attrs {
		if attr.Name.Local == name {
			return attr.Value, true
		}
	}
	return "", false
}

func (a *DomainController) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain controller")
	}
	if typ == "pci" {
		var pci domainControllerPCI
		err := d.DecodeElement(&pci, &start)
		if err != nil {
			return err
		}
		*a = DomainController(pci.domainController)
		a.PCI = &pci.DomainControllerPCI
		return nil
	} else if typ == "usb" {
		var usb domainControllerUSB
		err := d.DecodeElement(&usb, &start)
		if err != nil {
			return err
		}
		*a = DomainController(usb.domainController)
		a.USB = &usb.DomainControllerUSB
		return nil
	} else if typ == "virtio-serial" {
		var vioserial domainControllerVirtIOSerial
		err := d.DecodeElement(&vioserial, &start)
		if err != nil {
			return err
		}
		*a = DomainController(vioserial.domainController)
		a.VirtIOSerial = &vioserial.DomainControllerVirtIOSerial
		return nil
	} else if typ == "xenbus" {
		var xenbus domainControllerXenBus
		err := d.DecodeElement(&xenbus, &start)
		if err != nil {
			return err
		}
		*a = DomainController(xenbus.domainController)
		a.XenBus = &xenbus.DomainControllerXenBus
		return nil
	} else {
		var gen domainController
		err := d.DecodeElement(&gen, &start)
		if err != nil {
			return err
		}
		*a = DomainController(gen)
		return nil
	}
}

func (d *DomainGraphic) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainGraphic) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainController) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainController) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainDiskReservationsSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "source"
	src := DomainChardevSource(*a)
	typ := getChardevSourceType(&src)
	if typ != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, typ,
		})
	}
	return e.EncodeElement(&src, start)
}

func (a *DomainDiskReservationsSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "unix"
	}
	src := createChardevSource(typ)
	err := d.DecodeElement(&src, &start)
	if err != nil {
		return err
	}
	*a = DomainDiskReservationsSource(*src)
	return nil
}

func (a *DomainDiskSourceVHostUser) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "source"
	src := DomainChardevSource(*a)
	typ := getChardevSourceType(&src)
	if typ != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, typ,
		})
	}
	return e.EncodeElement(&src, start)
}

func (a *DomainDiskSourceVHostUser) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "unix"
	}
	src := createChardevSource(typ)
	err := d.DecodeElement(&src, &start)
	if err != nil {
		return err
	}
	*a = DomainDiskSourceVHostUser(*src)
	return nil
}

type domainDiskSource DomainDiskSource

type domainDiskSourceFile struct {
	DomainDiskSourceFile
	domainDiskSource
}

type domainDiskSourceBlock struct {
	DomainDiskSourceBlock
	domainDiskSource
}

type domainDiskSourceDir struct {
	DomainDiskSourceDir
	domainDiskSource
}

type domainDiskSourceNetwork struct {
	DomainDiskSourceNetwork
	domainDiskSource
}

type domainDiskSourceVolume struct {
	DomainDiskSourceVolume
	domainDiskSource
}

type domainDiskSourceNVMEPCI struct {
	DomainDiskSourceNVMEPCI
	domainDiskSource
}

type domainDiskSourceVHostUser struct {
	DomainDiskSourceVHostUser
	domainDiskSource
}

type domainDiskSourceVHostVDPA struct {
	DomainDiskSourceVHostVDPA
	domainDiskSource
}

func (a *DomainDiskSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.File != nil {
		if a.StartupPolicy == "" && a.Encryption == nil && a.File.File == "" {
			return nil
		}
		file := domainDiskSourceFile{
			*a.File, domainDiskSource(*a),
		}
		return e.EncodeElement(&file, start)
	} else if a.Block != nil {
		if a.StartupPolicy == "" && a.Encryption == nil && a.Block.Dev == "" {
			return nil
		}
		block := domainDiskSourceBlock{
			*a.Block, domainDiskSource(*a),
		}
		return e.EncodeElement(&block, start)
	} else if a.Dir != nil {
		dir := domainDiskSourceDir{
			*a.Dir, domainDiskSource(*a),
		}
		return e.EncodeElement(&dir, start)
	} else if a.Network != nil {
		network := domainDiskSourceNetwork{
			*a.Network, domainDiskSource(*a),
		}
		return e.EncodeElement(&network, start)
	} else if a.Volume != nil {
		if a.StartupPolicy == "" && a.Encryption == nil && a.Volume.Pool == "" && a.Volume.Volume == "" {
			return nil
		}
		volume := domainDiskSourceVolume{
			*a.Volume, domainDiskSource(*a),
		}
		return e.EncodeElement(&volume, start)
	} else if a.NVME != nil {
		if a.NVME.PCI != nil {
			nvme := domainDiskSourceNVMEPCI{
				*a.NVME.PCI, domainDiskSource(*a),
			}
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "pci",
			})
			return e.EncodeElement(&nvme, start)
		}
	} else if a.VHostUser != nil {
		vhost := domainDiskSourceVHostUser{
			*a.VHostUser, domainDiskSource(*a),
		}
		return e.EncodeElement(&vhost, start)
	} else if a.VHostVDPA != nil {
		vhost := domainDiskSourceVHostVDPA{
			*a.VHostVDPA, domainDiskSource(*a),
		}
		return e.EncodeElement(&vhost, start)
	}
	return nil
}

func (a *DomainDiskSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.File != nil {
		file := domainDiskSourceFile{
			*a.File, domainDiskSource(*a),
		}
		err := d.DecodeElement(&file, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(file.domainDiskSource)
		a.File = &file.DomainDiskSourceFile
	} else if a.Block != nil {
		block := domainDiskSourceBlock{
			*a.Block, domainDiskSource(*a),
		}
		err := d.DecodeElement(&block, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(block.domainDiskSource)
		a.Block = &block.DomainDiskSourceBlock
	} else if a.Dir != nil {
		dir := domainDiskSourceDir{
			*a.Dir, domainDiskSource(*a),
		}
		err := d.DecodeElement(&dir, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(dir.domainDiskSource)
		a.Dir = &dir.DomainDiskSourceDir
	} else if a.Network != nil {
		network := domainDiskSourceNetwork{
			*a.Network, domainDiskSource(*a),
		}
		err := d.DecodeElement(&network, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(network.domainDiskSource)
		a.Network = &network.DomainDiskSourceNetwork
	} else if a.Volume != nil {
		volume := domainDiskSourceVolume{
			*a.Volume, domainDiskSource(*a),
		}
		err := d.DecodeElement(&volume, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(volume.domainDiskSource)
		a.Volume = &volume.DomainDiskSourceVolume
	} else if a.NVME != nil {
		typ, ok := getAttr(start.Attr, "type")
		if !ok {
			return fmt.Errorf("Missing nvme source type")
		}
		if typ == "pci" {
			a.NVME.PCI = &DomainDiskSourceNVMEPCI{}
			nvme := domainDiskSourceNVMEPCI{
				*a.NVME.PCI, domainDiskSource(*a),
			}
			err := d.DecodeElement(&nvme, &start)
			if err != nil {
				return err
			}
			*a = DomainDiskSource(nvme.domainDiskSource)
			a.NVME.PCI = &nvme.DomainDiskSourceNVMEPCI
		}
	} else if a.VHostUser != nil {
		vhost := domainDiskSourceVHostUser{
			*a.VHostUser, domainDiskSource(*a),
		}
		err := d.DecodeElement(&vhost, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(vhost.domainDiskSource)
		a.VHostUser = &vhost.DomainDiskSourceVHostUser
	} else if a.VHostVDPA != nil {
		vhost := domainDiskSourceVHostVDPA{
			*a.VHostVDPA, domainDiskSource(*a),
		}
		err := d.DecodeElement(&vhost, &start)
		if err != nil {
			return err
		}
		*a = DomainDiskSource(vhost.domainDiskSource)
		a.VHostVDPA = &vhost.DomainDiskSourceVHostVDPA
	}
	return nil
}

type domainDiskBackingStore DomainDiskBackingStore

func (a *DomainDiskBackingStore) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "backingStore"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.VHostVDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostvdpa",
			})
		}
	}
	disk := domainDiskBackingStore(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainDiskBackingStore) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "file"
	}
	a.Source = &DomainDiskSource{}
	if typ == "file" {
		a.Source.File = &DomainDiskSourceFile{}
	} else if typ == "block" {
		a.Source.Block = &DomainDiskSourceBlock{}
	} else if typ == "network" {
		a.Source.Network = &DomainDiskSourceNetwork{}
	} else if typ == "dir" {
		a.Source.Dir = &DomainDiskSourceDir{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainDiskSourceVolume{}
	} else if typ == "vhostuser" {
		a.Source.VHostUser = &DomainDiskSourceVHostUser{}
	} else if typ == "vhostvdpa" {
		a.Source.VHostVDPA = &DomainDiskSourceVHostVDPA{}
	}
	disk := domainDiskBackingStore(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainDiskBackingStore(disk)
	if !ok && a.Source.File.File == "" {
		a.Source.File = nil
	}
	return nil
}

type domainDiskDataStore DomainDiskDataStore

func (a *DomainDiskDataStore) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "dataStore"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.VHostVDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostvdpa",
			})
		}
	}
	disk := domainDiskDataStore(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainDiskDataStore) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "file"
	}
	a.Source = &DomainDiskSource{}
	if typ == "file" {
		a.Source.File = &DomainDiskSourceFile{}
	} else if typ == "block" {
		a.Source.Block = &DomainDiskSourceBlock{}
	} else if typ == "network" {
		a.Source.Network = &DomainDiskSourceNetwork{}
	} else if typ == "dir" {
		a.Source.Dir = &DomainDiskSourceDir{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainDiskSourceVolume{}
	} else if typ == "vhostuser" {
		a.Source.VHostUser = &DomainDiskSourceVHostUser{}
	} else if typ == "vhostvdpa" {
		a.Source.VHostVDPA = &DomainDiskSourceVHostVDPA{}
	}
	disk := domainDiskDataStore(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainDiskDataStore(disk)
	if !ok && a.Source.File.File == "" {
		a.Source.File = nil
	}
	return nil
}

type domainDiskMirror DomainDiskMirror

func (a *DomainDiskMirror) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "mirror"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
			if a.Source.File.File != "" {
				start.Attr = append(start.Attr, xml.Attr{
					xml.Name{Local: "file"}, a.Source.File.File,
				})
			}
			if a.Format != nil && a.Format.Type != "" {
				start.Attr = append(start.Attr, xml.Attr{
					xml.Name{Local: "format"}, a.Format.Type,
				})
			}
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.VHostVDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostvdpa",
			})
		}
	}
	disk := domainDiskMirror(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainDiskMirror) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "file"
	}
	a.Source = &DomainDiskSource{}
	if typ == "file" {
		a.Source.File = &DomainDiskSourceFile{}
	} else if typ == "block" {
		a.Source.Block = &DomainDiskSourceBlock{}
	} else if typ == "network" {
		a.Source.Network = &DomainDiskSourceNetwork{}
	} else if typ == "dir" {
		a.Source.Dir = &DomainDiskSourceDir{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainDiskSourceVolume{}
	} else if typ == "vhostuser" {
		a.Source.VHostUser = &DomainDiskSourceVHostUser{}
	} else if typ == "vhostvdpa" {
		a.Source.VHostVDPA = &DomainDiskSourceVHostVDPA{}
	}
	disk := domainDiskMirror(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainDiskMirror(disk)
	if !ok {
		if a.Source.File.File == "" {
			file, ok := getAttr(start.Attr, "file")
			if ok {
				a.Source.File.File = file
			} else {
				a.Source.File = nil
			}
		}
		if a.Format == nil {
			format, ok := getAttr(start.Attr, "format")
			if ok {
				a.Format = &DomainDiskFormat{
					Type: format,
				}
			}
		}
	}
	return nil
}

type domainDisk DomainDisk

func (a *DomainDisk) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "disk"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		} else if a.Source.NVME != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "nvme",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.VHostVDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostvdpa",
			})
		}
	}
	disk := domainDisk(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainDisk) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "file"
	}
	a.Source = &DomainDiskSource{}
	if typ == "file" {
		a.Source.File = &DomainDiskSourceFile{}
	} else if typ == "block" {
		a.Source.Block = &DomainDiskSourceBlock{}
	} else if typ == "network" {
		a.Source.Network = &DomainDiskSourceNetwork{}
	} else if typ == "dir" {
		a.Source.Dir = &DomainDiskSourceDir{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainDiskSourceVolume{}
	} else if typ == "nvme" {
		a.Source.NVME = &DomainDiskSourceNVME{}
	} else if typ == "vhostuser" {
		a.Source.VHostUser = &DomainDiskSourceVHostUser{}
	} else if typ == "vhostvdpa" {
		a.Source.VHostVDPA = &DomainDiskSourceVHostVDPA{}
	}
	disk := domainDisk(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainDisk(disk)
	return nil
}

func (d *DomainDisk) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainDisk) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainInputSource DomainInputSource

type domainInputSourcePassthrough struct {
	DomainInputSourcePassthrough
	domainInputSource
}

type domainInputSourceEVDev struct {
	DomainInputSourceEVDev
	domainInputSource
}

func (a *DomainInputSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Passthrough != nil {
		passthrough := domainInputSourcePassthrough{
			*a.Passthrough, domainInputSource(*a),
		}
		return e.EncodeElement(&passthrough, start)
	} else if a.EVDev != nil {
		evdev := domainInputSourceEVDev{
			*a.EVDev, domainInputSource(*a),
		}
		return e.EncodeElement(&evdev, start)
	}
	return nil
}

func (a *DomainInputSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.Passthrough != nil {
		passthrough := domainInputSourcePassthrough{
			*a.Passthrough, domainInputSource(*a),
		}
		err := d.DecodeElement(&passthrough, &start)
		if err != nil {
			return err
		}
		*a = DomainInputSource(passthrough.domainInputSource)
		a.Passthrough = &passthrough.DomainInputSourcePassthrough
	} else if a.EVDev != nil {
		evdev := domainInputSourceEVDev{
			*a.EVDev, domainInputSource(*a),
		}
		err := d.DecodeElement(&evdev, &start)
		if err != nil {
			return err
		}
		*a = DomainInputSource(evdev.domainInputSource)
		a.EVDev = &evdev.DomainInputSourceEVDev
	}
	return nil
}

type domainInput DomainInput

func (a *DomainInput) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "input"
	input := domainInput(*a)
	return e.EncodeElement(input, start)
}

func (a *DomainInput) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if ok {
		a.Source = &DomainInputSource{}
		if typ == "passthrough" {
			a.Source.Passthrough = &DomainInputSourcePassthrough{}
		} else if typ == "evdev" {
			a.Source.EVDev = &DomainInputSourceEVDev{}
		}
	}
	input := domainInput(*a)
	err := d.DecodeElement(&input, &start)
	if err != nil {
		return err
	}
	*a = DomainInput(input)
	return nil
}

func (a *DomainFilesystemSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Mount != nil {
		return e.EncodeElement(a.Mount, start)
	} else if a.Block != nil {
		return e.EncodeElement(a.Block, start)
	} else if a.File != nil {
		return e.EncodeElement(a.File, start)
	} else if a.Template != nil {
		return e.EncodeElement(a.Template, start)
	} else if a.RAM != nil {
		return e.EncodeElement(a.RAM, start)
	} else if a.Bind != nil {
		return e.EncodeElement(a.Bind, start)
	} else if a.Volume != nil {
		return e.EncodeElement(a.Volume, start)
	}
	return nil
}

func (a *DomainFilesystemSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.Mount != nil {
		return d.DecodeElement(a.Mount, &start)
	} else if a.Block != nil {
		return d.DecodeElement(a.Block, &start)
	} else if a.File != nil {
		return d.DecodeElement(a.File, &start)
	} else if a.Template != nil {
		return d.DecodeElement(a.Template, &start)
	} else if a.RAM != nil {
		return d.DecodeElement(a.RAM, &start)
	} else if a.Bind != nil {
		return d.DecodeElement(a.Bind, &start)
	} else if a.Volume != nil {
		return d.DecodeElement(a.Volume, &start)
	}
	return nil
}

type domainFilesystem DomainFilesystem

func (a *DomainFilesystem) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "filesystem"
	if a.Source != nil {
		if a.Source.Mount != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "mount",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Template != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "template",
			})
		} else if a.Source.RAM != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "ram",
			})
		} else if a.Source.Bind != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "bind",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		}
	}
	fs := domainFilesystem(*a)
	return e.EncodeElement(fs, start)
}

func (a *DomainFilesystem) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "mount"
	}
	a.Source = &DomainFilesystemSource{}
	if typ == "mount" {
		a.Source.Mount = &DomainFilesystemSourceMount{}
	} else if typ == "block" {
		a.Source.Block = &DomainFilesystemSourceBlock{}
	} else if typ == "file" {
		a.Source.File = &DomainFilesystemSourceFile{}
	} else if typ == "template" {
		a.Source.Template = &DomainFilesystemSourceTemplate{}
	} else if typ == "ram" {
		a.Source.RAM = &DomainFilesystemSourceRAM{}
	} else if typ == "bind" {
		a.Source.Bind = &DomainFilesystemSourceBind{}
	} else if typ == "volume" {
		a.Source.Volume = &DomainFilesystemSourceVolume{}
	}
	fs := domainFilesystem(*a)
	err := d.DecodeElement(&fs, &start)
	if err != nil {
		return err
	}
	*a = DomainFilesystem(fs)
	return nil
}

func (d *DomainFilesystem) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainFilesystem) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainInterfaceVirtualPortParams) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "parameters"
	if a.Any != nil {
		return e.EncodeElement(a.Any, start)
	} else if a.VEPA8021QBG != nil {
		return e.EncodeElement(a.VEPA8021QBG, start)
	} else if a.VNTag8011QBH != nil {
		return e.EncodeElement(a.VNTag8011QBH, start)
	} else if a.OpenVSwitch != nil {
		return e.EncodeElement(a.OpenVSwitch, start)
	} else if a.MidoNet != nil {
		return e.EncodeElement(a.MidoNet, start)
	}
	return nil
}

func (a *DomainInterfaceVirtualPortParams) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.Any != nil {
		return d.DecodeElement(a.Any, &start)
	} else if a.VEPA8021QBG != nil {
		return d.DecodeElement(a.VEPA8021QBG, &start)
	} else if a.VNTag8011QBH != nil {
		return d.DecodeElement(a.VNTag8011QBH, &start)
	} else if a.OpenVSwitch != nil {
		return d.DecodeElement(a.OpenVSwitch, &start)
	} else if a.MidoNet != nil {
		return d.DecodeElement(a.MidoNet, &start)
	}
	return nil
}

type domainInterfaceVirtualPort DomainInterfaceVirtualPort

func (a *DomainInterfaceVirtualPort) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "virtualport"
	if a.Params != nil {
		if a.Params.Any != nil {
			/* no type attr wanted */
		} else if a.Params.VEPA8021QBG != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "802.1Qbg",
			})
		} else if a.Params.VNTag8011QBH != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "802.1Qbh",
			})
		} else if a.Params.OpenVSwitch != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "openvswitch",
			})
		} else if a.Params.MidoNet != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "midonet",
			})
		}
	}
	vp := domainInterfaceVirtualPort(*a)
	return e.EncodeElement(&vp, start)
}

func (a *DomainInterfaceVirtualPort) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	a.Params = &DomainInterfaceVirtualPortParams{}
	if !ok {
		var any DomainInterfaceVirtualPortParamsAny
		a.Params.Any = &any
	} else if typ == "802.1Qbg" {
		var vepa DomainInterfaceVirtualPortParamsVEPA8021QBG
		a.Params.VEPA8021QBG = &vepa
	} else if typ == "802.1Qbh" {
		var vntag DomainInterfaceVirtualPortParamsVNTag8021QBH
		a.Params.VNTag8011QBH = &vntag
	} else if typ == "openvswitch" {
		var ovs DomainInterfaceVirtualPortParamsOpenVSwitch
		a.Params.OpenVSwitch = &ovs
	} else if typ == "midonet" {
		var mido DomainInterfaceVirtualPortParamsMidoNet
		a.Params.MidoNet = &mido
	}

	vp := domainInterfaceVirtualPort(*a)
	err := d.DecodeElement(&vp, &start)
	if err != nil {
		return err
	}
	*a = DomainInterfaceVirtualPort(vp)
	return nil
}

func (a *DomainInterfaceSourceHostdev) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	if a.PCI != nil {
		addr := xml.StartElement{
			Name: xml.Name{Local: "address"},
		}
		addr.Attr = append(addr.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci",
		})
		e.EncodeElement(a.PCI.Address, addr)
	} else if a.USB != nil {
		addr := xml.StartElement{
			Name: xml.Name{Local: "address"},
		}
		addr.Attr = append(addr.Attr, xml.Attr{
			xml.Name{Local: "type"}, "usb",
		})
		e.EncodeElement(a.USB.Address, addr)
	}
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainInterfaceSourceHostdev) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for {
		tok, err := d.Token()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "address" {
				typ, ok := getAttr(tok.Attr, "type")
				if !ok {
					return fmt.Errorf("Missing hostdev address type attribute")
				}

				if typ == "pci" {
					a.PCI = &DomainHostdevSubsysPCISource{
						"",
						&DomainAddressPCI{},
					}
					err := d.DecodeElement(a.PCI.Address, &tok)
					if err != nil {
						return err
					}
				} else if typ == "usb" {
					a.USB = &DomainHostdevSubsysUSBSource{
						"",
						&DomainAddressUSB{},
					}
					err := d.DecodeElement(a.USB, &tok)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainInterfaceSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.User != nil {
		if a.User.Dev != "" {
			return e.EncodeElement(a.User, start)
		} else {
			return nil
		}
	} else if a.Ethernet != nil {
		if len(a.Ethernet.IP) > 0 && len(a.Ethernet.Route) > 0 {
			return e.EncodeElement(a.Ethernet, start)
		}
		return nil
	} else if a.VHostUser != nil {
		typ := getChardevSourceType(a.VHostUser)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
		return e.EncodeElement(a.VHostUser, start)
	} else if a.Server != nil {
		return e.EncodeElement(a.Server, start)
	} else if a.Client != nil {
		return e.EncodeElement(a.Client, start)
	} else if a.MCast != nil {
		return e.EncodeElement(a.MCast, start)
	} else if a.Network != nil {
		return e.EncodeElement(a.Network, start)
	} else if a.Bridge != nil {
		return e.EncodeElement(a.Bridge, start)
	} else if a.Internal != nil {
		return e.EncodeElement(a.Internal, start)
	} else if a.Direct != nil {
		return e.EncodeElement(a.Direct, start)
	} else if a.Hostdev != nil {
		return e.EncodeElement(a.Hostdev, start)
	} else if a.UDP != nil {
		return e.EncodeElement(a.UDP, start)
	} else if a.VDPA != nil {
		return e.EncodeElement(a.VDPA, start)
	} else if a.Null != nil {
		return e.EncodeElement(a.Null, start)
	} else if a.VDS != nil {
		return e.EncodeElement(a.VDS, start)
	}
	return nil
}

func (a *DomainInterfaceSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.User != nil {
		return d.DecodeElement(a.User, &start)
	} else if a.Ethernet != nil {
		return d.DecodeElement(a.Ethernet, &start)
	} else if a.VHostUser != nil {
		typ, ok := getAttr(start.Attr, "type")
		if !ok {
			typ = "pty"
		}
		a.VHostUser = createChardevSource(typ)
		return d.DecodeElement(a.VHostUser, &start)
	} else if a.Server != nil {
		return d.DecodeElement(a.Server, &start)
	} else if a.Client != nil {
		return d.DecodeElement(a.Client, &start)
	} else if a.MCast != nil {
		return d.DecodeElement(a.MCast, &start)
	} else if a.Network != nil {
		return d.DecodeElement(a.Network, &start)
	} else if a.Bridge != nil {
		return d.DecodeElement(a.Bridge, &start)
	} else if a.Internal != nil {
		return d.DecodeElement(a.Internal, &start)
	} else if a.Direct != nil {
		return d.DecodeElement(a.Direct, &start)
	} else if a.Hostdev != nil {
		return d.DecodeElement(a.Hostdev, &start)
	} else if a.UDP != nil {
		return d.DecodeElement(a.UDP, &start)
	} else if a.VDPA != nil {
		return d.DecodeElement(a.VDPA, &start)
	} else if a.Null != nil {
		return d.DecodeElement(a.Null, &start)
	} else if a.VDS != nil {
		return d.DecodeElement(a.VDS, &start)
	}
	return nil
}

type domainInterface DomainInterface

func (a *DomainInterface) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "interface"
	if a.Source != nil {
		if a.Source.User != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "user",
			})
		} else if a.Source.Ethernet != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "ethernet",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.Server != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "server",
			})
		} else if a.Source.Client != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "client",
			})
		} else if a.Source.MCast != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "mcast",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Bridge != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "bridge",
			})
		} else if a.Source.Internal != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "internal",
			})
		} else if a.Source.Direct != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "direct",
			})
		} else if a.Source.Hostdev != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "hostdev",
			})
		} else if a.Source.UDP != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "udp",
			})
		} else if a.Source.VDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vdpa",
			})
		} else if a.Source.Null != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "null",
			})
		} else if a.Source.VDS != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vds",
			})
		}
	}
	fs := domainInterface(*a)
	return e.EncodeElement(fs, start)
}

func (a *DomainInterface) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing interface type attribute")
	}
	a.Source = &DomainInterfaceSource{}
	if typ == "user" {
		a.Source.User = &DomainInterfaceSourceUser{}
	} else if typ == "ethernet" {
		a.Source.Ethernet = &DomainInterfaceSourceEthernet{}
	} else if typ == "vhostuser" {
		a.Source.VHostUser = &DomainChardevSource{}
	} else if typ == "server" {
		a.Source.Server = &DomainInterfaceSourceServer{}
	} else if typ == "client" {
		a.Source.Client = &DomainInterfaceSourceClient{}
	} else if typ == "mcast" {
		a.Source.MCast = &DomainInterfaceSourceMCast{}
	} else if typ == "network" {
		a.Source.Network = &DomainInterfaceSourceNetwork{}
	} else if typ == "bridge" {
		a.Source.Bridge = &DomainInterfaceSourceBridge{}
	} else if typ == "internal" {
		a.Source.Internal = &DomainInterfaceSourceInternal{}
	} else if typ == "direct" {
		a.Source.Direct = &DomainInterfaceSourceDirect{}
	} else if typ == "hostdev" {
		a.Source.Hostdev = &DomainInterfaceSourceHostdev{}
	} else if typ == "udp" {
		a.Source.UDP = &DomainInterfaceSourceUDP{}
	} else if typ == "vdpa" {
		a.Source.VDPA = &DomainInterfaceSourceVDPA{}
	} else if typ == "null" {
		a.Source.Null = &DomainInterfaceSourceNull{}
	} else if typ == "vds" {
		a.Source.VDS = &DomainInterfaceSourceVDS{}
	}
	fs := domainInterface(*a)
	err := d.DecodeElement(&fs, &start)
	if err != nil {
		return err
	}
	*a = DomainInterface(fs)
	return nil
}

func (d *DomainInterface) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainInterface) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainSmartcard DomainSmartcard

func (a *DomainSmartcard) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "smartcard"
	if a.Passthrough != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "passthrough",
		})
		typ := getChardevSourceType(a.Passthrough)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	} else if a.Host != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "host",
		})
	} else if len(a.HostCerts) != 0 {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "host-certificates",
		})
	}
	smartcard := domainSmartcard(*a)
	return e.EncodeElement(smartcard, start)
}

func (a *DomainSmartcard) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	mode, ok := getAttr(start.Attr, "mode")
	if !ok {
		return fmt.Errorf("Missing mode on smartcard device")
	}
	if mode == "host" {
		a.Host = &DomainSmartcardHost{}
	} else if mode == "passthrough" {
		typ, ok := getAttr(start.Attr, "type")
		if !ok {
			typ = "pty"
		}
		a.Passthrough = createChardevSource(typ)
	}
	smartcard := domainSmartcard(*a)
	err := d.DecodeElement(&smartcard, &start)
	if err != nil {
		return err
	}
	*a = DomainSmartcard(smartcard)
	return nil
}

func (d *DomainSmartcard) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainSmartcard) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainTPMBackendExternalSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "source"
	src := DomainChardevSource(*a)
	typ := getChardevSourceType(&src)
	if typ != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, typ,
		})
	}
	return e.EncodeElement(&src, start)
}

func (a *DomainTPMBackendExternalSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "unix"
	}
	src := createChardevSource(typ)
	err := d.DecodeElement(&src, &start)
	if err != nil {
		return err
	}
	*a = DomainTPMBackendExternalSource(*src)
	return nil
}

func (a *DomainTPMBackend) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "backend"
	if a.Passthrough != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "passthrough",
		})
		err := e.EncodeElement(a.Passthrough, start)
		if err != nil {
			return err
		}
	} else if a.Emulator != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "emulator",
		})
		err := e.EncodeElement(a.Emulator, start)
		if err != nil {
			return err
		}
	} else if a.External != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "external",
		})
		err := e.EncodeElement(a.External, start)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *DomainTPMBackend) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing TPM backend type")
	}
	if typ == "passthrough" {
		a.Passthrough = &DomainTPMBackendPassthrough{}
		err := d.DecodeElement(a.Passthrough, &start)
		if err != nil {
			return err
		}
	} else if typ == "emulator" {
		a.Emulator = &DomainTPMBackendEmulator{}
		err := d.DecodeElement(a.Emulator, &start)
		if err != nil {
			return err
		}
	} else if typ == "external" {
		a.External = &DomainTPMBackendExternal{}
		err := d.DecodeElement(a.External, &start)
		if err != nil {
			return err
		}
	} else {
		d.Skip()
	}
	return nil
}

func (a *DomainTPMBackendSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "source"
	if a.File != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "file",
		})
		err := e.EncodeElement(a.File, start)
		if err != nil {
			return err
		}
	} else if a.Dir != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "dir",
		})
		err := e.EncodeElement(a.Dir, start)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *DomainTPMBackendSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing TPM source type")
	}
	if typ == "file" {
		a.File = &DomainTPMBackendSourceFile{}
		err := d.DecodeElement(a.File, &start)
		if err != nil {
			return err
		}
	} else if typ == "dir" {
		a.Dir = &DomainTPMBackendSourceDir{}
		err := d.DecodeElement(a.Dir, &start)
		if err != nil {
			return err
		}
	} else {
		d.Skip()
	}
	return nil
}

func (d *DomainTPM) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainTPM) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainShmem) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainShmem) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func getChardevSourceType(s *DomainChardevSource) string {
	if s.Null != nil {
		return "null"
	} else if s.VC != nil {
		return "vc"
	} else if s.Pty != nil {
		return "pty"
	} else if s.Dev != nil {
		return "dev"
	} else if s.File != nil {
		return "file"
	} else if s.Pipe != nil {
		return "pipe"
	} else if s.StdIO != nil {
		return "stdio"
	} else if s.UDP != nil {
		return "udp"
	} else if s.TCP != nil {
		return "tcp"
	} else if s.UNIX != nil {
		return "unix"
	} else if s.SpiceVMC != nil {
		return "spicevmc"
	} else if s.SpicePort != nil {
		return "spiceport"
	} else if s.NMDM != nil {
		return "nmdm"
	} else if s.QEMUVDAgent != nil {
		return "qemu-vdagent"
	} else if s.DBus != nil {
		return "dbus"
	}
	return ""
}

func createChardevSource(typ string) *DomainChardevSource {
	switch typ {
	case "null":
		return &DomainChardevSource{
			Null: &DomainChardevSourceNull{},
		}
	case "vc":
		return &DomainChardevSource{
			VC: &DomainChardevSourceVC{},
		}
	case "pty":
		return &DomainChardevSource{
			Pty: &DomainChardevSourcePty{},
		}
	case "dev":
		return &DomainChardevSource{
			Dev: &DomainChardevSourceDev{},
		}
	case "file":
		return &DomainChardevSource{
			File: &DomainChardevSourceFile{},
		}
	case "pipe":
		return &DomainChardevSource{
			Pipe: &DomainChardevSourcePipe{},
		}
	case "stdio":
		return &DomainChardevSource{
			StdIO: &DomainChardevSourceStdIO{},
		}
	case "udp":
		return &DomainChardevSource{
			UDP: &DomainChardevSourceUDP{},
		}
	case "tcp":
		return &DomainChardevSource{
			TCP: &DomainChardevSourceTCP{},
		}
	case "unix":
		return &DomainChardevSource{
			UNIX: &DomainChardevSourceUNIX{},
		}
	case "spicevmc":
		return &DomainChardevSource{
			SpiceVMC: &DomainChardevSourceSpiceVMC{},
		}
	case "spiceport":
		return &DomainChardevSource{
			SpicePort: &DomainChardevSourceSpicePort{},
		}
	case "nmdm":
		return &DomainChardevSource{
			NMDM: &DomainChardevSourceNMDM{},
		}
	case "qemu-vdagent":
		return &DomainChardevSource{
			QEMUVDAgent: &DomainChardevSourceQEMUVDAgent{},
		}
	case "dbus":
		return &DomainChardevSource{
			DBus: &DomainChardevSourceDBus{},
		}
	}

	return nil
}

type domainChardevSourceUDPFlat struct {
	Mode    string `xml:"mode,attr"`
	Host    string `xml:"host,attr,omitempty"`
	Service string `xml:"service,attr,omitempty"`
}

func (a *DomainChardevSource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Null != nil {
		return nil
	} else if a.VC != nil {
		return nil
	} else if a.Pty != nil {
		if a.Pty.Path != "" {
			return e.EncodeElement(a.Pty, start)
		}
		return nil
	} else if a.Dev != nil {
		return e.EncodeElement(a.Dev, start)
	} else if a.File != nil {
		return e.EncodeElement(a.File, start)
	} else if a.Pipe != nil {
		return e.EncodeElement(a.Pipe, start)
	} else if a.StdIO != nil {
		return nil
	} else if a.UDP != nil {
		srcs := []domainChardevSourceUDPFlat{
			domainChardevSourceUDPFlat{
				Mode:    "bind",
				Host:    a.UDP.BindHost,
				Service: a.UDP.BindService,
			},
			domainChardevSourceUDPFlat{
				Mode:    "connect",
				Host:    a.UDP.ConnectHost,
				Service: a.UDP.ConnectService,
			},
		}
		if srcs[0].Host != "" || srcs[0].Service != "" {
			err := e.EncodeElement(&srcs[0], start)
			if err != nil {
				return err
			}
		}
		if srcs[1].Host != "" || srcs[1].Service != "" {
			err := e.EncodeElement(&srcs[1], start)
			if err != nil {
				return err
			}
		}
	} else if a.TCP != nil {
		return e.EncodeElement(a.TCP, start)
	} else if a.UNIX != nil {
		if a.UNIX.Path == "" && a.UNIX.Mode == "" {
			return nil
		}
		return e.EncodeElement(a.UNIX, start)
	} else if a.SpiceVMC != nil {
		return nil
	} else if a.SpicePort != nil {
		return e.EncodeElement(a.SpicePort, start)
	} else if a.NMDM != nil {
		return e.EncodeElement(a.NMDM, start)
	} else if a.QEMUVDAgent != nil {
		return e.EncodeElement(a.QEMUVDAgent, start)
	} else if a.DBus != nil {
		return e.EncodeElement(a.DBus, start)
	}
	return nil
}

func (a *DomainChardevSource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if a.Null != nil {
		d.Skip()
		return nil
	} else if a.VC != nil {
		d.Skip()
		return nil
	} else if a.Pty != nil {
		return d.DecodeElement(a.Pty, &start)
	} else if a.Dev != nil {
		return d.DecodeElement(a.Dev, &start)
	} else if a.File != nil {
		return d.DecodeElement(a.File, &start)
	} else if a.Pipe != nil {
		return d.DecodeElement(a.Pipe, &start)
	} else if a.StdIO != nil {
		d.Skip()
		return nil
	} else if a.UDP != nil {
		src := domainChardevSourceUDPFlat{}
		err := d.DecodeElement(&src, &start)
		if src.Mode == "connect" {
			a.UDP.ConnectHost = src.Host
			a.UDP.ConnectService = src.Service
		} else {
			a.UDP.BindHost = src.Host
			a.UDP.BindService = src.Service
		}
		return err
	} else if a.TCP != nil {
		return d.DecodeElement(a.TCP, &start)
	} else if a.UNIX != nil {
		return d.DecodeElement(a.UNIX, &start)
	} else if a.SpiceVMC != nil {
		d.Skip()
		return nil
	} else if a.SpicePort != nil {
		return d.DecodeElement(a.SpicePort, &start)
	} else if a.NMDM != nil {
		return d.DecodeElement(a.NMDM, &start)
	} else if a.QEMUVDAgent != nil {
		return d.DecodeElement(a.QEMUVDAgent, &start)
	} else if a.DBus != nil {
		return d.DecodeElement(a.DBus, &start)
	}
	return nil
}

type domainConsole DomainConsole

func (a *DomainConsole) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "console"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	fs := domainConsole(*a)
	return e.EncodeElement(fs, start)
}

func (a *DomainConsole) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainConsole(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainConsole(con)
	return nil
}

func (d *DomainConsole) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainConsole) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainSerial DomainSerial

func (a *DomainSerial) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "serial"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	s := domainSerial(*a)
	return e.EncodeElement(s, start)
}

func (a *DomainSerial) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainSerial(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainSerial(con)
	return nil
}

func (d *DomainSerial) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainSerial) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainParallel DomainParallel

func (a *DomainParallel) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "parallel"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	s := domainParallel(*a)
	return e.EncodeElement(s, start)
}

func (a *DomainParallel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainParallel(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainParallel(con)
	return nil
}

func (d *DomainParallel) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainParallel) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainInput) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainInput) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainVideo) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainVideo) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainChannelTarget DomainChannelTarget

func (a *DomainChannelTarget) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.VirtIO != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "virtio",
		})
		return e.EncodeElement(a.VirtIO, start)
	} else if a.Xen != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "xen",
		})
		return e.EncodeElement(a.Xen, start)
	} else if a.GuestFWD != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "guestfwd",
		})
		return e.EncodeElement(a.GuestFWD, start)
	}
	return nil
}

func (a *DomainChannelTarget) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing channel target type")
	}
	if typ == "virtio" {
		a.VirtIO = &DomainChannelTargetVirtIO{}
		return d.DecodeElement(a.VirtIO, &start)
	} else if typ == "xen" {
		a.Xen = &DomainChannelTargetXen{}
		return d.DecodeElement(a.Xen, &start)
	} else if typ == "guestfwd" {
		a.GuestFWD = &DomainChannelTargetGuestFWD{}
		return d.DecodeElement(a.GuestFWD, &start)
	}
	d.Skip()
	return nil
}

type domainChannel DomainChannel

func (a *DomainChannel) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "channel"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	fs := domainChannel(*a)
	return e.EncodeElement(fs, start)
}

func (a *DomainChannel) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainChannel(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainChannel(con)
	return nil
}

func (d *DomainChannel) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainChannel) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainRedirFilterUSB) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "class", a.Class, "0x%02x")
	marshalUintAttr(&start, "vendor", a.Vendor, "0x%04x")
	marshalUintAttr(&start, "product", a.Product, "0x%04x")
	if a.Version != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "version"}, a.Version,
		})
	}
	start.Attr = append(start.Attr, xml.Attr{
		xml.Name{Local: "allow"}, a.Allow,
	})
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainRedirFilterUSB) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "class" && attr.Value != "-1" {
			if err := unmarshalUintAttr(attr.Value, &a.Class, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "product" && attr.Value != "-1" {
			if err := unmarshalUintAttr(attr.Value, &a.Product, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "vendor" && attr.Value != "-1" {
			if err := unmarshalUintAttr(attr.Value, &a.Vendor, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "version" && attr.Value != "-1" {
			a.Version = attr.Value
		} else if attr.Name.Local == "allow" {
			a.Allow = attr.Value
		}
	}
	d.Skip()
	return nil
}

type domainRedirDev DomainRedirDev

func (a *DomainRedirDev) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "redirdev"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	fs := domainRedirDev(*a)
	return e.EncodeElement(fs, start)
}

func (a *DomainRedirDev) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainRedirDev(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainRedirDev(con)
	return nil
}

func (d *DomainRedirDev) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainRedirDev) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainMemBalloon) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainMemBalloon) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainVSock) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainVSock) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainSound) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainSound) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

type domainRNGBackendEGD DomainRNGBackendEGD

func (a *DomainRNGBackendEGD) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "backend"
	if a.Source != nil {
		typ := getChardevSourceType(a.Source)
		if typ != "" {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, typ,
			})
		}
	}
	egd := domainRNGBackendEGD(*a)
	return e.EncodeElement(egd, start)
}

func (a *DomainRNGBackendEGD) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		typ = "pty"
	}
	a.Source = createChardevSource(typ)
	con := domainRNGBackendEGD(*a)
	err := d.DecodeElement(&con, &start)
	if err != nil {
		return err
	}
	*a = DomainRNGBackendEGD(con)
	return nil
}

func (a *DomainRNGBackend) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Random != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "model"}, "random",
		})
		return e.EncodeElement(a.Random, start)
	} else if a.EGD != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "model"}, "egd",
		})
		return e.EncodeElement(a.EGD, start)
	} else if a.BuiltIn != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "model"}, "builtin",
		})
		return e.EncodeElement(a.BuiltIn, start)
	}
	return nil
}

func (a *DomainRNGBackend) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	model, ok := getAttr(start.Attr, "model")
	if !ok {
		return nil
	}
	if model == "random" {
		a.Random = &DomainRNGBackendRandom{}
		err := d.DecodeElement(a.Random, &start)
		if err != nil {
			return err
		}
	} else if model == "egd" {
		a.EGD = &DomainRNGBackendEGD{}
		err := d.DecodeElement(a.EGD, &start)
		if err != nil {
			return err
		}
	} else if model == "builtin" {
		a.BuiltIn = &DomainRNGBackendBuiltIn{}
		err := d.DecodeElement(a.BuiltIn, &start)
		if err != nil {
			return err
		}
	}
	d.Skip()
	return nil
}

func (d *DomainRNG) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainRNG) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainHostdevSubsysSCSISource) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.Host != nil {
		return e.EncodeElement(a.Host, start)
	} else if a.ISCSI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "protocol"}, "iscsi",
		})
		return e.EncodeElement(a.ISCSI, start)
	}
	return nil
}

func (a *DomainHostdevSubsysSCSISource) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	proto, ok := getAttr(start.Attr, "protocol")
	if !ok {
		a.Host = &DomainHostdevSubsysSCSISourceHost{}
		err := d.DecodeElement(a.Host, &start)
		if err != nil {
			return err
		}
	}
	if proto == "iscsi" {
		a.ISCSI = &DomainHostdevSubsysSCSISourceISCSI{}
		err := d.DecodeElement(a.ISCSI, &start)
		if err != nil {
			return err
		}
	}
	d.Skip()
	return nil
}

type domainHostdev DomainHostdev

type domainHostdevSubsysSCSI struct {
	DomainHostdevSubsysSCSI
	domainHostdev
}

type domainHostdevSubsysSCSIHost struct {
	DomainHostdevSubsysSCSIHost
	domainHostdev
}

type domainHostdevSubsysUSB struct {
	DomainHostdevSubsysUSB
	domainHostdev
}

type domainHostdevSubsysPCI struct {
	DomainHostdevSubsysPCI
	domainHostdev
}

type domainHostdevSubsysMDev struct {
	DomainHostdevSubsysMDev
	domainHostdev
}

type domainHostdevCapsStorage struct {
	DomainHostdevCapsStorage
	domainHostdev
}

type domainHostdevCapsMisc struct {
	DomainHostdevCapsMisc
	domainHostdev
}

type domainHostdevCapsNet struct {
	DomainHostdevCapsNet
	domainHostdev
}

func (a *DomainHostdev) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "hostdev"
	if a.SubsysSCSI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "subsystem",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "scsi",
		})
		scsi := domainHostdevSubsysSCSI{}
		scsi.domainHostdev = domainHostdev(*a)
		scsi.DomainHostdevSubsysSCSI = *a.SubsysSCSI
		return e.EncodeElement(scsi, start)
	} else if a.SubsysSCSIHost != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "subsystem",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "scsi_host",
		})
		scsi_host := domainHostdevSubsysSCSIHost{}
		scsi_host.domainHostdev = domainHostdev(*a)
		scsi_host.DomainHostdevSubsysSCSIHost = *a.SubsysSCSIHost
		return e.EncodeElement(scsi_host, start)
	} else if a.SubsysUSB != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "subsystem",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "usb",
		})
		usb := domainHostdevSubsysUSB{}
		usb.domainHostdev = domainHostdev(*a)
		usb.DomainHostdevSubsysUSB = *a.SubsysUSB
		return e.EncodeElement(usb, start)
	} else if a.SubsysPCI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "subsystem",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci",
		})
		pci := domainHostdevSubsysPCI{}
		pci.domainHostdev = domainHostdev(*a)
		pci.DomainHostdevSubsysPCI = *a.SubsysPCI
		return e.EncodeElement(pci, start)
	} else if a.SubsysMDev != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "subsystem",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "mdev",
		})
		mdev := domainHostdevSubsysMDev{}
		mdev.domainHostdev = domainHostdev(*a)
		mdev.DomainHostdevSubsysMDev = *a.SubsysMDev
		return e.EncodeElement(mdev, start)
	} else if a.CapsStorage != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "capabilities",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "storage",
		})
		storage := domainHostdevCapsStorage{}
		storage.domainHostdev = domainHostdev(*a)
		storage.DomainHostdevCapsStorage = *a.CapsStorage
		return e.EncodeElement(storage, start)
	} else if a.CapsMisc != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "capabilities",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "misc",
		})
		misc := domainHostdevCapsMisc{}
		misc.domainHostdev = domainHostdev(*a)
		misc.DomainHostdevCapsMisc = *a.CapsMisc
		return e.EncodeElement(misc, start)
	} else if a.CapsNet != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "capabilities",
		})
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "net",
		})
		net := domainHostdevCapsNet{}
		net.domainHostdev = domainHostdev(*a)
		net.DomainHostdevCapsNet = *a.CapsNet
		return e.EncodeElement(net, start)
	} else {
		gen := domainHostdev(*a)
		return e.EncodeElement(gen, start)
	}
}

func (a *DomainHostdev) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	mode, ok := getAttr(start.Attr, "mode")
	if !ok {
		return fmt.Errorf("Missing 'mode' attribute on domain hostdev")
	}
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain controller")
	}
	if mode == "subsystem" {
		if typ == "scsi" {
			var scsi domainHostdevSubsysSCSI
			err := d.DecodeElement(&scsi, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(scsi.domainHostdev)
			a.SubsysSCSI = &scsi.DomainHostdevSubsysSCSI
			return nil
		} else if typ == "scsi_host" {
			var scsi_host domainHostdevSubsysSCSIHost
			err := d.DecodeElement(&scsi_host, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(scsi_host.domainHostdev)
			a.SubsysSCSIHost = &scsi_host.DomainHostdevSubsysSCSIHost
			return nil
		} else if typ == "usb" {
			var usb domainHostdevSubsysUSB
			err := d.DecodeElement(&usb, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(usb.domainHostdev)
			a.SubsysUSB = &usb.DomainHostdevSubsysUSB
			return nil
		} else if typ == "pci" {
			var pci domainHostdevSubsysPCI
			err := d.DecodeElement(&pci, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(pci.domainHostdev)
			a.SubsysPCI = &pci.DomainHostdevSubsysPCI
			return nil
		} else if typ == "mdev" {
			var mdev domainHostdevSubsysMDev
			err := d.DecodeElement(&mdev, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(mdev.domainHostdev)
			a.SubsysMDev = &mdev.DomainHostdevSubsysMDev
			return nil
		}
	} else if mode == "capabilities" {
		if typ == "storage" {
			var storage domainHostdevCapsStorage
			err := d.DecodeElement(&storage, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(storage.domainHostdev)
			a.CapsStorage = &storage.DomainHostdevCapsStorage
			return nil
		} else if typ == "misc" {
			var misc domainHostdevCapsMisc
			err := d.DecodeElement(&misc, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(misc.domainHostdev)
			a.CapsMisc = &misc.DomainHostdevCapsMisc
			return nil
		} else if typ == "net" {
			var net domainHostdevCapsNet
			err := d.DecodeElement(&net, &start)
			if err != nil {
				return err
			}
			*a = DomainHostdev(net.domainHostdev)
			a.CapsNet = &net.DomainHostdevCapsNet
			return nil
		}
	}
	var gen domainHostdev
	err := d.DecodeElement(&gen, &start)
	if err != nil {
		return err
	}
	*a = DomainHostdev(gen)
	return nil
}

func (d *DomainHostdev) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainHostdev) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainGraphicListener) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "listen"
	if a.Address != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "address",
		})
		return e.EncodeElement(a.Address, start)
	} else if a.Network != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "network",
		})
		return e.EncodeElement(a.Network, start)
	} else if a.Socket != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "socket",
		})
		return e.EncodeElement(a.Socket, start)
	} else {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "none",
		})
		e.EncodeToken(start)
		e.EncodeToken(start.End())
	}
	return nil
}

func (a *DomainGraphicListener) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain graphics listen")
	}
	if typ == "address" {
		var addr DomainGraphicListenerAddress
		err := d.DecodeElement(&addr, &start)
		if err != nil {
			return err
		}
		a.Address = &addr
		return nil
	} else if typ == "network" {
		var net DomainGraphicListenerNetwork
		err := d.DecodeElement(&net, &start)
		if err != nil {
			return err
		}
		a.Network = &net
		return nil
	} else if typ == "socket" {
		var sock DomainGraphicListenerSocket
		err := d.DecodeElement(&sock, &start)
		if err != nil {
			return err
		}
		a.Socket = &sock
		return nil
	} else if typ == "none" {
		d.Skip()
	}
	return nil
}

type domainGraphicSDL struct {
	DomainGraphicSDL
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicVNC struct {
	DomainGraphicVNC
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicRDP struct {
	DomainGraphicRDP
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicDesktop struct {
	DomainGraphicDesktop
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicSpice struct {
	DomainGraphicSpice
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicEGLHeadless struct {
	DomainGraphicEGLHeadless
	Audio *DomainGraphicAudio `xml:"audio"`
}

type domainGraphicDBus struct {
	DomainGraphicDBus
	Audio *DomainGraphicAudio `xml:"audio"`
}

func (a *DomainGraphic) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "graphics"
	if a.SDL != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "sdl",
		})
		sdl := domainGraphicSDL{*a.SDL, a.Audio}
		return e.EncodeElement(sdl, start)
	} else if a.VNC != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "vnc",
		})
		vnc := domainGraphicVNC{*a.VNC, a.Audio}
		return e.EncodeElement(vnc, start)
	} else if a.RDP != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "rdp",
		})
		rdp := domainGraphicRDP{*a.RDP, a.Audio}
		return e.EncodeElement(rdp, start)
	} else if a.Desktop != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "desktop",
		})
		desktop := domainGraphicDesktop{*a.Desktop, a.Audio}
		return e.EncodeElement(desktop, start)
	} else if a.Spice != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "spice",
		})
		spice := domainGraphicSpice{*a.Spice, a.Audio}
		return e.EncodeElement(spice, start)
	} else if a.EGLHeadless != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "egl-headless",
		})
		egl := domainGraphicEGLHeadless{*a.EGLHeadless, a.Audio}
		return e.EncodeElement(egl, start)
	} else if a.DBus != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "dbus",
		})
		dbus := domainGraphicDBus{*a.DBus, a.Audio}
		return e.EncodeElement(dbus, start)
	}
	return nil
}

func (a *DomainGraphic) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain graphics")
	}
	if typ == "sdl" {
		var sdl domainGraphicSDL
		err := d.DecodeElement(&sdl, &start)
		if err != nil {
			return err
		}
		a.SDL = &sdl.DomainGraphicSDL
		a.Audio = sdl.Audio
		return nil
	} else if typ == "vnc" {
		var vnc domainGraphicVNC
		err := d.DecodeElement(&vnc, &start)
		if err != nil {
			return err
		}
		a.VNC = &vnc.DomainGraphicVNC
		a.Audio = vnc.Audio
		return nil
	} else if typ == "rdp" {
		var rdp domainGraphicRDP
		err := d.DecodeElement(&rdp, &start)
		if err != nil {
			return err
		}
		a.RDP = &rdp.DomainGraphicRDP
		a.Audio = rdp.Audio
		return nil
	} else if typ == "desktop" {
		var desktop domainGraphicDesktop
		err := d.DecodeElement(&desktop, &start)
		if err != nil {
			return err
		}
		a.Desktop = &desktop.DomainGraphicDesktop
		a.Audio = desktop.Audio
		return nil
	} else if typ == "spice" {
		var spice domainGraphicSpice
		err := d.DecodeElement(&spice, &start)
		if err != nil {
			return err
		}
		a.Spice = &spice.DomainGraphicSpice
		a.Audio = spice.Audio
		return nil
	} else if typ == "egl-headless" {
		var egl domainGraphicEGLHeadless
		err := d.DecodeElement(&egl, &start)
		if err != nil {
			return err
		}
		a.EGLHeadless = &egl.DomainGraphicEGLHeadless
		a.Audio = egl.Audio
		return nil
	} else if typ == "dbus" {
		var dbus domainGraphicDBus
		err := d.DecodeElement(&dbus, &start)
		if err != nil {
			return err
		}
		a.DBus = &dbus.DomainGraphicDBus
		a.Audio = dbus.Audio
		return nil
	}
	return nil
}

func (a *DomainAudio) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "audio"
	if a.ID != 0 {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "id"}, fmt.Sprintf("%d", a.ID),
		})
	}
	if a.TimerPeriod != 0 {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "timerPeriod"}, fmt.Sprintf("%d", a.TimerPeriod),
		})
	}
	if a.None != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "none",
		})
		return e.EncodeElement(a.None, start)
	} else if a.ALSA != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "alsa",
		})
		return e.EncodeElement(a.ALSA, start)
	} else if a.CoreAudio != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "coreaudio",
		})
		return e.EncodeElement(a.CoreAudio, start)
	} else if a.Jack != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "jack",
		})
		return e.EncodeElement(a.Jack, start)
	} else if a.OSS != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "oss",
		})
		return e.EncodeElement(a.OSS, start)
	} else if a.PulseAudio != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pulseaudio",
		})
		return e.EncodeElement(a.PulseAudio, start)
	} else if a.SDL != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "sdl",
		})
		return e.EncodeElement(a.SDL, start)
	} else if a.SPICE != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "spice",
		})
		return e.EncodeElement(a.SPICE, start)
	} else if a.File != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "file",
		})
		return e.EncodeElement(a.File, start)
	} else if a.DBus != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "dbus",
		})
		return e.EncodeElement(a.DBus, start)
	} else if a.PipeWire != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pipewire",
		})
		return e.EncodeElement(a.PipeWire, start)
	}
	return nil
}

func (a *DomainAudio) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain audio")
	}
	id, ok := getAttr(start.Attr, "id")
	if ok {
		idval, err := strconv.ParseInt(id, 10, 32)
		if err != nil {
			return err
		}
		a.ID = int(idval)
	}

	period, ok := getAttr(start.Attr, "timerPeriod")
	if ok {
		periodval, err := strconv.ParseUint(period, 10, 32)
		if err != nil {
			return err
		}
		a.TimerPeriod = uint(periodval)
	}

	if typ == "none" {
		var none DomainAudioNone
		err := d.DecodeElement(&none, &start)
		if err != nil {
			return err
		}
		a.None = &none
		return nil
	} else if typ == "alsa" {
		var alsa DomainAudioALSA
		err := d.DecodeElement(&alsa, &start)
		if err != nil {
			return err
		}
		a.ALSA = &alsa
		return nil
	} else if typ == "coreaudio" {
		var coreaudio DomainAudioCoreAudio
		err := d.DecodeElement(&coreaudio, &start)
		if err != nil {
			return err
		}
		a.CoreAudio = &coreaudio
		return nil
	} else if typ == "jack" {
		var jack DomainAudioJack
		err := d.DecodeElement(&jack, &start)
		if err != nil {
			return err
		}
		a.Jack = &jack
		return nil
	} else if typ == "oss" {
		var oss DomainAudioOSS
		err := d.DecodeElement(&oss, &start)
		if err != nil {
			return err
		}
		a.OSS = &oss
		return nil
	} else if typ == "pulseaudio" {
		var pulseaudio DomainAudioPulseAudio
		err := d.DecodeElement(&pulseaudio, &start)
		if err != nil {
			return err
		}
		a.PulseAudio = &pulseaudio
		return nil
	} else if typ == "sdl" {
		var sdl DomainAudioSDL
		err := d.DecodeElement(&sdl, &start)
		if err != nil {
			return err
		}
		a.SDL = &sdl
		return nil
	} else if typ == "spice" {
		var spice DomainAudioSPICE
		err := d.DecodeElement(&spice, &start)
		if err != nil {
			return err
		}
		a.SPICE = &spice
		return nil
	} else if typ == "file" {
		var file DomainAudioFile
		err := d.DecodeElement(&file, &start)
		if err != nil {
			return err
		}
		a.File = &file
		return nil
	} else if typ == "dbus" {
		var dbus DomainAudioDBus
		err := d.DecodeElement(&dbus, &start)
		if err != nil {
			return err
		}
		a.DBus = &dbus
		return nil
	} else if typ == "pipewire" {
		var pipewire DomainAudioPipeWire
		err := d.DecodeElement(&pipewire, &start)
		if err != nil {
			return err
		}
		a.PipeWire = &pipewire
		return nil
	}
	return nil
}

func (d *DomainMemorydev) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainMemorydev) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (d *DomainWatchdog) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainWatchdog) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainCryptoBackend) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "backend"
	if a.BuiltIn != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "model"}, "builtin",
		})
	} else if a.LKCF != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "model"}, "lkcf",
		})
	}
	marshalUintAttr(&start, "queues", &a.Queues, "%d")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainCryptoBackend) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "model")
	if !ok {
		return fmt.Errorf("Missing 'model' attribute on domain crypto backend")
	}
	for _, attr := range start.Attr {
		if attr.Name.Local == "queues" {
			var v *uint
			if err := unmarshalUintAttr(attr.Value, &v, 10); err != nil {
				return err
			}
			if v != nil {
				a.Queues = *v
			}
		}
	}

	if typ == "builtin" {
		var builtin DomainCryptoBackendBuiltIn
		a.BuiltIn = &builtin
		d.Skip()
		return nil
	} else if typ == "lkcf" {
		var lkcf DomainCryptoBackendLKCF
		a.LKCF = &lkcf
		d.Skip()
		return nil
	}

	return nil
}

func marshalUintAttr(start *xml.StartElement, name string, val *uint, format string) {
	if val != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: name}, fmt.Sprintf(format, *val),
		})
	}
}

func marshalUint64Attr(start *xml.StartElement, name string, val *uint64, format string) {
	if val != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: name}, fmt.Sprintf(format, *val),
		})
	}
}

func (a *DomainMemorydevTargetAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "base", a.Base, "0x%08x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressPCI) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "domain", a.Domain, "0x%04x")
	marshalUintAttr(&start, "bus", a.Bus, "0x%02x")
	marshalUintAttr(&start, "slot", a.Slot, "0x%02x")
	marshalUintAttr(&start, "function", a.Function, "0x%x")
	if a.MultiFunction != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "multifunction"}, a.MultiFunction,
		})
	}
	e.EncodeToken(start)
	if a.ZPCI != nil {
		zpci := xml.StartElement{}
		zpci.Name.Local = "zpci"
		err := e.EncodeElement(a.ZPCI, zpci)
		if err != nil {
			return err
		}
	}
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressZPCI) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "uid", a.UID, "0x%04x")
	marshalUintAttr(&start, "fid", a.FID, "0x%04x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressUSB) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "bus", a.Bus, "%d")
	if a.Port != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "port"}, a.Port,
		})
	}
	marshalUintAttr(&start, "device", a.Device, "%d")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressDrive) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "controller", a.Controller, "%d")
	marshalUintAttr(&start, "bus", a.Bus, "%d")
	marshalUintAttr(&start, "target", a.Target, "%d")
	marshalUintAttr(&start, "unit", a.Unit, "%d")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressDIMM) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "slot", a.Slot, "%d")
	marshalUint64Attr(&start, "base", a.Base, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressISA) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "iobase", a.IOBase, "0x%x")
	marshalUintAttr(&start, "irq", a.IRQ, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressVirtioMMIO) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressCCW) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "cssid", a.CSSID, "0x%x")
	marshalUintAttr(&start, "ssid", a.SSID, "0x%x")
	marshalUintAttr(&start, "devno", a.DevNo, "0x%04x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressVirtioSerial) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "controller", a.Controller, "%d")
	marshalUintAttr(&start, "bus", a.Bus, "%d")
	marshalUintAttr(&start, "port", a.Port, "%d")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressSpaprVIO) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUint64Attr(&start, "reg", a.Reg, "0x%x")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressCCID) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	marshalUintAttr(&start, "controller", a.Controller, "%d")
	marshalUintAttr(&start, "slot", a.Slot, "%d")
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressVirtioS390) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddressUnassigned) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	e.EncodeToken(start)
	e.EncodeToken(start.End())
	return nil
}

func (a *DomainAddress) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.USB != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "usb",
		})
		return e.EncodeElement(a.USB, start)
	} else if a.PCI != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "pci",
		})
		return e.EncodeElement(a.PCI, start)
	} else if a.Drive != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "drive",
		})
		return e.EncodeElement(a.Drive, start)
	} else if a.DIMM != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "dimm",
		})
		return e.EncodeElement(a.DIMM, start)
	} else if a.ISA != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "isa",
		})
		return e.EncodeElement(a.ISA, start)
	} else if a.VirtioMMIO != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "virtio-mmio",
		})
		return e.EncodeElement(a.VirtioMMIO, start)
	} else if a.CCW != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ccw",
		})
		return e.EncodeElement(a.CCW, start)
	} else if a.VirtioSerial != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "virtio-serial",
		})
		return e.EncodeElement(a.VirtioSerial, start)
	} else if a.SpaprVIO != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "spapr-vio",
		})
		return e.EncodeElement(a.SpaprVIO, start)
	} else if a.CCID != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "ccid",
		})
		return e.EncodeElement(a.CCID, start)
	} else if a.VirtioS390 != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "virtio-s390",
		})
		return e.EncodeElement(a.VirtioS390, start)
	} else if a.Unassigned != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "unassigned",
		})
		return e.EncodeElement(a.Unassigned, start)
	} else {
		return nil
	}
}

func unmarshalUint64Attr(valstr string, valptr **uint64, base int) error {
	if base == 16 {
		valstr = strings.TrimPrefix(valstr, "0x")
	}
	val, err := strconv.ParseUint(valstr, base, 64)
	if err != nil {
		return err
	}
	*valptr = &val
	return nil
}

func unmarshalUintAttr(valstr string, valptr **uint, base int) error {
	if base == 16 {
		valstr = strings.TrimPrefix(valstr, "0x")
	}
	val, err := strconv.ParseUint(valstr, base, 64)
	if err != nil {
		return err
	}
	vali := uint(val)
	*valptr = &vali
	return nil
}

func (a *DomainMemorydevTargetAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "base" {
			if err := unmarshalUintAttr(attr.Value, &a.Base, 0); err != nil {
				return err
			}
		}
	}

	d.Skip()
	return nil
}

func (a *DomainAddressUSB) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "bus" {
			if err := unmarshalUintAttr(attr.Value, &a.Bus, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "port" {
			a.Port = attr.Value
		} else if attr.Name.Local == "device" {
			if err := unmarshalUintAttr(attr.Value, &a.Device, 10); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressPCI) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "domain" {
			if err := unmarshalUintAttr(attr.Value, &a.Domain, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "bus" {
			if err := unmarshalUintAttr(attr.Value, &a.Bus, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "slot" {
			if err := unmarshalUintAttr(attr.Value, &a.Slot, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "function" {
			if err := unmarshalUintAttr(attr.Value, &a.Function, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "multifunction" {
			a.MultiFunction = attr.Value
		}
	}

	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "zpci" {
				a.ZPCI = &DomainAddressZPCI{}
				err = d.DecodeElement(a.ZPCI, &tok)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (a *DomainAddressZPCI) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "fid" {
			if err := unmarshalUintAttr(attr.Value, &a.FID, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "uid" {
			if err := unmarshalUintAttr(attr.Value, &a.UID, 0); err != nil {
				return err
			}
		}
	}

	d.Skip()
	return nil
}

func (a *DomainAddressDrive) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "controller" {
			if err := unmarshalUintAttr(attr.Value, &a.Controller, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "bus" {
			if err := unmarshalUintAttr(attr.Value, &a.Bus, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "target" {
			if err := unmarshalUintAttr(attr.Value, &a.Target, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "unit" {
			if err := unmarshalUintAttr(attr.Value, &a.Unit, 10); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressDIMM) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "slot" {
			if err := unmarshalUintAttr(attr.Value, &a.Slot, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "base" {
			if err := unmarshalUint64Attr(attr.Value, &a.Base, 16); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressISA) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "iobase" {
			if err := unmarshalUintAttr(attr.Value, &a.IOBase, 16); err != nil {
				return err
			}
		} else if attr.Name.Local == "irq" {
			if err := unmarshalUintAttr(attr.Value, &a.IRQ, 16); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressVirtioMMIO) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	d.Skip()
	return nil
}

func (a *DomainAddressCCW) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "cssid" {
			if err := unmarshalUintAttr(attr.Value, &a.CSSID, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "ssid" {
			if err := unmarshalUintAttr(attr.Value, &a.SSID, 0); err != nil {
				return err
			}
		} else if attr.Name.Local == "devno" {
			if err := unmarshalUintAttr(attr.Value, &a.DevNo, 0); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressVirtioSerial) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "controller" {
			if err := unmarshalUintAttr(attr.Value, &a.Controller, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "bus" {
			if err := unmarshalUintAttr(attr.Value, &a.Bus, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "port" {
			if err := unmarshalUintAttr(attr.Value, &a.Port, 10); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressSpaprVIO) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "reg" {
			if err := unmarshalUint64Attr(attr.Value, &a.Reg, 16); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressCCID) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "controller" {
			if err := unmarshalUintAttr(attr.Value, &a.Controller, 10); err != nil {
				return err
			}
		} else if attr.Name.Local == "slot" {
			if err := unmarshalUintAttr(attr.Value, &a.Slot, 10); err != nil {
				return err
			}
		}
	}
	d.Skip()
	return nil
}

func (a *DomainAddressVirtioS390) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	d.Skip()
	return nil
}

func (a *DomainAddressUnassigned) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	d.Skip()
	return nil
}

func (a *DomainAddress) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var typ string
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			typ = attr.Value
			break
		}
	}
	if typ == "" {
		d.Skip()
		return nil
	}

	if typ == "usb" {
		a.USB = &DomainAddressUSB{}
		return d.DecodeElement(a.USB, &start)
	} else if typ == "pci" {
		a.PCI = &DomainAddressPCI{}
		return d.DecodeElement(a.PCI, &start)
	} else if typ == "drive" {
		a.Drive = &DomainAddressDrive{}
		return d.DecodeElement(a.Drive, &start)
	} else if typ == "dimm" {
		a.DIMM = &DomainAddressDIMM{}
		return d.DecodeElement(a.DIMM, &start)
	} else if typ == "isa" {
		a.ISA = &DomainAddressISA{}
		return d.DecodeElement(a.ISA, &start)
	} else if typ == "virtio-mmio" {
		a.VirtioMMIO = &DomainAddressVirtioMMIO{}
		return d.DecodeElement(a.VirtioMMIO, &start)
	} else if typ == "ccw" {
		a.CCW = &DomainAddressCCW{}
		return d.DecodeElement(a.CCW, &start)
	} else if typ == "virtio-serial" {
		a.VirtioSerial = &DomainAddressVirtioSerial{}
		return d.DecodeElement(a.VirtioSerial, &start)
	} else if typ == "spapr-vio" {
		a.SpaprVIO = &DomainAddressSpaprVIO{}
		return d.DecodeElement(a.SpaprVIO, &start)
	} else if typ == "ccid" {
		a.CCID = &DomainAddressCCID{}
		return d.DecodeElement(a.CCID, &start)
	} else if typ == "virtio-s390" {
		a.VirtioS390 = &DomainAddressVirtioS390{}
		return d.DecodeElement(a.VirtioS390, &start)
	} else if typ == "unassigned" {
		a.Unassigned = &DomainAddressUnassigned{}
		return d.DecodeElement(a.Unassigned, &start)
	}

	return nil
}

func (d *DomainCPU) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), d)
}

func (d *DomainCPU) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}

func (a *DomainLaunchSecuritySEV) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	if a.KernelHashes != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "kernelHashes"}, a.KernelHashes,
		})
	}

	e.EncodeToken(start)

	if a.CBitPos != nil {
		cbitpos := xml.StartElement{
			Name: xml.Name{Local: "cbitpos"},
		}
		e.EncodeToken(cbitpos)
		e.EncodeToken(xml.CharData(fmt.Sprintf("%d", *a.CBitPos)))
		e.EncodeToken(cbitpos.End())
	}

	if a.ReducedPhysBits != nil {
		reducedPhysBits := xml.StartElement{
			Name: xml.Name{Local: "reducedPhysBits"},
		}
		e.EncodeToken(reducedPhysBits)
		e.EncodeToken(xml.CharData(fmt.Sprintf("%d", *a.ReducedPhysBits)))
		e.EncodeToken(reducedPhysBits.End())
	}

	if a.Policy != nil {
		policy := xml.StartElement{
			Name: xml.Name{Local: "policy"},
		}
		e.EncodeToken(policy)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%04x", *a.Policy)))
		e.EncodeToken(policy.End())
	}

	dhcert := xml.StartElement{
		Name: xml.Name{Local: "dhCert"},
	}
	e.EncodeToken(dhcert)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.DHCert)))
	e.EncodeToken(dhcert.End())

	session := xml.StartElement{
		Name: xml.Name{Local: "session"},
	}
	e.EncodeToken(session)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.Session)))
	e.EncodeToken(session.End())

	e.EncodeToken(start.End())

	return nil
}

func (a *DomainLaunchSecuritySEV) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "kernelHashes" {
			a.KernelHashes = attr.Value
		}
	}

	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "policy" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUintAttr(string(data), &a.Policy, 16); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "cbitpos" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUintAttr(string(data), &a.CBitPos, 10); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "reducedPhysBits" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUintAttr(string(data), &a.ReducedPhysBits, 10); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "dhCert" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.DHCert = string(data)
				}
			} else if tok.Name.Local == "session" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.Session = string(data)
				}
			}
		}
	}
	return nil
}

func (a *DomainLaunchSecuritySEVSNP) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	if a.KernelHashes != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "kernelHashes"}, a.KernelHashes,
		})
	}

	if a.AuthorKey != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "authorKey"}, a.AuthorKey,
		})
	}

	if a.VCEK != "" {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "vcek"}, a.VCEK,
		})
	}

	e.EncodeToken(start)

	if a.CBitPos != nil {
		cbitpos := xml.StartElement{
			Name: xml.Name{Local: "cbitpos"},
		}
		e.EncodeToken(cbitpos)
		e.EncodeToken(xml.CharData(fmt.Sprintf("%d", *a.CBitPos)))
		e.EncodeToken(cbitpos.End())
	}

	if a.ReducedPhysBits != nil {
		reducedPhysBits := xml.StartElement{
			Name: xml.Name{Local: "reducedPhysBits"},
		}
		e.EncodeToken(reducedPhysBits)
		e.EncodeToken(xml.CharData(fmt.Sprintf("%d", *a.ReducedPhysBits)))
		e.EncodeToken(reducedPhysBits.End())
	}

	if a.Policy != nil {
		policy := xml.StartElement{
			Name: xml.Name{Local: "policy"},
		}
		e.EncodeToken(policy)
		e.EncodeToken(xml.CharData(fmt.Sprintf("0x%08x", *a.Policy)))
		e.EncodeToken(policy.End())
	}

	gvwo := xml.StartElement{
		Name: xml.Name{Local: "guestVisibleWorkarounds"},
	}
	e.EncodeToken(gvwo)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.GuestVisibleWorkarounds)))
	e.EncodeToken(gvwo.End())

	idBlock := xml.StartElement{
		Name: xml.Name{Local: "idBlock"},
	}
	e.EncodeToken(idBlock)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.IDBlock)))
	e.EncodeToken(idBlock.End())

	idAuth := xml.StartElement{
		Name: xml.Name{Local: "idAuth"},
	}
	e.EncodeToken(idAuth)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.IDAuth)))
	e.EncodeToken(idAuth.End())

	hostData := xml.StartElement{
		Name: xml.Name{Local: "hostData"},
	}
	e.EncodeToken(hostData)
	e.EncodeToken(xml.CharData(fmt.Sprintf("%s", a.HostData)))
	e.EncodeToken(hostData.End())

	e.EncodeToken(start.End())

	return nil
}

func (a *DomainLaunchSecuritySEVSNP) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	for _, attr := range start.Attr {
		if attr.Name.Local == "kernelHashes" {
			a.KernelHashes = attr.Value
		} else if attr.Name.Local == "authorKey" {
			a.AuthorKey = attr.Value
		} else if attr.Name.Local == "vcek" {
			a.VCEK = attr.Value
		}
	}

	for {
		tok, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch tok := tok.(type) {
		case xml.StartElement:
			if tok.Name.Local == "policy" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUint64Attr(string(data), &a.Policy, 16); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "cbitpos" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUintAttr(string(data), &a.CBitPos, 10); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "reducedPhysBits" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					if err := unmarshalUintAttr(string(data), &a.ReducedPhysBits, 10); err != nil {
						return err
					}
				}
			} else if tok.Name.Local == "guestVisibleWorkarounds" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.GuestVisibleWorkarounds = string(data)
				}
			} else if tok.Name.Local == "idBlock" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.IDBlock = string(data)
				}
			} else if tok.Name.Local == "idAuth" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.IDAuth = string(data)
				}
			} else if tok.Name.Local == "hostData" {
				data, err := d.Token()
				if err != nil {
					return err
				}
				switch data := data.(type) {
				case xml.CharData:
					a.HostData = string(data)
				}
			}
		}
	}
	return nil
}

func (a *DomainLaunchSecurity) MarshalXML(e *xml.Encoder, start xml.StartElement) error {

	if a.SEV != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "sev",
		})
		return e.EncodeElement(a.SEV, start)
	} else if a.SEVSNP != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "sev-snp",
		})
		return e.EncodeElement(a.SEVSNP, start)
	} else if a.S390PV != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "s390-pv",
		})
		return e.EncodeElement(a.S390PV, start)
	} else {
		return nil
	}

}

func (a *DomainLaunchSecurity) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var typ string
	for _, attr := range start.Attr {
		if attr.Name.Local == "type" {
			typ = attr.Value
		}
	}

	if typ == "" {
		d.Skip()
		return nil
	}

	if typ == "sev" {
		a.SEV = &DomainLaunchSecuritySEV{}
		return d.DecodeElement(a.SEV, &start)
	} else if typ == "sev-snp" {
		a.SEVSNP = &DomainLaunchSecuritySEVSNP{}
		return d.DecodeElement(a.SEVSNP, &start)
	} else if typ == "s390-pv" {
		a.S390PV = &DomainLaunchSecurityS390PV{}
		return d.DecodeElement(a.S390PV, &start)
	}

	return nil
}

type domainSysInfo DomainSysInfo

type domainSysInfoSMBIOS struct {
	DomainSysInfoSMBIOS
	domainSysInfo
}

type domainSysInfoFWCfg struct {
	DomainSysInfoFWCfg
	domainSysInfo
}

func (a *DomainSysInfo) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "sysinfo"
	if a.SMBIOS != nil {
		smbios := domainSysInfoSMBIOS{}
		smbios.domainSysInfo = domainSysInfo(*a)
		smbios.DomainSysInfoSMBIOS = *a.SMBIOS
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "smbios",
		})
		return e.EncodeElement(smbios, start)
	} else if a.FWCfg != nil {
		fwcfg := domainSysInfoFWCfg{}
		fwcfg.domainSysInfo = domainSysInfo(*a)
		fwcfg.DomainSysInfoFWCfg = *a.FWCfg
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "type"}, "fwcfg",
		})
		return e.EncodeElement(fwcfg, start)
	} else {
		gen := domainSysInfo(*a)
		return e.EncodeElement(gen, start)
	}
}

func (a *DomainSysInfo) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if !ok {
		return fmt.Errorf("Missing 'type' attribute on domain controller")
	}
	if typ == "smbios" {
		var smbios domainSysInfoSMBIOS
		err := d.DecodeElement(&smbios, &start)
		if err != nil {
			return err
		}
		*a = DomainSysInfo(smbios.domainSysInfo)
		a.SMBIOS = &smbios.DomainSysInfoSMBIOS
		return nil
	} else if typ == "fwcfg" {
		var fwcfg domainSysInfoFWCfg
		err := d.DecodeElement(&fwcfg, &start)
		if err != nil {
			return err
		}
		*a = DomainSysInfo(fwcfg.domainSysInfo)
		a.FWCfg = &fwcfg.DomainSysInfoFWCfg
		return nil
	} else {
		var gen domainSysInfo
		err := d.DecodeElement(&gen, &start)
		if err != nil {
			return err
		}
		*a = DomainSysInfo(gen)
		return nil
	}
}

type domainNVRam DomainNVRam

func (a *DomainNVRam) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "nvram"
	if a.Source != nil {
		if a.Source.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Source.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Source.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Source.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Source.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		} else if a.Source.NVME != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "nvme",
			})
		} else if a.Source.VHostUser != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostuser",
			})
		} else if a.Source.VHostVDPA != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "vhostvdpa",
			})
		}
	}
	disk := domainNVRam(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainNVRam) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")

	if ok {
		a.Source = &DomainDiskSource{}
		if typ == "file" {
			a.Source.File = &DomainDiskSourceFile{}
		} else if typ == "block" {
			a.Source.Block = &DomainDiskSourceBlock{}
		} else if typ == "network" {
			a.Source.Network = &DomainDiskSourceNetwork{}
		} else if typ == "dir" {
			a.Source.Dir = &DomainDiskSourceDir{}
		} else if typ == "volume" {
			a.Source.Volume = &DomainDiskSourceVolume{}
		} else if typ == "nvme" {
			a.Source.NVME = &DomainDiskSourceNVME{}
		} else if typ == "vhostuser" {
			a.Source.VHostUser = &DomainDiskSourceVHostUser{}
		} else if typ == "vhostvdpa" {
			a.Source.VHostVDPA = &DomainDiskSourceVHostVDPA{}
		}
	}
	disk := domainNVRam(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	if a.Source != nil {
		a.NVRam = ""
	}
	*a = DomainNVRam(disk)
	return nil
}
