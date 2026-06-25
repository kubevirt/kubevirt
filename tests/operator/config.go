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
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/api/instancetype/v1beta1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	virtwait "kubevirt.io/kubevirt/pkg/apimachinery/wait"
	"kubevirt.io/kubevirt/pkg/certificates/triple"
	"kubevirt.io/kubevirt/pkg/certificates/triple/cert"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/framework/matcher"
	. "kubevirt.io/kubevirt/tests/framework/matcher"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-operator]Operator", Serial, decorators.SigOperator, func() {
	var (
		virtClient              kubecli.KubevirtClient
		originalKv              *v1.KubeVirt
		originalOperatorVersion string
		err                     error
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		originalKv = libkubevirt.GetCurrentKv(virtClient)

		_, _, _, _, version := parseOperatorImage()
		const prefix = ":"
		Expect(strings.HasPrefix(version, prefix)).To(BeTrue(), fmt.Sprintf("version %s is expected to start with %s", version, prefix))
		originalOperatorVersion = strings.TrimPrefix(version, prefix)

		verifyOperatorWebhookCertificate()

		DeferCleanup(func() {
			deleteAllKvAndWait(true, originalKv.Name)

			kvs := libkubevirt.GetKvList(virtClient)
			if len(kvs) == 0 {
				By("Re-creating the original KV to stabilize")
				createKv(copyOriginalKv(originalKv))
			}

			modified := patchOperator(nil, &originalOperatorVersion)
			if modified {
				waitForUpdateCondition(originalKv)
			}

			By("Waiting for original KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(originalKv, 420*time.Second)
			allKvInfraPodsAreReady(originalKv)

			verifyOperatorWebhookCertificate()

			_, err = virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "disks-images-provider", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("[rfe_id:4356]Node Placement", func() {
		It("[test_id:4927]should dynamically update infra config", func() {
			// This label shouldn't exist, but this isn't harmful
			// existing/running deployments will not be torn down until
			// new ones are stood up (and the new ones will get stuck in scheduling)
			fakeLabelKey := "kubevirt-test"
			fakeLabelValue := "test-label"
			infra := &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					NodeSelector: map[string]string{fakeLabelKey: fakeLabelValue},
				},
			}
			By("Adding fake label to Virt components")
			Expect(patchKVInfra(originalKv, infra)).To(Succeed())

			for _, deploymentName := range []string{"virt-controller", "virt-api"} {
				errMsg := "NodeSelector should be propagated to the deployment eventually"
				Eventually(func() bool {
					return nodeSelectorExistInDeployment(deploymentName, fakeLabelKey, fakeLabelValue)
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), errMsg)
				//The reason we check this is that sometime it takes a while until the pod is created and
				//if the pod is created after the call to allKvInfraPodsAreReady in the AfterEach scope
				//than we will run the next test with side effect of pending pods of virt-api and virt-controller
				//and increase flakiness
				errMsg = "the deployment should try to rollup the pods with the new selector and fail to schedule pods because the nodes don't have the fake label"
				Eventually(func() bool {
					return atLeastOnePendingPodExistInDeployment(deploymentName)
				}, 60*time.Second, 1*time.Second).Should(BeTrue(), errMsg)
			}
			Expect(patchKVInfra(originalKv, nil)).To(Succeed())
		})

		It("[test_id:4928]should dynamically update workloads config", func() {
			labelKey := "kubevirt-test"
			labelValue := "test-label"
			workloads := &v1.ComponentConfig{
				NodePlacement: &v1.NodePlacement{
					NodeSelector: map[string]string{labelKey: labelValue},
				},
			}
			Expect(patchKVWorkloads(originalKv, workloads)).To(Succeed())

			Eventually(func() bool {
				daemonset, err := virtClient.AppsV1().DaemonSets(flags.KubeVirtInstallNamespace).Get(context.Background(), "virt-handler", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				if daemonset.Spec.Template.Spec.NodeSelector == nil || daemonset.Spec.Template.Spec.NodeSelector[labelKey] != labelValue {
					return false
				}
				return true
			}, 60*time.Second, 1*time.Second).Should(BeTrue())

			Expect(patchKVWorkloads(originalKv, nil)).To(Succeed())
		})

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
		It("should dynamically adjust virt- pod count and PDBs", func() {
			for _, replicas := range []uint8{3, 1, 2} {
				By(fmt.Sprintf("Setting the replica count in kvInfra to %d", replicas))
				var infra = &v1.ComponentConfig{
					Replicas: &replicas,
				}

				Expect(patchKVInfra(originalKv, infra)).To(Succeed())

				By(fmt.Sprintf("Expecting %d replicas of virt-api and virt-controller", replicas))
				Eventually(func() []k8sv1.Pod {
					for _, name := range []string{"virt-api", "virt-controller"} {
						pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
						Expect(err).ToNot(HaveOccurred())
						return pods.Items
					}
					return nil
				}, 120*time.Second, 1*time.Second).Should(HaveLen(int(replicas)), fmt.Sprintf("Replicas of virt-api and virt-controller are not %d", replicas))

				if replicas == 1 {
					By(fmt.Sprintf("Expecting PDBs to disppear"))
					Eventually(func() bool {
						for _, name := range []string{"virt-api", "virt-controller"} {
							_, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							if err == nil {
								return false
							}
						}
						return true
					}, 60*time.Second, 1*time.Second).Should(BeTrue(), "PDBs have not disappeared")
				} else {
					By(fmt.Sprintf("Expecting minAvailable to become %d on the PDBs", replicas-1))
					Eventually(func() int {
						for _, name := range []string{"virt-api", "virt-controller"} {
							pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(flags.KubeVirtInstallNamespace).Get(context.Background(), name+"-pdb", metav1.GetOptions{})
							Expect(err).ToNot(HaveOccurred())
							return pdb.Spec.MinAvailable.IntValue()
						}
						return -1
					}, 60*time.Second, 1*time.Second).Should(BeEquivalentTo(int(replicas-1)), fmt.Sprintf("minAvailable is not become %d on the PDBs", replicas-1))
				}
			}
		})
		It("should update new single-replica CRs with a finalizer and be stable", func() {
			By("copying the original kv CR")
			kv := copyOriginalKv(originalKv)
			kvOrigInfra := kv.Spec.Infra.DeepCopy()

			By("storing the actual replica counts for the cluster")
			originalReplicaCounts := make(map[string]int)
			for _, name := range []string{"virt-api", "virt-controller"} {
				pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
				Expect(err).ToNot(HaveOccurred())
				originalReplicaCounts[name] = len(pods.Items)
			}

			By("deleting the kv CR")
			err = virtClient.KubeVirt(kv.Namespace).Delete(context.Background(), kv.Name, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for virt-api and virt-controller to be gone")
			Eventually(func() bool {
				for _, name := range []string{"virt-api", "virt-controller"} {
					pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
					Expect(err).ToNot(HaveOccurred())
					if len(pods.Items) != 0 {
						return false
					}
				}
				return true
			}, 120*time.Second, 4*time.Second).Should(BeTrue())

			By("waiting for the kv CR to be gone")
			Eventually(func() error {
				_, err := virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
				return err
			}, 120*time.Second, 4*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

			By("creating a new single-replica kv CR")
			if kv.Spec.Infra == nil {
				kv.Spec.Infra = &v1.ComponentConfig{}
			}
			var one uint8 = 1
			kv.Spec.Infra.Replicas = &one
			kv, err = virtClient.KubeVirt(kv.Namespace).Create(context.Background(), kv, metav1.CreateOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the kv CR to get a finalizer")
			Eventually(func() bool {
				kv, err = virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return len(kv.Finalizers) > 0
			}, 120*time.Second, 4*time.Second).Should(BeTrue())

			By("ensuring the CR generation is stable")
			Expect(err).ToNot(HaveOccurred())
			Consistently(func() int64 {
				kv2, err := virtClient.KubeVirt(kv.Namespace).Get(context.Background(), kv.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return kv2.GetGeneration()
			}, 30*time.Second, 2*time.Second).Should(Equal(kv.GetGeneration()))

			By("restoring the original replica count")
			Expect(patchKVInfra(originalKv, kvOrigInfra)).To(Succeed())

			By("waiting for virt-api and virt-controller replicas to respawn")
			Eventually(func() error {
				for _, name := range []string{"virt-api", "virt-controller"} {
					pods, err := virtClient.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(context.Background(), metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", v1.AppLabel, name)})
					Expect(err).ToNot(HaveOccurred())
					if len(pods.Items) != originalReplicaCounts[name] {
						return fmt.Errorf("expected %d replicas for %s, got %d", originalReplicaCounts[name], name, len(pods.Items))
					}
				}
				return nil
			}, 120*time.Second, 4*time.Second).ShouldNot(HaveOccurred())
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

		It("[test_id:6257]should accept valid cert rotation parameters", func() {
			kv := copyOriginalKv(originalKv)
			certRotationStrategy := v1.KubeVirtCertificateRotateStrategy{
				SelfSigned: certConfig,
			}

			By(fmt.Sprintf("update certificateRotateStrategy"))
			patches := patch.New(patch.WithReplace("/spec/certificateRotateStrategy", certRotationStrategy))
			patchKV(kv.Name, patches)
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

	Context("with ContainerPathVolumes feature gate toggled", func() {

		AfterEach(func() {
			kvconfig.EnableFeatureGate(featuregate.ContainerPathVolumesGate)
		})

		It("should delete and recreate virt-launcher-pod-mutator webhook", func() {
			By("Ensuring ContainerPathVolumes feature gate is enabled")
			kvconfig.EnableFeatureGate(featuregate.ContainerPathVolumesGate)

			By("Verifying virt-launcher-pod-mutator webhook exists")
			Eventually(func() error {
				_, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), components.VirtLauncherPodMutatingWebhookName, metav1.GetOptions{})
				return err
			}, time.Minute, time.Second*2).Should(Succeed(), "webhook should exist when feature gate is enabled")

			By("Disabling ContainerPathVolumes feature gate")
			kvconfig.DisableFeatureGate(featuregate.ContainerPathVolumesGate)

			By("Verifying virt-launcher-pod-mutator webhook is deleted")
			Eventually(func() error {
				_, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), components.VirtLauncherPodMutatingWebhookName, metav1.GetOptions{})
				return err
			}, time.Minute*5, time.Second*2).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"), "webhook should be deleted when feature gate is disabled")

			By("Re-enabling ContainerPathVolumes feature gate")
			kvconfig.EnableFeatureGate(featuregate.ContainerPathVolumesGate)

			By("Verifying virt-launcher-pod-mutator webhook is recreated")
			Eventually(func() error {
				_, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), components.VirtLauncherPodMutatingWebhookName, metav1.GetOptions{})
				return err
			}, time.Minute, time.Second*2).Should(Succeed(), "webhook should be recreated when feature gate is re-enabled")
		})
	})

	Context(" Seccomp configuration", Serial, func() {

		Context("Kubevirt profile", func() {
			var nodeName string

			const expectedSeccompProfilePath = "/proc/1/root/var/lib/kubelet/seccomp/kubevirt/kubevirt.json"

			enableSeccompFeature := func() {
				//Disable feature first to simulate addition
				kvconfig.DisableFeatureGate(featuregate.KubevirtSeccompProfile)
				kvconfig.EnableFeatureGate(featuregate.KubevirtSeccompProfile)
			}

			disableSeccompFeature := func() {
				//Enable feature first to simulate removal
				kvconfig.EnableFeatureGate(featuregate.KubevirtSeccompProfile)
				kvconfig.DisableFeatureGate(featuregate.KubevirtSeccompProfile)
			}

			enableKubevirtProfile := func(enable bool) {
				nodeName = libnode.GetAllSchedulableNodes(virtClient).Items[0].Name

				By("Removing profile if present")
				_, err := libnode.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/rm", "-f", expectedSeccompProfilePath})
				Expect(err).NotTo(HaveOccurred())

				By(fmt.Sprintf("Configuring KubevirtSeccompProfile feature gate to %t", enable))
				if enable {
					enableSeccompFeature()
				} else {
					disableSeccompFeature()
				}

				vmProfile := &v1.VirtualMachineInstanceProfile{
					CustomProfile: &v1.CustomProfile{
						LocalhostProfile: pointer.P("kubevirt/kubevirt.json"),
					},
				}
				if !enable {
					vmProfile = nil

				}

				kv := libkubevirt.GetCurrentKv(virtClient)
				kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{
					VirtualMachineInstanceProfile: vmProfile,
				}

				kvconfig.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
			}

			It("should install Kubevirt policy", func() {
				enableKubevirtProfile(true)

				By("Expecting to see the profile")
				Eventually(func() error {
					_, err = libnode.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/cat", expectedSeccompProfilePath})
					return err
				}, 1*time.Minute, 1*time.Second).Should(Not(HaveOccurred()))
			})

			It("should not install Kubevirt policy", func() {
				enableKubevirtProfile(false)

				By("Expecting to not see the profile")
				Consistently(func() error {
					_, err = libnode.ExecuteCommandInVirtHandlerPod(nodeName, []string{"/usr/bin/cat", expectedSeccompProfilePath})
					return err
				}, 1*time.Minute, 1*time.Second).Should(MatchError(Or(ContainSubstring("No such file"), ContainSubstring("container not found"))))
				Expect(err).To(MatchError(ContainSubstring("No such file")))
			})
		})

		Context("VirtualMachineInstance Profile", func() {
			DescribeTable("with VirtualMachineInstance Profile set to", func(virtualMachineProfile *v1.VirtualMachineInstanceProfile, expectedProfile *k8sv1.SeccompProfile) {
				By("Configuring VirtualMachineInstance Profile")
				kv := libkubevirt.GetCurrentKv(virtClient)
				if kv.Spec.Configuration.SeccompConfiguration == nil {
					kv.Spec.Configuration.SeccompConfiguration = &v1.SeccompConfiguration{}
				}
				kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile = virtualMachineProfile
				kvconfig.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

				By("Checking launcher seccomp policy")
				vmi, err := virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), libvmifact.NewGuestless(), metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				fetchVMI := matcher.ThisVMI(vmi)
				psaRelatedErrorDetected := false
				err = virtwait.PollImmediately(time.Second, 30*time.Second, func(_ context.Context) (done bool, err error) {
					vmi, err := fetchVMI()
					if err != nil {
						return done, err
					}

					if vmi.Status.Phase != v1.Pending {
						return true, nil
					}

					for _, condition := range vmi.Status.Conditions {
						if condition.Type == v1.VirtualMachineInstanceSynchronized {
							if condition.Status == k8sv1.ConditionFalse && strings.Contains(condition.Message, "needs a privileged namespace") {
								psaRelatedErrorDetected = true
								return true, nil
							}
						}
					}
					return
				})
				Expect(err).NotTo(HaveOccurred())
				// In case we are running on PSA cluster, the case were we don't specify seccomp will violate the policy.
				// Therefore the VMIs Pod will fail to be created and we can't check its configuration.
				// In that case the loop above needs to see that VMI contains PSA related error.
				// This is enough and we declare this test as passed.
				if psaRelatedErrorDetected && virtualMachineProfile == nil {
					return
				}
				Eventually(matcher.ThisVMI(vmi), 30*time.Second, time.Second).Should(BeInPhase(v1.Scheduled))

				pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
				Expect(err).NotTo(HaveOccurred())
				var podProfile *k8sv1.SeccompProfile
				if pod.Spec.SecurityContext != nil {
					podProfile = pod.Spec.SecurityContext.SeccompProfile
				}

				Expect(podProfile).To(Equal(expectedProfile))
			},
				Entry("default should not set profile", nil, nil),
				Entry("custom should use localhost", &v1.VirtualMachineInstanceProfile{
					CustomProfile: &v1.CustomProfile{
						LocalhostProfile: pointer.P("kubevirt/kubevirt.json"),
					},
				},
					&k8sv1.SeccompProfile{Type: k8sv1.SeccompProfileTypeLocalhost, LocalhostProfile: pointer.P("kubevirt/kubevirt.json")}),
			)
		})
	})

	Context(" Deployment of common-instancetypes", decorators.SigComputeInstancetype, Serial, func() {
		var (
			originalConfig *v1.CommonInstancetypesDeployment
			appComponent   string
			labelSelector  string
		)

		BeforeEach(func() {
			kv := libkubevirt.GetCurrentKv(virtClient)
			originalConfig = kv.Spec.Configuration.CommonInstancetypesDeployment.DeepCopy()

			// Do nothing if the deployment is already default
			if originalConfig != nil {
				defaultDeployment()
			}

			appComponent = apply.GetAppComponent(libkubevirt.GetCurrentKv(virtClient))
			labelSelector = labels.Set{
				v1.AppComponentLabel: appComponent,
				v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
			}.String()
		})

		AfterEach(func() {
			// Do nothing if the current config already matches the original
			kv := libkubevirt.GetCurrentKv(virtClient)
			if reflect.DeepEqual(originalConfig, kv.Spec.Configuration.CommonInstancetypesDeployment) {
				return
			}

			kv.Spec.Configuration.CommonInstancetypesDeployment = originalConfig.DeepCopy()
			updateConfigAndWait(kv.Spec.Configuration)
		})

		It("Should deploy common-instancetypes according to KubeVirt configurable", func() {
			// Default is to deploy the resources
			expectResourcesToExist(labelSelector)

			disableDeployment()
			expectResourcesToNotExist(labelSelector)

			enableDeployment()
			expectResourcesToExist(labelSelector)

			disableDeployment()
			expectResourcesToNotExist(labelSelector)
		})

		Context("Should take ownership", func() {
			const (
				appComponentChanged = "something"
				managedByChanged    = "someone"
			)

			It("of instancetypes and preferences", func() {
				By("Ensuring deployment is disabled")
				disableDeployment()

				By("Getting instancetypes to be deployed by virt-operator")
				instancetypes, err := components.NewClusterInstancetypes()
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetypes).ToNot(BeEmpty())

				By("Picking the first instancetype and changing its labels")
				instancetype := instancetypes[0]
				instancetype.Labels = map[string]string{
					v1.AppComponentLabel: appComponentChanged,
					v1.ManagedByLabel:    managedByChanged,
				}

				By("Creating the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponentChanged))
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, managedByChanged))

				By("Getting preferences to be deployed by virt-operator")
				preferences, err := components.NewClusterPreferences()
				Expect(err).ToNot(HaveOccurred())
				Expect(preferences).ToNot(BeEmpty())

				By("Picking the first preference and changing its labels")
				preference := preferences[0]
				preference.Labels = map[string]string{
					v1.AppComponentLabel: appComponentChanged,
					v1.ManagedByLabel:    managedByChanged,
				}

				By("Creating the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponentChanged))
				Expect(preference.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, managedByChanged))

				By("Enabling deployment and waiting for KubeVirt to be ready")
				enableDeployment()

				By("Verifying virt-operator took ownership of the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponent))
				Expect(instancetype.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))

				By("Verifying virt-operator took ownership of the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Labels).To(HaveKeyWithValue(v1.AppComponentLabel, appComponent))
				Expect(preference.Labels).To(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
			})
		})

		Context("Should delete resources not in install strategy", func() {
			It("with instancetypes and preferences", func() {
				instancetype := &v1beta1.VirtualMachineClusterInstancetype{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "clusterinstancetype-",
						Labels: map[string]string{
							v1.AppComponentLabel: appComponent,
							v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
						},
					},
				}

				By("Creating the instancetype")
				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Create(context.Background(), instancetype, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying virt-operator deleted the instancetype")
				Eventually(func(g Gomega) {
					_, err = virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
					g.Expect(err).Should(HaveOccurred())
					g.Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())

				preference := &v1beta1.VirtualMachineClusterPreference{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "clusterpreference-",
						Labels: map[string]string{
							v1.AppComponentLabel: appComponent,
							v1.ManagedByLabel:    v1.ManagedByLabelOperatorValue,
						},
					},
				}

				By("Creating the preference")
				preference, err = virtClient.VirtualMachineClusterPreference().Create(context.Background(), preference, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Verifying virt-operator deleted the preference")
				Eventually(func(g Gomega) {
					_, err = virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
					g.Expect(err).Should(HaveOccurred())
					g.Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonNotFound))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})
		})

		Context("Should revert changes", func() {
			const (
				keyTest     = "test"
				valModified = "modified"
				cpu         = uint32(1024)
			)

			var preferredTopology = v1beta1.Threads

			It("to instancetypes and preferences", func() {
				By("Getting the deployed instancetypes")
				instancetypes, err := virtClient.VirtualMachineClusterInstancetype().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetypes.Items).ToNot(BeEmpty())

				By("Modifying an instancetype")
				originalInstancetype := instancetypes.Items[0].DeepCopy()
				instancetype := originalInstancetype.DeepCopy()
				instancetype.Annotations[keyTest] = valModified
				instancetype.Labels[keyTest] = valModified
				instancetype.Spec = v1beta1.VirtualMachineInstancetypeSpec{
					CPU: v1beta1.CPUInstancetype{
						Guest: cpu,
					},
				}

				instancetype, err = virtClient.VirtualMachineClusterInstancetype().Update(context.Background(), instancetype, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(instancetype.Annotations).To(HaveKeyWithValue(keyTest, valModified))
				Expect(instancetype.Labels).To(HaveKeyWithValue(keyTest, valModified))
				Expect(instancetype.Spec.CPU.Guest).To(Equal(cpu))

				By("Verifying virt-operator reverts the changes")
				Eventually(func(g Gomega) {
					instancetype, err := virtClient.VirtualMachineClusterInstancetype().Get(context.Background(), instancetype.Name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(instancetype.Annotations).To(Equal(originalInstancetype.Annotations))
					g.Expect(instancetype.Labels).To(Equal(originalInstancetype.Labels))
					g.Expect(instancetype.Spec).To(Equal(originalInstancetype.Spec))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())

				By("Getting the deployed preferencess")
				preferences, err := virtClient.VirtualMachineClusterPreference().List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred())
				Expect(preferences.Items).ToNot(BeEmpty())

				By("Modifying a preference")
				originalPreference := preferences.Items[0].DeepCopy()
				preference := originalPreference.DeepCopy()
				preference.Annotations[keyTest] = valModified
				preference.Labels[keyTest] = valModified
				preference.Spec = v1beta1.VirtualMachinePreferenceSpec{
					CPU: &v1beta1.CPUPreferences{
						PreferredCPUTopology: &preferredTopology,
					},
				}

				preference, err = virtClient.VirtualMachineClusterPreference().Update(context.Background(), preference, metav1.UpdateOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(preference.Annotations).To(HaveKeyWithValue(keyTest, valModified))
				Expect(preference.Labels).To(HaveKeyWithValue(keyTest, valModified))
				Expect(preference.Spec.CPU).ToNot(BeNil())
				Expect(preference.Spec.CPU.PreferredCPUTopology).ToNot(BeNil())
				Expect(*preference.Spec.CPU.PreferredCPUTopology).To(Equal(preferredTopology))

				By("Verifying virt-operator reverts the changes")
				Eventually(func(g Gomega) {
					preference, err := virtClient.VirtualMachineClusterPreference().Get(context.Background(), preference.Name, metav1.GetOptions{})
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(preference.Annotations).To(Equal(originalPreference.Annotations))
					g.Expect(preference.Labels).To(Equal(originalPreference.Labels))
					g.Expect(preference.Spec).To(Equal(originalPreference.Spec))
				}, 1*time.Minute, 5*time.Second).Should(Succeed())
			})
		})
	})

	Context("virt-template deployment", func() {
		setVirtTemplateDeploymentEnabled := func(enabled bool) {
			kv := libkubevirt.GetCurrentKv(kubevirt.Client())
			kv.Spec.Configuration.VirtTemplateDeployment = &v1.VirtTemplateDeployment{
				Enabled: &enabled,
			}
			kvconfig.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)
		}

		// Note: virt-template requires the Snapshot feature gate for full functionality,
		// but these tests only verify deployment/removal behavior.
		DescribeTable("should deploy and remove virt-template", func(setup func(), enable func(), disable func()) {
			if setup != nil {
				setup()
			}

			By("Ensuring virt-template deployments do not exist initially")
			eventuallyVirtTemplateDeploymentsNotFound()

			By("Enabling virt-template deployment")
			enable()

			By("Verifying virt-template deployments are created")
			sanityCheckVirtTemplateDeploymentsExist()

			By("Disabling virt-template deployment")
			disable()

			By("Verifying virt-template deployments are removed")
			eventuallyVirtTemplateDeploymentsNotFound()
		},
			Entry("when feature gate is toggled",
				nil,
				func() { kvconfig.EnableFeatureGate(featuregate.Template) },
				func() { kvconfig.DisableFeatureGate(featuregate.Template) },
			),
			Entry("when VirtTemplateDeployment.Enabled is toggled",
				func() {
					setVirtTemplateDeploymentEnabled(false)
					kvconfig.EnableFeatureGate(featuregate.Template)
				},
				func() { setVirtTemplateDeploymentEnabled(true) },
				func() { setVirtTemplateDeploymentEnabled(false) },
			),
		)
	})

	Context("external CA", func() {
		createCrt := func(duration time.Duration) *tls.Certificate {
			caKeyPair, _ := triple.NewCA("test.kubevirt.io", duration)

			encodedCert := cert.EncodeCertPEM(caKeyPair.Cert)
			encodedKey := cert.EncodePrivateKeyPEM(caKeyPair.Key)

			crt, err := tls.X509KeyPair(encodedCert, encodedKey)
			Expect(err).ToNot(HaveOccurred())
			leaf, err := cert.ParseCertsPEM(encodedCert)
			Expect(err).ToNot(HaveOccurred())
			crt.Leaf = leaf[0]

			return &crt
		}

		It("should create a blank configmap", func() {
			Eventually(func(g Gomega) {
				cm, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.ExternalKubeVirtCAConfigMapName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(cm.Data).To(HaveKeyWithValue(components.CABundleKey, ""))
			}, 1*time.Minute, 5*time.Second).Should(Succeed())
		})

		It("should properly manage adding entries to the configmap", func() {
			cert1 := createCrt(time.Hour)
			cert2 := createCrt(time.Hour)
			cert3 := createCrt(time.Millisecond)
			cert1Encoded := cert.EncodeCertPEM(cert1.Leaf)
			cert2Encoded := cert.EncodeCertPEM(cert2.Leaf)
			cert3Encoded := cert.EncodeCertPEM(cert3.Leaf)
			// Sleep a bit to ensure the third cert is expired
			time.Sleep(10 * time.Millisecond)
			now := time.Now()
			By("ensure cert1 is valid")
			Expect(cert1.Leaf.NotBefore).To(BeTemporally("<", now))
			Expect(cert1.Leaf.NotAfter).To(BeTemporally(">", now))
			By("ensure cert2 is valid")
			Expect(cert2.Leaf.NotBefore).To(BeTemporally("<", now))
			Expect(cert2.Leaf.NotAfter).To(BeTemporally(">", now))
			By("ensure cert3 is expired")
			Expect(cert3.Leaf.NotBefore).To(BeTemporally("<", now))
			Expect(cert3.Leaf.NotAfter).To(BeTemporally("<", now))

			By("Adding the first cert")
			p, err := patch.New(patch.WithReplace("/data/"+components.CABundleKey, string(cert1Encoded))).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			configMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).
				Patch(context.Background(), components.ExternalKubeVirtCAConfigMapName, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(configMap.Data).To(HaveKeyWithValue(components.CABundleKey, string(cert1Encoded)))

			Eventually(func(g Gomega) {
				cm, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				val, ok := cm.Data[components.CABundleKey]
				g.Expect(ok).To(BeTrue())
				g.Expect(val).To(ContainSubstring(string(cert1Encoded)))
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Adding an invalid string to the configmap, should be ignored and removed from the external CA configmap")
			p, err = patch.New(patch.WithReplace("/data/"+components.CABundleKey, "invalid")).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			configMap, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).
				Patch(context.Background(), components.ExternalKubeVirtCAConfigMapName, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(configMap.Data).To(HaveKeyWithValue(components.CABundleKey, "invalid"))

			Eventually(func(g Gomega) {
				cm, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				val, ok := cm.Data[components.CABundleKey]
				g.Expect(ok).To(BeTrue())
				g.Expect(val).To(ContainSubstring(string(cert1Encoded)))
				g.Expect(val).ToNot(ContainSubstring("invalid"))
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Adding the second cert")
			p, err = patch.New(patch.WithReplace("/data/"+components.CABundleKey, string(cert2Encoded))).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			configMap, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).
				Patch(context.Background(), components.ExternalKubeVirtCAConfigMapName, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(configMap.Data).To(HaveKeyWithValue(components.CABundleKey, string(cert2Encoded)))

			Eventually(func(g Gomega) {
				kubevirtCAConfigMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				val, ok := kubevirtCAConfigMap.Data[components.CABundleKey]
				g.Expect(ok).To(BeTrue())
				g.Expect(val).To(ContainSubstring(string(cert1Encoded)))
				g.Expect(val).ToNot(ContainSubstring("invalid"))
				g.Expect(val).To(ContainSubstring(string(cert2Encoded)))
			}, 10*time.Second, time.Second).Should(Succeed())

			By("Adding the third cert, which is expired, it should not be added to the kubevirt-ca configmap")
			p, err = patch.New(patch.WithReplace("/data/"+components.CABundleKey, string(cert3Encoded))).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			configMap, err = virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).
				Patch(context.Background(), components.ExternalKubeVirtCAConfigMapName, types.JSONPatchType, p, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(configMap.Data).To(HaveKeyWithValue(components.CABundleKey, string(cert3Encoded)))

			Eventually(func(g Gomega) {
				kubevitCAConfigMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
				g.Expect(err).ToNot(HaveOccurred())
				val, ok := kubevitCAConfigMap.Data[components.CABundleKey]
				g.Expect(ok).To(BeTrue())
				g.Expect(val).To(ContainSubstring(string(cert1Encoded)))
				g.Expect(val).ToNot(ContainSubstring("invalid"))
				g.Expect(val).To(ContainSubstring(string(cert2Encoded)))
				g.Expect(val).ToNot(ContainSubstring(string(cert3Encoded)))
			}, 10*time.Second, time.Second).Should(Succeed())
			kubevitCAConfigMap, err := virtClient.CoreV1().ConfigMaps(flags.KubeVirtInstallNamespace).Get(context.Background(), components.KubeVirtCASecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			certBytes := kubevitCAConfigMap.Data[components.CABundleKey]
			certs, err := cert.ParseCertsPEM([]byte(certBytes))
			Expect(err).ToNot(HaveOccurred())
			count := 0
			cert1Hash := sha256.Sum256(cert1.Leaf.Raw)
			cert1HashString := hex.EncodeToString(cert1Hash[:])
			cert2Hash := sha256.Sum256(cert2.Leaf.Raw)
			cert2HashString := hex.EncodeToString(cert2Hash[:])
			cert3Hash := sha256.Sum256(cert3.Leaf.Raw)
			cert3HashString := hex.EncodeToString(cert3Hash[:])
			for _, cert := range certs {
				certHash := sha256.Sum256(cert.Raw)
				certHashString := hex.EncodeToString(certHash[:])
				if certHashString == cert1HashString || certHashString == cert2HashString {
					count++
				}
				if certHashString == cert3HashString {
					Fail("cert3 should not be in the kubevirt-ca configmap")
				}
			}
			Expect(count).To(Equal(2))
		})
	})

})
