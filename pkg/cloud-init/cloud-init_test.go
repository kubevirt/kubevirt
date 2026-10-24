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

package cloudinit

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	cloudinitdata "kubevirt.io/kubevirt/pkg/libvmi/cloudinit"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

var _ = Describe("CloudInit", func() {

	var (
		isoCreationFunc IsoCreationFunc
		tmpDir          string
	)

	BeforeEach(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "cloudinittest")
		Expect(err).ToNot(HaveOccurred())
		err = SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		isoCreationFunc = func(isoOutFile, volumeID string, inDir string) error {
			switch volumeID {
			case "cidata", "config-2":
				// Valid volume IDs for nocloud and configdrive
			default:
				return fmt.Errorf("unexpected volume ID '%s'", volumeID)
			}

			// fake creating the iso
			_, err := os.Create(isoOutFile)

			return err
		}
	})

	JustBeforeEach(func() {
		SetIsoCreationFunction(isoCreationFunc)
	})

	AfterEach(func() {
		os.RemoveAll(tmpDir)
	})
	Describe("Data Model", func() {
		Context("verify meta-data model", func() {
			It("should match the generated configdrive metadata", func() {
				exampleJSONParsed := `{
  "instance_type": "fake.fake-instancetype",
  "instance_id": "fake.fake-namespace",
  "local_hostname": "fake",
  "uuid": "fake.fake-namespace",
  "devices": [
    {
      "type": "nic",
      "bus": "pci",
      "address": "0000:01:00:0",
      "mac": "02:00:00:84:e9:58",
      "tags": [
        "testtag"
      ]
    }
  ],
  "public_keys": {
    "0": "somekey"
  }
}`
				devices := []DeviceData{
					{
						Type:    NICMetadataType,
						Bus:     api.AddressPCI,
						Address: "0000:01:00:0",
						MAC:     "02:00:00:84:e9:58",
						Tags:    []string{"testtag"},
					},
				}

				metadataStruct := ConfigDriveMetadata{
					InstanceType:  "fake.fake-instancetype",
					InstanceID:    "fake.fake-namespace",
					LocalHostname: "fake",
					UUID:          "fake.fake-namespace",
					Devices:       &devices,
					PublicSSHKeys: map[string]string{"0": "somekey"},
				}
				buf, err := json.MarshalIndent(metadataStruct, "", "  ")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(exampleJSONParsed))
			})
			It("should match the generated configdrive metadata for hostdev with numaNode", func() {
				exampleJSONParsed := `{
  "instance_id": "fake.fake-namespace",
  "local_hostname": "fake",
  "uuid": "fake.fake-namespace",
  "devices": [
    {
      "type": "hostdev",
      "bus": "pci",
      "address": "0000:65:10:0",
      "numaNode": 1,
      "alignedCPUs": [
        0,
        1
      ],
      "tags": [
        "testtag1"
      ]
    }
  ],
  "public_keys": {
    "0": "somekey"
  }
}`
				devices := []DeviceData{
					{
						Type:        HostDevMetadataType,
						Bus:         api.AddressPCI,
						Address:     "0000:65:10:0",
						MAC:         "",
						NumaNode:    uint32(1),
						AlignedCPUs: []uint32{0, 1},
						Tags:        []string{"testtag1"},
					},
				}

				metadataStruct := ConfigDriveMetadata{
					InstanceID:    "fake.fake-namespace",
					LocalHostname: "fake",
					UUID:          "fake.fake-namespace",
					Devices:       &devices,
					PublicSSHKeys: map[string]string{"0": "somekey"},
				}
				buf, err := json.MarshalIndent(metadataStruct, "", "  ")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(exampleJSONParsed))
			})
			It("should match the generated nocloud metadata", func() {
				exampleJSONParsed := `{
  "instance-type": "fake.fake-instancetype",
  "instance-id": "fake.fake-namespace",
  "local-hostname": "fake"
}`

				metadataStruct := NoCloudMetadata{
					InstanceType:  "fake.fake-instancetype",
					InstanceID:    "fake.fake-namespace",
					LocalHostname: "fake",
				}
				buf, err := json.MarshalIndent(metadataStruct, "", "  ")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(exampleJSONParsed))
			})
		})
	})
	Describe("Volume-based data source", func() {
		Context("when ISO generation fails", func() {
			timedOut := false

			BeforeEach(func() {
				isoCreationFunc = func(isoOutFile, volumeID string, inDir string) error {
					var args []string

					args = append(args, "10")
					cmd := exec.Command("sleep", args...)

					err := cmd.Start()
					if err != nil {
						return err
					}

					done := make(chan error)
					go func() { done <- cmd.Wait() }()

					timeout := time.After(1 * time.Second)

					for {
						select {
						case <-timeout:
							cmd.Process.Kill()
							timedOut = true
						case err := <-done:
							if err != nil {
								return err
							}
							return nil
						}
					}
				}
			})

			It("should fail local data generation", func() {
				userData := "fake\nuser\ndata\n"
				source := &v1.CloudInitNoCloudSource{
					UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
				}
				cloudInitData, _ := readCloudInitNoCloudSource(source)

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fake-domain",
						Namespace: "fake-namespace",
					},
				}
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				err := GenerateIso(vmi, cloudInitData)
				Expect(err).To(HaveOccurred())
				Expect(timedOut).To(BeTrue())
			})
		})

		Describe("A new VirtualMachineInstance definition", func() {
			verifyCloudInitData := func(cloudInitData *CloudInitData) {
				domain := "fake-domain"
				namespace := "fake-namespace"

				vmi := &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      domain,
						Namespace: namespace,
					},
				}
				v1.SetObjectDefaults_VirtualMachineInstance(vmi)
				err := GenerateIso(vmi, cloudInitData)
				Expect(err).ToNot(HaveOccurred())

				// verify iso is created
				var isoFile string
				switch cloudInitData.DataSource {
				case DataSourceNoCloud:
					isoFile = noCloudFile
				case DataSourceConfigDrive:
					isoFile = configDriveFile
				default:
					panic(fmt.Errorf("Invalid data source '%s'", cloudInitData.DataSource))
				}
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s/%s", tmpDir, namespace, domain, isoFile))
				Expect(err).ToNot(HaveOccurred())

				err = removeLocalData(domain, namespace)
				Expect(err).ToNot(HaveOccurred())

				// verify iso and entire dir is deleted
				_, err = os.Stat(fmt.Sprintf("%s/%s/%s", tmpDir, namespace, domain))
				if errors.Is(err, os.ErrNotExist) {
					err = nil
				}
				Expect(err).ToNot(HaveOccurred())
			}

			Context("with CloudInitNoCloud volume source", func() {
				verifyCloudInitNoCloudIso := func(source *v1.CloudInitNoCloudSource) {
					cloudInitData, _ := readCloudInitNoCloudSource(source)
					verifyCloudInitData(cloudInitData)
				}

				It("should succeed to verify userDataBase64 ", func() {
					userData := "fake\nuser\ndata\n"
					cloudInitData := &v1.CloudInitNoCloudSource{
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
					}
					verifyCloudInitNoCloudIso(cloudInitData)
				})

				It("should succeed to verify userDataBase64 and networkData", func() {
					userData := "fake\nuser\ndata\n"
					networkData := "fake\nnetwork\ndata\n"
					cloudInitData := &v1.CloudInitNoCloudSource{
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						NetworkData:    networkData,
					}
					verifyCloudInitNoCloudIso(cloudInitData)
				})

				It("should succeed to verify userDataBase64 and networkDataBase64", func() {
					userData := "fake\nuser\ndata\n"
					networkData := "fake\nnetwork\ndata\n"
					cloudInitData := &v1.CloudInitNoCloudSource{
						UserDataBase64:    base64.StdEncoding.EncodeToString([]byte(userData)),
						NetworkDataBase64: base64.StdEncoding.EncodeToString([]byte(networkData)),
					}
					verifyCloudInitNoCloudIso(cloudInitData)
				})

				It("should succeed to verify userData", func() {
					userData := "fake\nuser\ndata\n"
					cloudInitData := &v1.CloudInitNoCloudSource{
						UserData: userData,
					}
					verifyCloudInitNoCloudIso(cloudInitData)
				})

				It("should succeed to verify networkData if there is no userData", func() {
					networkData := "fake\nnetwork\ndata\n"
					cloudInitData := &v1.CloudInitNoCloudSource{
						NetworkData: networkData,
					}
					verifyCloudInitNoCloudIso(cloudInitData)
				})

				It("should fail to verify bad cloudInitNoCloud UserDataBase64", func() {
					source := &v1.CloudInitNoCloudSource{
						UserDataBase64: "#######garbage******",
					}
					_, err := readCloudInitNoCloudSource(source)
					Expect(err.Error()).Should(Equal("illegal base64 data at input byte 0"))
				})

				It("should fail to verify bad cloudInitNoCloud NetworkDataBase64", func() {
					source := &v1.CloudInitNoCloudSource{
						UserData:          "fake",
						NetworkDataBase64: "#######garbage******",
					}
					_, err := readCloudInitNoCloudSource(source)
					Expect(err.Error()).Should(Equal("illegal base64 data at input byte 0"))
				})

				It("should fail to verify if there is no userData nor networkData", func() {
					source := &v1.CloudInitNoCloudSource{}
					_, err := readCloudInitNoCloudSource(source)
					Expect(err).Should(MatchError("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source"))
				})

			})

			Context("with CloudInitConfigDrive volume source", func() {
				verifyCloudInitConfigDriveIso := func(source *v1.CloudInitConfigDriveSource) {
					cloudInitData, _ := readCloudInitConfigDriveSource(source)
					verifyCloudInitData(cloudInitData)
				}

				It("should succeed to verify userDataBase64 ", func() {
					userData := "fake\nuser\ndata\n"
					cloudInitData := &v1.CloudInitConfigDriveSource{
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
					}
					verifyCloudInitConfigDriveIso(cloudInitData)
				})

				It("should succeed to verify userDataBase64 and networkData", func() {
					userData := "fake\nuser\ndata\n"
					networkData := "fake\nnetwork\ndata\n"
					cloudInitData := &v1.CloudInitConfigDriveSource{
						UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
						NetworkData:    networkData,
					}
					verifyCloudInitConfigDriveIso(cloudInitData)
				})

				It("should succeed to verify userDataBase64 and networkDataBase64", func() {
					userData := "fake\nuser\ndata\n"
					networkData := "fake\nnetwork\ndata\n"
					cloudInitData := &v1.CloudInitConfigDriveSource{
						UserDataBase64:    base64.StdEncoding.EncodeToString([]byte(userData)),
						NetworkDataBase64: base64.StdEncoding.EncodeToString([]byte(networkData)),
					}
					verifyCloudInitConfigDriveIso(cloudInitData)
				})

				It("should succeed to verify userData", func() {
					userData := "fake\nuser\ndata\n"
					cloudInitData := &v1.CloudInitConfigDriveSource{
						UserData: userData,
					}
					verifyCloudInitConfigDriveIso(cloudInitData)
				})

				It("should fail to verify bad cloudInitNoCloud UserDataBase64", func() {
					source := &v1.CloudInitConfigDriveSource{
						UserDataBase64: "#######garbage******",
					}
					_, err := readCloudInitConfigDriveSource(source)
					Expect(err.Error()).Should(Equal("illegal base64 data at input byte 0"))
				})

				It("should fail to verify bad cloudInitNoCloud NetworkDataBase64", func() {
					source := &v1.CloudInitConfigDriveSource{
						UserData:          "fake",
						NetworkDataBase64: "#######garbage******",
					}
					_, err := readCloudInitConfigDriveSource(source)
					Expect(err.Error()).Should(Equal("illegal base64 data at input byte 0"))
				})

			})
		})
	})

	Describe("GenerateIso", func() {
		It("does not modify ConfigDrive metadata when adding devices", func() {
			vmi := libvmi.New(libvmi.WithName("guest"), libvmi.WithNamespace("default"))
			metadata := &ConfigDriveMetadata{InstanceID: "provided"}
			devices := []DeviceData{{Type: NICMetadataType, Bus: "pci"}}
			data := &CloudInitData{
				DataSource:          DataSourceConfigDrive,
				UserData:            "fake\nuser\ndata\n",
				ConfigDriveMetaData: metadata,
				DevicesData:         &devices,
			}

			Expect(GenerateIso(vmi, data)).To(Succeed())
			Expect(data.ConfigDriveMetaData).To(BeIdenticalTo(metadata))
			Expect(metadata.Devices).To(BeNil())
		})

		It("should cleanly run twice", func() {
			userData := "fake\nuser\ndata\n"

			vmi := &v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fake-domain",
					Namespace: "fake-namespace",
				},
			}
			source := &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			}
			cloudInitData, err := readCloudInitNoCloudSource(source)
			Expect(err).NotTo(HaveOccurred())
			v1.SetObjectDefaults_VirtualMachineInstance(vmi)
			err = GenerateIso(vmi, cloudInitData)
			Expect(err).NotTo(HaveOccurred())
			err = GenerateIso(vmi, cloudInitData)
			Expect(err).NotTo(HaveOccurred())

			entries, err := os.ReadDir(getDomainBasePath(vmi.Name, vmi.Namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Name()).To(Equal(noCloudFile))
		})
	})

	Describe("PrepareLocalPath", func() {
		It("should create the correct directory structure", func() {
			namespace := "fake-namespace"
			domain := "fake-domain"
			expectedPath := filepath.Join(tmpDir, namespace, domain)
			err := PrepareLocalPath(domain, namespace)
			Expect(err).ToNot(HaveOccurred())
			_, err = os.Stat(expectedPath)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func removeLocalData(domain string, namespace string) error {
	domainBasePath := getDomainBasePath(domain, namespace)
	err := os.RemoveAll(domainBasePath)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

var _ = Describe("ReadCloudInitVolumeDataSource", func() {
	var root string
	BeforeEach(func() { root = GinkgoT().TempDir() })
	It("returns nil when the VMI has no cloud-init volume", func() {
		vmi := libvmi.New()
		data, err := ReadCloudInitVolumeDataSource(vmi, root)
		Expect(err).NotTo(HaveOccurred())
		Expect(data).To(BeNil())
	})
	Context("NoCloud", func() {
		It("reads raw user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceNoCloud))
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("decodes base64 user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudEncodedUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudEncodedNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers raw payloads to malformed base64", func() {
			// Preserve existing raw-data precedence during this refactor.
			// Returning an error for malformed base64 would be more prudent
			// than silently ignoring it when raw data is also present.
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			volume := libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud
			volume.UserDataBase64 = "!"
			volume.NetworkDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("accepts network-only source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(BeEmpty())
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("rejects empty source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed user base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud.UserDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed network base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitNoCloud.NetworkDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("uses userData when userdata cannot be read", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userData", "secret-user")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("secret-user"))
		})
		It("prefers lowercase networkdata when both network filenames contain data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudNetworkDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "primary-network")
			writeFixture(root, "cloudinitdisk/networkData", "secondary-network")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NetworkData).To(Equal("primary-network"))
		})
		It("keeps inline data when the only requested secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("rejects two missing secret payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("missing-network-secret"),
				),
			)
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("no cloud-init data-source found at volume: cloudinitdisk"))
		})
		It("accepts a network secret when the user secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("network-secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "fake\nnetwork\ndata\n")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers lowercase cloudinitdisk/userdata even if empty", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("inline"),
					cloudinitdata.WithNoCloudUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userdata", "")
			writeFixture(root, "cloudinitdisk/userData", "ignored")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("orders selected public keys and ignores projected bookkeeping", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("first"),
				withConfigDriveSSHPublicKey("ignored"),
				withAccessCredential(v1.AccessCredential{}),
				withNoCloudSSHPublicKey(""),
				withNoCloudSSHPublicKey("second"),
			)

			writeFixture(root, "first-access-cred/z", "last")
			writeFixture(root, "first-access-cred/a", "first")
			writeFixture(root, "first-access-cred/empty", "")
			writeFixture(root, "second-access-cred/..version/key", "second")
			symlinkFixture(root, "second-access-cred/..data", "..version")
			symlinkFixture(root, "second-access-cred/link", "..data/key")
			symlinkFixture(root, "first-access-cred/..data", "missing")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.PublicSSHKeys).To(Equal(map[string]string{"0": "first", "1": "last", "2": "second"}))
		})
		It("returns a credential directory read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("keys"),
			)

			writeFixture(root, "keys-access-cred", "regular file")
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(filepath.Join(root, "keys-access-cred")))
		})
		It("reports the path of a credential file read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
				withNoCloudSSHPublicKey("keys"),
			)

			Expect(os.Mkdir(filepath.Join(root, "keys-access-cred"), 0700)).To(Succeed())
			symlinkFixture(root, "keys-access-cred/public", "missing")
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to read public keys found at volume: " + filepath.Join(root, "keys-access-cred/public")))
		})
	})
	Context("ConfigDrive", func() {
		It("reads raw user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(cloudinit.DataSourceConfigDrive))
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("decodes base64 user and network payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveEncodedUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveEncodedNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers raw payloads to malformed base64", func() {
			// Preserve existing raw-data precedence during this refactor.
			// Returning an error for malformed base64 would be more prudent
			// than silently ignoring it when raw data is also present.
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			volume := libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive
			volume.UserDataBase64 = "!"
			volume.NetworkDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("fake\nuser\ndata\n"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("accepts network-only source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(BeEmpty())
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("rejects empty source data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("userDataBase64, userData, networkDataBase64 or networkData is required for a cloud-init data source"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed user base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkData("fake\nnetwork\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive.UserDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("rejects malformed network base64", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			libvmi.GetCloudInitVolume(vmi).CloudInitConfigDrive.NetworkDataBase64 = "!"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("illegal base64 data at input byte 0"))
			Expect(data.VolumeName).To(Equal(libvmi.CloudInitDiskName))
		})
		It("uses userData when userdata cannot be read", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userData", "secret-user")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("secret-user"))
		})
		It("prefers lowercase networkdata when both network filenames contain data", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveNetworkDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "primary-network")
			writeFixture(root, "cloudinitdisk/networkData", "secondary-network")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NetworkData).To(Equal("primary-network"))
		})
		It("keeps inline data when the only requested secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("rejects two missing secret payloads", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("missing-network-secret"),
				),
			)
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(MatchError("no cloud-init data-source found at volume: cloudinitdisk"))
		})
		It("accepts a network secret when the user secret is missing", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("missing-user-secret"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("network-secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/networkdata", "fake\nnetwork\ndata\n")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
			Expect(data.NetworkData).To(Equal("fake\nnetwork\ndata\n"))
		})
		It("prefers lowercase cloudinitdisk/userdata even if empty", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("inline"),
					cloudinitdata.WithConfigDriveUserDataSecretName("secret"),
				),
			)
			writeFixture(root, "cloudinitdisk/userdata", "")
			writeFixture(root, "cloudinitdisk/userData", "ignored")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.UserData).To(Equal("inline"))
		})
		It("orders selected public keys and ignores projected bookkeeping", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("first"),
				withNoCloudSSHPublicKey("ignored"),
				withAccessCredential(v1.AccessCredential{}),
				withConfigDriveSSHPublicKey(""),
				withConfigDriveSSHPublicKey("second"),
			)

			writeFixture(root, "first-access-cred/z", "last")
			writeFixture(root, "first-access-cred/a", "first")
			writeFixture(root, "first-access-cred/empty", "")
			writeFixture(root, "second-access-cred/..version/key", "second")
			symlinkFixture(root, "second-access-cred/..data", "..version")
			symlinkFixture(root, "second-access-cred/link", "..data/key")
			symlinkFixture(root, "first-access-cred/..data", "missing")
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.PublicSSHKeys).To(Equal(map[string]string{"0": "first", "1": "last", "2": "second"}))
		})
		It("returns a credential directory read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("keys"),
			)

			writeFixture(root, "keys-access-cred", "regular file")
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(filepath.Join(root, "keys-access-cred")))
		})
		It("reports the path of a credential file read error", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
				withConfigDriveSSHPublicKey("keys"),
			)

			Expect(os.Mkdir(filepath.Join(root, "keys-access-cred"), 0700)).To(Succeed())
			symlinkFixture(root, "keys-access-cred/public", "missing")
			_, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Unable to read public keys found at volume: " + filepath.Join(root, "keys-access-cred/public")))
		})
	})
	Context("NoCloud metadata", func() {
		It("uses the firmware UUID in metadata", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceID).To(Equal("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"))
		})
		It("prefers the cluster instancetype annotation to the namespaced annotation", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithAnnotation(v1.ClusterInstancetypeAnnotation, "cluster"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceType).To(Equal("cluster"))
		})
		It("uses the namespaced instancetype when no cluster annotation exists", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceType).To(Equal("namespaced"))
		})
		It("preserves an explicitly empty firmware UUID when reading", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID(""),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.InstanceID).To(BeEmpty())
		})
		It("uses the VMI name without its domain suffix as the metadata hostname", func() {
			vmi := libvmi.New(
				libvmi.WithName("guest.example.com"),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal("guest"))
		})
		It("uses the VMI name truncated to 63 characters as the metadata hostname", func() {
			// Preserve existing silent truncation during this refactor. Returning
			// an error would be more prudent than discarding the hostname's tail.
			vmi := libvmi.New(
				libvmi.WithName(strings.Repeat("a", 70)),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal(strings.Repeat("a", 63)))
		})
		It("honors an explicitly configured hostname", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserData("fake\nuser\ndata\n"),
				),
			)
			vmi.Spec.Hostname = "guest-host"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.NoCloudMetaData.LocalHostname).To(Equal("guest-host"))
		})
	})
	Context("ConfigDrive metadata", func() {
		It("uses the firmware UUID in metadata", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.UUID).To(Equal("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"))
		})
		It("prefers the cluster instancetype annotation to the namespaced annotation", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithAnnotation(v1.ClusterInstancetypeAnnotation, "cluster"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.InstanceType).To(Equal("cluster"))
		})
		It("uses the namespaced instancetype when no cluster annotation exists", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithAnnotation(v1.InstancetypeAnnotation, "namespaced"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.InstanceType).To(Equal("namespaced"))
		})
		It("preserves an explicitly empty firmware UUID when reading", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID(""),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.UUID).To(BeEmpty())
		})
		It("uses the VMI name without its domain suffix as the metadata hostname", func() {
			vmi := libvmi.New(
				libvmi.WithName("guest.example.com"),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal("guest"))
		})
		It("uses the VMI name truncated to 63 characters as the metadata hostname", func() {
			// Preserve existing silent truncation during this refactor. Returning
			// an error would be more prudent than discarding the hostname's tail.
			vmi := libvmi.New(
				libvmi.WithName(strings.Repeat("a", 70)),
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal(strings.Repeat("a", 63)))
		})
		It("honors an explicitly configured hostname", func() {
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserData("fake\nuser\ndata\n"),
				),
			)
			vmi.Spec.Hostname = "guest-host"
			data, err := ReadCloudInitVolumeDataSource(vmi, root)
			Expect(err).NotTo(HaveOccurred())
			Expect(data.ConfigDriveMetaData.Hostname).To(Equal("guest-host"))
		})
	})
	It("returns data from the first cloud-init volume", func() {
		vmi := libvmi.New(
			libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
			libvmi.WithCloudInitNoCloud(cloudinitdata.WithNoCloudUserData("first")),
		)
		second := libvmi.New(libvmi.WithCloudInitNoCloud(
			cloudinitdata.WithNoCloudUserData("second"),
			cloudinitdata.WithNoCloudUserDataSecretName("secret"),
		))
		secondVolume := *libvmi.GetCloudInitVolume(second)
		secondVolume.Name = "second"
		third := libvmi.New(libvmi.WithCloudInitConfigDrive(
			cloudinitdata.WithConfigDriveUserData("third"),
		))
		thirdVolume := *libvmi.GetCloudInitVolume(third)
		thirdVolume.Name = "third"
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, secondVolume, thirdVolume)
		writeFixture(root, "second/userdata", "resolved")
		data, err := ReadCloudInitVolumeDataSource(vmi, root)
		Expect(err).NotTo(HaveOccurred())
		Expect(data.UserData).To(Equal("first"))
		Expect(data.DataSource).To(Equal(DataSourceNoCloud))
	})

})

