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

package v1

import (
	"encoding/json"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/core/v1"
)

var exampleJSON = `{
  "kind": "VirtualMachine",
  "apiVersion": "kubevirt.io/v1alpha1",
  "metadata": {
    "name": "testvm",
    "namespace": "default",
    "selfLink": "/apis/kubevirt.io/v1alpha1/namespaces/default/virtualmachines/testvm",
    "creationTimestamp": null
  },
  "spec": {
    "domain": {
      "resources": {
        "initial": {
          "memory": "8Mi"
        }
      },
      "devices": {
        "disks": [
          {
            "name": "disk0",
            "volumeName": "volume0",
            "disk": {
              "dev": "vda"
            }
          },
          {
            "name": "cdrom0",
            "volumeName": "volume1",
            "cdrom": {
              "dev": "vdb",
              "readonly": true,
              "tray": "open"
            }
          },
          {
            "name": "floppy0",
            "volumeName": "volume2",
            "floppy": {
              "dev": "vdc",
              "readonly": true,
              "tray": "open"
            }
          },
          {
            "name": "lun0",
            "volumeName": "volume3",
            "lun": {
              "dev": "vdd",
              "readonly": true
            }
          }
        ]
      }
    },
    "volumes": [
      {
        "name": "volume0",
        "registryDisk": {
          "image": "test/image"
        }
      },
      {
        "name": "volume1",
        "cloudInitNoCloud": {
          "secretRef": {
            "name": "testsecret"
          }
        }
      },
      {
        "name": "volume2",
        "iscsi": {
          "targetPortal": "1234",
          "iqn": "",
          "lun": 0,
          "secretRef": {
            "name": "testsecret"
          }
        }
      },
      {
        "name": "volume3",
        "persistentVolumeClaim": {
          "claimName": "testclaim"
        }
      }
    ]
  },
  "status": {
    "graphics": null
  }
}`

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the json above
	var exampleVM = NewMinimalVM("testvm")
	exampleVM.Spec.Domain.Devices.Disks = []Disk{
		{
			Name:       "disk0",
			VolumeName: "volume0",
			DiskDevice: DiskDevice{
				Disk: &DiskTarget{
					Device:   "vda",
					ReadOnly: false,
				},
			},
		},
		{
			Name:       "cdrom0",
			VolumeName: "volume1",
			DiskDevice: DiskDevice{
				CDRom: &CDRomTarget{
					Device:   "vdb",
					ReadOnly: _true,
					Tray:     "open",
				},
			},
		},
		{
			Name:       "floppy0",
			VolumeName: "volume2",
			DiskDevice: DiskDevice{
				Floppy: &FloppyTarget{
					Device:   "vdc",
					ReadOnly: true,
					Tray:     "open",
				},
			},
		},
		{
			Name:       "lun0",
			VolumeName: "volume3",
			DiskDevice: DiskDevice{
				LUN: &LunTarget{
					Device:   "vdd",
					ReadOnly: true,
				},
			},
		},
	}

	exampleVM.Spec.Volumes = []Volume{
		{
			Name: "volume0",
			VolumeSource: VolumeSource{
				RegistryDisk: &RegistryDiskSource{
					Image: "test/image",
				},
			},
		},
		{
			Name: "volume1",
			VolumeSource: VolumeSource{
				CloudInitNoCloud: &CloudInitNoCloudSource{
					UserDataSecretRef: &v1.LocalObjectReference{
						Name: "testsecret",
					},
				},
			},
		},
		{
			Name: "volume2",
			VolumeSource: VolumeSource{
				ISCSI: &v1.ISCSIVolumeSource{
					TargetPortal: "1234",
					SecretRef: &v1.LocalObjectReference{
						Name: "testsecret",
					},
				},
			},
		},
		{
			Name: "volume3",
			VolumeSource: VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: "testclaim",
				},
			},
		},
	}

	Context("With example schema in json", func() {
		It("Unmarshal json into struct", func() {
			newVM := &VirtualMachine{}
			err := json.Unmarshal([]byte(exampleJSON), newVM)
			Expect(err).To(BeNil())

			Expect(newVM).To(Equal(exampleVM))
		})
		It("Marshal struct into json", func() {
			buf, err := json.MarshalIndent(*exampleVM, "", "  ")
			Expect(err).To(BeNil())

			Expect(string(buf)).To(Equal(exampleJSON))
		})
	})
})
