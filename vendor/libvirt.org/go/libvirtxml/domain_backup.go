/* SPDX-License-Identifier: MIT */

package libvirtxml

import "encoding/xml"

type DomainBackupPullServerTCP struct {
	Name string `xml:"name,attr"`
	Port uint   `xml:"port,attr,omitempty"`
}

type DomainBackupPullServerUNIX struct {
	Socket string `xml:"socket,attr"`
}
type DomainBackupPullServerFD struct {
	FDGroup string `xml:"fdgroup,attr"`
}

type DomainBackupPullServer struct {
	TLS  string                      `xml:"tls,attr,omitempty"`
	TCP  *DomainBackupPullServerTCP  `xml:"-"`
	UNIX *DomainBackupPullServerUNIX `xml:"-"`
	FD   *DomainBackupPullServerFD   `xml:"-"`
}

type DomainBackupDiskDriver struct {
	Type string `xml:"type,attr,omitempty"`
}

type DomainBackupPushDisk struct {
	Name        string                  `xml:"name,attr"`
	Backup      string                  `xml:"backup,attr,omitempty"`
	BackupMode  string                  `xml:"backupmode,attr,omitempty"`
	Incremental string                  `xml:"incremental,attr,omitempty"`
	Driver      *DomainBackupDiskDriver `xml:"driver"`
	Target      *DomainDiskSource       `xml:"target"`
}

type DomainBackupPushDisks struct {
	Disks []DomainBackupPushDisk `xml:"disk"`
}

type DomainBackupPullDisk struct {
	Name         string                  `xml:"name,attr"`
	Backup       string                  `xml:"backup,attr,omitempty"`
	BackupMode   string                  `xml:"backupmode,attr,omitempty"`
	Incremental  string                  `xml:"incremental,attr,omitempty"`
	ExportName   string                  `xml:"exportname,attr,omitempty"`
	ExportBitmap string                  `xml:"exportbitmap,attr,omitempty"`
	Driver       *DomainBackupDiskDriver `xml:"driver"`
	Scratch      *DomainDiskSource       `xml:"scratch"`
}

type DomainBackupPullDisks struct {
	Disks []DomainBackupPullDisk `xml:"disk"`
}

type DomainBackupPush struct {
	Disks *DomainBackupPushDisks `xml:"disks"`
}

type DomainBackupPull struct {
	Server *DomainBackupPullServer `xml:"server"`
	Disks  *DomainBackupPullDisks  `xml:"disks"`
}

type DomainBackup struct {
	XMLName     xml.Name          `xml:"domainbackup"`
	Incremental string            `xml:"incremental,omitempty"`
	Push        *DomainBackupPush `xml:"-"`
	Pull        *DomainBackupPull `xml:"-"`
}

type domainBackupPullServer DomainBackupPullServer

type domainBackupPullServerTCP struct {
	DomainBackupPullServerTCP
	domainBackupPullServer
}

type domainBackupPullServerUNIX struct {
	DomainBackupPullServerUNIX
	domainBackupPullServer
}

type domainBackupPullServerFD struct {
	DomainBackupPullServerFD
	domainBackupPullServer
}

func (a *DomainBackupPullServer) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if a.TCP != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "transport"}, "tcp",
		})
		tmp := domainBackupPullServerTCP{}
		tmp.domainBackupPullServer = domainBackupPullServer(*a)
		tmp.DomainBackupPullServerTCP = *a.TCP
		return e.EncodeElement(tmp, start)
	} else if a.UNIX != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "transport"}, "unix",
		})
		tmp := domainBackupPullServerUNIX{}
		tmp.domainBackupPullServer = domainBackupPullServer(*a)
		tmp.DomainBackupPullServerUNIX = *a.UNIX
		return e.EncodeElement(tmp, start)
	} else if a.FD != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "transport"}, "fd",
		})
		tmp := domainBackupPullServerFD{}
		tmp.domainBackupPullServer = domainBackupPullServer(*a)
		tmp.DomainBackupPullServerFD = *a.FD
		return e.EncodeElement(tmp, start)
	}

	return nil
}

