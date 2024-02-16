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

package tests_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"

	kvtls "kubevirt.io/kubevirt/pkg/util/tls"

	"kubevirt.io/kubevirt/tests/exec"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/testsuite"

	expect "github.com/google/goexpect"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
	netutils "k8s.io/utils/net"

	"kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libreplicaset"

	"kubevirt.io/kubevirt/tests/util"

	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/downwardmetrics/vhostmd/api"

	"kubevirt.io/kubevirt/tests/libvmi"
	"kubevirt.io/kubevirt/tests/libwait"

	k8sv1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/util/retry"

	v1ext "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	clusterutil "kubevirt.io/kubevirt/pkg/util/cluster"
	"kubevirt.io/kubevirt/pkg/virt-controller/leaderelectionconfig"
	nodelabellerutil "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	crds "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests"
	"kubevirt.io/kubevirt/tests/console"
	cd "kubevirt.io/kubevirt/tests/containerdisk"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/libnet"
)

const (
	remoteCmdErrPattern = "failed running `%s` with stdout:\n %v \n stderr:\n %v \n err: \n %v \n"
)

var _ = Describe("[Serial][sig-compute]Infrastructure", Serial, decorators.SigCompute, func() {
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

	Describe("changes to the kubernetes client", func() {
		scheduledToRunning := func(vmis []v1.VirtualMachineInstance) time.Duration {
			var duration time.Duration
			for _, vmi := range vmis {
				start := metav1.Time{}
				stop := metav1.Time{}
				for _, timestamp := range vmi.Status.PhaseTransitionTimestamps {
					if timestamp.Phase == v1.Scheduled {
						start = timestamp.PhaseTransitionTimestamp
					} else if timestamp.Phase == v1.Running {
						stop = timestamp.PhaseTransitionTimestamp
					}
				}
				duration += stop.Sub(start.Time)
			}
			return duration
		}

		It("on the controller rate limiter should lead to delayed VMI starts", func() {
			By("first getting the basetime for a replicaset")
			replicaset := tests.NewRandomReplicaSetFromVMI(libvmi.NewCirros(libvmi.WithResourceMemory("1Mi")), int32(0))
			replicaset, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Create(replicaset)
			Expect(err).ToNot(HaveOccurred())
			start := time.Now()
			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 10)
			fastDuration := time.Now().Sub(start)
			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 0)

			By("reducing the throughput on controller")
			originalKubeVirt := util.GetCurrentKv(virtClient)
			originalKubeVirt.Spec.Configuration.ControllerConfiguration = &v1.ReloadableComponentConfiguration{
				RestClient: &v1.RESTClientConfiguration{
					RateLimiter: &v1.RateLimiter{
						TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
							Burst: 3,
							QPS:   2,
						},
					},
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
			By("starting a replicaset with reduced throughput")
			start = time.Now()
			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 10)
			slowDuration := time.Now().Sub(start)
			Expect(slowDuration.Seconds()).To(BeNumerically(">", 2*fastDuration.Seconds()))
		})

		It("on the virt handler rate limiter should lead to delayed VMI running states", func() {
			By("first getting the basetime for a replicaset")
			targetNode := libnode.GetAllSchedulableNodes(virtClient).Items[0]
			vmi := libvmi.New(
				libvmi.WithResourceMemory("1Mi"),
				libvmi.WithNodeSelectorFor(&targetNode),
			)

			replicaset := tests.NewRandomReplicaSetFromVMI(vmi, 0)
			replicaset, err = virtClient.ReplicaSet(testsuite.GetTestNamespace(nil)).Create(replicaset)
			Expect(err).ToNot(HaveOccurred())
			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 10)
			Eventually(matcher.AllVMIs(replicaset.Namespace), 90*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))
			vmis, err := matcher.AllVMIs(replicaset.Namespace)()
			Expect(err).ToNot(HaveOccurred())
			fastDuration := scheduledToRunning(vmis)

			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 0)
			Eventually(matcher.AllVMIs(replicaset.Namespace), 90*time.Second, 1*time.Second).Should(matcher.BeGone())

			By("reducing the throughput on handler")
			originalKubeVirt := util.GetCurrentKv(virtClient)
			originalKubeVirt.Spec.Configuration.HandlerConfiguration = &v1.ReloadableComponentConfiguration{
				RestClient: &v1.RESTClientConfiguration{
					RateLimiter: &v1.RateLimiter{
						TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
							Burst: 1,
							QPS:   1,
						},
					},
				},
			}
			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)

			By("starting a replicaset with reduced throughput")
			libreplicaset.DoScaleWithScaleSubresource(virtClient, replicaset.Name, 10)
			Eventually(matcher.AllVMIs(replicaset.Namespace), 180*time.Second, 1*time.Second).Should(matcher.BeInPhase(v1.Running))
			vmis, err = matcher.AllVMIs(replicaset.Namespace)()
			Expect(err).ToNot(HaveOccurred())
			slowDuration := scheduledToRunning(vmis)
			Expect(slowDuration.Seconds()).To(BeNumerically(">", 1.5*fastDuration.Seconds()))
		})
	})

	Describe("downwardMetrics", func() {
		It("[test_id:6535]should be published to a vmi and periodically updated", func() {
			vmi := libvmi.NewFedora()
			tests.AddDownwardMetricsVolume(vmi, "vhostmd")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			metrics, err := getDownwardMetrics(vmi)
			Expect(err).ToNot(HaveOccurred())
			timestamp := getTimeFromMetrics(metrics)

			vmi, err = virtClient.VirtualMachineInstance(vmi.Namespace).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() int {
				metrics, err = getDownwardMetrics(vmi)
				Expect(err).ToNot(HaveOccurred())
				return getTimeFromMetrics(metrics)
			}, 10*time.Second, 1*time.Second).ShouldNot(Equal(timestamp))
			Expect(getHostnameFromMetrics(metrics)).To(Equal(vmi.Status.NodeName))
		})

		It("metric ResourceProcessorLimit should be present", func() {
			vmi := libvmi.NewFedora(libvmi.WithCPUCount(1, 1, 1))
			tests.AddDownwardMetricsVolume(vmi, "vhostmd")
			vmi = tests.RunVMIAndExpectLaunch(vmi, 180)
			Expect(console.LoginToFedora(vmi)).To(Succeed())

			metrics, err := getDownwardMetrics(vmi)
			Expect(err).ToNot(HaveOccurred())

			//let's try to find the ResourceProcessorLimit metric
			found := false
			j := 0
			for i, metric := range metrics.Metrics {
				if metric.Name == "ResourceProcessorLimit" {
					j = i
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
			Expect(metrics.Metrics[j].Value).To(Equal("1"))
		})
	})

	Describe("CRDs", func() {
		It("[test_id:5177]Should have structural schema", func() {
			ourCRDs := []string{crds.VIRTUALMACHINE, crds.VIRTUALMACHINEINSTANCE, crds.VIRTUALMACHINEINSTANCEPRESET,
				crds.VIRTUALMACHINEINSTANCEREPLICASET, crds.VIRTUALMACHINEINSTANCEMIGRATION, crds.KUBEVIRT,
				crds.VIRTUALMACHINESNAPSHOT, crds.VIRTUALMACHINESNAPSHOTCONTENT,
			}

			for _, name := range ourCRDs {
				ext, err := extclient.NewForConfig(virtClient.Config())
				Expect(err).ToNot(HaveOccurred())

				crd, err := ext.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				Expect(crd).To(matcher.HaveConditionMissingOrFalse(v1ext.NonStructuralSchema))
			}
		})
	})

	Describe("[rfe_id:4102][crit:medium][vendor:cnv-qe@redhat.com][level:component]certificates", func() {

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
				return containsCrt(caBundle, newCA)
			}, 10*time.Second, 1*time.Second).Should(BeTrue(), "the new CA should be added to the config-map")

			By("checking that the ca bundle gets propagated to the validating webhook")
			Eventually(func() bool {
				webhook, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return containsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())
			By("checking that the ca bundle gets propagated to the mutating webhook")
			Eventually(func() bool {
				webhook, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if len(webhook.Webhooks) > 0 {
					return containsCrt(webhook.Webhooks[0].ClientConfig.CABundle, newCA)
				}
				return false
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that the ca bundle gets propagated to the apiservice")
			Eventually(func() bool {
				apiService, err := aggregatorClient.ApiregistrationV1().APIServices().Get(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return containsCrt(apiService.Spec.CABundle, newCA)
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("checking that we can still start virtual machines and connect to the VMI")
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
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
				vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
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

	Describe("tls configuration", func() {

		var cipher *tls.CipherSuite

		BeforeEach(func() {
			if !checks.HasFeature(virtconfig.VMExportGate) {
				Skip(fmt.Sprintf("Cluster has the %s featuregate disabled, skipping  the tests", virtconfig.VMExportGate))
			}

			// FIPS-compliant so we can test on different platforms (otherwise won't revert properly)
			cipher = &tls.CipherSuite{
				ID:   tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				Name: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
			}
			kvConfig := util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
			kvConfig.TLSConfiguration = &v1.TLSConfiguration{
				MinTLSVersion: v1.VersionTLS12,
				Ciphers:       []string{cipher.Name},
			}
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
			newKv := util.GetCurrentKv(virtClient)
			Expect(newKv.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(BeEquivalentTo(v1.VersionTLS12))
			Expect(newKv.Spec.Configuration.TLSConfiguration.Ciphers).To(BeEquivalentTo([]string{cipher.Name}))

		})

		It("[test_id:9306]should result only connections with the correct client-side tls configurations are accepted by the components", func() {
			labelSelectorList := []string{"kubevirt.io=virt-api", "kubevirt.io=virt-handler", "kubevirt.io=virt-exportproxy"}

			var podsToTest []k8sv1.Pod
			for _, labelSelector := range labelSelectorList {
				podList, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{
					LabelSelector: labelSelector,
				})
				Expect(err).ToNot(HaveOccurred())
				podsToTest = append(podsToTest, podList.Items...)
			}

			for i, pod := range podsToTest {
				func(i int, pod k8sv1.Pod) {
					stopChan := make(chan struct{})
					defer close(stopChan)
					Expect(tests.ForwardPorts(&pod, []string{fmt.Sprintf("844%d:%d", i, 8443)}, stopChan, 10*time.Second)).To(Succeed())

					acceptedTLSConfig := &tls.Config{
						InsecureSkipVerify: true,
						MaxVersion:         tls.VersionTLS12,
						CipherSuites:       kvtls.CipherSuiteIds([]string{cipher.Name}),
					}
					conn, err := tls.Dial("tcp", fmt.Sprintf("localhost:844%d", i), acceptedTLSConfig)
					Expect(err).ToNot(HaveOccurred())
					Expect(conn).ToNot(BeNil())
					Expect(conn.ConnectionState().Version).To(BeEquivalentTo(tls.VersionTLS12))
					Expect(conn.ConnectionState().CipherSuite).To(BeEquivalentTo(cipher.ID))

					rejectedTLSConfig := &tls.Config{
						InsecureSkipVerify: true,
						MaxVersion:         tls.VersionTLS11,
					}
					conn, err = tls.Dial("tcp", fmt.Sprintf("localhost:844%d", i), rejectedTLSConfig)
					Expect(err).To(HaveOccurred())
					Expect(conn).To(BeNil())
					Expect(err.Error()).To(SatisfyAny(
						BeEquivalentTo("remote error: tls: protocol version not supported"),
						// The error message changed with the golang 1.19 update
						BeEquivalentTo("tls: no supported versions satisfy MinVersion and MaxVersion"),
					))
				}(i, pod)
			}
		})
	})

	// start a VMI, wait for it to run and return the node it runs on
	startVMI := func(vmi *v1.VirtualMachineInstance) string {
		By("Starting a new VirtualMachineInstance")
		obj, err := virtClient.
			RestClient().
			Post().
			Resource("virtualmachineinstances").
			Namespace(testsuite.GetTestNamespace(vmi)).
			Body(vmi).
			Do(context.Background()).Get()
		Expect(err).ToNot(HaveOccurred(), "Should create VMI")
		vmiObj, ok := obj.(*v1.VirtualMachineInstance)
		Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")

		By("Waiting until the VM is ready")
		return libwait.WaitForSuccessfulVMIStart(vmiObj).Status.NodeName
	}

	Describe("[rfe_id:4126][crit:medium][vendor:cnv-qe@redhat.com][level:component]Taints and toleration", func() {

		Context("CriticalAddonsOnly taint set on a node", func() {
			var (
				possiblyTaintedNodeName string
				kubevirtPodsOnNode      []string
				deploymentsOnNode       []types.NamespacedName
			)

			BeforeEach(func() {
				possiblyTaintedNodeName = ""
				kubevirtPodsOnNode = nil
				deploymentsOnNode = nil

				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{})
				Expect(err).ShouldNot(HaveOccurred(), "failed listing kubevirt pods")
				Expect(pods.Items).ToNot(BeEmpty(), "no kubevirt pods found")

				nodeName := getNodeWithOneOfPods(virtClient, pods.Items)

				// It is possible to run this test on a cluster that simply does not have worker nodes.
				// Since KubeVirt can't control that, the only correct action is to halt the test.
				if nodeName == "" {
					Skip("Could not determine a node to safely taint")
				}

				podsOnNode, err := virtClient.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{
					FieldSelector: fields.OneTermEqualSelector("spec.nodeName", nodeName).String(),
				})
				Expect(err).NotTo(HaveOccurred())

				kubevirtPodsOnNode = filterKubevirtPods(podsOnNode.Items)
				deploymentsOnNode = getDeploymentsForPods(virtClient, podsOnNode.Items)

				By("tainting the selected node")
				selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				possiblyTaintedNodeName = nodeName

				taints := append(selectedNode.Spec.Taints, k8sv1.Taint{
					Key:    "CriticalAddonsOnly",
					Value:  "",
					Effect: k8sv1.TaintEffectNoExecute,
				})

				patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, taints)
				Expect(err).ToNot(HaveOccurred())
				selectedNode, err = virtClient.CoreV1().Nodes().Patch(context.Background(), selectedNode.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				if possiblyTaintedNodeName == "" {
					return
				}

				By("removing the taint from the tainted node")
				selectedNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), possiblyTaintedNodeName, metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())

				var hasTaint bool
				var otherTaints []k8sv1.Taint
				for _, taint := range selectedNode.Spec.Taints {
					if taint.Key == "CriticalAddonsOnly" {
						hasTaint = true
					} else {
						otherTaints = append(otherTaints, taint)
					}
				}

				if !hasTaint {
					return
				}

				patchData, err := patch.GenerateTestReplacePatch("/spec/taints", selectedNode.Spec.Taints, otherTaints)
				Expect(err).NotTo(HaveOccurred())
				selectedNode, err = virtClient.CoreV1().Nodes().Patch(context.Background(), selectedNode.Name, types.JSONPatchType, patchData, metav1.PatchOptions{})
				Expect(err).NotTo(HaveOccurred())

				// Waiting until all affected deployments have at least 1 ready replica
				checkedDeployments := map[types.NamespacedName]struct{}{}
				Eventually(func(g Gomega) {
					for _, namespacedName := range deploymentsOnNode {
						if _, ok := checkedDeployments[namespacedName]; ok {
							continue
						}

						deployment, err := virtClient.AppsV1().Deployments(namespacedName.Namespace).
							Get(context.Background(), namespacedName.Name, metav1.GetOptions{})
						if k8serrors.IsNotFound(err) {
							checkedDeployments[namespacedName] = struct{}{}
							continue
						}
						g.Expect(err).NotTo(HaveOccurred())

						if deployment.DeletionTimestamp != nil || *deployment.Spec.Replicas == 0 {
							checkedDeployments[namespacedName] = struct{}{}
							continue
						}
						g.Expect(deployment.Status.ReadyReplicas).To(
							BeNumerically(">=", 1),
							fmt.Sprintf("Deployment %s is not ready", namespacedName.String()),
						)
						checkedDeployments[namespacedName] = struct{}{}
					}
				}, time.Minute, time.Second).Should(Succeed())
			})

			It("[test_id:4134] kubevirt components on that node should not evict", func() {
				Consistently(func(g Gomega) {
					for _, podName := range kubevirtPodsOnNode {
						pod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), podName, metav1.GetOptions{})
						g.Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("error getting pod %s/%s", flags.KubeVirtInstallNamespace, podName))
						g.Expect(pod.DeletionTimestamp).To(BeNil(), fmt.Sprintf("pod %s/%s is being deleted", flags.KubeVirtInstallNamespace, podName))
						g.Expect(pod.Spec.NodeName).To(Equal(possiblyTaintedNodeName), fmt.Sprintf("pod %s/%s does not run on tainted node", flags.KubeVirtInstallNamespace, podName))
					}
				}, time.Second*10, time.Second).Should(Succeed())
			})

		})
	})

	Describe("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus scraped metrics", func() {

		/*
			This test is querying the metrics from Prometheus *after* they were
			scraped and processed by the different components on the way.
		*/

		BeforeEach(func() {
			onOCP, err := clusterutil.IsOnOpenShift(virtClient)
			Expect(err).ToNot(HaveOccurred(), "failed to detect cluster type")

			if !onOCP {
				Skip("test is verifying integration with OCP's cluster monitoring stack")
			}
		})

		It("[test_id:4135]should find VMI namespace on namespace label of the metric", func() {

			/*
				This test is required because in cases of misconfigurations on
				monitoring objects (such for the ServiceMonitor), our rules will
				still be picked up by the monitoring-operator, but Prometheus
				will fail to load it.
			*/

			By("creating a VMI in a user defined namespace")
			vmi := tests.NewRandomVMIWithEphemeralDisk(
				cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			startVMI(vmi)

			By("finding virt-operator pod")
			ops, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: "kubevirt.io=virt-operator"})
			Expect(err).ToNot(HaveOccurred(), "failed to list virt-operators")
			Expect(ops.Size).ToNot(Equal(0), "no virt-operators found")
			op := ops.Items[0]
			Expect(op).ToNot(BeNil(), "virt-operator pod should not be nil")

			var ep *k8sv1.Endpoints
			By("finding Prometheus endpoint")
			Eventually(func() bool {
				ep, err = virtClient.CoreV1().Endpoints("openshift-monitoring").Get(context.Background(), "prometheus-k8s", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred(), "failed to retrieve Prometheus endpoint")

				if len(ep.Subsets) == 0 || len(ep.Subsets[0].Addresses) == 0 {
					return false
				}
				return true
			}, 10*time.Second, time.Second).Should(BeTrue())

			promIP := ep.Subsets[0].Addresses[0].IP
			Expect(promIP).ToNot(Equal(0), "could not get Prometheus IP from endpoint")
			var promPort int32
			for _, port := range ep.Subsets[0].Ports {
				if port.Name == "web" {
					promPort = port.Port
				}
			}
			Expect(promPort).ToNot(Equal(0), "could not get Prometheus port from endpoint")

			// We need a token from a service account that can view all namespaces in the cluster
			By("extracting virt-operator sa token")
			cmd := []string{"cat", "/var/run/secrets/kubernetes.io/serviceaccount/token"}
			token, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, &op, "virt-operator", cmd)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, strings.Join(cmd, " "), token, stderr, err))
			Expect(token).ToNot(BeEmpty(), "virt-operator sa token returned empty")

			By("querying Prometheus API endpoint for a VMI exported metric")
			cmd = []string{
				"curl",
				"-L",
				"-k",
				fmt.Sprintf("https://%s:%d/api/v1/query", promIP, promPort),
				"-H",
				fmt.Sprintf("Authorization: Bearer %s", token),
				"--data-urlencode",
				fmt.Sprintf(
					`query=kubevirt_vmi_memory_resident_bytes{namespace="%s",name="%s"}`,
					vmi.Namespace,
					vmi.Name,
				)}

			stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, &op, "virt-operator", cmd)
			Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, strings.Join(cmd, " "), stdout, stderr, err))

			// the Prometheus go-client does not export queryResult, and
			// using an HTTP client for queries would require a port-forwarding
			// since the cluster is running in a different network.
			var queryResult map[string]json.RawMessage

			err = json.Unmarshal([]byte(stdout), &queryResult)
			Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query result")

			var status string
			err = json.Unmarshal(queryResult["status"], &status)
			Expect(err).ToNot(HaveOccurred(), "failed to unmarshal query status")
			Expect(status).To(Equal("success"))
		})
	})

	Describe("[rfe_id:3187][crit:medium][vendor:cnv-qe@redhat.com][level:component]Prometheus Endpoints", func() {
		var preparedVMIs []*v1.VirtualMachineInstance
		var pod *k8sv1.Pod
		var handlerMetricIPs []string
		var controllerMetricIPs []string
		var getKubevirtVMMetrics func(string) string

		pinVMIOnNode := func(vmi *v1.VirtualMachineInstance, nodeName string) *v1.VirtualMachineInstance {
			if vmi == nil {
				return nil
			}
			if vmi.Spec.NodeSelector == nil {
				vmi.Spec.NodeSelector = make(map[string]string)
			}
			vmi.Spec.NodeSelector["kubernetes.io/hostname"] = nodeName
			return vmi
		}

		// collect metrics whose key contains the given string, expects non-empty result
		collectMetrics := func(ip, metricSubstring string) map[string]float64 {
			By("Scraping the Prometheus endpoint")
			var metrics map[string]float64
			var lines []string

			Eventually(func() map[string]float64 {
				out := getKubevirtVMMetrics(ip)
				lines = takeMetricsWithPrefix(out, metricSubstring)
				metrics, err = parseMetricsToMap(lines)
				Expect(err).ToNot(HaveOccurred())
				return metrics
			}, 30*time.Second, 2*time.Second).ShouldNot(BeEmpty())

			// troubleshooting helper
			fmt.Fprintf(GinkgoWriter, "metrics [%s]:\nlines=%s\n%#v\n", metricSubstring, lines, metrics)
			Expect(len(metrics)).To(BeNumerically(">=", float64(1.0)))
			Expect(metrics).To(HaveLen(len(lines)))

			return metrics
		}

		prepareVMIForTests := func(preferredNodeName string) string {
			By("Creating the VirtualMachineInstance")

			// WARNING: we assume the VM will have a VirtIO disk (vda)
			// and we add our own vdb on which we do our test.
			// but if the default disk is not vda, the test will break
			// TODO: introspect the VMI and get the device name of this
			// block device?
			vmi := tests.NewRandomVMIWithEphemeralDisk(cd.ContainerDiskFor(cd.ContainerDiskAlpine))
			tests.AppendEmptyDisk(vmi, "testdisk", v1.VirtIO, "1Gi")

			if preferredNodeName != "" {
				pinVMIOnNode(vmi, preferredNodeName)
			}
			nodeName := startVMI(vmi)
			if preferredNodeName != "" {
				Expect(nodeName).To(Equal(preferredNodeName), "Should run VMIs on the same node")
			}

			By("Expecting the VirtualMachineInstance console")
			// This also serves as a sync point to make sure the VM completed the boot
			// (and reduce the risk of false negatives)
			Expect(console.LoginToAlpine(vmi)).To(Succeed())

			By("Writing some data to the disk")
			Expect(console.SafeExpectBatch(vmi, []expect.Batcher{
				&expect.BSnd{S: "dd if=/dev/zero of=/dev/vdb bs=1M count=1\n"},
				&expect.BExp{R: console.PromptExpression},
				&expect.BSnd{S: "sync\n"},
				&expect.BExp{R: console.PromptExpression},
			}, 10)).To(Succeed())

			preparedVMIs = append(preparedVMIs, vmi)
			return nodeName
		}

		BeforeEach(func() {
			preparedVMIs = []*v1.VirtualMachineInstance{}
			pod = nil
			handlerMetricIPs = []string{}
			controllerMetricIPs = []string{}

			By("Finding the virt-controller prometheus endpoint")
			virtControllerLeaderPodName := getLeader()
			leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), virtControllerLeaderPodName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-controller pod")

			for _, ip := range leaderPod.Status.PodIPs {
				controllerMetricIPs = append(controllerMetricIPs, ip.IP)
			}

			// The initial test for the metrics subsystem used only a single VM for the sake of simplicity.
			// However, testing a single entity is a corner case (do we test handling sequences? potential clashes
			// in maps? and so on).
			// Thus, we run now two VMIs per testcase. A more realistic test would use a random number of VMIs >= 3,
			// but we don't do now to make test run quickly and (more important) because lack of resources on CI.

			nodeName := prepareVMIForTests("")
			// any node is fine, we don't really care, as long as we run all VMIs on it.
			prepareVMIForTests(nodeName)

			By("Finding the virt-handler prometheus endpoint")
			pod, err = libnode.GetVirtHandlerPod(virtClient, nodeName)
			Expect(err).ToNot(HaveOccurred(), "Should find the virt-handler pod")
			for _, ip := range pod.Status.PodIPs {
				handlerMetricIPs = append(handlerMetricIPs, ip.IP)
			}

			// returns metrics from the node the VMI(s) runs on
			getKubevirtVMMetrics = tests.GetKubevirtVMMetricsFunc(&virtClient, pod)
		})

		PIt("[test_id:4136][flaky] should find one leading virt-controller and two ready", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			foundMetrics := map[string]int{
				"ready":   0,
				"leading": 0,
			}
			By("scraping the metrics endpoint on virt-controller pods")
			for _, ep := range endpoint.Subsets[0].Addresses {
				if !strings.HasPrefix(ep.TargetRef.Name, "virt-controller") {
					continue
				}

				cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", tests.FormatIPForURL(ep.IP))
				stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))

				scrapedData := strings.Split(stdout, "\n")
				for _, data := range scrapedData {
					if strings.HasPrefix(data, "#") {
						continue
					}
					switch data {
					case "kubevirt_virt_controller_leading 1":
						foundMetrics["leading"]++
					case "kubevirt_virt_controller_ready 1":
						foundMetrics["ready"]++
					}
				}
			}

			Expect(foundMetrics["ready"]).To(Equal(2), "expected 2 ready virt-controllers")
			Expect(foundMetrics["leading"]).To(Equal(1), "expected 1 leading virt-controller")
		})

		It("[test_id:4137]should find one leading virt-operator and two ready", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			foundMetrics := map[string]int{
				"ready":   0,
				"leading": 0,
			}
			By("scraping the metrics endpoint on virt-operator pods")
			for _, ep := range endpoint.Subsets[0].Addresses {
				if !strings.HasPrefix(ep.TargetRef.Name, "virt-operator") {
					continue
				}

				cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", tests.FormatIPForURL(ep.IP))
				stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))

				scrapedData := strings.Split(stdout, "\n")
				for _, data := range scrapedData {
					if strings.HasPrefix(data, "#") {
						continue
					}
					switch data {
					case "kubevirt_virt_operator_leading 1":
						foundMetrics["leading"]++
					case "kubevirt_virt_operator_ready 1":
						foundMetrics["ready"]++
					}
				}
			}

			Expect(foundMetrics["ready"]).To(Equal(2), "expected 2 ready virt-operators")
			Expect(foundMetrics["leading"]).To(Equal(1), "expected 1 leading virt-operator")
		})

		It("[test_id:4138]should be exposed and registered on the metrics endpoint", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			l, err := labels.Parse("prometheus.kubevirt.io=true")
			Expect(err).ToNot(HaveOccurred())
			pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: l.String()})
			Expect(err).ToNot(HaveOccurred())
			Expect(endpoint.Subsets).To(HaveLen(1))

			By("checking if the endpoint contains the metrics port and only one matching subset")
			Expect(endpoint.Subsets[0].Ports).To(HaveLen(1))
			Expect(endpoint.Subsets[0].Ports[0].Name).To(Equal("metrics"))
			Expect(endpoint.Subsets[0].Ports[0].Port).To(Equal(int32(8443)))

			By("checking if  the IPs in the subset match the KubeVirt system Pod count")
			Expect(len(pods.Items)).To(BeNumerically(">=", 3), "At least one api, controller and handler need to be present")
			Expect(endpoint.Subsets[0].Addresses).To(HaveLen(len(pods.Items)))

			ips := map[string]string{}
			for _, ep := range endpoint.Subsets[0].Addresses {
				ips[ep.IP] = ""
			}
			for _, pod := range pods.Items {
				Expect(ips).To(HaveKey(pod.Status.PodIP), fmt.Sprintf("IP of Pod %s not found in metrics endpoint", pod.Name))
			}
		})
		It("[test_id:4139]should return Prometheus metrics", func() {
			endpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), "kubevirt-prometheus-metrics", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			for _, ep := range endpoint.Subsets[0].Addresses {
				cmd := fmt.Sprintf("curl -L -k https://%s:8443/metrics", tests.FormatIPForURL(ep.IP))
				stdout, stderr, err := exec.ExecuteCommandOnPodWithResults(virtClient, pod, "virt-handler", strings.Fields(cmd))
				Expect(err).ToNot(HaveOccurred(), fmt.Sprintf(remoteCmdErrPattern, cmd, stdout, stderr, err))
				Expect(stdout).To(ContainSubstring("go_goroutines"))
			}
		})

		DescribeTable("should throttle the Prometheus metrics access", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			if netutils.IsIPv6String(ip) {
				Skip("Skip testing with IPv6 until https://github.com/kubevirt/kubevirt/issues/4145 is fixed")
			}

			concurrency := 100 // random value "much higher" than maxRequestsInFlight

			tr := &http.Transport{
				MaxIdleConnsPerHost: concurrency,
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			}

			client := http.Client{
				Timeout:   time.Duration(1 * time.Second),
				Transport: tr,
			}

			errorsChan := make(chan error)
			By("Scraping the Prometheus endpoint")
			metricsURL := tests.PrepareMetricsURL(ip, 8443)
			for ix := 0; ix < concurrency; ix++ {
				go func(ix int) {
					req, _ := http.NewRequest("GET", metricsURL, nil)
					resp, err := client.Do(req)
					if err != nil {
						fmt.Fprintf(GinkgoWriter, "client: request: %v #%d: %v\n", req, ix, err) // troubleshooting helper
					} else {
						resp.Body.Close()
					}
					errorsChan <- err
				}(ix)
			}

			err := validatedHTTPResponses(errorsChan, concurrency)
			Expect(err).ToNot(HaveOccurred(), "Should throttle HTTP access without unexpected errors")
		},
			Entry("[test_id:4140] by using IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6226] by using IPv6", k8sv1.IPv6Protocol),
		)

		DescribeTable("should include the metrics for a running VM", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			By("Scraping the Prometheus endpoint")
			Eventually(func() string {
				out := getKubevirtVMMetrics(ip)
				lines := takeMetricsWithPrefix(out, "kubevirt")
				return strings.Join(lines, "\n")
			}, 30*time.Second, 2*time.Second).Should(ContainSubstring("kubevirt"))
		},
			Entry("[test_id:4141] by using IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6227] by using IPv6", k8sv1.IPv6Protocol),
		)

		DescribeTable("should include the storage metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			metrics := collectMetrics(ip, metricSubstring)
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, vmi := range preparedVMIs {
				for _, vol := range vmi.Spec.Volumes {
					key := getMetricKeyForVmiDisk(keys, vmi.Name, vol.Name)
					Expect(key).To(Not(BeEmpty()))

					value := metrics[key]
					fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
					Expect(value).To(BeNumerically(operator, float64(0.0)))

				}
			}
		},
			Entry("[test_id:4142] storage flush requests metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
			Entry("[test_id:6228] storage flush requests metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_requests_total", ">="),
			Entry("[test_id:4142] time (ms) spent on cache flushing metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_flush_times_ms_total", ">="),
			Entry("[test_id:6229] time (ms) spent on cache flushing metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_flush_times_ms_total", ">="),
			Entry("[test_id:4142] I/O read operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
			Entry("[test_id:6230] I/O read operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_read_total", ">="),
			Entry("[test_id:4142] I/O write operations metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
			Entry("[test_id:6231] I/O write operations metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_iops_write_total", ">="),
			Entry("[test_id:4142] storage read operation time metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_times_ms_total", ">="),
			Entry("[test_id:6232] storage read operation time metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_times_ms_total", ">="),
			Entry("[test_id:4142] storage read traffic in bytes metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
			Entry("[test_id:6233] storage read traffic in bytes metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_read_traffic_bytes_total", ">="),
			Entry("[test_id:4142] storage write operation time metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_times_ms_total", ">="),
			Entry("[test_id:6234] storage write operation time metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_times_ms_total", ">="),
			Entry("[test_id:4142] storage write traffic in bytes metric by using IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
			Entry("[test_id:6235] storage write traffic in bytes metric by using IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_storage_write_traffic_bytes_total", ">="),
		)

		DescribeTable("should include metrics for a running VM", func(family k8sv1.IPFamily, metricSubstring, operator string) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			metrics := collectMetrics(ip, metricSubstring)
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				value := metrics[key]
				fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
				Expect(value).To(BeNumerically(operator, float64(0.0)))
			}
		},
			Entry("[test_id:4143] network metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_network_", ">="),
			Entry("[test_id:6236] network metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_network_", ">="),
			Entry("[test_id:4144] memory metrics by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory", ">="),
			Entry("[test_id:6237] memory metrics by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory", ">="),
			Entry("[test_id:4553] vcpu wait by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_wait", "=="),
			Entry("[test_id:6238] vcpu wait by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_wait", "=="),
			Entry("[test_id:4554] vcpu seconds by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_vcpu_seconds", ">="),
			Entry("[test_id:6239] vcpu seconds by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_vcpu_seconds", ">="),
			Entry("[test_id:4556] vmi unused memory by IPv4", k8sv1.IPv4Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
			Entry("[test_id:6240] vmi unused memory by IPv6", k8sv1.IPv6Protocol, "kubevirt_vmi_memory_unused_bytes", ">="),
		)

		DescribeTable("should include VMI infos for a running VM", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			nodeName := pod.Spec.NodeName

			nameMatchers := []gomegatypes.GomegaMatcher{}
			for _, vmi := range preparedVMIs {
				nameMatchers = append(nameMatchers, ContainSubstring(`name="%s"`, vmi.Name))
			}

			for _, key := range keys {
				// we don't care about the ordering of the labels
				if strings.HasPrefix(key, "kubevirt_vmi_phase_count") {
					// special case: namespace and name don't make sense for this metric
					Expect(key).To(ContainSubstring(`node="%s"`, nodeName))
					continue
				}

				Expect(key).To(SatisfyAll(
					ContainSubstring(`node="%s"`, nodeName),
					// all testing VMIs are on the same node and namespace,
					// so checking the namespace of any random VMI is fine
					ContainSubstring(`namespace="%s"`, preparedVMIs[0].Namespace),
					// otherwise, each key must refer to exactly one the prepared VMIs.
					SatisfyAny(nameMatchers...),
				))
			}
		},
			Entry("[test_id:4145] by IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6241] by IPv6", k8sv1.IPv6Protocol),
		)

		DescribeTable("should include VMI phase metrics for all running VMs", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				if strings.Contains(key, `phase="running"`) {
					value := metrics[key]
					Expect(value).To(Equal(float64(len(preparedVMIs))))
				}
			}
		},
			Entry("[test_id:4146] by IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6242] by IPv6", k8sv1.IPv6Protocol),
		)

		DescribeTable("should include VMI eviction blocker status for all running VMs", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(controllerMetricIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_non_evictable")
			By("Checking the collected metrics")
			keys := getKeysFromMetrics(metrics)
			for _, key := range keys {
				value := metrics[key]
				fmt.Fprintf(GinkgoWriter, "metric value was %f\n", value)
				Expect(value).To(BeNumerically(">=", float64(0.0)))
			}
		},
			Entry("[test_id:4148] by IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6243] by IPv6", k8sv1.IPv6Protocol),
		)

		DescribeTable("should include kubernetes labels to VMI metrics", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			// Every VMI is labeled with kubevirt.io/nodeName, so just creating a VMI should
			// be enough to its metrics to contain a kubernetes label
			metrics := collectMetrics(ip, "kubevirt_vmi_vcpu_seconds")
			By("Checking collected metrics")
			keys := getKeysFromMetrics(metrics)
			containK8sLabel := false
			for _, key := range keys {
				if strings.Contains(key, "kubernetes_vmi_label_") {
					containK8sLabel = true
				}
			}
			Expect(containK8sLabel).To(BeTrue())
		},
			Entry("[test_id:4147] by IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6244] by IPv6", k8sv1.IPv6Protocol),
		)

		// explicit test fo swap metrics as test_id:4144 doesn't catch if they are missing
		DescribeTable("should include swap metrics", func(family k8sv1.IPFamily) {
			libnet.SkipWhenClusterNotSupportIPFamily(family)

			ip := getSupportedIP(handlerMetricIPs, family)

			metrics := collectMetrics(ip, "kubevirt_vmi_memory_swap_")
			var in, out bool
			for k := range metrics {
				if in && out {
					break
				}
				if strings.Contains(k, `swap_in`) {
					in = true
				}
				if strings.Contains(k, `swap_out`) {
					out = true
				}
			}

			Expect(in).To(BeTrue())
			Expect(out).To(BeTrue())
		},
			Entry("[test_id:4555] by IPv4", k8sv1.IPv4Protocol),
			Entry("[test_id:6245] by IPv6", k8sv1.IPv6Protocol),
		)
	})

	Describe("Start a VirtualMachineInstance", func() {
		Context("when the controller pod is not running and an election happens", func() {
			It("[test_id:4642]should succeed afterwards", func() {
				// This test needs at least 2 controller pods. Skip on single-replica.
				checks.SkipIfSingleReplica(virtClient)

				newLeaderPod := getNewLeaderPod(virtClient)
				Expect(newLeaderPod).NotTo(BeNil())

				// TODO: It can be race condition when newly deployed pod receive leadership, in this case we will need
				// to reduce Deployment replica before destroying the pod and to restore it after the test
				By("Destroying the leading controller pod")
				Eventually(func() string {
					leaderPodName := getLeader()

					Expect(virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Delete(context.Background(), leaderPodName, metav1.DeleteOptions{})).To(Succeed())

					Eventually(getLeader, 30*time.Second, 5*time.Second).ShouldNot(Equal(leaderPodName))

					leaderPod, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).Get(context.Background(), getLeader(), metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return leaderPod.Name
				}, 90*time.Second, 5*time.Second).Should(Equal(newLeaderPod.Name))

				Expect(matcher.ThisPod(newLeaderPod)()).To(matcher.HaveConditionTrue(k8sv1.PodReady))

				vmi := tests.NewRandomVMI()

				By("Starting a new VirtualMachineInstance")
				obj, err := virtClient.RestClient().Post().Resource("virtualmachineinstances").Namespace(testsuite.GetTestNamespace(vmi)).Body(vmi).Do(context.Background()).Get()
				Expect(err).ToNot(HaveOccurred())
				vmiObj, ok := obj.(*v1.VirtualMachineInstance)
				Expect(ok).To(BeTrue(), "Object is not of type *v1.VirtualMachineInstance")
				libwait.WaitForSuccessfulVMIStart(vmiObj)
			})
		})

	})

	Describe("Node-labeller", func() {
		var nodesWithKVM []*k8sv1.Node
		var nonExistingCPUModelLabel = v1.CPUModelLabel + "someNonExistingCPUModel"
		type patch struct {
			Op    string            `json:"op"`
			Path  string            `json:"path"`
			Value map[string]string `json:"value"`
		}

		BeforeEach(func() {
			nodesWithKVM = libnode.GetNodesWithKVM()
			if len(nodesWithKVM) == 0 {
				Skip("Skip testing with node-labeller, because there are no nodes with kvm")
			}
		})
		AfterEach(func() {
			nodesWithKVM = libnode.GetNodesWithKVM()

			for _, node := range nodesWithKVM {
				libnode.RemoveLabelFromNode(node.Name, nonExistingCPUModelLabel)
				libnode.RemoveAnnotationFromNode(node.Name, v1.LabellerSkipNodeAnnotation)
			}
			wakeNodeLabellerUp(virtClient)

			for _, node := range nodesWithKVM {
				Eventually(func() error {
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					if _, exists := node.Labels[nonExistingCPUModelLabel]; exists {
						return fmt.Errorf("node %s is expected to not have label key %s", node.Name, nonExistingCPUModelLabel)
					}

					if _, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]; exists {
						return fmt.Errorf("node %s is expected to not have annotation key %s", node.Name, v1.LabellerSkipNodeAnnotation)
					}

					return nil
				}, 30*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
			}
		})

		expectNodeLabels := func(nodeName string, labelValidation func(map[string]string) (valid bool, errorMsg string)) {
			var errorMsg string

			EventuallyWithOffset(1, func() (isValid bool) {
				node, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodeName, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				isValid, errorMsg = labelValidation(node.Labels)

				return isValid
			}, 30*time.Second, 2*time.Second).Should(BeTrue(), errorMsg)
		}

		Context("basic labelling", func() {
			It("skip node reconciliation when node has skip annotation", func() {

				for i, node := range nodesWithKVM {
					node.Labels[nonExistingCPUModelLabel] = "true"
					p := []patch{
						{
							Op:    "add",
							Path:  "/metadata/labels",
							Value: node.Labels,
						},
					}
					if i == 0 {
						node.Annotations[v1.LabellerSkipNodeAnnotation] = "true"

						p = append(p, patch{
							Op:    "add",
							Path:  "/metadata/annotations",
							Value: node.Annotations,
						})
					}
					payloadBytes, err := json.Marshal(p)
					Expect(err).ToNot(HaveOccurred())

					_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), node.Name, types.JSONPatchType, payloadBytes, metav1.PatchOptions{})
					Expect(err).ToNot(HaveOccurred())
				}
				kvConfig := v1.KubeVirtConfiguration{ObsoleteCPUModels: map[string]bool{}}
				// trigger reconciliation
				tests.UpdateKubeVirtConfigValueAndWait(kvConfig)

				Eventually(func() bool {
					nodesWithKVM = libnode.GetNodesWithKVM()

					for _, node := range nodesWithKVM {
						_, skipAnnotationFound := node.Annotations[v1.LabellerSkipNodeAnnotation]
						_, customLabelFound := node.Labels[nonExistingCPUModelLabel]
						if customLabelFound && !skipAnnotationFound {
							return false
						}
					}
					return true
				}, 15*time.Second, 1*time.Second).Should(BeTrue())
			})

			It("[test_id:6246] label nodes with cpu model, cpu features and host cpu model", func() {
				for _, node := range nodesWithKVM {
					Expect(err).ToNot(HaveOccurred())
					cpuModelLabelPresent := false
					cpuFeatureLabelPresent := false
					hyperVLabelPresent := false
					hostCpuModelPresent := false
					hostCpuRequiredFeaturesPresent := false
					for key := range node.Labels {
						if strings.Contains(key, v1.CPUModelLabel) {
							cpuModelLabelPresent = true
						}
						if strings.Contains(key, v1.CPUFeatureLabel) {
							cpuFeatureLabelPresent = true
						}
						if strings.Contains(key, v1.HypervLabel) {
							hyperVLabelPresent = true
						}
						if strings.Contains(key, v1.HostModelCPULabel) {
							hostCpuModelPresent = true
						}
						if strings.Contains(key, v1.HostModelRequiredFeaturesLabel) {
							hostCpuRequiredFeaturesPresent = true
						}

						if cpuModelLabelPresent && cpuFeatureLabelPresent && hyperVLabelPresent && hostCpuModelPresent &&
							hostCpuRequiredFeaturesPresent {
							break
						}
					}

					errorMessageTemplate := "node " + node.Name + " does not contain %s label"
					Expect(cpuModelLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "cpu"))
					Expect(cpuFeatureLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "feature"))
					Expect(hyperVLabelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "hyperV"))
					Expect(hostCpuModelPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu model"))
					Expect(hostCpuRequiredFeaturesPresent).To(BeTrue(), fmt.Sprintf(errorMessageTemplate, "host cpu required featuers"))
				}
			})

			It("[test_id:6247] should set default obsolete cpu models filter when obsolete-cpus-models is not set in kubevirt config", func() {
				node := nodesWithKVM[0]

				for key := range node.Labels {
					if strings.Contains(key, v1.CPUModelLabel) {
						model := strings.TrimPrefix(key, v1.CPUModelLabel)
						Expect(nodelabellerutil.DefaultObsoleteCPUModels).ToNot(HaveKey(model),
							"Node can't contain label with cpu model, which is in default obsolete filter")
					}
				}
			})

			It("[test_id:6995]should expose tsc frequency and tsc scalability", func() {
				node := nodesWithKVM[0]
				Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-frequency"))
				Expect(node.Labels).To(HaveKey("cpu-timer.node.kubevirt.io/tsc-scalable"))
				Expect(node.Labels["cpu-timer.node.kubevirt.io/tsc-scalable"]).To(Or(Equal("true"), Equal("false")))
				val, err := strconv.ParseInt(node.Labels["cpu-timer.node.kubevirt.io/tsc-frequency"], 10, 64)
				Expect(err).ToNot(HaveOccurred())
				Expect(val).To(BeNumerically(">", 0))
			})
		})

		Context("advanced labelling", func() {
			var originalKubeVirt *v1.KubeVirt

			BeforeEach(func() {
				originalKubeVirt = util.GetCurrentKv(virtClient)
			})

			AfterEach(func() {
				tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
			})

			It("[test_id:6249] should update node with new cpu model label set", func() {
				obsoleteModel := ""
				node := nodesWithKVM[0]

				kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
				kvConfig.ObsoleteCPUModels = make(map[string]bool)

				for key := range node.Labels {
					if strings.Contains(key, v1.CPUModelLabel) {
						obsoleteModel = strings.TrimPrefix(key, v1.CPUModelLabel)
						kvConfig.ObsoleteCPUModels[obsoleteModel] = true
						break
					}
				}

				tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

				labelKeyExpectedToBeMissing := v1.CPUModelLabel + obsoleteModel
				expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
					_, exists := m[labelKeyExpectedToBeMissing]
					return !exists, fmt.Sprintf("node %s is expected to not have label key %s", node.Name, labelKeyExpectedToBeMissing)
				})
			})

			It("[test_id:6250] should update node with new cpu model vendor label", func() {
				nodes, err := virtClient.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
				Expect(err).ToNot(HaveOccurred())
				for _, node := range nodes.Items {
					for key := range node.Labels {
						if strings.HasPrefix(key, v1.CPUModelVendorLabel) {
							return
						}
					}
				}

				Fail("No node contains label " + v1.CPUModelVendorLabel)
			})

			It("[test_id:6252] should remove all cpu model labels (all cpu model are in obsolete list)", func() {
				node := nodesWithKVM[0]

				obsoleteModels := nodelabellerutil.DefaultObsoleteCPUModels

				for key := range node.Labels {
					if strings.Contains(key, v1.CPUModelLabel) {
						obsoleteModels[strings.TrimPrefix(key, v1.CPUModelLabel)] = true
					}
					if strings.Contains(key, v1.SupportedHostModelMigrationCPU) {
						obsoleteModels[strings.TrimPrefix(key, v1.SupportedHostModelMigrationCPU)] = true
					}
				}

				kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
				kvConfig.ObsoleteCPUModels = obsoleteModels
				tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

				expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
					found := false
					label := ""
					for key := range m {
						if strings.Contains(key, v1.CPUModelLabel) || strings.Contains(key, v1.SupportedHostModelMigrationCPU) {
							found = true
							label = key
							break
						}
					}

					return !found, fmt.Sprintf("node %s should not contain any cpu model label, but contains %s", node.Name, label)
				})
			})
		})

		Context("[Serial]node with obsolete host-model cpuModel", Serial, func() {

			expectSerialRun := func() {
				Expect(CurrentSpecReport().IsSerial).To(BeTrue(), "this test is supported for serial tests only")
			}

			expectAtLeastOneEvent := func(eventListOpts metav1.ListOptions, namespace string) *k8sv1.EventList {
				// This function is dangerous to use from parallel tests as events might override each other.
				// This can be removed in the future if these functions are used with great caution.
				expectSerialRun()
				var events *k8sv1.EventList

				Eventually(func() []k8sv1.Event {
					events, err = virtClient.CoreV1().Events(namespace).List(context.Background(), eventListOpts)
					Expect(err).ToNot(HaveOccurred())

					return events.Items
				}, 30*time.Second, 1*time.Second).ShouldNot(BeEmpty())

				return events
			}
			deleteEvents := func(eventListOpts metav1.ListOptions, eventList *k8sv1.EventList) {
				// See comment in expectAtLeastOneEvent() for more info on why that's needed.
				if len(eventList.Items) == 0 {
					return
				}
				namespace := eventList.Items[0].Namespace
				expectSerialRun()
				for _, event := range eventList.Items {
					err = virtClient.CoreV1().Events(event.Namespace).Delete(context.Background(), event.Name, metav1.DeleteOptions{})
					Expect(err).ToNot(HaveOccurred())
				}

				By("Expecting alert to be removed")
				Eventually(func() []k8sv1.Event {
					events, err := virtClient.CoreV1().Events(namespace).List(context.Background(), eventListOpts)
					Expect(err).ToNot(HaveOccurred())

					return events.Items
				}, 30*time.Second, 1*time.Second).Should(BeEmpty())
			}

			var node *k8sv1.Node
			var obsoleteModel string
			var kvConfig *v1.KubeVirtConfiguration
			var events *k8sv1.EventList
			var eventListOpts metav1.ListOptions

			BeforeEach(func() {
				node = &(libnode.GetAllSchedulableNodes(virtClient).Items[0])
				obsoleteModel = tests.GetNodeHostModel(node)

				By("Updating Kubevirt CR , this should wake node-labeller ")
				kvConfig = util.GetCurrentKv(virtClient).Spec.Configuration.DeepCopy()
				if kvConfig.ObsoleteCPUModels == nil {
					kvConfig.ObsoleteCPUModels = make(map[string]bool)
				}
				kvConfig.ObsoleteCPUModels[obsoleteModel] = true
				tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

				Eventually(func() error {
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					_, exists := node.Annotations[v1.LabellerSkipNodeAnnotation]
					if exists {
						return fmt.Errorf("node %s is expected to not have annotation %s", node.Name, v1.LabellerSkipNodeAnnotation)
					}

					obsoleteModelLabelFound := false
					for labelKey, _ := range node.Labels {
						if strings.Contains(labelKey, v1.NodeHostModelIsObsoleteLabel) {
							obsoleteModelLabelFound = true
							break
						}
					}
					if !obsoleteModelLabelFound {
						return fmt.Errorf("node %s is expected to have a label with %s substring. this means node-labeller is not enabled for the node", node.Name, v1.NodeHostModelIsObsoleteLabel)
					}

					return nil
				}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
			})

			AfterEach(func() {
				delete(kvConfig.ObsoleteCPUModels, obsoleteModel)
				tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
				deleteEvents(eventListOpts, events)

				Eventually(func() error {
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					Expect(err).ShouldNot(HaveOccurred())

					obsoleteHostModelLabel := false
					for labelKey, _ := range node.Labels {
						if strings.HasPrefix(labelKey, v1.NodeHostModelIsObsoleteLabel) {
							obsoleteHostModelLabel = true
							break
						}
					}
					if obsoleteHostModelLabel {
						return fmt.Errorf("node %s is expected to have a label with %s prefix. this means node-labeller is not enabled for the node", node.Name, v1.HostModelCPULabel)
					}

					return nil
				}, 30*time.Second, time.Second).ShouldNot(HaveOccurred())
			})

			It("[Serial]should not schedule vmi with host-model cpuModel to node with obsolete host-model cpuModel", func() {
				vmi := libvmi.NewFedora(
					libvmi.WithInterface(libvmi.InterfaceDeviceWithMasqueradeBinding()),
					libvmi.WithNetwork(v1.DefaultPodNetwork()),
				)
				By("Making sure the vmi start running on the source node and will be able to run only in source/target nodes")
				vmi.Spec.NodeSelector = map[string]string{"kubernetes.io/hostname": node.Name}

				By("Starting the VirtualMachineInstance")
				_, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi)
				Expect(err).ToNot(HaveOccurred())

				By("Checking that the VMI failed")
				Eventually(func() bool {
					vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Get(context.Background(), vmi.Name, &metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					for _, condition := range vmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceConditionType(k8sv1.PodScheduled) && condition.Status == k8sv1.ConditionFalse {
							return strings.Contains(condition.Message, "didn't match Pod's node affinity/selector")
						}
					}
					return false
				}, 3*time.Minute, 2*time.Second).Should(BeTrue())

				By("Expecting for an alert to be triggered")
				eventListOpts = metav1.ListOptions{
					FieldSelector: fmt.Sprintf("type=%s,reason=%s", k8sv1.EventTypeWarning, "HostModelIsObsolete"),
				}
				events = expectAtLeastOneEvent(eventListOpts, node.Namespace)
			})

		})

		Context("Clean up after old labeller", func() {
			nfdLabel := "feature.node.kubernetes.io/some-fancy-feature-which-should-not-be-deleted"
			var originalKubeVirt *v1.KubeVirt

			BeforeEach(func() {
				originalKubeVirt = util.GetCurrentKv(virtClient)

			})

			AfterEach(func() {
				originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				node := originalNode.DeepCopy()

				for key := range node.Labels {
					if strings.Contains(key, nfdLabel) {
						delete(node.Labels, nfdLabel)
					}
				}
				originalLabelsBytes, err := json.Marshal(originalNode.Labels)
				Expect(err).ToNot(HaveOccurred())

				labelsBytes, err := json.Marshal(node.Labels)
				Expect(err).ToNot(HaveOccurred())

				patchTestLabels := fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s}`, string(originalLabelsBytes))
				patchLabels := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s}`, string(labelsBytes))

				data := []byte(fmt.Sprintf("[ %s, %s ]", patchTestLabels, patchLabels))

				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodesWithKVM[0].Name, types.JSONPatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
			})

			It("[test_id:6253] should remove old labeller labels and annotations", func() {
				originalNode, err := virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				node := originalNode.DeepCopy()

				node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedcpuModelPrefix+"Penryn"] = "true"
				node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedcpuFeaturePrefix+"mmx"] = "true"
				node.Labels[nodelabellerutil.DeprecatedLabelNamespace+nodelabellerutil.DeprecatedHyperPrefix+"synic"] = "true"
				node.Labels[nfdLabel] = "true"
				node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedcpuModelPrefix+"Penryn"] = "true"
				node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedcpuFeaturePrefix+"mmx"] = "true"
				node.Annotations[nodelabellerutil.DeprecatedLabellerNamespaceAnnotation+nodelabellerutil.DeprecatedHyperPrefix+"synic"] = "true"

				originalLabelsBytes, err := json.Marshal(originalNode.Labels)
				Expect(err).ToNot(HaveOccurred())

				originalAnnotationsBytes, err := json.Marshal(originalNode.Annotations)
				Expect(err).ToNot(HaveOccurred())

				labelsBytes, err := json.Marshal(node.Labels)
				Expect(err).ToNot(HaveOccurred())

				annotationsBytes, err := json.Marshal(node.Annotations)
				Expect(err).ToNot(HaveOccurred())

				patchTestLabels := fmt.Sprintf(`{ "op": "test", "path": "/metadata/labels", "value": %s}`, string(originalLabelsBytes))
				patchTestAnnotations := fmt.Sprintf(`{ "op": "test", "path": "/metadata/annotations", "value": %s}`, string(originalAnnotationsBytes))
				patchLabels := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/labels", "value": %s}`, string(labelsBytes))
				patchAnnotations := fmt.Sprintf(`{ "op": "replace", "path": "/metadata/annotations", "value": %s}`, string(annotationsBytes))

				data := []byte(fmt.Sprintf("[ %s, %s, %s, %s ]", patchTestLabels, patchLabels, patchTestAnnotations, patchAnnotations))

				_, err = virtClient.CoreV1().Nodes().Patch(context.Background(), nodesWithKVM[0].Name, types.JSONPatchType, data, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
				kvConfig.ObsoleteCPUModels = map[string]bool{"486": true}
				tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)

				expectNodeLabels(node.Name, func(m map[string]string) (valid bool, errorMsg string) {
					foundSpecialLabel := false

					for key := range m {
						for _, deprecatedPrefix := range []string{nodelabellerutil.DeprecatedcpuModelPrefix, nodelabellerutil.DeprecatedcpuFeaturePrefix, nodelabellerutil.DeprecatedHyperPrefix} {
							fullDeprecationLabel := nodelabellerutil.DeprecatedLabelNamespace + deprecatedPrefix
							if strings.Contains(key, fullDeprecationLabel) {
								return false, fmt.Sprintf("node %s should not contain any label with prefix %s", node.Name, fullDeprecationLabel)
							}
						}

						if key == nfdLabel {
							foundSpecialLabel = true
						}
					}

					if !foundSpecialLabel {
						return false, "labeller should not delete NFD labels"
					}

					return true, ""
				})

				Eventually(func() error {
					node, err = virtClient.CoreV1().Nodes().Get(context.Background(), nodesWithKVM[0].Name, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					for key := range node.Annotations {
						if strings.Contains(key, nodelabellerutil.DeprecatedLabellerNamespaceAnnotation) {
							return fmt.Errorf("node %s shouldn't contain any annotations with prefix %s, but found annotation key %s", node.Name, nodelabellerutil.DeprecatedLabellerNamespaceAnnotation, key)
						}
					}

					return nil
				}, 30*time.Second, 2*time.Second).ShouldNot(HaveOccurred())
			})

		})
	})

	Describe("virt-handler", func() {
		var (
			originalKubeVirt *v1.KubeVirt
			nodesToEnableKSM []*k8sv1.Node
		)
		type ksmTestFunc func() (*v1.KSMConfiguration, []*k8sv1.Node)

		getNodesWithKSMAvailable := func(virtCli kubecli.KubevirtClient) []*k8sv1.Node {
			nodes := libnode.GetAllSchedulableNodes(virtCli)

			nodesWithKSM := make([]*k8sv1.Node, 0)
			for _, node := range nodes.Items {
				command := []string{"cat", "/sys/kernel/mm/ksm/run"}
				_, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
				if err == nil {
					nodesWithKSM = append(nodesWithKSM, &node)
				}
			}
			return nodesWithKSM
		}

		BeforeEach(func() {
			nodesToEnableKSM = getNodesWithKSMAvailable(virtClient)
			if len(nodesToEnableKSM) == 0 {
				Fail("There isn't any node with KSM available")
			}
			originalKubeVirt = util.GetCurrentKv(virtClient)
		})

		AfterEach(func() {
			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)
		})

		DescribeTable("should enable/disable ksm and add/remove annotation", decorators.KSMRequired, func(ksmConfigFun ksmTestFunc) {
			kvConfig := originalKubeVirt.Spec.Configuration.DeepCopy()
			ksmConfig, expectedEnabledNodes := ksmConfigFun()
			kvConfig.KSMConfiguration = ksmConfig
			tests.UpdateKubeVirtConfigValueAndWait(*kvConfig)
			By("Ensure ksm is enabled and annotation is added in the expected nodes")
			for _, node := range expectedEnabledNodes {
				Eventually(func() (string, error) {
					command := []string{"cat", "/sys/kernel/mm/ksm/run"}
					ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
					if err != nil {
						return "", err
					}

					return ksmValue, nil
				}, 30*time.Second, 2*time.Second).Should(BeEquivalentTo("1\n"), fmt.Sprintf("KSM should be enabled in node %s", node.Name))

				Eventually(func() (bool, error) {
					node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					_, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
					return found, nil
				}, 30*time.Second, 2*time.Second).Should(BeTrue(), fmt.Sprintf("Node %s should have %s annotation", node.Name, v1.KSMHandlerManagedAnnotation))
			}

			tests.UpdateKubeVirtConfigValueAndWait(originalKubeVirt.Spec.Configuration)

			By("Ensure ksm is disabled and annotation is removed in the expected nodes")
			for _, node := range expectedEnabledNodes {
				Eventually(func() (string, error) {
					command := []string{"cat", "/sys/kernel/mm/ksm/run"}
					ksmValue, err := tests.ExecuteCommandInVirtHandlerPod(node.Name, command)
					if err != nil {
						return "", err
					}

					return ksmValue, nil
				}, 30*time.Second, 2*time.Second).Should(BeEquivalentTo("0\n"), fmt.Sprintf("KSM should be disabled in node %s", node.Name))

				Eventually(func() (bool, error) {
					node, err := virtClient.CoreV1().Nodes().Get(context.Background(), node.Name, metav1.GetOptions{})
					if err != nil {
						return false, err
					}
					_, found := node.GetAnnotations()[v1.KSMHandlerManagedAnnotation]
					return found, nil
				}, 30*time.Second, 2*time.Second).Should(BeFalse(), fmt.Sprintf("Annotation %s should be removed from the node %s", v1.KSMHandlerManagedAnnotation, node.Name))
			}
		},
			Entry("in specific nodes when the selector with MatchLabels matches the node label", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"kubernetes.io/hostname": nodesToEnableKSM[0].Name,
						},
					},
				}, []*k8sv1.Node{nodesToEnableKSM[0]}
			}),
			Entry("in specific nodes when the selector with MatchExpressions matches the node label", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: metav1.LabelSelectorOpIn,
								Values:   []string{nodesToEnableKSM[0].Name},
							},
						},
					},
				}, []*k8sv1.Node{nodesToEnableKSM[0]}
			}),
			Entry("in all the nodes when the selector is empty", func() (*v1.KSMConfiguration, []*k8sv1.Node) {
				return &v1.KSMConfiguration{
					NodeLabelSelector: &metav1.LabelSelector{},
				}, nodesToEnableKSM
			}),
		)
	})

	Describe("cluster profiler for pprof data aggregation", func() {
		Context("when ClusterProfiler feature gate", func() {
			It("is disabled it should prevent subresource access", func() {
				tests.DisableFeatureGate("ClusterProfiler")

				err := virtClient.ClusterProfiler().Start()
				Expect(err).To(HaveOccurred())

				err = virtClient.ClusterProfiler().Stop()
				Expect(err).To(HaveOccurred())

				_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
				Expect(err).To(HaveOccurred())
			})
			It("is enabled it should allow subresource access", func() {
				tests.EnableFeatureGate("ClusterProfiler")

				err := virtClient.ClusterProfiler().Start()
				Expect(err).ToNot(HaveOccurred())

				err = virtClient.ClusterProfiler().Stop()
				Expect(err).ToNot(HaveOccurred())

				_, err = virtClient.ClusterProfiler().Dump(&v1.ClusterProfilerRequest{})
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

func getLeader() string {
	virtClient := kubevirt.Client()

	controllerEndpoint, err := virtClient.CoreV1().Endpoints(flags.KubeVirtInstallNamespace).Get(context.Background(), leaderelectionconfig.DefaultEndpointName, metav1.GetOptions{})
	util.PanicOnError(err)

	var record resourcelock.LeaderElectionRecord
	if recordBytes, found := controllerEndpoint.Annotations[resourcelock.LeaderElectionRecordAnnotationKey]; found {
		err := json.Unmarshal([]byte(recordBytes), &record)
		util.PanicOnError(err)
	}
	return record.HolderIdentity
}

func getNewLeaderPod(virtClient kubecli.KubevirtClient) *k8sv1.Pod {
	labelSelector, err := labels.Parse(fmt.Sprint(v1.AppLabel + "=virt-controller"))
	util.PanicOnError(err)
	fieldSelector := fields.ParseSelectorOrDie("status.phase=" + string(k8sv1.PodRunning))
	controllerPods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(),
		metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
	util.PanicOnError(err)
	leaderPodName := getLeader()
	for _, pod := range controllerPods.Items {
		if pod.Name != leaderPodName {
			return &pod
		}
	}
	return nil
}

func parseMetricsToMap(lines []string) (map[string]float64, error) {
	metrics := make(map[string]float64)
	for _, line := range lines {
		items := strings.Split(line, " ")
		if len(items) != 2 {
			return nil, fmt.Errorf("can't split properly line '%s'", line)
		}
		v, err := strconv.ParseFloat(items[1], 64)
		if err != nil {
			return nil, err
		}
		metrics[items[0]] = v
	}
	return metrics, nil
}

func takeMetricsWithPrefix(output, prefix string) []string {
	lines := strings.Split(output, "\n")
	var ret []string
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			ret = append(ret, line)
		}
	}
	return ret
}

func getKeysFromMetrics(metrics map[string]float64) []string {
	var keys []string
	for metric := range metrics {
		keys = append(keys, metric)
	}
	// we sort keys only to make debug of test failures easier
	sort.Strings(keys)
	return keys
}

// validatedHTTPResponses checks the HTTP responses.
// It expects timeout errors, due to the throttling on the producer side.
// In case of unexpected errors or no errors at all it would fail,
// returning the first unexpected error if any, or a custom error in case
// there were no errors at all.
func validatedHTTPResponses(errorsChan chan error, concurrency int) error {
	var expectedErrorsCount = 0
	var unexpectedError error
	for ix := 0; ix < concurrency; ix++ {
		err := <-errorsChan
		if unexpectedError == nil && err != nil {
			var e *neturl.Error
			if errors.As(err, &e) && e.Timeout() {
				expectedErrorsCount++
			} else {
				unexpectedError = err
			}
		}
	}

	if unexpectedError == nil && expectedErrorsCount == 0 {
		return fmt.Errorf("timeout errors were expected due to throttling")
	}

	return unexpectedError
}

func getSupportedIP(ips []string, family k8sv1.IPFamily) string {
	ip := libnet.GetIP(ips, family)
	ExpectWithOffset(1, ip).NotTo(BeEmpty())

	return ip
}

func getMetricKeyForVmiDisk(keys []string, vmiName string, diskName string) string {
	for _, key := range keys {
		if strings.Contains(key, "name=\""+vmiName+"\"") &&
			strings.Contains(key, "drive=\""+diskName+"\"") {
			return key
		}
	}
	return ""
}

func getDownwardMetrics(vmi *v1.VirtualMachineInstance) (*api.Metrics, error) {
	res, err := console.SafeExpectBatchWithResponse(vmi, []expect.Batcher{
		&expect.BSnd{S: `sudo vm-dump-metrics 2> /dev/null` + "\n"},
		&expect.BExp{R: `(?s)(<metrics>.+</metrics>)`},
	}, 5)
	if err != nil {
		return nil, err
	}
	metricsStr := res[0].Match[2]
	metrics := &api.Metrics{}
	Expect(xml.Unmarshal([]byte(metricsStr), metrics)).To(Succeed())
	return metrics, nil
}

func getTimeFromMetrics(metrics *api.Metrics) int {

	for _, m := range metrics.Metrics {
		if m.Name == "Time" {
			val, err := strconv.Atoi(m.Value)
			Expect(err).ToNot(HaveOccurred())
			return val
		}
	}
	Fail("no Time in metrics XML")
	return -1
}

func getHostnameFromMetrics(metrics *api.Metrics) string {
	for _, m := range metrics.Metrics {
		if m.Name == "HostName" {
			return m.Value
		}
	}
	Fail("no hostname in metrics XML")
	return ""
}

func containsCrt(bundle []byte, containedCrt []byte) bool {
	crts, err := cert.ParseCertsPEM(bundle)
	Expect(err).ToNot(HaveOccurred())
	attached := false
	for _, crt := range crts {
		crtBytes := cert.EncodeCertPEM(crt)
		if reflect.DeepEqual(crtBytes, containedCrt) {
			attached = true
			break
		}
	}
	return attached
}

func getNodeWithOneOfPods(virtClient kubecli.KubevirtClient, pods []k8sv1.Pod) string {
	schedulableNodesList := libnode.GetAllSchedulableNodes(virtClient)
	schedulableNodes := map[string]*k8sv1.Node{}
	for _, node := range schedulableNodesList.Items {
		schedulableNodes[node.Name] = node.DeepCopy()
	}

	// control-plane nodes should never have the CriticalAddonsOnly taint because core components might not
	// tolerate this taint because it is meant to be used on compute nodes only. If we set this taint
	// on a control-plane node, we risk in breaking the test cluster.
	for _, pod := range pods {
		node, ok := schedulableNodes[pod.Spec.NodeName]
		if !ok {
			// Pod is running on a non-schedulable node?
			continue
		}

		if _, isControlPlane := node.Labels["node-role.kubernetes.io/control-plane"]; isControlPlane {
			continue
		}

		return node.Name
	}
	return ""
}

func filterKubevirtPods(pods []k8sv1.Pod) []string {
	kubevirtPodPrefixes := []string{
		"virt-handler",
		"virt-controller",
		"virt-api",
		"virt-operator",
	}

	var result []string
	for _, pod := range pods {
		if pod.Namespace != flags.KubeVirtInstallNamespace {
			continue
		}
		for _, prefix := range kubevirtPodPrefixes {
			if strings.HasPrefix(pod.Name, prefix) {
				result = append(result, pod.Name)
				break
			}
		}
	}
	return result
}

func getDeploymentsForPods(virtClient kubecli.KubevirtClient, pods []k8sv1.Pod) []types.NamespacedName {
	// Listing all deployments to find which ones belong to the pods.
	allDeployments, err := virtClient.AppsV1().Deployments("").List(context.Background(), metav1.ListOptions{})
	Expect(err).NotTo(HaveOccurred())

	var result []types.NamespacedName
	for _, deployment := range allDeployments.Items {
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		Expect(err).NotTo(HaveOccurred())

		for _, pod := range pods {
			if selector.Matches(labels.Set(pod.Labels)) {
				result = append(result, types.NamespacedName{
					Namespace: deployment.Namespace,
					Name:      deployment.Name,
				})
				break
			}
		}
	}
	return result
}
