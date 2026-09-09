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

package operator

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

// This Describe holds operator specs that only read cluster state and assert on
// it (or attempt changes that are rejected by validation). None of them mutate
// the KubeVirt CR or any operator-managed resource, so they do not require the
// destructive teardown used by the main (Serial) operator suite and can run in
// parallel. Keeping them out of the Serial suite means they are not excluded
// from lanes that skip Serial specs.
var _ = Describe("[sig-operator]Operator readonly", decorators.SigOperator, func() {
	var virtClient kubecli.KubevirtClient
	var originalKv *v1.KubeVirt
	var err error

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv = libkubevirt.GetCurrentKv(virtClient)
	})

	It("[test_id:1746]should have created and available condition", func() {
		kv := libkubevirt.GetCurrentKv(virtClient)

		By("verifying that created and available condition is present")
		testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)
	})

	Context("[rfe_id:2897][crit:medium][vendor:cnv-qe@redhat.com][level:component]With OpenShift cluster", decorators.OpenShift, func() {

		It("[test_id:2910]Should have kubevirt SCCs created", func() {
			const OpenShiftSCCLabel = "openshift.io/scc"
			var expectedSCCs, sccs []string

			By("Checking if kubevirt SCCs have been created")
			secClient := virtClient.SecClient()
			operatorSCCs := components.GetAllSCC(flags.KubeVirtInstallNamespace)
			for _, scc := range operatorSCCs {
				expectedSCCs = append(expectedSCCs, scc.GetName())
			}

			createdSCCs, err := secClient.SecurityContextConstraints().List(context.Background(), metav1.ListOptions{LabelSelector: controller.OperatorLabel})
			Expect(err).NotTo(HaveOccurred())
			for _, scc := range createdSCCs.Items {
				sccs = append(sccs, scc.GetName())
			}
			Expect(sccs).To(ConsistOf(expectedSCCs))

			By("Checking if virt-handler is assigned to kubevirt-handler SCC")
			l, err := labels.Parse("kubevirt.io=virt-handler")
			Expect(err).ToNot(HaveOccurred())

			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: l.String()})
			Expect(err).ToNot(HaveOccurred(), "Should get virt-handler")
			Expect(pods.Items).ToNot(BeEmpty())
			Expect(pods.Items[0].Annotations[OpenShiftSCCLabel]).To(
				Equal("kubevirt-handler"), "Should virt-handler be assigned to kubevirt-handler SCC",
			)

			By("Checking if virt-launcher is assigned to kubevirt-controller SCC")
			vmi := libvmifact.NewGuestless()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			uid := vmi.GetObjectMeta().GetUID()
			labelSelector := v1.CreatedByLabel + "=" + string(uid)
			pods, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
			Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher")
			Expect(pods.Items).To(HaveLen(1))
			Expect(pods.Items[0].Annotations[OpenShiftSCCLabel]).To(
				Equal("kubevirt-controller"), "Should virt-launcher be assigned to kubevirt-controller SCC",
			)
		})
	})

	Context("With PrometheusRule Enabled", func() {

		BeforeEach(func() {
			if !prometheusRuleEnabled() {
				Skip("Test applies on when PrometheusRule is defined") //nolint:forbidigo
			}
		})

		It("[test_id:4614]Checks if the kubevirt PrometheusRule cr exists and verify it's spec", func() {
			monv1 := virtClient.PrometheusClient().MonitoringV1()
			prometheusRule, err := monv1.PrometheusRules(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KUBEVIRT_PROMETHEUS_RULE_NAME, metav1.GetOptions{})
			Expect(err).ShouldNot(HaveOccurred())
			Expect(prometheusRule.Spec).ToNot(BeNil())
			Expect(prometheusRule.Spec.Groups).ToNot(BeEmpty())
			Expect(prometheusRule.Spec.Groups[0].Rules).ToNot(BeEmpty())

		})
	})

	Context("[rfe_id:2937][crit:medium][vendor:cnv-qe@redhat.com][level:component]With ServiceMonitor Enabled", func() {

		BeforeEach(func() {
			if !serviceMonitorEnabled() {
				Skip("Test requires ServiceMonitor to be valid") //nolint:forbidigo
			}
		})

		It("[test_id:2936]Should allow Prometheus to scrape KubeVirt endpoints", func() {
			coreClient := virtClient.CoreV1()

			// we don't know when the prometheus toolchain will pick up our config, so we retry plenty of times
			// before to give up. TODO: there is a smarter way to wait?
			Eventually(func() string {
				By("Obtaining Prometheus' configuration data")
				var secret *k8sv1.Secret
				for _, monitoringNamespace := range util.DefaultMonitorNamespaces {
					secret, err = coreClient.Secrets(monitoringNamespace).Get(context.Background(), "prometheus-k8s", metav1.GetOptions{})
					if err == nil {
						break
					}
				}
				Expect(err).ToNot(HaveOccurred())

				data, ok := secret.Data["prometheus.yaml"]
				// In new versions of prometheus-operator, the configuration file is compressed in the secret
				if !ok {
					data, ok = secret.Data["prometheus.yaml.gz"]
					Expect(ok).To(BeTrue())

					By("Decompressing Prometheus' configuration data")
					gzreader, err := gzip.NewReader(bytes.NewReader(data))
					Expect(err).ToNot(HaveOccurred())

					decompressed, err := io.ReadAll(gzreader)
					Expect(err).ToNot(HaveOccurred())

					data = decompressed
				}

				Expect(data).ToNot(BeNil())

				By("Verifying that Prometheus is watching KubeVirt")
				return string(data)
			}, 90*time.Second, 3*time.Second).Should(ContainSubstring(flags.KubeVirtInstallNamespace), "Prometheus should be monitoring KubeVirt")
		})

		It("[test_id:4616]Should patch our namespace labels with openshift.io/cluster-monitoring=true", func() {
			By("Inspecting the labels on our namespace")
			namespace, err := virtClient.CoreV1().Namespaces().Get(context.Background(), flags.KubeVirtInstallNamespace, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			monitoringLabel, exists := namespace.ObjectMeta.Labels["openshift.io/cluster-monitoring"]
			Expect(exists).To(BeTrue())
			Expect(monitoringLabel).To(Equal("true"))
		})
	})

	Context("[rfe_id:4356]Node Placement", func() {
		It("should reject infra placement configuration with incorrect toleration operator", func() {
			const incorrectOperator = "foo"
			incorrectInfra := &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					Tolerations: []k8sv1.Toleration{{
						Key:      "someKey",
						Operator: k8sv1.TolerationOperator(incorrectOperator),
						Value:    "someValue",
					}},
				},
			}
			const errMsg = "spec.infra.nodePlacement.tolerations.operator in body should be one of"
			Expect(patchKVInfra(originalKv, incorrectInfra)).To(MatchError(ContainSubstring(errMsg)))
		})

		It("should reject workload placement configuration with incorrect toleraion operator", func() {
			const incorrectOperator = "foo"
			incorrectWorkload := &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					Tolerations: []k8sv1.Toleration{{
						Key:      "someKey",
						Operator: k8sv1.TolerationOperator(incorrectOperator),
						Value:    "someValue",
					}},
				},
			}
			const errMsg = "spec.workloads.nodePlacement.tolerations.operator in body should be one of"
			Expect(patchKVWorkloads(originalKv, incorrectWorkload)).To(MatchError(ContainSubstring(errMsg)))
		})

		It("[test_id:8235]should check if kubevirt components have linux node selector", func() {
			By("Listing only kubevirt components")

			kv := libkubevirt.GetCurrentKv(virtClient)
			productComponent := kv.Spec.ProductComponent
			if productComponent == "" {
				productComponent = "kubevirt"
			}

			labelReq, err := labels.NewRequirement("app.kubernetes.io/component", selection.In, []string{productComponent})
			Expect(err).ToNot(HaveOccurred())

			By("Looking for pods with " + productComponent + " component")

			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{
				LabelSelector: labels.NewSelector().Add(
					*labelReq,
				).String(),
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(pods.Items).NotTo(BeEmpty())

			By("Checking nodeselector")
			for _, pod := range pods.Items {
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue(k8sv1.LabelOSStable, "linux"), fmt.Sprintf("pod %s does not have linux node selector", pod.Name))
			}
		})
	})

	Context("Replicas", func() {
		It("should fail to set replicas to 0", func() {
			var replicas uint8 = 0
			infra := &v1.ComponentConfig{
				Replicas: &replicas,
			}

			Expect(patchKVInfra(originalKv, infra)).To(MatchError(ContainSubstring("infra replica count can't be 0")))
		})
	})

	Context("Certificate Rotation", func() {
		var certConfig *v1.KubeVirtSelfSignConfiguration
		BeforeEach(func() {
			certConfig = &v1.KubeVirtSelfSignConfiguration{
				CA: &v1.CertConfig{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 16 * time.Hour},
				},
				Server: &v1.CertConfig{
					Duration:    &metav1.Duration{Duration: 12 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 10 * time.Hour},
				},
			}
		})

		It("[test_id:6258]should reject combining deprecated and new cert rotation parameters", func() {
			kv := copyOriginalKv(originalKv)
			certConfig.CAOverlapInterval = &metav1.Duration{Duration: 8 * time.Hour}
			Expect(patchKvCertConfig(kv.Name, certConfig)).ToNot(Succeed())
		})

		It("[test_id:6259]should reject CA expires before rotation", func() {
			kv := copyOriginalKv(originalKv)
			certConfig.CA.Duration = &metav1.Duration{Duration: 14 * time.Hour}
			Expect(patchKvCertConfig(kv.Name, certConfig)).ToNot(Succeed())
		})

		It("[test_id:6260]should reject Cert expires before rotation", func() {
			kv := copyOriginalKv(originalKv)
			certConfig.Server.Duration = &metav1.Duration{Duration: 8 * time.Hour}
			Expect(patchKvCertConfig(kv.Name, certConfig)).ToNot(Succeed())
		})

		It("[test_id:6261]should reject Cert rotates after CA expires", func() {
			kv := copyOriginalKv(originalKv)
			certConfig.Server.Duration = &metav1.Duration{Duration: 48 * time.Hour}
			certConfig.Server.RenewBefore = &metav1.Duration{Duration: 36 * time.Hour}
			Expect(patchKvCertConfig(kv.Name, certConfig)).ToNot(Succeed())
		})
	})

	Context("external CA", func() {
		It("should create a blank configmap", func() {
			Eventually(func(g Gomega) {
				cm, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.ExternalKubeVirtCAConfigMapName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(cm.Data).To(HaveKeyWithValue(components.CABundleKey, ""))
			}, 1*time.Minute, 5*time.Second).Should(Succeed())
		})
	})
})