func (a *DomainBackupPullServer) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	transport, ok := getAttr(start.Attr, "transport")
	if !ok {
		transport = "tcp"
	}

	if transport == "tcp" {
		var tmp domainBackupPullServerTCP
		err := d.DecodeElement(&tmp, &start)
		if err != nil {
			return err
		}
		*a = DomainBackupPullServer(tmp.domainBackupPullServer)
		a.TCP = &tmp.DomainBackupPullServerTCP
		return nil
	} else if transport == "unix" {
		var tmp domainBackupPullServerUNIX
		err := d.DecodeElement(&tmp, &start)
		if err != nil {
			return err
		}
		*a = DomainBackupPullServer(tmp.domainBackupPullServer)
		a.UNIX = &tmp.DomainBackupPullServerUNIX
		return nil
	} else if transport == "fd" {
		var tmp domainBackupPullServerFD
		err := d.DecodeElement(&tmp, &start)
		if err != nil {
			return err
		}
		*a = DomainBackupPullServer(tmp.domainBackupPullServer)
		a.FD = &tmp.DomainBackupPullServerFD
		return nil
	}
	return nil
}

type domainBackupPushDisk DomainBackupPushDisk

func (a *DomainBackupPushDisk) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "disk"

	if a.Target != nil {
		if a.Target.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Target.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Target.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Target.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Target.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		}
	}
	disk := domainBackupPushDisk(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainBackupPushDisk) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if ok {
		a.Target = &DomainDiskSource{}
		if typ == "file" {
			a.Target.File = &DomainDiskSourceFile{}
		} else if typ == "block" {
			a.Target.Block = &DomainDiskSourceBlock{}
		} else if typ == "network" {
			a.Target.Network = &DomainDiskSourceNetwork{}
		} else if typ == "dir" {
			a.Target.Dir = &DomainDiskSourceDir{}
		} else if typ == "volume" {
			a.Target.Volume = &DomainDiskSourceVolume{}
		}
	}
	disk := domainBackupPushDisk(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainBackupPushDisk(disk)
	return nil
}

type domainBackupPullDisk DomainBackupPullDisk

func (a *DomainBackupPullDisk) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "disk"

	if a.Scratch != nil {
		if a.Scratch.File != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "file",
			})
		} else if a.Scratch.Block != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "block",
			})
		} else if a.Scratch.Dir != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "dir",
			})
		} else if a.Scratch.Network != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "network",
			})
		} else if a.Scratch.Volume != nil {
			start.Attr = append(start.Attr, xml.Attr{
				xml.Name{Local: "type"}, "volume",
			})
		}
	}

	disk := domainBackupPullDisk(*a)
	return e.EncodeElement(disk, start)
}

func (a *DomainBackupPullDisk) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	typ, ok := getAttr(start.Attr, "type")
	if ok {
		a.Scratch = &DomainDiskSource{}
		if typ == "file" {
			a.Scratch.File = &DomainDiskSourceFile{}
		} else if typ == "block" {
			a.Scratch.Block = &DomainDiskSourceBlock{}
		} else if typ == "network" {
			a.Scratch.Network = &DomainDiskSourceNetwork{}
		} else if typ == "dir" {
			a.Scratch.Dir = &DomainDiskSourceDir{}
		} else if typ == "volume" {
			a.Scratch.Volume = &DomainDiskSourceVolume{}
		}
	}

	disk := domainBackupPullDisk(*a)
	err := d.DecodeElement(&disk, &start)
	if err != nil {
		return err
	}
	*a = DomainBackupPullDisk(disk)
	return nil
}

type domainBackup DomainBackup

type domainBackupPull struct {
	DomainBackupPull
	domainBackup
}

type domainBackupPush struct {
	DomainBackupPush
	domainBackup
}

func (a *DomainBackup) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	start.Name.Local = "domainbackup"

	if a.Push != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "push",
		})
		tmp := domainBackupPush{}
		tmp.domainBackup = domainBackup(*a)
		tmp.DomainBackupPush = *a.Push
		return e.EncodeElement(tmp, start)
	} else if a.Pull != nil {
		start.Attr = append(start.Attr, xml.Attr{
			xml.Name{Local: "mode"}, "pull",
		})
		tmp := domainBackupPull{}
		tmp.domainBackup = domainBackup(*a)
		tmp.DomainBackupPull = *a.Pull
		return e.EncodeElement(tmp, start)
	}

	return nil
}

func (a *DomainBackup) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	mode, ok := getAttr(start.Attr, "mode")
	if !ok {
		mode = "push"
	}

	if mode == "push" {
		var tmp domainBackupPush
		err := d.DecodeElement(&tmp, &start)
		if err != nil {
			return err
		}
		*a = DomainBackup(tmp.domainBackup)
		a.Push = &tmp.DomainBackupPush
		return nil
	} else if mode == "pull" {
		var tmp domainBackupPull
		err := d.DecodeElement(&tmp, &start)
		if err != nil {
			return err
		}
		*a = DomainBackup(tmp.domainBackup)
		a.Pull = &tmp.DomainBackupPull
		return nil
	}
	return nil
}

func (s *DomainBackup) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *DomainBackup) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
