package hyperconverged

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"

	// TODO: Move to envtest to get an actual api server
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	sspopv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis"
	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	networkaddonsv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"

	// networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

// name and namespace of our primary resource
var name = "hyperconverged-cluster"
var namespace = "kubevirt-hyperconverged"

// Mock request to simulate Reconcile() being called on an event for a watched resource
var request = reconcile.Request{
	NamespacedName: types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	},
}

var _ = Describe("HyperconvergedController", func() {

	Describe("HyperConverged Components", func() {
		Context("KubeVirt Config", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtConfigForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtConfig(hco, log, request)).To(BeNil())

				foundResource := &corev1.ConfigMap{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtConfig(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})
		})

		Context("KubeVirt Storage Config", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtStorageConfig(hco, log, request)).To(BeNil())

				foundResource := &corev1.ConfigMap{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtStorageConfig(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("volumeMode should be block when platform is baremetal", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{
						BareMetalPlatform: true,
					},
				}

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["volumeMode"]).To(Equal("Block"))
			})

			It("volumeMode should be block when platform is not baremetal", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{
						BareMetalPlatform: false,
					},
				}

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["volumeMode"]).To(Equal("Filesystem"))
			})

			It("local storage class name should be available when specified", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{
						LocalStorageClassName: "local",
					},
				}

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["local.accessMode"]).To(Equal("ReadWriteOnce"))
				Expect(expectedResource.Data["local.volumeMode"]).To(Equal("Filesystem"))
			})
		})

		Context("KubeVirt", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirt(hco, log, request)).To(BeNil())

				foundResource := &kubevirtv1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirt(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				})))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				expectedResource.Status.Conditions = []kubevirtv1.KubeVirtCondition{
					kubevirtv1.KubeVirtCondition{
						Type:    kubevirtv1.KubeVirtConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "Foo",
						Message: "Bar",
					},
					kubevirtv1.KubeVirtCondition{
						Type:    kubevirtv1.KubeVirtConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "Foo",
						Message: "Bar",
					},
					kubevirtv1.KubeVirtCondition{
						Type:    kubevirtv1.KubeVirtConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "Foo",
						Message: "Bar",
					},
				}
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirt(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtNotAvailable",
					Message: "KubeVirt is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtProgressing",
					Message: "KubeVirt is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtProgressing",
					Message: "KubeVirt is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtDegraded",
					Message: "KubeVirt is degraded: Bar",
				})))
			})
		})

		Context("CDI", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newCDIForCR(hco, UndefinedNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureCDI(hco, log, request)).To(BeNil())

				foundResource := &cdiv1alpha1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newCDIForCR(hco, UndefinedNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureCDI(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				})))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newCDIForCR(hco, UndefinedNamespace)
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
				Expect(r.ensureCDI(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDINotAvailable",
					Message: "CDI is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIProgressing",
					Message: "CDI is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIProgressing",
					Message: "CDI is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIDegraded",
					Message: "CDI is degraded: Bar",
				})))
			})
		})

		Context("NetworkAddonsConfig", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newNetworkAddonsForCR(hco, UndefinedNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureNetworkAddons(hco, log, request)).To(BeNil())

				foundResource := &networkaddonsv1alpha1.NetworkAddonsConfig{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
				Expect(foundResource.Spec.Multus).To(Equal(&networkaddonsv1alpha1.Multus{}))
				Expect(foundResource.Spec.LinuxBridge).To(Equal(&networkaddonsv1alpha1.LinuxBridge{}))
				Expect(foundResource.Spec.KubeMacPool).To(Equal(&networkaddonsv1alpha1.KubeMacPool{}))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newNetworkAddonsForCR(hco, UndefinedNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureNetworkAddons(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				})))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newNetworkAddonsForCR(hco, UndefinedNamespace)
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
				Expect(r.ensureNetworkAddons(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigNotAvailable",
					Message: "NetworkAddonsConfig is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigProgressing",
					Message: "NetworkAddonsConfig is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigProgressing",
					Message: "NetworkAddonsConfig is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigDegraded",
					Message: "NetworkAddonsConfig is degraded: Bar",
				})))
			})
		})

		Context("KubeVirtCommonTemplatesBundle", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtCommonTemplateBundle(hco, log, request)).To(BeNil())

				foundResource := &sspv1.KubevirtCommonTemplatesBundle{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtCommonTemplateBundle(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

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
				Expect(r.ensureKubeVirtCommonTemplateBundle(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtCommonTemplatesBundleNotAvailable",
					Message: "KubevirtCommonTemplatesBundle is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtCommonTemplatesBundleProgressing",
					Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtCommonTemplatesBundleProgressing",
					Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtCommonTemplatesBundleDegraded",
					Message: "KubevirtCommonTemplatesBundle is degraded: Bar",
				})))
			})
		})

		Context("KubeVirtNodeLabellerBundle", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtNodeLabellerBundle(hco, log, request)).To(BeNil())

				foundResource := &sspv1.KubevirtNodeLabellerBundle{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtNodeLabellerBundle(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

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
				Expect(r.ensureKubeVirtNodeLabellerBundle(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtNodeLabellerBundleNotAvailable",
					Message: "KubevirtNodeLabellerBundle is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtNodeLabellerBundleProgressing",
					Message: "KubevirtNodeLabellerBundle is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtNodeLabellerBundleProgressing",
					Message: "KubevirtNodeLabellerBundle is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtNodeLabellerBundleDegraded",
					Message: "KubevirtNodeLabellerBundle is degraded: Bar",
				})))
			})
		})

		Context("KubeVirtTemplateValidator", func() {
			It("should create if not present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtTemplateValidator(hco, log, request)).To(BeNil())

				foundResource := &sspv1.KubevirtTemplateValidator{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Labels).Should(HaveKeyWithValue("app", name))
				Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			})

			It("should find if present", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				Expect(r.ensureKubeVirtTemplateValidator(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("should handle conditions", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

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
				Expect(r.ensureKubeVirtTemplateValidator(hco, log, request)).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorNotAvailable",
					Message: "KubevirtTemplateValidator is not available: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtTemplateValidatorProgressing",
					Message: "KubevirtTemplateValidator is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorProgressing",
					Message: "KubevirtTemplateValidator is progressing: Bar",
				})))
				Expect(r.conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtTemplateValidatorDegraded",
					Message: "KubevirtTemplateValidator is degraded: Bar",
				})))
			})
		})

		Context("Manage IMS Config", func() {
			It("should error if environment vars not specified", func() {
				os.Unsetenv("CONVERSION_CONTAINER")
				os.Unsetenv("VMWARE_CONTAINER")
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}

				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				Expect(r.ensureIMSConfig(hco, log, request)).ToNot(BeNil())
			})
		})
	})

	Describe("Reconcile HyperConverged", func() {
		Context("HCO Lifecycle", func() {

			It("should handle not found", func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)

				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))
			})

			It("should create all managed resources", func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
				}
				cl := initClient([]runtime.Object{hco})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				// Get the HCO
				foundResource := &hcov1alpha1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1alpha1.ConditionReconcileComplete,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
			})

			It("should find all managed resources", func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
					Status: hcov1alpha1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{
							conditionsv1.Condition{
								Type:    hcov1alpha1.ConditionReconcileComplete,
								Status:  corev1.ConditionTrue,
								Reason:  reconcileCompleted,
								Message: reconcileCompletedMessage,
							},
						},
					},
				}
				// These are all of the objects that we expect to "find" in the client because
				// we already created them in a previous reconcile.
				expectedKVConfig := newKubeVirtConfigForCR(hco, namespace)
				expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
				expectedKVStorageConfig := newKubeVirtStorageConfigForCR(hco, namespace)
				expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
				expectedKV := newKubeVirtForCR(hco, namespace)
				expectedKV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/kubevirts/%s", expectedKV.Namespace, expectedKV.Name)
				expectedCDI := newCDIForCR(hco, UndefinedNamespace)
				expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
				expectedCNA := newNetworkAddonsForCR(hco, UndefinedNamespace)
				expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
				expectedKVCTB := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
				expectedKVNLB := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
				expectedKVTV := newKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
				// Add all of the objects to the client
				cl := initClient([]runtime.Object{hco, expectedKVConfig, expectedKVStorageConfig, expectedKV, expectedCDI, expectedCNA, expectedKVCTB, expectedKVNLB, expectedKVTV})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1alpha1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1alpha1.ConditionReconcileComplete,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				// Why Template validator? Because it is the last to be checked, so the last missing overwrites everything
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorConditions",
					Message: "KubevirtTemplateValidator resource has no conditions",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtTemplateValidatorConditions",
					Message: "KubevirtTemplateValidator resource has no conditions",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorConditions",
					Message: "KubevirtTemplateValidator resource has no conditions",
				})))
			})

			It("should complete when components are finished", func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
					Status: hcov1alpha1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{
							conditionsv1.Condition{
								Type:    hcov1alpha1.ConditionReconcileComplete,
								Status:  corev1.ConditionTrue,
								Reason:  reconcileCompleted,
								Message: reconcileCompletedMessage,
							},
						},
					},
				}
				// These are all of the objects that we expect to "find" in the client because
				// we already created them in a previous reconcile.
				expectedKVConfig := newKubeVirtConfigForCR(hco, namespace)
				expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
				expectedKVStorageConfig := newKubeVirtStorageConfigForCR(hco, namespace)
				expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
				expectedKV := newKubeVirtForCR(hco, namespace)
				expectedKV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/kubevirts/%s", expectedKV.Namespace, expectedKV.Name)
				expectedKV.Status.Conditions = []kubevirtv1.KubeVirtCondition{
					kubevirtv1.KubeVirtCondition{
						Type:   kubevirtv1.KubeVirtConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					kubevirtv1.KubeVirtCondition{
						Type:   kubevirtv1.KubeVirtConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					kubevirtv1.KubeVirtCondition{
						Type:   kubevirtv1.KubeVirtConditionDegraded,
						Status: corev1.ConditionFalse,
					},
				}
				expectedCDI := newCDIForCR(hco, UndefinedNamespace)
				expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
				expectedCDI.Status.Conditions = []conditionsv1.Condition{
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionDegraded,
						Status: corev1.ConditionFalse,
					},
				}
				expectedCNA := newNetworkAddonsForCR(hco, UndefinedNamespace)
				expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
				expectedCNA.Status.Conditions = []conditionsv1.Condition{
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					conditionsv1.Condition{
						Type:   conditionsv1.ConditionDegraded,
						Status: corev1.ConditionFalse,
					},
				}
				expectedKVCTB := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
				expectedKVCTB.Status.Conditions = getGenericCompletedConditions()
				expectedKVNLB := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
				expectedKVNLB.Status.Conditions = getGenericCompletedConditions()
				expectedKVTV := newKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
				expectedKVTV.Status.Conditions = getGenericCompletedConditions()
				// Add all of the objects to the client
				cl := initClient([]runtime.Object{hco, expectedKVConfig, expectedKVStorageConfig, expectedKV, expectedCDI, expectedCNA, expectedKVCTB, expectedKVNLB, expectedKVTV})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1alpha1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1alpha1.ConditionReconcileComplete,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
			})
		})
	})
})

func getGenericCompletedConditions() []conditionsv1.Condition {
	return []conditionsv1.Condition{
		conditionsv1.Condition{
			Type:   conditionsv1.ConditionAvailable,
			Status: corev1.ConditionTrue,
		},
		conditionsv1.Condition{
			Type:   conditionsv1.ConditionProgressing,
			Status: corev1.ConditionFalse,
		},
		conditionsv1.Condition{
			Type:   conditionsv1.ConditionDegraded,
			Status: corev1.ConditionFalse,
		},
	}
}

func initClient(clientObjects []runtime.Object) client.Client {
	// Create a fake client to mock API calls
	return fake.NewFakeClient(clientObjects...)
}

func initReconciler(client client.Client) *ReconcileHyperConverged {
	// Setup Scheme for all resources
	s := scheme.Scheme
	for _, f := range []func(*runtime.Scheme) error{
		apis.AddToScheme,
		cdiv1alpha1.AddToScheme,
		networkaddons.AddToScheme,
		sspopv1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	// Create a ReconcileHyperConverged object with the scheme and fake client
	return &ReconcileHyperConverged{client: client, scheme: s}
}
