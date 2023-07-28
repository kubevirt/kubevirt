package heartbeat

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	device_manager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const (
	cpu_manager_static_path = "testdata/cpu_manager_state_static"
	cpu_manager_none_path   = "testdata/cpu_manager_state_none"
)

var _ = Describe("Heartbeat", func() {

	var fakeClient *fakeclientset.Clientset
	var fakeK8sClient *fake.Clientset

	BeforeEach(func() {
		fakeClient = fakeclientset.NewSimpleClientset(&v1.ShadowNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mynode",
			},
		})
		fakeK8sClient = fake.NewSimpleClientset(&k8sv1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "mynode",
			},
		})
	})
	Context("upgrade/downgrade", func() {

		PIt("should patch node if RBAC allows it", func() {
			nodePatchCountA := atomic.Int64{}
			shadowNodePatchCountA := atomic.Int64{}

			fakeK8sClient.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				nodePatchCountA.Add(1)
				return true, nil, nil
			})
			fakeClient.PrependReactor("*", "*", func(action testing.Action) (handled bool, obj runtime.Object, err error) {
				shadowNodePatchCountA.Add(1)
				return true, nil, nil
			})

			heartbeat := NewHeartBeat(fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.CoreV1().Nodes(), deviceController(true), config(), "mynode")
			stopChan := make(chan struct{})

			nodePatchCount := 0
			shadowNodePatchCount := 0

			_ = heartbeat.Run(100*time.Millisecond, stopChan)
			DeferCleanup(func() { close(stopChan) })
			ticker := time.NewTicker(100 * time.Millisecond)
			timeout := time.NewTicker(10 * 100 * time.Millisecond)

			DeferCleanup(func() { ticker.Stop(); timeout.Stop() })
			for _ = range ticker.C {
				nodePatchCount = int(nodePatchCountA.Load())
				shadowNodePatchCount = int(shadowNodePatchCountA.Load())
				select {
				case <-timeout.C:
					Fail(fmt.Sprintf("%d node patch count needs to be == %d shadownode path count", nodePatchCount, shadowNodePatchCount))
				default:
				}

				if nodePatchCount > 2 && nodePatchCount == shadowNodePatchCount {
					return
				}
			}
		})
	})

	Context("upon finishing", func() {
		It("should set the node to not schedulable", func() {
			heartbeat := NewHeartBeat(fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.CoreV1().Nodes(), deviceController(true), config(), "mynode")
			stopChan := make(chan struct{})
			done := heartbeat.Run(30*time.Second, stopChan)
			Eventually(func() map[string]string {
				node, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
				Expect(err).ToNot(HaveOccurred())
				return node.Labels
			}).Should(And(
				HaveKeyWithValue(v1.NodeSchedulable, "true"),
			))
			close(stopChan)
			<-done
			expectLabels(fakeK8sClient, fakeClient, HaveKeyWithValue(v1.NodeSchedulable, "false"))
		})
	})

	DescribeTable("with cpumanager featuregate should set the node to", func(deviceController device_manager.DeviceControllerInterface, cpuManagerPaths []string, schedulable string, cpumanager string) {
		heartbeat := NewHeartBeat(fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.CoreV1().Nodes(), deviceController, config(virtconfig.CPUManager), "mynode")
		heartbeat.cpuManagerPaths = cpuManagerPaths
		heartbeat.do()
		expectLabels(fakeK8sClient, fakeClient, SatisfyAll(
			HaveKeyWithValue(v1.NodeSchedulable, schedulable),
			HaveKeyWithValue(v1.CPUManager, cpumanager),
		))

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
		heartbeat := NewHeartBeat(fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.CoreV1().Nodes(), deviceController, config(), "mynode")
		heartbeat.do()
		expectLabels(fakeK8sClient, fakeClient, SatisfyAll(
			HaveKeyWithValue(v1.NodeSchedulable, schedulable),
			Not(HaveKeyWithValue(v1.CPUManager, false)),
		))
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
		heartbeat := NewHeartBeat(fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.CoreV1().Nodes(), deviceController, config(), "mynode")
		heartbeat.devicePluginWaitTimeout = 2 * time.Second
		heartbeat.devicePluginPollIntervall = 10 * time.Millisecond
		stopChan := make(chan struct{})
		done := heartbeat.Run(100*time.Second, stopChan)
		defer func() {
			close(stopChan)
			<-done
		}()
		Eventually(func(g Gomega) bool {
			node, shadowNode := getNodes(fakeK8sClient, fakeClient)
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			return true
		}).Should(BeTrue())
		Consistently(func(g Gomega) bool {
			node, shadowNode := getNodes(fakeK8sClient, fakeClient)
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			return true
		}, 500*time.Millisecond, 10*time.Millisecond).Should(BeTrue())
		Eventually(func(g Gomega) bool {
			node, shadowNode := getNodes(fakeK8sClient, fakeClient)
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))
			return true
		}).Should(BeTrue())
		Consistently(func(g Gomega) bool {
			node, shadowNode := getNodes(fakeK8sClient, fakeClient)
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))
			return true
		}, 500*time.Millisecond, 10*time.Millisecond).Should(BeTrue())
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
}

func config(featuregates ...string) *virtconfig.ClusterConfig {
	cfg := &v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
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
