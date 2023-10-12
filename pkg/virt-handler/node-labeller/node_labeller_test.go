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
	"os"
	"time"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"

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
	var customKSMFilePath = "fake_path"

	addNode := func(node *v1.Node) {
		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node)
		addedNode = node
		mockQueue.Wait()
	}

	expectPatch := func(expect bool, expectedPatches ...string) {
		kubeClient.Fake.PrependReactor("patch", "nodes", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
			patch, ok := action.(testing.PatchAction)
			Expect(ok).To(BeTrue())
			for _, expectedPatch := range expectedPatches {
				if expect {
					Expect(string(patch.GetPatch())).To(ContainSubstring(expectedPatch))
				} else {
					Expect(string(patch.GetPatch())).ToNot(ContainSubstring(expectedPatch))
				}
			}
			return true, nil, nil
		})
	}

	expectNodePatch := func(expectedPatches ...string) {
		expectPatch(true, expectedPatches...)
	}

	doNotExpectNodePatch := func(expectedPatches ...string) {
		expectPatch(false, expectedPatches...)
	}

	createCustomKSMFile := func(value string) {
		customKSMFile, err := os.CreateTemp("", "mock_ksm_run")
		Expect(err).ToNot(HaveOccurred())
		defer customKSMFile.Close()
		_, err = customKSMFile.WriteString(value)
		Expect(err).ToNot(HaveOccurred())
		customKSMFilePath = customKSMFile.Name()
	}

	initNodeLabeller := func(kubevirt *kubevirtv1.KubeVirt, ksmNodeValue string, nodeLabels, nodeAnnotations map[string]string) {
		var err error
		config, _, _ = testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder = record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		createCustomKSMFile(fmt.Sprint(ksmNodeValue))
		nlController, err = newNodeLabeller(config, virtClient, "testNode", k8sv1.NamespaceDefault, "testdata", recorder, customKSMFilePath)
		Expect(err).ToNot(HaveOccurred())

		mockQueue = testutils.NewMockWorkQueue(nlController.queue)

		nlController.queue = mockQueue
		addNode(newNode("testNode", nodeLabels, nodeAnnotations))
	}

	AfterEach(func() {
		diskutils.RemoveFilesIfExist(customKSMFilePath)
	})

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

		initNodeLabeller(kv, "1\n", make(map[string]string), make(map[string]string))
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

	It("should add SEV label", func() {
		expectNodePatch(kubevirtv1.SEVLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add SEVES label", func() {
		expectNodePatch(kubevirtv1.SEVESLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add usable cpu model labels for the host cpu model", func() {
		expectNodePatch(
			kubevirtv1.HostModelCPULabel+"Skylake-Client-IBRS",
			kubevirtv1.CPUModelLabel+"Skylake-Client-IBRS",
			kubevirtv1.SupportedHostModelMigrationCPU+"Skylake-Client-IBRS",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add usable cpu model labels if all required features are supported", func() {
		expectNodePatch(
			kubevirtv1.CPUModelLabel+"Penryn",
			kubevirtv1.SupportedHostModelMigrationCPU+"Penryn",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should not add usable cpu model labels if some features are not suported (svm)", func() {
		doNotExpectNodePatch(
			kubevirtv1.CPUModelLabel+"Opteron_G2",
			kubevirtv1.SupportedHostModelMigrationCPU+"Opteron_G2",
		)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	It("should add KSM label", func() {
		expectNodePatch(kubevirtv1.KSMEnabledLabel)
		res := nlController.execute()
		Expect(res).To(BeTrue())
	})

	Describe(", when ksmConfiguration is provided,", func() {
		var kv *kubevirtv1.KubeVirt
		BeforeEach(func() {
			kv = &kubevirtv1.KubeVirt{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kubevirt",
					Namespace: "kubevirt",
				},
				Spec: kubevirtv1.KubeVirtSpec{
					Configuration: kubevirtv1.KubeVirtConfiguration{
						KSMConfiguration: &kubevirtv1.KSMConfiguration{
							NodeLabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{
									"test_label": "true",
								},
							},
						},
					},
				},
			}
		})

		DescribeTable("should", func(initialKsmValue string, nodeLabels, nodeAnnotations map[string]string, expectedNodePatch []string, expectedKsmValue string) {
			initNodeLabeller(kv, initialKsmValue, nodeLabels, nodeAnnotations)
			expectNodePatch(expectedNodePatch...)
			res := nlController.execute()
			Expect(res).To(BeTrue())
			Expect(os.ReadFile(customKSMFilePath)).To(BeEquivalentTo([]byte(expectedKsmValue)))
		},
			Entry("enable ksm if the node labels match ksmConfiguration.nodeLabelSelector",
				"0\n", map[string]string{"test_label": "true"}, make(map[string]string),
				[]string{kubevirtv1.KSMEnabledLabel, kubevirtv1.KSMHandlerManagedAnnotation}, "1\n",
			),
			Entry("disable ksm if the node labels does not match ksmConfiguration.nodeLabelSelector and the node has the KSMHandlerManagedAnnotation annotation",
				"1\n", map[string]string{"test_label": "false"}, map[string]string{kubevirtv1.KSMHandlerManagedAnnotation: "true"},
				[]string{kubevirtv1.KSMHandlerManagedAnnotation}, "0\n",
			),
			Entry("not change ksm if the node labels does not match ksmConfiguration.nodeLabelSelector and the node does not have the KSMHandlerManagedAnnotation annotation",
				"1\n", map[string]string{"test_label": "false"}, make(map[string]string),
				nil, "1\n",
			),
		)
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
