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

package cloudinit

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("CloudInit", func() {

	var (
		isoCreationFunc IsoCreationFunc
		tmpDir          string
	)

	createEmptyVMIWithVolumes := func(volumes []v1.Volume) *v1.VirtualMachineInstance {
		return &v1.VirtualMachineInstance{
			Spec: v1.VirtualMachineInstanceSpec{
				Volumes: volumes,
			},
		}
	}

	fakeVolumeMountDir := func(dirName string, files map[string]string) string {
		volumeDir := filepath.Join(tmpDir, dirName)
		err := os.Mkdir(volumeDir, 0700)
		Expect(err).To(Not(HaveOccurred()), "could not create volume dir: ", volumeDir)
		for fileName, content := range files {
			err = ioutil.WriteFile(
				filepath.Join(volumeDir, fileName),
				[]byte(content),
				0644)
			Expect(err).To(Not(HaveOccurred()), "could not create file: ", fileName)
		}
		return volumeDir
	}

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "cloudinittest")
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
  "instance_id": "fake.fake-namespace",
  "local_hostname": "fake",
  "uuid": "5d307ca9-b3ef-428c-8861-06e72d69f223",
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
						Bus:     "pci",
						Address: "0000:01:00:0",
						MAC:     "02:00:00:84:e9:58",
						Tags:    []string{"testtag"},
					},
				}

				metadataStruct := ConfigDriveMetadata{
					InstanceID:    "fake.fake-namespace",
					LocalHostname: "fake",
					UUID:          "5d307ca9-b3ef-428c-8861-06e72d69f223",
					Devices:       &devices,
					PublicSSHKeys: map[string]string{"0": "somekey"},
				}
				buf, err := json.MarshalIndent(metadataStruct, "", "  ")
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal(exampleJSONParsed))
			})
			It("should match the generated nocloud metadata", func() {
				exampleJSONParsed := `{
  "instance-id": "fake.fake-namespace",
  "local-hostname": "fake"
}`

				metadataStruct := NoCloudMetadata{
					InstanceID:    "fake.fake-namespace",
					LocalHostname: "fake",
				}
				buf, err := json.MarshalIndent(metadataStruct, "", "  ")
				Expect(err).To(BeNil())
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
				namespace := "fake-namespace"
				domain := "fake-domain"
				userData := "fake\nuser\ndata\n"
				source := &v1.CloudInitNoCloudSource{
					UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
				}
				cloudInitData, _ := readCloudInitNoCloudSource(source)
				err := GenerateLocalData(domain, namespace, cloudInitData)
				Expect(err).To(HaveOccurred())
				Expect(timedOut).To(BeTrue())
			})

			Context("when local data does not exist", func() {
				It("should fail to remove local data", func() {
					namespace := "fake-namespace"
					domain := "fake-domain"
					err := removeLocalData(domain, namespace)
					Expect(err).ToNot(HaveOccurred())
				})
			})
		})

		Describe("A new VirtualMachineInstance definition", func() {
			verifyCloudInitData := func(cloudInitData *CloudInitData) {
				namespace := "fake-namespace"
				domain := "fake-domain"

				err := GenerateLocalData(domain, namespace, cloudInitData)
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
				if os.IsNotExist(err) {
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

				Context("with secretRefs", func() {
					createCloudInitSecretRefVolume := func(name, secret string) *v1.Volume {
						return &v1.Volume{
							Name: name,
							VolumeSource: v1.VolumeSource{
								CloudInitNoCloud: &v1.CloudInitNoCloudSource{
									UserDataSecretRef: &k8sv1.LocalObjectReference{
										Name: secret,
									},
								},
							},
						}
					}

					It("should resolve no-cloud data from volume", func() {
						testVolume := createCloudInitSecretRefVolume("test-volume", "test-secret")
						vmi := createEmptyVMIWithVolumes([]v1.Volume{*testVolume})
						fakeVolumeMountDir("test-volume", map[string]string{
							"userdata":    "secret-userdata",
							"networkdata": "secret-networkdata",
						})
						err := resolveNoCloudSecrets(vmi, tmpDir)
						Expect(err).To(Not(HaveOccurred()), "could not resolve secret volume")
						Expect(testVolume.CloudInitNoCloud.UserData).To(Equal("secret-userdata"))
						Expect(testVolume.CloudInitNoCloud.NetworkData).To(Equal("secret-networkdata"))
					})

					It("should resolve camel-case no-cloud data from volume", func() {
						testVolume := createCloudInitSecretRefVolume("test-volume", "test-secret")
						vmi := createEmptyVMIWithVolumes([]v1.Volume{*testVolume})
						fakeVolumeMountDir("test-volume", map[string]string{
							"userData":    "secret-userdata",
							"networkData": "secret-networkdata",
						})
						err := resolveNoCloudSecrets(vmi, tmpDir)
						Expect(err).To(Not(HaveOccurred()), "could not resolve secret volume")
						Expect(testVolume.CloudInitNoCloud.UserData).To(Equal("secret-userdata"))
						Expect(testVolume.CloudInitNoCloud.NetworkData).To(Equal("secret-networkdata"))
					})

					It("should resolve empty no-cloud volume and do nothing", func() {
						vmi := createEmptyVMIWithVolumes([]v1.Volume{})
						err := resolveNoCloudSecrets(vmi, tmpDir)
						Expect(err).To(Not(HaveOccurred()), "failed to resolve empty volumes")
					})

					It("should fail if both userdata and network data does not exist", func() {
						testVolume := createCloudInitSecretRefVolume("test-volume", "test-secret")
						vmi := createEmptyVMIWithVolumes([]v1.Volume{*testVolume})
						err := resolveNoCloudSecrets(vmi, tmpDir)
						Expect(err).To(HaveOccurred(), "expected a failure when no sources found")
						Expect(err.Error()).To(Equal("no cloud-init data-source found at volume: test-volume"))
					})
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

				Context("with secretRefs", func() {
					createCloudInitConfigDriveVolume := func(name, secret string) *v1.Volume {
						return &v1.Volume{
							Name: name,
							VolumeSource: v1.VolumeSource{
								CloudInitConfigDrive: &v1.CloudInitConfigDriveSource{
									UserDataSecretRef: &k8sv1.LocalObjectReference{
										Name: secret,
									},
								},
							},
						}
					}
					It("should resolve config-drive data from volume", func() {
						testVolume := createCloudInitConfigDriveVolume("test-volume", "test-secret")
						vmi := createEmptyVMIWithVolumes([]v1.Volume{*testVolume})

						vmi.Spec.AccessCredentials = []v1.AccessCredential{
							{
								SSHPublicKey: &v1.SSHPublicKeyAccessCredential{
									Source: v1.SSHPublicKeyAccessCredentialSource{
										Secret: &v1.AccessCredentialSecretSource{
											SecretName: "my-pkey",
										},
									},
									PropagationMethod: v1.SSHPublicKeyAccessCredentialPropagationMethod{
										ConfigDrive: &v1.ConfigDriveSSHPublicKeyAccessCredentialPropagation{},
									},
								},
							},
						}

						fakeVolumeMountDir("test-volume", map[string]string{
							"userdata":    "secret-userdata",
							"networkdata": "secret-networkdata",
						})

						fakeVolumeMountDir("my-pkey-access-cred", map[string]string{
							"somekey":      "ssh-1234",
							"someotherkey": "ssh-5678",
						})
						keys, err := resolveConfigDriveSecrets(vmi, tmpDir)
						Expect(err).To(Not(HaveOccurred()), "could not resolve secret volume")
						Expect(testVolume.CloudInitConfigDrive.UserData).To(Equal("secret-userdata"))
						Expect(testVolume.CloudInitConfigDrive.NetworkData).To(Equal("secret-networkdata"))
						Expect(len(keys)).To(Equal(2))
					})

					It("should resolve empty config-drive volume and do nothing", func() {
						vmi := createEmptyVMIWithVolumes([]v1.Volume{})
						keys, err := resolveConfigDriveSecrets(vmi, tmpDir)
						Expect(err).To(Not(HaveOccurred()), "failed to resolve empty volumes")
						Expect(len(keys)).To(Equal(0))
					})

					It("should fail if both userdata and network data does not exist", func() {
						testVolume := createCloudInitConfigDriveVolume("test-volume", "test-secret")
						vmi := createEmptyVMIWithVolumes([]v1.Volume{*testVolume})
						keys, err := resolveConfigDriveSecrets(vmi, tmpDir)
						Expect(err).To(HaveOccurred(), "expected a failure when no sources found")
						Expect(err.Error()).To(Equal("no cloud-init data-source found at volume: test-volume"))
						Expect(len(keys)).To(Equal(0))

					})
				})
			})
		})
	})

	Describe("GenerateLocalData", func() {
		It("should cleanly run twice", func() {
			namespace := "fake-namespace"
			domain := "fake-domain"
			userData := "fake\nuser\ndata\n"
			source := &v1.CloudInitNoCloudSource{
				UserDataBase64: base64.StdEncoding.EncodeToString([]byte(userData)),
			}
			cloudInitData, err := readCloudInitNoCloudSource(source)
			Expect(err).NotTo(HaveOccurred())
			err = GenerateLocalData(domain, namespace, cloudInitData)
			Expect(err).NotTo(HaveOccurred())
			err = GenerateLocalData(domain, namespace, cloudInitData)
			Expect(err).NotTo(HaveOccurred())
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
