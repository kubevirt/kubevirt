package operands

import (
	"context"
	"fmt"
	"os"

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
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))

			res := handler.ensure(req)

			Expect(res.Created).To(BeTrue())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
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
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Overwritten).To(BeFalse())
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
			req.HCOTriggered = false                                  // mock a reconciliation triggered by a change in vm-import CR

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.UpgradeDone).To(BeFalse())
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

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
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
			hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewVMImportForCR(hcoNodePlacement)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
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

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewVMImportForCR(hco)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Infra.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Created).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.UpgradeDone).To(BeFalse())
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

		It("should overwrite node placement if directly set on VMImport CR", func() {
			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewVMImportForCR(hco)

			// mock a reconciliation triggered by a change in VMImport CR
			req.HCOTriggered = false

			// now, modify VMImport node placement
			seconds3 := int64(3)
			existingResource.Spec.Infra.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			existingResource.Spec.Infra.NodeSelector["key1"] = "BADvalue1"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newVmImportHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Infra.Tolerations).To(HaveLen(3))
			Expect(existingResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("BADvalue1"))

			Expect(foundResource.Spec.Infra.Tolerations).To(HaveLen(2))
			Expect(foundResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("value1"))

			Expect(req.Conditions).To(BeEmpty())
		})

		Context("Cache", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := newVmImportHandler(cl, commonTestUtils.GetScheme())

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*vmImportHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*vmImportHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*vmImportHooks).cache == cr).Should(BeTrue())

					cdi1, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi1).ToNot(BeNil())
					Expect(cr == cdi1).Should(BeTrue())
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*vmImportHooks).reset()
				Expect(handler.hooks.(*vmImportHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*vmImportHooks).cache).ToNot(BeNil())

				handler.hooks.(*vmImportHooks).reset()
				Expect(handler.hooks.(*vmImportHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*vmImportHooks).cache).ToNot(BeNil())

				Expect(crI == crII).To(BeFalse())
				Expect(handler.hooks.(*vmImportHooks).cache == crI).To(BeFalse())
				Expect(handler.hooks.(*vmImportHooks).cache == crII).To(BeTrue())
			})
		})
	})

	Context("Manage IMS Config", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("CONVERSION_CONTAINER", "new-conversion-container-value")
			os.Setenv("VMWARE_CONTAINER", "new-vmware-container-value")
			os.Setenv("VIRTIOWIN_CONTAINER", "new-virtiowin-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should error if CONVERSION_CONTAINER environment var not specified", func() {
			os.Unsetenv("CONVERSION_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(BeNil())
		})

		It("should error if VMWARE_CONTAINER environment var not specified", func() {
			os.Unsetenv("VMWARE_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(BeNil())
		})

		It("should error if VIRTIOWIN_CONTAINER environment var not specified", func() {
			os.Unsetenv("VIRTIOWIN_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(BeNil())
		})

		It("should create if not present", func() {
			expectedResource, err := NewIMSConfigForCR(hco, commonTestUtils.Namespace)
			Expect(err).ToNot(HaveOccurred())
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
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
			expectedResource, err := NewIMSConfigForCR(hco, commonTestUtils.Namespace)
			Expect(err).ToNot(HaveOccurred())

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile according to env values and HCO CR", func() {
			convk := "v2v-conversion-image"
			vmwarek := "kubevirt-vmware-image"
			virtiowink := "virtio-win-image"
			vddkk := "vddk-init-image"
			updatableKeys := [...]string{convk, vmwarek, virtiowink}
			toBeRemovedKey := "toberemoved"

			expectedResource, err := NewIMSConfigForCR(hco, commonTestUtils.Namespace)
			Expect(err).ToNot(HaveOccurred())

			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			outdatedResource, err := NewIMSConfigForCR(hco, commonTestUtils.Namespace)
			Expect(err).ToNot(HaveOccurred())

			outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
			// values we should update
			outdatedResource.Data[convk] = "old-conversion-container-value-we-have-to-update"
			outdatedResource.Data[vmwarek] = "old-vmware-container-value-we-have-to-update"
			outdatedResource.Data[virtiowink] = "old-virtiowin-container-value-we-have-to-update"

			// add values we should remove
			outdatedResource.Data[toBeRemovedKey] = "value-we-should-remove"

			vddkInitImageValue := "new-vddk-value-we-have-to-update"
			hco.Spec.VddkInitImage = &vddkInitImageValue

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			handler := (*genericOperand)(newImsConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())

			for _, k := range updatableKeys {
				Expect(foundResource.Data[k]).To(Not(Equal(outdatedResource.Data[k])))
				Expect(foundResource.Data[k]).To(Equal(expectedResource.Data[k]))
			}

			Expect(foundResource.Data).To(Not(HaveKey(toBeRemovedKey)))
			Expect(foundResource.Data).To(HaveKeyWithValue(vddkk, vddkInitImageValue))

		})

	})

})
