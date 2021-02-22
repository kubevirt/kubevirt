package common

import (
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("HCO Conditions Tests", func() {
	Context("Test SetStatusCondition", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should create new condition", func() {
			conds.SetStatusCondition(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(corev1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))
		})

		It("Should update a condition if already exists", func() {
			conds.SetStatusCondition(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionTrue,
				Reason:  "reason2",
				Message: "another message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(corev1.ConditionTrue))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason2"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("another message"))
		})
	})

	Context("Test SetStatusConditionIfUnset", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should not update the condition", func() {
			By("Set initial condition")
			conds.SetStatusConditionIfUnset(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(corev1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))

			By("The condition should not be changed by this call")
			conds.SetStatusConditionIfUnset(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionTrue,
				Reason:  "reason2",
				Message: "another message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			By("Make sure the values are the same as before and were not changed")
			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(corev1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))
		})
	})

	Context("Test HasCondition", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should not contain the condition", func() {
			Expect(conds.HasCondition(hcov1beta1.ConditionReconcileComplete)).To(BeFalse())
			conds.SetStatusConditionIfUnset(conditionsv1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  corev1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.HasCondition(hcov1beta1.ConditionReconcileComplete)).To(BeTrue())
			Expect(conds.HasCondition(conditionsv1.ConditionAvailable)).To(BeFalse())
		})
	})
})
