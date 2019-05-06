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

package services

import (
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	fakenetworkclient "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/client/clientset/versioned/fake"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	kubev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	"kubevirt.io/kubevirt/pkg/kubecli"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

const namespaceKubevirt = "kubevirt"

var _ = Describe("Template", func() {

	log.Log.SetIOWriter(GinkgoWriter)

	pvcCache := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	var svc TemplateService

	ctrl := gomock.NewController(GinkgoT())
	virtClient := kubecli.NewMockKubevirtClient(ctrl)
	config, configMapInformer := testutils.NewFakeClusterConfig(&kubev1.ConfigMap{})

	enableFeatureGate := func(featureGate string) {
		testutils.UpdateFakeClusterConfig(configMapInformer, &kubev1.ConfigMap{
			Data: map[string]string{virtconfig.FeatureGatesKey: featureGate},
		})
	}
	disableFeatureGates := func() {
		testutils.UpdateFakeClusterConfig(configMapInformer, &kubev1.ConfigMap{})
	}

	BeforeEach(func() {
		svc = NewTemplateService("kubevirt/virt-launcher",
			"/var/run/kubevirt",
			"/var/run/kubevirt-ephemeral-disks",
			"pull-secret-1",
			pvcCache,
			virtClient,
			config,
		)

		// Set up mock clients
		networkClient := fakenetworkclient.NewSimpleClientset()
		virtClient.EXPECT().NetworkClient().Return(networkClient).AnyTimes()
		// Sadly, we cannot pass desired attachment objects into
		// Clientset constructor because UnsafeGuessKindToResource
		// calculates incorrect object kind (without dashes). Instead
		// of that, we use tracker Create function to register objects
		// under explicitly defined schema name
		gvr := schema.GroupVersionResource{
			Group:    "k8s.cni.cncf.io",
			Version:  "v1",
			Resource: "network-attachment-definitions",
		}
		for _, name := range []string{"default", "test1"} {
			network := &networkv1.NetworkAttachmentDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "default",
				},
			}
			err := networkClient.Tracker().Create(gvr, network, "default")
			Expect(err).To(Not(HaveOccurred()))
		}
		// create a network in a different namespace
		network := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test1",
				Namespace: "other-namespace",
			},
		}
		err := networkClient.Tracker().Create(gvr, network, "other-namespace")
		Expect(err).To(Not(HaveOccurred()))
	})

	AfterEach(func() {
		disableFeatureGates()
	})

	Describe("Rendering", func() {
		Context("launch template with correct parameters", func() {
			It("should work", func() {
				trueVar := true
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				pod, err := svc.RenderLaunchManifest(&v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns", UID: "1234", Annotations: annotations}, Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}}})
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.Annotations).To(Equal(map[string]string{
					v1.DomainAnnotation: "testvmi",
				}))
				Expect(pod.ObjectMeta.OwnerReferences).To(Equal([]metav1.OwnerReference{{
					APIVersion:         v1.VirtualMachineInstanceGroupVersionKind.GroupVersion().String(),
					Kind:               v1.VirtualMachineInstanceGroupVersionKind.Kind,
					Name:               "testvmi",
					UID:                "1234",
					Controller:         &trueVar,
					BlockOwnerDeletion: &trueVar,
				}}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					v1.NodeSchedulable: "true",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", "5m",
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "testns",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--readiness-file", "/var/run/kubevirt-infra/healthy",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1",
					"--less-pvc-space-toleration", "10"}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
				By("setting the right hostname")
				Expect(pod.Spec.Hostname).To(Equal("testvmi"))
				Expect(pod.Spec.Subdomain).To(BeEmpty())
			})
		})
		Context("with multus annotation", func() {
			It("should add multus networks in the pod annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
							{Name: "other-test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "other-namespace/test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(Equal(true))
				expectedIfaces := ("[" +
					"{\"interface\":\"net1\",\"name\":\"default\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net2\",\"name\":\"test1\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net3\",\"name\":\"test1\",\"namespace\":\"other-namespace\"}" +
					"]")
				Expect(value).To(Equal(expectedIfaces))
			})
			It("should add default multus networks in the multus default-network annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default", Default: true},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["v1.multus-cni.io/default-network"]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("default"))
				value, ok = pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("[{\"interface\":\"net1\",\"name\":\"test1\",\"namespace\":\"default\"}]"))
			})
			It("should add MAC address in the pod annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								Interfaces: []v1.Interface{
									v1.Interface{
										Name: "test1",
										InterfaceBindingMethod: v1.InterfaceBindingMethod{
											SRIOV: &v1.InterfaceSRIOV{},
										},
										MacAddress: "de:ad:00:00:be:af",
									},
								},
							},
						},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Multus: &v1.MultusNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["k8s.v1.cni.cncf.io/networks"]
				Expect(ok).To(Equal(true))
				expectedIfaces := ("[" +
					"{\"interface\":\"net1\",\"name\":\"default\",\"namespace\":\"default\"}," +
					"{\"interface\":\"net2\",\"mac\":\"de:ad:00:00:be:af\",\"name\":\"test1\",\"namespace\":\"default\"}" +
					"]")
				Expect(value).To(Equal(expectedIfaces))
			})
		})
		Context("with genie annotation", func() {
			It("should add genie networks in the pod annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
						Networks: []v1.Network{
							{Name: "default",
								NetworkSource: v1.NetworkSource{
									Genie: &v1.GenieNetwork{NetworkName: "default"},
								}},
							{Name: "test1",
								NetworkSource: v1.NetworkSource{
									Genie: &v1.GenieNetwork{NetworkName: "test1"},
								}},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations["cni"]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("default,test1"))
			})
		})
		Context("with masquerade interface", func() {
			It("should add the istio annotation", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Devices: v1.Devices{
								Interfaces: []v1.Interface{
									{Name: "default",
										InterfaceBindingMethod: v1.InterfaceBindingMethod{Masquerade: &v1.InterfaceMasquerade{}}},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				value, ok := pod.Annotations[ISTIO_KUBEVIRT_ANNOTATION]
				Expect(ok).To(Equal(true))
				Expect(value).To(Equal("k6t-eth0"))
			})
		})
		Context("with node selectors", func() {
			It("should add node selectors to template", func() {

				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234", Annotations: annotations}, Spec: v1.VirtualMachineInstanceSpec{NodeSelector: nodeSelector, Domain: v1.DomainSpec{}}}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Image).To(Equal("kubevirt/virt-launcher"))
				Expect(pod.ObjectMeta.Labels).To(Equal(map[string]string{
					v1.AppLabel:       "virt-launcher",
					v1.CreatedByLabel: "1234",
				}))
				Expect(pod.ObjectMeta.GenerateName).To(Equal("virt-launcher-testvmi-"))
				Expect(pod.Spec.NodeSelector).To(Equal(map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}))
				Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"/usr/bin/virt-launcher",
					"--qemu-timeout", "5m",
					"--name", "testvmi",
					"--uid", "1234",
					"--namespace", "default",
					"--kubevirt-share-dir", "/var/run/kubevirt",
					"--ephemeral-disk-dir", "/var/run/kubevirt-ephemeral-disks",
					"--readiness-file", "/var/run/kubevirt-infra/healthy",
					"--grace-period-seconds", "45",
					"--hook-sidecars", "1",
					"--less-pvc-space-toleration", "10"}))
				Expect(pod.Spec.Containers[1].Name).To(Equal("hook-sidecar-0"))
				Expect(pod.Spec.Containers[1].Image).To(Equal("some-image:v1"))
				Expect(pod.Spec.Containers[1].ImagePullPolicy).To(Equal(kubev1.PullPolicy("IfNotPresent")))
				Expect(pod.Spec.Containers[1].VolumeMounts[0].MountPath).To(Equal(hooks.HookSocketsSharedDirectory))
				Expect(pod.Spec.Volumes[0].EmptyDir.Medium).To(Equal(kubev1.StorageMedium("")))
				Expect(pod.Spec.Volumes[2].HostPath.Path).To(Equal("/var/run/kubevirt"))
				Expect(pod.Spec.Containers[0].VolumeMounts[1].MountPath).To(Equal("/var/run/kubevirt"))
				Expect(*pod.Spec.TerminationGracePeriodSeconds).To(Equal(int64(60)))
			})

			It("should add node selector for node discovery feature to template", func() {
				enableFeatureGate(virtconfig.CPUNodeDiscoveryGate)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{
								Model: "Conroe",
								Features: []v1.CPUFeature{
									{
										Name:   "lahf_lm",
										Policy: "require",
									},
									{
										Name:   "mmx",
										Policy: "disable",
									},
									{
										Name:   "ssse3",
										Policy: "forbid",
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				cpuModelLabel, err := CPUModelLabelFromCPUModel(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(cpuModelLabel, "true"))

				cpuFeatureLabels := CPUFeatureLabelsFromCPUFeatures(&vmi)
				for _, featureLabel := range cpuFeatureLabels {
					Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(featureLabel, "true"))
				}
			})

			It("should add node selectors from kubevirt-config configMap", func() {
				testutils.UpdateFakeClusterConfig(configMapInformer, &kubev1.ConfigMap{
					Data: map[string]string{virtconfig.NodeSelectorsKey: "kubernetes.io/hostname=node02\nnode-role.kubernetes.io/compute=true\n"},
				})
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: []v1.Volume{}, Domain: v1.DomainSpec{}},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue("kubernetes.io/hostname", "node02"))
				Expect(pod.Spec.NodeSelector).To(HaveKeyWithValue("node-role.kubernetes.io/compute", "true"))
			})

			It("should not add node selector for hyperv nodes if VMI does not request hyperv features", func() {
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
			})

			It("should not add node selector for hyperv nodes if VMI requests hyperv features, but feature gate is disabled", func() {
				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									SyNIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									Reenlightenment: &v1.FeatureState{
										Enabled: &enabled,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
			})

			It("should add node selector for hyperv nodes if VMI requests hyperv features which depend on host kernel", func() {
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									SyNIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									SyNICTimer: &v1.FeatureState{
										Enabled: &enabled,
									},
									Frequencies: &v1.FeatureState{
										Enabled: &enabled,
									},
									IPI: &v1.FeatureState{
										Enabled: &enabled,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"synic", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"synictimer", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"frequencies", "true"))
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(NFD_KVM_INFO_PREFIX+"ipi", "true"))
			})

			It("should not add node selector for hyperv nodes if VMI requests hyperv features which do not depend on host kernel", func() {
				enableFeatureGate(virtconfig.HypervStrictCheckGate)

				var retries uint32 = 4095
				enabled := true
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Features: &v1.Features{
								Hyperv: &v1.FeatureHyperv{
									Relaxed: &v1.FeatureState{
										Enabled: &enabled,
									},
									VAPIC: &v1.FeatureState{
										Enabled: &enabled,
									},
									Spinlocks: &v1.FeatureSpinlocks{
										Enabled: &enabled,
										Retries: &retries,
									},
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.NodeSelector).To(Not(HaveKey(ContainSubstring(NFD_KVM_INFO_PREFIX))))
			})

			It("should add default cpu/memory resources to the sidecar container if cpu pinning was requested", func() {
				nodeSelector := map[string]string{
					"kubernetes.io/hostname": "master",
					v1.NodeSchedulable:       "true",
				}
				annotations := map[string]string{
					hooks.HookSidecarListAnnotationName: `[{"image": "some-image:v1", "imagePullPolicy": "IfNotPresent"}]`,
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "testvmi",
						Namespace:   "default",
						UID:         "1234",
						Annotations: annotations,
					},
					Spec: v1.VirtualMachineInstanceSpec{
						NodeSelector: nodeSelector,
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{
								Cores:                 2,
								DedicatedCPUPlacement: true,
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				cpu := resource.MustParse("200m")
				mem := resource.MustParse("64M")
				Expect(pod.Spec.Containers[1].Resources.Limits.Memory().Cmp(mem)).To(BeZero())
				Expect(pod.Spec.Containers[1].Resources.Limits.Cpu().Cmp(cpu)).To(BeZero())

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_SYS_NICE {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted SYS_NICE capability")
				Expect(pod.Spec.NodeSelector).Should(HaveKeyWithValue(v1.CPUManager, "true"))
			})
			It("should add node affinity to pod", func() {
				nodeAffinity := kubev1.NodeAffinity{}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{NodeAffinity: &nodeAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{NodeAffinity: &nodeAffinity}))
			})

			It("should add pod affinity to pod", func() {
				podAffinity := kubev1.PodAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAffinity: &podAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAffinity: &podAffinity}))
			})

			It("should add pod anti-affinity to pod", func() {
				podAntiAffinity := kubev1.PodAntiAffinity{}
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Affinity: &kubev1.Affinity{PodAntiAffinity: &podAntiAffinity},
						Domain:   v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeEquivalentTo(&kubev1.Affinity{PodAntiAffinity: &podAntiAffinity}))
			})

			It("should add tolerations to pod", func() {
				podToleration := kubev1.Toleration{Key: "test"}
				var tolerationSeconds int64 = 14
				vm := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Tolerations: []kubev1.Toleration{
							{
								Key:               podToleration.Key,
								TolerationSeconds: &tolerationSeconds,
							},
						},
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vm)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Tolerations).To(BeEquivalentTo([]kubev1.Toleration{{Key: podToleration.Key, TolerationSeconds: &tolerationSeconds}}))
			})

			It("should use the hostname and subdomain if specified on the vm", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain:    v1.DomainSpec{},
						Hostname:  "myhost",
						Subdomain: "mydomain",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Hostname).To(Equal(vmi.Spec.Hostname))
				Expect(pod.Spec.Subdomain).To(Equal(vmi.Spec.Subdomain))
			})

			It("should add vmi labels to pod", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvmi",
						Namespace: "default",
						UID:       "1234",
						Labels: map[string]string{
							"key1": "val1",
							"key2": "val2",
						},
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Labels).To(Equal(
					map[string]string{
						"key1":            "val1",
						"key2":            "val2",
						v1.AppLabel:       "virt-launcher",
						v1.CreatedByLabel: "1234",
					},
				))
			})

			It("should not add empty affinity to pod", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "testvm", Namespace: "default", UID: "1234"},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{},
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Affinity).To(BeNil())
			})
		})
		Context("with cpu and memory constraints", func() {
			It("should add cpu and memory constraints to a template", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("2m"),
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Cpu().String()).To(Equal("2m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("1163507557"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("2163507557"))
			})
			It("should overcommit guest overhead if selected, by only adding the overhead to memory limits", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Resources: v1.ResourceRequirements{
								OvercommitGuestOverhead: true,
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("1G"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("2G"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().String()).To(Equal("1G"))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().String()).To(Equal("2163507557"))
			})
			It("should not add unset resources", func() {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(vmi.Spec.Domain.Resources.Requests.Memory().String()).To(Equal("64M"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Cpu().String()).To(Equal("1m"))
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(243)))

				// Limits for KVM and TUN devices should be requested.
				Expect(pod.Spec.Containers[0].Resources.Limits).ToNot(BeNil())
			})

			table.DescribeTable("should check autoattachGraphicsDevicse", func(autoAttach *bool, memory int) {

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{

							CPU: &v1.CPU{Cores: 3},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceCPU:    resource.MustParse("1m"),
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}
				vmi.Spec.Domain.Devices = v1.Devices{
					AutoattachGraphicsDevice: autoAttach,
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(memory)))
			},
				table.Entry("and consider graphics overhead if it is not set", nil, 243),
				table.Entry("and consider graphics overhead if it is set to true", True(), 243),
				table.Entry("and not consider graphics overhead if it is set to false", False(), 226),
			)
		})

		Context("with hugepages constraints", func() {
			table.DescribeTable("should add to the template constraints ", func(value string) {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testvmi",
						Namespace: "default",
						UID:       "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						Domain: v1.DomainSpec{
							Memory: &v1.Memory{
								Hugepages: &v1.Hugepages{
									PageSize: value,
								},
							},
							Resources: v1.ResourceRequirements{
								Requests: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
								Limits: kubev1.ResourceList{
									kubev1.ResourceMemory: resource.MustParse("64M"),
								},
							},
						},
					},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Containers[0].Resources.Requests.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(162)))
				Expect(pod.Spec.Containers[0].Resources.Limits.Memory().ToDec().ScaledValue(resource.Mega)).To(Equal(int64(162)))

				hugepageType := kubev1.ResourceName(kubev1.ResourceHugePagesPrefix + value)
				hugepagesRequest := pod.Spec.Containers[0].Resources.Requests[hugepageType]
				hugepagesLimit := pod.Spec.Containers[0].Resources.Limits[hugepageType]
				Expect(hugepagesRequest.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))
				Expect(hugepagesLimit.ToDec().ScaledValue(resource.Mega)).To(Equal(int64(64)))

				Expect(len(pod.Spec.Volumes)).To(Equal(5))
				Expect(pod.Spec.Volumes[0].EmptyDir).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].EmptyDir.Medium).To(Equal(kubev1.StorageMediumHugePages))

				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(5))
				Expect(pod.Spec.Containers[0].VolumeMounts[3].MountPath).To(Equal("/dev/hugepages"))
			},
				table.Entry("hugepages-2Mi", "2Mi"),
				table.Entry("hugepages-1Gi", "1Gi"),
			)
		})

		Context("with file mode pvc source", func() {
			It("should add volume to template", func() {
				namespace := "testns"
				pvcName := "pvcFile"
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")

				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).To(BeEmpty(), "No devices in manifest for 1st container")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(5), "4 mounts in manifest for 1st container")
				Expect(pod.Spec.Containers[0].VolumeMounts[3].Name).To(Equal(volumeName), "1st mount in manifest for 1st container has correct name")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(5), "Found 4 volumes in manifest")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with blockdevice mode pvc source", func() {
			It("should add device to template", func() {
				namespace := "testns"
				pvcName := "pvcDevice"
				mode := kubev1.PersistentVolumeBlock
				pvc := kubev1.PersistentVolumeClaim{
					TypeMeta:   metav1.TypeMeta{Kind: "PersistentVolumeClaim", APIVersion: "v1"},
					ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: pvcName},
					Spec: kubev1.PersistentVolumeClaimSpec{
						VolumeMode: &mode,
					},
				}
				err := pvcCache.Add(&pvc)
				Expect(err).ToNot(HaveOccurred(), "Added PVC to cache successfully")
				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred(), "Render manifest successfully")

				Expect(pod.Spec.Containers[0].VolumeDevices).ToNot(BeEmpty(), "Found some devices for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeDevices)).To(Equal(1), "Found 1 device for 1st container")
				Expect(pod.Spec.Containers[0].VolumeDevices[0].Name).To(Equal(volumeName), "Found device for 1st container with correct name")

				Expect(pod.Spec.Containers[0].VolumeMounts).ToNot(BeEmpty(), "Found some mounts in manifest for 1st container")
				Expect(len(pod.Spec.Containers[0].VolumeMounts)).To(Equal(4), "Found 3 mounts in manifest for 1st container")

				Expect(pod.Spec.Volumes).ToNot(BeEmpty(), "Found some volumes in manifest")
				Expect(len(pod.Spec.Volumes)).To(Equal(5), "Found 4 volumes in manifest")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim).ToNot(BeNil(), "Found PVC volume")
				Expect(pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName).To(Equal(pvcName), "Found PVC volume with correct name")
			})
		})

		Context("with non existing pvc source", func() {
			It("should result in an error", func() {
				namespace := "testns"
				pvcName := "pvcNotExisting"
				volumeName := "pvc-volume"
				volumes := []v1.Volume{
					{
						Name: volumeName,
						VolumeSource: v1.VolumeSource{
							PersistentVolumeClaim: &kubev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: namespace, UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				_, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).To(HaveOccurred(), "Render manifest results in an error")
				Expect(err).To(BeAssignableToTypeOf(PvcNotFoundError(errors.New(""))), "Render manifest results in an PvsNotFoundError")
			})
		})

		Context("with launcher's pull secret", func() {
			It("should contain launcher's secret in pod spec", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(1))
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-1"))
			})

		})

		Context("with ContainerDisk pull secrets", func() {
			volumes := []v1.Volume{
				{
					Name: "containerdisk1",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image:           "my-image-1",
							ImagePullSecret: "pull-secret-2",
						},
					},
				},
				{
					Name: "containerdisk2",
					VolumeSource: v1.VolumeSource{
						ContainerDisk: &v1.ContainerDiskSource{
							Image: "my-image-2",
						},
					},
				},
			}

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
			}

			It("should add secret to pod spec", func() {
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// ContainerDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})

			It("should deduplicate identical secrets", func() {
				volumes[1].VolumeSource.ContainerDisk.ImagePullSecret = "pull-secret-2"

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.ImagePullSecrets)).To(Equal(2))

				// ContainerDisk secrets come first
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("pull-secret-2"))
				Expect(pod.Spec.ImagePullSecrets[1].Name).To(Equal("pull-secret-1"))
			})
		})

		Context("with sriov interface", func() {
			It("should run privileged", func() {
				sriovInterface := v1.InterfaceSRIOV{}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &sriovInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(*pod.Spec.Containers[0].SecurityContext.Privileged).To(Equal(true))
			})
			It("should mount pci related host directories", func() {
				sriovInterface := v1.InterfaceSRIOV{}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{SRIOV: &sriovInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				// Skip first three mounts that are generic for all launcher pods
				Expect(pod.Spec.Containers[0].VolumeMounts[3].MountPath).To(Equal("/sys/bus/pci/"))
				Expect(pod.Spec.Containers[0].VolumeMounts[4].MountPath).To(Equal("/sys/devices/"))
				Expect(pod.Spec.Containers[0].VolumeMounts[5].MountPath).To(Equal("/dev/vfio/"))
				Expect(pod.Spec.Volumes[0].HostPath.Path).To(Equal("/sys/bus/pci/"))
				Expect(pod.Spec.Volumes[1].HostPath.Path).To(Equal("/sys/devices/"))
				Expect(pod.Spec.Volumes[2].HostPath.Path).To(Equal("/dev/vfio/"))
			})
		})
		Context("with slirp interface", func() {
			It("Should have empty port list in the pod manifest", func() {
				slirpInterface := v1.InterfaceSlirp{}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(0))
			})
			It("Should create a port list in the pod manifest", func() {
				slirpInterface := v1.InterfaceSlirp{}
				ports := []v1.Port{{Name: "http", Port: 80}, {Protocol: "UDP", Port: 80}, {Port: 90}, {Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{{Name: "testnet", Ports: ports, InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface}}}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(4))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("UDP")))
				Expect(pod.Spec.Containers[0].Ports[2].Name).To(Equal(""))
				Expect(pod.Spec.Containers[0].Ports[2].ContainerPort).To(Equal(int32(90)))
				Expect(pod.Spec.Containers[0].Ports[2].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[3].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[3].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[3].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
			It("Should create a port list in the pod manifest with multiple interfaces", func() {
				slirpInterface1 := v1.InterfaceSlirp{}
				slirpInterface2 := v1.InterfaceSlirp{}
				ports1 := []v1.Port{{Name: "http", Port: 80}}
				ports2 := []v1.Port{{Name: "other-http", Port: 80}}
				domain := v1.DomainSpec{}
				domain.Devices.Interfaces = []v1.Interface{
					{Name: "testnet",
						Ports:                  ports1,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface1}},
					{Name: "testnet",
						Ports:                  ports2,
						InterfaceBindingMethod: v1.InterfaceBindingMethod{Slirp: &slirpInterface2}}}

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(len(pod.Spec.Containers)).To(Equal(1))
				Expect(len(pod.Spec.Containers[0].Ports)).To(Equal(2))
				Expect(pod.Spec.Containers[0].Ports[0].Name).To(Equal("http"))
				Expect(pod.Spec.Containers[0].Ports[0].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[0].Protocol).To(Equal(kubev1.Protocol("TCP")))
				Expect(pod.Spec.Containers[0].Ports[1].Name).To(Equal("other-http"))
				Expect(pod.Spec.Containers[0].Ports[1].ContainerPort).To(Equal(int32(80)))
				Expect(pod.Spec.Containers[0].Ports[1].Protocol).To(Equal(kubev1.Protocol("TCP")))
			})
		})

		Context("with pod networking", func() {
			It("Should require tun device by default", func() {
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted NET_ADMIN capability")
			})

			It("Should require tun device if explicitly requested", func() {
				domain := v1.DomainSpec{}
				autoAttach := true
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				tun, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeTrue())
				Expect(int(tun.Value())).To(Equal(1))

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeTrue(), "Expected compute container to be granted NET_ADMIN capability")
			})

			It("Should not require tun device if explicitly rejected", func() {
				domain := v1.DomainSpec{}
				autoAttach := false
				domain.Devices.AutoattachPodInterface = &autoAttach

				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Domain: domain},
				}
				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				_, ok := pod.Spec.Containers[0].Resources.Limits[TunDevice]
				Expect(ok).To(BeFalse())

				found := false
				caps := pod.Spec.Containers[0].SecurityContext.Capabilities
				for _, cap := range caps.Add {
					if cap == CAP_NET_ADMIN {
						found = true
					}
				}
				Expect(found).To(BeFalse(), "Expected compute container to not be granted NET_ADMIN capability")
			})
		})

		Context("with a configMap volume source", func() {
			It("Should add the ConfigMap to template", func() {
				volumes := []v1.Volume{
					{
						Name: "configmap-volume",
						VolumeSource: v1.VolumeSource{
							ConfigMap: &v1.ConfigMapVolumeSource{
								LocalObjectReference: kubev1.LocalObjectReference{
									Name: "test-configmap",
								},
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(5))
				Expect(pod.Spec.Volumes[0].ConfigMap).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].ConfigMap.LocalObjectReference.Name).To(Equal("test-configmap"))
			})
		})

		Context("with a secret volume source", func() {
			It("should add the Secret to template", func() {
				volumes := []v1.Volume{
					{
						Name: "secret-volume",
						VolumeSource: v1.VolumeSource{
							Secret: &v1.SecretVolumeSource{
								SecretName: "test-secret",
							},
						},
					},
				}
				vmi := v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
				}

				pod, err := svc.RenderLaunchManifest(&vmi)
				Expect(err).ToNot(HaveOccurred())

				Expect(pod.Spec.Volumes).ToNot(BeEmpty())
				Expect(len(pod.Spec.Volumes)).To(Equal(5))
				Expect(pod.Spec.Volumes[0].Secret).ToNot(BeNil())
				Expect(pod.Spec.Volumes[0].Secret.SecretName).To(Equal("test-secret"))
			})
		})
		Context("with probes", func() {
			var vmi *v1.VirtualMachineInstance
			BeforeEach(func() {
				vmi = &v1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "testvmi", Namespace: "default", UID: "1234",
					},
					Spec: v1.VirtualMachineInstanceSpec{
						ReadinessProbe: &v1.Probe{
							InitialDelaySeconds: 2,
							TimeoutSeconds:      3,
							PeriodSeconds:       4,
							SuccessThreshold:    5,
							FailureThreshold:    6,
							Handler: v1.Handler{
								TCPSocket: &kubev1.TCPSocketAction{
									Port: intstr.Parse("80"),
									Host: "123",
								},
								HTTPGet: &kubev1.HTTPGetAction{
									Path: "test",
								},
							},
						},
						LivenessProbe: &v1.Probe{
							InitialDelaySeconds: 12,
							TimeoutSeconds:      13,
							PeriodSeconds:       14,
							SuccessThreshold:    15,
							FailureThreshold:    16,
							Handler: v1.Handler{
								TCPSocket: &kubev1.TCPSocketAction{
									Port: intstr.Parse("82"),
									Host: "1234",
								},
								HTTPGet: &kubev1.HTTPGetAction{
									Path: "test34",
								},
							},
						},
						Domain: v1.DomainSpec{}},
				}
			})
			It("should copy all specified probes", func() {
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				livenessProbe := pod.Spec.Containers[0].LivenessProbe
				readinessProbe := pod.Spec.Containers[0].ReadinessProbe
				Expect(livenessProbe.Handler.TCPSocket).To(Equal(vmi.Spec.LivenessProbe.TCPSocket))
				Expect(readinessProbe.Handler.TCPSocket).To(Equal(vmi.Spec.ReadinessProbe.TCPSocket))

				Expect(livenessProbe.Handler.HTTPGet).To(Equal(vmi.Spec.LivenessProbe.HTTPGet))
				Expect(readinessProbe.Handler.HTTPGet).To(Equal(vmi.Spec.ReadinessProbe.HTTPGet))

				Expect(livenessProbe.PeriodSeconds).To(Equal(vmi.Spec.LivenessProbe.PeriodSeconds))
				Expect(livenessProbe.InitialDelaySeconds).To(Equal(vmi.Spec.LivenessProbe.InitialDelaySeconds + LibvirtStartupDelay))
				Expect(livenessProbe.TimeoutSeconds).To(Equal(vmi.Spec.LivenessProbe.TimeoutSeconds))
				Expect(livenessProbe.SuccessThreshold).To(Equal(vmi.Spec.LivenessProbe.SuccessThreshold))
				Expect(livenessProbe.FailureThreshold).To(Equal(vmi.Spec.LivenessProbe.FailureThreshold))

				Expect(readinessProbe.PeriodSeconds).To(Equal(vmi.Spec.ReadinessProbe.PeriodSeconds))
				Expect(readinessProbe.InitialDelaySeconds).To(Equal(vmi.Spec.ReadinessProbe.InitialDelaySeconds + LibvirtStartupDelay))
				Expect(readinessProbe.TimeoutSeconds).To(Equal(vmi.Spec.ReadinessProbe.TimeoutSeconds))
				Expect(readinessProbe.SuccessThreshold).To(Equal(vmi.Spec.ReadinessProbe.SuccessThreshold))
				Expect(readinessProbe.FailureThreshold).To(Equal(vmi.Spec.ReadinessProbe.FailureThreshold))
			})

			It("should set a readiness probe on the pod, if no one was specified on the vmi", func() {
				vmi.Spec.ReadinessProbe = nil
				pod, err := svc.RenderLaunchManifest(vmi)
				Expect(err).ToNot(HaveOccurred())
				Expect(pod.Spec.Containers[0].ReadinessProbe).To(Not(BeNil()))
			})
		})

		It("should add the lessPVCSpaceToleration argument to the template", func() {
			expectedToleration := "42"
			testutils.UpdateFakeClusterConfig(configMapInformer, &kubev1.ConfigMap{
				Data: map[string]string{virtconfig.LessPVCSpaceTolerationKey: expectedToleration},
			})

			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: []v1.Volume{}, Domain: v1.DomainSpec{}},
			}
			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())

			Expect(pod.Spec.Containers[0].Command).To(ContainElement("--less-pvc-space-toleration"), "command arg key should be correct")
			Expect(pod.Spec.Containers[0].Command).To(ContainElement("42"), "command arg value should be correct")
		})

	})

	Describe("ServiceAccountName", func() {

		It("Should add service account if present", func() {
			serviceAccountName := "testAccount"
			volumes := []v1.Volume{
				{
					Name: "serviceaccount-volume",
					VolumeSource: v1.VolumeSource{
						ServiceAccount: &v1.ServiceAccountVolumeSource{
							ServiceAccountName: serviceAccountName,
						},
					},
				},
			}
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Volumes: volumes, Domain: v1.DomainSpec{}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(Equal(serviceAccountName), "ServiceAccount matches")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeTrue(), "Token automount is enabled")
		})

		It("Should not add service account if not present", func() {
			vmi := v1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "testvmi", Namespace: "default", UID: "1234",
				},
				Spec: v1.VirtualMachineInstanceSpec{Domain: v1.DomainSpec{}},
			}

			pod, err := svc.RenderLaunchManifest(&vmi)
			Expect(err).ToNot(HaveOccurred())
			Expect(pod.Spec.ServiceAccountName).To(BeEmpty(), "ServiceAccount is empty")
			Expect(*pod.Spec.AutomountServiceAccountToken).To(BeFalse(), "Token automount is disabled")
		})

	})
})

