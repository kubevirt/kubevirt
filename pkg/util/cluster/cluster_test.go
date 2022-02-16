package cluster

import (
	"strconv"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

var _ = Describe("OpenShift Test", func() {

	var ctrl *gomock.Controller
	var virtClient *kubecli.MockKubevirtClient
	var discoveryClient *discoveryFake.FakeDiscovery

	BeforeEach(func() {

		ctrl = gomock.NewController(GinkgoT())
		virtClient = kubecli.NewMockKubevirtClient(ctrl)
		discoveryClient = &discoveryFake.FakeDiscovery{
			Fake: &fake.NewSimpleClientset().Fake,
		}
		virtClient.EXPECT().DiscoveryClient().Return(discoveryClient).AnyTimes()

	})

	getServerResources := func(onOpenShift bool) []*metav1.APIResourceList {
		list := []*metav1.APIResourceList{
			{
				GroupVersion: v1.GroupVersion.String(),
				APIResources: []metav1.APIResource{
					{
						Name: "kubevirts",
					},
				},
			},
		}
		if onOpenShift {
			list = append(list, &metav1.APIResourceList{
				GroupVersion: secv1.GroupVersion.String(),
				APIResources: []metav1.APIResource{
					{
						Name: "securitycontextconstraints",
					},
				},
			})
		}
		return list
	}

	table.DescribeTable("Testing for OpenShift", func(onOpenShift bool) {

		discoveryClient.Fake.Resources = getServerResources(onOpenShift)
		isOnOpenShift, err := IsOnOpenShift(virtClient)
		Expect(err).ToNot(HaveOccurred(), "should not return an error")
		Expect(isOnOpenShift).To(Equal(onOpenShift), "should return "+strconv.FormatBool(onOpenShift))

	},
		table.Entry("on Kubernetes", false),
		table.Entry("on OpenShift", true),
	)

})
