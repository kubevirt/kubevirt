package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kvv1 "kubevirt.io/api/core/v1"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Check that the TuningPolicy annotation is configuring the KV object as expected", Serial, Label("TuningPolicy"), func() {
	tests.FlagParse()
	var (
		cli client.Client
		ctx context.Context
	)

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()
		ctx = context.Background()
	})

	AfterEach(func() {
		hc := tests.GetHCO(ctx, cli)

		delete(hc.Annotations, common.TuningPolicyAnnotationName)
		hc.Spec.TuningPolicy = ""

		tests.UpdateHCORetry(ctx, cli, hc)
	})

	It("should update KV with the tuningPolicy annotation", func() {
		hc := tests.GetHCO(ctx, cli)

		if hc.Annotations == nil {
			hc.Annotations = make(map[string]string)
		}
		hc.Annotations[common.TuningPolicyAnnotationName] = `{"qps":100,"burst":200}`
		hc.Spec.TuningPolicy = v1beta1.HyperConvergedAnnotationTuningPolicy

		tests.UpdateHCORetry(ctx, cli, hc)

		expected := kvv1.TokenBucketRateLimiter{
			Burst: 200,
			QPS:   100,
		}

		checkTuningPolicy(ctx, cli, expected)
	})

	It("should update KV with the highBurst tuningPolicy", func() {
		hc := tests.GetHCO(ctx, cli)

		delete(hc.Annotations, common.TuningPolicyAnnotationName)
		hc.Spec.TuningPolicy = v1beta1.HyperConvergedHighBurstProfile

		tests.UpdateHCORetry(ctx, cli, hc)

		expected := kvv1.TokenBucketRateLimiter{
			Burst: 400,
			QPS:   200,
		}

		checkTuningPolicy(ctx, cli, expected)
	})
})

func checkTuningPolicy(ctx context.Context, cli client.Client, expected kvv1.TokenBucketRateLimiter) {
	Eventually(func(g Gomega) {
		kv := &kvv1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt-kubevirt-hyperconverged",
				Namespace: tests.InstallNamespace,
			},
		}

		Expect(cli.Get(ctx, client.ObjectKeyFromObject(kv), kv)).To(Succeed())

		g.Expect(kv.Spec.Configuration).ToNot(BeNil())
		checkReloadableComponentConfiguration(g, kv.Spec.Configuration.APIConfiguration, expected)
		checkReloadableComponentConfiguration(g, kv.Spec.Configuration.ControllerConfiguration, expected)
		checkReloadableComponentConfiguration(g, kv.Spec.Configuration.HandlerConfiguration, expected)
		checkReloadableComponentConfiguration(g, kv.Spec.Configuration.WebhookConfiguration, expected)
	}).WithTimeout(time.Minute * 2).WithPolling(time.Second).Should(Succeed())

}

func checkReloadableComponentConfiguration(g Gomega, actual *kvv1.ReloadableComponentConfiguration, expected kvv1.TokenBucketRateLimiter) {
	g.ExpectWithOffset(1, actual).ShouldNot(BeNil())
	g.ExpectWithOffset(1, actual.RestClient).ShouldNot(BeNil())
	g.ExpectWithOffset(1, actual.RestClient.RateLimiter).ShouldNot(BeNil())
	g.ExpectWithOffset(1, actual.RestClient.RateLimiter.TokenBucketRateLimiter).Should(HaveValue(Equal(expected)))
}
