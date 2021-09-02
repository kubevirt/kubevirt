package util

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	operatorsapiv2 "github.com/operator-framework/api/pkg/operators/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("OperatorCondition", func() {
	DescribeTable("should return no error when setting the condition, in not-supported environments", func(ci ClusterInfo) {
		oc, err := NewOperatorCondition(ci, nil, operatorsapiv2.Upgradeable)
		Expect(err).To(BeNil())

		ctx := context.Background()
		err = oc.Set(ctx, metav1.ConditionTrue, "Reason", "message")
		Expect(err).To(BeNil())
	},
		Entry("should no-op when not managed by OLM", &ClusterInfoImp{
			managedByOLM:   false,
			runningLocally: false,
		}),
		Entry("should no-op when running locally", &ClusterInfoImp{
			managedByOLM:   true,
			runningLocally: true,
		}),
		Entry("should no-op when running locally and not managed by OLM", &ClusterInfoImp{
			managedByOLM:   false,
			runningLocally: true,
		}),
	)

	// Can't test real operator condition, as the library points to a constant path that is not exists in test env
	// TODO: enable this if there will be a fix to https://github.com/operator-framework/operator-lib/issues/50
	//
	//It("valid condition", func() {
	//	testScheme := scheme.Scheme
	//	err := operatorframeworkv2.AddToScheme(testScheme)
	//	Expect(err).ShouldNot(HaveOccurred())
	//
	//	os.Setenv("OPERATOR_CONDITION_NAME", "operator-condition-name")
	//
	//	cl := fake.NewClientBuilder().
	//		WithScheme(testScheme).
	//		Build()
	//
	//	oc, err := NewOperatorCondition(&ClusterInfoImp{
	//		managedByOLM:   true,
	//		runningLocally: false,
	//	}, cl, "testCondition")
	//	Expect(err).ShouldNot(HaveOccurred())
	//
	//	cond, err := oc.cond.Get(context.TODO())
	//	Expect(err).ShouldNot(HaveOccurred())
	//
	//	Expect(cond.Type).Should(Equal("testCondition"))
	//
	//})
})

func TestOperatorCondition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperatorCondition Suite")
}
