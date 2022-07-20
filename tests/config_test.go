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
	"fmt"
	"strconv"
	"strings"
	"time"

	k8sv1 "k8s.io/api/core/v1"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	"kubevirt.io/kubevirt/tests/util"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/libvmi"
)

var _ = Describe("[rfe_id:899][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Config", func() {

	var virtClient kubecli.KubevirtClient

	var CheckIsoVolumeSizes = func(vmi *v1.VirtualMachineInstance) {
		pod := tests.GetRunningPodByVirtualMachineInstance(vmi, vmi.Namespace)

		for _, volume := range vmi.Spec.Volumes {
			var path = ""
			if volume.ConfigMap != nil {
				path = config.GetConfigMapDiskPath(volume.Name)
				By(fmt.Sprintf("Checking ConfigMap at '%s' is 4k-block fs compatible", path))
			}
			if volume.Secret != nil {
				path = config.GetSecretDiskPath(volume.Name)
				By(fmt.Sprintf("Checking Secret at '%s' is 4k-block fs compatible", path))
			}
			if len(path) > 0 {
				cmdCheck := []string{"stat", "--printf='%s'", path}
				out, err := tests.ExecuteCommandOnPod(virtClient, pod, "compute", cmdCheck)
				Expect(err).NotTo(HaveOccurred())
				size, err := strconv.Atoi(strings.Trim(out, "'"))
				Expect(err).NotTo(HaveOccurred())
				Expect(size % 4096).To(Equal(0))
			}
		}
	}

	BeforeEach(func() {
		var err error

		virtClient, err = kubecli.GetKubevirtClient()
		util.PanicOnError(err)
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
				expectedOutput := "value1value2value3"

				By("Running VMI")
				vmi := libvmi.NewAlpine(withConfigMap(configMapName, configMapName))
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
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
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutput).To(Equal(expectedOutput))

				By("Checking mounted iso image")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount iso ConfigMap image
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/option1 /mnt/option2 /mnt/option3\n"},
					&expect.BExp{R: expectedOutput},
				}, 200)).To(Succeed())
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
				vmi := libvmi.NewAlpine(
					withConfigMap(configMaps[0], configMaps[0]),
					withConfigMap(configMaps[1], configMaps[1]),
					withConfigMap(configMaps[2], configMaps[2]))

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				CheckIsoVolumeSizes(vmi)
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
				expectedOutput := "adminredhat"

				By("Running VMI")
				vmi := libvmi.NewAlpine(withSecret(secretName, secretName))
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if Secret has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
				podOutput, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/user",
						secretPath + "/password",
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutput).To(Equal(expectedOutput))

				By("Checking mounted iso image")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount iso Secret image
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/user /mnt/password\n"},
					&expect.BExp{R: expectedOutput},
				}, 200)).To(Succeed())
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
				vmi := libvmi.NewAlpine(
					withSecret(secrets[0], secrets[0]),
					withSecret(secrets[1], secrets[1]),
					withSecret(secrets[2], secrets[2]))

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				CheckIsoVolumeSizes(vmi)
			})
		})

	})

	Context("With a ServiceAccount defined", func() {

		serviceAccountPath := config.ServiceAccountSourceDir

		It("[test_id:998]Should be the namespace and token the same for a pod and vmi", func() {
			By("Running VMI")
			vmi := libvmi.NewAlpine(withServiceAccount("default"))
			vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			CheckIsoVolumeSizes(vmi)

			By("Checking if ServiceAccount has been attached to the pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			namespace, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"cat",
					serviceAccountPath + "/namespace",
				},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(namespace).To(Equal(util.NamespaceTestDefault))

			token, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"tail", "-c", "20",
					serviceAccountPath + "/token",
				},
			)

			Expect(err).ToNot(HaveOccurred())

			By("Checking mounted iso image")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// mount service account iso image
				&expect.BSnd{S: "mount /dev/sda /mnt\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/namespace\n"},
				&expect.BExp{R: util.NamespaceTestDefault},
				&expect.BSnd{S: "tail -c 20 /mnt/token\n"},
				&expect.BExp{R: token},
			}, 200)).To(Succeed())
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
				vmi := libvmi.NewFedora(withConfigMap(configMapName, configMapName),
					withSecret(secretName, secretName),
					withLabelledConfigMap(configMapName, "random1", "configlabel"),
					withLabelledSecret(secretName, "random2", "secretlabel"))

				// Ensure virtio for consistent order
				for i := range vmi.Spec.Domain.Devices.Disks {
					vmi.Spec.Domain.Devices.Disks[i].DiskDevice = v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					}
				}

				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
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
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutputCfgMap).To(Equal(expectedOutputCfgMap), "Expected %s to Equal value1value2value3", podOutputCfgMap)

				By("Checking mounted ConfigMap image")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount ConfigMap image
					&expect.BSnd{S: "sudo su -\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "mount /dev/vdb /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/config1 /mnt/config2 /mnt/config3\n"},
					&expect.BExp{R: expectedOutputCfgMap},
				}, 200)).To(Succeed())

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
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutputSecret).To(Equal(expectedOutputSecret), "Expected %s to Equal adminredhat", podOutputSecret)

				By("Checking mounted secret image")

				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount Secret image
					&expect.BSnd{S: "mount /dev/vdc /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "cat /mnt/user /mnt/password\n"},
					&expect.BExp{R: expectedOutputSecret},
				}, 200)).To(Succeed())

				By("checking that all disk labels match the expectations")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdb\n"},
					&expect.BExp{R: "cfgdata"}, // default value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdc\n"},
					&expect.BExp{R: "cfgdata"}, // default value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vdd\n"},
					&expect.BExp{R: "configlabel"}, // custom value
					&expect.BSnd{S: "blkid -s LABEL -o value /dev/vde\n"},
					&expect.BExp{R: "secretlabel"}, // custom value
				}, 200)).To(Succeed())
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
				vmi := libvmi.NewAlpine(withSecret(secretName, secretName))
				vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if Secret has been attached to the pod")
				vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
				podOutput1, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/ssh-privatekey",
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutput1).To(Equal(expectedPrivateKey), "Expected pod output of private key to match genereated one.")

				podOutput2, err := tests.ExecuteCommandOnPod(
					virtClient,
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/ssh-publickey",
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutput2).To(Equal(expectedPublicKey), "Expected pod output of public key to match genereated one.")

				By("Checking mounted secrets sshkeys image")
				Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
					// mount iso Secret image
					&expect.BSnd{S: "sudo su -\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "mount /dev/sda /mnt\n"},
					&expect.BExp{R: console.PromptExpression},
					&expect.BSnd{S: "echo $?\n"},
					&expect.BExp{R: console.RetValue("0")},
					&expect.BSnd{S: "grep -c \"PRIVATE KEY\" /mnt/ssh-privatekey\n"},
					&expect.BExp{R: console.RetValue(`[1-9]\d*`)},
					&expect.BSnd{S: "grep -c ssh-rsa /mnt/ssh-publickey\n"},
					&expect.BExp{R: console.RetValue(`[1-9]\d*`)},
				}, 200)).To(Succeed())
			})
		})
	})

	Context("With a DownwardAPI defined", func() {

		downwardAPIName := "downwardapi-" + uuid.NewRandom().String()
		downwardAPIPath := config.GetDownwardAPISourcePath(downwardAPIName)

		testLabelKey := "kubevirt.io.testdownwardapi"
		testLabelVal := "downwardAPIValue"
		expectedOutput := testLabelKey + "=" + "\"" + testLabelVal + "\""

		It("[test_id:790]Should be the namespace and token the same for a pod and vmi", func() {
			By("Running VMI")
			vmi := libvmi.NewAlpine(
				libvmi.WithLabel(testLabelKey, testLabelVal),
				withDownwardAPI(downwardAPIName))
			vmi = tests.RunVMIAndExpectLaunch(vmi, 90)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			CheckIsoVolumeSizes(vmi)

			By("Checking if DownwardAPI has been attached to the pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, util.NamespaceTestDefault)
			podOutput, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"grep", testLabelKey,
					downwardAPIPath + "/labels",
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(podOutput).To(Equal(expectedOutput + "\n"))

			By("Checking mounted iso image")
			Expect(console.ExpectBatch(vmi, []expect.Batcher{
				// mount iso DownwardAPI image
				&expect.BSnd{S: "mount /dev/sda /mnt\n"},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "grep " + testLabelKey + " /mnt/labels\n"},
				&expect.BExp{R: expectedOutput},
			}, 200*time.Second)).To(Succeed())
		})
	})
})

