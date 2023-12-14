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
	"fmt"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const nodeName = "testNode"

var _ = Describe("Node-labeller ", func() {
	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	var ctrl *gomock.Controller
	var kubeClient *fake.Clientset
	var mockQueue *testutils.MockWorkQueue
	var recorder *record.FakeRecorder

	initNodeLabeller := func(kubevirt *v1.KubeVirt) {
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		var err error
		nlController, err = newNodeLabeller(config, virtClient, nodeName, k8sv1.NamespaceDefault, "testdata", recorder)
		Expect(err).ToNot(HaveOccurred())

		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		nlController.queue = mockQueue
	}

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		node := newNode(nodeName)

		kubeClient = fake.NewSimpleClientset(node)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()

		initNodeLabeller(&v1.KubeVirt{
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
		})
		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node)
		mockQueue.Wait()
	})

	// TODO, there is issue with empty labels
	// The node labeller can't replace/update labels if there is no label
	// This is very unlikely in real Kubernetes cluster
	It("should run node-labelling", func() {
		res := nlController.execute()
		node := retrieveNode(kubeClient)
		Expect(node.Labels).ToNot(BeEmpty())

		Expect(res).To(BeTrue(), "labeller should end with true result")
		Consistently(func() int {
			return nlController.queue.Len()
		}, 5*time.Second, time.Second).Should(Equal(0), "labeller should process all nodes from queue")
	})

	It("should re-queue node if node-labelling fail", func() {
		// node labelling will fail because of the Patch
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
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

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelCPULabel)))
	})
	It("should add host cpu required features", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(HavePrefix(v1.HostModelRequiredFeaturesLabel)))
	})

	It("should add SEV label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SEVLabel))
	})

	It("should add SEVES label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SEVESLabel))
	})

	It("should add usable cpu model labels for the host cpu model", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.HostModelCPULabel+"Skylake-Client-IBRS"),
			HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
		))
	})

	It("should add usable cpu model labels if all required features are supported", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Penryn"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Penryn"),
		))
	})

	It("should not add usable cpu model labels if some features are not suported (svm)", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).ToNot(SatisfyAny(
			HaveKey(v1.CPUModelLabel+"Opteron_G2"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Opteron_G2"),
		))
	})

	It("should remove not found cpu model and migration model", func() {
		node := retrieveNode(kubeClient)
		node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
		node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
		node, err := kubeClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node = retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Skylake-Client-IBRS"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS"),
		))
		Expect(node.Labels).ToNot(SatisfyAny(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))
	})
})

func newNode(name string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{},
			Labels:      map[string]string{"INeedToBeHere": "trustme"},
			Name:        name,
		},
		Spec: k8sv1.NodeSpec{},
	}
}

func retrieveNode(kubeClient *fake.Clientset) *k8sv1.Node {
	node, err := kubeClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return node
}
