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

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"golang.org/x/time/rate"
	k8sv1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	v1 "kubevirt.io/api/core/v1"
	kubevirtfake "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

var _ = Describe("Node-labeller ", func() {
	const hostName = "testNode"

	var (
		nlController   *NodeLabeller
		virtClient     *kubecli.MockKubevirtClient
		virtFakeClient *kubevirtfake.Clientset
		stop           chan struct{}
		ctrl           *gomock.Controller
		kubeClient     *fake.Clientset
		mockQueue      *testutils.MockWorkQueue
		config         *virtconfig.ClusterConfig
		recorder       *record.FakeRecorder
	)

	getNodes := func(k8sClient *fake.Clientset, kubevirtClient *kubevirtfake.Clientset) (*k8sv1.Node, *v1.ShadowNode) {
		node, err := k8sClient.CoreV1().Nodes().Get(context.TODO(), hostName, metav1.GetOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		shadowNode, err := kubevirtClient.KubevirtV1().ShadowNodes().Get(context.TODO(), hostName, metav1.GetOptions{})
		ExpectWithOffset(2, err).ToNot(HaveOccurred())

		return node, shadowNode
	}

	expectLabels := func(k8sClient *fake.Clientset, kubevirtClient *kubevirtfake.Clientset, matcher types.GomegaMatcher) {
		node, shadowNode := getNodes(k8sClient, kubevirtClient)
		ExpectWithOffset(1, node.Labels).To(matcher, "Node labels")
		ExpectWithOffset(1, shadowNode.Labels).To(matcher, "Shadownode labels")
	}

	addNode := func(node *k8sv1.Node) {
		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node)
		mockQueue.Wait()
	}

	initNodeLabeller := func(kubevirt *v1.KubeVirt) {
		var err error
		config, _, _ = testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		nlController, err = newNodeLabeller(config, virtClient, hostName, k8sv1.NamespaceDefault, "testdata", recorder)
		Expect(err).ToNot(HaveOccurred())

		// Override queue to have no rate limiting because we only execute "execute" once
		// and want to assert without wait time
		nlController.queue = workqueue.NewRateLimitingQueue(&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(10), 100)})
		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		nlController.queue = mockQueue
	}

	BeforeEach(func() {
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())
		labels := map[string]string{"test": "test"}
		annotations := map[string]string{"test": "test"}
		node := newNode(hostName, labels, annotations)

		kubeClient = fake.NewSimpleClientset(node)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtFakeClient = kubevirtfake.NewSimpleClientset(&v1.ShadowNode{ObjectMeta: *node.ObjectMeta.DeepCopy()})

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
		addNode(node)

		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().ShadowNodeClient().Return(virtFakeClient.KubevirtV1().ShadowNodes()).AnyTimes()

	})

	It("should run node-labelling", func() {
		testutils.ExpectNodePatch(kubeClient)
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Expect(nlController.queue.Len()).Should(Equal(0), "labeller should process all nodes from queue")
	})

	It("should re-queue node if node-labelling fail on shadownode", func() {
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, nil
		})
		virtFakeClient.PrependReactor("patch", "shadownodes", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, fmt.Errorf("problem")
		})
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Expect(nlController.queue.Len()).Should(Equal(1), "node should be re-queued if labeller process fails")
	})
	It("should re-queue node if node-labelling fail", func() {
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			return true, nil, fmt.Errorf("problem")
		})
		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should end with true result")
		Expect(nlController.queue.Len()).Should(Equal(1), "node should be re-queued if labeller process fails")
	})

	It("should add host cpu model label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, HaveKeyWithValue(HavePrefix(v1.HostModelCPULabel), "true"))
	})
	It("should add host cpu required features", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, HaveKeyWithValue(HavePrefix(v1.HostModelRequiredFeaturesLabel), "true"))
	})

	It("should add SEV label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, HaveKeyWithValue(v1.SEVLabel, BeEmpty()))
	})

	It("should add SEVES label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, HaveKeyWithValue(v1.SEVESLabel, BeEmpty()))
	})

	It("should add usable cpu model labels for the host cpu model", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, SatisfyAll(
			HaveKeyWithValue(v1.HostModelCPULabel+"Skylake-Client-IBRS", "true"),
			HaveKeyWithValue(v1.CPUModelLabel+"Skylake-Client-IBRS", "true"),
			HaveKeyWithValue(v1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS", "true"),
		))

	})

	It("should add usable cpu model labels if all required features are supported", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, SatisfyAll(
			HaveKeyWithValue(v1.CPUModelLabel+"Penryn", "true"),
			HaveKeyWithValue(v1.SupportedHostModelMigrationCPU+"Penryn", "true"),
		))

	})

	It("should not add usable cpu model labels if some features are not suported (svm)", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())
		expectLabels(kubeClient, virtFakeClient, Not(SatisfyAny(
			HaveKeyWithValue(v1.CPUModelLabel+"Opteron_G2", "true"),
			HaveKeyWithValue(v1.SupportedHostModelMigrationCPU+"Opteron_G2", "true"),
		)))

	})

	AfterEach(func() {
		close(stop)
	})
})

func newNode(name string, labels, annotations map[string]string) *k8sv1.Node {
	return &k8sv1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: annotations,
			Labels:      labels,
			Name:        name,
		},
		Spec: k8sv1.NodeSpec{},
	}
}
