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

package infrastructure

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/certificates/bootstrap"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/framework/k8s"

	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libvmops"
)

var _ = Describe(SIGSerial("[rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates", func() {
	var (
		virtClient       kubecli.KubevirtClient
		aggregatorClient *aggregatorclient.Clientset
	)
	const vmiLaunchTimeOut = libvmops.StartupTimeoutSecondsSmall
	BeforeEach(func() {
		virtClient = kubevirt.Client()

		if aggregatorClient == nil {
			config, err := kubecli.GetKubevirtClientConfig()
			if err != nil {
				panic(err)
			}

			aggregatorClient = aggregatorclient.NewForConfigOrDie(config)
		}
	})

	It("[test_id:4099] should be rotated when a new CA is created", func() {
		By("checking that the config-map gets the new CA bundle attached")
		Eventually(func() int {
			_, crts := libinfra.GetBundleFromConfigMap(context.Background(), components.KubeVirtCASecretName)
			return len(crts)
		}, 10*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

		By("destroying the certificate")
		secretPatch, err := patch.New(
			patch.WithReplace("/data/tls.crt", ""),
			patch.WithReplace("/data/tls.key", ""),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = k8s.Client().CoreV1().Secrets(flags.KubeVirtInstallNamespace).Patch(
			context.Background(), components.KubeVirtCASecretName,
			types.JSONPatchType, secretPatch, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking that the CA secret gets restored with a new ca bundle")
		var newCA []byte
		Eventually(func() []byte {
			newCA = getCertFromSecret(components.KubeVirtCASecretName)
			return newCA
		}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))

		By("checking that one of the CAs in the config-map is the new one")
		var caBundle []byte
		Eventually(func() bool {
			caBundle, _ = libinfra.GetBundleFromConfigMap(context.Background(), components.KubeVirtCASecretName)
			return libinfra.ContainsCrt(caBundle, newCA)
		}, 10*time.Second, 1*time.Second).Should(BeTrue(), "the new CA should be added to the config-map")

		By("checking that the ca bundle gets propagated to the validating webhook")
		Eventually(func() bool {
			webhook, err := k8s.Client().AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
				context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(webhook.Webhooks) > 0 {
				return libinfra.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
			}
			return false
		}, 10*time.Second, 1*time.Second).Should(BeTrue())
		By("checking that the ca bundle gets propagated to the mutating webhook")
		Eventually(func() bool {
			webhook, err := k8s.Client().AdmissionregistrationV1().MutatingWebhookConfigurations().Get(
				context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(webhook.Webhooks) > 0 {
				return libinfra.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
			}
			return false
		}, 10*time.Second, 1*time.Second).Should(BeTrue())

		By("checking that the ca bundle gets propagated to the apiservice")
		Eventually(func() bool {
			apiService, err := aggregatorClient.ApiregistrationV1().APIServices().Get(
				context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return libinfra.ContainsCrt(apiService.Spec.CABundle, newCA)
		}, 10*time.Second, 1*time.Second).Should(BeTrue())

		By("checking that we can still start virtual machines and connect to the VMI")
		vmi := libvmifact.NewAlpine()
		vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiLaunchTimeOut)
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
	})

	It("[sig-compute][test_id:4100] should be valid during the whole rotation process", func() {
		oldAPICert := libinfra.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
		oldHandlerCert := libinfra.EnsurePodsCertIsSynced(
			fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")

		By("destroying the CA certificate")
		err := k8s.Client().CoreV1().Secrets(flags.KubeVirtInstallNamespace).Delete(
			context.Background(), components.KubeVirtCASecretName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("repeatedly starting VMIs until virt-api and virt-handler certificates are updated")
		Eventually(func() (rotated bool) {
			vmi := libvmifact.NewAlpine()
			vmi = libvmops.RunVMIAndExpectLaunch(vmi, vmiLaunchTimeOut)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			apiCerts, err := libpod.GetCertsForPods(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
			Expect(err).ToNot(HaveOccurred())
			if !hasIdenticalCerts(apiCerts) {
				return false
			}
			newAPICert := apiCerts[0]

			handlerCerts, err := libpod.GetCertsForPods(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")
			Expect(err).ToNot(HaveOccurred())
			if !hasIdenticalCerts(handlerCerts) {
				return false
			}
			newHandlerCert := handlerCerts[0]

			return !reflect.DeepEqual(oldHandlerCert, newHandlerCert) && !reflect.DeepEqual(oldAPICert, newAPICert)
		}, 120*time.Second).Should(BeTrue())
	})

	DescribeTable("should be rotated when deleted for ", func(secretName string) {
		By("destroying the certificate")
		secretPatch, err := patch.New(
			patch.WithReplace("/data/tls.crt", ""),
			patch.WithReplace("/data/tls.key", ""),
		).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())
		_, err = k8s.Client().CoreV1().Secrets(flags.KubeVirtInstallNamespace).Patch(
			context.Background(), secretName, types.JSONPatchType, secretPatch, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking that the secret gets restored with a new certificate")
		Eventually(func() []byte {
			return getCertFromSecret(secretName)
		}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))
	},
		Entry("[test_id:4101] virt-operator", components.VirtOperatorCertSecretName),
		Entry("[test_id:4103] virt-api", components.VirtApiCertSecretName),
		Entry("[test_id:4104] virt-controller", components.VirtControllerCertSecretName),
		Entry("[test_id:4105] virt-handlers client side", components.VirtHandlerCertSecretName),
		Entry("[test_id:4106] virt-handlers server side", components.VirtHandlerServerCertSecretName),
	)
}))

func getCertFromSecret(secretName string) []byte {
	secret, err := k8s.Client().CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
	Expect(err).ToNot(HaveOccurred())
	if rawBundle, ok := secret.Data[bootstrap.CertBytesValue]; ok {
		return rawBundle
	}
	return nil
}

func hasIdenticalCerts(certs [][]byte) bool {
	if len(certs) == 0 {
		return false
	}
	for _, crt := range certs {
		if !reflect.DeepEqual(certs[0], crt) {
			return false
		}
	}

	return true
}
