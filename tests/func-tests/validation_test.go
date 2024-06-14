package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Check CR validation", Label("validation"), Serial, func() {
	var (
		cli client.Client
	)

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()

		tests.RestoreDefaults(ctx, cli)
	})

	Context("for AutoCPULimitNamespaceLabelSelector", func() {
		DescribeTable("should", func(ctx context.Context, allocationRatio *int, outcome gomegatypes.GomegaMatcher) {
			requirements := &v1beta1.OperandResourceRequirements{
				VmiCPUAllocationRatio: allocationRatio,
				AutoCPULimitNamespaceLabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"someLabel": "true"},
				},
			}
			Eventually(func(ctx context.Context) error {
				var err error
				hc := tests.GetHCO(ctx, cli)
				hc.Spec.ResourceRequirements = requirements
				_, err = tests.UpdateHCO(ctx, cli, hc)
				return err
			}).WithTimeout(10 * time.Second).WithPolling(500 * time.Millisecond).WithContext(ctx).Should(outcome)
		},
			Entry("succeed when VMI CPU allocation is nil", nil, Succeed()),
			Entry("fail when VMI CPU allocation is 1", ptr.To(1), MatchError(ContainSubstring("Automatic CPU limits are incompatible with a VMI CPU allocation ratio of 1"))),
			Entry("succeed when VMI CPU allocation is 2", ptr.To(2), Succeed()),
		)
	})
})
