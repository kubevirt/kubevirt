package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"kubevirt.io/client-go/kubecli"

	kvv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/kubevirt/tests/flags"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Check that the TuningPolicy annotation is configuring the KV object as expected", Serial, func() {
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

		checkTuningPolicy(cli, expected)
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

		checkTuningPolicy(cli, expected)
	})
})

func checkTuningPolicy(cli kubecli.KubevirtClient, expected kvv1.TokenBucketRateLimiter) {
	Eventually(func(g Gomega) {
		kv, err := cli.KubeVirt(flags.KubeVirtInstallNamespace).Get("kubevirt-kubevirt-hyperconverged", &metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(kv).ToNot(BeNil())
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
