package util

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsapiv2 "github.com/operator-framework/api/pkg/operators/v2"
	"github.com/operator-framework/operator-lib/conditions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubevirt/hyperconverged-cluster-operator/version"
)

var _ = Describe("OperatorCondition", func() {
	DescribeTable("should return no error when setting the condition, in not-supported environments", func(ci ClusterInfo) {
		oc, err := NewOperatorCondition(ci, nil, operatorsapiv2.Upgradeable)
		Expect(err).ToNot(HaveOccurred())

		ctx := context.Background()
		Expect(oc.Set(ctx, metav1.ConditionTrue, "Reason", "message")).To(Succeed())
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

	It("valid condition", func() {
		testScheme := scheme.Scheme
		Expect(operatorsapiv2.AddToScheme(testScheme)).To(Succeed())

		cl := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()

		GetFactory = func(cl client.Client) conditions.Factory {
			if operatorConditionFactory == nil {
				operatorConditionFactory = OpCondFactoryMock{
					Client: cl,
				}
			}
			return operatorConditionFactory
		}

		oc, err := NewOperatorCondition(&ClusterInfoImp{
			managedByOLM:   true,
			runningLocally: false,
		}, cl, "testCondition")
		Expect(err).ToNot(HaveOccurred())

		cond, err := oc.cond.Get(context.TODO())
		Expect(err).ToNot(HaveOccurred())

		Expect(cond.Type).To(Equal("testCondition"))

		Expect(
			oc.Set(context.TODO(), metav1.ConditionTrue, "myReason", "my message"),
		).To(Succeed())

		cond, err = oc.cond.Get(context.TODO())
		Expect(err).ToNot(HaveOccurred())
		Expect(cond.Type).To(Equal("testCondition"))
		Expect(cond.Reason).To(Equal("myReason"))
		Expect(cond.Message).To(Equal("my message"))
	})
})

type OpCondFactoryMock struct {
	Client client.Client
}

func (fm OpCondFactoryMock) NewCondition(typ operatorsapiv2.ConditionType) (conditions.Condition, error) {
	return &ConditionMock{condition: &metav1.Condition{Type: string(typ)}}, nil
}

func (fm OpCondFactoryMock) GetNamespacedName() (*types.NamespacedName, error) {
	return &types.NamespacedName{Name: HyperConvergedCluster + "." + version.Version, Namespace: HyperConvergedName}, nil
}

type ConditionMock struct {
	condition *metav1.Condition
}

func (c ConditionMock) Get(_ context.Context) (*metav1.Condition, error) {
	return c.condition, nil
}

func (c *ConditionMock) Set(_ context.Context, status metav1.ConditionStatus, options ...conditions.Option) error {
	c.condition.Status = status
	for _, opt := range options {
		opt(c.condition)
	}
	return nil
}
