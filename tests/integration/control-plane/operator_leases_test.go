package controlplane_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	coordinationv1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1 "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/apimachinery/patch"
	"kubevirt.io/kubevirt/pkg/pointer"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/tests/integration/framework"
)

// Lease handlers delay queue additions by five seconds. Observe longer than
// that delay, including after the watch cache has received each API mutation.
const leaseObservationWindow = 8 * time.Second

var _ = Describe("Operator Lease events", func() {
	var f *framework.Framework
	var operator *framework.Operator
	const namespace = "kubevirt"

	BeforeEach(func(ctx SpecContext) {
		f = framework.New()
		f.Start(context.Background())
		operator = f.StartOperator()
		_, err := f.VirtClient().KubeVirt(namespace).Create(ctx, &v1.KubeVirt{
			ObjectMeta: metav1.ObjectMeta{Name: "kubevirt", Namespace: namespace},
			Spec:       v1.KubeVirtSpec{ImageRegistry: "quay.io/kubevirt", ImageTag: "latest"},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())

		// Envtest has no Job controller. The operator becomes idle waiting for
		// its install-strategy Job, without deploying the KubeVirt workloads.
		Eventually(ctx, func(g Gomega) {
			jobs, err := f.K8sClient().BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(jobs.Items).To(HaveLen(1))
			kv, err := f.VirtClient().KubeVirt(namespace).Get(ctx, "kubevirt", metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(kv.Status.Phase).To(Equal(v1.KubeVirtPhaseDeploying))
		}, 30*time.Second, 100*time.Millisecond).Should(Succeed())

		// Drain delayed startup work before attributing runs to Lease events.
		lastCount := operator.Reconciliations()
		lastChange := time.Now()
		Eventually(ctx, func() time.Duration {
			if count := operator.Reconciliations(); count != lastCount {
				lastCount = count
				lastChange = time.Now()
			}
			return time.Since(lastChange)
		}, 30*time.Second, 100*time.Millisecond).Should(BeNumerically(">=", leaseObservationWindow))
	})

	waitForLease := func(ctx SpecContext, lease *coordinationv1.Lease) {
		GinkgoHelper()
		Eventually(ctx, func(g Gomega) {
			obj, exists, err := operator.Leases.GetStore().GetByKey(namespace + "/" + lease.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(exists).To(BeTrue())
			g.Expect(obj.(*coordinationv1.Lease).ResourceVersion).To(Equal(lease.ResourceVersion))
		}, 5*time.Second, 50*time.Millisecond).Should(Succeed())
	}

	createLease := func(ctx SpecContext, name string, holder *string) *coordinationv1.Lease {
		GinkgoHelper()
		lease, err := f.K8sClient().CoordinationV1().Leases(namespace).Create(ctx, &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: holder,
				RenewTime:      pointer.P(metav1.NewMicroTime(time.Now())),
			},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		waitForLease(ctx, lease)
		return lease
	}

	patchLease := func(ctx SpecContext, lease *coordinationv1.Lease) *coordinationv1.Lease {
		GinkgoHelper()
		payload, err := patch.New(patch.WithReplace("/spec", lease.Spec)).GeneratePayload()
		Expect(err).NotTo(HaveOccurred())
		updated, err := f.K8sClient().CoordinationV1().Leases(namespace).Patch(ctx, lease.Name,
			types.JSONPatchType, payload, metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())
		Expect(updated.ResourceVersion).NotTo(Equal(lease.ResourceVersion))
		waitForLease(ctx, updated)
		return updated
	}

	deleteLease := func(ctx SpecContext, lease *coordinationv1.Lease) {
		GinkgoHelper()
		Expect(f.K8sClient().CoordinationV1().Leases(namespace).Delete(ctx, lease.Name, metav1.DeleteOptions{})).To(Succeed())
		Eventually(ctx, func(g Gomega) {
			_, exists, err := operator.Leases.GetStore().GetByKey(namespace + "/" + lease.Name)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(exists).To(BeFalse())
		}, 5*time.Second, 50*time.Millisecond).Should(Succeed())
	}

	expectReconciliation := func(ctx SpecContext, before int64) {
		GinkgoHelper()
		Eventually(ctx, operator.Reconciliations, 15*time.Second, 100*time.Millisecond).
			Should(BeNumerically(">", before), "Lease event did not reconcile KubeVirt")
	}

	It("reconciles creation, leader changes, renewals and deletion of unrelated Leases", func(ctx SpecContext) {
		// FIXME(#19342): Unrelated Lease events should not reconcile KubeVirt.
		before := operator.Reconciliations()
		By("creating an unrelated leader-election Lease")
		lease := createLease(ctx, "virt-controller", pointer.P("leader-a"))
		expectReconciliation(ctx, before)

		By("renewing it and changing its leader")
		before = operator.Reconciliations()
		lease.Spec.RenewTime = pointer.P(metav1.NewMicroTime(time.Now()))
		lease.Spec.HolderIdentity = pointer.P("leader-b")
		lease = patchLease(ctx, lease)
		expectReconciliation(ctx, before)

		By("deleting the unrelated Lease")
		before = operator.Reconciliations()
		deleteLease(ctx, lease)
		expectReconciliation(ctx, before)
	})

	It("reconciles synchronization Lease renewals and its leader lifecycle", func(ctx SpecContext) {
		By("creating the synchronization Lease without a leader")
		before := operator.Reconciliations()
		lease := createLease(ctx, components.VirtSynchronizationControllerName, nil)
		expectReconciliation(ctx, before)

		By("acquiring leadership")
		before = operator.Reconciliations()
		lease.Spec.HolderIdentity = pointer.P("leader-a")
		lease = patchLease(ctx, lease)
		expectReconciliation(ctx, before)

		By("renewing the Lease without changing leadership")
		// FIXME(#19342): Renewals with unchanged leadership should not reconcile KubeVirt.
		before = operator.Reconciliations()
		for range 3 {
			lease.Spec.RenewTime = pointer.P(metav1.NewMicroTime(time.Now()))
			lease = patchLease(ctx, lease)
		}
		expectReconciliation(ctx, before)

		By("replacing the leader")
		before = operator.Reconciliations()
		lease.Spec.HolderIdentity = pointer.P("leader-b")
		lease = patchLease(ctx, lease)
		expectReconciliation(ctx, before)

		By("releasing leadership")
		before = operator.Reconciliations()
		lease.Spec.HolderIdentity = pointer.P("")
		lease = patchLease(ctx, lease)
		expectReconciliation(ctx, before)

		By("deleting the synchronization Lease")
		before = operator.Reconciliations()
		deleteLease(ctx, lease)
		expectReconciliation(ctx, before)
	})
})
