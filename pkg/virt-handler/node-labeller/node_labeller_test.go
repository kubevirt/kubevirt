//go:build amd64 || s390x

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
 * Copyright The KubeVirt Authors.
 *
 */

package nodelabeller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"libvirt.org/go/libvirtxml"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	util "kubevirt.io/kubevirt/pkg/virt-handler/node-labeller/util"
)

const nodeName = "testNode"

var _ = Describe("Node-labeller ", func() {
	var nlController *NodeLabeller
	var kubeClient *fake.Clientset
	var cpuCounter *libvirtxml.CapsHostCPUCounter
	var guestsCaps []libvirtxml.CapsGuest

	initNodeLabeller := func(kubevirt *v1.KubeVirt) {
		config, _, _ := testutils.NewFakeClusterConfigUsingKV(kubevirt)
		recorder := record.NewFakeRecorder(100)
		recorder.IncludeObject = true

		var err error
		nlController, err = newNodeLabeller(config, kubeClient.CoreV1().Nodes(), nodeName, "testdata", recorder, cpuCounter, guestsCaps)
		Expect(err).ToNot(HaveOccurred())
	}

	BeforeEach(func() {
		cpuCounter = &libvirtxml.CapsHostCPUCounter{
			Name:      "tsc",
			Frequency: 4008012000,
			Scaling:   "no",
		}

		guestsCaps = []libvirtxml.CapsGuest{
			{
				OSType: "test",
				Arch: libvirtxml.CapsGuestArch{
					Machines: []libvirtxml.CapsGuestMachine{
						{Name: "testmachine"},
					},
				},
			},
		}

		node := newNode(nodeName)
		kubeClient = fake.NewSimpleClientset(node)
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
		mockQueue := testutils.NewMockWorkQueue(nlController.queue)
		nlController.queue = mockQueue

		mockQueue.ExpectAdds(1)
		nlController.queue.Add(node.Name)
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
		Expect(nlController.queue.Len()).To(BeZero(), "labeller should process all nodes from queue")
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

	It("should add supported machine type labels", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SupportedMachineTypeLabel + "testmachine"))
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

	It("should not add SecureExecution label", func() {
		nlController.volumePath = "testdata/s390x"
		Expect(nlController.loadAll()).Should(Succeed())

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(Not(HaveKey(v1.SecureExecutionLabel)))
	})

	It("should  add SecureExecution label", func() {
		nlController.domCapabilitiesFileName = "s390x/domcapabilities_s390-pv.xml"
		Expect(nlController.loadAll()).Should(Succeed())

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.SecureExecutionLabel))
	})

	It("should add TDX label", func() {
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(HaveKey(v1.TDXLabel))
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

	DescribeTable("should add cpu tsc labels if tsc counter exists, its name is tsc and according to scaling value", func(scaling, result string) {
		nlController.cpuCounter.Scaling = scaling
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			HaveKeyWithValue(v1.CPUTimerLabel+"tsc-frequency", fmt.Sprintf("%d", cpuCounter.Frequency)),
			HaveKeyWithValue(v1.CPUTimerLabel+"tsc-scalable", result),
		))
	},
		Entry("scaling is set to no", "no", "false"),
		Entry("scaling is set to yes", "yes", "true"),
	)

	DescribeTable("should not add cpu tsc labels", func(counter *libvirtxml.CapsHostCPUCounter) {
		nlController.cpuCounter = counter
		res := nlController.execute()
		Expect(res).To(BeTrue())

		node := retrieveNode(kubeClient)
		Expect(node.Labels).To(SatisfyAll(
			Not(HaveKey(v1.CPUTimerLabel+"tsc-frequency")),
			Not(HaveKey(v1.CPUTimerLabel+"tsc-scalable")),
		))
	},
		Entry("cpuCounter name is not tsc", &libvirtxml.CapsHostCPUCounter{}),
		Entry("cpuCounter is nil", nil),
	)

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

	It("should not remove not found cpu model and migration model when skip is requested", func() {
		node := retrieveNode(kubeClient)
		node.Labels[v1.CPUModelLabel+"Cascadelake-Server"] = "true"
		node.Labels[v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"] = "true"
		// request skip
		node.Annotations[v1.LabellerSkipNodeAnnotation] = "true"

		node, err := kubeClient.CoreV1().Nodes().Update(context.TODO(), node, metav1.UpdateOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(SatisfyAll(
			HaveKey(v1.CPUModelLabel+"Cascadelake-Server"),
			HaveKey(v1.SupportedHostModelMigrationCPU+"Cascadelake-Server"),
		))

		res := nlController.execute()
		Expect(res).To(BeTrue())

		node = retrieveNode(kubeClient)
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

		node := retrieveNode(kubeClient)
		// Added in BeforeEach
		Expect(node.Labels).To(HaveKey("INeedToBeHere"))
	})

	DescribeTable("should add machine type labels", func(machines []libvirtxml.CapsGuestMachine, arch string) {
		guestsCaps[0].Arch.Machines = machines

		initNodeLabeller(&v1.KubeVirt{})
		nlController.arch = newArchLabeller(arch)
		mockQueue := testutils.NewMockWorkQueue(nlController.queue)
		nlController.queue = mockQueue

		mockQueue.ExpectAdds(1)
		nlController.queue.Add(nodeName)
		mockQueue.Wait()

		res := nlController.execute()
		Expect(res).To(BeTrue(), "labeller should complete successfully")

		node := retrieveNode(kubeClient)

		for _, machine := range machines {
			expectedLabelKey := v1.SupportedMachineTypeLabel + machine.Name
			Expect(node.Labels).To(HaveKey(expectedLabelKey), "expected machine type label %s to be present", expectedLabelKey)
		}
	},
		Entry("for amd64", []libvirtxml.CapsGuestMachine{{Name: "q35"}, {Name: "q35-rhel9.6.0"}}, amd64),
		Entry("for arm64", []libvirtxml.CapsGuestMachine{{Name: "virt"}, {Name: "virt-rhel9.6.0"}}, arm64),
	)

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
