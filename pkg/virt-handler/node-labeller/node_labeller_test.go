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
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"

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
	var recorder *record.FakeRecorder

	addNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node)
		addedNode = node
		mockQueue.Wait()
	}

	initNodeLabeller := func(kubevirt *kubevirtv1.KubeVirt, nodeLabels, nodeAnnotations map[string]string) {
		var err error
		config, _, _ = testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		nlController, err = newNodeLabeller(config, virtClient, "testNode", k8sv1.NamespaceDefault, "testdata", recorder)
		Expect(err).ToNot(HaveOccurred())

		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		nlController.queue = mockQueue
		addNode(newNode("testNode", nodeLabels, nodeAnnotations))
	}

	BeforeEach(func() {
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

		initNodeLabeller(kv, make(map[string]string), make(map[string]string))
	})

	It("should run node-labelling", func() {
		testutils.ExpectNodePatch(kubeClient)
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
		testutils.ExpectNodePatch(kubeClient, kubevirtv1.HostModelCPULabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})
	It("should add host cpu required features", func() {
		testutils.ExpectNodePatch(kubeClient, kubevirtv1.HostModelRequiredFeaturesLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add SEV label", func() {
		testutils.ExpectNodePatch(kubeClient, kubevirtv1.SEVLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add SEVES label", func() {
		testutils.ExpectNodePatch(kubeClient, kubevirtv1.SEVESLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add usable cpu model labels for the host cpu model", func() {
		testutils.ExpectNodePatch(kubeClient,
			kubevirtv1.HostModelCPULabel+"Skylake-Client-IBRS",
			kubevirtv1.CPUModelLabel+"Skylake-Client-IBRS",
			kubevirtv1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add usable cpu model labels if all required features are supported", func() {
		testutils.ExpectNodePatch(kubeClient,
			kubevirtv1.CPUModelLabel+"Penryn",
			kubevirtv1.SupportedHostModelMigrationCPU+"Penryn",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should not add usable cpu model labels if some features are not suported (svm)", func() {
		testutils.DoNotExpectNodePatch(kubeClient,
			kubevirtv1.CPUModelLabel+"Opteron_G2",
			kubevirtv1.SupportedHostModelMigrationCPU+"Opteron_G2",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	AfterEach(func() {
		close(stop)
	})
})

func newNode(name string, labels, annotations map[string]string) *v1.Node {
	return &v1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
			Name:        name,
		},
		Spec: v1.NodeSpec{},
	}
}
