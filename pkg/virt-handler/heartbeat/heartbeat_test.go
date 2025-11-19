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

package heartbeat

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	virtv1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const (
	cpu_manager_static_path = "testdata/cpu_manager_state_static"
	cpu_manager_none_path   = "testdata/cpu_manager_state_none"
)

var _ = Describe("Heartbeat", func() {

	var node *v1.Node
	var fakeClient *fake.Clientset

	BeforeEach(func() {
		node = &v1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mynode",
			},
		}
		fakeClient = fake.NewSimpleClientset(node)
	})
	Context("upon finishing", func() {
		It("should set the node to not schedulable", func() {
			heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController(true), config(), "mynode")
			stopChan := make(chan struct{})
			done := heartbeat.Run(30*time.Second, stopChan)
			Eventually(func() map[string]string {
				node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return node.Labels
			}).Should(And(
				HaveKeyWithValue(virtv1.NodeSchedulable, "true"),
			))
			close(stopChan)
			<-done
			node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Labels).To(HaveKeyWithValue(virtv1.NodeSchedulable, "false"))
		})
	})

	DescribeTable("with cpumanager featuregate should set the node to", func(deviceController device_manager.DeviceControllerInterface,
		cpuManagerPaths []string, expectedSchedulableValue string, expectedCPUManagerValue string) {
		heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController, config(), "mynode")
		heartbeat.cpuManagerPaths = cpuManagerPaths
		heartbeat.do()
		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.NodeSchedulable, expectedSchedulableValue))
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.DeprecatedCPUManager, expectedCPUManagerValue))
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.CPUManager, expectedCPUManagerValue))
	},
		Entry("not schedulable and no cpu manager with no cpu manager file and device plugins are not initialized",
			deviceController(false),
			[]string{"non/existent/cpumanager/statefile"},
			"false",
			"false",
		),
		Entry("schedulable and no cpu manager with no cpu manager file and plugins are not initialized",
			deviceController(true),
			[]string{"non/existent/cpumanager/statefile"},
			"true",
			"false",
		),
		Entry("schedulable and cpu manager with static cpu manager policy configured and device plugins are not initialized",
			deviceController(true),
			[]string{"non/existent/cpumanager/statefile", cpu_manager_static_path},
			"true",
			"true",
		),
		Entry("schedulable and no cpu manager with no cpu manager policy configured and device plugins are not initialized",
			deviceController(true),
			[]string{cpu_manager_none_path, "non/existent/cpumanager/statefile"},
			"true",
			"false",
		),
	)

	DescribeTable("without cpumanager featuregate should set the node to", func(deviceController device_manager.DeviceControllerInterface, schedulable string) {
		heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController, config(), "mynode")
		heartbeat.do()
		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.NodeSchedulable, schedulable))
		Expect(node.Labels).ToNot(HaveKeyWithValue(virtv1.DeprecatedCPUManager, false))
		Expect(node.Labels).ToNot(HaveKeyWithValue(virtv1.CPUManager, false))
	},
		Entry("not schedulable with no cpumanager label present",
			deviceController(false),
			"false",
		),
		Entry("schedulable with no cpumanger label present",
			deviceController(true),
			"true",
		),
	)

	DescribeTable("without deviceplugin and", func(deviceController device_manager.DeviceControllerInterface, initiallySchedulable string, finallySchedulable string) {
		heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController, config(), "mynode")
		heartbeat.devicePluginWaitTimeout = 2 * time.Second
		heartbeat.devicePluginPollIntervall = 10 * time.Millisecond
		stopChan := make(chan struct{})
		done := heartbeat.Run(100*time.Second, stopChan)
		defer func() {
			close(stopChan)
			<-done
		}()
		Eventually(func() map[string]string {
			node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return node.Labels
		}).Should(And(
			HaveKeyWithValue(virtv1.NodeSchedulable, initiallySchedulable),
		))
		Consistently(func() map[string]string {
			node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return node.Labels
		}, 500*time.Millisecond, 10*time.Millisecond).Should(And(
			HaveKeyWithValue(virtv1.NodeSchedulable, initiallySchedulable),
		))
		Eventually(func() map[string]string {
			node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return node.Labels
		}).Should(And(
			HaveKeyWithValue(virtv1.NodeSchedulable, finallySchedulable),
		))
		Consistently(func() map[string]string {
			node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			return node.Labels
		}, 500*time.Millisecond, 10*time.Millisecond).Should(And(
			HaveKeyWithValue(virtv1.NodeSchedulable, finallySchedulable),
		))
	},
		Entry("not becoming ready, node should be set to unschedulable immediately and stick to it",
			newProbeCountingDeviceController(probe{false, 1000}),
			"false",
			"false",
		),
		Entry("becoming ready after a few probes, node should be set to unschedulable immediately and switch earlier than one minute",
			newProbeCountingDeviceController(probe{false, 100}, probe{true, 100}),
			"false",
			"true",
		),
	)
})

type fakeDeviceController struct {
	initialized bool
}

func (f *fakeDeviceController) Initialized() bool {
	return f.initialized
}

func (f *fakeDeviceController) RefreshMediatedDeviceTypes() {
	return
}

func config(featuregates ...string) *virtconfig.ClusterConfig {
	cfg := &virtv1.KubeVirtConfiguration{
		DeveloperConfiguration: &virtv1.DeveloperConfiguration{
			FeatureGates: featuregates,
		},
	}
	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(cfg)
	return clusterConfig
}

func deviceController(initialized bool) device_manager.DeviceControllerInterface {
	return &fakeDeviceController{initialized: initialized}
}

type probeCountingDeviceController struct {
	probes []bool
	probed int
	lock   *sync.Mutex
}

func (f *probeCountingDeviceController) Initialized() bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.probed++
	return f.probes[f.probed-1]
}

func (f *probeCountingDeviceController) RefreshMediatedDeviceTypes() {
	return
}

func newProbeCountingDeviceController(probes ...probe) device_manager.DeviceControllerInterface {
	var probeArray []bool
	for _, p := range probes {
		for x := 0; x < p.repetitions; x++ {
			probeArray = append(probeArray, p.value)
		}
	}
	return &probeCountingDeviceController{probes: probeArray, lock: &sync.Mutex{}}
}

type probe struct {
	value       bool
	repetitions int
}
