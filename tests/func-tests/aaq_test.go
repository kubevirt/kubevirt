package tests_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	aaqv1alpha1 "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core/v1alpha1"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	setAAQFGPatchTemplate = `[{"op": "replace", "path": "/spec/enableApplicationAwareQuota", "value": %t}]`
)

var _ = Describe("Test AAQ", Label("AAQ"), Serial, Ordered, func() {
	tests.FlagParse()
	var (
		k8scli client.Client
	)

	BeforeEach(func(ctx context.Context) {
		k8scli = tests.GetControllerRuntimeClient()

		disableAAQFeatureGate(ctx, k8scli)
	})

	AfterAll(func(ctx context.Context) {
		disableAAQFeatureGate(ctx, k8scli)
	})

	When("set the applicationAwareConfig exists", func() {
		It("should create the AAQ CR and all the pods", func(ctx context.Context) {

			enableAAQFeatureGate(ctx, k8scli)

			By("check the AAQ CR")
			Eventually(func(g Gomega, ctx context.Context) bool {
				aaq, err := getAAQ(ctx, k8scli)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(aaq.Status.Conditions).ToNot(BeEmpty())
				return conditionsv1.IsStatusConditionTrue(aaq.Status.Conditions, conditionsv1.ConditionAvailable)
			}).WithTimeout(5 * time.Minute).WithPolling(time.Second).WithContext(ctx).ShouldNot(BeTrue())

			By("check AAQ pods")
			Eventually(func(g Gomega, ctx context.Context) {
				deps := &appsv1.DeploymentList{}
				Expect(
					k8scli.List(ctx, deps, client.MatchingLabels{"app.kubernetes.io/managed-by": "aaq-operator"}),
				).To(Succeed())
				g.Expect(deps.Items).To(HaveLen(2))

				expectedPods := int32(0)
				for _, dep := range deps.Items {
					g.Expect(dep.Status.ReadyReplicas).To(Equal(dep.Status.Replicas))
					expectedPods += dep.Status.Replicas
				}

				pods := &corev1.PodList{}
				Expect(k8scli.List(
					ctx, pods, client.MatchingLabels{"app.kubernetes.io/managed-by": "aaq-operator"}),
				).To(Succeed())
				g.Expect(pods.Items).To(HaveLen(int(expectedPods)))
			}).WithTimeout(5 * time.Minute).
				WithPolling(time.Second).
				WithContext(ctx).
				Should(Succeed())
		})
	})
})

func getAAQ(ctx context.Context, cli client.Client) (*aaqv1alpha1.AAQ, error) {
	aaq := &aaqv1alpha1.AAQ{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "aaq-" + hcoutil.HyperConvergedName,
			Namespace: tests.InstallNamespace,
		},
	}

	err := cli.Get(ctx, client.ObjectKeyFromObject(aaq), aaq)
	return aaq, err
}

func enableAAQFeatureGate(ctx context.Context, cli client.Client) {
	By("enable the AAQ FG")
	setAAQFeatureGate(ctx, cli, true)
}

func disableAAQFeatureGate(ctx context.Context, cli client.Client) {
	By("disable the AAQ FG")
	setAAQFeatureGate(ctx, cli, false)

	By("make sure the AAQ CR was removed")
	Eventually(func(ctx context.Context) error {
		_, err := getAAQ(ctx, cli)
		return err
	}).WithTimeout(5 * time.Minute).
		WithPolling(100 * time.Millisecond).
		WithOffset(1).
		WithContext(ctx).
		Should(MatchError(errors.IsNotFound, "not found error"))
}

func setAAQFeatureGate(ctx context.Context, cli client.Client, fgState bool) {
	patchBytes := []byte(fmt.Sprintf(setAAQFGPatchTemplate, fgState))

	Eventually(tests.PatchHCO).
		WithArguments(ctx, cli, patchBytes).
		WithTimeout(10 * time.Second).
		WithPolling(100 * time.Millisecond).
		WithContext(ctx).
		WithOffset(2).
		Should(Succeed())
}
