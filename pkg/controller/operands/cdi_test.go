package operands

import (
	"context"
	"fmt"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
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
			Expect(req.Conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "CDIConditions",
				Message: "CDI resource has no conditions",
			}))
			Expect(req.Conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "CDIConditions",
				Message: "CDI resource has no conditions",
			}))
			Expect(req.Conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
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
			Expect(*foundResource.Spec.UninstallStrategy).To(Equal(*expectedResource.Spec.UninstallStrategy))
		})

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

		It("should only set featureGate on Spec.Config if directly set on CDI CR", func() {
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
			Expect(*foundResource.Spec.Config.UploadProxyURLOverride).To(Equal(*expectedResource.Spec.Config.UploadProxyURLOverride))
			Expect(*foundResource.Spec.Config.ScratchSpaceStorageClass).To(Equal(*expectedResource.Spec.Config.ScratchSpaceStorageClass))
			Expect(*foundResource.Spec.Config.PodResourceRequirements).To(Equal(*expectedResource.Spec.Config.PodResourceRequirements))
			Expect(*foundResource.Spec.Config.FilesystemOverhead).To(Equal(*expectedResource.Spec.Config.FilesystemOverhead))
			Expect(foundResource.Spec.Config.FeatureGates).To(ContainElement("SomeFeatureGate"))
			// additionally contains HonorWaitForFirstConsumer
			Expect(foundResource.Spec.Config.FeatureGates).To(ContainElement("HonorWaitForFirstConsumer"))

		})

		It("should add HonorWaitForFirstConsumer featuregate if Spec.Config if empty", func() {
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
		})

		It("should handle conditions", func() {
			expectedResource, err := NewCDI(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			expectedResource.Status.Conditions = []conditionsv1.Condition{
				conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "Foo",
					Message: "Bar",
				},
				conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "Foo",
					Message: "Bar",
				},
				conditionsv1.Condition{
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
			Expect(req.Conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "CDINotAvailable",
				Message: "CDI is not available: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "CDIProgressing",
				Message: "CDI is progressing: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  "CDIProgressing",
				Message: "CDI is progressing: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionDegraded]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionDegraded,
				Status:  corev1.ConditionTrue,
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
		})

		It("volumeMode should be filesystem when platform is baremetal", func() {
			hco.Spec.BareMetalPlatform = true

			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			Expect(expectedResource.Data["volumeMode"]).To(Equal("Filesystem"))
		})

		It("volumeMode should be filesystem when platform is not baremetal", func() {
			hco.Spec.BareMetalPlatform = false

			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			Expect(expectedResource.Data["volumeMode"]).To(Equal("Filesystem"))
		})

		It("local storage class name should be available when specified", func() {
			hco.Spec.LocalStorageClassName = "local"

			expectedResource := NewKubeVirtStorageConfigForCR(hco, commonTestUtils.Namespace)
			Expect(expectedResource.Data["local.accessMode"]).To(Equal("ReadWriteOnce"))
			Expect(expectedResource.Data["local.volumeMode"]).To(Equal("Filesystem"))
		})
	})
})
