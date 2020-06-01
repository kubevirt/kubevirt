package hyperconverged

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	vmimportv1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1alpha1"
	k8sTime "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
)

// name and namespace of our primary resource
var name = "kubevirt-hyperconverged"
var namespace = "kubevirt-hyperconverged"

// Mock request to simulate Reconcile() being called on an event for a watched resource
var (
	request = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
)

func newHco() *hcov1alpha1.HyperConverged {
	return &hcov1alpha1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hcov1alpha1.HyperConvergedSpec{},
	}
}

func newReq(inst *hcov1alpha1.HyperConverged) *hcoRequest {
	return &hcoRequest{
		Request:    request,
		logger:     log,
		conditions: newHcoConditions(),
		ctx:        context.TODO(),
		instance:   inst,
	}
}

var _ = Describe("HyperconvergedController", func() {

	Describe("HyperConverged Components", func() {

		Context("KubeVirt Priority Classes", func() {

			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtPriorityClass()
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtPriorityClass(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				key, err := client.ObjectKeyFromObject(expectedResource)
				Expect(err).ToNot(HaveOccurred())
				foundResource := &schedulingv1.PriorityClass{}
				Expect(cl.Get(context.TODO(), key, foundResource)).To(BeNil())
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Value).To(Equal(expectedResource.Value))
				Expect(foundResource.GlobalDefault).To(Equal(expectedResource.GlobalDefault))
			})

			It("should do nothing if already exists", func() {
				expectedResource := newKubeVirtPriorityClass()
				cl := initClient([]runtime.Object{expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtPriorityClass(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			DescribeTable("should update if something changed", func(modifiedResource *schedulingv1.PriorityClass) {
				cl := initClient([]runtime.Object{modifiedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtPriorityClass(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				expectedResource := newKubeVirtPriorityClass()
				key, err := client.ObjectKeyFromObject(expectedResource)
				Expect(err).ToNot(HaveOccurred())
				foundResource := &schedulingv1.PriorityClass{}
				Expect(cl.Get(context.TODO(), key, foundResource))
				Expect(foundResource.Name).To(Equal(expectedResource.Name))
				Expect(foundResource.Value).To(Equal(expectedResource.Value))
				Expect(foundResource.GlobalDefault).To(Equal(expectedResource.GlobalDefault))
			},
				Entry("with modified value",
					&schedulingv1.PriorityClass{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "scheduling.k8s.io/v1",
							Kind:       "PriorityClass",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "kubevirt-cluster-critical",
						},
						Value:         1,
						GlobalDefault: false,
						Description:   "",
					}),
				Entry("with modified global default",
					&schedulingv1.PriorityClass{
						TypeMeta: metav1.TypeMeta{
							APIVersion: "scheduling.k8s.io/v1",
							Kind:       "PriorityClass",
						},
						ObjectMeta: metav1.ObjectMeta{
							Name: "kubevirt-cluster-critical",
						},
						Value:         1000000000,
						GlobalDefault: true,
						Description:   "",
					}),
			)

		})

		Context("KubeVirt Config", func() {

			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			updatableKeys := [...]string{virtconfig.SmbiosConfigKey, virtconfig.MachineTypeKey, virtconfig.SELinuxLauncherTypeKey}
			unupdatableKeys := [...]string{virtconfig.FeatureGatesKey, virtconfig.MigrationsConfigKey}

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)

				os.Setenv("SMBIOS", "new-smbios-value-that-we-have-to-set")
				os.Setenv("MACHINETYPE", "new-machinetype-value-that-we-have-to-set")
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtConfigForCR(req.instance, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("should update only a few keys and only when in upgrade mode", func() {
				expectedResource := newKubeVirtConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				outdatedResource := newKubeVirtConfigForCR(hco, namespace)
				outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
				// values we should update
				outdatedResource.Data[virtconfig.SmbiosConfigKey] = "old-smbios-value-that-we-have-to-update"
				outdatedResource.Data[virtconfig.MachineTypeKey] = "old-machinetype-value-that-we-have-to-update"
				outdatedResource.Data[virtconfig.SELinuxLauncherTypeKey] = "old-selinuxlauncher-value-that-we-have-to-update"
				// values we should preserve
				outdatedResource.Data[virtconfig.FeatureGatesKey] = "old-featuregates-value-that-we-should-preserve"
				outdatedResource.Data[virtconfig.MigrationsConfigKey] = "old-migrationsconfig-value-that-we-should-preserve"

				cl := initClient([]runtime.Object{hco, outdatedResource})
				r := initReconciler(cl)

				// force upgrade mode
				r.upgradeMode = true
				upgradeDone, err := r.ensureKubeVirtConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				for _, k := range unupdatableKeys {
					Expect(foundResource.Data[k]).To(Equal(outdatedResource.Data[k]))
					Expect(foundResource.Data[k]).To(Not(Equal(expectedResource.Data[k])))
				}
			})

			It("should not touch it when not in in upgrade mode", func() {
				expectedResource := newKubeVirtConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				outdatedResource := newKubeVirtConfigForCR(hco, namespace)
				outdatedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", outdatedResource.Namespace, outdatedResource.Name)
				// values we should update
				outdatedResource.Data[virtconfig.SmbiosConfigKey] = "old-smbios-value-that-we-have-to-update"
				outdatedResource.Data[virtconfig.MachineTypeKey] = "old-machinetype-value-that-we-have-to-update"
				outdatedResource.Data[virtconfig.SELinuxLauncherTypeKey] = "old-selinuxlauncher-value-that-we-have-to-update"
				// values we should preserve
				outdatedResource.Data[virtconfig.FeatureGatesKey] = "old-featuregates-value-that-we-should-preserve"
				outdatedResource.Data[virtconfig.MigrationsConfigKey] = "old-migrationsconfig-value-that-we-should-preserve"

				cl := initClient([]runtime.Object{hco, outdatedResource})
				r := initReconciler(cl)

				// ensure that we are not in upgrade mode
				r.upgradeMode = false

				upgradeDone, err := r.ensureKubeVirtConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				foundResource := &corev1.ConfigMap{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Data).To(Equal(outdatedResource.Data))
				Expect(foundResource.Data).To(Not(Equal(expectedResource.Data)))
			})
		})

		Context("KubeVirt Storage Config", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtStorageConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtStorageConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

			It("volumeMode should be filesystem when platform is baremetal", func() {
				hco.Spec.BareMetalPlatform = true

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["volumeMode"]).To(Equal("Filesystem"))
			})

			It("volumeMode should be filesystem when platform is not baremetal", func() {
				hco.Spec.BareMetalPlatform = false

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["volumeMode"]).To(Equal("Filesystem"))
			})

			It("local storage class name should be available when specified", func() {
				hco.Spec.LocalStorageClassName = "local"

				expectedResource := newKubeVirtStorageConfigForCR(hco, namespace)
				Expect(expectedResource.Data["local.accessMode"]).To(Equal("ReadWriteOnce"))
				Expect(expectedResource.Data["local.volumeMode"]).To(Equal("Filesystem"))
			})
		})

		Context("KubeVirt", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirt(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirt(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtConditions",
					Message: "KubeVirt resource has no conditions",
				}))
			})

			It("should set default UninstallStrategy if missing", func() {
				expectedResource := newKubeVirtForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				missingUSResource := newKubeVirtForCR(hco, namespace)
				missingUSResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", missingUSResource.Namespace, missingUSResource.Name)
				missingUSResource.Spec.UninstallStrategy = ""

				cl := initClient([]runtime.Object{hco, missingUSResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirt(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				foundResource := &kubevirtv1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(foundResource.Spec.UninstallStrategy).To(Equal(expectedResource.Spec.UninstallStrategy))
			})

			It("should handle conditions", func() {
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
				upgradeDone, err := r.ensureKubeVirt(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtNotAvailable",
					Message: "KubeVirt is not available: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtProgressing",
					Message: "KubeVirt is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubeVirtProgressing",
					Message: "KubeVirt is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionDegraded]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubeVirtDegraded",
					Message: "KubeVirt is degraded: Bar",
				}))
			})
		})

		Context("CDI", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newCDIForCR(hco, UndefinedNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureCDI(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newCDIForCR(hco, UndefinedNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureCDI(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIConditions",
					Message: "CDI resource has no conditions",
				}))
			})

			It("should set default UninstallStrategy if missing", func() {
				expectedResource := newCDIForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				missingUSResource := newCDIForCR(hco, namespace)
				missingUSResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", missingUSResource.Namespace, missingUSResource.Name)
				missingUSResource.Spec.UninstallStrategy = nil

				cl := initClient([]runtime.Object{hco, missingUSResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureCDI(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				foundResource := &cdiv1alpha1.CDI{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expectedResource.Name, Namespace: expectedResource.Namespace},
						foundResource),
				).To(BeNil())
				Expect(*foundResource.Spec.UninstallStrategy).To(Equal(*expectedResource.Spec.UninstallStrategy))
			})

			It("should handle conditions", func() {
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
				upgradeDone, err := r.ensureCDI(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDINotAvailable",
					Message: "CDI is not available: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIProgressing",
					Message: "CDI is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "CDIProgressing",
					Message: "CDI is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionDegraded]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "CDIDegraded",
					Message: "CDI is degraded: Bar",
				}))
			})
		})

		Context("NetworkAddonsConfig", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newNetworkAddonsForCR(hco, UndefinedNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureNetworkAddons(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newNetworkAddonsForCR(hco, UndefinedNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureNetworkAddons(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigConditions",
					Message: "NetworkAddonsConfig resource has no conditions",
				}))
			})

			It("should handle conditions", func() {
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
				upgradeDone, err := r.ensureNetworkAddons(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
				// Check conditions
				Expect(req.conditions[conditionsv1.ConditionAvailable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigNotAvailable",
					Message: "NetworkAddonsConfig is not available: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionProgressing]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigProgressing",
					Message: "NetworkAddonsConfig is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionUpgradeable]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "NetworkAddonsConfigProgressing",
					Message: "NetworkAddonsConfig is progressing: Bar",
				}))
				Expect(req.conditions[conditionsv1.ConditionDegraded]).To(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "NetworkAddonsConfigDegraded",
					Message: "NetworkAddonsConfig is degraded: Bar",
				}))
			})
		})

		Context("KubeVirtCommonTemplatesBundle", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtCommonTemplateBundle(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtCommonTemplateBundle(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})

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
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubevirtCommonTemplatesBundleNotAvailable",
						Message: "KubevirtCommonTemplatesBundle is not available: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "KubevirtCommonTemplatesBundleProgressing",
						Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubevirtCommonTemplatesBundleProgressing",
						Message: "KubevirtCommonTemplatesBundle is progressing: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "KubevirtCommonTemplatesBundleDegraded",
						Message: "KubevirtCommonTemplatesBundle is degraded: Bar",
					})))
				})
			*/
		})

		Context("KubeVirtNodeLabellerBundle", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtNodeLabellerBundle(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtNodeLabellerBundle(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
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
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionAvailable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubevirtNodeLabellerBundleNotAvailable",
						Message: "KubevirtNodeLabellerBundle is not available: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionProgressing,
						Status:  corev1.ConditionTrue,
						Reason:  "KubevirtNodeLabellerBundleProgressing",
						Message: "KubevirtNodeLabellerBundle is progressing: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionUpgradeable,
						Status:  corev1.ConditionFalse,
						Reason:  "KubevirtNodeLabellerBundleProgressing",
						Message: "KubevirtNodeLabellerBundle is progressing: Bar",
					})))
					Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
						Type:    conditionsv1.ConditionDegraded,
						Status:  corev1.ConditionTrue,
						Reason:  "KubevirtNodeLabellerBundleDegraded",
						Message: "KubevirtNodeLabellerBundle is degraded: Bar",
					})))
				})
			*/

			It("should request KVM without any extra setting", func() {
				os.Unsetenv("KVM_EMULATION")

				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				Expect(expectedResource.Spec.UseKVM).To(BeTrue())
			})

			It("should not request KVM if emulation requested", func() {
				err := os.Setenv("KVM_EMULATION", "true")
				Expect(err).NotTo(HaveOccurred())
				defer os.Unsetenv("KVM_EMULATION")

				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				Expect(expectedResource.Spec.UseKVM).To(BeFalse())
			})

			It("should request KVM if emulation value not set", func() {
				err := os.Setenv("KVM_EMULATION", "")
				Expect(err).NotTo(HaveOccurred())
				defer os.Unsetenv("KVM_EMULATION")

				expectedResource := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
				Expect(expectedResource.Spec.UseKVM).To(BeTrue())
			})
		})

		Context("KubeVirtTemplateValidator", func() {
			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtTemplateValidator(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

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
				expectedResource := newKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureKubeVirtTemplateValidator(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
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
				Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorNotAvailable",
					Message: "KubevirtTemplateValidator is not available: Bar",
				})))
				Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtTemplateValidatorProgressing",
					Message: "KubevirtTemplateValidator is progressing: Bar",
				})))
				Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  "KubevirtTemplateValidatorProgressing",
					Message: "KubevirtTemplateValidator is progressing: Bar",
				})))
				Expect(req.conditions[]).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  "KubevirtTemplateValidatorDegraded",
					Message: "KubevirtTemplateValidator is degraded: Bar",
				})))
			})*/
		})

		Context("Manage IMS Config", func() {
			It("should error if environment vars not specified", func() {
				os.Unsetenv("CONVERSION_CONTAINER")
				os.Unsetenv("VMWARE_CONTAINER")
				req := newReq(newHco())

				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureIMSConfig(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).ToNot(BeNil())
			})
		})

		Context("Vm Import", func() {

			var hco *hcov1alpha1.HyperConverged
			var req *hcoRequest

			BeforeEach(func() {
				hco = newHco()
				req = newReq(hco)
			})

			It("should create if not present", func() {
				expectedResource := newVMImportForCR(hco, namespace)
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)

				upgradeDone, err := r.ensureVMImport(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				foundResource := &vmimportv1.VMImportConfig{}
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
				expectedResource := newVMImportForCR(hco, namespace)
				expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedResource.Namespace, expectedResource.Name)
				cl := initClient([]runtime.Object{hco, expectedResource})
				r := initReconciler(cl)
				upgradeDone, err := r.ensureVMImport(req)
				Expect(upgradeDone).To(BeFalse())
				Expect(err).To(BeNil())

				// Check HCO's status
				Expect(hco.Status.RelatedObjects).To(Not(BeNil()))
				objectRef, err := reference.GetReference(r.scheme, expectedResource)
				Expect(err).To(BeNil())
				// ObjectReference should have been added
				Expect(hco.Status.RelatedObjects).To(ContainElement(*objectRef))
			})
		})
	})

	Describe("Reconcile HyperConverged", func() {
		Context("HCO Lifecycle", func() {

			BeforeEach(func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				os.Setenv("OPERATOR_NAMESPACE", namespace)
			})

			It("should handle not found", func() {
				cl := initClient([]runtime.Object{})
				r := initReconciler(cl)

				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))
			})

			It("should ignore invalid requests", func() {
				hco := &hcov1alpha1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid",
						Namespace: "invalid",
					},
					Spec: hcov1alpha1.HyperConvergedSpec{},
					Status: hcov1alpha1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{},
					},
				}
				cl := initClient([]runtime.Object{hco})
				r := initReconciler(cl)

				// Do the reconcile
				var invalidRequest = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "invalid",
						Namespace: "invalid",
					},
				}
				res, err := r.Reconcile(invalidRequest)
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
					Status:  corev1.ConditionFalse,
					Reason:  invalidRequestReason,
					Message: fmt.Sprintf(invalidRequestMessageFormat, name, namespace),
				})))
			})

			It("should create all managed resources", func() {
				hco := newHco()
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
					Status:  corev1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: reconcileInitMessage,
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
				// TODO: temporary ignoring KubevirtTemplateValidator conditions
				/*
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
				*/
			})

			It("should complete when components are finished", func() {
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
					{
						Type:   kubevirtv1.KubeVirtConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   kubevirtv1.KubeVirtConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   kubevirtv1.KubeVirtConditionDegraded,
						Status: corev1.ConditionFalse,
					},
				}
				expectedCDI := newCDIForCR(hco, UndefinedNamespace)
				expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
				expectedCDI.Status.Conditions = []conditionsv1.Condition{
					{
						Type:   conditionsv1.ConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   conditionsv1.ConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					{
						Type:   conditionsv1.ConditionDegraded,
						Status: corev1.ConditionFalse,
					},
				}
				expectedCNA := newNetworkAddonsForCR(hco, UndefinedNamespace)
				expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
				expectedCNA.Status.Conditions = []conditionsv1.Condition{
					{
						Type:   conditionsv1.ConditionAvailable,
						Status: corev1.ConditionTrue,
					},
					{
						Type:   conditionsv1.ConditionProgressing,
						Status: corev1.ConditionFalse,
					},
					{
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

			It(`should be not available when components with missing "Available" condition`, func() {

				expected := getBasicDeployment()

				origKvConds := expected.kv.Status.Conditions
				expected.kv.Status.Conditions = expected.kv.Status.Conditions[1:]

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				expected.kv.Status.Conditions = origKvConds
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				origConds := expected.cdi.Status.Conditions
				expected.cdi.Status.Conditions = expected.cdi.Status.Conditions[1:]
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				expected.cdi.Status.Conditions = origConds
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				origConds = expected.cna.Status.Conditions
				expected.cna.Status.Conditions = expected.cdi.Status.Conditions[1:]
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				expected.cna.Status.Conditions = origConds
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// TODO: temporary avoid checking conditions on KubevirtCommonTemplatesBundle because it's currently
				// broken on k8s. Revert this when we will be able to fix it
				/*
					origConds = expected.kvCtb.Status.Conditions
					expected.kvCtb.Status.Conditions = expected.cdi.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionFalse)

					expected.kvCtb.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionTrue)
				*/

				// TODO: temporary avoid checking conditions on KubevirtNodeLabellerBundle because it's currently
				// broken on k8s. Revert this when we will be able to fix it
				/*
					origConds = expected.kvNlb.Status.Conditions
					expected.kvNlb.Status.Conditions = expected.cdi.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionFalse)

					expected.kvNlb.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionTrue)
				*/

				// TODO: temporary avoid checking conditions on KubevirtTemplateValidator because it's currently
				// broken on k8s. Revert this when we will be able to fix it
				/*
					origConds = expected.kvTv.Status.Conditions
					expected.kvTv.Status.Conditions = expected.cdi.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionFalse)

					expected.kvTv.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, requeue = doReconcile(cl, expected.hco)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, corev1.ConditionTrue)
				*/
			})

			It(`should delete HCO`, func() {

				// First, create HCO and check it
				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl)
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				foundResource := &hcov1alpha1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(len(foundResource.Status.RelatedObjects)).Should(Equal(12))
				Expect(foundResource.ObjectMeta.Finalizers).Should(Equal([]string{FinalizerName}))

				// Now, delete HCO
				delTime := time.Now().UTC().Add(-1 * time.Minute)
				expected.hco.ObjectMeta.DeletionTimestamp = &k8sTime.Time{Time: delTime}
				expected.hco.ObjectMeta.Finalizers = []string{FinalizerName}
				cl = expected.initClient()

				r = initReconciler(cl)
				res, err = r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				foundResource = &hcov1alpha1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).To(BeNil())
				Expect(foundResource.ObjectMeta.Finalizers).To(BeNil())

			})
		})

		Context("Upgrade Mode", func() {
			expected := getBasicDeployment()
			okConds := expected.hco.Status.Conditions

			const (
				oldVersion          = "1.1.0"
				newVersion          = "1.2.0" // TODO: avoid hard-coding values
				oldComponentVersion = "1.2.0"
				newComponentVersion = "1.2.3"
			)

			BeforeEach(func() {
				os.Setenv("CONVERSION_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-v2v-conversion:v2.0.0")
				os.Setenv("VMWARE_CONTAINER", "registry.redhat.io/container-native-virtualization/kubevirt-vmware:v2.0.0}")
				os.Setenv("OPERATOR_NAMESPACE", namespace)

				expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
				os.Setenv(util.KubevirtVersionEnvV, newComponentVersion)

				expected.cdi.Status.ObservedVersion = newComponentVersion
				os.Setenv(util.CdiVersionEnvV, newComponentVersion)

				expected.cna.Status.ObservedVersion = newComponentVersion
				os.Setenv(util.CnaoVersionEnvV, newComponentVersion)

				expected.vmi.Status.ObservedVersion = newComponentVersion
				os.Setenv(util.VMImportEnvV, newComponentVersion)

				os.Setenv(util.HcoKvIoVersionName, newVersion)
			})

			It("Should update HCO Version Id in the CR on init", func() {

				expected.hco.Status.Conditions = nil

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, corev1.ConditionFalse)

				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == conditionsv1.ConditionAvailable {
						Expect(cond.Reason).Should(Equal("Init"))
						break
					}
				}
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))

				expected.hco.Status.Conditions = okConds
			})

			It("detect upgrade existing HCO Version", func() {
				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the HCO version is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
			})

			It("detect upgrade w/o HCO Version", func() {
				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()
				expected.hco.Status.Versions = nil

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				fmt.Fprintln(GinkgoWriter, "foundResource.Status.Versions", foundResource.Status.Versions)
				Expect(ok).To(BeFalse())
				Expect(ver).Should(BeEmpty())

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
			})

			It("don't complete upgrade if kubevirt version is not match to the kubevirt version env ver", func() {
				os.Setenv(util.HcoKvIoVersionName, newVersion)

				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				expected.kv.Status.ObservedKubeVirtVersion = oldComponentVersion

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond = conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
				Expect(cond.Reason).Should(Equal(reconcileCompleted))
				Expect(cond.Message).Should(Equal(reconcileCompletedMessage))
			})

			It("don't complete upgrade if CDI version is not match to the CDI version env ver", func() {
				os.Setenv(util.HcoKvIoVersionName, newVersion)

				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				expected.cdi.Status.ObservedVersion = oldComponentVersion

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.cdi.Status.ObservedVersion = newComponentVersion
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond = conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
				Expect(cond.Reason).Should(Equal(reconcileCompleted))
				Expect(cond.Message).Should(Equal(reconcileCompletedMessage))
			})

			It("don't complete upgrade if CNA version is not match to the CNA version env ver", func() {
				os.Setenv(util.HcoKvIoVersionName, newVersion)

				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				expected.cna.Status.ObservedVersion = oldComponentVersion

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.cna.Status.ObservedVersion = newComponentVersion
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond = conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
				Expect(cond.Reason).Should(Equal(reconcileCompleted))
				Expect(cond.Message).Should(Equal(reconcileCompletedMessage))
			})

			It("don't complete upgrade if VM-Import version is not match to the VM-Import version env ver", func() {
				os.Setenv(util.HcoKvIoVersionName, newVersion)

				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				expected.vmi.Status.ObservedVersion = ""

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.vmi.Status.ObservedVersion = newComponentVersion
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that HCO version is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond = conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
				Expect(cond.Reason).Should(Equal(reconcileCompleted))
				Expect(cond.Message).Should(Equal(reconcileCompletedMessage))
			})
		})

		Context("Aggregate Negative Conditions", func() {
			const errorReason = "CdiTestError1"
			os.Setenv(util.HcoKvIoVersionName, version.Version)
			It("should be degraded when a component is degraded", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
				Expect(cd.Message).Should(Equal("HCO is not available due to degraded components"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
				Expect(cd.Message).Should(Equal("HCO is not Upgradeable due to degraded components"))

			})

			It("should be degraded when a component is degraded + Progressing", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  "progressingError",
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
			})

			It("should be degraded when a component is degraded + !Available", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "AvailableError",
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
			})

			It("should be Progressing when a component is Progressing", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
			})

			It("should be Progressing when a component is Progressing + !Available", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "AvailableError",
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
			})

			It("should be not Available when a component is not Available", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "AvailableError",
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
			})

			It("should be with all positive condition when all components working properly", func() {
				expected := getBasicDeployment()
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
			})

			It("should set the status of the last faulty component", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "AvailableError",
					Message: "CDI Test Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cna.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionAvailable,
					Status:  corev1.ConditionFalse,
					Reason:  "AvailableError",
					Message: "CNA Test Error message",
				})
				cl := expected.initClient()
				foundResource, _ := doReconcile(cl, expected.hco)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := conditionsv1.FindStatusCondition(conditions, hcov1alpha1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal("NetworkAddonsConfigNotAvailable"))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo("False"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = conditionsv1.FindStatusCondition(conditions, conditionsv1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo("True"))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
			})
		})
	})
})

