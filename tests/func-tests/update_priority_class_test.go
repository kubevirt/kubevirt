package tests_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tests "github.com/kubevirt/hyperconverged-cluster-operator/tests/func-tests"
)

const priorityClassName = "kubevirt-cluster-critical"

var _ = Describe("check update priorityClass", Ordered, Serial, func() {
	var (
		cli                 client.Client
		cliSet              *kubernetes.Clientset
		oldPriorityClassUID types.UID
	)

	tests.FlagParse()

	BeforeAll(func(ctx context.Context) {
		var err error
		cli = tests.GetControllerRuntimeClient()
		cliSet = tests.GetK8sClientSet()

		pc, err := cliSet.SchedulingV1().PriorityClasses().Get(ctx, priorityClassName, metav1.GetOptions{})
		Expect(err).ToNot(HaveOccurred())

		Expect(pc.UID).ToNot(BeEmpty())
		oldPriorityClassUID = pc.UID
	})

	It("should have the right reference for the priorityClass in the HyperConverged CR", func(ctx context.Context) {
		uid := getPriorityClassHCORef(ctx, cli)
		Expect(uid).To(Equal(oldPriorityClassUID))
	})

	It("should recreate the priorityClass on update", func(ctx context.Context) {
		GinkgoWriter.Printf("oldPriorityClassUID: %q\n", oldPriorityClassUID)
		// `~1` is the jsonpatch escapoe sequence for `\`
		patch := []byte(`[{"op": "replace", "path": "/metadata/labels/app.kubernetes.io~1managed-by", "value": "test"}]`)

		Eventually(func(ctx context.Context) error {
			_, err := cliSet.SchedulingV1().PriorityClasses().Patch(ctx, priorityClassName, types.JSONPatchType, patch, metav1.PatchOptions{})
			return err
		}).WithTimeout(time.Second * 5).WithPolling(time.Millisecond * 100).WithContext(ctx).Should(Succeed())

		var newUID types.UID
		Eventually(func(g Gomega, ctx context.Context) {
			By("make sure a new priority class was created, by checking its UID")
			pc, err := cliSet.SchedulingV1().PriorityClasses().Get(ctx, priorityClassName, metav1.GetOptions{})
			g.Expect(err).ToNot(HaveOccurred())

			newUID = pc.UID
			g.Expect(newUID).ToNot(Or(Equal(types.UID("")), Equal(oldPriorityClassUID)))
			g.Expect(pc.GetLabels()).ToNot(HaveKey("test"))
		}).WithTimeout(30 * time.Second).
			WithPolling(100 * time.Millisecond).
			WithContext(ctx).
			Should(Succeed())

		GinkgoWriter.Printf("oldPriorityClassUID: %q; newUID: %q\n", oldPriorityClassUID, newUID)
		Eventually(func(ctx context.Context) types.UID {
			return getPriorityClassHCORef(ctx, cli)
		}).
			WithTimeout(5 * time.Minute).
			WithPolling(time.Second).
			WithContext(ctx).
			Should(And(Not(BeEmpty()), Equal(newUID)))
	})
})

func getPriorityClassHCORef(ctx context.Context, cli client.Client) types.UID {
	hc := tests.GetHCO(ctx, cli)

	for _, obj := range hc.Status.RelatedObjects {
		if obj.Kind == "PriorityClass" && obj.Name == priorityClassName {
			return obj.UID
		}
	}
	return ""
}
