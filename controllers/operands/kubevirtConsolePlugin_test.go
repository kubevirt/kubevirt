package operands

import (
	"context"
	"reflect"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

var _ = Describe("Kubevirt Console Plugin", func() {
	Context("Console Plugin CR", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		var expectedConsoleConfig = &operatorv1.Console{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Console",
				APIVersion: "operator.openshift.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
		}

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		It("should create plugin CR if not present", func() {
			expectedResource := NewKVConsolePlugin(hco)
			cl := commontestutils.InitClient([]client.Object{})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find plugin CR if present", func() {
			expectedResource := NewKVConsolePlugin(hco)

			cl := commontestutils.InitClient([]client.Object{hco, expectedResource, expectedConsoleConfig})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commontestutils.GetScheme(), expectedResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile plugin spec to default if changed", func() {
			expectedResource := NewKVConsolePlugin(hco)
			outdatedResource := NewKVConsolePlugin(hco)

			outdatedResource.Spec.Backend.Service.Port = int32(6666)
			outdatedResource.Spec.DisplayName = "fake plugin name"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource, expectedConsoleConfig})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(expectedResource.Spec, foundResource.Spec)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commontestutils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		})

		It("should reconcile plugin labels to default if changed", func() {
			expectedResource := NewKVConsolePlugin(hco)
			outdatedResource := NewKVConsolePlugin(hco)

			outdatedResource.Labels[hcoutil.AppLabel] = "changed"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource, expectedConsoleConfig})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(foundResource.Labels, expectedResource.Labels)).To(BeTrue())
		})

		It("should reconcile plugin labels to default if added", func() {
			expectedResource := NewKVConsolePlugin(hco)
			outdatedResource := NewKVConsolePlugin(hco)

			outdatedResource.Labels["fake_label"] = "something"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource, expectedConsoleConfig})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(foundResource.Labels, expectedResource.Labels)).To(BeTrue())
		})

		It("should reconcile plugin labels to default if deleted", func() {
			expectedResource := NewKVConsolePlugin(hco)
			outdatedResource := NewKVConsolePlugin(hco)

			delete(outdatedResource.Labels, hcoutil.AppLabel)

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource, expectedConsoleConfig})
			handler, _ := newKvUIPluginCRHandler(logger, cl, commontestutils.GetScheme(), hco)

			res := handler[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &consolev1.ConsolePlugin{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).ToNot(HaveOccurred())

			Expect(reflect.DeepEqual(foundResource.Labels, expectedResource.Labels)).To(BeTrue())
		})
	})

	Context("Kubevirt Console Plugin and UI Proxy Deployments", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		DescribeTable("should create if not present", func(appComponent hcoutil.AppComponent,
			deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
			expectedResource := deploymentManifestor(hco)

			cl := commontestutils.InitClient([]client.Object{})
			handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handlers[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Spec.Template.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Spec.Template.Labels).Should(HaveKeyWithValue(hcoutil.AppLabelComponent, string(appComponent)))
			Expect(foundResource.Spec.Template.Labels).Should(HaveKeyWithValue(hcoutil.AppLabelManagedBy, hcoutil.OperatorName))
			Expect(foundResource.Spec.Template.Labels).Should(HaveKeyWithValue(hcoutil.AppLabelVersion, hcoutil.GetHcoKvIoVersion()))
			Expect(foundResource.Spec.Template.Labels).Should(HaveKeyWithValue(hcoutil.AppLabelPartOf, hcoutil.HyperConvergedCluster))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			Expect(reflect.DeepEqual(expectedResource.Spec, foundResource.Spec)).To(BeTrue())
		},
			Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
			Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
		)

		DescribeTable("should find deployment if present", func(appComponent hcoutil.AppComponent,
			deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
			expectedResource := deploymentManifestor(hco)

			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)
			Expect(err).ToNot(HaveOccurred())

			res := handlers[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		},
			Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
			Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
		)

		DescribeTable("should reconcile deployment to default if changed - (updatable fields)", func(appComponent hcoutil.AppComponent,
			deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
			expectedResource := deploymentManifestor(hco)
			outdatedResource := deploymentManifestor(hco)

			outdatedResource.ObjectMeta.UID = "oldObjectUID"
			outdatedResource.ObjectMeta.ResourceVersion = "1234"

			outdatedResource.Spec.Replicas = ptr.To(int32(123))
			outdatedResource.Spec.Template.Spec.Containers[0].Image = "quay.io/fake/image:latest"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

			Expect(err).ToNot(HaveOccurred())
			res := handlers[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(foundResource.Spec.Replicas).ToNot(Equal(outdatedResource.Spec.Replicas))
			Expect(foundResource.Spec.Replicas).To(Equal(expectedResource.Spec.Replicas))
			Expect(foundResource.Spec.Template.Spec.Containers[0].Image).To(Equal(expectedResource.Spec.Template.Spec.Containers[0].Image))
			Expect(reflect.DeepEqual(expectedResource.Spec, foundResource.Spec)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commontestutils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

			// let's check the object UID to ensure that the object get updated and not deleted and recreated
			Expect(foundResource.GetUID()).To(Equal(types.UID("oldObjectUID")))
		},
			Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
			Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
		)

		DescribeTable("should reconcile deployment to default if changed - (immutable fields)", func(appComponent hcoutil.AppComponent,
			deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
			expectedResource := deploymentManifestor(hco)
			outdatedResource := deploymentManifestor(hco)

			outdatedResource.ObjectMeta.UID = "oldObjectUID"
			outdatedResource.ObjectMeta.ResourceVersion = "1234"

			outdatedResource.ObjectMeta.Labels[hcoutil.AppLabel] = "wrong label"
			outdatedResource.Spec.Selector.MatchLabels[hcoutil.AppLabel] = "wrong label"
			outdatedResource.Spec.Template.Labels[hcoutil.AppLabel] = "wrong label"

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

			Expect(err).ToNot(HaveOccurred())
			res := handlers[0].ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &appsv1.Deployment{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(foundResource.ObjectMeta.Labels).ToNot(Equal(outdatedResource.ObjectMeta.Labels))
			Expect(foundResource.ObjectMeta.Labels).To(Equal(expectedResource.ObjectMeta.Labels))
			Expect(reflect.DeepEqual(expectedResource.Spec, foundResource.Spec)).To(BeTrue())

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commontestutils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))

			// let's check the object UID to ensure that the object get really deleted and recreated
			Expect(foundResource.GetUID()).ToNot(Equal(types.UID("oldObjectUID")))
		},
			Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
			Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
		)

		Context("Node Placement", func() {
			DescribeTable("should add node placement if missing", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
				existingResource := deploymentManifestor(hco)

				hco.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.NodeSelector).To(BeEmpty())
				Expect(existingResource.Spec.Template.Spec.Affinity).To(BeNil())
				Expect(existingResource.Spec.Template.Spec.Tolerations).To(BeEmpty())

				Expect(foundResource.Spec.Template.Spec.NodeSelector).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.NodeSelector))
				Expect(foundResource.Spec.Template.Spec.Affinity).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.Affinity))
				Expect(foundResource.Spec.Template.Spec.Tolerations).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.Tolerations))
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("should remove node placement if missing in HCO CR", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
				hcoNodePlacement := commontestutils.NewHco()
				hcoNodePlacement.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hcoNodePlacement.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()

				existingResource := deploymentManifestor(hcoNodePlacement)

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.NodeSelector).ToNot(BeEmpty())
				Expect(existingResource.Spec.Template.Spec.Affinity).ToNot(BeNil())
				Expect(existingResource.Spec.Template.Spec.Tolerations).ToNot(BeEmpty())
				Expect(foundResource.Spec.Template.Spec.NodeSelector).To(BeEmpty())
				Expect(foundResource.Spec.Template.Spec.Affinity).To(BeNil())
				Expect(foundResource.Spec.Template.Spec.Tolerations).To(BeEmpty())
				Expect(req.Conditions).To(BeEmpty())
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("should modify node placement according to HCO CR", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {

				hco.Spec.Workloads.NodePlacement = commontestutils.NewNodePlacement()
				hco.Spec.Infra.NodePlacement = commontestutils.NewOtherNodePlacement()

				existingResource := deploymentManifestor(hco)

				// now, modify HCO's node placement
				hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, v1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: ptr.To[int64](34),
				})
				hco.Spec.Infra.NodePlacement.NodeSelector["key3"] = "something entirely else"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(existingResource.Spec.Template.Spec.Tolerations).To(HaveLen(2))
				Expect(existingResource.Spec.Template.Spec.NodeSelector).Should(HaveKeyWithValue("key3", "value3"))

				Expect(foundResource.Spec.Template.Spec.Affinity.NodeAffinity).ToNot(BeNil())
				Expect(foundResource.Spec.Template.Spec.Tolerations).To(HaveLen(3))
				Expect(foundResource.Spec.Template.Spec.NodeSelector).Should(HaveKeyWithValue("key3", "something entirely else"))

				Expect(req.Conditions).To(BeEmpty())
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("should overwrite node placement if directly set on Kubevirt Console Plugin Deployment", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(*hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {

				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewOtherNodePlacement()}
				existingResource := deploymentManifestor(hco)

				// mock a reconciliation triggered by a change in the deployment
				req.HCOTriggered = false

				// now, modify deployment Kubevirt Console Plugin Deployment node placement
				existingResource.Spec.Template.Spec.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, v1.Toleration{
					Key: "key34", Operator: "operator34", Value: "value34", Effect: "effect34", TolerationSeconds: ptr.To[int64](34),
				})
				existingResource.Spec.Template.Spec.NodeSelector["key3"] = "BADvalue3"

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeTrue())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Template.Spec.NodeSelector).Should(HaveKeyWithValue("key3", "BADvalue3"))

				Expect(foundResource.Spec.Template.Spec.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Template.Spec.NodeSelector).Should(HaveKeyWithValue("key3", "value3"))

				Expect(req.Conditions).To(BeEmpty())
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("apply only NodeSelector if missing", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(converged *hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
				existingResource := deploymentManifestor(hco)

				hco.Spec.Infra.NodePlacement = &sdkapi.NodePlacement{}
				hco.Spec.Infra.NodePlacement.NodeSelector = commontestutils.NewNodePlacement().NodeSelector

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.NodeSelector).To(BeEmpty())
				Expect(foundResource.Spec.Template.Spec.NodeSelector).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.NodeSelector))
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("apply only Affinity if missing", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(converged *hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
				existingResource := deploymentManifestor(hco)

				hco.Spec.Infra.NodePlacement = &sdkapi.NodePlacement{}
				hco.Spec.Infra.NodePlacement.Affinity = commontestutils.NewNodePlacement().Affinity

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.Affinity).To(BeNil())
				Expect(foundResource.Spec.Template.Spec.Affinity).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.Affinity))
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)

			DescribeTable("apply only Tolerations if missing", func(appComponent hcoutil.AppComponent,
				deploymentManifestor func(converged *hcov1beta1.HyperConverged) *appsv1.Deployment, handlerFunc GetHandler) {
				existingResource := deploymentManifestor(hco)

				hco.Spec.Infra.NodePlacement = &sdkapi.NodePlacement{}
				hco.Spec.Infra.NodePlacement.Tolerations = commontestutils.NewNodePlacement().Tolerations

				cl := commontestutils.InitClient([]client.Object{hco, existingResource})
				handlers, err := handlerFunc(logger, cl, commontestutils.GetScheme(), hco)

				Expect(err).ToNot(HaveOccurred())
				res := handlers[0].ensure(req)
				Expect(res.Created).To(BeFalse())
				Expect(res.Updated).To(BeTrue())
				Expect(res.Overwritten).To(BeFalse())
				Expect(res.UpgradeDone).To(BeFalse())
				Expect(res.Err).ToNot(HaveOccurred())

				foundResource := &appsv1.Deployment{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Template.Spec.Tolerations).To(BeEmpty())
				Expect(foundResource.Spec.Template.Spec.Tolerations).To(BeEquivalentTo(hco.Spec.Infra.NodePlacement.Tolerations))
			},
				Entry("plugin deployment", hcoutil.AppComponentUIPlugin, NewKvUIPluginDeployment, newKvUIPluginDeploymentHandler),
				Entry("proxy deployment", hcoutil.AppComponentUIProxy, NewKvUIProxyDeployment, newKvUIProxyDeploymentHandler),
			)
		})
	})

	Context("Kubevirt Plugin and UI Proxy Service", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commontestutils.NewHco()
			req = commontestutils.NewReq(hco)
		})

		DescribeTable("should create service if not present", func(appComponent hcoutil.AppComponent,
			serviceManifestor func(*hcov1beta1.HyperConverged) *v1.Service) {
			var expectedResource *v1.Service
			var handler *genericOperand
			cl := commontestutils.InitClient([]client.Object{})
			expectedResource = serviceManifestor(hco)
			handler = (*genericOperand)(newServiceHandler(cl, commontestutils.GetScheme(), serviceManifestor))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, commontestutils.Name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		},
			Entry("ui plugin service", hcoutil.AppComponentUIPlugin, NewKvUIPluginSvc),
			Entry("ui proxy service", hcoutil.AppComponentUIProxy, NewKvUIProxySvc),
		)

		DescribeTable("should find service if present", func(appComponent hcoutil.AppComponent,
			serviceManifestor func(*hcov1beta1.HyperConverged) *v1.Service) {

			expectedResource := serviceManifestor(hco)
			cl := commontestutils.InitClient([]client.Object{hco, expectedResource})
			handler := (*genericOperand)(newServiceHandler(cl, commontestutils.GetScheme(), serviceManifestor))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		},
			Entry("ui plugin service", hcoutil.AppComponentUIPlugin, NewKvUIPluginSvc),
			Entry("ui proxy service", hcoutil.AppComponentUIProxy, NewKvUIProxySvc),
		)

		DescribeTable("should reconcile service to default if changed", func(appComponent hcoutil.AppComponent,
			serviceManifestor func(*hcov1beta1.HyperConverged) *v1.Service) {

			expectedResource := serviceManifestor(hco)
			outdatedResource := serviceManifestor(hco)

			outdatedResource.ObjectMeta.Labels[hcoutil.AppLabel] = "wrong label"
			outdatedResource.Spec.Ports[0].Port = 6666

			cl := commontestutils.InitClient([]client.Object{hco, outdatedResource})
			handler := (*genericOperand)(newServiceHandler(cl, commontestutils.GetScheme(), serviceManifestor))

			res := handler.ensure(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).ToNot(HaveOccurred())

			foundResource := &v1.Service{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(Succeed())

			Expect(foundResource.ObjectMeta.Labels).ToNot(Equal(outdatedResource.ObjectMeta.Labels))
			Expect(foundResource.ObjectMeta.Labels).To(Equal(expectedResource.ObjectMeta.Labels))
			Expect(foundResource.Spec.Ports).ToNot(Equal(outdatedResource.Spec.Ports))
			Expect(foundResource.Spec.Ports).To(Equal(expectedResource.Spec.Ports))

			// ObjectReference should have been updated
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRefOutdated, err := reference.GetReference(commontestutils.GetScheme(), outdatedResource)
			Expect(err).ToNot(HaveOccurred())
			objectRefFound, err := reference.GetReference(commontestutils.GetScheme(), foundResource)
			Expect(err).ToNot(HaveOccurred())
			Expect(hco.Status.RelatedObjects).To(Not(ContainElement(*objectRefOutdated)))
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRefFound))
		},
			Entry("ui plugin service", hcoutil.AppComponentUIPlugin, NewKvUIPluginSvc),
			Entry("ui proxy service", hcoutil.AppComponentUIProxy, NewKvUIProxySvc),
		)
	})

})
