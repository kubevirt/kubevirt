package hyperconverged

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/util/yaml"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	v1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	k8sTime "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	// TODO: Move to envtest to get an actual api server
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
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
				_ = os.Setenv("CONVERSION_CONTAINER", commonTestUtils.ConversionImage)
				_ = os.Setenv("VMWARE_CONTAINER", commonTestUtils.VmwareImage)
				_ = os.Setenv("VIRTIOWIN_CONTAINER", commonTestUtils.VirtioWinImage)
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)
				_ = os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
			})

			It("should handle not found", func() {
				cl := commonTestUtils.InitClient([]runtime.Object{})
				r := initReconciler(cl, nil)

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
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
						Conditions: []metav1.Condition{},
					},
				}
				cl := commonTestUtils.InitClient([]runtime.Object{hco})
				r := initReconciler(cl, nil)

				// Do the reconcile
				var invalidRequest = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "invalid",
						Namespace: "invalid",
					},
				}
				res, err := r.Reconcile(context.TODO(), invalidRequest)
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
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionFalse,
					Reason:  invalidRequestReason,
					Message: fmt.Sprintf(invalidRequestMessageFormat, name, namespace),
				})))
			})

			It("should create all managed resources", func() {
				hco := commonTestUtils.NewHco()
				hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
					WithHostPassthroughCPU: true,
				}

				cl := commonTestUtils.InitClient([]runtime.Object{hco})
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
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
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: reconcileInitMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionAvailable,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionDegraded,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionUpgradeable,
					Status:  metav1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))

				// Get the KV
				kvList := &kubevirtv1.KubeVirtList{}
				Expect(cl.List(context.TODO(), kvList)).To(BeNil())
				Expect(kvList.Items).Should(HaveLen(1))
				kv := kvList.Items[0]
				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(12))

				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(
					"DataVolumes",
					"SRIOV",
					"LiveMigration",
					"CPUManager",
					"CPUNodeDiscovery",
					"Snapshot",
					"HotplugVolumes",
					"GPU",
					"HostDevices",
					"WithHostModelCPU",
					"HypervStrictCheck"),
				)
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElement("WithHostPassthroughCPU"))
			})

			It("should find all managed resources", func() {

				expected := getBasicDeployment()

				expected.kv.Status.Conditions = nil
				expected.cdi.Status.Conditions = nil
				expected.cna.Status.Conditions = nil
				expected.ssp.Status.Conditions = nil
				expected.vmi.Status.Conditions = nil
				cl := expected.initClient()

				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				// Why SSP? Because it is the last to be checked, so the last missing overwrites everything
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionAvailable,
					Status:  metav1.ConditionFalse,
					Reason:  "SSPConditions",
					Message: "SSP resource has no conditions",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  "SSPConditions",
					Message: "SSP resource has no conditions",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionUpgradeable,
					Status:  metav1.ConditionFalse,
					Reason:  "SSPConditions",
					Message: "SSP resource has no conditions",
				})))
			})

			It("should label all managed resources", func() {
				expected := getBasicDeployment()

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				// Check whether related objects have the labels or not
				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				for _, relatedObj := range foundResource.Status.RelatedObjects {
					foundRelatedObj := &unstructured.Unstructured{}
					foundRelatedObj.SetGroupVersionKind(relatedObj.GetObjectKind().GroupVersionKind())
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: relatedObj.Name, Namespace: relatedObj.Namespace},
							foundRelatedObj),
					).To(BeNil())

					foundLabels := foundRelatedObj.GetLabels()
					Expect(foundLabels[hcoutil.AppLabel]).Should(Equal(expected.hco.Name))
					Expect(foundLabels[hcoutil.AppLabelPartOf]).Should(Equal(hcoutil.HyperConvergedCluster))
					Expect(foundLabels[hcoutil.AppLabelManagedBy]).Should(Equal(hcoutil.OperatorName))
					Expect(foundLabels[hcoutil.AppLabelVersion]).Should(Equal(version.Version))
					Expect(foundLabels[hcoutil.AppLabelComponent]).ShouldNot(BeNil())
				}
			})

			It("should set different template namespace to ssp CR", func() {
				expected := getBasicDeployment()
				expected.hco.Spec.CommonTemplatesNamespace = &expected.hco.Namespace

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				foundResource := &sspv1beta1.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.ssp.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Spec.CommonTemplates.Namespace).To(Equal(expected.hco.Namespace), "common-templates namespace should be "+expected.hco.Namespace)
			})
			It("should complete when components are finished", func() {
				expected := getBasicDeployment()

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionAvailable,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionProgressing,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionDegraded,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionUpgradeable,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
			})

			It("should increment counter when out-of-band change overwritten", func() {
				hco := commonTestUtils.NewHco()
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				existingResource, err := operands.NewKubeVirt(hco, namespace)
				Expect(err).ToNot(HaveOccurred())
				existingResource.Kind = kubevirtv1.KubeVirtGroupVersionKind.Kind // necessary for metrics

				// now, modify KV's node placement
				seconds3 := int64(3)
				existingResource.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})
				existingResource.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})

				existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"] = "BADvalue2"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				r := initReconciler(cl, nil)

				// mock a reconciliation triggered by a change in secondary CR
				ph, err := getSecondaryCRPlaceholder()
				Expect(err).To(BeNil())
				rq := request
				rq.NamespacedName = ph

				counterValueBefore, err := metrics.HcoMetrics.GetOverwrittenModificationsCount(existingResource.Kind, existingResource.Name)
				Expect(err).To(BeNil())

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), rq)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				foundResource := &kubevirtv1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).Should(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).Should(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).Should(Equal("value2"))

				counterValueAfter, err := metrics.HcoMetrics.GetOverwrittenModificationsCount(foundResource.Kind, foundResource.Name)
				Expect(err).To(BeNil())
				Expect(counterValueAfter).To(Equal(counterValueBefore + 1))

			})

			It("should not increment counter when CR was changed by HCO", func() {
				hco := commonTestUtils.NewHco()
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commonTestUtils.NewNodePlacement()}
				existingResource, err := operands.NewKubeVirt(hco, namespace)
				Expect(err).ToNot(HaveOccurred())
				existingResource.Kind = kubevirtv1.KubeVirtGroupVersionKind.Kind // necessary for metrics

				// now, modify KV's node placement
				seconds3 := int64(3)
				existingResource.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})
				existingResource.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: &seconds3,
				})

				existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"] = "BADvalue2"

				cl := commonTestUtils.InitClient([]runtime.Object{hco, existingResource})
				r := initReconciler(cl, nil)

				counterValueBefore, err := metrics.HcoMetrics.GetOverwrittenModificationsCount(existingResource.Kind, existingResource.Name)
				Expect(err).To(BeNil())

				// Do the reconcile triggered by HCO
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				foundResource := &kubevirtv1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(BeNil())

				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).Should(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).Should(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).Should(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).Should(Equal("value2"))

				counterValueAfter, err := metrics.HcoMetrics.GetOverwrittenModificationsCount(foundResource.Kind, foundResource.Name)
				Expect(err).To(BeNil())
				Expect(counterValueAfter).To(Equal(counterValueBefore))

			})

			It(`should be not available when components with missing "Available" condition`, func() {
				expected := getBasicDeployment()

				var cl *commonTestUtils.HcoTestClient
				By("Check KV", func() {
					origKvConds := expected.kv.Status.Conditions
					expected.kv.Status.Conditions = expected.kv.Status.Conditions[1:]

					cl = expected.initClient()
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionFalse)

					expected.kv.Status.Conditions = origKvConds
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)
				})

				By("Check CDI", func() {
					origConds := expected.cdi.Status.Conditions
					expected.cdi.Status.Conditions = expected.cdi.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionFalse)

					expected.cdi.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)
				})

				By("Check CNA", func() {
					origConds := expected.cna.Status.Conditions

					expected.cna.Status.Conditions = expected.cna.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionFalse)

					expected.cna.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)
				})
				By("Check SSP", func() {
					origConds := expected.ssp.Status.Conditions
					expected.ssp.Status.Conditions = expected.ssp.Status.Conditions[1:]
					cl = expected.initClient()
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionFalse)

					expected.ssp.Status.Conditions = origConds
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)
				})
			})

			It(`should delete HCO`, func() {

				// First, create HCO and check it
				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl, nil)
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(len(foundResource.Status.RelatedObjects)).Should(Equal(14))
				Expect(foundResource.ObjectMeta.Finalizers).Should(Equal([]string{FinalizerName}))

				// Now, delete HCO
				delTime := time.Now().UTC().Add(-1 * time.Minute)
				expected.hco.ObjectMeta.DeletionTimestamp = &k8sTime.Time{Time: delTime}
				expected.hco.ObjectMeta.Finalizers = []string{FinalizerName}
				cl = expected.initClient()

				r = initReconciler(cl, nil)
				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: false}))

				foundResource = &hcov1beta1.HyperConverged{}
				err = cl.Get(context.TODO(),
					types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
					foundResource)
				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It(`should set a finalizer on HCO CR`, func() {
				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl, nil)
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(foundResource.ObjectMeta.Finalizers).Should(Equal([]string{FinalizerName}))
			})

			It(`should replace a finalizer with a bad name if there`, func() {
				expected := getBasicDeployment()
				expected.hco.ObjectMeta.Finalizers = []string{badFinalizerName}
				cl := expected.initClient()
				r := initReconciler(cl, nil)
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(BeNil())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(foundResource.ObjectMeta.Finalizers).Should(Equal([]string{FinalizerName}))
			})

			It("Should not be ready if one of the operands is returns error, on create", func() {
				hcoutil.SetReady(true)
				Expect(checkHcoReady()).To(BeTrue())
				hco := commonTestUtils.NewHco()
				cl := commonTestUtils.InitClient([]runtime.Object{hco})
				cl.InitiateCreateErrors(func(obj client.Object) error {
					if _, ok := obj.(*cdiv1beta1.CDI); ok {
						return errors.New("fake create error")
					}
					return nil
				})
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())

				// Check condition
				foundCond := false
				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionReconcileComplete {
						foundCond = true
						Expect(cond.Status).Should(Equal(metav1.ConditionFalse))
						Expect(cond.Message).Should(ContainSubstring("fake create error"))
						break
					}
				}
				Expect(foundCond).To(BeTrue())

				Expect(checkHcoReady()).To(BeFalse())
			})

			It("Should not be ready if one of the operands is returns error, on update", func() {
				expected := getBasicDeployment()
				expected.kv.Spec.Configuration.DeveloperConfiguration = &kubevirtv1.DeveloperConfiguration{
					FeatureGates: []string{"fakeFg"}, // force update
				}
				cl := expected.initClient()
				cl.InitiateUpdateErrors(func(obj client.Object) error {
					if _, ok := obj.(*kubevirtv1.KubeVirt); ok {
						return errors.New("fake update error")
					}
					return nil
				})

				hcoutil.SetReady(true)
				Expect(checkHcoReady()).To(BeTrue())

				hco := commonTestUtils.NewHco()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(BeNil())
				Expect(res).Should(Equal(reconcile.Result{Requeue: false}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(BeNil())

				// Check condition
				foundCond := false
				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionReconcileComplete {
						foundCond = true
						Expect(cond.Status).Should(Equal(metav1.ConditionFalse))
						Expect(cond.Message).Should(ContainSubstring("fake update error"))
						break
					}
				}
				Expect(foundCond).To(BeTrue())

				Expect(checkHcoReady()).To(BeFalse())
			})

			It("should remove the kubevirt-config CM not in upgrade", func() {
				cm := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      kvCmName,
						Namespace: commonTestUtils.Namespace,
					},
					Data: map[string]string{
						"fakeKey": "fakeValue",
					},
				}
				expected := getBasicDeployment()
				cl := commonTestUtils.InitClient(append(expected.toArray(), cm))

				By("Make sure the CM is there before starting", func() {
					res := &corev1.ConfigMap{}
					err := cl.Get(context.TODO(),
						types.NamespacedName{Name: kvCmName, Namespace: namespace},
						res)

					Expect(err).ToNot(HaveOccurred())
					Expect(res.Data["fakeKey"]).Should(Equal("fakeValue"))
				})

				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionTrue)

				By("should not find the configMap")
				res := &corev1.ConfigMap{}
				err := cl.Get(context.TODO(),
					types.NamespacedName{Name: kvCmName, Namespace: namespace},
					res)

				Expect(apierrors.IsNotFound(err)).To(BeTrue())

				By("should not find backup in this case")
				err = cl.Get(context.TODO(),
					types.NamespacedName{Name: backupKvCmName, Namespace: namespace},
					foundResource)

				Expect(apierrors.IsNotFound(err)).To(BeTrue())
			})

			It("Should upgrade the status.observedGeneration field", func() {
				expected := getBasicDeployment()
				expected.hco.ObjectMeta.Generation = 10
				cl := expected.initClient()
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				Expect(foundResource.Status.ObservedGeneration).Should(BeEquivalentTo(10))
			})
		})

		Context("Validate OLM required fields", func() {
			var (
				expected  *BasicExpected
				origConds []metav1.Condition
			)

			BeforeEach(func() {
				_ = os.Setenv("CONVERSION_CONTAINER", commonTestUtils.ConversionImage)
				_ = os.Setenv("VMWARE_CONTAINER", commonTestUtils.VmwareImage)
				_ = os.Setenv("VIRTIOWIN_CONTAINER", commonTestUtils.VirtioWinImage)
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)
				_ = os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)

				expected = getBasicDeployment()
				origConds = expected.hco.Status.Conditions
			})

			It("Should set required fields on init", func() {
				expected.hco.Status.Conditions = nil

				cl := expected.initClient()
				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())

				Expect(foundResource.ObjectMeta.Labels[hcoutil.AppLabel]).Should(Equal(hcoutil.HyperConvergedName))
			})

			It("Should set required fields when missing", func() {
				expected.hco.Status.Conditions = origConds
				// old HCO Version is set
				cl := expected.initClient()
				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeFalse())

				Expect(foundResource.ObjectMeta.Labels[hcoutil.AppLabel]).Should(Equal(hcoutil.HyperConvergedName))
			})
		})

		Context("Upgrade Mode", func() {
			const (
				oldVersion          = "1.4.0"
				newVersion          = "1.5.0" // TODO: avoid hard-coding values
				oldComponentVersion = "1.5.0"
				newComponentVersion = "1.5.3"
			)

			// this is used for version label and the tests below
			// assumes there is no change in labels. Therefore, it should be
			// set before getBasicDeployment so that the existing resource can
			// have the correct labels
			_ = os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

			var (
				expected       *BasicExpected
				origConditions []metav1.Condition
				okConds        []metav1.Condition
			)

			BeforeEach(func() {
				expected = getBasicDeployment()
				origConditions = expected.hco.Status.Conditions
				okConds = expected.hco.Status.Conditions
			})

			BeforeEach(func() {
				_ = os.Setenv("CONVERSION_CONTAINER", commonTestUtils.ConversionImage)
				_ = os.Setenv("VMWARE_CONTAINER", commonTestUtils.VmwareImage)
				_ = os.Setenv("VIRTIOWIN_CONTAINER", commonTestUtils.VirtioWinImage)
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)

				expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
				_ = os.Setenv(hcoutil.KubevirtVersionEnvV, newComponentVersion)

				expected.cdi.Status.ObservedVersion = newComponentVersion
				_ = os.Setenv(hcoutil.CdiVersionEnvV, newComponentVersion)

				expected.cna.Status.ObservedVersion = newComponentVersion
				_ = os.Setenv(hcoutil.CnaoVersionEnvV, newComponentVersion)

				expected.vmi.Status.ObservedVersion = newComponentVersion
				_ = os.Setenv(hcoutil.VMImportEnvV, newComponentVersion)

				_ = os.Setenv(hcoutil.SspVersionEnvV, newComponentVersion)
				expected.ssp.Status.ObservedVersion = newComponentVersion

				expected.hco.Status.Conditions = origConditions
			})

			It("Should update HCO Version Id in the CR on init", func() {

				expected.hco.Status.Conditions = nil

				cl := expected.initClient()
				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionFalse)

				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionAvailable {
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
				foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionFalse)
				// check that the HCO version is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))

				// Call again - requeue
				foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionFalse)

				// check that the HCO version is not set, because upgrade is not completed
				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				cl = expected.initClient()
				foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionTrue)

				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(oldVersion))
				cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionTrue))

				// Call again, to start complete the upgrade
				// check that the image Id is set, now, when upgrade is completed
				foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionTrue)

				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))
				cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
			})

			It("detect upgrade w/o HCO Version", func() {
				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()
				expected.hco.Status.Versions = nil

				cl := expected.initClient()
				foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionFalse)

				expected.hco = foundResource
				cl = expected.initClient()
				foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionFalse)

				// check that the image Id is not set, because upgrade is not completed
				ver, ok := foundResource.Status.GetVersion(hcoVersionName)
				_, _ = fmt.Fprintln(GinkgoWriter, "foundResource.Status.Versions", foundResource.Status.Versions)
				Expect(ok).To(BeFalse())
				Expect(ver).Should(BeEmpty())

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.hco = foundResource
				cl = expected.initClient()
				foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionTrue)

				_, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionFalse))

				ver, ok = foundResource.Status.GetVersion(hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).Should(Equal(newVersion))

				cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
			})

			DescribeTable(
				"don't complete upgrade if a component version is not match to the component's version env ver",
				func(makeComponentNotReady, makeComponentReady, updateComponentVersion func()) {
					_ = os.Setenv(hcoutil.HcoKvIoVersionName, newVersion)

					// old HCO Version is set
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					makeComponentNotReady()

					cl := expected.initClient()
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionFalse)

					expected.hco = foundResource
					cl = expected.initClient()
					foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionFalse)

					// check that the image Id is not set, because upgrade is not completed
					ver, ok := foundResource.Status.GetVersion(hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).Should(Equal(oldVersion))
					cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
					Expect(cond.Reason).Should(Equal("HCOUpgrading"))
					Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

					hcoReady := checkHcoReady()
					Expect(hcoReady).To(BeFalse())

					// check that the upgrade is not done if the not all the versions are match.
					// Conditions are valid
					makeComponentReady()

					expected.hco = foundResource
					cl = expected.initClient()
					foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					// check that the image Id is set, now, when upgrade is completed
					ver, ok = foundResource.Status.GetVersion(hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).Should(Equal(oldVersion))
					cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
					Expect(cond.Reason).Should(Equal("HCOUpgrading"))
					Expect(cond.Message).Should(Equal("HCO is now upgrading to version " + newVersion))

					hcoReady = checkHcoReady()
					Expect(hcoReady).To(BeFalse())

					// now, complete the upgrade
					updateComponentVersion()

					expected.hco = foundResource
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					// check that the image Id is set, now, when upgrade is completed
					ver, ok = foundResource.Status.GetVersion(hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).Should(Equal(newVersion))
					cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
					Expect(cond.Reason).Should(Equal("ReconcileCompleted"))

					hcoReady = checkHcoReady()
					Expect(hcoReady).To(BeTrue())
				},
				Entry(
					"don't complete upgrade if kubevirt version is not match to the kubevirt version env ver",
					func() {
						expected.kv.Status.ObservedKubeVirtVersion = oldComponentVersion
						expected.kv.Status.Conditions[0].Status = "False"
					},
					func() {
						expected.kv.Status.Conditions[0].Status = "True"
					},
					func() {
						expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
					},
				),
				Entry(
					"don't complete upgrade if CDI version is not match to the CDI version env ver",
					func() {
						expected.cdi.Status.ObservedVersion = oldComponentVersion
						// CDI is not ready
						expected.cdi.Status.Conditions = getGenericProgressingConditions()
					},
					func() {
						// CDI is now ready
						expected.cdi.Status.Conditions = getGenericCompletedConditions()
					},
					func() {
						expected.cdi.Status.ObservedVersion = newComponentVersion
					},
				),
				Entry(
					"don't complete upgrade if CNA version is not match to the CNA version env ver",
					func() {
						expected.cna.Status.ObservedVersion = oldComponentVersion
						// CNA is not ready
						expected.cna.Status.Conditions = getGenericProgressingConditions()
					},
					func() {
						// CNA is now ready
						expected.cna.Status.Conditions = getGenericCompletedConditions()
					},
					func() {
						expected.cna.Status.ObservedVersion = newComponentVersion
					},
				),
				Entry(
					"don't complete upgrade if VM-Import version is not match to the VM-Import version env ver",
					func() {
						expected.vmi.Status.ObservedVersion = ""
						// VM-Import is not ready
						expected.vmi.Status.Conditions = getGenericProgressingConditions()
					},
					func() {
						// VM-Import is now ready
						expected.vmi.Status.Conditions = getGenericCompletedConditions()
					},
					func() {
						expected.vmi.Status.ObservedVersion = newComponentVersion
					},
				),
			)

			Context("Drop KubeVirt configMap", func() {
				bandwidthPerMigration := "64Mi"
				completionTimeoutPerGiB := int64(800)
				parallelMigrationsPerCluster := uint32(5)
				parallelOutboundMigrationsPerNode := uint32(2)
				progressTimeout := int64(150)

				It("should drop KubeVirt configMap and create backup", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					kvCM := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					kvCMRef, err := reference.GetReference(commonTestUtils.GetScheme(), kvCM)
					Expect(err).ToNot(HaveOccurred())
					Expect(v1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvCMRef)).ToNot(HaveOccurred())

					resources := append(expected.toArray(), kvCM)

					cl := commonTestUtils.InitClient(resources)
					foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeFalse())
				})

				It("should adopt KubeVirt configMap into HC CR, drop it and create backup", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					kvCM := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
parallelOutboundMigrationsPerNode: 2
bandwidthPerMigration: 64Mi
completionTimeoutPerGiB: 800
progressTimeout: 150`,
						},
					}
					kvCMRef, err := reference.GetReference(commonTestUtils.GetScheme(), kvCM)
					Expect(err).ToNot(HaveOccurred())
					Expect(v1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvCMRef)).ToNot(HaveOccurred())

					resources := append(expected.toArray(), kvCM)

					cl := commonTestUtils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionUnknown)

					By("Check that the LifeMigrationConfig field contains the configmap values")
					lmc := foundResource.Spec.LiveMigrationConfig
					Expect(*lmc.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*lmc.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*lmc.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*lmc.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*lmc.ProgressTimeout).Should(Equal(progressTimeout))

					By("Check that KV's MigrationConfiguration field does not contain the configmap values, yet")
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).To(BeNil())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeTrue())

					By("Run reconcile again")
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					By("Check that KV's MigrationConfiguration field contains the configmap values")
					kv = operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(*kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).Should(Equal(resource.MustParse(bandwidthPerMigration)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).Should(Equal(progressTimeout))

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeFalse())
				})

				It("should adopt KubeVirt configMap into HC CR if the values are different, drop the cm and create backup", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					kvCM := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 4
parallelOutboundMigrationsPerNode: 4
bandwidthPerMigration: 16Mi
completionTimeoutPerGiB: 400
progressTimeout: 300`,
						},
					}
					kvCMRef, err := reference.GetReference(commonTestUtils.GetScheme(), kvCM)
					Expect(err).ToNot(HaveOccurred())
					Expect(v1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvCMRef)).ToNot(HaveOccurred())
					resources := append(expected.toArray(), kvCM)

					cl := commonTestUtils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionUnknown)

					By("Check that the LifeMigrationConfig field contains the configmap values")
					lmc := foundResource.Spec.LiveMigrationConfig
					Expect(*lmc.BandwidthPerMigration).Should(Equal("16Mi"))
					Expect(*lmc.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*lmc.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*lmc.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(4)))
					Expect(*lmc.ProgressTimeout).Should(Equal(int64(300)))

					By("Check that KV's MigrationConfiguration field does not contain the configmap values, yet")
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).To(BeNil())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeTrue())

					By("Run reconcile again")
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					By("Check that KV's MigrationConfiguration field contains the configmap values")
					kv = operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(*kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).Should(Equal(resource.MustParse("16Mi")))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(4)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).Should(Equal(int64(300)))

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeFalse())
				})

				It("should adopt real KubeVirt configMap into HC CR if the values are different, drop the cm and create backup", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					cmFile, err := os.Open(getTestFilesLocation() + "/kubevirt-config.yaml")
					Expect(err).ToNot(HaveOccurred())

					kvCM := &corev1.ConfigMap{}
					yamlReader := yaml.NewYAMLOrJSONDecoder(cmFile, 1024)
					err = yamlReader.Decode(kvCM)
					Expect(err).ToNot(HaveOccurred())

					kvCMRef, err := reference.GetReference(commonTestUtils.GetScheme(), kvCM)
					Expect(err).ToNot(HaveOccurred())
					Expect(v1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvCMRef)).ToNot(HaveOccurred())
					resources := append(expected.toArray(), kvCM)

					cl := commonTestUtils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionUnknown)

					By("Check that the LifeMigrationConfig field contains the configmap values")
					lmc := foundResource.Spec.LiveMigrationConfig
					Expect(*lmc.BandwidthPerMigration).Should(Equal("32Mi"))
					Expect(*lmc.CompletionTimeoutPerGiB).Should(Equal(int64(444)))
					Expect(*lmc.ParallelMigrationsPerCluster).Should(Equal(uint32(3)))
					Expect(*lmc.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(2)))
					Expect(*lmc.ProgressTimeout).Should(Equal(int64(150)))

					By("Check that KV's MigrationConfiguration field does not contain the configmap values, yet")
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).To(BeNil())
					Expect(kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).To(BeNil())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeTrue())

					By("Run reconcile again")
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					By("Check that KV's MigrationConfiguration field contains the configmap values")
					kv = operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(*kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).Should(Equal(resource.MustParse("32Mi")))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).Should(Equal(int64(444)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).Should(Equal(uint32(3)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(2)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).Should(Equal(int64(150)))

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeFalse())
				})

				It("should ignore KubeVirt configMap into HC CR if there is no change, drop the cm and create backup", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					kvCM := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
parallelOutboundMigrationsPerNode: 2
bandwidthPerMigration: 64Mi
completionTimeoutPerGiB: 800
progressTimeout: 150`,
						},
					}
					kvCMRef, err := reference.GetReference(commonTestUtils.GetScheme(), kvCM)
					Expect(err).ToNot(HaveOccurred())
					Expect(v1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvCMRef)).ToNot(HaveOccurred())
					resources := append(expected.toArray(), kvCM)

					cl := commonTestUtils.InitClient(resources)
					foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					By("Check that the LifeMigrationConfig field contains the configmap values")
					lmc := foundResource.Spec.LiveMigrationConfig
					Expect(*lmc.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*lmc.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*lmc.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*lmc.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*lmc.ProgressTimeout).Should(Equal(progressTimeout))

					By("Check that KV's MigrationConfiguration field contains the configmap values")
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					err = hcoutil.GetRuntimeObject(context.TODO(), cl, kv, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(*kv.Spec.Configuration.MigrationConfiguration.BandwidthPerMigration).Should(Equal(resource.MustParse(bandwidthPerMigration)))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*kv.Spec.Configuration.MigrationConfiguration.ProgressTimeout).Should(Equal(progressTimeout))

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConfigMap", kvCmName)).To(BeFalse())
				})
			})

			Context("Adopt CDI Config on upgrade", func() {
				testResourceReqs := &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				}

				It("should adopt CDI config into HC CR", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					ScratchSpaceStorageClassValue := "ScratchSpaceStorageClassValue"

					expected.cdi.Spec.Config = &cdiv1beta1.CDIConfigSpec{
						ScratchSpaceStorageClass: &ScratchSpaceStorageClassValue,
						PodResourceRequirements:  testResourceReqs,
					}

					resources := expected.toArray()
					cl := commonTestUtils.InitClient(resources)
					foundHC, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionUnknown)

					By("Check that the spec.ScratchSpaceStorageClass is now populated")
					Expect(foundHC.Spec.ScratchSpaceStorageClass).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ScratchSpaceStorageClass).Should(Equal(ScratchSpaceStorageClassValue))
					By("Check that the spec.ResourceRequirements is now populated")
					Expect(foundHC.Spec.ResourceRequirements).ShouldNot(BeNil())
					Expect(foundHC.Spec.ResourceRequirements.StorageWorkloads).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ResourceRequirements.StorageWorkloads).Should(Equal(*testResourceReqs))

					By("Run reconcile again")
					foundHC, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that CDI's still contains the expected values")
					cdi := operands.NewCDIWithNameOnly(foundHC)
					err := hcoutil.GetRuntimeObject(context.TODO(), cl, cdi, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi.Spec.Config).ShouldNot(BeNil())
					Expect(cdi.Spec.Config.ScratchSpaceStorageClass).ShouldNot(BeNil())
					Expect(*cdi.Spec.Config.ScratchSpaceStorageClass).Should(Equal(ScratchSpaceStorageClassValue))
					Expect(cdi.Spec.Config.PodResourceRequirements).ShouldNot(BeNil())
					Expect(*cdi.Spec.Config.PodResourceRequirements).Should(Equal(*testResourceReqs))
				})

				It("should ignore CDI configurations if already exists in HC CR", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					hcoScratchSpaceStorageClass := "hcoScratchSpaceStorageClass"
					hcoPodResourceRequirements := &corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("700m"),
							corev1.ResourceMemory: resource.MustParse("4Gi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("2Gi"),
						},
					}

					cdiScratchSpaceStorageClass := "cdiScratchSpaceStorageClass"

					expected.hco.Spec.ScratchSpaceStorageClass = &hcoScratchSpaceStorageClass
					expected.hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
						StorageWorkloads: hcoPodResourceRequirements,
					}

					expected.cdi.Spec.Config = &cdiv1beta1.CDIConfigSpec{
						ScratchSpaceStorageClass: &cdiScratchSpaceStorageClass,
						PodResourceRequirements:  testResourceReqs,
					}

					cl := commonTestUtils.InitClient(expected.toArray())
					foundHC, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that the spec.ScratchSpaceStorageClass is now populated")
					Expect(foundHC.Spec.ScratchSpaceStorageClass).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ScratchSpaceStorageClass).Should(Equal(hcoScratchSpaceStorageClass))
					By("Check that the spec.ResourceRequirements is now populated")
					Expect(foundHC.Spec.ResourceRequirements).ShouldNot(BeNil())
					Expect(foundHC.Spec.ResourceRequirements.StorageWorkloads).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ResourceRequirements.StorageWorkloads).Should(Equal(*hcoPodResourceRequirements))

					By("Check that CDI's configuration are now the same as HCO's")
					cdi := operands.NewCDIWithNameOnly(foundHC)
					err := hcoutil.GetRuntimeObject(context.TODO(), cl, cdi, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi.Spec.Config).ToNot(BeNil())
					Expect(cdi.Spec.Config.ScratchSpaceStorageClass).ToNot(BeNil())
					Expect(*cdi.Spec.Config.ScratchSpaceStorageClass).Should(Equal(hcoScratchSpaceStorageClass))
					Expect(cdi.Spec.Config.PodResourceRequirements).ToNot(BeNil())
					Expect(*cdi.Spec.Config.PodResourceRequirements).Should(Equal(*hcoPodResourceRequirements))
				})

				It("should ignore CDI config into HC CR if there is no change", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					aScratchSpaceStorageClassValue := "aScratchSpaceStorageClassValue"

					expected.hco.Spec.ScratchSpaceStorageClass = &aScratchSpaceStorageClassValue
					expected.hco.Spec.ResourceRequirements = &hcov1beta1.OperandResourceRequirements{
						StorageWorkloads: testResourceReqs,
					}

					expected.cdi.Spec.Config = &cdiv1beta1.CDIConfigSpec{
						ScratchSpaceStorageClass: &aScratchSpaceStorageClassValue,
						PodResourceRequirements:  testResourceReqs,
					}

					cl := commonTestUtils.InitClient(expected.toArray())
					foundHC, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that the spec.ScratchSpaceStorageClass is now populated")
					Expect(foundHC.Spec.ScratchSpaceStorageClass).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ScratchSpaceStorageClass).Should(Equal(aScratchSpaceStorageClassValue))
					By("Check that the spec.ResourceRequirements is now populated")
					Expect(foundHC.Spec.ResourceRequirements).ShouldNot(BeNil())
					Expect(foundHC.Spec.ResourceRequirements.StorageWorkloads).ShouldNot(BeNil())
					Expect(*foundHC.Spec.ResourceRequirements.StorageWorkloads).Should(Equal(*testResourceReqs))

					By("Check that CDI's configuration are now the same as HCO's")
					cdi := operands.NewCDIWithNameOnly(foundHC)
					err := hcoutil.GetRuntimeObject(context.TODO(), cl, cdi, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(cdi.Spec.Config).ToNot(BeNil())
					Expect(cdi.Spec.Config.ScratchSpaceStorageClass).ToNot(BeNil())
					Expect(*cdi.Spec.Config.ScratchSpaceStorageClass).Should(Equal(aScratchSpaceStorageClassValue))
					Expect(cdi.Spec.Config.PodResourceRequirements).ToNot(BeNil())
					Expect(*cdi.Spec.Config.PodResourceRequirements).Should(Equal(*testResourceReqs))
				})
			})

			Context("Adopt IMS Config on upgrade", func() {

				It("should adopt IMS config into HC CR", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					vddkInitImageValue := "vddk-init-image-value-to-be-preserved"
					vddkk := "vddk-init-image"

					expected.imsConfig.Data[vddkk] = vddkInitImageValue

					cl := expected.initClient()
					foundHC, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionUnknown)

					By("Check that the spec.VddkInitImage is now populated")
					Expect(foundHC.Spec.VddkInitImage).ShouldNot(BeNil())
					Expect(*foundHC.Spec.VddkInitImage).Should(Equal(vddkInitImageValue))

					By("Run reconcile again")
					expected.hco = foundHC
					cl = expected.initClient()
					foundHC, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that IMS cm still contains the expected values")
					vmiCM, err := operands.NewIMSConfigForCR(foundHC, namespace)
					Expect(err).ToNot(HaveOccurred())

					err = hcoutil.GetRuntimeObject(context.TODO(), cl, vmiCM, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(vmiCM.Data).ShouldNot(BeNil())
					Expect(vmiCM.Data).To(HaveKeyWithValue(vddkk, vddkInitImageValue))

					// call again, make sure this time the requeue is false
					foundHC, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					ver, ok := foundHC.Status.GetVersion(hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).Should(Equal(newVersion))
				})

				It("should ignore IMS value if already exists in HC CR", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					vddkInitImageValue := "vddk-init-image-value-to-be-overwritten"
					vddkk := "vddk-init-image"

					expected.imsConfig.Data[vddkk] = vddkInitImageValue

					hcoVddkInitImageValue := "vddk-init-image-value-to-be-preserved"
					expected.hco.Spec.VddkInitImage = &hcoVddkInitImageValue

					resources := expected.toArray()
					cl := commonTestUtils.InitClient(resources)

					foundHC, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that the spec.VddkInitImage has not been updated on HCO CR")
					Expect(foundHC.Spec.VddkInitImage).ShouldNot(BeNil())
					Expect(*foundHC.Spec.VddkInitImage).Should(Equal(hcoVddkInitImageValue))
					Expect(*foundHC.Spec.VddkInitImage).Should(Not(Equal(vddkInitImageValue)))

					By("Check that IMS CM value is now the same as HCO's")
					vmiCM, err := operands.NewIMSConfigForCR(foundHC, namespace)
					Expect(err).ToNot(HaveOccurred())

					err = hcoutil.GetRuntimeObject(context.TODO(), cl, vmiCM, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(vmiCM.Data).ShouldNot(BeNil())
					Expect(vmiCM.Data).To(HaveKeyWithValue(vddkk, hcoVddkInitImageValue))

				})

				It("TODO: should ignore IMS config into HC CR if there is no change", func() {
					expected.hco.Status.UpdateVersion(hcoVersionName, oldVersion)

					vddkInitImageValue := "vddk-init-image-value-to-be-preserved"
					vddkk := "vddk-init-image"

					expected.imsConfig.Data[vddkk] = vddkInitImageValue

					expected.hco.Spec.VddkInitImage = &vddkInitImageValue

					resources := expected.toArray()
					cl := commonTestUtils.InitClient(resources)

					foundHC, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundHC, metav1.ConditionTrue)

					By("Check that the spec.VddkInitImage has not been updated on HCO CR")
					Expect(foundHC.Spec.VddkInitImage).ShouldNot(BeNil())
					Expect(*foundHC.Spec.VddkInitImage).Should(Equal(vddkInitImageValue))

					By("Check that IMS CM value is still the same as HCO's")
					vmiCM, err := operands.NewIMSConfigForCR(foundHC, namespace)
					Expect(err).ToNot(HaveOccurred())

					err = hcoutil.GetRuntimeObject(context.TODO(), cl, vmiCM, log)
					Expect(err).ToNot(HaveOccurred())
					Expect(vmiCM.Data).ShouldNot(BeNil())
					Expect(vmiCM.Data).To(HaveKeyWithValue(vddkk, vddkInitImageValue))
				})

			})

		})

		Context("Aggregate Negative Conditions", func() {
			const errorReason = "CdiTestError1"
			_ = os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
			It("should be degraded when a component is degraded", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
				Expect(cd.Message).Should(Equal("HCO is not available due to degraded components"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(commonDegradedReason))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
			})

			It("should be with all positive condition when all components working properly", func() {
				expected := getBasicDeployment()
				cl := expected.initClient()
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
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
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal("NetworkAddonsConfigNotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).Should(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).Should(Equal(reconcileCompleted))
			})
		})

		Context("Update Conflict Error", func() {
			It("Should requeue in case of update conflict", func() {
				expected := getBasicDeployment()
				expected.hco.Labels = nil
				cl := expected.initClient()
				rsc := schema.GroupResource{Group: hcoutil.APIVersionGroup, Resource: "hyperconvergeds.hco.kubevirt.io"}
				cl.InitiateUpdateErrors(func(obj client.Object) error {
					if _, ok := obj.(*hcov1beta1.HyperConverged); ok {
						return apierrors.NewConflict(rsc, "hco", errors.New("test error"))
					}
					return nil
				})
				r := initReconciler(cl, nil)

				r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
				if r.ownVersion == "" {
					r.ownVersion = version.Version
				}

				res, err := r.Reconcile(context.TODO(), request)

				Expect(err).To(HaveOccurred())
				Expect(apierrors.IsConflict(err)).To(BeTrue())
				Expect(res.Requeue).To(BeTrue())
			})

			It("Should requeue in case of update status conflict", func() {
				expected := getBasicDeployment()
				expected.hco.Status.Conditions = nil
				cl := expected.initClient()
				rs := schema.GroupResource{Group: hcoutil.APIVersionGroup, Resource: "hyperconvergeds.hco.kubevirt.io"}
				cl.Status().(*commonTestUtils.HcoTestStatusWriter).InitiateErrors(apierrors.NewConflict(rs, "hco", errors.New("test error")))
				r := initReconciler(cl, nil)

				r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
				if r.ownVersion == "" {
					r.ownVersion = version.Version
				}

				res, err := r.Reconcile(context.TODO(), request)

				Expect(err).ToNot(BeNil())
				Expect(apierrors.IsConflict(err)).To(BeTrue())
				Expect(res.Requeue).To(BeTrue())

			})
		})

		Context("Detection of a tainted configuration", func() {
			Context("Detection of a tainted configuration for kubevirt", func() {

				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.ObjectMeta.Annotations = map[string]string{
						common.JSONPatchKVAnnotationName: `
						[
							{
								"op": "add",
								"path": "/spec/configuration/migrations",
								"value": {"allowPostCopy": true}
							}
						]`,
					}
					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchKVAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
							Type:    hcov1beta1.ConditionTaintedConfiguration,
							Status:  metav1.ConditionTrue,
							Reason:  taintedConfigurationReason,
							Message: taintedConfigurationMessage,
						})))
					})

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(1, common.JSONPatchKVAnnotationName)
					})

					By("Verify that KV was modified by the annotation", func() {
						kv := operands.NewKubeVirtWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: kv.Name, Namespace: kv.Namespace},
								kv),
						).To(BeNil())

						Expect(kv.Spec.Configuration.MigrationConfiguration).ToNot(BeNil())
						Expect(kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).ToNot(BeNil())
						Expect(*kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).To(BeTrue())
					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchKVAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).To(BeNil())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchKVAnnotationName)
					})
				})

				It("Removes the TaintedConfiguration condition if the annotation is wrong", func() {
					hco := commonTestUtils.NewHco()
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchKVAnnotationName)

					hco.ObjectMeta.Annotations = map[string]string{
						// Set bad json format (missing comma)
						common.JSONPatchKVAnnotationName: `
						[
							{
								"op": "add"
								"path": "/spec/configuration/migrations",
								"value": {"allowPostCopy": true}
							}
						]`,
					}

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchKVAnnotationName)
					})
				})
			})

			Context("Detection of a tainted configuration for cdi", func() {

				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.ObjectMeta.Annotations = map[string]string{
						common.JSONPatchCDIAnnotationName: `[
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
				]`,
					}

					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchCDIAnnotationName)
					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
							Type:    hcov1beta1.ConditionTaintedConfiguration,
							Status:  metav1.ConditionTrue,
							Reason:  taintedConfigurationReason,
							Message: taintedConfigurationMessage,
						})))
					})

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(2, common.JSONPatchCDIAnnotationName)
					})

					By("Verify that CDI was modified by the annotation", func() {
						cdi := operands.NewCDIWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: cdi.Name, Namespace: cdi.Namespace},
								cdi),
						).To(BeNil())

						Expect(cdi.Spec.Config.FeatureGates).Should(ContainElement("fg1"))
						Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.Global).Should(BeEquivalentTo("50"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).Should(BeEquivalentTo("75"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).Should(BeEquivalentTo("25"))

					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchCDIAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).To(BeNil())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchKVAnnotationName)
					})
				})

				It("Removes the TaintedConfiguration condition if the annotation is wrong", func() {
					hco := commonTestUtils.NewHco()

					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchCDIAnnotationName)

					hco.ObjectMeta.Annotations = map[string]string{
						// Set bad json format (missing comma)
						common.JSONPatchKVAnnotationName: `[{`,
					}

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchKVAnnotationName)
					})
				})
			})

			Context("Detection of a tainted configuration for cna", func() {

				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.ObjectMeta.Annotations = map[string]string{
						common.JSONPatchCNAOAnnotationName: `[
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
						]`,
					}

					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchCNAOAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
							Type:    hcov1beta1.ConditionTaintedConfiguration,
							Status:  metav1.ConditionTrue,
							Reason:  taintedConfigurationReason,
							Message: taintedConfigurationMessage,
						})))
					})

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(2, common.JSONPatchCNAOAnnotationName)
					})

					By("Verify that CNA was modified by the annotation", func() {
						cna := operands.NewNetworkAddonsWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: cna.Name, Namespace: cna.Namespace},
								cna),
						).To(BeNil())

						Expect(cna.Spec.KubeMacPool).ToNot(BeNil())
						Expect(cna.Spec.KubeMacPool.RangeStart).Should(Equal("1.1.1.1.1.1"))
						Expect(cna.Spec.KubeMacPool.RangeEnd).Should(Equal("5.5.5.5.5.5"))
						Expect(cna.Spec.ImagePullPolicy).Should(BeEquivalentTo("Always"))
					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})
					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchCNAOAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).To(BeNil())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res).Should(Equal(reconcile.Result{Requeue: true}))

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchCNAOAnnotationName)
					})
				})

				It("Removes the TaintedConfiguration condition if the annotation is wrong", func() {
					hco := commonTestUtils.NewHco()
					hco.ObjectMeta.Annotations = map[string]string{
						// Set bad json
						common.JSONPatchKVAnnotationName: `[{`,
					}
					metrics.HcoMetrics.SetUnsafeModificationCount(5, common.JSONPatchCNAOAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					// Check conditions
					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchCNAOAnnotationName)
					})
				})
			})

			Context("Detection of a tainted configuration for all the annotations", func() {
				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco := commonTestUtils.NewHco()
					hco.ObjectMeta.Annotations = map[string]string{
						common.JSONPatchKVAnnotationName: `
						[
							{
								"op": "add",
								"path": "/spec/configuration/migrations",
								"value": {"allowPostCopy": true}
							}
						]`,
						common.JSONPatchCDIAnnotationName: `[
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
						]`,
						common.JSONPatchCNAOAnnotationName: `[
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
						]`,
					}
					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchKVAnnotationName)
					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchCDIAnnotationName)
					metrics.HcoMetrics.SetUnsafeModificationCount(0, common.JSONPatchCNAOAnnotationName)

					cl := commonTestUtils.InitClient([]runtime.Object{hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).Should(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(BeNil())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commonTestUtils.RepresentCondition(metav1.Condition{
							Type:    hcov1beta1.ConditionTaintedConfiguration,
							Status:  metav1.ConditionTrue,
							Reason:  taintedConfigurationReason,
							Message: taintedConfigurationMessage,
						})))
					})

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(1, common.JSONPatchKVAnnotationName)
						verifyUnsafeMetrics(2, common.JSONPatchCDIAnnotationName)
						verifyUnsafeMetrics(2, common.JSONPatchCNAOAnnotationName)
					})

					By("Verify that KV was modified by the annotation", func() {
						kv := operands.NewKubeVirtWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: kv.Name, Namespace: kv.Namespace},
								kv),
						).To(BeNil())

						Expect(kv.Spec.Configuration.MigrationConfiguration).ToNot(BeNil())
						Expect(kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).ToNot(BeNil())
						Expect(*kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).To(BeTrue())
					})
					By("Verify that CDI was modified by the annotation", func() {
						cdi := operands.NewCDIWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: cdi.Name, Namespace: cdi.Namespace},
								cdi),
						).To(BeNil())

						Expect(cdi.Spec.Config.FeatureGates).Should(ContainElement("fg1"))
						Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.Global).Should(BeEquivalentTo("50"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).Should(BeEquivalentTo("75"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).Should(BeEquivalentTo("25"))

					})
					By("Verify that CNA was modified by the annotation", func() {
						cna := operands.NewNetworkAddonsWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: cna.Name, Namespace: cna.Namespace},
								cna),
						).To(BeNil())

						Expect(cna.Spec.KubeMacPool).ToNot(BeNil())
						Expect(cna.Spec.KubeMacPool.RangeStart).Should(Equal("1.1.1.1.1.1"))
						Expect(cna.Spec.KubeMacPool.RangeEnd).Should(Equal("5.5.5.5.5.5"))
						Expect(cna.Spec.ImagePullPolicy).Should(BeEquivalentTo("Always"))
					})
				})
			})
		})

		Context("Test migrateBeforeUpgrade", func() {
			bandwidthPerMigration := "64Mi"
			completionTimeoutPerGiB := int64(800)
			parallelMigrationsPerCluster := uint32(5)
			parallelOutboundMigrationsPerNode := uint32(2)
			progressTimeout := int64(150)

			var expected *BasicExpected
			BeforeEach(func() {
				expected = getBasicDeployment()
			})

			createBackupEvent := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Created",
					Msg:       "Created ConfigMap kubevirt-config-backup",
				},
			}
			rmCmEvent := []commonTestUtils.MockEvent{
				{
					EventType: corev1.EventTypeNormal,
					Reason:    "Killing",
					Msg:       "Removed ConfigMap kubevirt-config",
				},
			}

			Context("Positive Tests - KV Config", func() {
				It("Should delete the CM and create a backup", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)

					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					By("Check events")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}

					Expect(events.CheckEvents(expectedEvents)).To(BeTrue())
				})

				It("Should do nothing if the kv configMap does not exists", func() {
					cl := expected.initClient()

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeFalse())

					By("Check events")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())

					expectedEvents = []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())
				})

				It("Should drop the KV CM if both kv configMap and its backup exist", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})

					resources = append(resources, &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      backupKvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})

					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					By("Don't expect the create backup event")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())

					By("Expect the delete cm event")
					expectedEvents = []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeTrue())
				})

				It("Should adopt KV configuration from the configMap, if missing in HCO", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
parallelOutboundMigrationsPerNode: 2
bandwidthPerMigration: 64Mi
completionTimeoutPerGiB: 800
progressTimeout: 150`,
						},
					})
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))

					By("Check events")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					Expect(events.CheckEvents(createBackupEvent)).To(BeTrue())
					Expect(events.CheckEvents(rmCmEvent)).To(BeFalse())

					By("Run again, to simulate the next reconcile loop")
					events.Reset()
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))

					By("Check events")
					Expect(events.CheckEvents(createBackupEvent)).To(BeFalse())
					Expect(events.CheckEvents(rmCmEvent)).To(BeTrue())
				})

				It("Should adopt KV configuration from a real configMap, if missing in HCO", func() {
					yamlFile, err := os.Open(getTestFilesLocation() + "/kubevirt-config.yaml")
					Expect(err).ToNot(HaveOccurred())

					realBandwidthPerMigration := "32Mi"
					realCompletionTimeoutPerGiB := int64(444)
					realParallelMigrationsPerCluster := uint32(3)

					cm := &corev1.ConfigMap{}
					yamlDec := yaml.NewYAMLOrJSONDecoder(yamlFile, 1024)
					err = yamlDec.Decode(cm)
					Expect(err).ToNot(HaveOccurred())

					resources := append(expected.toArray(), cm)
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(realBandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(realCompletionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(realParallelMigrationsPerCluster))
					Expect(req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(BeNil())
					Expect(req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(BeNil())

					By("Check events")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					Expect(events.CheckEvents(createBackupEvent)).To(BeTrue())
					Expect(events.CheckEvents(rmCmEvent)).To(BeFalse())

					By("Run again, to simulate the next reconcile loop")
					events.Reset()
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(realBandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(realCompletionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(realParallelMigrationsPerCluster))
					Expect(req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(BeNil())
					Expect(req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(BeNil())

					By("Check events")
					Expect(events.CheckEvents(createBackupEvent)).To(BeFalse())
					Expect(events.CheckEvents(rmCmEvent)).To(BeTrue())
				})

				It("Should adopt KV configuration from the configMap, with unknown key", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
parallelOutboundMigrationsPerNode: 2
bandwidthPerMigration: 64Mi
completionTimeoutPerGiB: 800
progressTimeout: 150
unknownKey: 42`,
						},
					})
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))

					By("Run again, to simulate the next reconcile loop")
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))
				})

				It("Should adopt KV configuration from the configMap, with missing keys", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
completionTimeoutPerGiB: 800
progressTimeout: 150`,
						},
					})
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(BeNil())
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(BeNil())
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))

					By("Run again, to simulate the next reconcile loop")
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(BeNil())
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(BeNil())
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))
				})

				It("Should ignore KV configuration from the configMap, if if HCO contains the same values", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 5
parallelOutboundMigrationsPerNode: 2
bandwidthPerMigration: 64Mi
completionTimeoutPerGiB: 800
progressTimeout: 150`,
						},
					})

					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(completionTimeoutPerGiB))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(parallelMigrationsPerCluster))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(progressTimeout))
				})

				It("Should adopt KV configuration from the configMap, if if HCO contains the different values", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 4
parallelOutboundMigrationsPerNode: 4
bandwidthPerMigration: 16Mi
completionTimeoutPerGiB: 400
progressTimeout: 300`,
						},
					})

					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal("16Mi"))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(int64(300)))

					By("Run again, to simulate the next reconcile loop")
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal("16Mi"))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(int64(300)))
				})

				It("Should adopt KV configuration from the configMap with missing keys, if if HCO contains the livemigration config", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `parallelMigrationsPerCluster: 4
completionTimeoutPerGiB: 400
progressTimeout: 300`,
						},
					})

					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = &bandwidthPerMigration
					expected.hco.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB = &completionTimeoutPerGiB
					expected.hco.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode = &parallelOutboundMigrationsPerNode
					expected.hco.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster = &parallelMigrationsPerCluster
					expected.hco.Spec.LiveMigrationConfig.ProgressTimeout = &progressTimeout

					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeTrue())

					foundKvCm, foundBackup := searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeTrue())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(int64(300)))

					By("Run again, to simulate the next reconcile loop")
					modified, err = r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).To(BeFalse())

					foundKvCm, foundBackup = searchKvConfigMaps(cl)
					Expect(foundKvCm).To(BeFalse())
					Expect(foundBackup).To(BeTrue())

					Expect(*req.Instance.Spec.LiveMigrationConfig.BandwidthPerMigration).Should(Equal(bandwidthPerMigration))
					Expect(*req.Instance.Spec.LiveMigrationConfig.CompletionTimeoutPerGiB).Should(Equal(int64(400)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelMigrationsPerCluster).Should(Equal(uint32(4)))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ParallelOutboundMigrationsPerNode).Should(Equal(parallelOutboundMigrationsPerNode))
					Expect(*req.Instance.Spec.LiveMigrationConfig.ProgressTimeout).Should(Equal(int64(300)))
				})
			})

			Context("Test Errors - KV Config", func() {
				It("Should return error if failed to read KV CM", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})
					cl := commonTestUtils.InitClient(resources)
					fakeError := fmt.Errorf("fake read error")
					cl.InitiateGetErrors(func(key client.ObjectKey) error {
						if key.Name == kvCmName {
							return fakeError
						}
						return nil
					})

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).To(HaveOccurred())
					Expect(err).Should(Equal(fakeError))
					Expect(modified).Should(BeFalse())

					By("Make sure that no event emitted")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())

					expectedEvents = []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())
				})

				It("Should return error if failed to create the KV backup CM", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})
					cl := commonTestUtils.InitClient(resources)
					fakeError := fmt.Errorf("fake create error")
					cl.InitiateCreateErrors(func(obj client.Object) error {
						cm, ok := obj.(*corev1.ConfigMap)
						if ok && cm.Name == backupKvCmName {
							return fakeError
						}
						return nil
					})

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).To(HaveOccurred())
					Expect(err).Should(Equal(fakeError))
					Expect(modified).Should(BeFalse())

					By("Make sure that no event emitted")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())

					expectedEvents = []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())
				})

				It("Should return error if format of the MC is wrong", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
						Data: map[string]string{
							liveMigrationKey: `wrong yaml format`,
						},
					})
					cl := commonTestUtils.InitClient(resources)

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).ToNot(HaveOccurred())
					Expect(modified).Should(BeFalse())

					By("check that the create backup and the delete cm events were emitted")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeTrue())
				})

				It("Should return error if failed to delete KV CM", func() {
					resources := append(expected.toArray(), &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      kvCmName,
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					})

					cl := commonTestUtils.InitClient(resources)

					fakeError := fmt.Errorf("fake delete error")
					cl.InitiateDeleteErrors(func(obj client.Object) error {
						if unstructed, ok := obj.(runtime.Unstructured); ok {
							kind := unstructed.GetObjectKind()
							if kind.GroupVersionKind().Kind == "ConfigMap" && obj.GetName() == kvCmName {
								return fakeError
							}
						}
						return nil
					})

					r := initReconciler(cl, nil)
					req := commonTestUtils.NewReq(expected.hco)

					modified, err := r.migrateBeforeUpgrade(req)
					Expect(err).To(HaveOccurred())
					Expect(err).Should(Equal(fakeError))
					Expect(modified).Should(BeFalse())

					By("check that the create backup event was emitted")
					events := r.eventEmitter.(*commonTestUtils.EventEmitterMock)
					expectedEvents := []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Created",
							Msg:       "Created ConfigMap kubevirt-config-backup",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeTrue())

					By("check that the delete cm event was not emitted")
					expectedEvents = []commonTestUtils.MockEvent{
						{
							EventType: corev1.EventTypeNormal,
							Reason:    "Killing",
							Msg:       "Removed ConfigMap kubevirt-config",
						},
					}
					Expect(events.CheckEvents(expectedEvents)).To(BeFalse())
				})
			})
		})
	})
})

func verifyUnsafeMetrics(expected int, annotation string) {
	count, err := metrics.HcoMetrics.GetUnsafeModificationsCount(annotation)
	ExpectWithOffset(1, err).ShouldNot(HaveOccurred())
	ExpectWithOffset(1, count).Should(BeEquivalentTo(expected))
}

func searchKvConfigMaps(cl client.Client) (bool, bool) {
	cms := &corev1.ConfigMapList{}

	err := cl.List(context.TODO(), cms, client.InNamespace(namespace))
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	foundKvCm := false
	foundBackup := false
	for _, cm := range cms.Items {
		if cm.Name == kvCmName {
			foundKvCm = true
		}

		if cm.Name == backupKvCmName {
			foundBackup = true
		}
	}

	return foundKvCm, foundBackup
}

func searchInRelatedObjects(relatedObjects []corev1.ObjectReference, kind, name string) bool {
	for _, obj := range relatedObjects {
		if obj.Kind == kind && obj.Name == name {
			return true
		}
	}
	return false
}
