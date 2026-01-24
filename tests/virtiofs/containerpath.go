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

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libsecret"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-storage] ContainerPath virtiofs volumes", decorators.SigStorage, decorators.VirtioFS, func() {
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

		var webhookPod *k8sv1.Pod
		var webhookService *k8sv1.Service
		var webhookConfig *admissionregistrationv1.MutatingWebhookConfiguration

		BeforeEach(func() {
			webhookArgs := []string{
				fmt.Sprintf("--port=%d", webhookPort),
				"--volume-type=emptydir",
			}
			webhookPod, webhookService, webhookConfig = setupWebhook(webhookName, webhookSecretName, webhookPort, webhookArgs)
		})

		AfterEach(func() {
			teardownWebhook(webhookPod, webhookService, webhookConfig, webhookSecretName)
		})

		It("Should access webhook-injected emptyDir via ContainerPath virtiofs", func() {
			By("Creating VMI with ContainerPath pointing to injected volume")
			vmi := libvmifact.NewAlpine(
				libvmi.WithFilesystemContainerPath(containerPathFilesystemName, injectedVolumePath),
			)

			By("Starting the VMI")
			vmi = libvmops.RunVMIAndExpectLaunchIgnoreWarnings(vmi, 300)

			By("Logging into the VMI")
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Verifying the injected volume exists in the pod")
			vmiPod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).ToNot(HaveOccurred())

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
})

// setupWebhook creates and deploys a test webhook with TLS certificates
func setupWebhook(webhookName, webhookSecretName string, webhookPort int32, webhookArgs []string) (*k8sv1.Pod, *k8sv1.Service, *admissionregistrationv1.MutatingWebhookConfiguration) {
	virtClient := kubevirt.Client()
	testNamespace := testsuite.GetTestNamespace(nil)

	By("Generating TLS certificates for webhook")
	certPEM, keyPEM, caBundlePEM, err := libinfra.GenerateWebhookCertificates(webhookName, testNamespace, 24*time.Hour)
	Expect(err).ToNot(HaveOccurred())

	By("Creating secret with webhook certificates")
	secret := libsecret.New(webhookSecretName, libsecret.DataBytes{
		"tls.crt": certPEM,
		"tls.key": keyPEM,
	})
	_, err = virtClient.CoreV1().Secrets(testNamespace).Create(context.Background(), secret, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Deploying test-pod-mutator webhook")
	webhookPod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: testNamespace,
			Labels: map[string]string{
				"app": webhookName,
			},
		},
		Spec: k8sv1.PodSpec{
			SecurityContext: &k8sv1.PodSecurityContext{
				RunAsNonRoot: pointer.P(true),
				SeccompProfile: &k8sv1.SeccompProfile{
					Type: k8sv1.SeccompProfileTypeRuntimeDefault,
				},
			},
			Containers: []k8sv1.Container{{
				Name:    webhookName,
				Image:   fmt.Sprintf("%s/test-helpers:%s", flags.KubeVirtRepoPrefix, flags.KubeVirtVersionTag),
				Command: []string{"/usr/bin/test-pod-mutator"},
				Args:    webhookArgs,
				Ports: []k8sv1.ContainerPort{{
					ContainerPort: webhookPort,
					Name:          "https",
				}},
				VolumeMounts: []k8sv1.VolumeMount{{
					Name:      "certs",
					MountPath: "/etc/webhook/certs",
					ReadOnly:  true,
				}},
				SecurityContext: &k8sv1.SecurityContext{
					AllowPrivilegeEscalation: pointer.P(false),
					Capabilities: &k8sv1.Capabilities{
						Drop: []k8sv1.Capability{"ALL"},
					},
					RunAsNonRoot: pointer.P(true),
					SeccompProfile: &k8sv1.SeccompProfile{
						Type: k8sv1.SeccompProfileTypeRuntimeDefault,
					},
				},
			}},
			Volumes: []k8sv1.Volume{{
				Name: "certs",
				VolumeSource: k8sv1.VolumeSource{
					Secret: &k8sv1.SecretVolumeSource{
						SecretName: webhookSecretName,
					},
				},
			}},
		},
	}
	webhookPod, err = virtClient.CoreV1().Pods(testNamespace).Create(context.Background(), webhookPod, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Waiting for webhook pod to be ready")
	Eventually(func() bool {
		pod, err := virtClient.CoreV1().Pods(testNamespace).Get(context.Background(), webhookName, metav1.GetOptions{})
		if err != nil {
			return false
		}
		return pod.Status.Phase == k8sv1.PodRunning
	}, 60*time.Second, time.Second).Should(BeTrue(), "Webhook pod should be running")

	By("Creating service for webhook")
	webhookService := &k8sv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: testNamespace,
		},
		Spec: k8sv1.ServiceSpec{
			Selector: map[string]string{
				"app": webhookName,
			},
			Ports: []k8sv1.ServicePort{{
				Port:       webhookPort,
				TargetPort: intstr.FromInt(int(webhookPort)),
				Name:       "https",
			}},
		},
	}
	webhookService, err = virtClient.CoreV1().Services(testNamespace).Create(context.Background(), webhookService, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	By("Creating MutatingWebhookConfiguration with CA bundle")
	failPolicy := admissionregistrationv1.Fail
	sideEffects := admissionregistrationv1.SideEffectClassNone
	webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", webhookName, testNamespace),
		},
		Webhooks: []admissionregistrationv1.MutatingWebhook{{
			Name: fmt.Sprintf("%s.kubevirt.io", webhookName),
			ClientConfig: admissionregistrationv1.WebhookClientConfig{
				CABundle: caBundlePEM,
				Service: &admissionregistrationv1.ServiceReference{
					Namespace: testNamespace,
					Name:      webhookName,
					Path:      pointer.P("/mutate"),
					Port:      pointer.P(webhookPort),
				},
			},
			Rules: []admissionregistrationv1.RuleWithOperations{{
				Operations: []admissionregistrationv1.OperationType{
					admissionregistrationv1.Create,
				},
				Rule: admissionregistrationv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
				},
			}},
			FailurePolicy: &failPolicy,
			SideEffects:   &sideEffects,
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": testNamespace,
				},
			},
			AdmissionReviewVersions: []string{"v1"},
		}},
	}
	webhookConfig, err = virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.Background(), webhookConfig, metav1.CreateOptions{})
	Expect(err).ToNot(HaveOccurred())

	return webhookPod, webhookService, webhookConfig
}

// teardownWebhook cleans up webhook resources
func teardownWebhook(webhookPod *k8sv1.Pod, webhookService *k8sv1.Service, webhookConfig *admissionregistrationv1.MutatingWebhookConfiguration, webhookSecretName string) {
	virtClient := kubevirt.Client()
	testNamespace := testsuite.GetTestNamespace(nil)

	if webhookConfig != nil {
		err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Delete(context.Background(), webhookConfig.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	if webhookService != nil {
		err := virtClient.CoreV1().Services(testNamespace).Delete(context.Background(), webhookService.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	if webhookPod != nil {
		err := virtClient.CoreV1().Pods(testNamespace).Delete(context.Background(), webhookPod.Name, metav1.DeleteOptions{})
		if !errors.IsNotFound(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
	err := virtClient.CoreV1().Secrets(testNamespace).Delete(context.Background(), webhookSecretName, metav1.DeleteOptions{})
	if !errors.IsNotFound(err) {
		Expect(err).ToNot(HaveOccurred())
	}
}
