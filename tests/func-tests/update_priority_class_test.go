package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"kubevirt.io/client-go/kubecli"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const priorityClassName = "kubevirt-cluster-critical"

var _ = Describe("check update priorityClass", Ordered, Serial, func() {
	var (
		cli                 kubecli.KubevirtClient
		ctx                 context.Context
		oldPriorityClassUID types.UID
	)

	tests.FlagParse()

	getPriorityClassHCORef := func() types.UID {
		hc := tests.GetHCO(ctx, cli)

		for _, obj := range hc.Status.RelatedObjects {
			if obj.Kind == "PriorityClass" && obj.Name == priorityClassName {
				return obj.UID
			}
		}
		return ""
	}

	BeforeAll(func() {
		var err error
		cli, err = kubecli.GetKubevirtClient()
		Expect(err).ToNot(HaveOccurred())

		ctx = context.Background()
		pc, err := cli.SchedulingV1().PriorityClasses().Get(ctx, priorityClassName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(pc.UID).ToNot(BeEmpty())
		oldPriorityClassUID = pc.UID
	})

	It("should have the right reference for the priorityClass in the HyperConverged CR", func() {
		uid := getPriorityClassHCORef()
		Expect(uid).To(Equal(oldPriorityClassUID))
	})

	It("should recreate the priorityClass on update", func() {
		GinkgoWriter.Printf("oldPriorityClassUID: %q\n", oldPriorityClassUID)
		// `~1` is the jsonpatch escapoe sequence for `\`
		patch := []byte(`[{"op": "replace", "path": "/metadata/labels/app.kubernetes.io~1managed-by", "value": "test"}]`)

		Eventually(func() error {
			_, err := cli.SchedulingV1().PriorityClasses().Patch(ctx, priorityClassName, types.JSONPatchType, patch, metav1.PatchOptions{})
			return err
		}).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).Should(Succeed())

		var newUID types.UID
		Eventually(func(g Gomega) {
			By("make sure a new priority class was created, by checking its UID")
			pc, err := cli.SchedulingV1().PriorityClasses().Get(ctx, priorityClassName, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			newUID = pc.UID
			g.Expect(newUID).ToNot(Or(Equal(types.UID("")), Equal(oldPriorityClassUID)))
			g.Expect(pc.GetLabels()).ToNot(HaveKey("test"))
		}).WithTimeout(30 * time.Second).
			WithPolling(100 * time.Millisecond).
			Should(Succeed())

		GinkgoWriter.Printf("oldPriorityClassUID: %q; newUID: %q\n", oldPriorityClassUID, newUID)
		Eventually(getPriorityClassHCORef).
			WithTimeout(5 * time.Minute).
			WithPolling(time.Second).
			Should(And(Not(BeEmpty()), Equal(newUID)))
	})
})