type basicExpected struct {
	hco             *hcov1alpha1.HyperConverged
	pc              *schedulingv1.PriorityClass
	kvConfig        *corev1.ConfigMap
	kvStorageConfig *corev1.ConfigMap
	kv              *kubevirtv1.KubeVirt
	cdi             *cdiv1alpha1.CDI
	cna             *networkaddonsv1alpha1.NetworkAddonsConfig
	kvCtb           *sspv1.KubevirtCommonTemplatesBundle
	kvNlb           *sspv1.KubevirtNodeLabellerBundle
	kvTv            *sspv1.KubevirtTemplateValidator
	vmi             *vmimportv1.VMImportConfig
	kvMtAg          *sspv1.KubevirtMetricsAggregation
	imsConfig       *corev1.ConfigMap
}

func (be basicExpected) toArray() []runtime.Object {
	return []runtime.Object{
		be.hco,
		be.pc,
		be.kvConfig,
		be.kvStorageConfig,
		be.kv,
		be.cdi,
		be.cna,
		be.kvCtb,
		be.kvNlb,
		be.kvTv,
		be.vmi,
		be.kvMtAg,
		be.imsConfig,
	}
}

func (be basicExpected) initClient() client.Client {
	return initClient(be.toArray())
}

func getBasicDeployment() *basicExpected {

	res := &basicExpected{}

	hco := &hcov1alpha1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hcov1alpha1.HyperConvergedSpec{},
		Status: hcov1alpha1.HyperConvergedStatus{
			Conditions: []conditionsv1.Condition{
				{
					Type:    hcov1alpha1.ConditionReconcileComplete,
					Status:  corev1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				},
			},
			Versions: hcov1alpha1.Versions{
				{Name: hcoVersionName, Version: version.Version},
			},
		},
	}
	res.hco = hco

	res.pc = newKubeVirtPriorityClass()
	// These are all of the objects that we expect to "find" in the client because
	// we already created them in a previous reconcile.
	expectedKVConfig := newKubeVirtConfigForCR(hco, namespace)
	expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
	res.kvConfig = expectedKVConfig

	expectedKVStorageConfig := newKubeVirtStorageConfigForCR(hco, namespace)
	expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageConfig = expectedKVStorageConfig

	expectedKV := newKubeVirtForCR(hco, namespace)
	expectedKV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/kubevirts/%s", expectedKV.Namespace, expectedKV.Name)
	expectedKV.Status.Conditions = []kubevirtv1.KubeVirtCondition{
		{
			Type:   kubevirtv1.KubeVirtConditionAvailable,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   kubevirtv1.KubeVirtConditionProgressing,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   kubevirtv1.KubeVirtConditionDegraded,
			Status: corev1.ConditionFalse,
		},
	}
	res.kv = expectedKV

	expectedCDI := newCDIForCR(hco, UndefinedNamespace)
	expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
	expectedCDI.Status.Conditions = getGenericCompletedConditions()
	res.cdi = expectedCDI

	expectedCNA := newNetworkAddonsForCR(hco, UndefinedNamespace)
	expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
	expectedCNA.Status.Conditions = getGenericCompletedConditions()
	res.cna = expectedCNA

	expectedKVCTB := newKubeVirtCommonTemplateBundleForCR(hco, OpenshiftNamespace)
	expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
	expectedKVCTB.Status.Conditions = getGenericCompletedConditions()
	res.kvCtb = expectedKVCTB

	expectedKVNLB := newKubeVirtNodeLabellerBundleForCR(hco, namespace)
	expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
	expectedKVNLB.Status.Conditions = getGenericCompletedConditions()
	res.kvNlb = expectedKVNLB

	expectedKVTV := newKubeVirtTemplateValidatorForCR(hco, namespace)
	expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
	expectedKVTV.Status.Conditions = getGenericCompletedConditions()
	res.kvTv = expectedKVTV

	expectedVMI := newVMImportForCR(hco, namespace)
	expectedVMI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedVMI.Namespace, expectedVMI.Name)
	expectedVMI.Status.Conditions = getGenericCompletedConditions()
	res.vmi = expectedVMI

	kvMtAg := newKubeVirtMetricsAggregationForCR(hco, namespace)
	kvMtAg.Status.Conditions = getGenericCompletedConditions()
	res.kvMtAg = kvMtAg

	res.imsConfig = newIMSConfigForCR(hco, namespace)

	return res
}

