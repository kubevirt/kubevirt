//go:build amd64

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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/time/rate"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	"kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const nodeName = "testNode"

var _ = Describe("Node-labeller ", func() {
	var nlController *NodeLabeller
	var fakeClient *fakeclientset.Clientset
	var fakeK8sClient *fake.Clientset

	Context("with node", func() {

		BeforeEach(func() {
			labels := map[string]string{"test": "test"}
			node := newNode(nodeName, labels)
			shadowNode := newShadowNode(nodeName, nil)
			fakeK8sClient = fake.NewSimpleClientset(node)
			fakeClient = fakeclientset.NewSimpleClientset(shadowNode)
			kubevirt := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						ObsoleteCPUModels: util.DefaultObsoleteCPUModels,
						MinCPUModel:       "Penryn",
					},
				},
			}

			nlController = initNodeLabeller(kubevirt, fakeClient, fakeK8sClient)
			mockQueue := testutils.NewMockWorkQueue(nlController.queue)
			nlController.queue = mockQueue

			mockQueue.ExpectAdds(1)
			nlController.queue.Add(node)
			mockQueue.Wait()
		})

		// TODO, there is issue with empty labels
		// The node labeller can't replace/update labels if there is no label
		// This is very unlikely in real Kubernetes cluster
		It("should run node-labelling", func() {
			res := nlController.execute()
			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).ToNot(BeEmpty())

			Expect(res).To(BeTrue(), "labeller should end with true result")
			Consistently(func() int {
				return nlController.queue.Len()
			}, 5*time.Second, time.Second).Should(Equal(0), "labeller should process all nodes from queue")
		})

		It("should re-queue node if node-labelling fail", func() {
			// node labelling will fail because of the Patch
			fakeK8sClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				return true, nil, fmt.Errorf("failed")
			})

			res := nlController.execute()
			Expect(res).To(BeTrue(), "labeller should end with true result")
			Eventually(func() int {
				return nlController.queue.Len()
			}, 5*time.Second, time.Second).Should(Equal(1), "node should be re-queued if labeller process fails")
		})

		It("should add host cpu model label", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelCPULabel)))
		})
		It("should add host cpu required features", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelRequiredFeaturesLabel)))
		})

		It("should add SEV label", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(HaveKey(v1.SEVLabel))
		})

		It("should add SEVES label", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(HaveKey(v1.SEVESLabel))
		})

		It("should add usable cpu model labels for the host cpu model", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.HostModelCPULabel+"Skylake-Client-IBRS"),
				HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
			))
		})

		It("should add usable cpu model labels if all required features are supported", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.CPUModelLabel+"Penryn"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Penryn"),
			))
		})

		It("should not add usable cpu model labels if some features are not suported (svm)", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			Expect(node.Labels).ToNot(SatisfyAny(
				HaveKey(v1.CPUModelLabel+"Opteron_G2"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Opteron_G2"),
			))
		})

		It("should remove not found cpu model and migration model", func() {
			node := retrieveNode(fakeK8sClient)
			node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
			node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
			node, err := fakeK8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
			))

			res := nlController.execute()
			Expect(res).To(BeTrue())

			node = retrieveNode(fakeK8sClient)
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
			))
			Expect(node.Labels).ToNot(SatisfyAny(
				HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
			))
		})

		It("should not remove not found cpu model and migration model when skip is requested", func() {
			node := retrieveNode(fakeK8sClient)
			node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
			node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
			// request skip
			node.Annotations[v1.LabellerSkipNodeAnnotation] = "true"

			node, err := fakeK8sClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
			))

			res := nlController.execute()
			Expect(res).To(BeTrue())

			node = retrieveNode(fakeK8sClient)
			Expect(node.Labels).ToNot(SatisfyAny(
				HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
			))
			Expect(node.Labels).To(SatisfyAll(
				HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
				HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
			))
		})

		It("should emit event if cpu model is obsolete", func() {
			nlController.clusterConfig.GetConfig().ObsoleteCPUModels["Skylake-Client-IBRS"] = true

			res := nlController.execute()
			Expect(res).To(BeTrue())

			recorder := nlController.recorder.(*record.FakeRecorder)
			Expect(recorder.Events).To(Receive(ContainSubstring("in ObsoleteCPUModels")))
		})

		It("should keep existing label that is not owned by node labeller", func() {
			res := nlController.execute()
			Expect(res).To(BeTrue())

			node := retrieveNode(fakeK8sClient)
			// Added in BeforeEach
			Expect(node.Labels).To(HaveKey("test"))
		})

	})
	Context("with shadow node", func() {
		someRandomLabels := map[string]string{
			"arrival": "2016",
		}

		setup := func(node *k8sv1.Node, shadownode *v1.ShadowNode) {
			virtClient := kubecli.NewMockKubevirtClient(gomock.NewController(GinkgoT()))

			fakeK8sClient = fake.NewSimpleClientset(node)
			fakeClient = fakeclientset.NewSimpleClientset(shadownode)

			kubevirt := &v1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: v1.KubeVirtSpec{
					Configuration: v1.KubeVirtConfiguration{
						ObsoleteCPUModels: util.DefaultObsoleteCPUModels,
						MinCPUModel:       "Penryn",
					},
				},
			}

			nlController = initNodeLabeller(kubevirt, fakeClient, fakeK8sClient)
			nlController.queue.Add(node)
			Eventually(nlController.queue.Len, time.Second).Should(Equal(1))
			virtClient.EXPECT().CoreV1().Return(fakeK8sClient.CoreV1()).AnyTimes()
			virtClient.EXPECT().ShadowNodeClient().Return(fakeClient.KubevirtV1().ShadowNodes()).AnyTimes()
		}

		It("should keep cpu-model labels", func() {
			labels := map[string]string{
				v1.CPUModelLabel + "Penryn": "true",
			}
			setupNode := newNode(nodeName, someRandomLabels)
			setupShadowNode := newShadowNode(nodeName, labels)

			setup(setupNode, setupShadowNode)
			res := nlController.execute()
			Expect(res).To(BeTrue())

			shadowNode := retrieveShadowNode(fakeClient)
			Expect(shadowNode.Labels).To(
				HaveKeyWithValue(v1.CPUModelLabel+"Penryn", "true"),
			)
		})

		It("should keep ksm labels", func() {
			labels := map[string]string{
				v1.KSMEnabledLabel: "true",
			}
			setupNode := newNode(nodeName, someRandomLabels)
			setupShadowNode := newShadowNode(nodeName, labels)

			setup(setupNode, setupShadowNode)
			res := nlController.execute()
			Expect(res).To(BeTrue())

			shadowNode := retrieveShadowNode(fakeClient)
			Expect(shadowNode.Labels).To(
				HaveKeyWithValue(v1.KSMEnabledLabel, "true"),
			)
		})

		It("should keep hearbeat labels and annotations", func() {
			now, err := json.Marshal(metav1.Now())
			Expect(err).ToNot(HaveOccurred())

			labels := map[string]string{
				v1.NodeSchedulable: "true",
				v1.CPUManager:      "true",
			}
			annotations := map[string]string{
				v1.VirtHandlerHeartbeat: string(now),
			}
			setupNode := &k8sv1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
					Labels:      someRandomLabels,
					Name:        nodeName,
				},
				Spec: k8sv1.NodeSpec{},
			}
			setupShadowNode := &v1.ShadowNode{ObjectMeta: metav1.ObjectMeta{Name: nodeName, Labels: labels, Annotations: annotations}}

			setup(setupNode, setupShadowNode)
			res := nlController.execute()
			Expect(res).To(BeTrue())
			shadowNode := retrieveShadowNode(fakeClient)
			Expect(shadowNode.Labels).To(SatisfyAll(
				HaveKeyWithValue(v1.NodeSchedulable, "true"),
				HaveKeyWithValue(v1.CPUManager, "true"),
			))
			Expect(shadowNode.Annotations).To(
				HaveKeyWithValue(v1.VirtHandlerHeartbeat, string(now)),
			)
		})

		It("should remove irrelevant label", func() {
			labels := map[string]string{
				v1.CPUModelLabel: "true",
			}
			setupNode := newNode(nodeName, labels)
			setupShadowNode := newShadowNode(nodeName, labels)

			setup(setupNode, setupShadowNode)
			res := nlController.execute()
			Expect(res).To(BeTrue())

			shadowNode := retrieveShadowNode(fakeClient)
			Expect(shadowNode.Labels).To(SatisfyAll(
				Not(HaveKeyWithValue(v1.CPUModelLabel, "true")),
				HaveKeyWithValue(v1.CPUModelLabel+"Penryn", "true"),
			))
		})

		It("should update relevant label", func() {
			labels := map[string]string{
				v1.CPUModelLabel + "Opteron_G2": "true",
			}
			setupNode := newNode(nodeName, labels)
			setupShadowNode := newShadowNode(nodeName, labels)

			setup(setupNode, setupShadowNode)
			res := nlController.execute()
			Expect(res).To(BeTrue())

			shadowNode := retrieveShadowNode(fakeClient)
			Expect(shadowNode.Labels).To(SatisfyAll(
				Not(HaveKeyWithValue(v1.CPUModelLabel+"Opteron_G2", "true")),
				HaveKeyWithValue(v1.CPUModelLabel+"Penryn", "true"),
			))
		})
	})
})

func newNode(name string, labels map[string]string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      labels,
			Name:        name,
		},
		Spec: k8sv1.NodeSpec{},
	}
}

func retrieveNode(fakeK8sClient *fake.Clientset) *k8sv1.Node {
	node, err := fakeK8sClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return node
}

func newShadowNode(name string, labels map[string]string) *v1.ShadowNode {
	return &v1.ShadowNode{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      labels,
			Name:        name,
		},
	}
}

func retrieveShadowNode(fakeClient *fakeclientset.Clientset) *v1.ShadowNode {
	shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return shadowNode
}

func initNodeLabeller(kubevirt *v1.KubeVirt, fakeClient *fakeclientset.Clientset, fakeK8sClient *fake.Clientset) *NodeLabeller {
	config, _, _ := testutils.NewFakeClusterConfigUsingKV(kubevirt)
	recorder := record.NewFakeRecorder(100)
	recorder.IncludeObject = true

	var err error
	nlController, err := newNodeLabeller(config, fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), nodeName, "testdata", recorder)
	Expect(err).ToNot(HaveOccurred())

	// Override queue to have no rate limiting because we only execute "execute" once
	// and want to assert without wait time
	nlController.queue = workqueue.NewRateLimitingQueue(&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)})

	return nlController
}
