package heartbeat

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
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

	table.DescribeTable("with cpumanager featuregate should set the node to", func(deviceController device_manager.DeviceControllerInterface, cpuManagerPaths []string, schedulable string, cpumanager string) {
		heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController, config(virtconfig.CPUManager), "mynode")
		heartbeat.cpuManagerPaths = cpuManagerPaths
		heartbeat.do()
		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.NodeSchedulable, schedulable))
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.CPUManager, cpumanager))
	},
		table.Entry("not schedulable and no cpu manager with no cpu manager file and device plugins are not initialized",
			deviceController(false),
			[]string{"non/existent/cpumanager/statefile"},
			"false",
			"false",
		),
		table.Entry("schedulable and no cpu manager with no cpu manager file and plugins are not initialized",
			deviceController(true),
			[]string{"non/existent/cpumanager/statefile"},
			"true",
			"false",
		),
		table.Entry("schedulable and cpu manager with static cpu manager policy configured and device plugins are not initialized",
			deviceController(true),
			[]string{"non/existent/cpumanager/statefile", cpu_manager_static_path},
			"true",
			"true",
		),
		table.Entry("schedulable and no cpu manager with no cpu manager policy configured and device plugins are not initialized",
			deviceController(true),
			[]string{cpu_manager_none_path, "non/existent/cpumanager/statefile"},
			"true",
			"false",
		),
	)

	table.DescribeTable("without cpumanager featuregate should set the node to", func(deviceController device_manager.DeviceControllerInterface, schedulable string) {
		heartbeat := NewHeartBeat(fakeClient.CoreV1(), deviceController, config(), "mynode")
		heartbeat.do()
		node, err := fakeClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(virtv1.NodeSchedulable, schedulable))
		Expect(node.Labels).ToNot(HaveKeyWithValue(virtv1.CPUManager, false))
	},
		table.Entry("not schedulable with no cpumanager label present",
			deviceController(false),
			"false",
		),
		table.Entry("schedulable with no cpumanger label present",
			deviceController(true),
			"true",
		),
	)

	table.DescribeTable("without deviceplugin and", func(deviceController device_manager.DeviceControllerInterface, initiallySchedulable string, finallySchedulable string) {
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
		table.Entry("not becoming ready, node should be set to unschedulable immediately and stick to it",
			newProbeCountingDeviceController(probe{false, 1000}),
			"false",
			"false",
		),
		table.Entry("becoming ready after a few probes, node should be set to unschedulable immediately and switch earlier than one minute",
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
