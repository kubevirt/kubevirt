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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package api

import (
	"encoding/xml"

	"github.com/jeevatkm/go-model"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

var exampleXML = `<domain type="qemu">
  <name>testvm</name>
  <memory unit="KiB">8192</memory>
  <os>
    <type>hvm</type>
  </os>
  <devices>
    <interface type="network">
      <source network="default"></source>
    </interface>
    <video>
      <model type="vga"></model>
    </video>
    <video>
      <model type="qxl"></model>
    </video>
    <disk device="disk" type="network">
      <source protocol="iscsi" name="iqn.2013-07.com.example:iscsi-nopool/2">
        <host name="example.com" port="3260"></host>
      </source>
      <target dev="vda"></target>
      <driver name="qemu" type="raw"></driver>
    </disk>
    <serial type="pty">
      <target port="123"></target>
    </serial>
    <console type="pty">
      <target type="serial" port="123"></target>
    </console>
    <watchdog model="i6300esb" action="poweroff"></watchdog>
  </devices>
  <metadata>
    <graceperiod xmlns="http://kubevirt.io">
      <period>0</period>
      <start>0</start>
    </graceperiod>
  </metadata>
</domain>`

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the xml above
	var exampleDomain = NewMinimalDomainSpec("testvm")
	exampleDomain.Devices.Disks = []Disk{
		{Type: "network",
			Device: "disk",
			Driver: &DiskDriver{Name: "qemu",
				Type: "raw"},
			Source: DiskSource{Protocol: "iscsi",
				Name: "iqn.2013-07.com.example:iscsi-nopool/2",
				Host: &DiskSourceHost{Name: "example.com", Port: "3260"}},
			Target: DiskTarget{Device: "vda"},
		},
	}
	exampleDomain.Devices.Video = []Video{
		{Model: VideoModel{Type: "vga"}},
		{Model: VideoModel{Type: "qxl"}},
	}
	exampleDomain.Devices.Serials = []Serial{
		{Type: "pty", Target: &SerialTarget{Port: newUInt(123)}},
	}
	exampleDomain.Devices.Consoles = []Console{
		{Type: "pty", Target: &ConsoleTarget{Type: newString("serial"), Port: newUInt(123)}},
	}
	exampleDomain.Devices.Watchdog = &Watchdog{
		Model:  "i6300esb",
		Action: "poweroff",
	}

	Context("With schema", func() {
		It("Generate expected libvirt xml", func() {
			domain := NewMinimalDomainSpec("testvm")
			buf, err := xml.Marshal(domain)
			Expect(err).To(BeNil())

			newDomain := DomainSpec{}
			err = xml.Unmarshal(buf, &newDomain)
			Expect(err).To(BeNil())

			domain.XMLName.Local = "domain"
			Expect(newDomain).To(Equal(*domain))
		})
	})
	Context("With example schema", func() {
		It("Unmarshal into struct", func() {
			newDomain := DomainSpec{}
			err := xml.Unmarshal([]byte(exampleXML), &newDomain)
			newDomain.XMLName.Local = ""
			Expect(err).To(BeNil())

			Expect(newDomain).To(Equal(*exampleDomain))
		})
		It("Marshal into xml", func() {
			buf, err := xml.MarshalIndent(*exampleDomain, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(exampleXML))
		})

	})
	Context("With v1.DomainSpec", func() {
		var v1DomainSpec = v1.NewMinimalDomainSpec()
		v1DomainSpec.Devices.Disks = []v1.Disk{
			{Type: "network",
				Device: "disk",
				Driver: &v1.DiskDriver{Name: "qemu",
					Type: "raw"},
				Source: v1.DiskSource{Protocol: "iscsi",
					Name: "iqn.2013-07.com.example:iscsi-nopool/2",
					Host: &v1.DiskSourceHost{Name: "example.com", Port: "3260"}},
				Target: v1.DiskTarget{Device: "vda"},
			},
		}
		v1DomainSpec.Devices.Video = []v1.Video{
			{Type: "vga"},
			{Type: "qxl"},
		}
		v1DomainSpec.Devices.Serials = []v1.Serial{
			{Type: "pty", Target: &v1.SerialTarget{Port: newUInt(123)}},
		}
		v1DomainSpec.Devices.Consoles = []v1.Console{
			{Type: "pty", Target: &v1.ConsoleTarget{Type: newString("serial"), Port: newUInt(123)}},
		}
		v1DomainSpec.Devices.Watchdog = &v1.Watchdog{
			Model:  "i6300esb",
			Action: "poweroff",
		}

		It("converts to libvirt.DomainSpec", func() {
			virtDomainSpec := DomainSpec{}
			errs := model.Copy(&virtDomainSpec, v1DomainSpec)
			virtDomainSpec.Name = "testvm"
			Expect(virtDomainSpec).To(Equal(*exampleDomain))
			Expect(errs).To(BeEmpty())
		})
		It("converts to v1.DomainSpec", func() {
			convertedDomainSpec := v1.DomainSpec{}
			errs := model.Copy(&convertedDomainSpec, exampleDomain)
			Expect(convertedDomainSpec).To(Equal(*v1DomainSpec))
			Expect(errs).To(BeEmpty())
		})
	})
})

func newUInt(v uint) *uint {
	return &v
}

func newString(v string) *string {
	return &v
}
