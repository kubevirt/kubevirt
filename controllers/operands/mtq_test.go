package operands

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	mtqv1alpha1 "kubevirt.io/managed-tenant-quota/staging/src/kubevirt.io/managed-tenant-quota-api/pkg/apis/core/v1alpha1"

	"github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("MTQ tests", func() {
	var (
		hco *v1beta1.HyperConverged
		req *common.HcoRequest
		cl  client.Client
	)

	getClusterInfo := hcoutil.GetClusterInfo

	BeforeEach(func() {
		hco = commontestutils.NewHco()
		req = commontestutils.NewReq(hco)
		hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
			return &commontestutils.ClusterInfoMock{}
		}
	})

	AfterEach(func() {
		hcoutil.GetClusterInfo = getClusterInfo
	})

	Context("check FG is deprecated", func() {
		It("should delete MTQ even if the FG is set", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			mtq := NewMTQWithNameOnly(hco)
			cl = commontestutils.InitClient([]client.Object{hco, mtq})

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Name).To(Equal(mtq.Name))
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeTrue())

			foundMTQs := &mtqv1alpha1.MTQList{}
			Expect(cl.List(context.Background(), foundMTQs)).To(Succeed())
			Expect(foundMTQs.Items).To(BeEmpty())
		})

		It("should not create MTQ even if the FG is set", func() {
			hco.Spec.FeatureGates.EnableManagedTenantQuota = ptr.To(true)
			cl = commontestutils.InitClient([]client.Object{hco})

			handler := newMtqHandler(cl, commontestutils.GetScheme())

			res := handler.ensure(req)

			Expect(res.Err).ToNot(HaveOccurred())
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Deleted).To(BeFalse())

			foundMTQs := &mtqv1alpha1.MTQList{}
			Expect(cl.List(context.Background(), foundMTQs)).To(Succeed())
			Expect(foundMTQs.Items).To(BeEmpty())
		})
	})
})
