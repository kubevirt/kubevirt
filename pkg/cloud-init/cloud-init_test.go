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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/precond"
)

var _ = Describe("CloudInit", func() {

	var (
		ctrl            *gomock.Controller
		virtClient      *kubecli.MockKubevirtClient
		isoCreationFunc IsoCreationFunc
	)

	tmpDir, _ := ioutil.TempDir("", "cloudinittest")

	owner, err := user.Current()
	if err != nil {
		panic(err)
	}

	BeforeSuite(func() {
		err := SetLocalDirectory(tmpDir)
		if err != nil {
			panic(err)
		}
		SetLocalDataOwner(owner.Username)
	})

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
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

	AfterSuite(func() {
		os.RemoveAll(tmpDir)
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
					err = removeLocalData(domain, namespace)
					Expect(err).ToNot(HaveOccurred())
				})
			})

			Context("with multiple data dirs and files", func() {
				It("should list all VirtualMachineInstance's", func() {
					domains := []string{
						"fakens1/fakedomain1",
						"fakens1/fakedomain2",
						"fakens2/fakedomain1",
						"fakens2/fakedomain2",
						"fakens3/fakedomain1",
						"fakens4/fakedomain1",
					}
					msg := "fake content"
					bytes := []byte(msg)

					for _, dom := range domains {
						err := os.MkdirAll(fmt.Sprintf("%s/%s/some-other-dir", tmpDir, dom), 0755)
						Expect(err).ToNot(HaveOccurred())
						err = ioutil.WriteFile(fmt.Sprintf("%s/%s/some-file", tmpDir, dom), bytes, 0644)
						Expect(err).ToNot(HaveOccurred())
					}

					vmis, err := listVmWithLocalData()
					for _, vmi := range vmis {
						namespace := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetNamespace())
						domain := precond.MustNotBeEmpty(vmi.GetObjectMeta().GetName())

						Expect(namespace).To(ContainSubstring("fakens"))
						Expect(domain).To(ContainSubstring("fakedomain"))
					}

					Expect(len(vmis)).To(Equal(len(domains)))
					Expect(err).ToNot(HaveOccurred())

					// verify a vmi from each namespace is present
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

				It("should fail to verify networkData without userData", func() {
					networkData := "FakeNetwork"
					source := &v1.CloudInitNoCloudSource{
						NetworkData: networkData,
					}
					_, err := readCloudInitNoCloudSource(source)
					Expect(err).Should(MatchError("userDataBase64 or userData is required for a cloud-init data source"))
				})

				Context("with secretRefs", func() {
					userDataSecretName := "userDataSecretName"
					networkDataSecretName := "networkDataSecretName"
					namespace := "testing"

					createSecret := func(name, dataKey, dataValue string) *k8sv1.Secret {
						return &k8sv1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Type: "Opaque",
							Data: map[string][]byte{
								dataKey: []byte(dataValue),
							},
						}
					}
					createUserDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(userDataSecretName, "userdata", data)
					}
					createBadUserDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(userDataSecretName, "baduserdara", data)
					}
					createNetworkDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(networkDataSecretName, "networkdata", data)
					}
					createBadNetworkDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(networkDataSecretName, "badnetworkdata", data)
					}

					It("should succeed to verify userDataSecretRef", func() {
						userSecret := createUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()).AnyTimes()

						cloudInitData := &v1.CloudInitNoCloudSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
						Expect(cloudInitData.UserData).To(Equal("secretUserData"))
					})

					It("should succeed to verify userDataSecretRef and networkDataSecretRef", func() {
						userSecret := createUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						networkSecret := createNetworkDataSecret("secretNetworkData")
						networkClient := fake.NewSimpleClientset(networkSecret)

						gomock.InOrder(
							virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()),
							virtClient.EXPECT().CoreV1().Return(networkClient.CoreV1()),
						)

						cloudInitData := &v1.CloudInitNoCloudSource{
							UserDataSecretRef:    &k8sv1.LocalObjectReference{Name: userDataSecretName},
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
						Expect(cloudInitData.UserData).To(Equal("secretUserData"))
						Expect(cloudInitData.NetworkData).To(Equal("secretNetworkData"))
					})

					It("should succeed to verify nothing", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())
						cloudInitData := &v1.CloudInitNoCloudSource{}
						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
					})

					It("should fail to verify UserDataSecretRef without a secret", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())

						cloudInitData := &v1.CloudInitNoCloudSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("secrets \"%s\" not found", userDataSecretName)))
					})

					It("should fail to verify NetworkDataSecretRef without a secret", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())

						cloudInitData := &v1.CloudInitNoCloudSource{
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("secrets \"%s\" not found", networkDataSecretName)))
					})

					It("should fail to verify UserDataSecretRef with a misnamed secret", func() {
						userSecret := createBadUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()).AnyTimes()

						cloudInitData := &v1.CloudInitNoCloudSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("userdata key not found in k8s secret %s <nil>", userDataSecretName)))
					})

					It("should fail to verify NetworkDataSecretRef with a misnamed secret", func() {
						networkSecret := createBadNetworkDataSecret("secretNetworkData")
						networkClient := fake.NewSimpleClientset(networkSecret)
						virtClient.EXPECT().CoreV1().Return(networkClient.CoreV1())

						cloudInitData := &v1.CloudInitNoCloudSource{
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveNoCloudSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("networkdata key not found in k8s secret %s <nil>", networkDataSecretName)))
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

				It("should fail to verify networkData without userData", func() {
					networkData := "FakeNetwork"
					source := &v1.CloudInitConfigDriveSource{
						NetworkData: networkData,
					}
					_, err := readCloudInitConfigDriveSource(source)
					Expect(err).Should(MatchError("userDataBase64 or userData is required for a cloud-init data source"))
				})

				Context("with secretRefs", func() {
					userDataSecretName := "userDataSecretName"
					networkDataSecretName := "networkDataSecretName"
					namespace := "testing"

					createSecret := func(name, dataKey, dataValue string) *k8sv1.Secret {
						return &k8sv1.Secret{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Type: "Opaque",
							Data: map[string][]byte{
								dataKey: []byte(dataValue),
							},
						}
					}
					createUserDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(userDataSecretName, "userdata", data)
					}
					createBadUserDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(userDataSecretName, "baduserdara", data)
					}
					createNetworkDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(networkDataSecretName, "networkdata", data)
					}
					createBadNetworkDataSecret := func(data string) *k8sv1.Secret {
						return createSecret(networkDataSecretName, "badnetworkdata", data)
					}

					It("should succeed to verify userDataSecretRef", func() {
						userSecret := createUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()).AnyTimes()

						cloudInitData := &v1.CloudInitConfigDriveSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
						Expect(cloudInitData.UserData).To(Equal("secretUserData"))
					})

					It("should succeed to verify userDataSecretRef and networkDataSecretRef", func() {
						userSecret := createUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						networkSecret := createNetworkDataSecret("secretNetworkData")
						networkClient := fake.NewSimpleClientset(networkSecret)

						gomock.InOrder(
							virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()),
							virtClient.EXPECT().CoreV1().Return(networkClient.CoreV1()),
						)

						cloudInitData := &v1.CloudInitConfigDriveSource{
							UserDataSecretRef:    &k8sv1.LocalObjectReference{Name: userDataSecretName},
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
						Expect(cloudInitData.UserData).To(Equal("secretUserData"))
						Expect(cloudInitData.NetworkData).To(Equal("secretNetworkData"))
					})

					It("should succeed to verify nothing", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())
						cloudInitData := &v1.CloudInitConfigDriveSource{}
						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err).To(BeNil())
					})

					It("should fail to verify UserDataSecretRef without a secret", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())

						cloudInitData := &v1.CloudInitConfigDriveSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("secrets \"%s\" not found", userDataSecretName)))
					})

					It("should fail to verify NetworkDataSecretRef without a secret", func() {
						fakeClient := fake.NewSimpleClientset()
						virtClient.EXPECT().CoreV1().Return(fakeClient.CoreV1())

						cloudInitData := &v1.CloudInitConfigDriveSource{
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("secrets \"%s\" not found", networkDataSecretName)))
					})

					It("should fail to verify UserDataSecretRef with a misnamed secret", func() {
						userSecret := createBadUserDataSecret("secretUserData")
						userClient := fake.NewSimpleClientset(userSecret)
						virtClient.EXPECT().CoreV1().Return(userClient.CoreV1()).AnyTimes()

						cloudInitData := &v1.CloudInitConfigDriveSource{
							UserDataSecretRef: &k8sv1.LocalObjectReference{Name: userDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("userdata key not found in k8s secret %s <nil>", userDataSecretName)))
					})

					It("should fail to verify NetworkDataSecretRef with a misnamed secret", func() {
						networkSecret := createBadNetworkDataSecret("secretNetworkData")
						networkClient := fake.NewSimpleClientset(networkSecret)
						virtClient.EXPECT().CoreV1().Return(networkClient.CoreV1())

						cloudInitData := &v1.CloudInitConfigDriveSource{
							NetworkDataSecretRef: &k8sv1.LocalObjectReference{Name: networkDataSecretName},
						}

						err := resolveConfigDriveSecrets(cloudInitData, namespace, virtClient)
						Expect(err.Error()).To(Equal(fmt.Sprintf("networkdata key not found in k8s secret %s <nil>", networkDataSecretName)))
					})
				})
			})
		})
	})
})
