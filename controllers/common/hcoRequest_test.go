package common

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("Test HcoRequest", func() {
	It("should set all the fields", func() {
		ctx := context.TODO()
		req := NewHcoRequest(
			ctx,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			logf.Log,
			false,
			true,
		)

		Expect(req.Name).Should(Equal("name"))
		Expect(req.Namespace).Should(Equal("namespace"))
		Expect(req.Ctx).Should(Equal(ctx))
		Expect(req.Conditions).ToNot(BeNil())
		Expect(req.Conditions).To(BeEmpty())
		Expect(req.UpgradeMode).To(BeFalse())
		Expect(req.ComponentUpgradeInProgress).To(BeFalse())
		Expect(req.Dirty).To(BeFalse())
		Expect(req.StatusDirty).To(BeFalse())
	})

	It("should set set upgrade mode to true", func() {
		ctx := context.TODO()
		req := NewHcoRequest(
			ctx,
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "name",
					Namespace: "namespace",
				},
			},
			logf.Log,
			false,
			true,
		)

		Expect(req.ComponentUpgradeInProgress).To(BeFalse())
		Expect(req.Dirty).To(BeFalse())

		req.SetUpgradeMode(true)
		Expect(req.UpgradeMode).To(BeTrue())
		Expect(req.ComponentUpgradeInProgress).To(BeTrue())

		req.SetUpgradeMode(false)
		Expect(req.UpgradeMode).To(BeFalse())
		Expect(req.ComponentUpgradeInProgress).To(BeFalse())
	})
})
