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
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v12 "kubevirt.io/api/core/v1"
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
    "selfLink": "/apis/kubevirt.io/%s/namespaces/default/virtualmachineinstances/testvmi"
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
    "guestOSInfo": {},
    "runtimeUser": 0
  }
}`

var exampleJSON = fmt.Sprintf(exampleJSONFmt, v12.ApiLatestVersion, v12.ApiLatestVersion)

var _ = Describe("Schema", func() {
	//The example domain should stay in sync to the json above
	var exampleVMI *v12.VirtualMachineInstance

	BeforeEach(func() {
		exampleVMI = NewMinimalVMI("testvmi")

		pointer.BoolPtr(true)
		exampleVMI.Spec.Domain.Devices.Disks = []v12.Disk{
			{
				Name: "disk0",
				DiskDevice: v12.DiskDevice{
					Disk: &v12.DiskTarget{
						Bus:      v12.VirtIO,
						ReadOnly: false,
					},
				},
				DedicatedIOThread: pointer.BoolPtr(true),
			},
			{
				Name: "cdrom0",
				DiskDevice: v12.DiskDevice{
					CDRom: &v12.CDRomTarget{
						Bus:      v12.VirtIO,
						ReadOnly: pointer.BoolPtr(true),
						Tray:     "open",
					},
				},
			},
			{
				Name: "lun0",
				DiskDevice: v12.DiskDevice{
					LUN: &v12.LunTarget{
						Bus:      v12.VirtIO,
						ReadOnly: true,
					},
				},
			},
			{
				Name:   "disk1",
				Serial: "sn-11223344",
				DiskDevice: v12.DiskDevice{
					Disk: &v12.DiskTarget{
						Bus:      v12.VirtIO,
						ReadOnly: false,
					},
				},
			},
		}

		exampleVMI.Spec.Domain.Devices.Rng = &v12.Rng{}
		exampleVMI.Spec.Domain.Devices.Inputs = []v12.Input{
			{
				Bus:  v12.VirtIO,
				Type: "tablet",
				Name: "tablet0",
			},
		}
		exampleVMI.Spec.Domain.Devices.BlockMultiQueue = pointer.BoolPtr(true)

		exampleVMI.Spec.Volumes = []v12.Volume{
			{
				Name: "disk0",
				VolumeSource: v12.VolumeSource{
					ContainerDisk: &v12.ContainerDiskSource{
						Image: "test/image",
						Path:  "/disk.img",
					},
				},
			},
			{
				Name: "cdrom0",
				VolumeSource: v12.VolumeSource{
					CloudInitNoCloud: &v12.CloudInitNoCloudSource{
						UserDataSecretRef: &v1.LocalObjectReference{
							Name: "testsecret",
						},
						NetworkDataSecretRef: &v1.LocalObjectReference{
							Name: "testnetworksecret",
						},
					},
				},
			},
		}
		exampleVMI.Spec.Domain.Features = &v12.Features{
			ACPI:       v12.FeatureState{Enabled: pointer.BoolPtr(false)},
			SMM:        &v12.FeatureState{Enabled: pointer.BoolPtr(true)},
			APIC:       &v12.FeatureAPIC{Enabled: pointer.BoolPtr(true)},
			KVM:        &v12.FeatureKVM{Hidden: true},
			Pvspinlock: &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
			Hyperv: &v12.FeatureHyperv{
				Relaxed:         &v12.FeatureState{Enabled: pointer.BoolPtr(true)},
				VAPIC:           &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				Spinlocks:       &v12.FeatureSpinlocks{Enabled: pointer.BoolPtr(true)},
				VPIndex:         &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				Runtime:         &v12.FeatureState{Enabled: pointer.BoolPtr(true)},
				SyNIC:           &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				SyNICTimer:      &v12.SyNICTimer{Enabled: pointer.BoolPtr(true), Direct: &v12.FeatureState{Enabled: pointer.BoolPtr(true)}},
				Reset:           &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				VendorID:        &v12.FeatureVendorID{Enabled: pointer.BoolPtr(true), VendorID: "vendor"},
				Frequencies:     &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				Reenlightenment: &v12.FeatureState{Enabled: pointer.BoolPtr(false)},
				TLBFlush:        &v12.FeatureState{Enabled: pointer.BoolPtr(true)},
			},
		}
		exampleVMI.Spec.Domain.Clock = &v12.Clock{
			ClockOffset: v12.ClockOffset{
				UTC: &v12.ClockOffsetUTC{},
			},
			Timer: &v12.Timer{
				HPET:   &v12.HPETTimer{},
				KVM:    &v12.KVMTimer{},
				PIT:    &v12.PITTimer{},
				RTC:    &v12.RTCTimer{},
				Hyperv: &v12.HypervTimer{},
			},
		}
		exampleVMI.Spec.Domain.Firmware = &v12.Firmware{
			UUID: "28a42a60-44ef-4428-9c10-1a6aee94627f",
		}
		exampleVMI.Spec.Domain.CPU = &v12.CPU{
			Cores:   3,
			Sockets: 1,
			Threads: 1,
			Model:   "Conroe",
			Features: []v12.CPUFeature{
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
		exampleVMI.Spec.Networks = []v12.Network{
			v12.Network{
				Name: "default",
				NetworkSource: v12.NetworkSource{
					Pod: &v12.PodNetwork{},
				},
			},
		}

		policy := v12.IOThreadsPolicyShared
		exampleVMI.Spec.Domain.IOThreadsPolicy = &policy

		v12.SetObjectDefaults_VirtualMachineInstance(exampleVMI)
	})
	Context("With example schema in json use pod network and bridge interface", func() {
		It("Unmarshal json into struct", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []v12.Interface{
				v12.Interface{
					Name: "default",
					InterfaceBindingMethod: v12.InterfaceBindingMethod{
						Bridge: &v12.InterfaceBridge{},
					},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"bridge": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).ToNot(HaveOccurred())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).ToNot(HaveOccurred())
			newVMI := &v12.VirtualMachineInstance{}
			err = json.Unmarshal(tpl.Bytes(), newVMI)
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI).To(Equal(exampleVMI))
		})
		It("Marshal struct into json", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []v12.Interface{
				v12.Interface{
					Name: "default",
					InterfaceBindingMethod: v12.InterfaceBindingMethod{
						Bridge: &v12.InterfaceBridge{},
					},
				},
			}

			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"bridge": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).ToNot(HaveOccurred())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).ToNot(HaveOccurred())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
	})
	Context("With example schema in json use pod network and slirp interface", func() {
		It("Unmarshal json into struct", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []v12.Interface{
				v12.Interface{
					Name: "default",
					InterfaceBindingMethod: v12.InterfaceBindingMethod{
						DeprecatedSlirp: &v12.DeprecatedInterfaceSlirp{},
					},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).ToNot(HaveOccurred())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).ToNot(HaveOccurred())
			newVMI := &v12.VirtualMachineInstance{}
			err = json.Unmarshal(tpl.Bytes(), newVMI)
			Expect(err).ToNot(HaveOccurred())
			Expect(newVMI).To(Equal(exampleVMI))
		})
		It("Marshal struct into json", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []v12.Interface{
				v12.Interface{
					Name: "default",
					InterfaceBindingMethod: v12.InterfaceBindingMethod{
						DeprecatedSlirp: &v12.DeprecatedInterfaceSlirp{},
					},
				},
			}

			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {}`}
			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).ToNot(HaveOccurred())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).ToNot(HaveOccurred())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
		It("Marshal struct into json with port configure", func() {
			exampleVMI.Spec.Domain.Devices.Interfaces = []v12.Interface{
				v12.Interface{
					Name: "default",
					InterfaceBindingMethod: v12.InterfaceBindingMethod{
						DeprecatedSlirp: &v12.DeprecatedInterfaceSlirp{}},
					Ports: []v12.Port{{Port: 80}},
				},
			}
			networkTemplateData := NetworkTemplateConfig{InterfaceConfig: `"slirp": {},
            "ports": [
              {
                "port": 80
              }
            ]`}

			tmpl, err := template.New("vmexample").Parse(exampleJSON)
			Expect(err).ToNot(HaveOccurred())
			var tpl bytes.Buffer
			err = tmpl.Execute(&tpl, networkTemplateData)
			Expect(err).ToNot(HaveOccurred())
			exampleJSONParsed := tpl.String()
			buf, err := json.MarshalIndent(*exampleVMI, "", "  ")
			Expect(err).ToNot(HaveOccurred())
			Expect(string(buf)).To(Equal(exampleJSONParsed))
		})
	})
})
