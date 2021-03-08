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
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

type NetworkTemplateConfig struct {
	InterfaceConfig string
}

var exampleJSONFmt = `{
  "kind": "VirtualMachineInstance",
  "apiVersion": "kubevirt.io/%s",
  "metadata": {
    "name": "testvmi",
    "namespace": "default",
    "selfLink": "/apis/kubevirt.io/%s/namespaces/default/virtualmachineinstances/testvmi",
    "creationTimestamp": null
  },
  "spec": {
    "domain": {
      "resources": {
        "requests": {
          "memory": "8Mi"
        }
      },
      "cpu": {
        "cores": 3,
        "sockets": 1,
        "threads": 1,
        "model": "Conroe",
        "features": [
          {
            "name": "pcid",
            "policy": "require"
          },
          {
            "name": "monitor",
            "policy": "disable"
          }
        ],
        "dedicatedCpuPlacement": true
      },
      "machine": {
        "type": ""
      },
      "firmware": {
        "uuid": "28a42a60-44ef-4428-9c10-1a6aee94627f"
      },
      "clock": {
        "utc": {},
        "timer": {
          "hpet": {
            "present": true
          },
          "kvm": {
            "present": true
          },
          "pit": {
            "present": true
          },
          "rtc": {
            "present": true
          },
          "hyperv": {
            "present": true
          }
        }
      },
      "features": {
        "acpi": {
          "enabled": false
        },
        "apic": {
          "enabled": true
        },
        "hyperv": {
          "relaxed": {
            "enabled": true
          },
          "vapic": {
            "enabled": false
          },
          "spinlocks": {
            "enabled": true,
            "spinlocks": 4096
          },
          "vpindex": {
            "enabled": false
          },
          "runtime": {
            "enabled": true
          },
          "synic": {
            "enabled": false
          },
          "synictimer": {
            "enabled": true,
            "direct": {
              "enabled": true
            }
          },
          "reset": {
            "enabled": false
          },
          "vendorid": {
            "enabled": true,
            "vendorid": "vendor"
          },
          "frequencies": {
            "enabled": false
          },
          "reenlightenment": {
            "enabled": false
          },
          "tlbflush": {
            "enabled": true
          }
        },
        "smm": {
          "enabled": true
        },
        "kvm": {
          "hidden": true
        },
        "pvspinlock": {
          "enabled": false
        }
      },
      "devices": {
        "disks": [
          {
            "name": "disk0",
            "disk": {
              "bus": "virtio"
            },
            "dedicatedIOThread": true
          },
          {
            "name": "cdrom0",
            "cdrom": {
              "bus": "virtio",
              "readonly": true,
              "tray": "open"
            }
          },
          {
            "name": "floppy0",
            "floppy": {
              "readonly": true,
              "tray": "open"
            }
          },
          {
            "name": "lun0",
            "lun": {
              "bus": "virtio",
              "readonly": true
            }
          },
          {
            "name": "disk1",
            "disk": {
              "bus": "virtio"
            },
            "serial": "sn-11223344"
          }
        ],
        "interfaces": [
          {
            "name": "default",
            {{.InterfaceConfig}}
          }
        ],
        "inputs": [
          {
            "bus": "virtio",
            "type": "tablet",
            "name": "tablet0"
          }
        ],
        "rng": {},
        "blockMultiQueue": true
      },
      "ioThreadsPolicy": "shared"
    },
    "volumes": [
      {
        "name": "disk0",
        "containerDisk": {
          "image": "test/image",
          "path": "/disk.img"
        }
      },
      {
        "name": "cdrom0",
        "cloudInitNoCloud": {
          "secretRef": {
            "name": "testsecret"
          },
          "networkDataSecretRef": {
            "name": "testnetworksecret"
          }
        }
      },
      {
        "name": "floppy0",
        "persistentVolumeClaim": {
          "claimName": "testclaim"
        }
      }
    ],
    "networks": [
      {
        "name": "default",
        "pod": {}
      }
    ]
  },
  "status": {
    "guestOSInfo": {}
  }
}`

