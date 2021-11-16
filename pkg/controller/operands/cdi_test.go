package operands

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"

	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("CDI Operand", func() {
	Context("CDI", func() {
		var (
			hco *hcov1beta1.HyperConverged
			req *common.HcoRequest
		)

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewCDIWithNameOnly(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			Expect(foundResource.Annotations).To(Equal(map[string]string{cdiConfigAuthorityAnnotation: ""}))
		})

		It("should find if present", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "CDIConditions",
				Message: "CDI resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "CDIConditions",
				Message: "CDI resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "CDIConditions",
				Message: "CDI resource has no conditions",
			}))
		})

		It("should set default UninstallStrategy if missing", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			missingUSResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			missingUSResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/%s/dummies/%s", missingUSResource.Namespace, missingUSResource.Name)
			missingUSResource.Spec.UninstallStrategy = nil

			cl := commonTestUtils.InitClient([]runtime.Object{hco, missingUSResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(*foundResource.Spec.UninstallStrategy).To(Equal(cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist))
		})

		Context("Test node placement", func() {
			It("should add node placement if missing in CDI", func() {
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.Affinity).To(BeNil())
				Expect(existingResource.Spec.Infra.Tolerations).To(BeEmpty())
				Expect(existingResource.Spec.Infra.NodeSelector).To(BeNil())
				Expect(existingResource.Spec.Workloads.Affinity).To(BeNil())
				Expect(existingResource.Spec.Workloads.Tolerations).To(BeEmpty())
				Expect(existingResource.Spec.Workloads.NodeSelector).To(BeNil())

				Expect(foundResource.Spec.Infra.Affinity).ToNot(BeNil())
				Expect(foundResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(foundResource.Spec.Infra.NodeSelector["key2"]).Should(Equal("value2"))

				Expect(foundResource.Spec.Workloads).ToNot(BeNil())
				Expect(foundResource.Spec.Workloads.Tolerations).Should(Equal(hco.Spec.Workloads.NodePlacement.Tolerations))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should remove node placement if missing in HCO CR", func() {

				hcoNodePlacement := commonTestUtils.NewHco()
				hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				existingResource, err := NewCDI(hcoNodePlacement)
				Expect(err).ToNot(HaveOccurred())

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.Affinity).ToNot(BeNil())
				Expect(existingResource.Spec.Infra.Tolerations).ToNot(BeEmpty())
				Expect(existingResource.Spec.Infra.NodeSelector).ToNot(BeNil())
				Expect(existingResource.Spec.Workloads.Affinity).ToNot(BeNil())
				Expect(existingResource.Spec.Workloads.Tolerations).ToNot(BeEmpty())
				Expect(existingResource.Spec.Workloads.NodeSelector).ToNot(BeNil())

				Expect(foundResource.Spec.Infra.Affinity).To(BeNil())
				Expect(foundResource.Spec.Infra.Tolerations).To(BeEmpty())
				Expect(foundResource.Spec.Infra.NodeSelector).To(BeNil())
				Expect(foundResource.Spec.Workloads.Affinity).To(BeNil())
				Expect(foundResource.Spec.Workloads.Tolerations).To(BeEmpty())
				Expect(foundResource.Spec.Workloads.NodeSelector).To(BeNil())

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should modify node placement according to HCO CR", func() {
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				// now, modify HCO's node placement
				seconds3 := int64(3)
				hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})

				hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.Workloads.NodeSelector["key1"]).Should(Equal("value1"))

				Expect(foundResource.Spec.Infra.Tolerations).To(HaveLen(3))
				Expect(foundResource.Spec.Workloads.NodeSelector["key1"]).Should(Equal("something else"))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite node placement if directly set on CDI CR", func() {
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in CDI CR
				req.HCOTriggered = false

				// now, modify CDI's node placement
				seconds3 := int64(3)
				existingResource.Spec.Infra.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})
				existingResource.Spec.Workloads.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})

				existingResource.Spec.Infra.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodeSelector["key2"] = "BADvalue2"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodeSelector["key2"]).Should(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodeSelector["key2"]).Should(Equal("value2"))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

		Context("Test Resource Requirements", func() {
			It("should add Resource Requirements if missing in CDI", func() {
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					StorageWorkloads: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Spec.Config).ToNot(BeNil())
				Expect(foundResource.Spec.Config.PodResourceRequirements).ToNot(BeNil())
				Expect(foundResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceCPU]).Should(Equal(resource.MustParse("500m")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceMemory]).Should(Equal(resource.MustParse("2Gi")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceCPU]).Should(Equal(resource.MustParse("250m")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceMemory]).Should(Equal(resource.MustParse("1Gi")))
			})

			It("should remove Resource Requirements if missing in HCO CR", func() {

				hcoResourceRequirements := commonTestUtils.NewHco()
				hcoResourceRequirements.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					StorageWorkloads: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				}

				existingResource, err := NewCDI(hcoResourceRequirements)
				Expect(err).ToNot(HaveOccurred())

				Expect(existingResource.Spec.Config).ToNot(BeNil())
				Expect(existingResource.Spec.Config.PodResourceRequirements).ToNot(BeNil())
				Expect(existingResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceCPU]).Should(Equal(resource.MustParse("500m")))
				Expect(existingResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceMemory]).Should(Equal(resource.MustParse("2Gi")))
				Expect(existingResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceCPU]).Should(Equal(resource.MustParse("250m")))
				Expect(existingResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceMemory]).Should(Equal(resource.MustParse("1Gi")))

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Spec.Config).ToNot(BeNil())
				Expect(foundResource.Spec.Config.PodResourceRequirements).To(BeNil())
			})

			It("should modify Resource Requirements according to HCO CR", func() {
				hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
					StorageWorkloads: &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("250m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				}
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Spec.ResourceRequirements.StorageWorkloads.Limits[corev1.ResourceCPU] = resource.MustParse("1024m")
				hco.Spec.ResourceRequirements.StorageWorkloads.Limits[corev1.ResourceMemory] = resource.MustParse("4Gi")
				hco.Spec.ResourceRequirements.StorageWorkloads.Requests[corev1.ResourceCPU] = resource.MustParse("500m")
				hco.Spec.ResourceRequirements.StorageWorkloads.Requests[corev1.ResourceMemory] = resource.MustParse("2Gi")

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundResource := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Spec.Config.PodResourceRequirements.Limits).To(HaveLen(2))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceCPU]).Should(Equal(resource.MustParse("1024m")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("4Gi")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Requests).To(HaveLen(2))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceCPU]).Should(Equal(resource.MustParse("500m")))
				Expect(foundResource.Spec.Config.PodResourceRequirements.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("2Gi")))
			})
		})

		Context("Test ScratchSpaceStorageClass", func() {

			hcoScratchSpaceStorageClassValue := "hcoScratchSpaceStorageClassValue"
			cdiScratchSpaceStorageClassValue := "cdiScratchSpaceStorageClassValue"

			It("should add ScratchSpaceStorageClass if missing in CDI", func() {
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())
				hco.Spec.ScratchSpaceStorageClass = &hcoScratchSpaceStorageClassValue

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCdi := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundCdi),
				).To(BeNil())

				Expect(foundCdi.Spec.Config).ToNot(BeNil())
				Expect(foundCdi.Spec.Config.ScratchSpaceStorageClass).ToNot(BeNil())
				Expect(*foundCdi.Spec.Config.ScratchSpaceStorageClass).Should(Equal(hcoScratchSpaceStorageClassValue))
			})

			It("should remove ScratchSpaceStorageClass if missing in HCO CR", func() {
				hcoResourceRequirements := commonTestUtils.NewHco()

				existingCdi, err := NewCDI(hcoResourceRequirements)
				Expect(err).ToNot(HaveOccurred())
				existingCdi.Spec.Config.ScratchSpaceStorageClass = &cdiScratchSpaceStorageClassValue

				Expect(existingCdi.Spec.Config).ToNot(BeNil())
				Expect(existingCdi.Spec.Config.ScratchSpaceStorageClass).ToNot(BeNil())
				Expect(*existingCdi.Spec.Config.ScratchSpaceStorageClass).Should(Equal(cdiScratchSpaceStorageClassValue))

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingCdi})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCDI := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingCdi.Name, Namespace: existingCdi.Namespace},
						foundCDI),
				).To(BeNil())

				Expect(foundCDI.Spec.Config).ToNot(BeNil())
				Expect(foundCDI.Spec.Config.ScratchSpaceStorageClass).To(BeNil())
			})

			It("should modify ScratchSpaceStorageClass according to HCO CR", func() {
				hco.Spec.ScratchSpaceStorageClass = &cdiScratchSpaceStorageClassValue
				existingCDI, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				Expect(existingCDI.Spec.Config).ToNot(BeNil())
				Expect(*existingCDI.Spec.Config.ScratchSpaceStorageClass).To(Equal(cdiScratchSpaceStorageClassValue))

				hco.Spec.ScratchSpaceStorageClass = &hcoScratchSpaceStorageClassValue

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingCDI})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCDI := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingCDI.Name, Namespace: existingCDI.Namespace},
						foundCDI),
				).To(BeNil())

				Expect(foundCDI.Spec.Config.ScratchSpaceStorageClass).ToNot(BeNil())
				Expect(*foundCDI.Spec.Config.ScratchSpaceStorageClass).To(Equal(hcoScratchSpaceStorageClassValue))
			})
		})

		Context("Test StorageImport", func() {

			It("should add InsecureRegistries if exists in HC and missing in CDI", func() {
				existingResource, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())
				hco.Spec.StorageImport = &hcov1beta1.StorageImportConfig{
					InsecureRegistries: []string{"first:5000", "second:5000", "third:5000"},
				}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCdi := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundCdi),
				).To(BeNil())

				Expect(foundCdi.Spec.Config).ToNot(BeNil())
				Expect(foundCdi.Spec.Config.InsecureRegistries).ToNot(BeEmpty())
				Expect(foundCdi.Spec.Config.InsecureRegistries).Should(HaveLen(3))
				Expect(foundCdi.Spec.Config.InsecureRegistries).Should(ContainElements("first:5000", "second:5000", "third:5000"))
			})

			It("should remove InsecureRegistries if missing in HCO CR", func() {
				existingCdi, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())
				existingCdi.Spec.Config.InsecureRegistries = []string{"first:5000", "second:5000", "third:5000"}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingCdi})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCDI := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingCdi.Name, Namespace: existingCdi.Namespace},
						foundCDI),
				).To(BeNil())

				Expect(foundCDI.Spec.Config).ToNot(BeNil())
				Expect(foundCDI.Spec.Config.InsecureRegistries).To(BeNil())
			})

			It("should modify InsecureRegistries according to HCO CR", func() {
				existingCDI, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())
				existingCDI.Spec.Config.InsecureRegistries = []string{"first:5000", "second:5000", "third:5000"}

				hco.Spec.StorageImport = &hcov1beta1.StorageImportConfig{
					InsecureRegistries: []string{"other1:5000", "other2:5000"},
				}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingCDI})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.Err).To(BeNil())

				foundCDI := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingCDI.Name, Namespace: existingCDI.Namespace},
						foundCDI),
				).To(BeNil())

				Expect(foundCDI.Spec.Config.InsecureRegistries).To(HaveLen(2))
				Expect(foundCDI.Spec.Config.InsecureRegistries).To(ContainElements("other1:5000", "other2:5000"))
			})
		})

		It("should override CDI config field", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)

			// mock a reconciliation triggered by a change in CDI CR
			req.HCOTriggered = false

			// modify a cfg
			storageClass := "aa"
			proxyURLOverride := "proxyOverride"
			expectedResource.Spec.Config = &cdiv1beta1.CDIConfigSpec{
				UploadProxyURLOverride:   &proxyURLOverride,
				ScratchSpaceStorageClass: &storageClass,
				PodResourceRequirements:  &corev1.ResourceRequirements{},
				FeatureGates:             []string{"SomeFeatureGate"},
				FilesystemOverhead:       &cdiv1beta1.FilesystemOverhead{Global: "5"},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Config).ToNot(BeNil())
			// contains all that was found
			Expect(foundResource.Spec.Config.UploadProxyURLOverride).To(BeNil())
			Expect(foundResource.Spec.Config.ScratchSpaceStorageClass).To(BeNil())
			Expect(foundResource.Spec.Config.PodResourceRequirements).To(BeNil())
			Expect(foundResource.Spec.Config.FilesystemOverhead).To(BeNil())
			Expect(foundResource.Spec.Config.FeatureGates).To(HaveLen(1))
			Expect(foundResource.Spec.Config.FeatureGates).To(ContainElement("HonorWaitForFirstConsumer"))
			Expect(*foundResource.Spec.UninstallStrategy).To(Equal(cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist))
		})

		It("should add HonorWaitForFirstConsumer feature gate if Spec.Config if empty", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			expectedResource.Spec.Config = nil

			// mock a reconciliation triggered by a change in CDI CR
			req.HCOTriggered = false

			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Config).ToNot(BeNil())
			Expect(foundResource.Spec.Config.FeatureGates).To(ContainElement("HonorWaitForFirstConsumer"))
			Expect(*foundResource.Spec.UninstallStrategy).To(Equal(cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist))
		})

		It("should add cert configuration if missing in CDI", func() {
			existingResource := NewCDIWithNameOnly(hco)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)

			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.CertConfig).To(BeNil())

			Expect(foundResource.Spec.CertConfig).ToNot(BeNil())
			Expect(foundResource.Spec.CertConfig.CA.Duration.Duration.String()).Should(Equal("48h0m0s"))
			Expect(foundResource.Spec.CertConfig.CA.RenewBefore.Duration.String()).Should(Equal("24h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.Duration.Duration.String()).Should(Equal("24h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.RenewBefore.Duration.String()).Should(Equal("12h0m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should set cert config to defaults if missing in HCO CR", func() {
			existingResource := NewCDIWithNameOnly(hco)

			cl := commonTestUtils.InitClient([]runtime.Object{hco})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.CertConfig).To(BeNil())

			Expect(foundResource.Spec.CertConfig).ToNot(BeNil())
			Expect(foundResource.Spec.CertConfig.CA.Duration.Duration.String()).Should(Equal("48h0m0s"))
			Expect(foundResource.Spec.CertConfig.CA.RenewBefore.Duration.String()).Should(Equal("24h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.Duration.Duration.String()).Should(Equal("24h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.RenewBefore.Duration.String()).Should(Equal("12h0m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify cert configuration according to HCO CR", func() {
			hcoCertConfig := commonTestUtils.NewHco()

			existingResource, err := NewCDI(hcoCertConfig)
			Expect(err).ToNot(HaveOccurred())

			hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
				CA: hcov1beta1.CertRotateConfigCA{
					Duration:    metav1.Duration{Duration: 5 * time.Hour},
					RenewBefore: metav1.Duration{Duration: 6 * time.Hour},
				},
				Server: hcov1beta1.CertRotateConfigServer{
					Duration:    metav1.Duration{Duration: 7 * time.Hour},
					RenewBefore: metav1.Duration{Duration: 8 * time.Hour},
				},
			}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &cdiv1beta1.CDI{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.CertConfig).ToNot(BeNil())
			Expect(existingResource.Spec.CertConfig.CA.Duration.Duration.String()).Should(Equal("48h0m0s"))
			Expect(existingResource.Spec.CertConfig.CA.RenewBefore.Duration.String()).Should(Equal("24h0m0s"))
			Expect(existingResource.Spec.CertConfig.Server.Duration.Duration.String()).Should(Equal("24h0m0s"))
			Expect(existingResource.Spec.CertConfig.Server.RenewBefore.Duration.String()).Should(Equal("12h0m0s"))

			Expect(foundResource.Spec.CertConfig).ToNot(BeNil())
			Expect(foundResource.Spec.CertConfig.CA.Duration.Duration.String()).Should(Equal("5h0m0s"))
			Expect(foundResource.Spec.CertConfig.CA.RenewBefore.Duration.String()).Should(Equal("6h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.Duration.Duration.String()).Should(Equal("7h0m0s"))
			Expect(foundResource.Spec.CertConfig.Server.RenewBefore.Duration.String()).Should(Equal("8h0m0s"))
			Expect(req.Conditions).To(BeEmpty())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		It("should handle conditions", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			expectedResource.Status.Conditions = []conditionsv1.Condition{
				{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "Foo",
					Message: "Bar",
				},
				{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "Foo",
					Message: "Bar",
				},
				{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "Foo",
					Message: "Bar",
				},
			}
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "CDINotAvailable",
				Message: "CDI is not available: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "CDIProgressing",
				Message: "CDI is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "CDIProgressing",
				Message: "CDI is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionDegraded]).To(commonTestUtils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "CDIDegraded",
				Message: "CDI is degraded: Bar",
			}))
		})

		Context("Jsonpatch Annotation", func() {
			It("Should create CDI object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/config/featureGates/-",
						"value": "fg1"
					},
					{
						"op": "add",
						"path": "/spec/config/filesystemOverhead",
						"value": {"global": "50", "storageClass": {"AAA": "75", "BBB": "25"}}
					}
				]`}

				cdi, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cdi.Spec.Config.FeatureGates).To(HaveLen(2))
				Expect(cdi.Spec.Config.FeatureGates).To(ContainElement("fg1"))
				Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
				Expect(cdi.Spec.Config.FilesystemOverhead.Global).Should(BeEquivalentTo("50"))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).To(HaveLen(2))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).Should(BeEquivalentTo("75"))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).Should(BeEquivalentTo("25"))
			})

			It("Should fail to create CDI object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/config/featureGates/-",
						"value": "fg1"
					}
				]`}

				_, err := NewCDI(hco)
				Expect(err).To(HaveOccurred())
			})

			It("Ensure func should create CDI object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/config/featureGates/-",
						"value": "fg1"
					},
					{
						"op": "add",
						"path": "/spec/config/filesystemOverhead",
						"value": {"global": "50", "storageClass": {"AAA": "75", "BBB": "25"}}
					}
				]`}

				expectedResource := NewCDIWithNameOnly(hco)
				cl := commonTestUtils.InitClient([]runtime.Object{})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).To(BeNil())

				cdi := &cdiv1beta1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cdi),
				).ToNot(HaveOccurred())

				Expect(cdi.Spec.Config.FeatureGates).To(HaveLen(2))
				Expect(cdi.Spec.Config.FeatureGates).To(ContainElement("fg1"))
				Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
				Expect(cdi.Spec.Config.FilesystemOverhead.Global).Should(BeEquivalentTo("50"))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).To(HaveLen(2))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).Should(BeEquivalentTo("75"))
				Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).Should(BeEquivalentTo("25"))
			})

			It("Ensure func should fail to create CDI object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/config/featureGates/-",
						"value": "fg1"
					}
				]`}

				expectedResource := NewCDIWithNameOnly(hco)
				cl := commonTestUtils.InitClient([]runtime.Object{})
				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				cdi := &cdiv1beta1.CDI{}

				err := cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					cdi)

				Expect(err).To(HaveOccurred())
				Expect(errors.IsNotFound(err)).To(BeTrue())
			})

			It("Ensure func should update CDI object with changes from the annotation", func() {
				existsCdi, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/cloneStrategyOverride",
						"value": "copy"
					},
					{
						"op": "add",
						"path": "/spec/ImagePullPolicy",
						"value": "Always"
					}
				]`}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existsCdi})

				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())
				Expect(res.UpgradeDone).To(BeFalse())

				cdi := &cdiv1beta1.CDI{}

				expectedResource := NewCDIWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cdi),
				).ToNot(HaveOccurred())

				Expect(cdi.Spec.ImagePullPolicy).Should(BeEquivalentTo("Always"))
				Expect(cdi.Spec.CloneStrategyOverride).ToNot(BeNil())
				Expect(*cdi.Spec.CloneStrategyOverride).Should(BeEquivalentTo("copy"))
			})

			It("Ensure func should fail to update CDI object with wrong jsonPatch", func() {
				existsCdi, err := NewCDI(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchCDIAnnotationName: `[
					{
						"op": "notExistsOp",
						"path": "/spec/cloneStrategyOverride",
						"value": "copy"
					},
					{
						"op": "add",
						"path": "/spec/ImagePullPolicy",
						"value": "Always"
					}
				]`}

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existsCdi})

				handler := (*genericOperand)(newCdiHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				cdi := &cdiv1beta1.CDI{}

				expectedResource := NewCDIWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cdi),
				).ToNot(HaveOccurred())

				Expect(cdi.Spec.ImagePullPolicy).Should(BeEmpty())
				Expect(cdi.Spec.CloneStrategyOverride).To(BeNil())

			})
		})

		Context("Cache", func() {
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := newCdiHandler(cl, commonTestUtils.GetScheme())

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*cdiHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*cdiHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*cdiHooks).cache == cr).Should(BeTrue())

					cdi1, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi1).ToNot(BeNil())
					Expect(cr == cdi1).Should(BeTrue())
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*cdiHooks).reset()
				Expect(handler.hooks.(*cdiHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*cdiHooks).cache).ToNot(BeNil())

				handler.hooks.(*cdiHooks).reset()
				Expect(handler.hooks.(*cdiHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*cdiHooks).cache).ToNot(BeNil())

				Expect(crI == crII).To(BeFalse())
				Expect(handler.hooks.(*cdiHooks).cache == crI).To(BeFalse())
				Expect(handler.hooks.(*cdiHooks).cache == crII).To(BeTrue())
			})
		})

		Context("Config Reader Role", func() {
			It("should do nothing if exists", func() {
				expectedRole := NewConfigReaderRoleForCR(hco, hco.Namespace)
				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRole})

				handler := (*genericOperand)(newConfigReaderRoleHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundRole := &rbacv1.Role{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
						foundRole),
				).ToNot(HaveOccurred())

				Expect(expectedRole.ObjectMeta).Should(Equal(foundRole.ObjectMeta))
				Expect(expectedRole.Rules).Should(Equal(foundRole.Rules))
			})

			It("should update if labels are missing", func() {
				expectedRole := NewConfigReaderRoleForCR(hco, hco.Namespace)
				expectedLabels := expectedRole.Labels
				expectedRole.Labels = nil

				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRole})

				handler := (*genericOperand)(newConfigReaderRoleHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundRole := &rbacv1.Role{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedRole.Name, Namespace: expectedRole.Namespace},
						foundRole),
				).ToNot(HaveOccurred())

				Expect(reflect.DeepEqual(expectedLabels, foundRole.Labels)).To(BeTrue())
			})
		})

		Context("Config Reader Role Binding", func() {
			It("should do nothing if exists", func() {
				expectedRoleBinding := NewConfigReaderRoleBindingForCR(hco, hco.Namespace)

				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRoleBinding})

				handler := (*genericOperand)(newConfigReaderRoleBindingHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundRoleBinding := &rbacv1.RoleBinding{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
						foundRoleBinding),
				).ToNot(HaveOccurred())

				Expect(reflect.DeepEqual(expectedRoleBinding.Labels, foundRoleBinding.Labels)).To(BeTrue())
			})

			It("should update if labels are missing", func() {
				expectedRoleBinding := NewConfigReaderRoleBindingForCR(hco, hco.Namespace)
				expectedLabels := expectedRoleBinding.Labels
				expectedRoleBinding.Labels = nil

				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedRoleBinding})

				handler := (*genericOperand)(newConfigReaderRoleBindingHandler(cl, commonTestUtils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())

				foundRoleBinding := &rbacv1.RoleBinding{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedRoleBinding.Name, Namespace: expectedRoleBinding.Namespace},
						foundRoleBinding),
				).ToNot(HaveOccurred())

				Expect(reflect.DeepEqual(expectedLabels, foundRoleBinding.Labels)).To(BeTrue())
			})
		})
	})

	Context("KubeVirt Storage Config", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newStorageConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
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
			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newStorageConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should update if created in the past with a different configuration", func() {
			newKeys := [...]string{"ocs-storagecluster-ceph-rbd.accessMode", "ocs-storagecluster-ceph-rbd.volumeMode"}

			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			outdatedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			// remove value that wasn't there in the past
			for _, k := range newKeys {
				delete(outdatedResource.Data, k)
			}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			handler := (*genericOperand)(newStorageConfigHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())

			for _, k := range newKeys {
				Expect(expectedResource.Data).To(HaveKey(k))
				Expect(outdatedResource.Data).To(Not(HaveKey(k)))
				Expect(foundResource.Data).To(HaveKey(k))
				Expect(foundResource.Data[k]).To(Equal(expectedResource.Data[k]))
			}

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, outdatedResource)
			Expect(err).To(BeNil())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).To(BeNil())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		It("local storage class name should be available when specified", func() {
			hco.Spec.LocalStorageClassName = "local"

			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			Expect(expectedResource.Data["local.accessMode"]).To(Equal("ReadWriteOnce"))
			Expect(expectedResource.Data["local.volumeMode"]).To(Equal("Filesystem"))
		})
	})
})
