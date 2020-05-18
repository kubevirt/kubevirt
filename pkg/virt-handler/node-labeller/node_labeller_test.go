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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package nodelabeller

import (
	"os"
	"strings"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	framework "k8s.io/client-go/tools/cache/testing"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

var _ = Describe("Node-labeller ", func() {
	var nlController *NodeLabeller
	var virtClient *kubecli.MockKubevirtClient
	var stop chan struct{}
	var ctrl *gomock.Controller
	var kubeClient *fake.Clientset
	var mockQueue *testutils.MockWorkQueue
	var nodeInformer cache.SharedIndexInformer
	var nodeSource *framework.FakeControllerSource
	var configMapInformer cache.SharedIndexInformer

	syncCaches := func(stop chan struct{}) {
		go nodeInformer.Run(stop)
		go configMapInformer.Run(stop)
		Expect(cache.WaitForCacheSync(stop, nodeInformer.HasSynced, configMapInformer.HasSynced)).To(BeTrue())
	}

	addNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nodeSource.Add(node)
		nlController.Queue.Add(node)

		mockQueue.Wait()
	}

	BeforeEach(func() {
		os.MkdirAll(nodeLabellerVolumePath+"/cpu_map", 0777)
		stop = make(chan struct{})
		ctrl = gomock.NewController(GinkgoT())

		kubeClient = fake.NewSimpleClientset()

		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		virtClient.EXPECT().CoreV1().Return(kubeClient.CoreV1()).AnyTimes()
		kubeClient.CoreV1().Nodes().Create(newNode("testNode"))

		nodeInformer, nodeSource = testutils.NewFakeInformerFor(&v1.Node{})
		configMapInformer, _ = testutils.NewFakeInformerFor(&v1.ConfigMap{})

		clusterConfig, _, _ := testutils.NewFakeClusterConfig(&v1.ConfigMap{
			Data: map[string]string{
				virtconfig.ObsoleteCPUsKey: "486, pentium, pentium2, pentium3, pentiumpro, coreduo, n270, core2duo, Conroe, athlon, phenom",
				virtconfig.MinCPUKey:       "Penryn",
			},
		})
		nlController = NewNodeLabeller(&device_manager.DeviceController{}, clusterConfig, nodeInformer, configMapInformer, virtClient, "testNode", k8sv1.NamespaceDefault)
		mockQueue = testutils.NewMockWorkQueue(nlController.Queue)
		nlController.Queue = mockQueue

		syncCaches(stop)

	})

	It("should run node-labelling", func() {
		addNode(newNode("testNode"))
		kubeClient.Fake.PrependReactor("*", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			update, _ := action.(testing.PatchAction)
			Expect(update.GetName()).To(Equal("testNode"), "names should equal")
			containCorrectLabel := strings.Contains(string(update.GetPatch()), "Penryn")
			Expect(containCorrectLabel).To(Equal(true), "labels should contain cpu model")
			return true, nil, nil
		})

		prepareFileDomCapabilities()
		prepareFilesFeatures()

		nlController.Execute()
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
