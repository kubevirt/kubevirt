package hyperconverged

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	"os"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubevirt/hyperconverged-cluster-operator/version"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	k8sTime "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	// TODO: Move to envtest to get an actual api server
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/openshift/custom-resource-status/testlib"

	// networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
)

// name and namespace of our primary resource
const (
	name      = "kubevirt-hyperconverged"
	namespace = "kubevirt-hyperconverged"
)

var _ = Describe("HyperconvergedController", func() {

	Describe("Reconcile HyperConverged", func() {
		Context("HCO Lifecycle", func() {

			BeforeEach(func() {
				os.Setenv("CONVERSION_CONTAINER", commonTestUtils.Conversion_image)
				os.Setenv("VMWARE_CONTAINER", commonTestUtils.Vmware_image)
				os.Setenv("OPERATOR_NAMESPACE", namespace)
			})

			It("should handle not found", func() {
				cl := commonTestUtils.InitClient([]runtime.Object{})
				r := initReconciler(cl)

				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))
			})

			It("should ignore invalid requests", func() {
				hco := &hcov1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid",
						Namespace: "invalid",
					},
					Spec: hcov1beta1.HyperConvergedSpec{},
					Status: hcov1beta1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{},
					},
				}
				cl := commonTestUtils.InitClient([]runtime.Object{hco})
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
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  corev1.ConditionFalse,
					Reason:  invalidRequestReason,
					Message: fmt.Sprintf(invalidRequestMessageFormat, name, namespace),
				})))
			})

			It("should create all managed resources", func() {
				hco := commonTestUtils.NewHco()
				cl := commonTestUtils.InitClient([]runtime.Object{hco})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
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
				hco := &hcov1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1beta1.HyperConvergedSpec{},
					Status: hcov1beta1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{
							conditionsv1.Condition{
								Type:    hcov1beta1.ConditionReconcileComplete,
								Status:  corev1.ConditionTrue,
								Reason:  reconcileCompleted,
								Message: reconcileCompletedMessage,
							},
						},
					},
				}
				// These are all of the objects that we expect to "find" in the client because
				// we already created them in a previous reconcile.
				expectedKVConfig := operands.NewKubeVirtConfigForCR(hco, namespace)
				expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
				expectedKVStorageConfig := operands.NewKubeVirtStorageConfigForCR(hco, namespace)
				expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
				expectedKVStorageRole := operands.NewKubeVirtStorageRoleForCR(hco, namespace)
				expectedKVStorageRole.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/roles/%s", expectedKVStorageRole.Namespace, expectedKVStorageRole.Name)
				expectedKVStorageRoleBinding := operands.NewKubeVirtStorageRoleBindingForCR(hco, namespace)
				expectedKVStorageRoleBinding.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/rolebindings/%s", expectedKVStorageRoleBinding.Namespace, expectedKVStorageRoleBinding.Name)
				expectedKV := hco.NewKubeVirt(namespace)
				expectedKV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/kubevirts/%s", expectedKV.Namespace, expectedKV.Name)
				expectedCDI := hco.NewCDI()
				expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
				expectedCNA := hco.NewNetworkAddons()
				expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
				expectedKVCTB := hco.NewKubeVirtCommonTemplateBundle()
				expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
				expectedKVNLB := operands.NewKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
				expectedKVTV := operands.NewKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
				// Add all of the objects to the client
				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedKVConfig, expectedKVStorageConfig, expectedKVStorageRole, expectedKVStorageRoleBinding, expectedKV, expectedCDI, expectedCNA, expectedKVCTB, expectedKVNLB, expectedKVTV})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
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
				hco := &hcov1beta1.HyperConverged{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: hcov1beta1.HyperConvergedSpec{},
					Status: hcov1beta1.HyperConvergedStatus{
						Conditions: []conditionsv1.Condition{
							conditionsv1.Condition{
								Type:    hcov1beta1.ConditionReconcileComplete,
								Status:  corev1.ConditionTrue,
								Reason:  reconcileCompleted,
								Message: reconcileCompletedMessage,
							},
						},
					},
				}
				// These are all of the objects that we expect to "find" in the client because
				// we already created them in a previous reconcile.
				expectedKVConfig := operands.NewKubeVirtConfigForCR(hco, namespace)
				expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
				expectedKVStorageConfig := operands.NewKubeVirtStorageConfigForCR(hco, namespace)
				expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
				expectedKVStorageRole := operands.NewKubeVirtStorageRoleForCR(hco, namespace)
				expectedKVStorageRole.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/role/%s", expectedKVStorageRole.Namespace, expectedKVStorageRole.Name)
				expectedKVStorageRoleBinding := operands.NewKubeVirtStorageRoleBindingForCR(hco, namespace)
				expectedKVStorageRoleBinding.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/role/%s", expectedKVStorageRoleBinding.Namespace, expectedKVStorageRoleBinding.Name)
				expectedKV := hco.NewKubeVirt(namespace)
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
				expectedCDI := hco.NewCDI()
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
				expectedCNA := hco.NewNetworkAddons()
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
				expectedKVCTB := hco.NewKubeVirtCommonTemplateBundle()
				expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
				expectedKVCTB.Status.Conditions = getGenericCompletedConditions()
				expectedKVNLB := operands.NewKubeVirtNodeLabellerBundleForCR(hco, namespace)
				expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
				expectedKVNLB.Status.Conditions = getGenericCompletedConditions()
				expectedKVTV := operands.NewKubeVirtTemplateValidatorForCR(hco, namespace)
				expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
				expectedKVTV.Status.Conditions = getGenericCompletedConditions()
				// Add all of the objects to the client
				cl := commonTestUtils.InitClient([]runtime.Object{hco, expectedKVConfig, expectedKVStorageConfig, expectedKV, expectedCDI, expectedCNA, expectedKVCTB, expectedKVNLB, expectedKVTV})
				r := initReconciler(cl)

				// Do the reconcile
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(testlib.RepresentCondition(conditionsv1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
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
			})

			It(`should delete HCO`, func() {

				// First, create HCO and check it
				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl)
				res, err := r.Reconcile(request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(len(foundResource.Status.RelatedObjects)).Should(Equal(15))
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

				foundResource = &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).To(BeNil())
				Expect(foundResource.ObjectMeta.Finalizers).To(BeNil())

			})
		})

		Context("Validate OLM required fields", func() {
			expected := getBasicDeployment()
			origConds := expected.hco.Status.Conditions

			BeforeEach(func() {
				os.Setenv("CONVERSION_CONTAINER", commonTestUtils.Conversion_image)
				os.Setenv("VMWARE_CONTAINER", commonTestUtils.Vmware_image)
				os.Setenv("OPERATOR_NAMESPACE", namespace)
				os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
			})

			It("Should set required fields on init", func() {
				expected.hco.Status.Conditions = nil

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeTrue())

				Expect(foundResource.ObjectMeta.Labels[hcoutil.AppLabel]).Should(Equal(hcoutil.HyperConvergedName))
			})

			It("Should set required fields when missing", func() {
				expected.hco.Status.Conditions = origConds
				// old HCO Version is set
				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())

				Expect(foundResource.ObjectMeta.Labels[hcoutil.AppLabel]).Should(Equal(hcoutil.HyperConvergedName))
			})
		})

		Context("Upgrade Mode", func() {
			expected := getBasicDeployment()
			origConditions := expected.hco.Status.Conditions
			okConds := expected.hco.Status.Conditions

			const (
				oldVersion          = "1.2.0"
				newVersion          = "1.3.0" // TODO: avoid hard-coding values
				oldComponentVersion = "1.3.0"
				newComponentVersion = "1.3.3"
			)

			BeforeEach(func() {
				os.Setenv("CONVERSION_CONTAINER", commonTestUtils.Conversion_image)
				os.Setenv("VMWARE_CONTAINER", commonTestUtils.Vmware_image)
				os.Setenv("OPERATOR_NAMESPACE", namespace)

				expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
				os.Setenv(hcoutil.KubevirtVersionEnvV, newComponentVersion)

				expected.cdi.Status.ObservedVersion = newComponentVersion
				os.Setenv(hcoutil.CdiVersionEnvV, newComponentVersion)

				expected.cna.Status.ObservedVersion = newComponentVersion
				os.Setenv(hcoutil.CnaoVersionEnvV, newComponentVersion)

				expected.vmi.Status.ObservedVersion = newComponentVersion
				os.Setenv(hcoutil.VMImportEnvV, newComponentVersion)

				os.Setenv(hcoutil.SspVersionEnvV, newComponentVersion)
				expected.kvCtb.Status.ObservedVersion = newComponentVersion
				expected.kvNlb.Status.ObservedVersion = newComponentVersion
				expected.kvTv.Status.ObservedVersion = newComponentVersion
				expected.kvMtAg.Status.ObservedVersion = newComponentVersion

				os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

				expected.hco.Status.Conditions = origConditions
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

				Expect(foundResource.Spec.Version).Should(Equal(newVersion))

				expected.hco.Status.Conditions = okConds
			})

			It("detect upgrade existing HCO Version", func() {
				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)
				expected.hco.Spec.Version = oldVersion

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

				Expect(foundResource.Spec.Version).Should(Equal(oldVersion))

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
				Expect(foundResource.Spec.Version).Should(Equal(newVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
			})

			It("detect upgrade w/o HCO Version", func() {
				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()
				expected.hco.Status.Versions = nil
				expected.hco.Spec.Version = ""

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				fmt.Fprintln(GinkgoWriter, "foundResource.Status.Versions", foundResource.Status.Versions)
				Expect(ok).To(BeFalse())
				Expect(ver).Should(BeEmpty())
				Expect(foundResource.Spec.Version).To(BeEmpty())

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
				Expect(foundResource.Spec.Version).Should(Equal(newVersion))

				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("False"))
			})

			It("don't complete upgrade if kubevirt version is not match to the kubevirt version env ver", func() {
				os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

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

				hcoReady, err := checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeFalse())

				// check that the upgrade is not done if the not all the versions are match.
				// Conditions are valid
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				cl = expected.initClient()
				foundResource, requeue = doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionTrue)

				// check that the image Id is set, now, when upgrade is completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond = conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				hcoReady, err = checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeFalse())

				// now, complete the upgrade
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

				hcoReady, err = checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeTrue())
			})

			It("don't complete upgrade if CDI version is not match to the CDI version env ver", func() {
				os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

				// old HCO Version is set
				expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

				expected.cdi.Status.ObservedVersion = oldComponentVersion

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, requeue := doReconcile(cl, expected.hco)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, corev1.ConditionFalse)

				hcoReady, err := checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeFalse())

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := conditionsv1.FindStatusCondition(foundResource.Status.Conditions, conditionsv1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo("True"))
				Expect(cond.Reason).Should(Equal("HCOUpgrading"))
				Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

				hcoReady, err = checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeFalse())

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

				hcoReady, err = checkHcoReady()
				Expect(err).To(BeNil())
				Expect(hcoReady).To(BeTrue())

			})

			It("don't complete upgrade if CNA version is not match to the CNA version env ver", func() {
				os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

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
				os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

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
			os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

				cd := conditionsv1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
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

		Context("Update Conflict Error", func() {
			It("Should requeue in case of update conflict", func() {
				expected := getBasicDeployment()
				expected.hco.Status.Conditions = nil
				cl := expected.initClient()
				rsc := schema.GroupResource{Group: hcoutil.APIVersionGroup, Resource: "hyperconvergeds.hco.kubevirt.io"}
				cl.InitiateWriteErrors(
					nil,
					apierrors.NewConflict(rsc, "hco", errors.New("test error")),
				)
				r := initReconciler(cl)

				r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
				if r.ownVersion == "" {
					r.ownVersion = version.Version
				}

				res, err := r.Reconcile(request)

				Expect(err).ToNot(BeNil())
				Expect(apierrors.IsConflict(err)).To(BeTrue())
				Expect(res.Requeue).To(BeTrue())
			})

			It("Should requeue in case of update status conflict", func() {
				expected := getBasicDeployment()
				expected.hco.Status.Conditions = nil
				cl := expected.initClient()
				rs := schema.GroupResource{hcoutil.APIVersionGroup, "hyperconvergeds.hco.kubevirt.io"}
				cl.Status().(*commonTestUtils.HcoTestStatusWriter).InitiateErrors(apierrors.NewConflict(rs, "hco", errors.New("test error")))
				r := initReconciler(cl)

				r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
				if r.ownVersion == "" {
					r.ownVersion = version.Version
				}

				res, err := r.Reconcile(request)

				Expect(err).ToNot(BeNil())
				Expect(apierrors.IsConflict(err)).To(BeTrue())
				Expect(res.Requeue).To(BeTrue())

			})
		})
	})
})