var exampleJSON = fmt.Sprintf(exampleJSONFmt, ApiLatestVersion, ApiLatestVersion)

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the json above
	var exampleVMI *VirtualMachineInstance

	BeforeEach(func() {
		exampleVMI = NewMinimalVMI("testvmi")

		exampleVMI.Spec.Domain.Devices.Disks = []Disk{
			{
				Name: "disk0",
				DiskDevice: DiskDevice{
					Disk: &DiskTarget{
						Bus:      "virtio",
						ReadOnly: false,
					},
				},
				DedicatedIOThread: _true,
			},
			{
				Name: "cdrom0",
				DiskDevice: DiskDevice{
					CDRom: &CDRomTarget{
						Bus:      "virtio",
						ReadOnly: _true,
						Tray:     "open",
					},
				},
			},
			{
				Name: "floppy0",
				DiskDevice: DiskDevice{
					Floppy: &FloppyTarget{
						ReadOnly: true,
						Tray:     "open",
					},
				},
			},
			{
				Name: "lun0",
				DiskDevice: DiskDevice{
					LUN: &LunTarget{
						Bus:      "virtio",
						ReadOnly: true,
					},
				},
			},
			{
				Name:   "disk1",
				Serial: "sn-11223344",
				DiskDevice: DiskDevice{
					Disk: &DiskTarget{
						Bus:      "virtio",
						ReadOnly: false,
					},
				},
			},
		}

		exampleVMI.Spec.Domain.Devices.Rng = &Rng{}
		exampleVMI.Spec.Domain.Devices.Inputs = []Input{
			{
				Bus:  "virtio",
				Type: "tablet",
				Name: "tablet0",
			},
		}
		exampleVMI.Spec.Domain.Devices.BlockMultiQueue = _true

		exampleVMI.Spec.Volumes = []Volume{
			{
				Name: "disk0",
				VolumeSource: VolumeSource{
					ContainerDisk: &ContainerDiskSource{
						Image: "test/image",
						Path:  "/disk.img",
					},
				},
			},
			{
				Name: "cdrom0",
				VolumeSource: VolumeSource{
					CloudInitNoCloud: &CloudInitNoCloudSource{
						UserDataSecretRef: &v1.LocalObjectReference{
							Name: "testsecret",
						},
						NetworkDataSecretRef: &v1.LocalObjectReference{
							Name: "testnetworksecret",
						},
					},
				},
			},
			{
				Name: "floppy0",
				VolumeSource: VolumeSource{
					PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
						ClaimName: "testclaim",
					},
				},
			},
		}
		exampleVMI.Spec.Domain.Features = &Features{
			ACPI:       FeatureState{Enabled: _false},
			SMM:        &FeatureState{Enabled: _true},
			APIC:       &FeatureAPIC{Enabled: _true},
			KVM:        &FeatureKVM{Hidden: true},
			Pvspinlock: &FeatureState{Enabled: _false},
			Hyperv: &FeatureHyperv{
				Relaxed:         &FeatureState{Enabled: _true},
				VAPIC:           &FeatureState{Enabled: _false},
				Spinlocks:       &FeatureSpinlocks{Enabled: _true},
				VPIndex:         &FeatureState{Enabled: _false},
				Runtime:         &FeatureState{Enabled: _true},
				SyNIC:           &FeatureState{Enabled: _false},
				SyNICTimer:      &SyNICTimer{Enabled: _true, Direct: &FeatureState{Enabled: _true}},
				Reset:           &FeatureState{Enabled: _false},
				VendorID:        &FeatureVendorID{Enabled: _true, VendorID: "vendor"},
				Frequencies:     &FeatureState{Enabled: _false},
				Reenlightenment: &FeatureState{Enabled: _false},
				TLBFlush:        &FeatureState{Enabled: _true},
			},
		}
		exampleVMI.Spec.Domain.Clock = &Clock{
			ClockOffset: ClockOffset{
				UTC: &ClockOffsetUTC{},
			},
			Timer: &Timer{
				HPET:   &HPETTimer{},
				KVM:    &KVMTimer{},
				PIT:    &PITTimer{},
				RTC:    &RTCTimer{},
				Hyperv: &HypervTimer{},
			},
		}
		exampleVMI.Spec.Domain.Firmware = &Firmware{
			UUID: "28a42a60-44ef-4428-9c10-1a6aee94627f",
		}
		exampleVMI.Spec.Domain.CPU = &CPU{
			Cores:   3,
			Sockets: 1,
			Threads: 1,
			Model:   "Conroe",
			Features: []CPUFeature{
				{
					Name:   "pcid",
					Policy: "require",
				},
				{
					Name:   "monitor",
					Policy: "disable",
				},
			},
			DedicatedCPUPlacement: true,
		}
		exampleVMI.Spec.Networks = []Network{
			Network{
				Name: "default",
				NetworkSource: NetworkSource{
					Pod: &PodNetwork{},
				},
			},
		}

		policy := IOThreadsPolicyShared
		exampleVMI.Spec.Domain.IOThreadsPolicy = &policy

		SetObjectDefaults_VirtualMachineInstance(exampleVMI)
	})
	Context("With example schema in json use pod network and bridge interface", func() {
		It("Unmarshal json into struct", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []Interface{
				Interface{
					Name: "default",
					InterfaceBindingMethod: InterfaceBindingMethod{
						Bridge: &InterfaceBridge{},
					},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"bridge": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).To(BeNil())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).To(BeNil())
			newVMI := &VirtualMachineInstance{}
			err = json.Unmarshal(tpl.Bytes(), newVMI)
			Expect(err).To(BeNil())
			Expect(newVMI).To(Equal(exampleVMI))
		})
		It("Marshal struct into json", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []Interface{
				Interface{
					Name: "default",
					InterfaceBindingMethod: InterfaceBindingMethod{
						Bridge: &InterfaceBridge{},
					},
				},
			}

			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"bridge": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).To(BeNil())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).To(BeNil())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
	})
	Context("With example schema in json use pod network and slirp interface", func() {
		It("Unmarshal json into struct", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []Interface{
				Interface{
					Name: "default",
					InterfaceBindingMethod: InterfaceBindingMethod{
						Slirp: &InterfaceSlirp{},
					},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).To(BeNil())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).To(BeNil())
			newVMI := &VirtualMachineInstance{}
			err = json.Unmarshal(tpl.Bytes(), newVMI)
			Expect(err).To(BeNil())
			Expect(newVMI).To(Equal(exampleVMI))
		})
		It("Marshal struct into json", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []Interface{
				Interface{
					Name: "default",
					InterfaceBindingMethod: InterfaceBindingMethod{
						Slirp: &InterfaceSlirp{},
					},
				},
			}

			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).To(BeNil())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).To(BeNil())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
		It("Marshal struct into json with port configure", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []Interface{
				Interface{
					Name: "default",
					InterfaceBindingMethod: InterfaceBindingMethod{
						Slirp: &InterfaceSlirp{}},
					Ports: []Port{{Port: 80}},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {},
            "ports": [
              {
                "port": 80
              }
            ]`}

			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).To(BeNil())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).To(BeNil())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).To(BeNil())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
	})
})