func withSecret(secretName, volumeName string) libvmi.Option {
	return withLabelledSecret(secretName, volumeName, "")
}

func withLabelledSecret(secretName, volumeName, label string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName:  secretName,
					VolumeLabel: label,
				},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: volumeName,
		})
	}
}

func withConfigMap(configMapName, volumeName string) libvmi.Option {
	return withLabelledConfigMap(configMapName, volumeName, "")
}

func withLabelledConfigMap(configMapName, volumeName, label string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: k8sv1.LocalObjectReference{
						Name: configMapName,
					},
					VolumeLabel: label,
				},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: volumeName,
		})
	}
}

func withServiceAccount(name string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: name + "-disk",
			VolumeSource: v1.VolumeSource{
				ServiceAccount: &v1.ServiceAccountVolumeSource{
					ServiceAccountName: name,
				},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: name + "-disk",
		})
	}
}

func withDownwardAPI(name string) libvmi.Option {
	return func(vmi *v1.VirtualMachineInstance) {
		vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
			Name: name,
			VolumeSource: v1.VolumeSource{
				DownwardAPI: &v1.DownwardAPIVolumeSource{
					Fields: []k8sv1.DownwardAPIVolumeFile{
						{
							Path: "labels",
							FieldRef: &k8sv1.ObjectFieldSelector{
								FieldPath: "metadata.labels",
							},
						},
					},
					VolumeLabel: "",
				},
			},
		})

		vmi.Spec.Domain.Devices.Disks = append(vmi.Spec.Domain.Devices.Disks, v1.Disk{
			Name: name,
		})
	}
}
