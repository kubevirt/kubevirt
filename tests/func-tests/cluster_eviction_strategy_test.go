package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Cluster level evictionStrategy default value", Serial, Ordered, func() {
	tests.FlagParse()
	var cli kubecli.KubevirtClient
	ctx := context.TODO()

	var (
		initialEvictionStrategy *v1.EvictionStrategy
		singleWorkerCluster     bool
	)

	BeforeEach(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(cli).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())

		singleWorkerCluster, err = isSingleWorkerCluster(cli)
		Expect(err).ToNot(HaveOccurred())

		tests.BeforeEach()
		hc := tests.GetHCO(ctx, cli)
		initialEvictionStrategy = hc.Spec.EvictionStrategy
	})

	AfterEach(func() {
		hc := tests.GetHCO(ctx, cli)
		hc.Spec.EvictionStrategy = initialEvictionStrategy
		_ = tests.UpdateHCORetry(ctx, cli, hc)
	})

	It("Should set spec.evictionStrategy = None by default on single worker clusters", Label(tests.SingleNodeLabel), func() {
		Expect(singleWorkerCluster).To(BeTrue(), "this test requires single worker cluster; use the %q label to skip this test", tests.SingleNodeLabel)

		hco := tests.GetHCO(ctx, cli)
		hco.Spec.EvictionStrategy = nil
		hco = tests.UpdateHCORetry(ctx, cli, hco)
		noneEvictionStrategy := v1.EvictionStrategyNone
		Expect(hco.Spec.EvictionStrategy).To(Not(BeNil()))
		Expect(hco.Spec.EvictionStrategy).To(Equal(&noneEvictionStrategy))
	})

	It("Should set spec.evictionStrategy = LiveMigrate by default with multiple worker node", Label(tests.HighlyAvailableClusterLabel), func() {
		tests.FailIfSingleNode(singleWorkerCluster)
		hco := tests.GetHCO(ctx, cli)
		hco.Spec.EvictionStrategy = nil
		hco = tests.UpdateHCORetry(ctx, cli, hco)
		lmEvictionStrategy := v1.EvictionStrategyLiveMigrate
		Expect(hco.Spec.EvictionStrategy).To(Not(BeNil()))
		Expect(hco.Spec.EvictionStrategy).To(Equal(&lmEvictionStrategy))
	})

})

func isSingleWorkerCluster(cli kubecli.KubevirtClient) (bool, error) {
	workerNodes, err := cli.CoreV1().Nodes().List(context.TODO(), k8smetav1.ListOptions{LabelSelector: "node-role.kubernetes.io/worker"})
	if err != nil {
		return false, err
	}

	return len(workerNodes.Items) == 1, nil
}
