package tests_test

import (
	"context"
	"errors"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/observability"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/alertmanager"
	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const testName = "observability_controller"

var _ = Describe("Observability Controller", Label(tests.OpenshiftLabel, testName), func() {
	var cli client.Client

	BeforeEach(func(ctx context.Context) {
		cli = tests.GetControllerRuntimeClient()
		tests.FailIfNotOpenShift(ctx, cli, testName)
	})

	Context("PodDisruptionBudgetAtLimit", func() {
		BeforeEach(func(ctx context.Context) {
			certExists, err := serviceAccountTlsCertPathExists()
			Expect(err).ToNot(HaveOccurred())

			if !certExists {
				Fail("Service account TLS certificate path does not exist")
			}
		})

		It("should be silenced", func(ctx context.Context) {
			httpClient, err := observability.NewHTTPClient()
			Expect(err).ToNot(HaveOccurred())
			amApi := alertmanager.NewAPI(*httpClient, observability.AlertmanagerSvcHost, tests.GetClientConfig().BearerToken)

			amSilences, err := amApi.ListSilences()
			Expect(err).ToNot(HaveOccurred())

			// PodDisruptionBudgetAtLimit silence should have been created by the controller
			podDisruptionBudgetAtLimitSilence := observability.FindPodDisruptionBudgetAtLimitSilence(amSilences)
			Expect(podDisruptionBudgetAtLimitSilence).ToNot(BeNil())

			err = amApi.DeleteSilence(podDisruptionBudgetAtLimitSilence.ID)
			Expect(err).ToNot(HaveOccurred())

			// Restart pod to force reconcile (reconcile periodicity is 1h)
			var hcoPods v1.PodList
			err = cli.List(ctx, &hcoPods, &client.MatchingLabels{
				"name": "hyperconverged-cluster-operator",
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(hcoPods.Items).ToNot(BeEmpty())

			for _, pod := range hcoPods.Items {
				err = cli.Delete(ctx, &pod)
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

func serviceAccountTlsCertPathExists() (bool, error) {
	_, err := os.Stat(observability.ServiceAccountTlsCertPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}
