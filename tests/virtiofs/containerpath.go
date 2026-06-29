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

package virtiofs

import (
	"context"
	"fmt"
	"time"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libmigration"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libpodmutator"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] ContainerPath virtiofs volumes", decorators.SigStorage, decorators.ConfigVolumesVirtiofs, func() {
	Context("With a ContainerPath volume pointing to non-existent path", func() {
		const (
			containerPathFilesystemName = "nonexistent-path"
			nonExistentPath             = "/this/path/does/not/exist"
		)

		It("Should set Synchronized=False with MissingVirtiofsContainers reason", func() {
			virtClient := kubevirt.Client()

			By("Creating a VMI with ContainerPath pointing to non-existent path")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, nonExistentPath),
			)

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for VMI to have Synchronized=False condition with MissingVirtiofsContainers reason")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				for _, condition := range vmi.Status.Conditions {
					if condition.Type == v1.VirtualMachineInstanceSynchronized &&
						condition.Status == k8sv1.ConditionFalse &&
						condition.Reason == v1.MissingVirtiofsContainersReason {
						return true
					}
				}
				return false
			}, 120*time.Second, time.Second).Should(BeTrue(), "VMI should have Synchronized=False with MissingVirtiofsContainers reason")
		})
	})

	Context("With webhook-injected emptyDir volume", func() {
		const (
			webhookName                 = "test-pod-mutator"
			webhookPort                 = 8443
			webhookSecretName           = "webhook-certs"
			containerPathFilesystemName = "injected-fs"
			injectedVolumePath          = "/opt/test-injected"
			testFileName                = "test-file"
			testContent                 = "Hello from webhook-injected volume!"
		)

		var webhook *libpodmutator.Webhook

		BeforeEach(func() {
			webhook = libpodmutator.Setup(libpodmutator.Options{
				Name:       webhookName,
				SecretName: webhookSecretName,
				Port:       webhookPort,
				VolumeInjection: &libpodmutator.VolumeInjection{
					Type: libpodmutator.VolumeTypeEmptyDir,
				},
			})
		})

		AfterEach(func() {
			libpodmutator.Teardown(webhook, webhookSecretName)
		})

		It("Should access webhook-injected emptyDir via ContainerPath virtiofs", func() {
			virtClient := kubevirt.Client()

			By("Creating VMI with ContainerPath pointing to injected volume")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, injectedVolumePath),
			)
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-launcher pod and verifying virtiofsd container exists")
			vmiPod := waitForVirtiofsContainerInPod(vmi, containerPathFilesystemName)

			By("Waiting for VMI to be running")
			// Ignore transient webhook errors - virt-controller will retry and succeed once webhook is ready
			vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running},
				libwait.WithWarningsIgnoreList([]string{"failed calling webhook"}))

			By("Logging into the VMI")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			// Write test file to the injected volume from the pod
			_, err = exec.ExecuteCommandOnPod(
				vmiPod,
				"compute",
				[]string{"sh", "-c", fmt.Sprintf("echo '%s' > %s/%s", testContent, injectedVolumePath, testFileName)},
			)
			Expect(err).ToNot(HaveOccurred())

			By("Mounting the ContainerPath filesystem via virtiofs and reading the test file")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// Mount ContainerPath via virtiofs
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt\n", containerPathFilesystemName)},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				// Read the test file that was written from the pod
				&expect.BSnd{S: fmt.Sprintf("cat /mnt/%s\n", testFileName)},
				&expect.BExp{R: testContent},
			}, 200)).To(Succeed())
		})
	})

	Context("With webhook-injected ConfigMap volume and migration", func() {
		const (
			webhookName                 = "test-pod-mutator-cm"
			webhookPort                 = 8443
			webhookSecretName           = "webhook-certs-cm"
			configMapName               = "test-migration-cm"
			containerPathFilesystemName = "injected-cm-fs"
			injectedVolumePath          = "/opt/test-injected"
			testDataKey                 = "test-data"
			testDataValue               = "Hello from migrated ConfigMap!"
		)

		var webhook *libpodmutator.Webhook

		BeforeEach(func() {
			virtClient := kubevirt.Client()
			testNamespace := testsuite.GetTestNamespace(nil)

			By("Creating ConfigMap with test data")
			configMap := &k8sv1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configMapName,
					Namespace: testNamespace,
				},
				Data: map[string]string{
					testDataKey: testDataValue,
				},
			}
			_, err := virtClient.CoreV1().ConfigMaps(testNamespace).Create(context.Background(), configMap, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			webhook = libpodmutator.Setup(libpodmutator.Options{
				Name:       webhookName,
				SecretName: webhookSecretName,
				Port:       webhookPort,
				VolumeInjection: &libpodmutator.VolumeInjection{
					Type:          libpodmutator.VolumeTypeConfigMap,
					ConfigMapName: configMapName,
				},
			})
		})

		AfterEach(func() {
			virtClient := kubevirt.Client()
			testNamespace := testsuite.GetTestNamespace(nil)

			libpodmutator.Teardown(webhook, webhookSecretName)

			err := virtClient.CoreV1().ConfigMaps(testNamespace).Delete(context.Background(), configMapName, metav1.DeleteOptions{})
			if !errors.IsNotFound(err) {
				Expect(err).ToNot(HaveOccurred())
			}
		})

		It("Should preserve ConfigMap data accessible via ContainerPath after migration", decorators.RequiresTwoSchedulableNodes, func() {
			virtClient := kubevirt.Client()

			By("Creating VMI with ContainerPath pointing to webhook-injected ConfigMap volume")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, injectedVolumePath),
				libnet.WithMasqueradeNetworking(),
			)

			By("Creating the VMI")
			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-launcher pod and verifying virtiofsd container exists")
			_ = waitForVirtiofsContainerInPod(vmi, containerPathFilesystemName)

			By("Waiting for VMI to be running")
			// Ignore transient webhook errors - virt-controller will retry and succeed once webhook is ready
			vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running},
				libwait.WithWarningsIgnoreList([]string{"failed calling webhook"}))

			By("Logging into the VMI")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Mounting ContainerPath filesystem and verifying ConfigMap data is accessible")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// Mount ContainerPath via virtiofs
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt\n", containerPathFilesystemName)},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				// Read ConfigMap data
				&expect.BSnd{S: fmt.Sprintf("cat /mnt/%s\n", testDataKey)},
				&expect.BExp{R: testDataValue},
			}, 200)).To(Succeed())

			By("Starting the migration")
			migration := libmigration.New(vmi.Name, vmi.Namespace)
			migration = libmigration.RunMigrationAndExpectToCompleteWithDefaultTimeout(virtClient, migration)

			By("Verifying migration succeeded")
			Expect(migration.Status.Phase).To(Equal(v1.MigrationSucceeded))

			By("Verifying VMI is still running on the target node")
			Eventually(func() bool {
				vmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vmi.Status.Phase == v1.Running
			}, 30*time.Second, time.Second).Should(BeTrue())

			By("Verifying ConfigMap data is still accessible via ContainerPath after migration")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				// ConfigMap data should still be readable
				&expect.BSnd{S: fmt.Sprintf("cat /mnt/%s\n", testDataKey)},
				&expect.BExp{R: testDataValue},
			}, 200)).To(Succeed())
		})
	})

	Context("With projected ServiceAccountToken volume via native k8s SA projection", func() {
		const (
			containerPathFilesystemName = "sa-projected-fs"
			saTokenPath                 = "/var/run/secrets/kubernetes.io/serviceaccount"
		)

		It("Should access projected ServiceAccountToken volume via ContainerPath virtiofs", func() {
			virtClient := kubevirt.Client()

			By("Creating VMI with ContainerPath pointing to SA token projected volume")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, saTokenPath),
			)
			vmi.Spec.Volumes = append(vmi.Spec.Volumes, v1.Volume{
				Name: "sa-enabler",
				VolumeSource: v1.VolumeSource{
					ServiceAccount: &v1.ServiceAccountVolumeSource{
						ServiceAccountName: "default",
					},
				},
			})

			vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-launcher pod and verifying virtiofsd container exists")
			_ = waitForVirtiofsContainerInPod(vmi, containerPathFilesystemName)

			By("Waiting for VMI to be running")
			vmi = libwait.WaitForVMIPhase(vmi, []v1.VirtualMachineInstancePhase{v1.Running})

			By("Logging into the VMI")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Mounting and reading the projected SA token file via virtiofs")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: fmt.Sprintf("mount -t virtiofs %s /mnt\n", containerPathFilesystemName)},
				&expect.BExp{R: ""},
				&expect.BSnd{S: "echo $?\n"},
				&expect.BExp{R: console.RetValue("0")},
				&expect.BSnd{S: "cat /mnt/token > /dev/null 2>&1 && echo ok\n"},
				&expect.BExp{R: "ok"},
			}, 200)).To(Succeed())
		})
	})
})

// waitForVirtiofsContainerInPod waits for the virt-launcher pod to be running and verifies
// it has the expected virtiofsd container for a ContainerPath volume. Returns the pod for further use.
func waitForVirtiofsContainerInPod(vmi *v1.VirtualMachineInstance, volumeName string) *k8sv1.Pod {
	var vmiPod *k8sv1.Pod
	EventuallyWithOffset(1, func() error {
		var err error
		vmiPod, err = libpod.GetRunningPodByLabel(string(vmi.UID), v1.CreatedByLabel, vmi.Namespace, "")
		return err
	}, 120*time.Second, time.Second).Should(Succeed(), "virt-launcher pod should be running")

	virtiofsContainerName := fmt.Sprintf("virtiofs-%s", volumeName)
	var found bool
	for _, container := range vmiPod.Spec.Containers {
		if container.Name == virtiofsContainerName {
			found = true
			break
		}
	}

	ExpectWithOffset(1, found).To(BeTrue(),
		"virt-launcher pod should have virtiofsd container %s for ContainerPath volume %s", virtiofsContainerName, volumeName)

	return vmiPod
}