var _ = Describe("getResourceNameForNetwork", func() {
	It("should return empty string when resource name is not specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{}
		Expect(getResourceNameForNetwork(network)).To(Equal(""))
	})

	It("should return resource name if specified", func() {
		network := &networkv1.NetworkAttachmentDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					MULTUS_RESOURCE_NAME_ANNOTATION: "fake.com/fakeResource",
				},
			},
		}
		Expect(getResourceNameForNetwork(network)).To(Equal("fake.com/fakeResource"))
	})
})

var _ = Describe("getNamespaceAndNetworkName", func() {
	It("should return vmi namespace when namespace is implicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi, "testnet")
		Expect(namespace).To(Equal("testns"))
		Expect(networkName).To(Equal("testnet"))
	})

	It("should return namespace from networkName when namespace is explicit", func() {
		vmi := &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "testvmi", Namespace: "testns"}}
		namespace, networkName := getNamespaceAndNetworkName(vmi, "otherns/testnet")
		Expect(namespace).To(Equal("otherns"))
		Expect(networkName).To(Equal("testnet"))
	})
})

var _ = Describe("requestResource", func() {
	It("should register resource in limits and requests", func() {
		resources := kubev1.ResourceRequirements{}
		resources.Requests = make(kubev1.ResourceList)
		resources.Limits = make(kubev1.ResourceList)

		resource := "intel.com/sriov"
		resourceName := kubev1.ResourceName(resource)

		for i := int64(1); i <= 5; i++ {
			requestResource(&resources, resource)

			val, ok := resources.Limits[resourceName]
			Expect(ok).To(BeTrue())

			valInt, isInt := val.AsInt64()
			Expect(isInt).To(BeTrue())
			Expect(valInt).To(Equal(i))

			val, ok = resources.Requests[resourceName]
			Expect(ok).To(BeTrue())

			valInt, isInt = val.AsInt64()
			Expect(isInt).To(BeTrue())
			Expect(valInt).To(Equal(i))
		}
	})
})

func True() *bool {
	b := true
	return &b
}

func False() *bool {
	b := false
	return &b
}

func TestTemplate(t *testing.T) {
	log.Log.SetIOWriter(GinkgoWriter)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Template")
}
