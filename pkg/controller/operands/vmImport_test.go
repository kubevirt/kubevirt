package operands

import (
	"context"
	"fmt"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
)

var _ = Describe("VM-Import", func() {
	Context("Vm Import", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewVMImportForCR(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)

			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := NewVMImportForCR(hco)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := NewVMImportForCR(hco)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.ImagePullPolicy = corev1.PullAlways // set non-default value

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.ImagePullPolicy).To(BeEmpty())
		})

		It("should add node placement if missing in VM-Import", func() {
			existingResource := NewVMImportForCR(hco)

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Infra.Affinity).To(BeNil())
			Expect(existingResource.Spec.Infra.NodeSelector).To(BeEmpty())
			Expect(existingResource.Spec.Infra.Tolerations).To(BeEmpty())

			Expect(foundResource.Spec.Infra.Affinity).ToNot(BeNil())
			Expect(foundResource.Spec.Infra.NodeSelector).ToNot(BeEmpty())
			Expect(foundResource.Spec.Infra.Tolerations).ToNot(BeEmpty())

			infra := foundResource.Spec.Infra
			Expect(infra.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(infra.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(infra.Tolerations).Should(Equal(hco.Spec.Infra.NodePlacement.Tolerations))
			Expect(infra.Affinity).Should(Equal(hco.Spec.Infra.NodePlacement.Affinity))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should remove node placement if missing in HCO CR", func() {

			hcoNodePlacement := commonTestUtils.NewHco()
			hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			existingResource := NewVMImportForCR(hcoNodePlacement)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Infra.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(existingResource.Spec.Infra.NodeSelector).ToNot(BeEmpty())
			Expect(existingResource.Spec.Infra.Tolerations).ToNot(BeEmpty())

			Expect(foundResource.Spec.Infra.Affinity).To(BeNil())
			Expect(foundResource.Spec.Infra.NodeSelector).To(BeEmpty())
			Expect(foundResource.Spec.Infra.Tolerations).To(BeEmpty())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify node placement according to HCO CR", func() {

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			existingResource := NewVMImportForCR(hco)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Infra.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &VmImportHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Infra.Affinity).ToNot(BeNil())
			Expect(existingResource.Spec.Infra.Tolerations).To(HaveLen(2))
			Expect(existingResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("value1"))

			Expect(foundResource.Spec.Infra.Affinity).ToNot(BeNil())
			Expect(foundResource.Spec.Infra.Tolerations).To(HaveLen(3))
			Expect(foundResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("something else"))

			Expect(req.Conditions).To(BeEmpty())
		})
	})
})