func checkAvailability(hco *hcov1alpha1.HyperConverged, expected corev1.ConditionStatus) {
	found := false
	for _, cond := range hco.Status.Conditions {
		if cond.Type == conditionsv1.ConditionType(kubevirtv1.KubeVirtConditionAvailable) {
			found = true
			Expect(cond.Status).To(Equal(expected))
			break
		}
	}

	if !found {
		Fail(fmt.Sprintf(`Can't find 'Available' condition; %v`, hco.Status.Conditions))
	}
}

// returns the HCO after reconcile, and the returned requeue
func doReconcile(cl client.Client, hco *hcov1alpha1.HyperConverged) (*hcov1alpha1.HyperConverged, bool) {
	r := initReconciler(cl)

	r.ownVersion = os.Getenv(util.HcoKvIoVersionName)
	if r.ownVersion == "" {
		r.ownVersion = version.Version
	}

	res, err := r.Reconcile(request)
	Expect(err).To(BeNil())

	foundResource := &hcov1alpha1.HyperConverged{}
	Expect(
		cl.Get(context.TODO(),
			types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
			foundResource),
	).To(BeNil())

	return foundResource, res.Requeue
}

func getGenericCompletedConditions() []conditionsv1.Condition {
	return []conditionsv1.Condition{
		{
			Type:   conditionsv1.ConditionAvailable,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   conditionsv1.ConditionProgressing,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   conditionsv1.ConditionDegraded,
			Status: corev1.ConditionFalse,
		},
	}
}

func getGenericProgressingConditions() []conditionsv1.Condition {
	return []conditionsv1.Condition{
		{
			Type:   conditionsv1.ConditionAvailable,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   conditionsv1.ConditionProgressing,
			Status: corev1.ConditionTrue,
		},
		{
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
		vmimportv1.AddToScheme,
	} {
		Expect(f(s)).To(BeNil())
	}

	// Create a ReconcileHyperConverged object with the scheme and fake client
	return &ReconcileHyperConverged{client: client, scheme: s}
}
