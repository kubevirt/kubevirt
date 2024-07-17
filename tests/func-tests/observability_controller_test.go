package tests_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/observability"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

var _ = Describe("Observability Controller", Label(tests.OpenshiftLabel, "observability_controller"), func() {
	Context("PodDisruptionBudgetAtLimit", func() {
		It("should be silenced", func() {
			r := observability.NewReconciler(tests.GetClientConfig())

			amApi, err := r.NewAlertmanagerApi()
			Expect(err).ToNot(HaveOccurred())

			amSilences, err := amApi.ListSilences()
			Expect(err).ToNot(HaveOccurred())

			// PodDisruptionBudgetAtLimit silence should have been created by the controller
			podDisruptionBudgetAtLimitSilence := observability.FindPodDisruptionBudgetAtLimitSilence(amSilences)
			Expect(podDisruptionBudgetAtLimitSilence).ToNot(BeNil())

			err = amApi.DeleteSilence(podDisruptionBudgetAtLimitSilence.ID)
			Expect(err).ToNot(HaveOccurred())

			// Restart pod to force reconcile (reconcile periodicity is 1h)
			cli := tests.GetControllerRuntimeClient()
			var hcoPods v1.PodList
			err = cli.List(context.Background(), &hcoPods, &client.MatchingLabels{
				"name": "hyperconverged-cluster-operator",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(hcoPods.Items).ToNot(BeEmpty())

			for _, pod := range hcoPods.Items {
				err = cli.Delete(context.Background(), &pod)
				Expect(err).ToNot(HaveOccurred())
			}

			// Wait for the controller to recreate the silence
			Eventually(func() bool {
				amSilences, err := amApi.ListSilences()
				Expect(err).ToNot(HaveOccurred())

				return observability.FindPodDisruptionBudgetAtLimitSilence(amSilences) != nil
			}, "5m", "10s").Should(BeTrue())
		})
	})
})
