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
	"slices"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	aggregatorclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/controller"
	"kubevirt.io/kubevirt/pkg/libvmi"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-config/featuregate"
	"kubevirt.io/kubevirt/pkg/virt-controller/services"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/apply"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/pkg/virt-operator/util"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/flags"
	"kubevirt.io/kubevirt/tests/framework/checks"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"
	"kubevirt.io/kubevirt/tests/libhypervisor"
	"kubevirt.io/kubevirt/tests/libkubevirt"
	kvconfig "kubevirt.io/kubevirt/tests/libkubevirt/config"
	"kubevirt.io/kubevirt/tests/libnet"
	"kubevirt.io/kubevirt/tests/libnode"
	"kubevirt.io/kubevirt/tests/libpod"
	"kubevirt.io/kubevirt/tests/libvmifact"
	"kubevirt.io/kubevirt/tests/libwait"
	"kubevirt.io/kubevirt/tests/testsuite"
)

var _ = Describe("[sig-operator]Operator", Serial, decorators.SigOperator, func() {
	var (
		virtClient              kubecli.KubevirtClient
		aggregatorClient        *aggregatorclient.Clientset
		originalKv              *v1.KubeVirt
		originalOperatorVersion string
		err                     error
	)

	BeforeEach(func() {
		virtClient = kubevirt.Client()
		config, err := kubecli.GetKubevirtClientConfig()
		Expect(err).ToNot(HaveOccurred())
		aggregatorClient = aggregatorclient.NewForConfigOrDie(config)

		originalKv = libkubevirt.GetCurrentKv(virtClient)

		_, _, _, _, version := parseOperatorImage()
		const prefix = ":"
		Expect(strings.HasPrefix(version, prefix)).To(BeTrue(), fmt.Sprintf("version %s is expected to start with %s", version, prefix))
		originalOperatorVersion = strings.TrimPrefix(version, prefix)

		verifyOperatorWebhookCertificate()
	})

	AfterEach(func() {
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

	It("[test_id:1746]should have created and available condition", func() {
		kv := libkubevirt.GetCurrentKv(virtClient)

		By("verifying that created and available condition is present")
		testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)
	})

	Describe("should reconcile components", Serial, func() {

		deploymentName := "virt-controller"
		daemonSetName := "virt-handler"
		envVarDeploymentKeyToUpdate := "USER_ADDED_ENV"

		crdName := "virtualmachines.kubevirt.io"
		shortNameAdded := "new"

		DescribeTable("checking updating resource is reverted to original state for ", func(changeResource func(), getResource func() runtime.Object, compareResource func() bool) {
			resource := getResource()
			By("Updating KubeVirt Object")
			changeResource()

			var generation int64
			By("Test that the added envvar was removed")
			Eventually(func() bool {
				equal := compareResource()
				if equal {
					r := getResource()
					o := r.(metav1.Object)
					generation = o.GetGeneration()
				}

				return equal
			}, 120*time.Second, 5*time.Second).Should(BeTrue(), "waiting for deployment to revert to original state")

			Eventually(func() int64 {
				currentKV := libkubevirt.GetCurrentKv(virtClient)
				return apply.GetExpectedGeneration(resource, currentKV.Status.Generations)
			}, 60*time.Second, 5*time.Second).Should(Equal(generation), "reverted deployment generation should be set on KV resource")

			By("Test that the expected generation is unchanged")
			Consistently(func() int64 {
				currentKV := libkubevirt.GetCurrentKv(virtClient)
				return apply.GetExpectedGeneration(resource, currentKV.Status.Generations)
			}, 30*time.Second, 5*time.Second).Should(Equal(generation))
		},

			Entry("[test_id:6254] deployments",

				func() {
					patchBytes, err := patch.New(
						patch.WithAdd(
							"/spec/template/spec/containers/0/env/-",
							k8sv1.EnvVar{
								Name:  envVarDeploymentKeyToUpdate,
								Value: "value",
							}),
					).GeneratePayload()
					Expect(err).ToNot(HaveOccurred())

					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Patch(context.Background(), deploymentName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vc.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{
						Name:  envVarDeploymentKeyToUpdate,
						Value: "value",
					}))
				},

				func() runtime.Object {
					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vc
				},

				func() bool {
					vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), deploymentName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return !slices.ContainsFunc(vc.Spec.Template.Spec.Containers[0].Env, func(env k8sv1.EnvVar) bool {
						return env.Name == envVarDeploymentKeyToUpdate
					})
				}),

			Entry("[test_id:6255] customresourcedefinitions",

				func() {
					patchBytes, err := patch.New(
						patch.WithAdd("/spec/names/shortNames/-", shortNameAdded),
					).GeneratePayload()
					Expect(err).ToNot(HaveOccurred())

					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), crdName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vmcrd.Spec.Names.ShortNames).To(ContainElement(shortNameAdded))
				},

				func() runtime.Object {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return vmcrd
				},

				func() bool {
					vmcrd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return !slices.Contains(vmcrd.Spec.Names.ShortNames, shortNameAdded)
				}),
			Entry("[test_id:6256] poddisruptionbudgets", decorators.MultiReplica,
				func() {
					patchBytes, err := patch.New(
						patch.WithAdd("/spec/selector/matchLabels",
							map[string]string{
								"kubevirt.io": "dne",
							}),
					).GeneratePayload()
					Expect(err).ToNot(HaveOccurred())

					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Patch(context.Background(), "virt-controller-pdb", types.JSONPatchType, patchBytes, metav1.PatchOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(pdb.Spec.Selector.MatchLabels["kubevirt.io"]).To(Equal("dne"))
				},

				func() runtime.Object {
					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())
					return pdb
				},

				func() bool {
					pdb, err := virtClient.PolicyV1().PodDisruptionBudgets(originalKv.Namespace).Get(context.Background(), "virt-controller-pdb", metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return pdb.Spec.Selector.MatchLabels["kubevirt.io"] != "dne"
				}),
			Entry("[test_id:6308] daemonsets",
				func() {
					patchBytes, err := patch.New(
						patch.WithAdd(
							"/spec/template/spec/containers/0/env/-",
							k8sv1.EnvVar{
								Name:  envVarDeploymentKeyToUpdate,
								Value: "value",
							}),
					).GeneratePayload()

					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Patch(context.Background(), daemonSetName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
					Expect(err).ToNot(HaveOccurred())
					Expect(vc.Spec.Template.Spec.Containers[0].Env).To(ContainElement(k8sv1.EnvVar{
						Name:  envVarDeploymentKeyToUpdate,
						Value: "value",
					}))
				},

				func() runtime.Object {
					var ds *appsv1.DaemonSet

					// wait for virt-handler readiness
					Eventually(func() bool {
						var err error
						ds, err = virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
						Expect(err).ToNot(HaveOccurred())
						return ds.Status.DesiredNumberScheduled == ds.Status.NumberReady && ds.Spec.UpdateStrategy.RollingUpdate.MaxUnavailable.IntValue() == 1
					}, 60*time.Second, 1*time.Second).Should(BeTrue(), "waiting for daemonSet to be ready")
					return ds
				},

				func() bool {
					vc, err := virtClient.AppsV1().DaemonSets(originalKv.Namespace).Get(context.Background(), daemonSetName, metav1.GetOptions{})
					Expect(err).ToNot(HaveOccurred())

					return !slices.ContainsFunc(vc.Spec.Template.Spec.Containers[0].Env, func(env k8sv1.EnvVar) bool {
						return env.Name == envVarDeploymentKeyToUpdate
					})
				}),
		)

		It("[test_id:6309] checking updating service is reverted to original state", func() {
			service, err := virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())

			originalPort := service.Spec.Ports[0].Port
			service.Spec.Ports[0].Port = 123

			By("Update service with undesired port")
			service, err = virtClient.CoreV1().Services(originalKv.Namespace).Update(context.Background(), service, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(service.Spec.Ports[0].Port).To(Equal(int32(123)))

			By("Test that the port is reverted to the original")
			Eventually(func() int32 {
				service, err := virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return service.Spec.Ports[0].Port
			}).WithTimeout(120*time.Second).WithPolling(5*time.Second).Should(Equal(originalPort), "waiting for service to revert to original state")

			By("Test that the revert of the service stays consistent")
			Consistently(func() int32 {
				service, err = virtClient.CoreV1().Services(originalKv.Namespace).Get(context.Background(), "virt-api", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())

				return service.Spec.Ports[0].Port
			}).WithTimeout(20 * time.Second).WithPolling(5 * time.Second).Should(Equal(originalPort))
		})
	})

	Describe("[test_id:6987]should apply component configuration", func() {

		It("test VirtualMachineInstancesPerNode", func() {
			newVirtualMachineInstancesPerNode := 10

			By("Patching KubeVirt Object")
			kv, err := virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.Spec.Configuration.VirtualMachineInstancesPerNode).ToNot(Equal(&newVirtualMachineInstancesPerNode))

			hypervisorDevice := libhypervisor.GetHypervisorDeviceName(virtClient)
			hypervisorResource := k8sv1.ResourceName(services.K8sDevicePrefix + "/" + hypervisorDevice)

			newVMIPerNodePatch, err := patch.New(
				patch.WithAdd("/spec/configuration/virtualMachineInstancesPerNode", newVirtualMachineInstancesPerNode)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kv.Name, types.JSONPatchType, newVMIPerNodePatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 120*time.Second)

			By("Test that worker nodes have the correct allocatable hypervisor devices according to virtualMachineInstancesPerNode setting")
			Eventually(func() error {
				nodesWithHypervisor := libnode.GetNodesWithHypervisor(hypervisorDevice)
				for _, node := range nodesWithHypervisor {
					hypervisorDevices, _ := node.Status.Allocatable[hypervisorResource]
					if int(hypervisorDevices.Value()) != newVirtualMachineInstancesPerNode {
						return fmt.Errorf("node %s does not have the expected allocatable hypervisor %s devices: %d, got: %d", node.Name, hypervisorDevice, newVirtualMachineInstancesPerNode, hypervisorDevices.Value())
					}
				}
				return nil
			}, 60*time.Second, 5*time.Second).ShouldNot(HaveOccurred())

			By("Deleting patch from KubeVirt object")
			newVMIPerNodeRemovePatch, err := patch.New(
				patch.WithRemove("/spec/configuration/virtualMachineInstancesPerNode")).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())

			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), kv.Name, types.JSONPatchType, newVMIPerNodeRemovePatch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 120*time.Second)

			By("Check that worker nodes resumed the default amount of allocatable hypervisor devices")
			const defaultHypervisorDevices = "1k"
			defaultHypervisorDevicesQuant := resource.MustParse(defaultHypervisorDevices)

			Eventually(func(g Gomega) {
				nodesWithHypervisor := libnode.GetNodesWithHypervisor(hypervisorDevice)
				for _, node := range nodesWithHypervisor {
					g.Expect(node.Status.Allocatable).To(HaveKeyWithValue(hypervisorResource, defaultHypervisorDevicesQuant), "node %s does not have the expected allocatable hypervisor %s devices", node.Name, hypervisorDevice)
				}
			}).WithTimeout(60 * time.Second).WithPolling(5 * time.Second).Should(Succeed())
		})
	})

	Describe("[test_id:4744]should apply component customization", Serial, func() {

		It("test applying and removing a patch", func() {
			annotationPatchValue := "new-annotation-value"
			annotationPatchKey := "applied-patch"

			By("Patching KubeVirt Object")

			ccs := v1.CustomizeComponents{
				Patches: []v1.CustomizeComponentsPatch{
					{
						ResourceName: "virt-controller",
						ResourceType: "Deployment",
						Patch:        fmt.Sprintf(`{"spec":{"template": {"metadata": { "annotations": {"%s":"%s"}}}}}`, annotationPatchKey, annotationPatchValue),
						Type:         v1.StrategicMergePatchType,
					},
				},
			}

			patchPayload, err := patch.New(patch.WithReplace("/spec/customizeComponents", ccs)).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			generation := kv.GetGeneration()

			By("waiting for operator to patch the virt-controller component")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vc.Annotations[v1.KubeVirtGenerationAnnotation]
			}, 90*time.Second, 5*time.Second).Should(Equal(strconv.FormatInt(generation, 10)),
				"Resource generation numbers should be identical on both the Kubevirt CR and the virt-controller resource")

			By("Test that patch was applied to deployment")
			vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]).To(Equal(annotationPatchValue))

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 120*time.Second)

			By("Check that KubeVirt CR generation does not get updated when applying patch")
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv.GetGeneration()).To(Equal(generation))

			By("Deleting patch from KubeVirt object")

			patchPayload, err = patch.New(patch.WithRemove("/spec/customizeComponents")).GeneratePayload()
			Expect(err).ToNot(HaveOccurred())
			kv, err = virtClient.KubeVirt(flags.KubeVirtInstallNamespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchPayload, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			generation = kv.GetGeneration()

			By("waiting for operator to patch the virt-controller component")
			Eventually(func() string {
				vc, err := virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return vc.Annotations[v1.KubeVirtGenerationAnnotation]
			}, 90*time.Second, 5*time.Second).Should(Equal(strconv.FormatInt(generation, 10)),
				"Resource generation numbers should be identical on both the Kubevirt CR and the virt-controller resource")

			By("Test that patch was removed from deployment")
			vc, err = virtClient.AppsV1().Deployments(originalKv.Namespace).Get(context.Background(), "virt-controller", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(vc.Spec.Template.ObjectMeta.Annotations[annotationPatchKey]).To(BeEmpty())

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 120*time.Second)
		})
	})

	Describe("imagePullSecrets", func() {
		It("should not be present if not specified on the KubeVirt CR", func() {

			By("Check that KubeVirt CR has empty imagePullSecrets")
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(kv.Spec.ImagePullSecrets).To(BeEmpty())

			By("Ensuring that all virt components have empty image pull secrets")
			checkVirtComponents(originalKv.Namespace, nil)

			By("Starting a VMI")
			vmi := libvmi.New(libvmi.WithMemoryRequest("1Mi"))
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring that virt-launcher pod does not have additional image pull secrets")
			checkVirtLauncherPod(vmi)

		})

		It("should be propagated if applied on the KubeVirt CR", func() {

			const imagePullSecretName = "testmyregistrykey"
			var imagePullSecrets = []k8sv1.LocalObjectReference{{Name: imagePullSecretName}}

			By("Delete existing image pull secret")
			_ = virtClient.CoreV1().Secrets(originalKv.Namespace).Delete(context.Background(), imagePullSecretName, metav1.DeleteOptions{})
			By("Create image pull secret")
			_, err := virtClient.CoreV1().Secrets(originalKv.Namespace).Create(context.Background(), &k8sv1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      imagePullSecretName,
					Namespace: originalKv.Namespace,
				},
				Type: k8sv1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					".dockerconfigjson": []byte(`{"auths":{"http://foo.example.com":{"username":"foo","password":"bar","email":"foo@example.com"}}}`),
				},
			}, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			DeferCleanup(func() {
				By("Cleaning up image pull secret")
				err = virtClient.CoreV1().Secrets(originalKv.Namespace).Delete(context.Background(), imagePullSecretName, metav1.DeleteOptions{})
				Expect(err).NotTo(HaveOccurred())
			})

			By("Updating KubeVirt Object")
			kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			kv.Spec.ImagePullSecrets = imagePullSecrets
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Ensuring that all virt components have expected image pull secrets")
			checkVirtComponents(originalKv.Namespace, imagePullSecrets)

			By("Starting a VMI")
			vmi := libvmi.New(libvmi.WithMemoryRequest("1Mi"))
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).NotTo(HaveOccurred())
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Ensuring that virt-launcher pod does not have additional image pull secrets")
			checkVirtLauncherPod(vmi)

			By("Deleting imagePullSecrets from KubeVirt object")
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			kv.Spec.ImagePullSecrets = []k8sv1.LocalObjectReference{}
			kv, err = virtClient.KubeVirt(originalKv.Namespace).Update(context.Background(), kv, metav1.UpdateOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for virt-operator to apply changes to component")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Ensuring that all virt components have empty image pull secrets")
			checkVirtComponents(originalKv.Namespace, nil)

		})

	})

	Describe("[rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]infrastructure management", func() {
		It("[test_id:3146]should be able to delete and re-create kubevirt install", decorators.Upgrade, func() {
			allKvInfraPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			// This ensures that we can remove kubevirt while workloads are running
			By("Starting some vmis")
			var vmis []*v1.VirtualMachineInstance
			if checks.HasAtLeastTwoNodes() {
				vmis, err = generateMigratableVMIs(2)
				Expect(err).ToNot(HaveOccurred())

				netAttachDef := libnet.NewBridgeNetAttachDef(secondaryNetworkName, secondaryNetworkName)
				_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(vmis[0]), netAttachDef)
				Expect(err).NotTo(HaveOccurred())
				createRunningVMIs(vmis)
			}

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false, originalKv.Name)

			// this is just verifying some common known components do in fact get deleted.
			By("Sanity Checking Deployments infrastructure is deleted")
			eventuallyDeploymentNotFound(virtApiDepName)
			eventuallyDeploymentNotFound(virtControllerDepName)

			By("ensuring that namespaces can be successfully created and deleted")
			_, err := virtClient.CoreV1().Namespaces().Create(context.Background(), &k8sv1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testsuite.NamespaceTestOperator}}, metav1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				Expect(err).ToNot(HaveOccurred())
			}
			err = virtClient.CoreV1().Namespaces().Delete(context.Background(), testsuite.NamespaceTestOperator, metav1.DeleteOptions{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() error {
				_, err := virtClient.CoreV1().Namespaces().Get(context.Background(), testsuite.NamespaceTestOperator, metav1.GetOptions{})
				return err
			}, 60*time.Second, 1*time.Second).Should(MatchError(errors.IsNotFound, "k8serrors.IsNotFound"))

			By("Creating KubeVirt Object")
			createKv(copyOriginalKv(originalKv))

			By("Creating KubeVirt Object Created and Ready Condition")
			testsuite.EnsureKubevirtReadyWithTimeout(originalKv, 300*time.Second)

			By("Verifying infrastructure is Ready")
			allKvInfraPodsAreReady(originalKv)
			// We're just verifying that a few common components that
			// should always exist get re-deployed.
			sanityCheckDeploymentsExist()
		})

		Describe("[rfe_id:3578][crit:high][vendor:cnv-qe@redhat.com][level:component] deleting with BlockUninstallIfWorkloadsExist", func() {
			BeforeEach(func() {
				By("setting the right uninstall strategy")
				patchBytes, err := patch.New(patch.WithAdd("/spec/uninstallStrategy", v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist)).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				_, err = virtClient.KubeVirt(originalKv.Namespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() (v1.KubeVirtUninstallStrategy, error) {
					kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
					return kv.Spec.UninstallStrategy, err
				}, 60*time.Second, time.Second).Should(Equal(v1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist))

				By("waiting for the operator to finish reconciling after the patch")
				allKvInfraPodsAreReady(originalKv)
				sanityCheckDeploymentsExist()
			})

			AfterEach(func() {
				By("cleaning the uninstall strategy")
				patchBytes, err := patch.New(patch.WithRemove("/spec/uninstallStrategy")).GeneratePayload()
				Expect(err).ToNot(HaveOccurred())
				_, err = virtClient.KubeVirt(originalKv.Namespace).Patch(context.Background(), originalKv.Name, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
				Expect(err).ToNot(HaveOccurred())
				Eventually(func() (v1.KubeVirtUninstallStrategy, error) {
					kv, err := virtClient.KubeVirt(originalKv.Namespace).Get(context.Background(), originalKv.Name, metav1.GetOptions{})
					return kv.Spec.UninstallStrategy, err
				}, 60*time.Second, time.Second).Should(BeEmpty())

				By("waiting for the operator to finish reconciling after the patch")
				allKvInfraPodsAreReady(originalKv)
				sanityCheckDeploymentsExist()
			})

			It("[test_id:3683]should be blocked if a workload exists", func() {
				By("creating a simple VMI")
				vmi := libvmifact.NewGuestless()
				_, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(nil)).Create(context.Background(), vmi, metav1.CreateOptions{})
				Expect(err).ToNot(HaveOccurred())

				By("Deleting KubeVirt object")
				err = virtClient.KubeVirt(originalKv.Namespace).Delete(context.Background(), originalKv.Name, metav1.DeleteOptions{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("there are still Virtual Machine Instances present"))
			})
		})

		It("[test_id:3148]should be able to create kubevirt install with custom image tag", decorators.Upgrade, func() {

			if flags.KubeVirtVersionTagAlt == "" {
				Fail("KubeVirtVersionTagAlt must be configured for custom image tag tests")
			}

			allKvInfraPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			kv := copyOriginalKv(originalKv)
			kv.Name = "kubevirt-alt-install"
			kv.Spec = v1.KubeVirtSpec{
				ImageTag:      flags.KubeVirtVersionTagAlt,
				ImageRegistry: flags.KubeVirtRepoPrefix,
			}
			reinstallKubeVirt(kv, 300*time.Second)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false, originalKv.Name)
		})

		// this test ensures that we can deal with image prefixes in case they are not used for tests already
		It("[test_id:3149]should be able to create kubevirt install with image prefix", decorators.Upgrade, func() {

			if flags.ImagePrefixAlt == "" {
				Fail("ImagePrefixAlt must be configured for image prefix tests")
			}

			kv := copyOriginalKv(originalKv)

			allKvInfraPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			_, _, _, oldImageName, _ := parseOperatorImage()

			By("Update Operator using imagePrefixAlt")
			newImageName := flags.ImagePrefixAlt + oldImageName
			patchOperator(&newImageName, nil)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Verifying infrastructure Is Updated")
			allKvInfraPodsAreReady(kv)

			By("Verifying deployments have correct image name")
			for _, name := range []string{"virt-operator", "virt-api", "virt-controller"} {
				_, _, _, actualImageName, _ := parseDeployment(name)
				Expect(actualImageName).To(ContainSubstring(flags.ImagePrefixAlt), fmt.Sprintf("%s should have correct image prefix", name))
			}
			handlerImageName := getDaemonsetImage("virt-handler")
			Expect(handlerImageName).To(ContainSubstring(flags.ImagePrefixAlt), "virt-handler should have correct image prefix")

			By("Verifying VMs are working")
			vmi := libvmifact.NewGuestless()
			vmi, err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Create(context.Background(), vmi, metav1.CreateOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "Create VMI successfully")
			libwait.WaitForSuccessfulVMIStart(vmi)

			By("Verifying virt-launcher image is also prefixed")
			pod, err := libpod.GetPodByVirtualMachineInstance(vmi, vmi.Namespace)
			Expect(err).NotTo(HaveOccurred())

			for _, container := range pod.Spec.Containers {
				if container.Name == "compute" {
					_, imageName, _ := parseImage(container.Image)
					Expect(imageName).To(ContainSubstring(flags.ImagePrefixAlt), "launcher image should have prefix")
				}
			}

			By("Deleting VM")
			err = virtClient.VirtualMachineInstance(testsuite.GetTestNamespace(vmi)).Delete(context.Background(), vmi.Name, metav1.DeleteOptions{})
			Expect(err).ShouldNot(HaveOccurred(), "Delete VMI successfully")

			By("Restore Operator using original imagePrefix ")
			patchOperator(&oldImageName, nil)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Verifying infrastructure Is Restored to original version")
			allKvInfraPodsAreReady(kv)
		})

		It("[test_id:3150]should be able to update kubevirt install with custom image tag", decorators.Upgrade, func() {
			if flags.KubeVirtVersionTagAlt == "" {
				Fail("KubeVirtVersionTagAlt must be configured for custom image tag tests")
			}

			var vmis []*v1.VirtualMachineInstance
			if checks.HasAtLeastTwoNodes() {
				vmis, err = generateMigratableVMIs(2)
				Expect(err).NotTo(HaveOccurred())
			}
			vmisNonMigratable := []*v1.VirtualMachineInstance{libvmifact.NewAlpine(), libvmifact.NewAlpine()}

			allKvInfraPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			kv := copyOriginalKv(originalKv)
			kv.Name = "kubevirt-alt-install"
			kv.Spec.Configuration.NetworkConfiguration = &v1.NetworkConfiguration{
				PermitBridgeInterfaceOnPodNetwork: pointer.P(true),
			}
			kv.Spec.WorkloadUpdateStrategy.WorkloadUpdateMethods = []v1.WorkloadUpdateMethod{v1.WorkloadUpdateMethodLiveMigrate, v1.WorkloadUpdateMethodEvict}
			reinstallKubeVirt(kv, 300*time.Second)

			By("Starting multiple migratable VMIs before performing update")

			if len(vmis) > 0 {
				netAttachDef := libnet.NewBridgeNetAttachDef(secondaryNetworkName, secondaryNetworkName)
				_, err := libnet.CreateNetAttachDef(context.Background(), testsuite.GetTestNamespace(vmis[0]), netAttachDef)
				Expect(err).NotTo(HaveOccurred())
				vmis = createRunningVMIs(vmis)
			}

			vmisNonMigratable = createRunningVMIs(vmisNonMigratable)

			By("Updating KubeVirtObject With Alt Tag")
			patches := patch.New(patch.WithAdd("/spec/imageTag", flags.KubeVirtVersionTagAlt))
			patchKV(kv.Name, patches)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Verifying infrastructure Is Updated")
			allKvInfraPodsAreReady(kv)

			By("Verifying all non-migratable vmi workloads are shutdown")
			verifyVMIsEvicted(vmisNonMigratable)

			By("Verifying all migratable vmi workloads are updated via live migration")
			verifyVMIsUpdated(vmis)

			By("Deleting VMIs")
			deleteVMIs(vmis)
			deleteVMIs(vmisNonMigratable)

			By("Deleting KubeVirt object")
			deleteAllKvAndWait(false, originalKv.Name)
		})

		// NOTE - this test verifies new operators can grab the leader election lease
		// during operator updates. The only way the new infrastructure is deployed
		// is if the update operator is capable of getting the lease.
		It("[test_id:3151]should be able to update kubevirt install when operator updates if no custom image tag is set", decorators.Upgrade, func() {

			if flags.KubeVirtVersionTagAlt == "" {
				Fail("KubeVirtVersionTagAlt must be configured for custom image tag tests")
			}

			kv := copyOriginalKv(originalKv)

			allKvInfraPodsAreReady(originalKv)
			sanityCheckDeploymentsExist()

			By("Update Virt-Operator using  Alt Tag")
			patchOperator(nil, &flags.KubeVirtVersionTagAlt)

			// should result in kubevirt cr entering updating state
			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Verifying infrastructure Is Updated")
			allKvInfraPodsAreReady(kv)

			// by using the tag, we also test if resetting (in AfterEach) from tag to sha for the same "version" works
			By("Restore Operator Version using original tag. ")
			patchOperator(nil, &flags.KubeVirtVersionTag)

			By("Wait for Updating Condition")
			waitForUpdateCondition(kv)

			By("Waiting for KV to stabilize")
			testsuite.EnsureKubevirtReadyWithTimeout(kv, 300*time.Second)

			By("Verifying infrastructure Is Restored to original version")
			allKvInfraPodsAreReady(kv)
		})

		// TODO: not Serial
		It("[test_id:3152]should fail if KV object already exists", func() {

			newKv := copyOriginalKv(originalKv)
			newKv.Name = "someother-kubevirt"

			By("Creating another KubeVirt object")
			_, err = virtClient.KubeVirt(newKv.Namespace).Create(context.Background(), newKv, metav1.CreateOptions{})
			Expect(err).To(MatchError(ContainSubstring("Kubevirt is already created")))
		})

		It("[test_id:4612]should create non-namespaces resources without owner references", func() {
			crd, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(crd.ObjectMeta.OwnerReferences).To(BeEmpty())
		})

		It("[test_id:4613]should remove owner references on non-namespaces resources when updating a resource", func() {
			By("getting existing resource to reference")
			cm, err := virtClient.CoreV1().ConfigMaps(originalKv.Namespace).Get(context.Background(), "kubevirt-ca", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			ownerRef := []metav1.OwnerReference{
				*metav1.NewControllerRef(&k8sv1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name: cm.Name,
						UID:  cm.UID,
					},
				}, schema.GroupVersionKind{Version: "v1", Kind: "ConfigMap", Group: ""}),
			}

			By("adding an owner reference")
			origCRD, err := virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			crd := origCRD.DeepCopy()
			crd.OwnerReferences = ownerRef
			patch := patchCRD(origCRD, crd)
			_, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), "virtualmachineinstances.kubevirt.io", types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())
			By("verifying that the owner reference is there")
			origCRD, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(origCRD.OwnerReferences).ToNot(BeEmpty())

			By("changing the install version to force an update")
			crd = origCRD.DeepCopy()
			crd.Annotations[v1.InstallStrategyVersionAnnotation] = "outdated"
			patch = patchCRD(origCRD, crd)
			_, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Patch(context.Background(), "virtualmachineinstances.kubevirt.io", types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).ToNot(HaveOccurred())

			By("waiting until the owner reference disappears again")
			Eventually(func() []metav1.OwnerReference {
				crd, err = virtClient.ExtensionsClient().ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), "virtualmachineinstances.kubevirt.io", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return crd.OwnerReferences
			}, 20*time.Second, 1*time.Second).Should(BeEmpty())
			Expect(crd.ObjectMeta.OwnerReferences).To(BeEmpty())
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
				labelSelector := fmt.Sprintf(v1.CreatedByLabel + "=" + string(uid))
				pods, err = virtClient.CoreV1().Pods(testsuite.GetTestNamespace(vmi)).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
				Expect(err).ToNot(HaveOccurred(), "Should get virt-launcher")
				Expect(pods.Items).To(HaveLen(1))
				Expect(pods.Items[0].Annotations[OpenShiftSCCLabel]).To(
					Equal("kubevirt-controller"), "Should virt-launcher be assigned to kubevirt-controller SCC",
				)
			})
		})
	})

	Context("With PrometheusRule Enabled", func() {

		BeforeEach(func() {
			if !prometheusRuleEnabled() {
				Skip("Test applies on when PrometheusRule is defined")
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
				Skip("Test requires ServiceMonitor to be valid")
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

	It("[test_id:4617]should adopt previously unmanaged entities by updating its metadata", func() {
		By("removing registration metadata")
		patchData, err := patch.New(patch.WithReplace("/metadata/labels", struct{}{})).GeneratePayload()
		Expect(err).ToNot(HaveOccurred())

		_, err = virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Patch(context.Background(), components.VirtApiCertSecretName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = aggregatorClient.ApiregistrationV1().APIServices().Patch(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Patch(context.Background(), components.VirtAPIValidatingWebhookName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())
		_, err = virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Patch(context.Background(), components.VirtAPIMutatingWebhookName, types.JSONPatchType, patchData, metav1.PatchOptions{})
		Expect(err).ToNot(HaveOccurred())

		By("checking that it gets added again")
		Eventually(func() map[string]string {
			secret, err := virtClient.CoreV1().Secrets(flags.KubeVirtInstallNamespace).Get(context.Background(), components.VirtApiCertSecretName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return secret.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			apiService, err := aggregatorClient.ApiregistrationV1().APIServices().Get(context.Background(), fmt.Sprintf("%s.subresources.kubevirt.io", v1.ApiLatestVersion), metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return apiService.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			validatingWebhook, err := virtClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.Background(), components.VirtAPIValidatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return validatingWebhook.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
		Eventually(func() map[string]string {
			mutatingWebhook, err := virtClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.Background(), components.VirtAPIMutatingWebhookName, metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return mutatingWebhook.Labels
		}, 20*time.Second, 1*time.Second).Should(HaveKeyWithValue(v1.ManagedByLabel, v1.ManagedByLabelOperatorValue))
	})

	Context("RoleAggregationStrategy", func() {
		clusterRolesWithAggregateLabels := map[string]string{
			rbac.ClusterRoleAdmin: "rbac.authorization.k8s.io/aggregate-to-admin",
			rbac.ClusterRoleEdit:  "rbac.authorization.k8s.io/aggregate-to-edit",
			rbac.ClusterRoleView:  "rbac.authorization.k8s.io/aggregate-to-view",
		}

		It("should disable aggregate labels when set to Manual and restore them when set to AggregateToDefault", func() {
			By("Verifying aggregate labels are present by default")
			for name, labelKey := range clusterRolesWithAggregateLabels {
				clusterRole, err := kubevirt.Client().RbacV1().ClusterRoles().Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterRole.Labels).To(HaveKeyWithValue(labelKey, "true"),
					"ClusterRole %s should have label %s", name, labelKey)
			}

			By("Setting RoleAggregationStrategy to Manual with OptOutRoleAggregation feature gate")
			currentKV := libkubevirt.GetCurrentKv(kubevirt.Client())
			if currentKV.Spec.Configuration.DeveloperConfiguration == nil {
				currentKV.Spec.Configuration.DeveloperConfiguration = &v1.DeveloperConfiguration{}
			}
			currentKV.Spec.Configuration.DeveloperConfiguration.FeatureGates = append(
				currentKV.Spec.Configuration.DeveloperConfiguration.FeatureGates,
				featuregate.OptOutRoleAggregation,
			)
			currentKV.Spec.Configuration.RoleAggregationStrategy = pointer.P(v1.RoleAggregationStrategyManual)
			kv := kvconfig.UpdateKubeVirtConfigValueAndWait(currentKV.Spec.Configuration)

			By("Verifying aggregate labels are set to false")
			for name, labelKey := range clusterRolesWithAggregateLabels {
				clusterRole, err := kubevirt.Client().RbacV1().ClusterRoles().Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterRole.Labels).To(HaveKeyWithValue(labelKey, "false"),
					"ClusterRole %s should have label %s set to false when RoleAggregationStrategy is Manual", name, labelKey)
			}

			By("Setting RoleAggregationStrategy to AggregateToDefault")
			kv.Spec.Configuration.RoleAggregationStrategy = pointer.P(v1.RoleAggregationStrategyAggregateToDefault)
			kvconfig.UpdateKubeVirtConfigValueAndWait(kv.Spec.Configuration)

			By("Verifying aggregate labels are restored")
			for name, labelKey := range clusterRolesWithAggregateLabels {
				clusterRole, err := kubevirt.Client().RbacV1().ClusterRoles().Get(context.Background(), name, metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterRole.Labels).To(HaveKeyWithValue(labelKey, "true"),
					"ClusterRole %s should have label %s when RoleAggregationStrategy is AggregateToDefault", name, labelKey)
			}
		})
	})
})
