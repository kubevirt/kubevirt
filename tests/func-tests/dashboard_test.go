package tests_test

import (
	"context"
	"flag"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5108][crit:medium][vendor:cnv-qe@redhat.com][level:system]Dashboard configmaps", Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var (
		cli *kubernetes.Clientset
		ctx context.Context
	)

	BeforeEach(func() {
		tests.BeforeEach()

		k8sCli := tests.GetControllerRuntimeClient()
		ctx = context.Background()

		tests.FailIfNotOpenShift(ctx, k8sCli, "Dashboard configmaps")

		cli = tests.GetK8sClientSet()
	})

	It("[test_id:5919]should create configmaps for OCP Dashboard", Label("test_id:5919"), func() {
		By("Checking expected configmaps")
		items := tests.GetConfig().Dashboard.TestItems

		if len(items) == 0 {
			GinkgoLogr.Info("There is no test item for dashboard tests.")
		}

		for _, item := range items {
			cm, err := cli.CoreV1().ConfigMaps(item.Namespace).Get(ctx, item.Name, v1.GetOptions{})
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			for _, key := range item.Keys {
				_, ok := cm.Data[key]
				ExpectWithOffset(1, ok).Should(BeTrue())
			}
		}
	})
})
