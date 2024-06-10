package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "kubevirt.io/api/core/v1"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Cluster level evictionStrategy default value", Serial, Ordered, Label("evictionStrategy"), func() {
	tests.FlagParse()
	var (
		cli client.Client
		ctx context.Context

		initialEvictionStrategy *v1.EvictionStrategy
		singleWorkerCluster     bool
	)

	BeforeEach(func() {
		cli = tests.GetControllerRuntimeClient()

		ctx = context.Background()

		var err error
		singleWorkerCluster, err = isSingleWorkerCluster(ctx, cli)
		Expect(err).ToNot(HaveOccurred())

		tests.BeforeEach()
		hc := tests.GetHCO(ctx, cli)
		initialEvictionStrategy = hc.Spec.EvictionStrategy
	})

	AfterEach(func() {
		hc := tests.GetHCO(ctx, cli)
		hc.Spec.EvictionStrategy = initialEvictionStrategy
		tests.UpdateHCORetry(ctx, cli, hc)
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

func isSingleWorkerCluster(ctx context.Context, cli client.Client) (bool, error) {
	workerNodes := &corev1.NodeList{}
	err := cli.List(ctx, workerNodes, client.MatchingLabels{"node-role.kubernetes.io/worker": ""})

	if err != nil {
		return false, err
	}

	return len(workerNodes.Items) == 1, nil
}
