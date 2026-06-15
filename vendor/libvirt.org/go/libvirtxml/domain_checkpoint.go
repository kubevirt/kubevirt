/* SPDX-License-Identifier: MIT */

package libvirtxml

import "encoding/xml"

type DomainCheckpointParent struct {
	Name string `xml:"name"`
}

type DomainCheckpointDisk struct {
	Name       string `xml:"name,attr"`
	Checkpoint string `xml:"checkpoint,attr,omitempty"`
	Bitmap     string `xml:"bitmap,attr,omitempty"`
	Size       uint64 `xml:"size,attr,omitempty"`
}

type DomainCheckpointDisks struct {
	Disks []DomainCheckpointDisk `xml:"disk"`
}

type DomainCheckpoint struct {
	XMLName      xml.Name                `xml:"domaincheckpoint"`
	Name         string                  `xml:"name,omitempty"`
	Description  string                  `xml:"description,omitempty"`
	State        string                  `xml:"state,omitempty"`
	CreationTime string                  `xml:"creationTime,omitempty"`
	Parent       *DomainCheckpointParent `xml:"parent"`
	Disks        *DomainCheckpointDisks  `xml:"disks"`
	Domain       *Domain                 `xml:"domain"`
}

func (s *DomainCheckpoint) Unmarshal(doc string) error {
	return xml.Unmarshal([]byte(doc), s)
}

func (s *DomainCheckpoint) Marshal() (string, error) {
	doc, err := xml.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(doc), nil
}
