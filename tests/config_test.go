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
 * Copyright 2018 Red Hat, Inc.
 *
 */
package tests_test

import (
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	v1 "kubevirt.io/client-go/api/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/tests"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
)

var _ = Describe("[Serial][rfe_id:899][crit:medium][vendor:cnv-qe@redhat.com][level:component]Config", func() {

	var virtClient kubecli.KubevirtClient

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		tests.BeforeTestCleanup()
	})

	Context("With a ConfigMap defined", func() {

		Context("With a single volume", func() {
			var (
				configMapName string
				configMapPath string
			)

			BeforeEach(func() {
				configMapName = "configmap-" + uuid.NewRandom().String()
				configMapPath = config.GetConfigMapSourcePath(configMapName)

				data := map[string]string{
					"option1": "value1",
					"option2": "value2",
					"option3": "value3",
				}
				tests.CreateConfigMap(configMapName, data)
			})

			AfterEach(func() {
				tests.DeleteConfigMap(configMapName)
			})

			It("[test_id:782]Should be the fs layout the same for a pod and vmi", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()

				expectedOutput := "value1value2value3"

				By("Running VMI")
				vmi := tests.NewRandomVMIWithConfigMap(configMapName)
				tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podOutput, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						configMapPath + "/option1",
						configMapPath + "/option2",
						configMapPath + "/option3",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutput).To(Equal(expectedOutput))

				By("Checking mounted iso image")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				_, err = expecter.ExpectBatch([]expect.Batcher{
					// mount iso ConfigMap image
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/option1 /mnt/option2 /mnt/option3\n"},
					&expect.BExp{R: expectedOutput},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With multiple volumes", func() {
			var (
				configMaps    []string
				configMapsCnt int = 3
			)

			BeforeEach(func() {
				for i := 0; i < configMapsCnt; i++ {
					name := "configmap-" + uuid.NewRandom().String()
					tests.CreateConfigMap(name, map[string]string{"option": "value"})
					configMaps = append(configMaps, name)
				}
			})

			AfterEach(func() {
				for _, configMap := range configMaps {
					tests.DeleteConfigMap(configMap)
				}
				configMaps = nil
			})

			It("[test_id:783]Should start VMI with multiple ConfigMaps", func() {
				vmi := tests.NewRandomVMIWithConfigMap(configMaps[0])
				tests.AddConfigMapDisk(vmi, configMaps[1], configMaps[1])
				tests.AddConfigMapDisk(vmi, configMaps[2], configMaps[2])

				tests.RunVMIAndExpectLaunch(vmi, 90)
			})
		})
	})

	Context("With a Secret defined", func() {

		Context("With a single volume", func() {
			var (
				secretName string
				secretPath string
			)

			BeforeEach(func() {
				secretName = "secret-" + uuid.NewRandom().String()
				secretPath = config.GetSecretSourcePath(secretName)

				data := map[string]string{
					"user":     "admin",
					"password": "redhat",
				}
				tests.CreateSecret(secretName, data)
			})

			AfterEach(func() {
				tests.DeleteSecret(secretName)
			})

			It("[test_id:779]Should be the fs layout the same for a pod and vmi", func() {
				tests.SkipPVCTestIfRunnigOnKindInfra()

				expectedOutput := "adminredhat"

				By("Running VMI")
				vmi := tests.NewRandomVMIWithSecret(secretName)
				tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking if Secret has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podOutput, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/user",
						secretPath + "/password",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutput).To(Equal(expectedOutput))

				By("Checking mounted iso image")
				expecter, err := tests.LoggedInAlpineExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				_, err = expecter.ExpectBatch([]expect.Batcher{
					// mount iso Secret image
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/user /mnt/password\n"},
					&expect.BExp{R: expectedOutput},
				}, 200*time.Second)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("With multiple volumes", func() {
			var (
				secrets    []string
				secretsCnt int = 3
			)

			BeforeEach(func() {
				for i := 0; i < secretsCnt; i++ {
					name := "secret-" + uuid.NewRandom().String()
					tests.CreateSecret(name, map[string]string{"option": "value"})
					secrets = append(secrets, name)
				}
			})

			AfterEach(func() {
				for _, secret := range secrets {
					tests.DeleteSecret(secret)
				}
				secrets = nil
			})

			It("[test_id:780]Should start VMI with multiple Secrets", func() {
				vmi := tests.NewRandomVMIWithSecret(secrets[0])
				tests.AddSecretDisk(vmi, secrets[1], secrets[1])
				tests.AddSecretDisk(vmi, secrets[2], secrets[2])

				tests.RunVMIAndExpectLaunch(vmi, 90)
			})
		})

	})

	Context("With a ServiceAccount defined", func() {

		serviceAccountPath := config.ServiceAccountSourceDir

		It("[test_id:998]Should be the namespace and token the same for a pod and vmi", func() {
			tests.SkipPVCTestIfRunnigOnKindInfra()

			By("Running VMI")
			vmi := tests.NewRandomVMIWithServiceAccount("default")
			tests.RunVMIAndExpectLaunch(vmi, 90)

			By("Checking if ServiceAccount has been attached to the pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			namespace, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"cat",
					serviceAccountPath + "/namespace",
				},
			)

			Expect(err).To(BeNil())
			Expect(namespace).To(Equal(tests.NamespaceTestDefault))

			token, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"tail", "-c", "20",
					serviceAccountPath + "/token",
				},
			)

			By("Checking mounted iso image")
			expecter, err := tests.LoggedInAlpineExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			_, err = expecter.ExpectBatch([]expect.Batcher{
				// mount service account iso image
				&expect.BSnd{S: "mount /dev/sda /mnt\n"},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: tests.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/namespace\n"},
				&expect.BExp{R: tests.NamespaceTestDefault},
				&expect.BSnd{S: "tail -c 20 /mnt/token\n"},
				&expect.BExp{R: token},
			}, 200*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("With a Secret and a ConfigMap defined", func() {

		Context("With a single volume", func() {
			var (
				configMapName string
				configMapPath string
				secretName    string
				secretPath    string
			)

			BeforeEach(func() {
				configMapName = "configmap-" + uuid.NewRandom().String()
				configMapPath = config.GetConfigMapSourcePath(configMapName)
				secretName = "secret-" + uuid.NewRandom().String()
				secretPath = config.GetSecretSourcePath(secretName)

				configData := map[string]string{
					"config1": "value1",
					"config2": "value2",
					"config3": "value3",
				}

				secretData := map[string]string{
					"user":     "admin",
					"password": "redhat",
				}

				tests.CreateConfigMap(configMapName, configData)

				tests.CreateSecret(secretName, secretData)
			})

			AfterEach(func() {
				tests.DeleteConfigMap(configMapName)
				tests.DeleteSecret(secretName)
			})

			It("[test_id:786]Should be that cfgMap and secret fs layout same for the pod and vmi", func() {
				expectedOutputCfgMap := "value1value2value3"
				expectedOutputSecret := "adminredhat"

				By("Running VMI")

				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(
					cd.ContainerDiskFor(
						cd.ContainerDiskFedora), "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\n")
				tests.AddConfigMapDisk(vmi, configMapName, configMapName)
				tests.AddSecretDisk(vmi, secretName, secretName)
				tests.AddConfigMapDiskWithCustomLabel(vmi, configMapName, "random1", "configlabel")
				tests.AddSecretDiskWithCustomLabel(vmi, secretName, "random2", "secretlabel")

				// Ensure virtio for consistent order
				for i, _ := range vmi.Spec.Domain.Devices.Disks {
					vmi.Spec.Domain.Devices.Disks[i].Disk = &v1.DiskTarget{Bus: "virtio"}
				}

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podOutputCfgMap, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						configMapPath + "/config1",
						configMapPath + "/config2",
						configMapPath + "/config3",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutputCfgMap).To(Equal(expectedOutputCfgMap), "Expected %s to Equal value1value2value3", podOutputCfgMap)

				By("Checking mounted ConfigMap image")
				expecter, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				res, err := expecter.ExpectBatch([]expect.Batcher{
					// mount ConfigMap image
					&expect.BSnd{S: "sudo su -\n"},
					&expect.BExp{R: "#"},
					&expect.BSnd{S: "mount /dev/vdc /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/config1 /mnt/config2 /mnt/config3\n"},
					&expect.BExp{R: expectedOutputCfgMap},
				}, 200*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

				By("Checking if Secret has also been attached to the same pod")
				podOutputSecret, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/user",
						secretPath + "/password",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutputSecret).To(Equal(expectedOutputSecret), "Expected %s to Equal adminredhat", podOutputSecret)

				By("Checking mounted secret image")

				res, err = expecter.ExpectBatch([]expect.Batcher{
					// mount Secret image
					&expect.BSnd{S: "mount /dev/vdd /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/user /mnt/password\n"},
					&expect.BExp{R: expectedOutputSecret},
				}, 200*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())

				By("checking that all disk labels match the expectations")
				res, err = expecter.ExpectBatch([]expect.Batcher{
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdc\n"},
					&expect.BExp{R: "cfgdata"}, // default value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdd\n"},
					&expect.BExp{R: "cfgdata"}, // default value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vde\n"},
					&expect.BExp{R: "configlabel"}, // custom value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdf\n"},
					&expect.BExp{R: "secretlabel"}, // custom value
				}, 200*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Context("With SSH Keys as a Secret defined", func() {

		Context("With a single volume", func() {
			var (
				secretName string
				secretPath string
			)

			var bitSize int = 2048
			privateKey, _ := tests.GeneratePrivateKey(bitSize)
			publicKeyBytes, _ := tests.GeneratePublicKey(&privateKey.PublicKey)
			privateKeyBytes := tests.EncodePrivateKeyToPEM(privateKey)

			BeforeEach(func() {
				secretName = "secret-" + uuid.NewRandom().String()
				secretPath = config.GetSecretSourcePath(secretName)

				data := map[string]string{
					"ssh-privatekey": string(privateKeyBytes),
					"ssh-publickey":  string(publicKeyBytes),
				}
				tests.CreateSecret(secretName, data)
			})

			AfterEach(func() {
				tests.DeleteSecret(secretName)
			})

			It("[test_id:778]Should be the fs layout the same for a pod and vmi", func() {
				expectedPrivateKey := string(privateKeyBytes)
				expectedPublicKey := string(publicKeyBytes)

				By("Running VMI")
				vmi := tests.NewRandomVMIWithEphemeralDiskAndUserdataHighMemory(
					cd.ContainerDiskFor(
						cd.ContainerDiskFedora), "#!/bin/bash\necho \"fedora\" | passwd fedora --stdin\n")
				tests.AddSecretDisk(vmi, secretName, secretName)
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)

				By("Checking if Secret has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
				podOutput1, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/ssh-privatekey",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutput1).To(Equal(expectedPrivateKey), "Expected pod output of private key to match genereated one.")

				podOutput2, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/ssh-publickey",
					},
				)
				Expect(err).To(BeNil())
				Expect(podOutput2).To(Equal(expectedPublicKey), "Expected pod output of public key to match genereated one.")

				By("Checking mounted secrets sshkeys image")
				expecter, err := tests.LoggedInFedoraExpecter(vmi)
				Expect(err).ToNot(HaveOccurred())
				defer expecter.Close()

				res, err := expecter.ExpectBatch([]expect.Batcher{
					// mount iso Secret image
					&expect.BSnd{S: "sudo su -\n"},
					&expect.BExp{R: "\\#"},
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "grep \"PRIVATE KEY\" /mnt/ssh-privatekey\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
					&expect.BSnd{S: "grep ssh-rsa /mnt/ssh-publickey\n"},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: tests.RetValue("0")},
				}, 200*time.Second)
				log.DefaultLogger().Object(vmi).Infof("%v", res)
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})
