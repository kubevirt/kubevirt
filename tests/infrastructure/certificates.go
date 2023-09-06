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

package infrastructure

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"kubevirt.io/kubevirt/tests/libinfra"
	"kubevirt.io/kubevirt/tests/libvmi"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	"kubevirt.io/kubevirt/tests/flags"
)

var _ = DescribeInfra("[rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates", func() {
	var (
		virtClient       kubecli.KubevirtClient
		aggregatorClient *aggregatorclient.Clientset
		err              error
	)
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
			_, crts := tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
			return len(crts)
		}, 10*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

		By("destroying the certificate")
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			secret.Data = map[string][]byte{
				"tls.crt": []byte(""),
				"tls.key": []byte(""),
			}

			_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Update(context.Background(), secret, metav1.UpdateOptions{})
			return err
		})
		Expect(err).ToNot(HaveOccurred())

		By("checking that the CA secret gets restored with a new ca bundle")
		var newCA []byte
		Eventually(func() []byte {
			newCA = tests.GetCertFromSecret(components.KubeVirtCASecretName)
			return newCA
		}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))

		By("checking that one of the CAs in the config-map is the new one")
		var caBundle []byte
		Eventually(func() bool {
			caBundle, _ = tests.GetBundleFromConfigMap(components.KubeVirtCASecretName)
			return libinfra.ContainsCrt(caBundle, newCA)
		}, 10*time.Second, 1*time.Second).Should(BeTrue(), "the new CA should be added to the config-map")

		By("checking that the ca bundle gets propagated to the validating webhook")
		Eventually(func() bool {
			webhook, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(webhook.Webhooks) > 0 {
				return libinfra.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
			}
			return false
		}, 10*time.Second, 1*time.Second).Should(BeTrue())
		By("checking that the ca bundle gets propagated to the mutating webhook")
		Eventually(func() bool {
			webhook, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			if len(webhook.Webhooks) > 0 {
				return libinfra.ContainsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
			}
			return false
		}, 10*time.Second, 1*time.Second).Should(BeTrue())

		By("checking that the ca bundle gets propagated to the apiservice")
		Eventually(func() bool {
			apiService, err := aggregatorClient.ApiregistrationV1().APIServices().Get(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return libinfra.ContainsCrt(apiService.Spec.CABundle, newCA)
		}, 10*time.Second, 1*time.Second).Should(BeTrue())

		By("checking that we can still start virtual machines and connect to the VMI")
		vmi := libvmi.NewAlpine()
		vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
		Expect(console.LoginToAlpine(vmi)).To(Succeed())
	})

	It("[sig-compute][test_id:4100] should be valid during the whole rotation process", func() {
		oldAPICert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
		oldHandlerCert := tests.EnsurePodsCertIsSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")
		Expect(err).ToNot(HaveOccurred())

		By("destroying the CA certificate")
		err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Delete(context.Background(), components.KubeVirtCASecretName, metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("repeatedly starting VMIs until virt-api and virt-handler certificates are updated")
		Eventually(func() (rotated bool) {
			vmi := libvmi.NewAlpine()
			vmi = tests.RunVMIAndExpectLaunch(vmi, 60)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())
			err = virtClient.VirtualMachineInstance(vmi.Namespace).Delete(context.Background(), vmi.Name, &metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			newAPICert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-api"), flags.KubeVirtInstallNamespace, "8443")
			Expect(err).ToNot(HaveOccurred())
			newHandlerCert, _, err := tests.GetPodsCertIfSynced(fmt.Sprintf("%s=%s", v1.AppLabel, "virt-handler"), flags.KubeVirtInstallNamespace, "8186")
			Expect(err).ToNot(HaveOccurred())
			return !reflect.DeepEqual(oldHandlerCert, newHandlerCert) && !reflect.DeepEqual(oldAPICert, newAPICert)
		}, 120*time.Second).Should(BeTrue())
	})

	DescribeTable("should be rotated when deleted for ", func(secretName string) {
		By("destroying the certificate")
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), secretName, metav1.GetOptions{})
			if err != nil {
				return err
			}
			secret.Data = map[string][]byte{
				"tls.crt": []byte(""),
				"tls.key": []byte(""),
			}
			_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Update(context.Background(), secret, metav1.UpdateOptions{})

			return err
		})
		Expect(err).ToNot(HaveOccurred())

		By("checking that the secret gets restored with a new certificate")
		Eventually(func() []byte {
			return tests.GetCertFromSecret(secretName)
		}, 10*time.Second, 1*time.Second).Should(Not(BeEmpty()))
	},
		Entry("[test_id:4101] virt-operator", components.VirtOperatorCertSecretName),
		Entry("[test_id:4103] virt-api", components.VirtApiCertSecretName),
		Entry("[test_id:4104] virt-controller", components.VirtControllerCertSecretName),
		Entry("[test_id:4105] virt-handlers client side", components.VirtHandlerCertSecretName),
		Entry("[test_id:4106] virt-handlers server side", components.VirtHandlerServerCertSecretName),
	)
})
