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

package admitters

import (
	"context"
	"fmt"
	"net/http"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	admissionv1 "k8s.io/api/admission/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

var _ = Describe("Pod eviction admitter", func() {

	testns := "kubevirt-test-ns"
	var ctrl *gomock.Controller

	var kubeClient *fake.Clientset
	var virtClient *kubecli.MockKubevirtClient
	var vmiClient *kubecli.MockVirtualMachineInstanceInterface
	var podEvictionAdmitter PodEvictionAdmitter
	var clusterConfig *virtconfig.ClusterConfig

	newClusterConfig := func() *virtconfig.ClusterConfig {
		kv := kubecli.NewMinimalKubeVirt(testns)
		kv.Namespace = "kubevirt"
		if kv.Spec.Configuration.DeveloperConfiguration == nil {
			kv.Spec.Configuration.DeveloperConfiguration = &virtv1.DeveloperConfiguration{}
		}

		clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKV(kv)
		return clusterConfig
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		vmiClient = kubecli.NewMockVirtualMachineInstanceInterface(ctrl)
		kubeClient = fake.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().VirtualMachineInstance(testns).Return(vmiClient).AnyTimes()
		clusterConfig = newClusterConfig()
		podEvictionAdmitter = PodEvictionAdmitter{
			ClusterConfig: clusterConfig,
			VirtClient:    virtClient,
		}

		// Make sure that any unexpected call to the client will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})
	AfterEach(func() {
		ctrl.Finish()
	})

	Context("Migratable and evictable VMI", func() {

		var vmi *virtv1.VirtualMachineInstance
		liveMigrateStrategy := virtv1.EvictionStrategyLiveMigrate
		nodeName := "node01"

		BeforeEach(func() {
			vmi = &virtv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testns,
					Name:      "testvmi",
				},
				Status: virtv1.VirtualMachineInstanceStatus{
					Conditions: []virtv1.VirtualMachineInstanceCondition{
						{
							Type:   virtv1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionTrue,
						},
					},
					NodeName: nodeName,
				},
				Spec: virtv1.VirtualMachineInstanceSpec{
					EvictionStrategy: &liveMigrateStrategy,
				},
			}
		})

		It("Should deny review requests when updating the VMI fails", func() {

			By("Composing a dummy admission request on a virt-launcher pod")
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: testns,
					Annotations: map[string]string{
						virtv1.DomainAnnotation: vmi.Name,
					},
					Labels: map[string]string{
						virtv1.AppLabel: "virt-launcher",
					},
				},
				Spec: k8sv1.PodSpec{
					NodeName: nodeName,
				},
				Status: k8sv1.PodStatus{},
			}

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			vmiClient.EXPECT().Get(context.Background(), vmi.Name, &metav1.GetOptions{}).Return(vmi, nil)

			data := fmt.Sprintf(`[{ "op": "add", "path": "/status/evacuationNodeName", "value": "%s" }]`, nodeName)
			vmiClient.
				EXPECT().
				Patch(context.Background(),
					vmi.Name,
					types.JSONPatchType,
					[]byte(data),
					&metav1.PatchOptions{}).
				Return(nil, fmt.Errorf("err"))

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Code).To(Equal(int32(http.StatusTooManyRequests)))
			Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		})

		It("Should allow review requests when eviction strategy is not configured", func() {

			By("Removing eviction strategy from the VMI")
			vmi.Spec.EvictionStrategy = nil

			By("Composing a dummy admission request on a virt-launcher pod")
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: testns,
					Annotations: map[string]string{
						virtv1.DomainAnnotation: vmi.Name,
					},
					Labels: map[string]string{
						virtv1.AppLabel: "virt-launcher",
					},
				},
				Spec: k8sv1.PodSpec{
					NodeName: nodeName,
				},
				Status: k8sv1.PodStatus{},
			}

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			vmiClient.EXPECT().Get(context.Background(), vmi.Name, &metav1.GetOptions{}).Return(vmi, nil)

			data := fmt.Sprintf(`[{ "op": "add", "path": "/status/evacuationNodeName", "value": "%s" }]`, nodeName)
			vmiClient.
				EXPECT().
				Patch(context.Background(),
					vmi.Name,
					types.JSONPatchType,
					[]byte(data),
					&metav1.PatchOptions{}).
				Return(nil, fmt.Errorf("err")).AnyTimes()

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
			Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
		})

		DescribeTable("Should allow  review requests that are on a virt-launcher pod", func(dryRun bool) {
			By("Composing a dummy admission request on a virt-launcher pod")
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "testpod",
					Namespace: testns,
					Annotations: map[string]string{
						virtv1.DomainAnnotation: vmi.Name,
					},
					Labels: map[string]string{
						virtv1.AppLabel: "virt-launcher",
					},
				},
				Spec: k8sv1.PodSpec{
					NodeName: nodeName,
				},
				Status: k8sv1.PodStatus{},
			}

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
					DryRun:    &dryRun,
				},
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			if !dryRun {
				data := fmt.Sprintf(`[{ "op": "add", "path": "/status/evacuationNodeName", "value": "%s" }]`, nodeName)
				vmiClient.
					EXPECT().
					Patch(context.Background(),
						vmi.Name,
						types.JSONPatchType,
						[]byte(data),
						&metav1.PatchOptions{}).
					Return(nil, nil)
			}
			vmiClient.EXPECT().Get(context.Background(), vmi.Name, &metav1.GetOptions{}).Return(vmi, nil)

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
			actions := kubeClient.Fake.Actions()
			Expect(actions).To(HaveLen(1))
		},
			Entry("and should mark the VMI when not in dry-run mode", false),
			Entry("and should not mark the VMI when in dry-run mode", true),
		)

		Context("With EvictionStrategy cluster setting set to 'LiveMigrate'", func() {
			var vmi *virtv1.VirtualMachineInstance

			BeforeEach(func() {
				//clusterConfig := newClusterConfigWithEvictionStrategy(virtv1.EvictionStrategyLiveMigrate)
				podEvictionAdmitter = PodEvictionAdmitter{
					ClusterConfig: clusterConfig,
					VirtClient:    virtClient,
				}

				vmi = &virtv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: testns,
						Name:      "testvmi",
					},
					Status: virtv1.VirtualMachineInstanceStatus{
						Conditions: []virtv1.VirtualMachineInstanceCondition{
							{
								Type:   virtv1.VirtualMachineInstanceIsMigratable,
								Status: k8sv1.ConditionTrue,
							},
						},
					},
					Spec: virtv1.VirtualMachineInstanceSpec{},
				}
			})

			DescribeTable("Should allow review requests", func(markVMI bool, vmiEvictionStrategy virtv1.EvictionStrategy) {
				vmi.Spec.EvictionStrategy = &vmiEvictionStrategy

				By("Composing a dummy admission request on a virt-launcher pod")
				pod := &k8sv1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "testpod",
						Namespace: testns,
						Annotations: map[string]string{
							virtv1.DomainAnnotation: vmi.Name,
						},
						Labels: map[string]string{
							virtv1.AppLabel: "virt-launcher",
						},
					},
					Spec:   k8sv1.PodSpec{},
					Status: k8sv1.PodStatus{},
				}

				ar := &admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Name:      pod.Name,
						Namespace: pod.Namespace,
					},
				}

				kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
					get, ok := action.(testing.GetAction)
					Expect(ok).To(BeTrue())
					Expect(pod.Namespace).To(Equal(get.GetNamespace()))
					Expect(pod.Name).To(Equal(get.GetName()))
					return true, pod, nil
				})

				vmiClient.EXPECT().Get(context.Background(), vmi.Name, &metav1.GetOptions{}).Return(vmi, nil)

				if markVMI {
					vmiClient.EXPECT().Update(context.Background(), gomock.Any()).Return(nil, nil).AnyTimes()
				}

				resp := podEvictionAdmitter.Admit(ar)
				Expect(resp.Allowed).To(BeTrue())
				Expect(kubeClient.Fake.Actions()).To(HaveLen(1))
			},
				Entry("and should mark the VMI", true, nil),
				Entry("and should not mark the VMI", false, virtv1.EvictionStrategyNone),
			)
		})
	})

	Context("Not a virt launcher pod", func() {

		It("Should allow any review requests", func() {

			By("Composing a dummy admission request on a virt-launcher pod")
			pod := &k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: testns,
					Name:      "foo",
				},
			}

			ar := &admissionv1.AdmissionReview{
				Request: &admissionv1.AdmissionRequest{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				},
			}

			kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				get, ok := action.(testing.GetAction)
				Expect(ok).To(BeTrue())
				Expect(pod.Namespace).To(Equal(get.GetNamespace()))
				Expect(pod.Name).To(Equal(get.GetName()))
				return true, pod, nil
			})

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
		})
	})

	prepareAdmissionReview := func(vmi *virtv1.VirtualMachineInstance, nodeName string, migratable bool) *admissionv1.AdmissionReview {
		dryRun := false
		pod := &k8sv1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "testpod",
				Namespace: testns,
				Annotations: map[string]string{
					virtv1.DomainAnnotation: vmi.Name,
				},
				Labels: map[string]string{
					virtv1.AppLabel: "virt-launcher",
				},
			},
			Spec: k8sv1.PodSpec{
				NodeName: nodeName,
			},
			Status: k8sv1.PodStatus{},
		}

		ar := &admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				DryRun:    &dryRun,
			},
		}

		kubeClient.Fake.PrependReactor("get", "pods", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			get, ok := action.(testing.GetAction)
			Expect(ok).To(BeTrue())
			Expect(pod.Namespace).To(Equal(get.GetNamespace()))
			Expect(pod.Name).To(Equal(get.GetName()))
			return true, pod, nil
		})

		if migratable {
			data := fmt.Sprintf(`[{ "op": "add", "path": "/status/evacuationNodeName", "value": "%s" }]`, nodeName)
			vmiClient.
				EXPECT().
				Patch(context.Background(),
					vmi.Name,
					types.JSONPatchType,
					[]byte(data),
					&metav1.PatchOptions{}).
				Return(nil, nil)
		}
		vmiClient.EXPECT().Get(context.Background(), vmi.Name, &metav1.GetOptions{}).Return(vmi, nil)

		return ar
	}

	prepareVMI := func(nodeName string, evictionStrategy virtv1.EvictionStrategy, migratable k8sv1.ConditionStatus) *virtv1.VirtualMachineInstance {
		return &virtv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testns,
				Name:      "testvmi",
			},
			Status: virtv1.VirtualMachineInstanceStatus{
				Conditions: []virtv1.VirtualMachineInstanceCondition{
					{
						Type:   virtv1.VirtualMachineInstanceIsMigratable,
						Status: migratable,
					},
				},
				NodeName: nodeName,
			},
			Spec: virtv1.VirtualMachineInstanceSpec{
				EvictionStrategy: &evictionStrategy,
			},
		}
	}

	Context("Eviction strategy external", func() {

		externalMigrateStrategy := virtv1.EvictionStrategyExternal
		nodeName := "node01"

		It("Should allow any review request and set status.EvacuationNodeName", func() {
			By("Composing a dummy admission request on a virt-launcher pod")
			vmi := prepareVMI(nodeName, externalMigrateStrategy, k8sv1.ConditionTrue)
			ar := prepareAdmissionReview(vmi, nodeName, true)

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
			actions := kubeClient.Fake.Actions()
			Expect(actions).To(HaveLen(1))
		})
	})

	Context("Eviction strategy LiveMigrateIfPossible", func() {

		evictionStrategy := virtv1.EvictionStrategyLiveMigrateIfPossible
		nodeName := "node01"

		It("Should allow on migratable VMIs any review request and set status.EvacuationNodeName", func() {

			vmi := prepareVMI(nodeName, evictionStrategy, k8sv1.ConditionTrue)

			ar := prepareAdmissionReview(vmi, nodeName, true)

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
			actions := kubeClient.Fake.Actions()
			Expect(actions).To(HaveLen(1))
		})

		It("Should allow on non-migratable VMIs any review request and not set status.EvacuationNodeName", func() {

			vmi := prepareVMI(nodeName, evictionStrategy, k8sv1.ConditionFalse)

			ar := prepareAdmissionReview(vmi, nodeName, false)

			resp := podEvictionAdmitter.Admit(ar)
			Expect(resp.Allowed).To(BeTrue())
			actions := kubeClient.Fake.Actions()
			Expect(actions).To(HaveLen(1))
		})
	})
})
