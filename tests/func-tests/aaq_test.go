package tests_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	aaqv1alpha1 "kubevirt.io/application-aware-quota/staging/src/kubevirt.io/application-aware-quota-api/pkg/apis/core/v1alpha1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/flags"

	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const (
	setAAQFGPatchTemplate = `[{"op": "replace", "path": "/spec/featureGates/enableApplicationAwareQuota", "value": %t}]`
)

var _ = Describe("Test AAQ", Label("AAQ"), Serial, Ordered, func() {
	tests.FlagParse()
	var (
		cli kubecli.KubevirtClient
		ctx context.Context
	)

	BeforeEach(func() {
		var err error

		cli, err = kubecli.GetKubevirtClient()
		Expect(cli).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()

		disableAAQFeatureGate(ctx, cli)
	})

	AfterAll(func() {
		disableAAQFeatureGate(ctx, cli)
	})

	When("set the applicationAwareConfig exists", func() {
		It("should create the AAQ CR and all the pods", func() {

			enableAAQFeatureGate(ctx, cli)

			By("check the AAQ CR")
			Eventually(func(g Gomega) bool {
				aaq := getAAQ(ctx, cli, g)
				g.Expect(aaq.Status.Conditions).ShouldNot(BeEmpty())
				return conditionsv1.IsStatusConditionTrue(aaq.Status.Conditions, conditionsv1.ConditionAvailable)
			}).WithTimeout(5 * time.Minute).WithPolling(time.Second).ShouldNot(BeTrue())

			By("check AAQ pods")
			Eventually(func(g Gomega) {
				deps, err := cli.AppsV1().Deployments(flags.KubeVirtInstallNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=aaq-operator"})
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(deps.Items).To(HaveLen(2))

				expectedPods := int32(0)
				for _, dep := range deps.Items {
					g.Expect(dep.Status.ReadyReplicas).Should(Equal(dep.Status.Replicas))
					expectedPods += dep.Status.Replicas
				}

				pods, err := cli.CoreV1().Pods(flags.KubeVirtInstallNamespace).List(ctx, metav1.ListOptions{LabelSelector: "app.kubernetes.io/managed-by=aaq-operator"})
				g.Expect(err).ShouldNot(HaveOccurred())
				g.Expect(pods.Items).Should(HaveLen(int(expectedPods)))
			}).WithTimeout(5 * time.Minute).
				WithPolling(time.Second).
				Should(Succeed())
		})
	})
})

func getAAQ(ctx context.Context, cli kubecli.KubevirtClient, g Gomega) *aaqv1alpha1.AAQ {
	aaq := &aaqv1alpha1.AAQ{}

	unstAAQ, err := getAAQResource(ctx, cli)
	g.ExpectWithOffset(1, err).ToNot(HaveOccurred())
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstAAQ.Object, aaq)
	g.ExpectWithOffset(1, err).ToNot(HaveOccurred())

	return aaq
}

func getAAQResource(ctx context.Context, client kubecli.KubevirtClient) (*unstructured.Unstructured, error) {
	aaqGVR := schema.GroupVersionResource{Group: aaqv1alpha1.SchemeGroupVersion.Group, Version: aaqv1alpha1.SchemeGroupVersion.Version, Resource: "aaqs"}

	return client.DynamicClient().Resource(aaqGVR).Get(ctx, "aaq-"+hcoutil.HyperConvergedName, metav1.GetOptions{})
}

func enableAAQFeatureGate(ctx context.Context, cli kubecli.KubevirtClient) {
	By("enable the AAQ FG")
	setAAQFeatureGate(ctx, cli, true)
}

func disableAAQFeatureGate(ctx context.Context, cli kubecli.KubevirtClient) {
	By("disable the AAQ FG")
	setAAQFeatureGate(ctx, cli, false)

	By("make sure the AAQ CR was removed")
	Eventually(func() error {
		_, err := getAAQResource(ctx, cli)
		return err
	}).WithTimeout(5 * time.Minute).
		WithPolling(100 * time.Millisecond).
		WithOffset(1).
		Should(MatchError(errors.IsNotFound, "not found error"))
}

func setAAQFeatureGate(ctx context.Context, cli kubecli.KubevirtClient, fgState bool) {
	patch := []byte(fmt.Sprintf(setAAQFGPatchTemplate, fgState))
	Eventually(tests.PatchHCO).
		WithArguments(ctx, cli, patch).
		WithTimeout(10 * time.Second).
		WithPolling(100 * time.Millisecond).
		WithOffset(2).
		Should(Succeed())
}
