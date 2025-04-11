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
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strconv"
	"strings"
	"time"

	expect "github.com/google/goexpect"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/crypto/ssh"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/libvmi"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libconfigmap"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[rfe_id:899][crit:medium][vendor:cnv-qe@redhat.com][level:component][sig-compute]Config", decorators.SigCompute, decorators.WgS390x, func() {

	var CheckIsoVolumeSizes = func(vmi *v1.VirtualMachineInstance) {
		pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
		Expect(err).NotTo(HaveOccurred())

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
				out, err := exec.ExecuteCommandOnPod(pod, "compute", cmdCheck)
				Expect(err).NotTo(HaveOccurred())
				size, err := strconv.Atoi(strings.Trim(out, "'"))
				Expect(err).NotTo(HaveOccurred())
				Expect(size % 4096).To(Equal(0))
			}
		}
	}

	Context("With a ConfigMap defined", func() {

		Context("With a single volume", func() {
			var (
				configMapName string
				configMapPath string
			)

			BeforeEach(func() {
				configMapName = "configmap-" + uuid.NewString()
				configMapPath = config.GetConfigMapSourcePath(configMapName)

				data := map[string]string{
					"option1": "value1",
					"option2": "value2",
					"option3": "value3",
				}
				cm := libconfigmap.New(configMapName, data)
				cm, err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(nil)).Delete(context.Background(), configMapName, metav1.DeleteOptions{}); err != nil {
					Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
				}
			})

			It("[test_id:782]Should be the fs layout the same for a pod and vmi", func() {
				expectedOutput := "value1value2value3"

				By("Running VMI")
				vmi := libvmifact.NewAlpine(libvmi.WithConfigMapDisk(configMapName, configMapName))
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				podOutput, err := exec.ExecuteCommandOnPod(
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
					name := "configmap-" + uuid.NewString()
					cm := libconfigmap.New(name, map[string]string{"option": "value"})
					cm, err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
					Expect(err).ToNot(HaveOccurred())
					configMaps = append(configMaps, name)
				}
			})

			AfterEach(func() {
				for _, configMapIface := range configMaps {
					if err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(nil)).Delete(context.Background(), configMapIface, metav1.DeleteOptions{}); err != nil {
						Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
					}
				}
				configMaps = nil
			})

			It("[test_id:783]Should start VMI with multiple ConfigMaps", func() {
				vmi := libvmifact.NewAlpine(
					libvmi.WithConfigMapDisk(configMaps[0], configMaps[0]),
					libvmi.WithConfigMapDisk(configMaps[1], configMaps[1]),
					libvmi.WithConfigMapDisk(configMaps[2], configMaps[2]))

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
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
				secretName = "secret-" + uuid.NewString()
				secretPath = config.GetSecretSourcePath(secretName)

				secret := libsecret.New(secretName, libsecret.DataString{"user": "admin", "password": "redhat"})
				_, err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
				if !errors.IsAlreadyExists(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterEach(func() {
				if err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Delete(context.Background(), secretName, metav1.DeleteOptions{}); err != nil {
					Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
				}
			})

			It("[test_id:779]Should be the fs layout the same for a pod and vmi", func() {
				expectedOutput := "adminredhat"

				By("Running VMI")
				vmi := libvmifact.NewAlpine(libvmi.WithSecretDisk(secretName, secretName))
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if Secret has been attached to the pod")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				podOutput, err := exec.ExecuteCommandOnPod(
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
					name := "secret-" + uuid.NewString()
					secret := libsecret.New(name, libsecret.DataString{"option": "value"})
					_, err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
					if !errors.IsAlreadyExists(err) {
						Expect(err).ToNot(HaveOccurred())
					}

					secrets = append(secrets, name)
				}
			})

			AfterEach(func() {
				for _, secret := range secrets {
					if err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Delete(context.Background(), secret, metav1.DeleteOptions{}); err != nil {
						Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
					}
				}
				secrets = nil
			})

			It("[test_id:780]Should start VMI with multiple Secrets", func() {
				vmi := libvmifact.NewAlpine(
					libvmi.WithSecretDisk(secrets[0], secrets[0]),
					libvmi.WithSecretDisk(secrets[1], secrets[1]),
					libvmi.WithSecretDisk(secrets[2], secrets[2]))

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				CheckIsoVolumeSizes(vmi)
			})
		})

	})

	Context("With a ServiceAccount defined", func() {

		serviceAccountPath := config.ServiceAccountSourceDir

		It("[test_id:998]Should be the namespace and token the same for a pod and vmi", func() {
			By("Running VMI")
			vmi := libvmifact.NewAlpine(libvmi.WithServiceAccountDisk("default"))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			CheckIsoVolumeSizes(vmi)

			By("Checking if ServiceAccount has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			namespace, err := exec.ExecuteCommandOnPod(
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"cat",
					serviceAccountPath + "/namespace",
				},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(namespace).To(Equal(testsuite.GetTestNamespace(vmi)))

			token, err := exec.ExecuteCommandOnPod(
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
				&expect.BExp{R: testsuite.GetTestNamespace(vmi)},
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
				configMapName = "configmap-" + uuid.NewString()
				configMapPath = config.GetConfigMapSourcePath(configMapName)
				secretName = "secret-" + uuid.NewString()
				secretPath = config.GetSecretSourcePath(secretName)

				configData := map[string]string{
					"config1": "value1",
					"config2": "value2",
					"config3": "value3",
				}

				cm := libconfigmap.New(configMapName, configData)
				cm, err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(cm)).Create(context.Background(), cm, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				secret := libsecret.New(secretName, libsecret.DataString{"user": "admin", "password": "redhat"})
				_, err = kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
				if !errors.IsAlreadyExists(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterEach(func() {
				if err := kubevirt.Client().CoreV1().ConfigMaps(testsuite.GetTestNamespace(nil)).Delete(context.Background(), configMapName, metav1.DeleteOptions{}); err != nil {
					Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
				}
				if err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Delete(context.Background(), secretName, metav1.DeleteOptions{}); err != nil {
					Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
				}
			})

			It("[test_id:786]Should be that cfgMap and secret fs layout same for the pod and vmi", func() {
				expectedOutputCfgMap := "value1value2value3"
				expectedOutputSecret := "adminredhat"

				By("Running VMI")
				vmi := libvmifact.NewFedora(libvmi.WithConfigMapDisk(configMapName, configMapName),
					libvmi.WithSecretDisk(secretName, secretName),
					libvmi.WithLabelledConfigMapDisk(configMapName, "random1", "configlabel"),
					libvmi.WithLabelledSecretDisk(secretName, "random2", "secretlabel"))

				// Ensure virtio for consistent order
				for i := range vmi.Spec.Domain.Devices.Disks {
					vmi.Spec.Domain.Devices.Disks[i].DiskDevice = v1.DiskDevice{
						Disk: &v1.DiskTarget{
							Bus: v1.DiskBusVirtio,
						},
					}
				}

				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToFedora(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if ConfigMap has been attached to the pod")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				podOutputCfgMap, err := exec.ExecuteCommandOnPod(
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
				podOutputSecret, err := exec.ExecuteCommandOnPod(
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
			privateKey, _ := generatePrivateKey(bitSize)
			publicKeyBytes, _ := generatePublicKey(&privateKey.PublicKey)
			privateKeyBytes := encodePrivateKeyToPEM(privateKey)

			BeforeEach(func() {
				secretName = "secret-" + uuid.NewString()
				secretPath = config.GetSecretSourcePath(secretName)

				secret := libsecret.New(secretName, libsecret.DataBytes{"ssh-privatekey": privateKeyBytes, "ssh-publickey": publicKeyBytes})
				_, err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Create(context.Background(), secret, metav1.CreateOptions{})
				if !errors.IsAlreadyExists(err) {
					Expect(err).ToNot(HaveOccurred())
				}
			})

			AfterEach(func() {
				if err := kubevirt.Client().CoreV1().Secrets(testsuite.GetTestNamespace(nil)).Delete(context.Background(), secretName, metav1.DeleteOptions{}); err != nil {
					Expect(err).To(MatchError(errors.IsNotFound, "IsNotFound"))
				}
			})

			It("[test_id:778]Should be the fs layout the same for a pod and vmi", func() {
				expectedPrivateKey := string(privateKeyBytes)
				expectedPublicKey := string(publicKeyBytes)

				By("Running VMI")
				vmi := libvmifact.NewAlpine(libvmi.WithSecretDisk(secretName, secretName))
				vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
				Expect(console.LoginToAlpine(vmi)).To(Succeed())

				CheckIsoVolumeSizes(vmi)

				By("Checking if Secret has been attached to the pod")
				vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())

				podOutput1, err := exec.ExecuteCommandOnPod(
					vmiPod,
					vmiPod.Spec.Containers[0].Name,
					[]string{"cat",
						secretPath + "/ssh-privatekey",
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(podOutput1).To(Equal(expectedPrivateKey), "Expected pod output of private key to match genereated one.")

				podOutput2, err := exec.ExecuteCommandOnPod(
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

		downwardAPIName := "downwardapi-" + uuid.NewString()
		downwardAPIPath := config.GetDownwardAPISourcePath(downwardAPIName)

		testLabelKey := "kubevirt.io.testdownwardapi"
		testLabelVal := "downwardAPIValue"
		expectedOutput := testLabelKey + "=" + "\"" + testLabelVal + "\""

		It("[test_id:790]Should be the namespace and token the same for a pod and vmi", func() {
			By("Running VMI")
			vmi := libvmifact.NewAlpine(
				libvmi.WithLabel(testLabelKey, testLabelVal),
				libvmi.WithDownwardAPIDisk(downwardAPIName))
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, 90)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			CheckIsoVolumeSizes(vmi)

			By("Checking if DownwardAPI has been attached to the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			podOutput, err := exec.ExecuteCommandOnPod(
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

// generatePrivateKey creates a RSA Private Key of specified byte size
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(cryptorand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

// generatePublicKey will return in the format "ssh-rsa ..."
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	return publicKeyBytes, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privateBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privateBlock)

	return privatePEM
}
