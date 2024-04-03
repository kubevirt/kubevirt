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
		singleworkerCluster     bool
	)

	BeforeEach(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(cli).ToNot(BeNil())
		Expect(err).ToNot(HaveOccurred())

		singleworkerCluster, err = isSingleWorkerCluster(cli)
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

	It("Should set spec.evictionStrategy = None by default on single worker clusters", Label(singleNodeLabel), func() {
		if !singleworkerCluster {
			Skip("Skipping single worker cluster test having more than one worker node")
		}
		hco := tests.GetHCO(ctx, cli)
		hco.Spec.EvictionStrategy = nil
		hco = tests.UpdateHCORetry(ctx, cli, hco)
		noneEvictionStrategy := v1.EvictionStrategyNone
		Expect(hco.Spec.EvictionStrategy).To(Not(BeNil()))
		Expect(hco.Spec.EvictionStrategy).To(Equal(&noneEvictionStrategy))
	})

	It("Should set spec.evictionStrategy = LiveMigrate by default with multiple worker node", Label(highlyAvailableClusterLabel), func() {
		if singleworkerCluster {
			Skip("Skipping not single worker cluster test having a single worker node")
		}
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
