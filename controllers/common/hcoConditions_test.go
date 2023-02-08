package common

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
)

var _ = Describe("HCO Conditions Tests", func() {
	Context("Test SetStatusCondition", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should create new condition", func() {
			conds.SetStatusCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(metav1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))
		})

		It("Should update a condition if already exists", func() {
			conds.SetStatusCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionTrue,
				Reason:  "reason2",
				Message: "another message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(metav1.ConditionTrue))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason2"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("another message"))
		})
	})

	Context("Test SetStatusConditionIfUnset", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should not update the condition", func() {
			By("Set initial condition")
			conds.SetStatusConditionIfUnset(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(metav1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))

			By("The condition should not be changed by this call")
			conds.SetStatusConditionIfUnset(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionTrue,
				Reason:  "reason2",
				Message: "another message",
			})

			Expect(conds.IsEmpty()).To(BeFalse())
			Expect(conds).To(HaveLen(1))

			By("Make sure the values are the same as before and were not changed")
			Expect(conds[hcov1beta1.ConditionReconcileComplete]).ToNot(BeNil())
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Type).Should(Equal(hcov1beta1.ConditionReconcileComplete))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Status).Should(Equal(metav1.ConditionFalse))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Reason).Should(Equal("reason"))
			Expect(conds[hcov1beta1.ConditionReconcileComplete].Message).Should(Equal("a message"))
		})
	})

	Context("Test HasCondition", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should not contain the condition", func() {
			Expect(conds.HasCondition(hcov1beta1.ConditionReconcileComplete)).To(BeFalse())
			conds.SetStatusConditionIfUnset(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.HasCondition(hcov1beta1.ConditionReconcileComplete)).To(BeTrue())
			Expect(conds.HasCondition(hcov1beta1.ConditionAvailable)).To(BeFalse())
		})
	})

	Context("Test IsStatusConditionTrue", func() {
		conds := NewHcoConditions()
		Expect(conds.IsEmpty()).To(BeTrue())

		It("Should return false when the conditionType is not present", func() {
			Expect(conds.IsStatusConditionTrue(hcov1beta1.ConditionReconcileComplete)).To(BeFalse())
		})

		It("Should return false when the conditionType is present but not set to True", func() {
			conds.SetStatusConditionIfUnset(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionFalse,
				Reason:  "reason",
				Message: "a message",
			})

			Expect(conds.IsStatusConditionTrue(hcov1beta1.ConditionReconcileComplete)).To(BeFalse())
		})

		It("Should return true when the conditionType is present and set to True", func() {
			conds.SetStatusCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionReconcileComplete,
				Status:  metav1.ConditionTrue,
				Reason:  "reason2",
				Message: "another message",
			})

			Expect(conds.IsStatusConditionTrue(hcov1beta1.ConditionReconcileComplete)).To(BeTrue())
		})
	})
})
