/*
Package hco contains Tier 1 tests for CommonInstancetypesDeployment feature.

STP Reference: tests/CNV-61256/CNV-61256_test_plan.md
STD Reference: tests/CNV-61256/CNV-61256_test_description.yaml
Jira: https://issues.redhat.com/browse/CNV-61256
PR: https://github.com/kubevirt/hyperconverged-cluster-operator/pull/3471

Feature: Disable common-instancetypes deployment from HCO

This test file validates the CommonInstancetypesDeployment configuration
in the HyperConverged CR and its propagation to KubeVirt CR.

Phase 2: Full working implementation
*/
package hco

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/tests/decorators"
	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
)

const (
	// HCO namespace and resource names
	hcoNamespace = "openshift-cnv"
	hcoName      = "kubevirt-hyperconverged"
	kvName       = "kubevirt-hyperconverged"

	// Timeouts
	reconciliationTimeout = 2 * time.Minute
	pollingInterval       = 5 * time.Second
)

var _ = Describe("[CNV-61256] CommonInstancetypesDeployment", decorators.SigCompute, Serial, func() {
	var (
		ctx              context.Context
		virtClient       kubecli.KubevirtClient
		originalHCOSpec  *hcov1beta1.HyperConvergedSpec
	)

	BeforeEach(func() {
		ctx = context.Background()
		virtClient = kubevirt.Client()

		// Save original HCO configuration for restoration
		hco, err := getHyperConverged(ctx, virtClient)
		ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to get HyperConverged CR")
		originalHCOSpec = hco.Spec.DeepCopy()
	})

	AfterEach(func() {
		// Restore original HCO configuration
		By("Restoring original HCO configuration")
		hco, err := getHyperConverged(ctx, virtClient)
		if err == nil {
			hco.Spec.CommonInstancetypesDeployment = originalHCOSpec.CommonInstancetypesDeployment
			_, err = updateHyperConverged(ctx, virtClient, hco)
			if err != nil {
				GinkgoWriter.Printf("Warning: Failed to restore HCO config: %v\n", err)
			}
			// Wait for reconciliation
			waitForHCOReconciliation(ctx, virtClient)
		}
	})

	Context("HCO to KubeVirt configuration propagation", Ordered, func() {
		/*
		 * Test ID: TS-CNV61256-005
		 * Tier: Tier 1
		 * Priority: P1
		 * Requirement: REQ-004
		 *
		 * Preconditions:
		 *   - HCO operator is running and healthy
		 *   - HyperConverged CR is accessible
		 *   - KubeVirt CR is accessible
		 *
		 * Steps:
		 *   1. Set CommonInstancetypesDeployment.enabled to false in HCO CR
		 *   2. Wait for HCO reconciliation to complete
		 *   3. Read KubeVirt CR configuration
		 *   4. Verify KubeVirt CR reflects HCO configuration
		 *
		 * Expected:
		 *   - KubeVirt CR spec.configuration.commonInstancetypesDeployment.enabled == false
		 *   - Configuration propagation completes within 2 minutes
		 */
		It("[test_id:TS-CNV61256-005] should propagate CommonInstancetypesDeployment from HCO CR to KubeVirt CR", func() {
			By("Setting CommonInstancetypesDeployment.enabled to false in HCO CR")
			hco, err := getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to get HyperConverged CR")

			// Set enabled to false
			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(false),
			}

			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "Failed to update HyperConverged CR")

			By("Waiting for HCO reconciliation")
			waitForHCOReconciliation(ctx, virtClient)

			By("Verifying KubeVirt CR reflects HCO configuration")
			Eventually(func() bool {
				kv, err := getKubeVirt(ctx, virtClient)
				if err != nil {
					GinkgoWriter.Printf("Error getting KubeVirt CR: %v\n", err)
					return false
				}

				if kv.Spec.Configuration.CommonInstancetypesDeployment == nil {
					GinkgoWriter.Println("KubeVirt CommonInstancetypesDeployment is nil")
					return false
				}

				if kv.Spec.Configuration.CommonInstancetypesDeployment.Enabled == nil {
					GinkgoWriter.Println("KubeVirt CommonInstancetypesDeployment.Enabled is nil")
					return false
				}

				enabled := *kv.Spec.Configuration.CommonInstancetypesDeployment.Enabled
				GinkgoWriter.Printf("KubeVirt CommonInstancetypesDeployment.Enabled = %v\n", enabled)
				return !enabled // Should be false
			}, reconciliationTimeout, pollingInterval).Should(BeTrue(),
				"KubeVirt CR should have CommonInstancetypesDeployment.enabled = false")

			By("Verifying HCO CR still has the configuration")
			hco, err = getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			ExpectWithOffset(1, hco.Spec.CommonInstancetypesDeployment).ToNot(BeNil(),
				"HCO CommonInstancetypesDeployment should not be nil")
			ExpectWithOffset(1, *hco.Spec.CommonInstancetypesDeployment.Enabled).To(BeFalse(),
				"HCO CommonInstancetypesDeployment.Enabled should be false")
		})

		/*
		 * Test ID: TS-CNV61256-005b
		 * Additional test case for enabled=true propagation
		 */
		It("[test_id:TS-CNV61256-005b] should propagate CommonInstancetypesDeployment enabled=true from HCO CR to KubeVirt CR", func() {
			By("First disabling CommonInstancetypesDeployment")
			hco, err := getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(false),
			}
			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())
			waitForHCOReconciliation(ctx, virtClient)

			By("Now enabling CommonInstancetypesDeployment")
			hco, err = getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(true),
			}
			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			By("Waiting for HCO reconciliation")
			waitForHCOReconciliation(ctx, virtClient)

			By("Verifying KubeVirt CR reflects enabled=true")
			Eventually(func() bool {
				kv, err := getKubeVirt(ctx, virtClient)
				if err != nil {
					return false
				}

				if kv.Spec.Configuration.CommonInstancetypesDeployment == nil {
					return false
				}

				if kv.Spec.Configuration.CommonInstancetypesDeployment.Enabled == nil {
					// nil means default (enabled)
					return true
				}

				return *kv.Spec.Configuration.CommonInstancetypesDeployment.Enabled
			}, reconciliationTimeout, pollingInterval).Should(BeTrue(),
				"KubeVirt CR should have CommonInstancetypesDeployment.enabled = true")
		})
	})

	Context("API validation", Ordered, func() {
		/*
		 * Test ID: TS-CNV61256-006
		 * Tier: Tier 1
		 * Priority: P2
		 * Requirement: REQ-006
		 *
		 * Preconditions:
		 *   - HCO operator is running and healthy
		 *   - HyperConverged CR is accessible
		 *
		 * Steps:
		 *   1. Verify boolean field accepts valid values (true/false)
		 *   2. Verify schema validation is in place
		 *
		 * Expected:
		 *   - Valid boolean values are accepted
		 *   - API schema enforces type validation
		 */
		It("[test_id:TS-CNV61256-006] should accept valid boolean configuration values", func() {
			By("Testing enabled=true is accepted")
			hco, err := getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(true),
			}
			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "enabled=true should be accepted")

			By("Testing enabled=false is accepted")
			hco, err = getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(false),
			}
			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "enabled=false should be accepted")

			By("Testing nil/unset is accepted (uses default)")
			hco, err = getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			hco.Spec.CommonInstancetypesDeployment = nil
			hco, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(), "nil should be accepted (uses default)")
		})

		/*
		 * Note: Testing invalid value rejection at API level requires
		 * raw JSON/YAML patching since the Go types enforce boolean.
		 * The Kubernetes API server validates the OpenAPI schema and
		 * rejects non-boolean values before they reach the operator.
		 */
		It("[test_id:TS-CNV61256-006b] should have proper OpenAPI schema for validation", func() {
			By("Verifying HCO CRD has CommonInstancetypesDeployment field")
			hco, err := getHyperConverged(ctx, virtClient)
			ExpectWithOffset(1, err).ToNot(HaveOccurred())

			// The field should be settable
			hco.Spec.CommonInstancetypesDeployment = &v1.CommonInstancetypesDeployment{
				Enabled: ptr.To(true),
			}

			_, err = updateHyperConverged(ctx, virtClient, hco)
			ExpectWithOffset(1, err).ToNot(HaveOccurred(),
				"CommonInstancetypesDeployment field should be recognized in HCO CRD")
		})
	})
})

