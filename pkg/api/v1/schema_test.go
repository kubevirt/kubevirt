package v1

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var exampleJSON = `{
  "name": "testvm",
  "memory": {
    "value": 8192,
    "unit": "KiB"
  },
  "type": "qemu",
  "os": {
    "type": {
      "os": "hvm"
    },
    "bootOrder": null
  },
  "devices": {
    "emulator": "/usr/local/bin/qemu-x86_64",
    "interfaces": [
      {
        "type": "network",
        "source": {
          "network": "default"
        }
      }
    ],
    "disks": [
      {
        "device": "disk",
        "type": "network",
        "source": {
          "protocol": "iscsi",
          "name": "iqn.2013-07.com.example:iscsi-nopool/2",
          "host": {
            "name": "example.com",
            "port": "3260"
          }
        },
        "target": {
          "dev": "vda"
        },
        "driver": {
          "name": "qemu",
          "type": "raw"
        }
      }
    ]
  }
}`

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the json above
	var exampleVM = NewMinimalDomainSpec("testvm")
	exampleVM.Devices.Disks = []Disk{
		{
			Type:   "network",
			Device: "disk",
			Driver: &DiskDriver{Name: "qemu",
				Type: "raw"},
			Source: DiskSource{Protocol: "iscsi",
				Name: "iqn.2013-07.com.example:iscsi-nopool/2",
				Host: &DiskSourceHost{Name: "example.com", Port: "3260"}},
			Target: DiskTarget{Device: "vda"},
		},
	}

	Context("With example schema in json", func() {
		It("Unmarshal json into struct", func() {
			newDomain := DomainSpec{}
			err := json.Unmarshal([]byte(exampleJSON), &newDomain)
			Expect(err).To(BeNil())

			Expect(newDomain).To(Equal(*exampleVM))
		})
		It("Marshal struct into json", func() {
			buf, err := json.MarshalIndent(*exampleVM, "", "  ")
			Expect(err).To(BeNil())

			Expect(string(buf)).To(Equal(exampleJSON))
		})
	})
})
