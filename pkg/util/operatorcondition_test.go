package util

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
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
		err = oc.Set(ctx, metav1.ConditionTrue, "Reason", "message")
		Expect(err).ToNot(HaveOccurred())
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
		err := operatorsapiv2.AddToScheme(testScheme)
		Expect(err).ShouldNot(HaveOccurred())

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
		Expect(err).ShouldNot(HaveOccurred())

		cond, err := oc.cond.Get(context.TODO())
		Expect(err).ShouldNot(HaveOccurred())

		Expect(cond.Type).Should(Equal("testCondition"))

		err = oc.Set(context.TODO(), metav1.ConditionTrue, "myReason", "my message")
		Expect(err).ShouldNot(HaveOccurred())

		cond, err = oc.cond.Get(context.TODO())
		Expect(err).ShouldNot(HaveOccurred())
		Expect(cond.Type).Should(Equal("testCondition"))
		Expect(cond.Reason).Should(Equal("myReason"))
		Expect(cond.Message).Should(Equal("my message"))
	})
})

func TestOperatorCondition(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OperatorCondition Suite")
}

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
