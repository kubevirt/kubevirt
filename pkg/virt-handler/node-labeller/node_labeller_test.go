// +build amd64

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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
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
	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	var stop chan struct{}
	var ctrl *gomock.Controller
	var kubeClient *fake.Clientset
	var mockQueue *testutils.MockWorkQueue
	var config *virtconfig.ClusterConfig
	var addedNode *v1.Node

	addNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node)
		addedNode = node
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

	BeforeEach(func() {
		var err error
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())

		kubeClient = fake.NewSimpleClientset()
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		kubeClient.Fake.PrependReactor("get", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, addedNode, nil
		})

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		kv := &kubevirtv1.KubeVirt{
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

		nlController, err = newNodeLabeller(config, virtClient, "testNode", k8sv1.NamespaceDefault, "testdata")
		Expect(err).ToNot(HaveOccurred())

		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		nlController.queue = mockQueue
		addNode(newNode("testNode"))
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
