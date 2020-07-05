package hyperconverged

import (
	"os"

	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddonsv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	vmimportv1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1alpha1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	"context"
	"fmt"

	"k8s.io/client-go/tools/reference"
)

var _ = Describe("HyperConverged Components", func() {

	Context("KubeVirt Priority Classes", func() {

		var hco *hcov1alpha1.HyperConverged
		var req *hcoRequest

		BeforeEach(func() {
			hco = newHco()
			req = newReq(hco)
		})

		It("should create if not present", func() {
			expectedResource := hco.NewKubeVirtPriorityClass()
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
			expectedResource := hco.NewKubeVirtPriorityClass()
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

			expectedResource := hco.NewKubeVirtPriorityClass()
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
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
			expectedResource := hco.NewKubeVirt(namespace)
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewKubeVirt(namespace)
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
			expectedResource := hco.NewKubeVirt(namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			missingUSResource := hco.NewKubeVirt(namespace)
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
			expectedResource := hco.NewKubeVirt(namespace)
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
			expectedResource := hco.NewCDI()
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewCDI()
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
			expectedResource := hco.NewCDI(namespace)
			expectedResource.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/dummies/%s", expectedResource.Namespace, expectedResource.Name)
			missingUSResource := hco.NewCDI(namespace)
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
			expectedResource := hco.NewCDI()
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
			expectedResource := hco.NewNetworkAddons()
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
			Expect(foundResource.Spec.Multus).To(Equal(&networkaddonsv1alpha1.Multus{}))
			Expect(foundResource.Spec.LinuxBridge).To(Equal(&networkaddonsv1alpha1.LinuxBridge{}))
			Expect(foundResource.Spec.KubeMacPool).To(Equal(&networkaddonsv1alpha1.KubeMacPool{}))
		})

		It("should find if present", func() {
			expectedResource := hco.NewNetworkAddons()
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
			expectedResource := hco.NewKubeVirtCommonTemplateBundle()
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
			Expect(foundResource.Namespace).To(Equal(expectedResource.Namespace))
		})

		It("should find if present", func() {
			expectedResource := hco.NewKubeVirtCommonTemplateBundle()
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
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
			Expect(foundResource.Labels).Should(HaveKeyWithValue(hcoutil.AppLabel, name))
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
