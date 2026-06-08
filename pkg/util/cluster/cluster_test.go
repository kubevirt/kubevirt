package cluster

import (
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	secv1 "github.com/openshift/api/security/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	discoveryFake "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"

	v1 "kubevirt.io/api/core/v1"
)

var _ = Describe("OpenShift Test", func() {

	var discoveryClient *discoveryFake.FakeDiscovery

	BeforeEach(func() {
		discoveryClient = &discoveryFake.FakeDiscovery{
			Fake: &fake.NewSimpleClientset().Fake,
		}
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

	DescribeTable("Testing for OpenShift", func(onOpenShift bool) {

		discoveryClient.Fake.Resources = getServerResources(onOpenShift)
		isOnOpenShift, err := IsOnOpenShift(discoveryClient)
		Expect(err).ToNot(HaveOccurred(), "should not return an error")
		Expect(isOnOpenShift).To(Equal(onOpenShift), "should return "+strconv.FormatBool(onOpenShift))

	},
		Entry("on Kubernetes", false),
		Entry("on OpenShift", true),
	)

})
