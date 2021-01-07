package operands

import (
	"context"
	"fmt"

	networkaddonsshared "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"
)

var _ = Describe("CNA Operand", func() {

	Context("NetworkAddonsConfig", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := NewNetworkAddons(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commonTestUtils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			Expect(foundResource.Spec.Multus).To(Equal(&networkaddonsshared.Multus{}))
			Expect(foundResource.Spec.LinuxBridge).To(Equal(&networkaddonsshared.LinuxBridge{}))
			Expect(foundResource.Spec.KubeMacPool).To(Equal(&networkaddonsshared.KubeMacPool{}))
		})

		It("should find if present", func() {
			expectedResource := NewNetworkAddons(hco)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
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
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
			Expect(req.Conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
			Expect(req.Conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
		})

		It("should find reconcile to default", func() {
			existingResource := NewNetworkAddons(hco)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)
			existingResource.Spec.ImagePullPolicy = corev1.PullAlways // set non-default value

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.ImagePullPolicy).To(BeEmpty())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should add node placement if missing in CNAO", func() {
			existingResource := NewNetworkAddons(hco)

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{commonTestUtils.NewNodePlacement()}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.PlacementConfiguration).To(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration).ToNot(BeNil())
			placementConfig := foundResource.Spec.PlacementConfiguration
			Expect(placementConfig.Infra).ToNot(BeNil())
			Expect(placementConfig.Infra.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(placementConfig.Infra.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(placementConfig.Workloads).ToNot(BeNil())
			Expect(placementConfig.Workloads.Tolerations).Should(Equal(hco.Spec.Workloads.NodePlacement.Tolerations))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should remove node placement if missing in HCO CR", func() {

			hcoNodePlacement := commonTestUtils.NewHco()
			hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewNetworkAddons(hcoNodePlacement)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration).To(BeNil())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify node placement according to HCO CR", func() {

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewNetworkAddons(hco)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(existingResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(2))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key1"]).Should(Equal("value1"))

			Expect(foundResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(3))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key1"]).Should(Equal("something else"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should overwrite node placement if directly set on CNAO CR", func() {
			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
			existingResource := NewNetworkAddons(hco)

			// mock a reconciliation triggered by a change in CNAO CR
			req.HCOTriggered = false

			// now, modify CNAO node placement
			seconds3 := int64(3)
			existingResource.Spec.PlacementConfiguration.Infra.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})
			existingResource.Spec.PlacementConfiguration.Workloads.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			existingResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"] = "BADvalue1"
			existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"] = "BADvalue2"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(3))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.Tolerations).To(HaveLen(3))
			Expect(existingResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"]).Should(Equal("BADvalue1"))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"]).Should(Equal("BADvalue2"))

			Expect(foundResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(2))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.Tolerations).To(HaveLen(2))
			Expect(foundResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(req.Conditions).To(BeEmpty())
		})

		type ovsAnnotationParams struct {
			annotationExists  bool
			annotationValue   string
			ovsDeployExpected bool
		}
		table.DescribeTable("when reconciling ovs-cni", func(o ovsAnnotationParams) {
			hcoOVSConfig := commonTestUtils.NewHco()
			hcoOVSConfig.Annotations = map[string]string{}

			if o.annotationExists {
				hcoOVSConfig.Annotations["deployOVS"] = o.annotationValue
			}

			existingResource := NewNetworkAddons(hcoOVSConfig)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			if o.ovsDeployExpected {
				Expect(existingResource.Spec.Ovs).ToNot(BeNil(), "Ovs spec should be added")
			} else {
				Expect(existingResource.Spec.Ovs).To(BeNil(), "Ovs spec should not be added")
			}
		},
			table.Entry("should have ovs if deployOVS annotation is set to true", ovsAnnotationParams{
				annotationExists:  true,
				annotationValue:   "true",
				ovsDeployExpected: true,
			}),
			table.Entry("should not have ovs if deployOVS annotation is not set to true", ovsAnnotationParams{
				annotationExists:  true,
				annotationValue:   "false",
				ovsDeployExpected: false,
			}),
			table.Entry("should not have ovs if deployOVS annotation does not exist", ovsAnnotationParams{
				annotationExists:  false,
				annotationValue:   "",
				ovsDeployExpected: false,
			}),
		)

		It("should handle conditions", func() {
			expectedResource := NewNetworkAddons(hco)
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
			handler := (*genericOperand)(newCnaHandler(cl, commonTestUtils.GetScheme()))
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
				Reason:  "NetworkAddonsConfigNotAvailable",
				Message: "NetworkAddonsConfig is not available: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "NetworkAddonsConfigProgressing",
				Message: "NetworkAddonsConfig is progressing: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  "NetworkAddonsConfigProgressing",
				Message: "NetworkAddonsConfig is progressing: Bar",
			}))
			Expect(req.Conditions[conditionsv1.ConditionDegraded]).To(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionDegraded,
				Status:  corev1.ConditionTrue,
				Reason:  "NetworkAddonsConfigDegraded",
				Message: "NetworkAddonsConfig is degraded: Bar",
			}))
		})
	})

	Context("hcoConfig2CnaoPlacement", func() {
		seconds1, seconds2 := int64(1), int64(2)
		tolr1 := corev1.Toleration{
			Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: &seconds1,
		}
		tolr2 := corev1.Toleration{
			Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: &seconds2,
		}

		It("Should return nil if HCO's input is empty", func() {
			Expect(hcoConfig2CnaoPlacement(&sdkapi.NodePlacement{})).To(BeNil())
		})

		It("Should return only NodeSelector", func() {
			hcoConf := &sdkapi.NodePlacement{
				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).ToNot(BeNil())
			Expect(cnaoPlacement.Tolerations).To(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).Should(Equal("value2"))
		})

		It("Should return only Tolerations", func() {
			hcoConf := &sdkapi.NodePlacement{
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).To(BeNil())
			Expect(cnaoPlacement.Tolerations).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.Tolerations[0]).Should(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).Should(Equal(tolr2))
		})

		It("Should return only Affinity", func() {
			affinity := &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
								MatchFields: []corev1.NodeSelectorRequirement{
									{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
									{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
								},
							},
						},
					},
				},
			}
			hcoConf := &sdkapi.NodePlacement{
				Affinity: affinity,
			}
			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).To(BeNil())
			Expect(cnaoPlacement.Tolerations).To(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.PodAffinity).To(BeNil())
			Expect(cnaoPlacement.Affinity.PodAntiAffinity).To(BeNil())

			Expect(cnaoPlacement.Affinity.NodeAffinity).Should(Equal(affinity.NodeAffinity))
		})

		It("Should return the whole object", func() {
			hcoConf := &sdkapi.NodePlacement{

				NodeSelector: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
									MatchFields: []corev1.NodeSelectorRequirement{
										{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
										{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
									},
								},
							},
						},
					},
				},
				Tolerations: []corev1.Toleration{tolr1, tolr2},
			}

			cnaoPlacement := hcoConfig2CnaoPlacement(hcoConf)
			Expect(cnaoPlacement).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector).ToNot(BeNil())
			Expect(cnaoPlacement.Tolerations).ToNot(BeNil())
			Expect(cnaoPlacement.Affinity.NodeAffinity).ToNot(BeNil())

			Expect(cnaoPlacement.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(cnaoPlacement.Tolerations[0]).Should(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).Should(Equal(tolr2))

			Expect(cnaoPlacement.Affinity.NodeAffinity).Should(Equal(hcoConf.Affinity.NodeAffinity))
		})
	})
})
