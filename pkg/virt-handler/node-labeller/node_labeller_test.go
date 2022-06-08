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
	"fmt"
	"time"

	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	kubevirtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var _ = Describe("Node-labeller ", func() {
	const nodeName = "testNode"

	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	var stop chan struct{}
	var ctrl *gomock.Controller
	var kubeClient *fake.Clientset
	var mockQueue *testutils.MockWorkQueue
	var config *virtconfig.ClusterConfig
	var addedNode *v1.Node
	var nodeInformer cache.SharedIndexInformer
	var nodeSource *framework.FakeControllerSource
	var kvInformer cache.SharedIndexInformer
	var kvSource *framework.FakeControllerSource
	var kv *kubevirtv1.KubeVirt

	addNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Add(node)
		mockQueue.Wait()
		addedNode = node
	}

	modifyNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Modify(node)
		mockQueue.Wait()
		addedNode = node
	}

	addKubevirtCR := func(kv *kubevirtv1.KubeVirt) {
		mockQueue.ExpectAdds(1)
		kvSource.Add(kv)
		mockQueue.Wait()
	}

	expectNodePatch := func(expectedPatches ...string) {
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			for _, expectedPatch := range expectedPatches {
				Expect(string(patch.GetPatch())).To(ContainSubstring(expectedPatch))
			}
			return true, nil, nil
		})
	}

	emptyQueue := func() {
		By("Running node labeller once")
		expectNodePatch()
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")

		By("Making sure queue is empty")
		Expect(nlController.queue.Len()).To(Equal(0), "queue is expected to be empty")
	}

	syncCaches := func(stop chan struct{}) {
		go nodeInformer.Run(stop)
		go kvInformer.Run(stop)

		Expect(
			cache.WaitForCacheSync(stop,
				nodeInformer.HasSynced,
				kvInformer.HasSynced,
			),
		).To(BeTrue())
	}

	BeforeEach(func() {
		var err error
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&v1.Node{})
		kvInformer, kvSource = testutils.NewFakeInformerFor(&kubevirtv1.KubeVirt{})

		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, addedNode, nil
		})

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		kv = &kubevirtv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt",
				Namespace: "kubevirt",
			},
			Spec: kubevirtv1.KubeVirtSpec{
				Configuration: kubevirtv1.KubeVirtConfiguration{
					ObsoleteCPUModels: util.DefaultObsoleteCPUModels,
					MinCPUModel:       "Penryn",
				},
			},
		}

		config, _, _ = testutils.NewFakeClusterConfigUsingKV(kv)

		nlController, err = newNodeLabeller(config, virtClient, nodeName, k8sv1.NamespaceDefault, nodeInformer, kvInformer, "testdata")
		Expect(err).ToNot(HaveOccurred())

		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		syncCaches(stop)

		nlController.queue = mockQueue
		addNode(newNode(nodeName))

		// Make sure that all unexpected calls to kubeClient will fail
		kubeClient.Fake.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			Expect(action).To(BeNil())
			return true, nil, nil
		})
	})

	It("should run node-labelling", func() {
		expectNodePatch()
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Consistently(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(0), "labeller should process all nodes from queue")
	})

	It("should re-queue node if node-labelling fail", func() {
		// node labelling will fail because the Patch executed inside execute() will fail due to missed Reactor
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			_, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())

			return true, nil, fmt.Errorf("fake error")
		})
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Eventually(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(1), "node should be re-queued if labeller process fails")
	})

	It("should add host cpu model label", func() {
		expectNodePatch(kubevirtv1.HostModelCPULabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})
	It("should add host cpu required features", func() {
		expectNodePatch(kubevirtv1.HostModelRequiredFeaturesLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add SEV label", func() {
		expectNodePatch(kubevirtv1.SEVLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	DescribeTable("should re-enqueue node if its updating its", func(updateNode func(node *v1.Node)) {
		emptyQueue()

		updatedNode := addedNode.DeepCopy()
		updateNode(updatedNode)
		modifyNode(updatedNode)

		Eventually(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(1), "queue is expected to with length of 1")
	},
		Entry("labels", func(node *v1.Node) { node.Labels["new-key"] = "new-value" }),
		Entry("annotations", func(node *v1.Node) { node.Annotations["new-key"] = "new-value" }),
	)

	It("should not re-enqueue node if it updated something other than labels / annotations", func() {
		emptyQueue()

		updatedNode := addedNode.DeepCopy()
		updatedNode.Spec.PodCIDR = "fake"
		mockQueue.ExpectAdds(1)
		nodeSource.Modify(updatedNode)

		Consistently(func() int {
			return nlController.queue.Len()
		}, 3*time.Second, time.Second).Should(Equal(0), "queue is expected to be empty")
	})

	It("should re-enqueue node if KubvevirtCR updates Configuration", func() {
		emptyQueue()

		updatedKv := kv.DeepCopy()
		updatedKv.Spec.Configuration.OVMFPath = "fake-path"
		addKubevirtCR(updatedKv)

		Eventually(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(1), "queue is expected to with length of 1")
	})

	It("should not re-enqueue node if KubvevirtCR updates anything other than Configuration", func() {
		emptyQueue()

		err := kvInformer.GetStore().Add(kv)
		Expect(err).ShouldNot(HaveOccurred())

		updatedKv := kv.DeepCopy()
		updatedKv.Spec.ImageTag = "fake-tag"
		mockQueue.ExpectAdds(1)
		kvSource.Modify(updatedKv)

		Consistently(func() int {
			return nlController.queue.Len()
		}, 3*time.Second, time.Second).Should(Equal(0), "queue is expected to be empty")
	})

	AfterEach(func() {
		close(stop)
	})
})

func newNode(name string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: make(map[string]string),
			Labels:      make(map[string]string),
			Name:        name,
		},
		Spec: v1.NodeSpec{},
	}
}
