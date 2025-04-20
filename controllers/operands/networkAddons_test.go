package operands

import (
	"context"
	"maps"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	networkaddonsshared "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("CNA Operand", func() {

	Context("NetworkAddonsConfig", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).To(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			Expect(foundResource.Spec.Multus).To(Equal(&networkaddonsshared.Multus{}))
			Expect(foundResource.Spec.LinuxBridge).To(Equal(&networkaddonsshared.LinuxBridge{}))
			Expect(foundResource.Spec.KubeMacPool).To(Equal(&networkaddonsshared.KubeMacPool{}))
			Expect(foundResource.Spec.KubevirtIpamController).To(Equal(&networkaddonsshared.KubevirtIpamController{}))
		})

		It("should find if present", func() {
			expectedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigConditions",
				Message: "NetworkAddonsConfig resource has no conditions",
			}))
		})

		It("should find reconcile to default", func() {
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			existingResource.Spec.ImagePullPolicy = corev1.PullAlways // set non-default value

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())
			Expect(foundResource.Spec.ImagePullPolicy).To(BeEmpty())

			Expect(req.Conditions).To(BeEmpty())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(handler.Scheme, existingResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(handler.Scheme, foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

		})

		It("should reconcile managed labels to default without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			for k, v := range expectedLabels {
				outdatedResource.Labels[k] = "wrong_" + v
			}
			outdatedResource.Labels[userLabelKey] = userLabelValue

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should reconcile managed labels to default on label deletion without touching user added ones", func() {
			const userLabelKey = "userLabelKey"
			const userLabelValue = "userLabelValue"
			outdatedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedLabels := maps.Clone(outdatedResource.Labels)
			outdatedResource.Labels[userLabelKey] = userLabelValue
			delete(outdatedResource.Labels, hcoutil.AppLabelVersion)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: outdatedResource.Name, Namespace: outdatedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			for k, v := range expectedLabels {
				Expect(foundResource.Labels).To(HaveKeyWithValue(k, v))
			}
			Expect(foundResource.Labels).To(HaveKeyWithValue(userLabelKey, userLabelValue))
		})

		It("should add node placement if missing in CNAO", func() {
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.PlacementConfiguration).To(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration).ToNot(BeNil())
			placementConfig := foundResource.Spec.PlacementConfiguration
			Expect(placementConfig.Infra).ToNot(BeNil())
			Expect(placementConfig.Infra.NodeSelector["key1"]).To(Equal("value1"))
			Expect(placementConfig.Infra.NodeSelector["key2"]).To(Equal("value2"))

			Expect(placementConfig.Workloads).ToNot(BeNil())
			Expect(placementConfig.Workloads.Tolerations).To(Equal(hco.Spec.Workloads.NodePlacement.Tolerations))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should remove node placement if missing in HCO CR", func() {

			hcoNodePlacement := commontestutils.NewHco()
			hcoNodePlacement.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			hcoNodePlacement.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			existingResource, err := NewNetworkAddons(hcoNodePlacement)
			Expect(err).ToNot(HaveOccurred())

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration).To(BeNil())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify node placement according to HCO CR", func() {

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			// now, modify HCO's node placement
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
			})

			hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(existingResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(2))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key1"]).To(Equal("value1"))

			Expect(foundResource.Spec.PlacementConfiguration).ToNot(BeNil())
			Expect(foundResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(3))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key1"]).To(Equal("something else"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should overwrite node placement if directly set on CNAO CR", func() {
			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			// mock a reconciliation triggered by a change in CNAO CR
			req.HCOTriggered = false

			// now, modify CNAO node placement
			existingResource.Spec.PlacementConfiguration.Infra.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
			})
			existingResource.Spec.PlacementConfiguration.Workloads.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
			})

			existingResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"] = "BADvalue1"
			existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"] = "BADvalue2"

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(3))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.Tolerations).To(HaveLen(3))
			Expect(existingResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"]).To(Equal("BADvalue1"))
			Expect(existingResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"]).To(Equal("BADvalue2"))

			Expect(foundResource.Spec.PlacementConfiguration.Infra.Tolerations).To(HaveLen(2))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.Tolerations).To(HaveLen(2))
			Expect(foundResource.Spec.PlacementConfiguration.Infra.NodeSelector["key1"]).To(Equal("value1"))
			Expect(foundResource.Spec.PlacementConfiguration.Workloads.NodeSelector["key2"]).To(Equal("value2"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should add self signed configuration if missing in CNAO", func() {
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
				CA: hcov1beta1.CertRotateConfigCA{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
				},
				Server: hcov1beta1.CertRotateConfigServer{
					Duration:    &metav1.Duration{Duration: 12 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
				},
			}

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(foundResource.Spec.SelfSignConfiguration).ToNot(BeNil())
			selfSignedConfig := foundResource.Spec.SelfSignConfiguration
			Expect(selfSignedConfig.CARotateInterval).To(Equal("24h0m0s"))
			Expect(selfSignedConfig.CAOverlapInterval).To(Equal("1h0m0s"))
			Expect(selfSignedConfig.CertRotateInterval).To(Equal("12h0m0s"))
			Expect(selfSignedConfig.CertOverlapInterval).To(Equal("30m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should set self signed configuration to defaults if missing in HCO CR", func() {
			existingResource := NewNetworkAddonsWithNameOnly(hco)

			cl := commontestutils.InitClient([]client.Object{hco})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.SelfSignConfiguration).To(BeNil())

			Expect(foundResource.Spec.SelfSignConfiguration.CARotateInterval).ToNot(BeNil())
			selfSignedConfig := foundResource.Spec.SelfSignConfiguration
			Expect(selfSignedConfig.CARotateInterval).To(Equal("48h0m0s"))
			Expect(selfSignedConfig.CAOverlapInterval).To(Equal("24h0m0s"))
			Expect(selfSignedConfig.CertRotateInterval).To(Equal("24h0m0s"))
			Expect(selfSignedConfig.CertOverlapInterval).To(Equal("12h0m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify self signed configuration according to HCO CR", func() {

			hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
				CA: hcov1beta1.CertRotateConfigCA{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
				},
				Server: hcov1beta1.CertRotateConfigServer{
					Duration:    &metav1.Duration{Duration: 12 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
				},
			}
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			By("Modify HCO's cert configuration")
			hco.Spec.CertConfig.CA.Duration.Duration *= 2
			hco.Spec.CertConfig.CA.RenewBefore.Duration *= 2
			hco.Spec.CertConfig.Server.Duration.Duration *= 2
			hco.Spec.CertConfig.Server.RenewBefore.Duration *= 2

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.SelfSignConfiguration).ToNot(BeNil())
			existingSelfSignedConfig := existingResource.Spec.SelfSignConfiguration
			Expect(existingSelfSignedConfig.CARotateInterval).To(Equal("24h0m0s"))
			Expect(existingSelfSignedConfig.CAOverlapInterval).To(Equal("1h0m0s"))
			Expect(existingSelfSignedConfig.CertRotateInterval).To(Equal("12h0m0s"))
			Expect(existingSelfSignedConfig.CertOverlapInterval).To(Equal("30m0s"))

			Expect(foundResource.Spec.SelfSignConfiguration).ToNot(BeNil())
			foundSelfSignedConfig := foundResource.Spec.SelfSignConfiguration
			Expect(foundSelfSignedConfig.CARotateInterval).To(Equal("48h0m0s"))
			Expect(foundSelfSignedConfig.CAOverlapInterval).To(Equal("2h0m0s"))
			Expect(foundSelfSignedConfig.CertRotateInterval).To(Equal("24h0m0s"))
			Expect(foundSelfSignedConfig.CertOverlapInterval).To(Equal("1h0m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should overwrite self signed configuration if directly set on CNAO CR", func() {

			hco.Spec.CertConfig = hcov1beta1.HyperConvergedCertConfig{
				CA: hcov1beta1.CertRotateConfigCA{
					Duration:    &metav1.Duration{Duration: 24 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 1 * time.Hour},
				},
				Server: hcov1beta1.CertRotateConfigServer{
					Duration:    &metav1.Duration{Duration: 12 * time.Hour},
					RenewBefore: &metav1.Duration{Duration: 30 * time.Minute},
				},
			}
			existingResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())

			By("Mock a reconciliation triggered by a change in CNAO CR")
			req.HCOTriggered = false

			By("Modify CNAO's cert configuration")
			existingResource.Spec.SelfSignConfiguration.CARotateInterval = "48h0m0s"
			existingResource.Spec.SelfSignConfiguration.CAOverlapInterval = "2h0m0s"
			existingResource.Spec.SelfSignConfiguration.CertRotateInterval = "24h0m0s"
			existingResource.Spec.SelfSignConfiguration.CertOverlapInterval = "1h0m0s"

			cl := commontestutils.InitClient([]client.Object{hco, existingResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Overwritten).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(existingResource.Spec.SelfSignConfiguration).ToNot(BeNil())
			existingSelfSignedConfig := existingResource.Spec.SelfSignConfiguration
			Expect(existingSelfSignedConfig.CARotateInterval).To(Equal("48h0m0s"))
			Expect(existingSelfSignedConfig.CAOverlapInterval).To(Equal("2h0m0s"))
			Expect(existingSelfSignedConfig.CertRotateInterval).To(Equal("24h0m0s"))
			Expect(existingSelfSignedConfig.CertOverlapInterval).To(Equal("1h0m0s"))

			Expect(foundResource.Spec.SelfSignConfiguration).ToNot(BeNil())
			foundSelfSignedConfig := foundResource.Spec.SelfSignConfiguration
			Expect(foundSelfSignedConfig.CARotateInterval).To(Equal("24h0m0s"))
			Expect(foundSelfSignedConfig.CAOverlapInterval).To(Equal("1h0m0s"))
			Expect(foundSelfSignedConfig.CertRotateInterval).To(Equal("12h0m0s"))
			Expect(foundSelfSignedConfig.CertOverlapInterval).To(Equal("30m0s"))

			Expect(req.Conditions).To(BeEmpty())
		})

		type ovsAnnotationParams struct {
			ovsExists         bool
			setAnnotation     bool
			annotationValue   string
			ovsDeployExpected bool
		}
		DescribeTable("when reconciling ovs-cni", func(o ovsAnnotationParams) {
			existingCNAO, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			if o.ovsExists {
				existingCNAO.Spec.Ovs = &networkaddonsshared.Ovs{}
			}

			if o.setAnnotation {
				hco.Annotations = map[string]string{
					"deployOVS": o.annotationValue,
				}
			}

			cl := commontestutils.InitClient([]client.Object{hco, existingCNAO})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundCNAO := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingCNAO.Name, Namespace: existingCNAO.Namespace},
					foundCNAO),
			).To(Succeed())

			if o.ovsDeployExpected {
				Expect(foundCNAO.Spec.Ovs).ToNot(BeNil(), "OVS spec should be added")
			} else {
				Expect(foundCNAO.Spec.Ovs).To(BeNil(), "OVS spec should not be added")
			}
		},
			Entry("should have OVS if deployOVS annotation is set to true", ovsAnnotationParams{
				ovsExists:         false,
				setAnnotation:     true,
				annotationValue:   "true",
				ovsDeployExpected: true,
			}),
			Entry("should not have ovs if deployOVS annotation is set to false", ovsAnnotationParams{
				ovsExists:         true,
				setAnnotation:     true,
				annotationValue:   "false",
				ovsDeployExpected: false,
			}),
			Entry("should not have ovs if deployOVS annotation is not set to true", ovsAnnotationParams{
				ovsExists:         true,
				setAnnotation:     true,
				annotationValue:   "someValue",
				ovsDeployExpected: false,
			}),
			Entry("should not have ovs if deployOVS annotation is empty", ovsAnnotationParams{
				ovsExists:         true,
				setAnnotation:     true,
				annotationValue:   "",
				ovsDeployExpected: false,
			}),
			Entry("should not have ovs if deployOVS annotation does not exist", ovsAnnotationParams{
				ovsExists:         false,
				setAnnotation:     false,
				annotationValue:   "",
				ovsDeployExpected: false,
			}),
		)

		type ksdAnnotationParams struct {
			ksdExists          bool
			setFeatureGate     bool
			featureGateValue   bool
			ksdDeployExpected  bool
			expectedBaseDomain string
		}

		ksdTester := func(o ksdAnnotationParams) {
			existingCNAO, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			if o.ksdExists {
				existingCNAO.Spec.KubeSecondaryDNS = &networkaddonsshared.KubeSecondaryDNS{}
			}

			const kubeSecondaryDNSNameServerIP = "127.0.0.1"
			if o.setFeatureGate {
				hco.Spec.FeatureGates.DeployKubeSecondaryDNS = ptr.To(o.featureGateValue)
				hco.Spec.KubeSecondaryDNSNameServerIP = ptr.To(kubeSecondaryDNSNameServerIP)
			}

			cl := commontestutils.InitClient([]client.Object{hco, existingCNAO})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundCNAO := &networkaddonsv1.NetworkAddonsConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingCNAO.Name, Namespace: existingCNAO.Namespace},
					foundCNAO),
			).To(Succeed())

			if o.ksdDeployExpected {
				Expect(foundCNAO.Spec.KubeSecondaryDNS).ToNot(BeNil(), "KSD spec should be added")
				Expect(foundCNAO.Spec.KubeSecondaryDNS.Domain).To(Equal(o.expectedBaseDomain),
					"Expected domain should be set on KSD spec")
				Expect(foundCNAO.Spec.KubeSecondaryDNS.NameServerIP).To(Equal(kubeSecondaryDNSNameServerIP),
					"Expected NameServerIP should be set on KSD spec")
			} else {
				Expect(foundCNAO.Spec.KubeSecondaryDNS).To(BeNil(), "KSD spec should not be added")
			}
		}

		Context("With K8s", func() {
			DescribeTable("when reconciling kube-secondary-dns", ksdTester,
				Entry("should have KSD if feature gate is set to true", ksdAnnotationParams{
					ksdExists:          false,
					setFeatureGate:     true,
					featureGateValue:   true,
					ksdDeployExpected:  true,
					expectedBaseDomain: "",
				}),
				Entry("should not have KSD if feature gate is set to false", ksdAnnotationParams{
					ksdExists:          true,
					setFeatureGate:     true,
					featureGateValue:   false,
					ksdDeployExpected:  false,
					expectedBaseDomain: "",
				}),
				Entry("should not have KSD if feature gate does not exist", ksdAnnotationParams{
					ksdExists:          true,
					setFeatureGate:     false,
					featureGateValue:   false,
					ksdDeployExpected:  false,
					expectedBaseDomain: "",
				}),
			)
		})

		Context("With Openshift Mock", func() {
			BeforeEach(func() {
				getClusterInfo := hcoutil.GetClusterInfo

				hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
					return &commontestutils.ClusterInfoMock{}
				}

				DeferCleanup(func() {
					hcoutil.GetClusterInfo = getClusterInfo
				})
			})

			DescribeTable("when reconciling kube-secondary-dns", ksdTester,
				Entry("should have KSD if feature gate is set to true", ksdAnnotationParams{
					ksdExists:          false,
					setFeatureGate:     true,
					featureGateValue:   true,
					ksdDeployExpected:  true,
					expectedBaseDomain: commontestutils.BaseDomain,
				}),
				Entry("should not have KSD if feature gate is set to false", ksdAnnotationParams{
					ksdExists:          true,
					setFeatureGate:     true,
					featureGateValue:   false,
					ksdDeployExpected:  false,
					expectedBaseDomain: "",
				}),
				Entry("should not have KSD if feature gate does not exist", ksdAnnotationParams{
					ksdExists:          true,
					setFeatureGate:     false,
					featureGateValue:   false,
					ksdDeployExpected:  false,
					expectedBaseDomain: "",
				}),
			)
		})

		It("should handle conditions", func() {
			expectedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
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
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[hcov1beta1.ConditionAvailable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionAvailable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigNotAvailable",
				Message: "NetworkAddonsConfig is not available: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionProgressing]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "NetworkAddonsConfigProgressing",
				Message: "NetworkAddonsConfig is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigProgressing",
				Message: "NetworkAddonsConfig is progressing: Bar",
			}))
			Expect(req.Conditions[hcov1beta1.ConditionDegraded]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionDegraded,
				Status:  metav1.ConditionTrue,
				Reason:  "NetworkAddonsConfigDegraded",
				Message: "NetworkAddonsConfig is degraded: Bar",
			}))
		})

		It("should handle upgrade condition", func() {
			expectedResource, err := NewNetworkAddons(hco)
			Expect(err).ToNot(HaveOccurred())
			expectedResource.Status.Conditions = []conditionsv1.Condition{
				{
					Type:   conditionsv1.ConditionAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "Foo",
					Message: "Bar",
				},
			}
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions).To(HaveLen(1))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigNotUpgradeable",
				Message: "NetworkAddonsConfig is not upgradeable: Bar",
			}))
		})

		It("should override an existing upgrade condition, if the operand one is false", func() {
			expectedResource, err := NewNetworkAddons(hco)
			req.Conditions.SetStatusCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "another reason",
				Message: "another message",
			})

			Expect(err).ToNot(HaveOccurred())
			expectedResource.Status.Conditions = []conditionsv1.Condition{
				{
					Type:   conditionsv1.ConditionAvailable,
					Status: corev1.ConditionTrue,
				},
				{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "Foo",
					Message: "Bar",
				},
			}
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(handler.Scheme, expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions).To(HaveLen(1))
			Expect(req.Conditions[hcov1beta1.ConditionUpgradeable]).To(commontestutils.RepresentCondition(metav1.Condition{
				Type:    hcov1beta1.ConditionUpgradeable,
				Status:  metav1.ConditionFalse,
				Reason:  "NetworkAddonsConfigNotUpgradeable",
				Message: "NetworkAddonsConfig is not upgradeable: Bar",
			}))
		})

		Context("jsonpath Annotation", func() {
			It("Should create CNA object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					},
					{
						"op": "add",
						"path": "/spec/imagePullPolicy",
						"value": "Always"
					}
				]`}

				cna, err := NewNetworkAddons(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cna).ToNot(BeNil())
				Expect(cna.Spec.KubeMacPool.RangeStart).To(Equal("1.1.1.1.1.1"))
				Expect(cna.Spec.KubeMacPool.RangeEnd).To(Equal("5.5.5.5.5.5"))
				Expect(cna.Spec.ImagePullPolicy).To(BeEquivalentTo("Always"))
			})

			It("Should fail to create CNA object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					}
				]`}

				_, err := NewNetworkAddons(hco)
				Expect(err).To(HaveOccurred())
			})

			It("Ensure func should create CNA object with changes from the annotation", func() {
				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					},
					{
						"op": "add",
						"path": "/spec/imagePullPolicy",
						"value": "Always"
					}
				]`}

				expectedResource := NewNetworkAddonsWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{})
				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				cna := &networkaddonsv1.NetworkAddonsConfig{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cna),
				).To(Succeed())

				Expect(cna).ToNot(BeNil())
				Expect(cna.Spec.KubeMacPool.RangeStart).To(Equal("1.1.1.1.1.1"))
				Expect(cna.Spec.KubeMacPool.RangeEnd).To(Equal("5.5.5.5.5.5"))
				Expect(cna.Spec.ImagePullPolicy).To(BeEquivalentTo("Always"))
			})

			It("Ensure func should fail to create CNA object with wrong jsonPatch", func() {
				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					}
				]`}

				expectedResource := NewNetworkAddonsWithNameOnly(hco)
				cl := commontestutils.InitClient([]client.Object{})
				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				cna := &networkaddonsv1.NetworkAddonsConfig{}

				Expect(cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					cna,
				)).To(MatchError(errors.IsNotFound, "not found error"))
			})

			It("Ensure func should update CNA object with changes from the annotation", func() {
				existsCna, err := NewNetworkAddons(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "add",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					},
					{
						"op": "add",
						"path": "/spec/imagePullPolicy",
						"value": "Always"
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsCna})

				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).ToNot(HaveOccurred())
				Expect(res.Updated).To(BeTrue())
				Expect(res.UpgradeDone).To(BeFalse())

				cna := &networkaddonsv1.NetworkAddonsConfig{}

				expectedResource := NewNetworkAddonsWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cna),
				).To(Succeed())

				Expect(cna.Spec.KubeMacPool.RangeStart).To(Equal("1.1.1.1.1.1"))
				Expect(cna.Spec.KubeMacPool.RangeEnd).To(Equal("5.5.5.5.5.5"))
				Expect(cna.Spec.ImagePullPolicy).To(BeEquivalentTo("Always"))
			})

			It("Ensure func should fail to update CNA object with wrong jsonPatch", func() {
				existsCna, err := NewNetworkAddons(hco)
				Expect(err).ToNot(HaveOccurred())

				hco.Annotations = map[string]string{common.JSONPatchCNAOAnnotationName: `[
					{
						"op": "notExists",
						"path": "/spec/kubeMacPool",
						"value": {"rangeStart": "1.1.1.1.1.1", "rangeEnd": "5.5.5.5.5.5" }
					}
				]`}

				cl := commontestutils.InitClient([]client.Object{hco, existsCna})

				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.Err).To(HaveOccurred())

				cna := &networkaddonsv1.NetworkAddonsConfig{}

				expectedResource := NewNetworkAddonsWithNameOnly(hco)
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						cna),
				).To(Succeed())

				Expect(cna.Spec.KubeMacPool.RangeStart).To(BeEmpty())
				Expect(cna.Spec.KubeMacPool.RangeEnd).To(BeEmpty())
				Expect(cna.Spec.ImagePullPolicy).To(BeEmpty())
			})
		})

		Context("Cache", func() {
			cl := commontestutils.InitClient([]client.Object{})
			handler := newCnaHandler(cl, commontestutils.GetScheme())

			It("should start with empty cache", func() {
				Expect(handler.hooks.(*cnaHooks).cache).To(BeNil())
			})

			It("should update the cache when reading full CR", func() {
				cr, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(cr).ToNot(BeNil())
				Expect(handler.hooks.(*cnaHooks).cache).ToNot(BeNil())

				By("compare pointers to make sure cache is working", func() {
					Expect(handler.hooks.(*cnaHooks).cache).To(BeIdenticalTo(cr))

					crII, err := handler.hooks.getFullCr(hco)
					Expect(err).ToNot(HaveOccurred())
					Expect(crII).ToNot(BeNil())
					Expect(cr).To(BeIdenticalTo(crII))
				})
			})

			It("should remove the cache on reset", func() {
				handler.hooks.(*cnaHooks).reset()
				Expect(handler.hooks.(*cnaHooks).cache).To(BeNil())
			})

			It("check that reset actually cause creating of a new cached instance", func() {
				crI, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crI).ToNot(BeNil())
				Expect(handler.hooks.(*cnaHooks).cache).ToNot(BeNil())

				handler.hooks.(*cnaHooks).reset()
				Expect(handler.hooks.(*cnaHooks).cache).To(BeNil())

				crII, err := handler.hooks.getFullCr(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(crII).ToNot(BeNil())
				Expect(handler.hooks.(*cnaHooks).cache).ToNot(BeNil())

				Expect(crI).ToNot(BeIdenticalTo(crII))
				Expect(handler.hooks.(*cnaHooks).cache).ToNot(BeIdenticalTo(crI))
				Expect(handler.hooks.(*cnaHooks).cache).To(BeIdenticalTo(crII))
			})

			Context("Requested components", func() {
				It("should not request nmstate", func() {
					expectedResource, err := NewNetworkAddons(hco)
					Expect(err).ToNot(HaveOccurred())

					Expect(expectedResource.Spec.NMState).To(BeNil())
				})
			})
		})

		Context("TLSSecurityProfile", func() {

			intermediateTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:         openshiftconfigv1.TLSProfileIntermediateType,
				Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
			}
			modernTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
				Type:   openshiftconfigv1.TLSProfileModernType,
				Modern: &openshiftconfigv1.ModernTLSProfile{},
			}

			It("should modify TLSSecurityProfile on CNAO CR according to ApiServer or HCO CR", func() {
				existingResource, err := NewNetworkAddons(hco)
				Expect(err).ToNot(HaveOccurred())
				Expect(existingResource.Spec.TLSSecurityProfile).To(Equal(intermediateTLSSecurityProfile))

				// now, modify HCO's TLSSecurityProfile
				hco.Spec.TLSSecurityProfile = modernTLSSecurityProfile

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &networkaddonsv1.NetworkAddonsConfig{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(foundResource.Spec.TLSSecurityProfile).To(Equal(modernTLSSecurityProfile))

				Expect(req.Conditions).To(BeEmpty())
			})

			It("should overwrite TLSSecurityProfile if directly set on CNAO CR", func() {
				hco.Spec.TLSSecurityProfile = intermediateTLSSecurityProfile
				existingResource, err := NewNetworkAddons(hco)
				Expect(err).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in CNAO CR
				req.HCOTriggered = false

				// now, modify CNAO node placement
				existingResource.Spec.TLSSecurityProfile = modernTLSSecurityProfile

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handler := (*genericOperand)(newCnaHandler(cl, commontestutils.GetScheme()))
				res := handler.ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &networkaddonsv1.NetworkAddonsConfig{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(foundResource.Spec.TLSSecurityProfile).To(Equal(hco.Spec.TLSSecurityProfile))
				Expect(foundResource.Spec.TLSSecurityProfile).ToNot(Equal(existingResource.Spec.TLSSecurityProfile))

				Expect(req.Conditions).To(BeEmpty())
			})
		})

	})

	Context("hcoConfig2CnaoPlacement", func() {
		tolr1 := corev1.Toleration{
			Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: ptr.To[int64](1),
		}
		tolr2 := corev1.Toleration{
			Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: ptr.To[int64](2),
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

			Expect(cnaoPlacement.NodeSelector["key1"]).To(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).To(Equal("value2"))
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

			Expect(cnaoPlacement.Tolerations[0]).To(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).To(Equal(tolr2))
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

			Expect(cnaoPlacement.Affinity.NodeAffinity).To(Equal(affinity.NodeAffinity))
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

			Expect(cnaoPlacement.NodeSelector["key1"]).To(Equal("value1"))
			Expect(cnaoPlacement.NodeSelector["key2"]).To(Equal("value2"))

			Expect(cnaoPlacement.Tolerations[0]).To(Equal(tolr1))
			Expect(cnaoPlacement.Tolerations[1]).To(Equal(tolr2))

			Expect(cnaoPlacement.Affinity.NodeAffinity).To(Equal(hcoConf.Affinity.NodeAffinity))
		})
	})
})
