package hyperconverged

import (
	"os"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	consolev1 "github.com/openshift/api/console/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"context"
	"fmt"

	"k8s.io/client-go/tools/reference"
)

var _ = Describe("HyperConverged Components", func() {

	Context("KubeVirtCommonTemplatesBundle", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := hco.NewKubeVirtCommonTemplateBundle()
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureKubeVirtCommonTemplateBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtCommonTemplatesBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewKubeVirtCommonTemplateBundle()
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtCommonTemplateBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := hco.NewKubeVirtCommonTemplateBundle()
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Version = "Non default value"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtCommonTemplateBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtCommonTemplatesBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Version).To(BeEmpty())
		})

		// TODO: add tests to ensure that HCO properly propagates NodePlacement from its CR

		// TODO: temporary avoid checking conditions on KubevirtCommonTemplatesBundle because it's currently
		// broken on k8s. Revert this when we will be able to fix it
		/*
			It("should handle conditions", func() {
				expectedResource := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
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
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtCommonTemplateBundle(req)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtCommonTemplatesBundleNotAvailable",
					Message: "KubevirtCommonTemplatesBundle is not available: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtCommonTemplatesBundleProgressing",
					Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtCommonTemplatesBundleProgressing",
					Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtCommonTemplatesBundleDegraded",
					Message: "KubevirtCommonTemplatesBundle is degraded: Bar",
				})))
			})
		*/
	})

	Context("KubeVirtNodeLabellerBundle", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtNodeLabellerBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Version = "Non default value"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtNodeLabellerBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Version).To(BeEmpty())
		})

		It("should add node placement if missing in KubeVirtNodeLabellerBundle", func() {
			existingResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)

			hco.Spec.Workloads.NodePlacement = commonTestUtils.NewHyperConvergedConfig()

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtNodeLabellerBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).To(BeNil())
			Expect(existingResource.Spec.Affinity.PodAffinity).To(BeNil())
			Expect(existingResource.Spec.Affinity.PodAntiAffinity).To(BeNil())
			Expect(foundResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(foundResource.Spec.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(foundResource.Spec.Tolerations).Should(Equal(hco.Spec.Workloads.NodePlacement.Tolerations))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should remove node placement if missing in HCO CR", func() {

			hcoNodePlacement := commonTestUtils.NewHco()
			hcoNodePlacement.Spec.Workloads.NodePlacement = commonTestUtils.NewHyperConvergedConfig()
			existingResource := newKubeVirtNodeLabellerBundleForCR(hcoNodePlacement, namespace)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtNodeLabellerBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.Affinity.NodeAffinity).To(BeNil())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify node placement according to HCO CR", func() {

			hco.Spec.Workloads.NodePlacement = commonTestUtils.NewHyperConvergedConfig()
			existingResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Workloads.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtNodeLabellerBundle(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtNodeLabellerBundle{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(existingResource.Spec.Tolerations).To(HaveLen(2))
			Expect(existingResource.Spec.NodeSelector["key1"]).Should(Equal("value1"))

			Expect(foundResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.Tolerations).To(HaveLen(3))
			Expect(foundResource.Spec.NodeSelector["key1"]).Should(Equal("something else"))

			Expect(req.Conditions).To(BeEmpty())
		})

		// TODO: temporary avoid checking conditions on KubevirtNodeLabellerBundle because it's currently
		// broken on k8s. Revert this when we will be able to fix it
		/*
			It("should handle conditions", func() {
				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
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
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtNodeLabellerBundle(req)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtNodeLabellerBundleNotAvailable",
					Message: "KubevirtNodeLabellerBundle is not available: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtNodeLabellerBundleProgressing",
					Message: "KubevirtNodeLabellerBundle is progressing: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtNodeLabellerBundleProgressing",
					Message: "KubevirtNodeLabellerBundle is progressing: Bar",
				})))
				Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtNodeLabellerBundleDegraded",
					Message: "KubevirtNodeLabellerBundle is degraded: Bar",
				})))
			})
		*/

		//It("should request KVM without any extra setting", func() {
		//	os.Unsetenv("KVM_EMULATION")
		//
		//	expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
		//	Expect(expectedResource.Spec.UseKVM).To(BeTrue())
		//})
		//
		//It("should not request KVM if emulation requested", func() {
		//	err := os.Setenv("KVM_EMULATION", "true")
		//	Expect(err).NotTo(HaveOccurred())
		//	defer os.Unsetenv("KVM_EMULATION")
		//
		//	expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
		//	Expect(expectedResource.Spec.UseKVM).To(BeFalse())
		//})

		//It("should request KVM if emulation value not set", func() {
		//	err := os.Setenv("KVM_EMULATION", "")
		//	Expect(err).NotTo(HaveOccurred())
		//	defer os.Unsetenv("KVM_EMULATION")
		//
		//	expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
		//	Expect(expectedResource.Spec.UseKVM).To(BeTrue())
		//})
	})

	Context("KubeVirtTemplateValidator", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtTemplateValidator{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.TemplateValidatorReplicas = 5 // set non-default value

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtTemplateValidator{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.TemplateValidatorReplicas).To(BeZero())
		})

		It("should add node placement if missing in KubeVirtTemplateValidator", func() {
			existingResource := newKubeVirtTemplateValidatorForCR(hco, namespace)

			hco.Spec.Infra.NodePlacement = commonTestUtils.NewHyperConvergedConfig()

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtTemplateValidator{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).To(BeNil())
			Expect(existingResource.Spec.Affinity.PodAffinity).To(BeNil())
			Expect(existingResource.Spec.Affinity.PodAntiAffinity).To(BeNil())
			Expect(foundResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.NodeSelector["key1"]).Should(Equal("value1"))
			Expect(foundResource.Spec.NodeSelector["key2"]).Should(Equal("value2"))

			Expect(foundResource.Spec.Tolerations).Should(Equal(hco.Spec.Infra.NodePlacement.Tolerations))

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should remove node placement if missing in HCO CR", func() {

			hcoNodePlacement := commonTestUtils.NewHco()
			hcoNodePlacement.Spec.Infra.NodePlacement = commonTestUtils.NewHyperConvergedConfig()
			existingResource := newKubeVirtTemplateValidatorForCR(hcoNodePlacement, namespace)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtTemplateValidator{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.Affinity.NodeAffinity).To(BeNil())

			Expect(req.Conditions).To(BeEmpty())
		})

		It("should modify node placement according to HCO CR", func() {

			hco.Spec.Infra.NodePlacement = commonTestUtils.NewHyperConvergedConfig()
			existingResource := newKubeVirtTemplateValidatorForCR(hco, namespace)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Infra.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtTemplateValidator(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtTemplateValidator{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())

			Expect(existingResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(existingResource.Spec.Tolerations).To(HaveLen(2))
			Expect(existingResource.Spec.NodeSelector["key1"]).Should(Equal("value1"))

			Expect(foundResource.Spec.Affinity.NodeAffinity).ToNot(BeNil())
			Expect(foundResource.Spec.Tolerations).To(HaveLen(3))
			Expect(foundResource.Spec.NodeSelector["key1"]).Should(Equal("something else"))

			Expect(req.Conditions).To(BeEmpty())
		})

		// TODO: temporary avoid checking conditions on KubevirtTemplateValidator because it's currently
		// broken on k8s. Revert this when we will be able to fix it
		/*It("should handle conditions", func() {
			expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
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
			cl := initClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			Expect(r.ensureKubeVirtTemplateValidator(req)).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "KubevirtTemplateValidatorNotAvailable",
				Message: "KubevirtTemplateValidator is not available: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "KubevirtTemplateValidatorProgressing",
				Message: "KubevirtTemplateValidator is progressing: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  "KubevirtTemplateValidatorProgressing",
				Message: "KubevirtTemplateValidator is progressing: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionDegraded,
				Status:  corev1.ConditionTrue,
				Reason:  "KubevirtTemplateValidatorDegraded",
				Message: "KubevirtTemplateValidator is degraded: Bar",
			})))
		})*/
	})

	Context("KubeVirtMetricsAggregation", func() {
		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := newKubeVirtMetricsAggregationForCR(hco, namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureKubeVirtMetricsAggregation(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtMetricsAggregation{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newKubeVirtMetricsAggregationForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtMetricsAggregation(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := newKubeVirtMetricsAggregationForCR(hco, namespace)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.Version = "non-default value"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureKubeVirtMetricsAggregation(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Updated).To(BeTrue())
			Expect(res.Err).To(BeNil())

			foundResource := &sspv1.KubevirtMetricsAggregation{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Spec.Version).To(BeEmpty())
		})

		// TODO: add tests to ensure that HCO properly propagates NodePlacement from its CR

		// TODO: temporary avoid checking conditions on KubevirtTemplateValidator because it's currently
		// broken on k8s. Revert this when we will be able to fix it
		/*It("should handle conditions", func() {
			expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
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
			cl := initClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			Expect(r.ensureKubeVirtTemplateValidator(req)).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			// Check conditions
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionAvailable,
				Status:  corev1.ConditionFalse,
				Reason:  "KubevirtTemplateValidatorNotAvailable",
				Message: "KubevirtTemplateValidator is not available: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionProgressing,
				Status:  corev1.ConditionTrue,
				Reason:  "KubevirtTemplateValidatorProgressing",
				Message: "KubevirtTemplateValidator is progressing: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionUpgradeable,
				Status:  corev1.ConditionFalse,
				Reason:  "KubevirtTemplateValidatorProgressing",
				Message: "KubevirtTemplateValidator is progressing: Bar",
			})))
			Expect(req.Conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
				Type:    conditionsv1.ConditionDegraded,
				Status:  corev1.ConditionTrue,
				Reason:  "KubevirtTemplateValidatorDegraded",
				Message: "KubevirtTemplateValidator is degraded: Bar",
			})))
		})*/
	})

	Context("Manage IMS Config", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			os.Setenv("CONVERSION_CONTAINER", "new-conversion-container-value")
			os.Setenv("VMWARE_CONTAINER", "new-vmware-container-value")
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should error if environment vars not specified", func() {
			os.Unsetenv("CONVERSION_CONTAINER")
			os.Unsetenv("VMWARE_CONTAINER")

			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).ToNot(BeNil())
		})

		It("should create if not present", func() {
			expectedResource := newIMSConfigForCR(hco, namespace)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &corev1.ConfigMap{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newIMSConfigForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureIMSConfig(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		// in an ideal world HCO should be managing the whole config map,
		// now due to a bad design only a few values of this config map are
		// really managed by HCO while others are managed by other entities
		// TODO: fix this bad design splitting the config map into two distinct objects and reconcile the whole object here
		It("should (partially!!!) reconcile according to env values", func() {
			convk := "v2v-conversion-image"
			vmwarek := "kubevirt-vmware-image"
			updatableKeys := [...]string{convk, vmwarek}
			unupdatableKeyValues := map[string]string{
				"ext_key_1": "ext_value_1",
				"ext_key_2": "ext_value_2",
			}

			expectedResource := newIMSConfigForCR(hco, namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			outdatedResource := newIMSConfigForCR(hco, namespace)
			outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
			// values we should update
			outdatedResource.Data[convk] = "old-conversion-container-value-we-have-to-update"
			outdatedResource.Data[vmwarek] = "old-vmware-container-value-we-have-to-update"
			// add values we should not touch
			for k, v := range unupdatableKeyValues {
				outdatedResource.Data[k] = v
			}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, outdatedResource})
			r := initReconciler(cl)

			res := r.ensureIMSConfig(req)
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

			for k, v := range unupdatableKeyValues {
				Expect(foundResource.Data).To(HaveKeyWithValue(k, v))
			}

		})

	})

	Context("Vm Import", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := newVMImportForCR(hco)
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)

			res := r.ensureVMImport(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			foundResource := &vmimportv1beta1.VMImportConfig{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := newVMImportForCR(hco)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			res := r.ensureVMImport(req)
			Expect(res.UpgradeDone).To(BeFalse())
			Expect(res.Err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		It("should reconcile to default", func() {
			existingResource := newVMImportForCR(hco)
			existingResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", existingResource.Namespace, existingResource.Name)

			existingResource.Spec.ImagePullPolicy = corev1.PullAlways // set non-default value

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureVMImport(req)
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
			existingResource := newVMImportForCR(hco)

			hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}
			hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewHyperConvergedConfig()}

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureVMImport(req)
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
			existingResource := newVMImportForCR(hcoNodePlacement)

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureVMImport(req)
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
			existingResource := newVMImportForCR(hco)

			// now, modify HCO's node placement
			seconds3 := int64(3)
			hco.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
				Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
			})

			hco.Spec.Infra.NodePlacement.NodeSelector["key1"] = "something else"

			cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
			r := initReconciler(cl)
			res := r.ensureVMImport(req)
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

	Context("ConsoleCLIDownload", func() {

		var hco *hcov1beta1.HyperConverged
		var req *common.HcoRequest

		BeforeEach(func() {
			hco = commonTestUtils.NewHco()
			req = commonTestUtils.NewReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := hco.NewConsoleCLIDownload()
			cl := commonTestUtils.InitClient([]runtime.Object{})
			r := initReconciler(cl)

			err := r.ensureConsoleCLIDownload(req)
			Expect(err).To(BeNil())

			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(
				cl.Get(context.TODO(),
					types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
					foundResource),
			).To(BeNil())
			Expect(foundResource.Name).To(Equal(expectedResource.Name))
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewConsoleCLIDownload()
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/consoleclidownloads/%s", expectedResource.Namespace, expectedResource.Name)
			cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedResource})
			r := initReconciler(cl)
			err := r.ensureConsoleCLIDownload(req)
			Expect(err).To(BeNil())

			// Check HCO's status
			Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
			objectRef, err := reference.GetReference(r.scheme, expectedResource)
			Expect(err).To(BeNil())
			// ObjectReference should have been added
			Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
		})

		DescribeTable("should update if something changed", func(modifiedResource *consolev1.ConsoleCLIDownload) {
			os.Setenv(hcoutil.KubevirtVersionEnvV, "100")
			cl := commonTestUtils.InitClient([]runtime.Object{modifiedResource})
			r := initReconciler(cl)
			err := r.ensureConsoleCLIDownload(req)
			Expect(err).To(BeNil())
			expectedResource := hco.NewConsoleCLIDownload()
			key, err := client.ObjectKeyFromObject(expectedResource)
			Expect(err).ToNot(HaveOccurred())
			foundResource := &consolev1.ConsoleCLIDownload{}
			Expect(cl.Get(context.TODO(), key, foundResource))
			Expect(foundResource.Spec.Links[0].Href).To(Equal(expectedResource.Spec.Links[0].Href))
			Expect(foundResource.Spec.Links[0].Text).To(Equal(expectedResource.Spec.Links[0].Text))
		},
			Entry("with modified download link",
				&consolev1.ConsoleCLIDownload{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "console.openshift.io/v1",
						Kind:       "ConsoleCLIDownload",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "virtctl-clidownloads-kubevirt-hyperconverged",
					},

					Spec: consolev1.ConsoleCLIDownloadSpec{
						Links: []consolev1.CLIDownloadLink{
							{
								Href: "https://dummy.url1.com",
								Text: "KubeVirt 100 release downloads",
							},
						},
					},
				}),
			Entry("with modified download text",
				&consolev1.ConsoleCLIDownload{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "console.openshift.io/v1",
						Kind:       "ConsoleCLIDownload",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "virtctl-clidownloads-kubevirt-hyperconverged",
					},
					Spec: consolev1.ConsoleCLIDownloadSpec{
						Links: []consolev1.CLIDownloadLink{
							{
								Href: "https://github.com/kubevirt/kubevirt/releases/100",
								Text: "dummy text 1",
							},
						},
					},
				}),
		)
	})
})