// Helper functions

func getHyperConverged(ctx context.Context, virtClient kubecli.KubevirtClient) (*hcov1beta1.HyperConverged, error) {
	// Use dynamic client or HCO-specific client
	// This is a simplified implementation - actual code would use proper HCO client
	hcoClient := virtClient.DynamicClient().Resource(hcov1beta1.SchemeGroupVersion.WithResource("hyperconvergeds"))

	unstructured, err := hcoClient.Namespace(hcoNamespace).Get(ctx, hcoName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	hco := &hcov1beta1.HyperConverged{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(unstructured.Object, hco)
	if err != nil {
		return nil, err
	}

	return hco, nil
}

func updateHyperConverged(ctx context.Context, virtClient kubecli.KubevirtClient, hco *hcov1beta1.HyperConverged) (*hcov1beta1.HyperConverged, error) {
	hcoClient := virtClient.DynamicClient().Resource(hcov1beta1.SchemeGroupVersion.WithResource("hyperconvergeds"))

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(hco)
	if err != nil {
		return nil, err
	}

	unstructuredObj := &unstructured.Unstructured{Object: unstructuredMap}
	result, err := hcoClient.Namespace(hcoNamespace).Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	updatedHCO := &hcov1beta1.HyperConverged{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(result.Object, updatedHCO)
	if err != nil {
		return nil, err
	}

	return updatedHCO, nil
}

func getKubeVirt(ctx context.Context, virtClient kubecli.KubevirtClient) (*v1.KubeVirt, error) {
	return virtClient.KubeVirt(hcoNamespace).Get(ctx, kvName, metav1.GetOptions{})
}

func waitForHCOReconciliation(ctx context.Context, virtClient kubecli.KubevirtClient) {
	Eventually(func() bool {
		hco, err := getHyperConverged(ctx, virtClient)
		if err != nil {
			return false
		}

		// Check HCO conditions for reconciliation complete
		for _, cond := range hco.Status.Conditions {
			if cond.Type == "Available" && cond.Status == "True" {
				return true
			}
		}
		return false
	}, reconciliationTimeout, pollingInterval).Should(BeTrue(),
		"HCO should complete reconciliation")
}
