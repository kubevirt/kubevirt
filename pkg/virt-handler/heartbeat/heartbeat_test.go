package heartbeat

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	authorizationv1 "k8s.io/api/authorization/v1"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/testing"

	v1 "kubevirt.io/api/core/v1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/kubevirt/pkg/testutils"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	devicemanager "kubevirt.io/kubevirt/pkg/virt-handler/device-manager"
)

const (
	cpuManagerStaticPath = "testdata/cpu_manager_state_static"
	cpuManagerNonePath   = "testdata/cpu_manager_state_none"
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

		fakeK8sClient.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			review := action.(testing.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
			review.Status.Allowed = true
			return true, review, nil
		})
	})
	Context("upon finishing", func() {
		It("should set the node to not schedulable", func() {
			heartbeat := NewHeartBeat(fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.AuthorizationV1(), deviceController(true), config(), "mynode")
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
			node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, "false"))

			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, "false"))
		})
	})

	DescribeTable("with cpumanager featuregate should set the node to", func(deviceController devicemanager.DeviceControllerInterface, cpuManagerPaths []string, schedulable string, cpumanager string) {
		heartbeat := NewHeartBeat(fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.AuthorizationV1(), deviceController, config(virtconfig.CPUManager), "mynode")
		heartbeat.cpuManagerPaths = cpuManagerPaths
		heartbeat.do()

		node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, schedulable))
		Expect(node.Labels).To(HaveKeyWithValue(v1.CPUManager, cpumanager))

		shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, schedulable))
		Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.CPUManager, cpumanager))
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
			[]string{"non/existent/cpumanager/statefile", cpuManagerStaticPath},
			"true",
			"true",
		),
		Entry("schedulable and no cpu manager with no cpu manager policy configured and device plugins are not initialized",
			deviceController(true),
			[]string{cpuManagerNonePath, "non/existent/cpumanager/statefile"},
			"true",
			"false",
		),
	)

	DescribeTable("without cpumanager featuregate should set the node to", func(deviceController devicemanager.DeviceControllerInterface, schedulable string) {
		heartbeat := NewHeartBeat(fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.AuthorizationV1(), deviceController, config(), "mynode")
		heartbeat.do()

		node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, schedulable))
		Expect(node.Labels).ToNot(HaveKeyWithValue(v1.CPUManager, false))

		shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, schedulable))
		Expect(shadowNode.Labels).ToNot(HaveKeyWithValue(v1.CPUManager, false))
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

	DescribeTable("without deviceplugin and", func(deviceController devicemanager.DeviceControllerInterface, initiallySchedulable string, finallySchedulable string) {
		heartbeat := NewHeartBeat(fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.AuthorizationV1(), deviceController, config(), "mynode")
		heartbeat.devicePluginWaitTimeout = 2 * time.Second
		heartbeat.devicePluginPollIntervall = 10 * time.Millisecond
		stopChan := make(chan struct{})
		done := heartbeat.Run(100*time.Second, stopChan)
		defer func() {
			close(stopChan)
			<-done
		}()
		Eventually(func(g Gomega) bool {
			node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))

			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			return true
		}).Should(BeTrue())
		Consistently(func(g Gomega) bool {
			node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))

			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, initiallySchedulable))
			return true
		}, 500*time.Millisecond, 10*time.Millisecond).Should(BeTrue())
		Eventually(func(g Gomega) bool {
			node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))

			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(shadowNode.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))
			return true
		}).Should(BeTrue())
		Consistently(func(g Gomega) bool {
			node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
			g.Expect(node.Labels).To(HaveKeyWithValue(v1.NodeSchedulable, finallySchedulable))

			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			Expect(err).ToNot(HaveOccurred())
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
	It("should not patch the node after patch node RBAC is removed", func() {
		fakeK8sClient.Fake.PrependReactor("create", "selfsubjectaccessreviews", func(action testing.Action) (handled bool, ret runtime.Object, err error) {
			review := action.(testing.CreateAction).GetObject().(*authorizationv1.SelfSubjectAccessReview)
			review.Status.Allowed = false
			return true, review, nil
		})
		deviceController := deviceController(false)
		cpuManagerPaths := []string{"non/existent/cpumanager/statefile"}
		schedulable := "false"
		cpumanager := "false"

		heartbeat := NewHeartBeat(fakeK8sClient.CoreV1().Nodes(), fakeClient.KubevirtV1().ShadowNodes(), fakeK8sClient.AuthorizationV1(), deviceController, config(virtconfig.CPUManager), "mynode")
		heartbeat.cpuManagerPaths = cpuManagerPaths
		heartbeat.do()

		node, err := fakeK8sClient.CoreV1().Nodes().Get(context.Background(), "mynode", metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(node.Labels).ToNot(HaveKeyWithValue(v1.NodeSchedulable, schedulable))
		Expect(node.Labels).ToNot(HaveKeyWithValue(v1.CPUManager, cpumanager))
	})

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
	cfg := &v1.KubeVirtConfiguration{
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			FeatureGates: featuregates,
		},
	}
	clusterConfig, _, _ := testutils.NewFakeClusterConfigUsingKVConfig(cfg)
	return clusterConfig
}

func deviceController(initialized bool) devicemanager.DeviceControllerInterface {
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

func newProbeCountingDeviceController(probes ...probe) devicemanager.DeviceControllerInterface {
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
