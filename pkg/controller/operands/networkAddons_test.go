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
	. "github.com/onsi/gomega"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
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
			expectedResource := hco.NewNetworkAddons()
			cl := commonTestUtils.InitClient([]runtime.Object{})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
			expectedResource := hco.NewNetworkAddons()
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
			existingResource := hco.NewNetworkAddons()
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)
			existingResource.Spec.ImagePullPolicy = corev1.PullAlways // set non-default value

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
			existingResource := hco.NewNetworkAddons()

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{commonTestUtils.NewHyperConvergedConfig()}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
			hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			existingResource := hcoNodePlacement.NewNetworkAddons()

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			existingResource := hco.NewNetworkAddons()

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			existingResource := hco.NewNetworkAddons()

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
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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

		It("should handle conditions", func() {
			expectedResource := hco.NewNetworkAddons()
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
			handler := &cnaHandler{Client: cl, Scheme: commonTestUtils.GetScheme()}
			res := handler.Ensure(req)
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
})
