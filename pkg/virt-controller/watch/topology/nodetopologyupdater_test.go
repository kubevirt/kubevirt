package topology

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	g "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	v1 "kubevirt.io/api/core/v1"
	fakeclientset "kubevirt.io/client-go/generated/kubevirt/clientset/versioned/fake"

	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("Nodetopologyupdater", func() {

	var topologyUpdater *nodeTopologyUpdater
	var ctrl *gomock.Controller
	var hinter *MockHinter
	var virtClient *kubecli.MockKubevirtClient
	var fakeK8sClient *fake.Clientset
	var fakeClient *fakeclientset.Clientset

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		hinter = NewMockHinter(ctrl)
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		topologyUpdater = &nodeTopologyUpdater{
			hinter: hinter,
			client: virtClient,
		}
		fakeK8sClient = fake.NewSimpleClientset()
		fakeClient = fakeclientset.NewSimpleClientset()
		virtClient.EXPECT().CoreV1().Return(fakeK8sClient.CoreV1()).AnyTimes()
		virtClient.EXPECT().ShadowNodeClient().Return(fakeClient.KubevirtV1().ShadowNodes()).AnyTimes()
	})

	Context("with no VMs with TSC frequency set running", func() {

		BeforeEach(func() {
			hinter.EXPECT().LowestTSCFrequencyOnCluster().Return(int64(100), nil)
			hinter.EXPECT().TSCFrequenciesInUse().Return(nil)
		})

		It("should add the lowest scheduling frequency to a node", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true)}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 0, 1)
			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			g.Expect(err).ToNot(g.HaveOccurred())
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(123), "true"))
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(100), "true"))
		})

		It("should continue if it encounters invalid nodes", func() {
			nodes := []*k8sv1.Node{
				NodeWithTSC("mynode1", 123, true),
				NodeWithTSC("syncednode", 123, true, 123, 100),
				NodeWithInvalidTSC("invalid"),
				NodeWithTSC("mynode2", 123, true),
			}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 1, 1, 2)
		})

		It("should only add the nodes own frequency if the node is not schedulable", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, false)}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 0, 1)
			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			g.Expect(err).ToNot(g.HaveOccurred())
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(123), "true"))
			g.Expect(shadowNode.Labels).ToNot(g.HaveKeyWithValue(ToTSCSchedulableLabel(100), "true"))
		})

		It("should do nothing if all frequencies are already present", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true, 100, 123)}
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 1, 0)
		})

		It("should remove inappropriate labels", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true, 99, 200, 123)}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 0, 1)
			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			g.Expect(err).ToNot(g.HaveOccurred())
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(123), "true"))
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(100), "true"))
			g.Expect(shadowNode.Labels).ToNot(g.HaveKeyWithValue(ToTSCSchedulableLabel(99), "true"))
			g.Expect(shadowNode.Labels).ToNot(g.HaveKeyWithValue(ToTSCSchedulableLabel(200), "true"))
		})

	})

	Context("with repeated labels", func() {
		BeforeEach(func() {
			hinter.EXPECT().LowestTSCFrequencyOnCluster().Return(int64(100), nil)
			hinter.EXPECT().TSCFrequenciesInUse().Return([]int64{80, 80, 80, 60})
		})

		It("should do nothing if all frequencies are already present", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true, 100, 123, 80, 60)}
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 1, 0)
		})
	})

	Context("with VMs with TSC frequency running", func() {
		BeforeEach(func() {
			hinter.EXPECT().LowestTSCFrequencyOnCluster().Return(int64(100), nil)
			hinter.EXPECT().TSCFrequenciesInUse().Return([]int64{99, 101})
		})

		It("should keep old cluster minimums if still used by VMs", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true, 98, 99, 101, 200, 123)}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, 0, 1)
			shadowNode, err := fakeClient.KubevirtV1().ShadowNodes().Get(context.Background(), "mynode", metav1.GetOptions{})
			g.Expect(err).ToNot(g.HaveOccurred())
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(123), "true"))
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(100), "true"))
			g.Expect(shadowNode.Labels).ToNot(g.HaveKeyWithValue(ToTSCSchedulableLabel(98), "true"))
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(99), "true"))
			g.Expect(shadowNode.Labels).To(g.HaveKeyWithValue(ToTSCSchedulableLabel(101), "true"))
			g.Expect(shadowNode.Labels).ToNot(g.HaveKeyWithValue(ToTSCSchedulableLabel(200), "true"))
		})
	})

	Context("if not minimum TSC frequency can be determined", func() {
		BeforeEach(func() {
			hinter.EXPECT().LowestTSCFrequencyOnCluster().Return(int64(100), fmt.Errorf("no node with a frequency"))
		})

		It("should do nothing", func() {
			nodes := []*k8sv1.Node{NodeWithTSC("mynode", 123, true, 98, 99, 101, 200, 123)}
			trackNodes(fakeK8sClient, nodes...)
			shadowNodes := createShadowNodeFromNodes(nodes)
			trackShadowNodes(fakeClient, shadowNodes...)
			stats := topologyUpdater.sync(nodes)
			expectUpdates(stats, 0, len(nodes), 0)
		})
	})
})

func createShadowNodeFromNodes(nodes []*k8sv1.Node) []*v1.ShadowNode {
	var shadowNodes []*v1.ShadowNode
	for i, _ := range nodes {
		shadowNodes = append(shadowNodes, &v1.ShadowNode{
			ObjectMeta: metav1.ObjectMeta{
				Name:   nodes[i].Name,
				Labels: nodes[i].Labels,
			},
		})
	}
	return shadowNodes
}

func trackNodes(clientset *fake.Clientset, nodes ...*k8sv1.Node) {
	for i, _ := range nodes {
		g.ExpectWithOffset(1, clientset.Tracker().Add(nodes[i])).To(g.Succeed())
	}
}

func trackShadowNodes(clientset *fakeclientset.Clientset, shadowNodes ...*v1.ShadowNode) {
	for i, _ := range shadowNodes {
		g.ExpectWithOffset(1, clientset.Tracker().Add(shadowNodes[i])).To(g.Succeed())
	}
}

func expectUpdates(stats *updateStats, errors int, skipped int, updated int) {
	g.ExpectWithOffset(1, stats.error).To(g.Equal(errors), "errors")
	g.ExpectWithOffset(1, stats.skipped).To(g.Equal(skipped), "skipped")
	g.ExpectWithOffset(1, stats.updated).To(g.Equal(updated), "updated")
}