func withAccessCredential(credential v1.AccessCredential) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.AccessCredentials = append(vmi.Spec.AccessCredentials, credential)
	}
}

func withNoCloudSSHPublicKey(secretName string) libvmi.Option {
	return withAccessCredential(v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{SecretName: secretName},
			},
			PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
				NoCloud: &v1.NoCloudSSHPublicKeyAccessCredentialPropagation{},
			},
		},
	})
}

func withConfigDriveSSHPublicKey(secretName string) libvmi.Option {
	return withAccessCredential(v1.AccessCredential{
		SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
			Source: v1.SSHPublicKeyAccessCredentialSource{
				Secret: &v1.AccessCredentialSecretSource{SecretName: secretName},
			},
			PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
				ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
			},
		},
	})
}

func writeFixture(root, name, data string) {
	GinkgoHelper()
	path := filepath.Join(root, name)
	Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
	Expect(os.WriteFile(path, []byte(data), 0600)).To(Succeed())
}

func symlinkFixture(root, name, target string) {
	GinkgoHelper()
	path := filepath.Join(root, name)
	Expect(os.MkdirAll(filepath.Dir(path), 0700)).To(Succeed())
	Expect(os.Symlink(target, path)).To(Succeed())
}

var _ = Describe("ReadCloudInitVolumeDataSource", func() {
	Context("NoCloud", func() {
		It("reads user and network data from mounted secrets", func() {
			root := GinkgoT().TempDir()
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitNoCloud(
					cloudinitdata.WithNoCloudUserDataSecretName("user"),
					cloudinitdata.WithNoCloudNetworkDataSecretName("network"),
				),
			)
			writeFixture(root, "cloudinitdisk/..version/userdata", "secret-user")
			symlinkFixture(root, "cloudinitdisk/..data", "..version")
			symlinkFixture(root, "cloudinitdisk/userdata", "..data/userdata")
			writeFixture(root, "cloudinitdisk/networkdata", "secret-network")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(DataSourceNoCloud))
			Expect(data.UserData).To(Equal("secret-user"))
			Expect(data.NetworkData).To(Equal("secret-network"))
		})
	})

	Context("ConfigDrive", func() {
		It("reads user and network data from mounted secrets", func() {
			root := GinkgoT().TempDir()
			vmi := libvmi.New(
				libvmi.WithFirmwareUUID("cf73b4f4-6975-4d93-b51c-b7ae392bdea5"),
				libvmi.WithCloudInitConfigDrive(
					cloudinitdata.WithConfigDriveUserDataSecretName("user"),
					cloudinitdata.WithConfigDriveNetworkDataSecretName("network"),
				),
			)
			writeFixture(root, "cloudinitdisk/..version/userdata", "secret-user")
			symlinkFixture(root, "cloudinitdisk/..data", "..version")
			symlinkFixture(root, "cloudinitdisk/userdata", "..data/userdata")
			writeFixture(root, "cloudinitdisk/networkdata", "secret-network")

			data, err := ReadCloudInitVolumeDataSource(vmi, root)

			Expect(err).NotTo(HaveOccurred())
			Expect(data.DataSource).To(Equal(DataSourceConfigDrive))
			Expect(data.UserData).To(Equal("secret-user"))
			Expect(data.NetworkData).To(Equal("secret-network"))
		})
	})
})
