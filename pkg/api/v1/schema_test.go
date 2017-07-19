/*
 * This file is part of the kubevirt project
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

package v1

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var exampleJSON = `{
  "memory": {
    "value": 8
  },
  "type": "qemu",
  "os": {
    "type": {
      "os": "hvm"
    }
  },
  "devices": {
    "interfaces": [
      {}
    ],
    "disks": [
      {
        "device": "disk",
        "source": {
          "iscsi": {
            "targetPortal": "example.com:3260",
            "iqn": "iqn.2013-07.com.example:iscsi-nopool",
            "lun": 2
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
	var exampleVM = NewMinimalDomainSpec()
	exampleVM.Devices.Disks = []Disk{
		{
			Device: "disk",
			Driver: &DiskDriver{
				Name: "qemu",
				Type: "raw",
			},
			Source: DiskSource{
				ISCSI: &DiskSourceISCSI{
					IQN:          "iqn.2013-07.com.example:iscsi-nopool",
					Lun:          2,
					TargetPortal: "example.com:3260",
				},
			},
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
