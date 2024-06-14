package tests_test

import (
	"context"
	"flag"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("[rfe_id:5882][crit:high][vendor:cnv-qe@redhat.com][level:system]ConsoleQuickStart objects", Label(tests.OpenshiftLabel), func() {
	flag.Parse()

	var (
		cli client.Client
	)
	BeforeEach(func(ctx context.Context) {
		tests.BeforeEach(ctx)
		cli = tests.GetControllerRuntimeClient()

		tests.FailIfNotOpenShift(ctx, cli, "quickstart")
	})

	It("[test_id:5883]should create ConsoleQuickStart objects", Label("test_id:5883"), func(ctx context.Context) {
		By("Checking expected quickstart objects")
		s := scheme.Scheme
		_ = consolev1.Install(s)
		s.AddKnownTypes(consolev1.GroupVersion)

		items := tests.GetConfig().QuickStart.TestItems

		if len(items) == 0 {
			GinkgoLogr.Info("There is no quickstart test item for quickstart tests.")
		}

		for _, qs := range items {
			// use a fresh object for each loop. get requests only override non-empty fields
			cqs := &consolev1.ConsoleQuickStart{
				ObjectMeta: metav1.ObjectMeta{
					Name: qs.Name,
				},
			}

			Expect(cli.Get(ctx, client.ObjectKeyFromObject(cqs), cqs)).To(Succeed())

			Expect(cqs.Spec.DisplayName).To(Equal(qs.DisplayName))
		}
	})

})
