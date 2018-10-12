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
	"flag"
	"time"

	"github.com/google/goexpect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pborman/uuid"

	"kubevirt.io/kubevirt/pkg/config"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/tests"
)

var _ = Describe("Config", func() {

	flag.Parse()

	BeforeEach(func() {
		tests.BeforeTestCleanup()
	})

	Context("With a ConfigMap defined", func() {

		Context("With a single volume", func() {
			var (
				configMapName string
				configMapPath string
			)

			virtClient, err := kubecli.GetKubevirtClient()
			tests.PanicOnError(err)

			BeforeEach(func() {
				configMapName = "configmap-" + uuid.NewRandom().String()
				configMapPath = config.GetConfigMapSourcePath(configMapName + "-vol")

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

			It("Should be the fs layout the same for a pod and vmi", func() {
				expectedOutput := "value1value2value3"

				By("Running VMI")
				vmi := tests.NewRandomVMIWithConfigMap(configMapName)
				tests.RunVMIAndExpectLaunch(vmi, false, 90)

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
					&expect.BExp{R: "0"},
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

			It("Should start VMI with multiple ConfigMaps", func() {
				vmi := tests.NewRandomVMIWithConfigMap(configMaps[0])
				tests.AddConfigMapDisk(vmi, configMaps[1])
				tests.AddConfigMapDisk(vmi, configMaps[2])

				tests.RunVMIAndExpectLaunch(vmi, false, 90)
			})
		})
	})

	Context("With a Secret defined", func() {

		Context("With a single volume", func() {
			var (
				secretName string
				secretPath string
			)

			virtClient, err := kubecli.GetKubevirtClient()
			tests.PanicOnError(err)

			BeforeEach(func() {
				secretName = "secret-" + uuid.NewRandom().String()
				secretPath = config.GetSecretSourcePath(secretName + "-vol")

				data := map[string]string{
					"user":     "admin",
					"password": "redhat",
				}
				tests.CreateSecret(secretName, data)
			})

			AfterEach(func() {
				tests.DeleteSecret(secretName)
			})

			It("Should be the fs layout the same for a pod and vmi", func() {
				expectedOutput := "adminredhat"

				By("Running VMI")
				vmi := tests.NewRandomVMIWithSecret(secretName)
				tests.RunVMIAndExpectLaunch(vmi, false, 90)

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
					&expect.BExp{R: "0"},
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

			It("Should start VMI with multiple Secrets", func() {
				vmi := tests.NewRandomVMIWithSecret(secrets[0])
				tests.AddSecretDisk(vmi, secrets[1])
				tests.AddSecretDisk(vmi, secrets[2])

				tests.RunVMIAndExpectLaunch(vmi, false, 90)
			})
		})

	})

	Context("With a ServiceAccount defined", func() {

		virtClient, err := kubecli.GetKubevirtClient()
		tests.PanicOnError(err)

		serviceAccountPath := config.ServiceAccountSourceDir

		It("Should be the fs layout the same for a pod and vmi", func() {
			By("Running VMI")
			vmi := tests.NewRandomVMIWithServiceAccount("default")
			tests.RunVMIAndExpectLaunch(vmi, false, 90)

			By("Checking if ServiceAccount has been attached to the pod")
			vmiPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			podOutput, err := tests.ExecuteCommandOnPod(
				virtClient,
				vmiPod,
				vmiPod.Spec.Containers[0].Name,
				[]string{"cat",
					serviceAccountPath + "/namespace",
				},
			)
			Expect(err).To(BeNil())
			Expect(podOutput).To(Equal(tests.NamespaceTestDefault))

			By("Checking mounted iso image")
			expecter, err := tests.LoggedInAlpineExpecter(vmi)
			Expect(err).ToNot(HaveOccurred())
			defer expecter.Close()

			_, err = expecter.ExpectBatch([]expect.Batcher{
				// mount service account iso image
				&expect.BSnd{S: "mount /dev/sda /mnt\n"},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: "0"},
				&expect.BSnd{S: "cat /mnt/namespace\n"},
				&expect.BExp{R: tests.NamespaceTestDefault},
			}, 200*time.Second)
			Expect(err).ToNot(HaveOccurred())
		})

	})
})
