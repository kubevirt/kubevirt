package hyperconverged

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/blang/semver/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimetav1 "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/reference"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta2 "kubevirt.io/ssp-operator/api/v1beta2"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/alerts"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commontestutils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/reqresolver"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/monitoring/hyperconverged/metrics"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
)

// name and namespace of our primary resource
const (
	name      = "kubevirt-hyperconverged"
	namespace = "kubevirt-hyperconverged"
)

var _ = Describe("HyperconvergedController", func() {

	_ = os.Setenv(hcoutil.OperatorConditionNameEnvVar, "OPERATOR_CONDITION")

	getClusterInfo := hcoutil.GetClusterInfo

	Describe("Reconcile HyperConverged", func() {

		BeforeEach(func() {
			hcoutil.GetClusterInfo = func() hcoutil.ClusterInfo {
				return commontestutils.ClusterInfoMock{}
			}
		})

		AfterEach(func() {
			hcoutil.GetClusterInfo = getClusterInfo
		})

		Context("HCO Lifecycle", func() {

			var (
				hcoNamespace *corev1.Namespace
			)

			BeforeEach(func() {
				_ = os.Setenv("VIRTIOWIN_CONTAINER", commontestutils.VirtioWinImage)
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)
				_ = os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
				hcoNamespace = commontestutils.NewHcoNamespace()

				reqresolver.GeneratePlaceHolders()
			})

			It("should handle not found", func() {
				cl := commontestutils.InitClient([]client.Object{})
				r := initReconciler(cl, nil)

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
				verifyHyperConvergedCRExistsMetricFalse()
			})

			It("should ignore invalid requests", func() {
				hco := commontestutils.NewHco()
				hco.ObjectMeta = metav1.ObjectMeta{
					Name:      "invalid",
					Namespace: "invalid",
				}
				cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
				r := initReconciler(cl, nil)

				// Do the reconcile
				var invalidRequest = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      "invalid",
						Namespace: "invalid",
					},
				}
				res, err := r.Reconcile(context.TODO(), invalidRequest)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionFalse,
					Reason:  invalidRequestReason,
					Message: fmt.Sprintf(invalidRequestMessageFormat, name, namespace),
				})))
			})

			It("should create all managed resources", func() {

				hco := commontestutils.NewHco()
				hco.Spec.FeatureGates = hcov1beta1.HyperConvergedFeatureGates{
					DownwardMetrics: ptr.To(true),
				}

				ci := hcoutil.GetClusterInfo()
				cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco, ci.GetCSV()})
				monitoringReconciler := alerts.NewMonitoringReconciler(ci, cl, commontestutils.NewEventEmitterMock(), commontestutils.GetScheme())

				r := initReconciler(cl, nil)
				r.monitoringReconciler = monitoringReconciler

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
				verifyHyperConvergedCRExistsMetricTrue()

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: reconcileInitMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionAvailable,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionProgressing,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionDegraded,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionUpgradeable,
					Status:  metav1.ConditionUnknown,
					Reason:  reconcileInit,
					Message: "Initializing HyperConverged cluster",
				})))

				verifySystemHealthStatusError(foundResource)

				expectedFeatureGates := []string{
					"CPUManager",
					"Snapshot",
					"HotplugVolumes",
					"HostDevices",
					"WithHostModelCPU",
					"HypervStrictCheck",
					"ExpandDisks",
					"DownwardMetrics",
					"VMExport",
					"KubevirtSeccompProfile",
					"VMPersistentState",
					"InstancetypeReferencePolicy",
				}
				// Get the KV
				kvList := &kubevirtcorev1.KubeVirtList{}
				Expect(cl.List(context.TODO(), kvList)).To(Succeed())
				Expect(kvList.Items).To(HaveLen(1))
				kv := kvList.Items[0]
				Expect(kv.Spec.Configuration.DeveloperConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(HaveLen(len(expectedFeatureGates)))
				Expect(kv.Spec.Configuration.DeveloperConfiguration.FeatureGates).To(ContainElements(expectedFeatureGates))

				// Ensure the KubeVirt seccomp profile is set
				Expect(kv.Spec.Configuration.SeccompConfiguration).ToNot(BeNil())
				Expect(kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile).ToNot(BeNil())
				Expect(kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile.CustomProfile).ToNot(BeNil())
				Expect(kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile.CustomProfile.RuntimeDefaultProfile).To(BeFalse())
				Expect(*kv.Spec.Configuration.SeccompConfiguration.VirtualMachineInstanceProfile.CustomProfile.LocalhostProfile).To(Equal("kubevirt/kubevirt.json"))

				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false}))
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
				verifyHyperConvergedCRExistsMetricTrue()

				// Get the HCO
				foundResource = &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				// Check conditions
				Expect(foundResource.Status.RelatedObjects).To(HaveLen(27))
				expectedRef := corev1.ObjectReference{
					Kind:            "PrometheusRule",
					Namespace:       namespace,
					Name:            "kubevirt-hyperconverged-prometheus-rule",
					APIVersion:      "monitoring.coreos.com/v1",
					ResourceVersion: "1",
				}
				Expect(foundResource.Status.RelatedObjects).To(ContainElement(expectedRef))
			})

			It("should find all managed resources", func() {

				expected := getBasicDeployment()

				expected.kv.Status.Conditions = nil
				expected.cdi.Status.Conditions = nil
				expected.cna.Status.Conditions = nil
				expected.ssp.Status.Conditions = nil

				pm := &monitoringv1.PrometheusRule{
					TypeMeta: metav1.TypeMeta{
						Kind:       monitoringv1.PrometheusRuleKind,
						APIVersion: "monitoring.coreos.com/v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Namespace:       namespace,
						Name:            "kubevirt-hyperconverged-prometheus-rule",
						UID:             "1234567890",
						ResourceVersion: "123",
					},
					Spec: monitoringv1.PrometheusRuleSpec{},
				}

				resources := expected.toArray()
				resources = append(resources, pm)
				cl := commontestutils.InitClient(resources)

				r := initReconciler(cl, nil)
				r.monitoringReconciler = alerts.NewMonitoringReconciler(hcoutil.GetClusterInfo(), cl, commontestutils.NewEventEmitterMock(), commontestutils.GetScheme())

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				verifyHyperConvergedCRExistsMetricTrue()

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))

				verifySystemHealthStatusError(foundResource)

				Expect(foundResource.Status.RelatedObjects).To(HaveLen(22))
				expectedRef := corev1.ObjectReference{
					Kind:            "PrometheusRule",
					Namespace:       namespace,
					Name:            "kubevirt-hyperconverged-prometheus-rule",
					APIVersion:      "monitoring.coreos.com/v1",
					ResourceVersion: "124",
					UID:             "1234567890",
				}
				Expect(foundResource.Status.RelatedObjects).To(ContainElement(expectedRef))
			})

			It("should label all managed resources", func() {
				expected := getBasicDeployment()

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				// Check whether related objects have the labels or not
				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				for _, relatedObj := range foundResource.Status.RelatedObjects {
					foundRelatedObj := &unstructured.Unstructured{}
					foundRelatedObj.SetGroupVersionKind(relatedObj.GetObjectKind().GroupVersionKind())
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: relatedObj.Name, Namespace: relatedObj.Namespace},
							foundRelatedObj),
					).ToNot(HaveOccurred())

					foundLabels := foundRelatedObj.GetLabels()
					Expect(foundLabels[hcoutil.AppLabel]).To(Equal(expected.hco.Name))
					Expect(foundLabels[hcoutil.AppLabelPartOf]).To(Equal(hcoutil.HyperConvergedCluster))
					Expect(foundLabels[hcoutil.AppLabelManagedBy]).To(Equal(hcoutil.OperatorName))
					Expect(foundLabels[hcoutil.AppLabelVersion]).To(Equal(version.Version))
					Expect(foundLabels[hcoutil.AppLabelComponent]).ToNot(BeNil())
				}
			})

			It("should update resource versions of objects in relatedObjects", func() {

				expected := getBasicDeployment()
				cl := expected.initClient()

				r := initReconciler(cl, nil)

				// Reconcile to get all related objects under HCO's status
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Update Kubevirt (an example of secondary CR)
				foundKubevirt := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.kv.Name, Namespace: expected.kv.Namespace},
						foundKubevirt),
				).ToNot(HaveOccurred())
				foundKubevirt.Labels = map[string]string{"key": "value"}
				Expect(cl.Update(context.TODO(), foundKubevirt)).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in secondary CR
				rq := reqresolver.GetSecondaryCRRequest()

				// Reconcile again to update HCO's status
				res, err = r.Reconcile(context.TODO(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the latest objects
				latestHCO := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						latestHCO),
				).ToNot(HaveOccurred())

				latestKubevirt := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.kv.Name, Namespace: expected.kv.Namespace},
						latestKubevirt),
				).ToNot(HaveOccurred())

				kubevirtRef, err := reference.GetReference(cl.Scheme(), latestKubevirt)
				Expect(err).ToNot(HaveOccurred())
				// This fails when resource versions are not up-to-date
				Expect(latestHCO.Status.RelatedObjects).To(ContainElement(*kubevirtRef))
			})

			It("should update APIVersion of objects in relatedObjects", func() {

				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Reconcile to get all related objects under HCO's status
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the latest objects
				HCO := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						HCO),
				).ToNot(HaveOccurred())

				// Mock an outdated APIVersion on one of the resources
				consolePlugin := &consolev1.ConsolePlugin{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.consolePlugin.Name, Namespace: expected.consolePlugin.Namespace},
						consolePlugin),
				).ToNot(HaveOccurred())
				newCpRef, err := reference.GetReference(cl.Scheme(), consolePlugin)
				Expect(err).ToNot(HaveOccurred())
				outdatedCpRef := newCpRef.DeepCopy()
				outdatedCpRef.APIVersion = "console.openshift.io/v1alpha1"
				Expect(objectreferencesv1.RemoveObjectReference(&HCO.Status.RelatedObjects, *newCpRef)).ToNot(HaveOccurred())
				Expect(objectreferencesv1.SetObjectReference(&HCO.Status.RelatedObjects, *outdatedCpRef)).ToNot(HaveOccurred())
				Expect(
					cl.Status().Update(context.TODO(), HCO),
				).ToNot(HaveOccurred())

				HCO = &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						HCO),
				).ToNot(HaveOccurred())
				Expect(HCO.Status.RelatedObjects).ToNot(ContainElement(*newCpRef))
				Expect(HCO.Status.RelatedObjects).To(ContainElement(*outdatedCpRef))

				// Update Kubevirt (an example of secondary CR)
				foundKubevirt := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.kv.Name, Namespace: expected.kv.Namespace},
						foundKubevirt),
				).ToNot(HaveOccurred())
				foundKubevirt.Labels = map[string]string{"key": "value"}
				Expect(cl.Update(context.TODO(), foundKubevirt)).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in secondary CR
				rq := reqresolver.GetSecondaryCRRequest()

				// Reconcile again to update HCO's status
				res, err = r.Reconcile(context.TODO(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the latest objects
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						HCO),
				).ToNot(HaveOccurred())

				Expect(HCO.Status.RelatedObjects).ToNot(ContainElement(*outdatedCpRef))
				Expect(HCO.Status.RelatedObjects).To(ContainElement(*newCpRef))

			})

			It("should update resource versions of objects in relatedObjects even when there is no update on secondary CR", func() {

				expected := getBasicDeployment()
				cl := expected.initClient()

				r := initReconciler(cl, nil)

				// Reconcile to get all related objects under HCO's status
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Update Kubevirt's resource version (an example of secondary CR)
				foundKubevirt := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.kv.Name, Namespace: expected.kv.Namespace},
						foundKubevirt),
				).ToNot(HaveOccurred())
				// no change. only to bump resource version
				Expect(cl.Update(context.TODO(), foundKubevirt)).ToNot(HaveOccurred())

				// mock a reconciliation triggered by a change in secondary CR
				rq := reqresolver.GetSecondaryCRRequest()

				// Reconcile again to update HCO's status
				res, err = r.Reconcile(context.TODO(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the latest objects
				latestHCO := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						latestHCO),
				).ToNot(HaveOccurred())

				latestKubevirt := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.kv.Name, Namespace: expected.kv.Namespace},
						latestKubevirt),
				).ToNot(HaveOccurred())

				kubevirtRef, err := reference.GetReference(cl.Scheme(), latestKubevirt)
				Expect(err).ToNot(HaveOccurred())
				// This fails when resource versions are not up-to-date
				Expect(latestHCO.Status.RelatedObjects).To(ContainElement(*kubevirtRef))
			})

			It("should set different template namespace to ssp CR", func() {
				expected := getBasicDeployment()
				expected.hco.Spec.CommonTemplatesNamespace = &expected.hco.Namespace

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				foundResource := &sspv1beta2.SSP{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.ssp.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())

				Expect(foundResource.Spec.CommonTemplates.Namespace).To(Equal(expected.hco.Namespace), "common-templates namespace should be "+expected.hco.Namespace)
			})

			It("should complete when components are finished", func() {
				expected := getBasicDeployment()

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				// Check conditions
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionAvailable,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionProgressing,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionDegraded,
					Status:  metav1.ConditionFalse,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))
				Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
					Type:    hcov1beta1.ConditionUpgradeable,
					Status:  metav1.ConditionTrue,
					Reason:  reconcileCompleted,
					Message: reconcileCompletedMessage,
				})))

				verifySystemHealthStatusHealthy(foundResource)
			})

			It("should increment counter when out-of-band change overwritten", func() {
				hco := commontestutils.NewHco()
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				existingResource, err := operands.NewKubeVirt(hco, namespace)
				Expect(err).ToNot(HaveOccurred())
				existingResource.APIVersion, existingResource.Kind = kubevirtcorev1.KubeVirtGroupVersionKind.ToAPIVersionAndKind() // necessary for metrics

				// now, modify KV's node placement
				existingResource.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})
				existingResource.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})

				existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"] = "BADvalue2"

				cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco, existingResource})
				r := initReconciler(cl, nil)

				// mock a reconciliation triggered by a change in secondary CR
				rq := reqresolver.GetSecondaryCRRequest()

				counterValueBefore, err := metrics.GetOverwrittenModificationsCount(existingResource.Kind, existingResource.Name)
				Expect(err).ToNot(HaveOccurred())

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("value2"))

				counterValueAfter, err := metrics.GetOverwrittenModificationsCount(foundResource.Kind, foundResource.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(counterValueAfter).To(Equal(counterValueBefore + 1))

			})

			It("should not increment counter when CR was changed by HCO", func() {
				hco := commontestutils.NewHco()
				hco.Spec.Infra = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				hco.Spec.Workloads = hcov1beta1.HyperConvergedConfig{NodePlacement: commontestutils.NewNodePlacement()}
				existingResource, err := operands.NewKubeVirt(hco, namespace)
				Expect(err).ToNot(HaveOccurred())
				existingResource.Kind = kubevirtcorev1.KubeVirtGroupVersionKind.Kind // necessary for metrics

				// now, modify KV's node placement
				existingResource.Spec.Infra.NodePlacement.Tolerations = append(hco.Spec.Infra.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})
				existingResource.Spec.Workloads.NodePlacement.Tolerations = append(hco.Spec.Workloads.NodePlacement.Tolerations, corev1.Toleration{
					Key: "key3", Operator: "operator3", Value: "value3", Effect: "effect3", TolerationSeconds: ptr.To[int64](3),
				})

				existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"] = "BADvalue1"
				existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"] = "BADvalue2"

				cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco, existingResource})
				r := initReconciler(cl, nil)

				counterValueBefore, err := metrics.GetOverwrittenModificationsCount(existingResource.Kind, existingResource.Name)
				Expect(err).ToNot(HaveOccurred())

				// Do the reconcile triggered by HCO
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))

				foundResource := &kubevirtcorev1.KubeVirt{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: existingResource.Name, Namespace: existingResource.Namespace},
						foundResource),
				).To(Succeed())

				Expect(existingResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(3))
				Expect(existingResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("BADvalue1"))
				Expect(existingResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("BADvalue2"))

				Expect(foundResource.Spec.Infra.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Workloads.NodePlacement.Tolerations).To(HaveLen(2))
				Expect(foundResource.Spec.Infra.NodePlacement.NodeSelector["key1"]).To(Equal("value1"))
				Expect(foundResource.Spec.Workloads.NodePlacement.NodeSelector["key2"]).To(Equal("value2"))

				counterValueAfter, err := metrics.GetOverwrittenModificationsCount(foundResource.Kind, foundResource.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(counterValueAfter).To(Equal(counterValueBefore))

			})

			It(`should be not available when components with missing "Available" condition`, func() {
				expected := getBasicDeployment()

				var cl *commontestutils.HcoTestClient
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
				monitoringReconciler := alerts.NewMonitoringReconciler(hcoutil.GetClusterInfo(), cl, commontestutils.NewEventEmitterMock(), commontestutils.GetScheme())
				r.monitoringReconciler = monitoringReconciler

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(Succeed())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(foundResource.Status.RelatedObjects).To(HaveLen(22))
				Expect(foundResource.Finalizers).To(Equal([]string{FinalizerName}))

				// Now, delete HCO
				delTime := time.Now().UTC().Add(-1 * time.Minute)
				expected.hco.DeletionTimestamp = &metav1.Time{Time: delTime}
				expected.hco.Finalizers = []string{FinalizerName}
				cl = expected.initClient()

				r = initReconciler(cl, nil)
				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))

				res, err = r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false}))

				foundResource = &hcov1beta1.HyperConverged{}
				err = cl.Get(context.TODO(),
					types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
					foundResource)
				Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

				verifyHyperConvergedCRExistsMetricFalse()
			})

			It(`should set a finalizer on HCO CR`, func() {
				expected := getBasicDeployment()
				cl := expected.initClient()
				r := initReconciler(cl, nil)
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).To(Succeed())

				Expect(foundResource.Status.RelatedObjects).ToNot(BeNil())
				Expect(foundResource.Finalizers).To(Equal([]string{FinalizerName}))
			})

			It("Should not be ready if one of the operands is returns error, on create", func() {
				hco := commontestutils.NewHco()
				cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
				cl.InitiateCreateErrors(func(obj client.Object) error {
					if _, ok := obj.(*cdiv1beta1.CDI); ok {
						return errors.New("fake create error")
					}
					return nil
				})
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: true}))

				// Get the HCO
				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundResource),
				).To(Succeed())

				// Check condition
				foundCond := false
				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionReconcileComplete {
						foundCond = true
						Expect(cond.Status).To(Equal(metav1.ConditionFalse))
						Expect(cond.Message).To(ContainSubstring("fake create error"))
						break
					}
				}
				Expect(foundCond).To(BeTrue())
			})

			It("Should be ready even if one of the operands is returns error, on update", func() {
				expected := getBasicDeployment()
				expected.kv.Spec.Configuration.DeveloperConfiguration = &kubevirtcorev1.DeveloperConfiguration{
					FeatureGates: []string{"fakeFg"}, // force update
				}
				cl := expected.initClient()
				cl.InitiateUpdateErrors(func(obj client.Object) error {
					if _, ok := obj.(*kubevirtcorev1.KubeVirt); ok {
						return errors.New("fake update error")
					}
					return nil
				})

				hco := commontestutils.NewHco()
				r := initReconciler(cl, nil)

				// Do the reconcile
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal(reconcile.Result{Requeue: false}))

				// Get the HCO
				foundHyperConverged := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
						foundHyperConverged),
				).To(Succeed())

				// Check condition
				foundCond := false
				for _, cond := range foundHyperConverged.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionReconcileComplete {
						foundCond = true
						Expect(cond.Status).To(Equal(metav1.ConditionFalse))
						Expect(cond.Message).To(ContainSubstring("fake update error"))
						break
					}
				}
				Expect(foundCond).To(BeTrue())
			})

			It("Should upgrade the status.observedGeneration field", func() {
				expected := getBasicDeployment()
				expected.hco.Generation = 10
				cl := expected.initClient()
				foundResource, _, _ := doReconcile(cl, expected.hco, nil)

				Expect(foundResource.Status.ObservedGeneration).To(BeEquivalentTo(10))
			})

		})

		Context("APIServer CR", func() {

			externalClusterInfo := hcoutil.GetClusterInfo

			BeforeEach(func() {
				hcoutil.GetClusterInfo = getClusterInfo
			})

			AfterEach(func() {
				hcoutil.GetClusterInfo = externalClusterInfo
			})

			It("Should refresh cached APIServer if the reconciliation is caused by a change there", func() {

				initialTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
					Type:         openshiftconfigv1.TLSProfileIntermediateType,
					Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
				}
				customTLSSecurityProfile := &openshiftconfigv1.TLSSecurityProfile{
					Type:   openshiftconfigv1.TLSProfileModernType,
					Modern: &openshiftconfigv1.ModernTLSProfile{},
				}

				clusterVersion := &openshiftconfigv1.ClusterVersion{
					ObjectMeta: metav1.ObjectMeta{
						Name: "version",
					},
					Spec: openshiftconfigv1.ClusterVersionSpec{
						ClusterID: "clusterId",
					},
				}

				infrastructure := &openshiftconfigv1.Infrastructure{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: openshiftconfigv1.InfrastructureStatus{
						ControlPlaneTopology:   openshiftconfigv1.HighlyAvailableTopologyMode,
						InfrastructureTopology: openshiftconfigv1.HighlyAvailableTopologyMode,
						PlatformStatus: &openshiftconfigv1.PlatformStatus{
							Type: "mocked",
						},
					},
				}

				ingress := &openshiftconfigv1.Ingress{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.IngressSpec{
						Domain: "domain",
					},
				}

				apiServer := &openshiftconfigv1.APIServer{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.APIServerSpec{
						TLSSecurityProfile: initialTLSSecurityProfile,
					},
				}

				dns := &openshiftconfigv1.DNS{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Spec: openshiftconfigv1.DNSSpec{
						BaseDomain: commontestutils.BaseDomain,
					},
				}

				ipv4network := &openshiftconfigv1.Network{
					ObjectMeta: metav1.ObjectMeta{
						Name: "cluster",
					},
					Status: openshiftconfigv1.NetworkStatus{
						ClusterNetwork: []openshiftconfigv1.ClusterNetworkEntry{
							{
								CIDR: "10.128.0.0/14",
							},
						},
					},
				}

				expected := getBasicDeployment()
				Expect(expected.hco.Spec.TLSSecurityProfile).To(BeNil())

				expected.csv = commontestutils.ClusterInfoMock{}.GetCSV()
				resources := expected.toArray()
				resources = append(resources, clusterVersion, infrastructure, ingress, apiServer, dns, ipv4network)
				cl := commontestutils.InitClient(resources)

				logger := zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("hyperconverged_controller_test")
				Expect(hcoutil.GetClusterInfo().Init(context.TODO(), cl, logger)).To(Succeed())

				Expect(initialTLSSecurityProfile).ToNot(Equal(customTLSSecurityProfile), "customTLSSecurityProfile should be a different value")

				r := initReconciler(cl, nil)

				// Reconcile to get all related objects under HCO's status
				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				foundResource := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				checkAvailability(foundResource, metav1.ConditionTrue)
				Expect(foundResource.Spec.TLSSecurityProfile).To(BeNil(), "TLSSecurityProfile on HCO CR should still be nil")

				By("Verify that Kubevirt was properly configured with initialTLSSecurityProfile", func() {
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: kv.Name, Namespace: kv.Namespace},
							kv),
					).To(Succeed())

					Expect(kv.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(Equal(kubevirtcorev1.VersionTLS12))
					Expect(kv.Spec.Configuration.TLSConfiguration.Ciphers).To(Equal([]string{
						"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
						"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
						"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
						"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
						"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256",
						"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256",
					}))
				})
				By("Verify that CDI was properly configured with initialTLSSecurityProfile", func() {
					cdi := operands.NewCDIWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: cdi.Name, Namespace: cdi.Namespace},
							cdi),
					).To(Succeed())

					Expect(cdi.Spec.Config.TLSSecurityProfile).To(Equal(openshift2CdiSecProfile(initialTLSSecurityProfile)))

				})
				By("Verify that CNA was properly configured with initialTLSSecurityProfile", func() {
					cna := operands.NewNetworkAddonsWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: cna.Name, Namespace: cna.Namespace},
							cna),
					).To(Succeed())

					Expect(cna.Spec.TLSSecurityProfile).To(Equal(initialTLSSecurityProfile))
				})
				By("Verify that SSP was properly configured with initialTLSSecurityProfile", func() {
					ssp := operands.NewSSPWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: ssp.Name, Namespace: ssp.Namespace},
							ssp),
					).To(Succeed())

					Expect(ssp.Spec.TLSSecurityProfile).To(Equal(initialTLSSecurityProfile))
				})

				// Update ApiServer CR
				apiServer.Spec.TLSSecurityProfile = customTLSSecurityProfile
				Expect(cl.Update(context.TODO(), apiServer)).To(Succeed())
				Expect(hcoutil.GetClusterInfo().GetTLSSecurityProfile(expected.hco.Spec.TLSSecurityProfile)).To(Equal(initialTLSSecurityProfile), "should still return the cached value (initial value)")

				// mock a reconciliation triggered by a change in the APIServer CR
				rq := reqresolver.GetAPIServerCRRequest()

				// Reconcile again to refresh ApiServer CR in memory
				res, err = r.Reconcile(context.TODO(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Requeue).To(BeFalse())
				Expect(res).To(Equal(reconcile.Result{}))

				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundResource),
				).ToNot(HaveOccurred())
				checkAvailability(foundResource, metav1.ConditionTrue)
				Expect(foundResource.Spec.TLSSecurityProfile).To(BeNil(), "TLSSecurityProfile on HCO CR should still be nil")

				Expect(hcoutil.GetClusterInfo().GetTLSSecurityProfile(expected.hco.Spec.TLSSecurityProfile)).To(Equal(customTLSSecurityProfile), "should return the up-to-date value")

				By("Verify that Kubevirt was properly updated with customTLSSecurityProfile", func() {
					kv := operands.NewKubeVirtWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: kv.Name, Namespace: kv.Namespace},
							kv),
					).To(Succeed())

					Expect(kv.Spec.Configuration.TLSConfiguration.MinTLSVersion).To(Equal(kubevirtcorev1.VersionTLS13))
					// it's not possible to specify ciphers when minTLSVersion is 1.3
					Expect(kv.Spec.Configuration.TLSConfiguration.Ciphers).To(BeNil())

				})
				By("Verify that CDI was properly updated with customTLSSecurityProfile", func() {
					cdi := operands.NewCDIWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: cdi.Name, Namespace: cdi.Namespace},
							cdi),
					).To(Succeed())

					Expect(cdi.Spec.Config.TLSSecurityProfile).To(Equal(openshift2CdiSecProfile(customTLSSecurityProfile)))

				})
				By("Verify that CNA was properly updated with customTLSSecurityProfile", func() {
					cna := operands.NewNetworkAddonsWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: cna.Name, Namespace: cna.Namespace},
							cna),
					).To(Succeed())

					Expect(cna.Spec.TLSSecurityProfile).To(Equal(customTLSSecurityProfile))
				})
				By("Verify that SSP was properly updated with customTLSSecurityProfile", func() {
					ssp := operands.NewSSPWithNameOnly(foundResource)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: ssp.Name, Namespace: ssp.Namespace},
							ssp),
					).To(Succeed())

					Expect(ssp.Spec.TLSSecurityProfile).To(Equal(customTLSSecurityProfile))
				})

			})

		})

		Context("Validate OLM required fields", func() {
			var (
				expected  *BasicExpected
				origConds []metav1.Condition
			)

			BeforeEach(func() {
				_ = os.Setenv("VIRTIOWIN_CONTAINER", commontestutils.VirtioWinImage)
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

				Expect(foundResource.Labels[hcoutil.AppLabel]).To(Equal(hcoutil.HyperConvergedName))
			})

			It("Should set required fields when missing", func() {
				expected.hco.Status.Conditions = origConds
				// old HCO Version is set
				cl := expected.initClient()
				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeFalse())

				Expect(foundResource.Labels[hcoutil.AppLabel]).To(Equal(hcoutil.HyperConvergedName))
			})
		})

		Context("Upgrade Mode", func() {
			var (
				oldVersion          string // to be sure to cover v2v CRDs removal during upgrades
				newHCOVersion       string
				oldComponentVersion string
				newComponentVersion string
				expected            *BasicExpected
				origConditions      []metav1.Condition
				okConds             []metav1.Condition
			)

			BeforeEach(func() {
				newHCOVersion = version.Version
				oldComponentVersion = version.Version

				verComp := semver.MustParse(version.Version)
				verComp.Patch += 3
				newComponentVersion = verComp.String()

				verComp = semver.MustParse(version.Version)
				verComp.Minor--
				oldVersion = verComp.String()

				// this is used for version label and the tests below
				// assumes there is no change in labels. Therefore, it should be
				// set before getBasicDeployment so that the existing resource can
				// have the correct labels
				_ = os.Setenv(hcoutil.HcoKvIoVersionName, newHCOVersion)

				expected = getBasicDeployment()
				origConditions = expected.hco.Status.Conditions
				okConds = expected.hco.Status.Conditions

				_ = os.Setenv("VIRTIOWIN_CONTAINER", commontestutils.VirtioWinImage)
				_ = os.Setenv("OPERATOR_NAMESPACE", namespace)

				expected.kv.Status.ObservedKubeVirtVersion = newComponentVersion
				_ = os.Setenv(hcoutil.KubevirtVersionEnvV, newComponentVersion)

				expected.cdi.Status.ObservedVersion = newComponentVersion
				_ = os.Setenv(hcoutil.CdiVersionEnvV, newComponentVersion)

				expected.cna.Status.ObservedVersion = newComponentVersion
				_ = os.Setenv(hcoutil.CnaoVersionEnvV, newComponentVersion)

				_ = os.Setenv(hcoutil.SspVersionEnvV, newComponentVersion)
				expected.ssp.Status.ObservedVersion = newComponentVersion

				_ = os.Setenv(hcoutil.AaqVersionEnvV, newComponentVersion)

				expected.hco.Status.Conditions = origConditions
			})

			It("Should update OperatorCondition Upgradeable to False", func() {
				_ = commontestutils.GetScheme() // ensure the scheme is loaded so this test can be focused

				// old HCO Version is set
				UpdateVersion(&expected.hco.Status, hcoVersionName, oldVersion)

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				r.ownVersion = cmp.Or(os.Getenv(hcoutil.HcoKvIoVersionName), version.Version)

				_, err := r.Reconcile(context.TODO(), request)
				Expect(err).ToNot(HaveOccurred())

				validateOperatorCondition(r, metav1.ConditionFalse, hcoutil.UpgradeableUpgradingReason, hcoutil.UpgradeableUpgradingMessage)
			})

			It("Should update HCO Version Id in the CR on init", func() {

				expected.hco.Status.Conditions = nil

				cl := expected.initClient()
				foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionFalse)

				for _, cond := range foundResource.Status.Conditions {
					if cond.Type == hcov1beta1.ConditionAvailable {
						Expect(cond.Reason).To(Equal("Init"))
						break
					}
				}
				ver, ok := GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(newHCOVersion))

				expected.hco.Status.Conditions = okConds
			})

			It("detect upgrade existing HCO Version", func() {
				// old HCO Version is set
				UpdateVersion(&expected.hco.Status, hcoVersionName, oldVersion)

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()

				cl := expected.initClient()
				foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionFalse)
				// check that the HCO version is not set, because upgrade is not completed
				ver, ok := GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(oldVersion))

				// ensure we are not hot-looping setting the version
				_, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())

				validateOperatorCondition(reconciler, metav1.ConditionFalse, hcoutil.UpgradeableUpgradingReason, hcoutil.UpgradeableUpgradingMessage)

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				cl = expected.initClient()
				foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeTrue())
				checkAvailability(foundResource, metav1.ConditionTrue)

				ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(oldVersion))
				cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionTrue))

				// Call again, to start complete the upgrade
				// check that the image Id is set, now, when upgrade is completed
				foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionTrue)

				ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(newHCOVersion))
				cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				validateOperatorCondition(reconciler, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)

				// Call again, to start complete the upgrade
				// check that the image Id is set, now, when upgrade is completed
				_, _, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				validateOperatorCondition(reconciler, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
			})

			It("don't increase the overwrittenModifications metric during upgrade", func() {
				// old HCO Version is set
				UpdateVersion(&expected.hco.Status, hcoVersionName, oldVersion)

				// CDI is not ready
				expected.cdi.Status.Conditions = getGenericProgressingConditions()
				expected.cdi.Spec.Config.FeatureGates = []string{"fake_feature_gate"}

				cl := expected.initClient()
				r := initReconciler(cl, nil)

				rq := reqresolver.GetSecondaryCRRequest()

				counterValueBefore, err := metrics.GetOverwrittenModificationsCount(expected.cdi.Kind, expected.cdi.Name)
				Expect(err).ToNot(HaveOccurred())

				result, err := r.Reconcile(context.Background(), rq)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Requeue).To(BeTrue())

				foundHC := &hcov1beta1.HyperConverged{}
				Expect(
					cl.Get(context.TODO(),
						types.NamespacedName{Name: expected.hco.Name, Namespace: expected.hco.Namespace},
						foundHC),
				).ToNot(HaveOccurred())

				// check that the HCO version is not set, because upgrade is not completed
				ver, ok := GetVersion(&foundHC.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(oldVersion))

				counterValueAfter, err := metrics.GetOverwrittenModificationsCount(expected.cdi.Kind, expected.cdi.Name)
				Expect(err).ToNot(HaveOccurred())
				Expect(counterValueAfter).To(Equal(counterValueBefore))
			})

			DescribeTable(
				"be tolerant parsing parse version",
				func(testHcoVersion string, acceptableVersion bool, errorMessage string) {
					foundResource := &hcov1beta1.HyperConverged{}
					UpdateVersion(&expected.hco.Status, hcoVersionName, testHcoVersion)

					cl := expected.initClient()

					r := initReconciler(cl, nil)
					r.firstLoop = false
					r.ownVersion = newHCOVersion

					res, err := r.Reconcile(context.TODO(), request)
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: request.Name, Namespace: request.Namespace},
							foundResource),
					).To(Succeed())
					ver, ok := GetVersion(&foundResource.Status, hcoVersionName)

					if acceptableVersion {
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(BeTrue())
						Expect(ok).To(BeTrue())
						Expect(ver).To(Equal(testHcoVersion))
						// reconcile again to complete the upgrade
						res, err = r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(BeFalse())
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: request.Name, Namespace: request.Namespace},
								foundResource),
						).To(Succeed())
						ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
						Expect(ok).To(BeTrue())
						Expect(ver).To(Equal(newHCOVersion))
					} else {
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(errorMessage))
						Expect(res.Requeue).To(BeTrue())
						Expect(ok).To(BeTrue())
						Expect(ver).To(Equal(testHcoVersion))
						// try a second time
						res, err = r.Reconcile(context.TODO(), request)
						Expect(err).To(HaveOccurred())
						Expect(err.Error()).To(ContainSubstring(errorMessage))
						Expect(res.Requeue).To(BeTrue())
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: request.Name, Namespace: request.Namespace},
								foundResource),
						).To(Succeed())
						ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
						Expect(ok).To(BeTrue())
						Expect(ver).To(Equal(testHcoVersion))
						// and a third
						res, err = r.Reconcile(context.TODO(), request)
						Expect(err).To(MatchError(ContainSubstring(errorMessage)))
						Expect(res.Requeue).To(BeTrue())
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: request.Name, Namespace: request.Namespace},
								foundResource),
						).To(Succeed())
						ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
						Expect(ok).To(BeTrue())
						Expect(ver).To(Equal(testHcoVersion))
					}
				},
				Entry(
					"semver",
					"1.11.0",
					true,
					"",
				),
				Entry(
					"semver with leading spaces",
					"  1.11.0",
					true,
					"",
				),
				Entry(
					"semver with trailing spaces",
					"1.11.0  ",
					true,
					"",
				),
				Entry(
					"semver with leading and trailing spaces",
					"  1.11.0  ",
					true,
					"",
				),
				Entry(
					"quasi semver with leading v",
					"  v1.11.0  ",
					true,
					"",
				),
				Entry(
					"quasi semver with leading v",
					"v1.11.0",
					true,
					"",
				),
				Entry(
					"only major and minor",
					"1.11",
					true,
					"",
				),
				Entry(
					"only major",
					"1",
					true,
					"",
				),
				Entry(
					"only major with leading v",
					"v1",
					true,
					"",
				),
				Entry(
					"additional zeros",
					"0000001.0000012.000000",
					true,
					"",
				),
				Entry(
					"negative numbers",
					"-1.7.0",
					false,
					"Invalid character(s) found in major number",
				),
				Entry(
					"additional dots",
					"1...12..0",
					false,
					"invalid syntax",
				),
				Entry(
					"x.y.z",
					"x.y.z",
					false,
					"Invalid character(s) found in",
				),
				Entry(
					"completely broken version",
					"completelyBrokenVersion",
					false,
					"Invalid character(s) found in major number",
				),
			)

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
				ver, ok := GetVersion(&foundResource.Status, hcoVersionName)
				_, _ = fmt.Fprintln(GinkgoWriter, "foundResource.Status.Versions", foundResource.Status.Versions)
				Expect(ok).To(BeFalse())
				Expect(ver).To(BeEmpty())

				// now, complete the upgrade
				expected.cdi.Status.Conditions = getGenericCompletedConditions()
				expected.hco = foundResource
				cl = expected.initClient()
				foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
				Expect(requeue).To(BeFalse())
				checkAvailability(foundResource, metav1.ConditionTrue)

				_, ok = GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionFalse))

				ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
				Expect(ok).To(BeTrue())
				Expect(ver).To(Equal(newHCOVersion))

				cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
				Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionFalse))
			})

			DescribeTable(
				"don't complete upgrade if a component version is not match to the component's version env ver",
				func(makeComponentNotReady, makeComponentReady, updateComponentVersion func()) {
					_ = os.Setenv(hcoutil.HcoKvIoVersionName, newHCOVersion)

					// old HCO Version is set
					UpdateVersion(&expected.hco.Status, hcoVersionName, oldVersion)

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
					ver, ok := GetVersion(&foundResource.Status, hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).To(Equal(oldVersion))
					cond := apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionTrue))
					Expect(cond.Reason).To(Equal("HCOUpgrading"))
					Expect(cond.Message).To(Equal("HCO is now upgrading to version " + newHCOVersion))

					// check that the upgrade is not done if the not all the versions are match.
					// Conditions are valid
					makeComponentReady()

					expected.hco = foundResource
					cl = expected.initClient()
					foundResource, reconciler, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					// check that the image Id is set, now, when upgrade is completed
					ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).To(Equal(oldVersion))
					cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionTrue))
					Expect(cond.Reason).To(Equal("HCOUpgrading"))
					Expect(cond.Message).To(Equal("HCO is now upgrading to version " + newHCOVersion))

					// now, complete the upgrade
					updateComponentVersion()

					expected.hco = foundResource
					cl = expected.initClient()
					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					// check that the image Id is set, now, when upgrade is completed
					ver, ok = GetVersion(&foundResource.Status, hcoVersionName)
					Expect(ok).To(BeTrue())
					Expect(ver).To(Equal(newHCOVersion))
					cond = apimetav1.FindStatusCondition(foundResource.Status.Conditions, hcov1beta1.ConditionProgressing)
					Expect(cond.Status).To(BeEquivalentTo(metav1.ConditionFalse))
					Expect(cond.Reason).To(Equal("ReconcileCompleted"))
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
			)

			Context("Amend bad defaults", func() {
				const (
					badBandwidthPerMigration    = "64Mi"
					customBandwidthPerMigration = "32Mi"
				)

				It("should drop spec.livemigrationconfig.bandwidthpermigration if == 64Mi when upgrading from < 1.5.0", func() {
					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.4.99")
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = ptr.To(badBandwidthPerMigration)

					cl := expected.initClient()
					_, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					foundResource, _, requeue := doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeTrue())
					_, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					Expect(foundResource.Spec.LiveMigrationConfig.BandwidthPerMigration).To(BeNil())
				})

				It("should preserve spec.livemigrationconfig.bandwidthpermigration if != 64Mi when upgrading from < 1.5.0", func() {
					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.4.99")
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = ptr.To(customBandwidthPerMigration)

					cl := expected.initClient()
					_, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					foundResource, _, requeue := doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					Expect(foundResource.Spec.LiveMigrationConfig.BandwidthPerMigration).To(HaveValue(Equal(customBandwidthPerMigration)))
				})

				It("should preserve spec.livemigrationconfig.bandwidthpermigration even if == 64Mi when upgrading from >= 1.5.1", func() {
					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.5.1")
					expected.hco.Spec.LiveMigrationConfig.BandwidthPerMigration = ptr.To(badBandwidthPerMigration)

					cl := expected.initClient()
					_, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					foundResource, _, requeue := doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					Expect(foundResource.Spec.LiveMigrationConfig.BandwidthPerMigration).To(HaveValue(Equal(badBandwidthPerMigration)))
				})
			})

			Context("remove old quickstart guides", func() {
				It("should drop old quickstart guide", func() {
					const oldQSName = "old-quickstart-guide"
					UpdateVersion(&expected.hco.Status, hcoVersionName, oldVersion)

					oldQs := &consolev1.ConsoleQuickStart{
						ObjectMeta: metav1.ObjectMeta{
							Name: oldQSName,
							Labels: map[string]string{
								hcoutil.AppLabel:          expected.hco.Name,
								hcoutil.AppLabelManagedBy: hcoutil.OperatorName,
							},
						},
					}

					kvRef, err := reference.GetReference(commontestutils.GetScheme(), expected.kv)
					Expect(err).ToNot(HaveOccurred())
					Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, *kvRef)).ToNot(HaveOccurred())

					oldQsRef, err := reference.GetReference(commontestutils.GetScheme(), oldQs)
					Expect(err).ToNot(HaveOccurred())
					Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, *oldQsRef)).ToNot(HaveOccurred())

					resources := append(expected.toArray(), oldQs)

					cl := commontestutils.InitClient(resources)
					foundResource, _, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundOldQs := &consolev1.ConsoleQuickStart{
						ObjectMeta: metav1.ObjectMeta{
							Name: "old-quickstart-guide",
						},
					}
					Expect(cl.Get(context.Background(), client.ObjectKeyFromObject(oldQs), foundOldQs)).To(HaveOccurred())

					Expect(searchInRelatedObjects(foundResource.Status.RelatedObjects, "ConsoleQuickStart", oldQSName)).To(BeFalse())
				})

			})

			Context("remove leftovers on upgrades", func() {

				It("should remove ConfigMap v2v-vmware upgrading from <= 1.6.0", func() {

					cmToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "v2v-vmware",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					cmToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "vm-import-controller-config",
							Namespace: namespace,
						},
					}
					cmNotToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "v2v-vmware",
							Namespace: "different" + namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					cmNotToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					toBeRemovedRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmToBeRemoved1.Name,
							Namespace:       cmToBeRemoved1.Namespace,
							ResourceVersion: "999",
						},
					}
					otherRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmNotToBeRemoved1.Name,
							Namespace:       cmNotToBeRemoved1.Namespace,
							ResourceVersion: "999",
						},
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmNotToBeRemoved2.Name,
							Namespace:       cmNotToBeRemoved2.Namespace,
							ResourceVersion: "999",
						},
					}

					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.4.99")

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}
					for _, objRef := range otherRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}

					resources := append(expected.toArray(), cmToBeRemoved1, cmToBeRemoved2, cmNotToBeRemoved1, cmNotToBeRemoved2)

					cl := commontestutils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundCM := &corev1.ConfigMap{}

					err := cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved1), foundCM)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					err = cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved2), foundCM)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved2), foundCM)).To(Succeed())

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).ToNot(ContainElement(objRef))
					}
					for _, objRef := range otherRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).To(ContainElement(objRef))
					}

				})

				It("should not remove ConfigMap v2v-vmware upgrading from >= 1.6.1", func() {

					cmToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "v2v-vmware",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					cmToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "vm-import-controller-config",
							Namespace: namespace,
						},
					}
					cmNotToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "v2v-vmware",
							Namespace: "different" + namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					cmNotToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.6.1")

					resources := append(expected.toArray(), cmToBeRemoved1, cmToBeRemoved2, cmNotToBeRemoved1, cmNotToBeRemoved2)

					cl := commontestutils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundCM := &corev1.ConfigMap{}
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved2), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved2), foundCM)).To(Succeed())
				})

				It("should remove ConfigMap kubevirt-storage-class-defaults upgrading from < 1.7.0", func() {
					cmToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kubevirt-storage-class-defaults",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					roleToBeRemoved := &rbacv1.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hco.kubevirt.io:config-reader",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					roleBindingToBeRemoved := &rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hco.kubevirt.io:config-reader",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					cmNotToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kubevirt-storage-class-defaults",
							Namespace: "different" + namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					cmNotToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					toBeRemovedRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmToBeRemoved1.Name,
							Namespace:       cmToBeRemoved1.Namespace,
							ResourceVersion: "999",
						},
						{
							APIVersion:      "rbac.authorization.k8s.io/v1",
							Kind:            "Role",
							Name:            roleToBeRemoved.Name,
							Namespace:       roleToBeRemoved.Namespace,
							ResourceVersion: "999",
						},
						{
							APIVersion:      "rbac.authorization.k8s.io/v1",
							Kind:            "RoleBinding",
							Name:            roleBindingToBeRemoved.Name,
							Namespace:       roleBindingToBeRemoved.Namespace,
							ResourceVersion: "999",
						},
					}
					otherRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmNotToBeRemoved1.Name,
							Namespace:       cmNotToBeRemoved1.Namespace,
							ResourceVersion: "999",
						},
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmNotToBeRemoved2.Name,
							Namespace:       cmNotToBeRemoved2.Namespace,
							ResourceVersion: "999",
						},
					}

					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.6.9")

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}
					for _, objRef := range otherRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}

					resources := append(expected.toArray(), cmToBeRemoved1, roleToBeRemoved, roleBindingToBeRemoved, cmNotToBeRemoved1, cmNotToBeRemoved2)

					cl := commontestutils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundCM := &corev1.ConfigMap{}
					foundRole := &rbacv1.Role{}
					foundRoleBinding := &rbacv1.RoleBinding{}

					err := cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved1), foundCM)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					err = cl.Get(context.TODO(), client.ObjectKeyFromObject(roleToBeRemoved), foundRole)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					err = cl.Get(context.TODO(), client.ObjectKeyFromObject(roleBindingToBeRemoved), foundRoleBinding)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved2), foundCM)).To(Succeed())

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).ToNot(ContainElement(objRef))
					}
					for _, objRef := range otherRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).To(ContainElement(objRef))
					}

				})

				It("should not remove ConfigMap kubevirt-storage-class-defaults upgrading from > 1.7.0", func() {
					cmToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kubevirt-storage-class-defaults",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					roleToBeRemoved := &rbacv1.Role{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "hco.kubevirt.io:config-reader",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}
					roleBindingToBeRemoved := &rbacv1.RoleBinding{
						ObjectMeta: metav1.ObjectMeta{
							Name: "hco.kubevirt.io:config-reader",
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
							Namespace: namespace,
						},
					}
					cmNotToBeRemoved1 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "kubevirt-storage-class-defaults",
							Namespace: "different" + namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					cmNotToBeRemoved2 := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.7.1")

					resources := append(expected.toArray(), cmToBeRemoved1, roleToBeRemoved, roleBindingToBeRemoved, cmNotToBeRemoved1, cmNotToBeRemoved2)

					cl := commontestutils.InitClient(resources)
					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundCM := &corev1.ConfigMap{}
					foundRole := &rbacv1.Role{}
					foundRoleBinding := &rbacv1.RoleBinding{}

					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(roleToBeRemoved), foundRole)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(roleBindingToBeRemoved), foundRoleBinding)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved1), foundCM)).To(Succeed())
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved2), foundCM)).To(Succeed())
				})

				It("should remove TTO CRD upgrading from < 1.10.0", func() {
					crdToBeRemoved := &apiextensionsv1.CustomResourceDefinition{
						ObjectMeta: metav1.ObjectMeta{
							Name: "tektontasks.tektontasks.kubevirt.io",
						},
					}

					cmNotToBeRemoved := &corev1.ConfigMap{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other",
							Namespace: namespace,
							Labels: map[string]string{
								hcoutil.AppLabel: expected.hco.Name,
							},
						},
					}

					toBeRemovedRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "tektontasks.kubevirt.io/v1alpha1",
							Kind:            "TektonTasks",
							Name:            "tto-kubevirt-hyperconverged",
							Namespace:       "kubevirt-hyperconverged",
							ResourceVersion: "999",
						},
					}
					otherRelatedObjects := []corev1.ObjectReference{
						{
							APIVersion:      "v1",
							Kind:            "ConfigMap",
							Name:            cmNotToBeRemoved.Name,
							Namespace:       cmNotToBeRemoved.Namespace,
							ResourceVersion: "999",
						},
					}

					UpdateVersion(&expected.hco.Status, hcoVersionName, "1.9.0")

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}
					for _, objRef := range otherRelatedObjects {
						Expect(objectreferencesv1.SetObjectReference(&expected.hco.Status.RelatedObjects, objRef)).ToNot(HaveOccurred())
					}

					resources := append(expected.toArray(), crdToBeRemoved, cmNotToBeRemoved)

					cl := commontestutils.InitClient(resources)
					restMapper := cl.RESTMapper()
					Expect(restMapper).To(Not(BeNil()))
					dRestMapper := restMapper.(*apimetav1.DefaultRESTMapper)
					dRestMapper.AddSpecific(
						schema.GroupVersionKind{Group: "tektontasks.kubevirt.io", Version: "v1alpha1", Kind: "TektonTasks"},
						schema.GroupVersionResource{Group: "tektontasks.kubevirt.io", Version: "v1alpha1", Resource: "tektontasks"},
						schema.GroupVersionResource{Group: "tektontasks.kubevirt.io", Version: "v1alpha1", Resource: "tektontask"},
						apimetav1.RESTScopeNamespace)

					foundResource, reconciler, requeue := doReconcile(cl, expected.hco, nil)
					Expect(requeue).To(BeTrue())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundResource, _, requeue = doReconcile(cl, expected.hco, reconciler)
					Expect(requeue).To(BeFalse())
					checkAvailability(foundResource, metav1.ConditionTrue)

					foundCRD := &apiextensionsv1.CustomResourceDefinition{}

					err := cl.Get(context.TODO(), client.ObjectKeyFromObject(crdToBeRemoved), foundCRD)
					Expect(err).To(MatchError(apierrors.IsNotFound, "not found error"))

					foundCM := &corev1.ConfigMap{}
					Expect(cl.Get(context.TODO(), client.ObjectKeyFromObject(cmNotToBeRemoved), foundCM)).To(Succeed())

					for _, objRef := range toBeRemovedRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).ToNot(ContainElement(objRef))
					}
					for _, objRef := range otherRelatedObjects {
						Expect(foundResource.Status.RelatedObjects).To(ContainElement(objRef))
					}

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
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(commonDegradedReason))
				Expect(cd.Message).To(Equal("HCO is not available due to degraded components"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(commonDegradedReason))
				Expect(cd.Message).To(Equal("HCO is not Upgradeable due to degraded components"))

				By("operator condition should be true even the upgradeable is false")
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
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
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(commonDegradedReason))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDIProgressing"))

				By("operator condition should be true even the upgradeable is false")
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
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
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIDegraded"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(commonDegradedReason))

				By("operator condition should be true even the upgradeable is false")
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
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
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDIProgressing"))

				By("operator condition should be true even the upgradeable is false")
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
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
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				_, _ = fmt.Fprintln(GinkgoWriter, "\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIProgressing"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDIProgressing"))

				By("operator condition should be true even the upgradeable is false")
				validateOperatorCondition(r, metav1.ConditionTrue, hcoutil.UpgradeableAllowReason, hcoutil.UpgradeableAllowMessage)
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
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
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
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
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
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("NetworkAddonsConfigNotAvailable"))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))
			})

			It("should not be upgradeable when a component is not upgradeable", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})
				cl := expected.initClient()
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				GinkgoWriter.Println("\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotUpgradeable"))
				Expect(cd.Message).To(Equal("CDI is not upgradeable: CDI Test Error message"))

				By("operator condition should be false")
				validateOperatorCondition(r, metav1.ConditionFalse, "CDINotUpgradeable", "is not upgradeable:")
			})

			It("should not be with its own reason and message if a component is not upgradeable, even if there are it also progressing", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  errorReason,
					Message: "CDI Upgrade Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionProgressing,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})

				cl := expected.initClient()
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				GinkgoWriter.Println("\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIProgressing"))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotUpgradeable"))
				Expect(cd.Message).To(Equal("CDI is not upgradeable: CDI Upgrade Error message"))

				By("operator condition should be false")
				validateOperatorCondition(r, metav1.ConditionFalse, "CDINotUpgradeable", "is not upgradeable:")
			})

			It("should not be with its own reason and message if a component is not upgradeable, even if there are it also degraded", func() {
				expected := getBasicDeployment()
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionUpgradeable,
					Status:  corev1.ConditionFalse,
					Reason:  errorReason,
					Message: "CDI Upgrade Error message",
				})
				conditionsv1.SetStatusCondition(&expected.cdi.Status.Conditions, conditionsv1.Condition{
					Type:    conditionsv1.ConditionDegraded,
					Status:  corev1.ConditionTrue,
					Reason:  errorReason,
					Message: "CDI Test Error message",
				})

				cl := expected.initClient()
				foundResource, r, _ := doReconcile(cl, expected.hco, nil)

				conditions := foundResource.Status.Conditions
				GinkgoWriter.Println("\nActual Conditions:")
				wr := json.NewEncoder(GinkgoWriter)
				wr.SetIndent("", "  ")
				_ = wr.Encode(conditions)

				cd := apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionReconcileComplete)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionAvailable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(commonDegradedReason))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionProgressing)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal(reconcileCompleted))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionDegraded)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionTrue))
				Expect(cd.Reason).To(Equal("CDIDegraded"))

				cd = apimetav1.FindStatusCondition(conditions, hcov1beta1.ConditionUpgradeable)
				Expect(cd.Status).To(BeEquivalentTo(metav1.ConditionFalse))
				Expect(cd.Reason).To(Equal("CDINotUpgradeable"))
				Expect(cd.Message).To(Equal("CDI is not upgradeable: CDI Upgrade Error message"))

				By("operator condition should be false")
				validateOperatorCondition(r, metav1.ConditionFalse, "CDINotUpgradeable", "is not upgradeable:")
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

				r.ownVersion = cmp.Or(os.Getenv(hcoutil.HcoKvIoVersionName), version.Version)

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(MatchError(apierrors.IsConflict, "conflict error"))
				Expect(res.Requeue).To(BeTrue())
			})

			It("Should requeue in case of update status conflict", func() {
				expected := getBasicDeployment()
				expected.hco.Status.Conditions = nil
				cl := expected.initClient()
				rs := schema.GroupResource{Group: hcoutil.APIVersionGroup, Resource: "hyperconvergeds.hco.kubevirt.io"}
				cl.Status().(*commontestutils.HcoTestStatusWriter).InitiateErrors(apierrors.NewConflict(rs, "hco", errors.New("test error")))
				r := initReconciler(cl, nil)

				r.ownVersion = cmp.Or(os.Getenv(hcoutil.HcoKvIoVersionName), version.Version)

				res, err := r.Reconcile(context.TODO(), request)
				Expect(err).To(MatchError(apierrors.IsConflict, "conflict error"))
				Expect(res.Requeue).To(BeTrue())

			})
		})

		Context("Detection of a tainted configuration", func() {
			var (
				hcoNamespace *corev1.Namespace
				hco          *hcov1beta1.HyperConverged
			)
			BeforeEach(func() {
				hcoNamespace = commontestutils.NewHcoNamespace()
				hco = commontestutils.NewHco()
				UpdateVersion(&hco.Status, hcoVersionName, version.Version)
				_ = os.Setenv(hcoutil.HcoKvIoVersionName, version.Version)
			})

			Context("Detection of a tainted configuration for kubevirt", func() {

				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco.Annotations = map[string]string{
						common.JSONPatchKVAnnotationName: `
						[
							{
								"op": "add",
								"path": "/spec/configuration/migrations",
								"value": {"allowPostCopy": true}
							}
						]`,
					}
					metrics.SetUnsafeModificationCount(0, common.JSONPatchKVAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
						).To(Succeed())

						Expect(kv.Spec.Configuration.MigrationConfiguration).ToNot(BeNil())
						Expect(kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).ToNot(BeNil())
						Expect(*kv.Spec.Configuration.MigrationConfiguration.AllowPostCopy).To(BeTrue())
					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.SetUnsafeModificationCount(5, common.JSONPatchKVAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).ToNot(HaveOccurred())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res).To(Equal(reconcile.Result{Requeue: false}))

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.SetUnsafeModificationCount(5, common.JSONPatchKVAnnotationName)

					hco.Annotations = map[string]string{
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

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(BeFalse())
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
					hco.Annotations = map[string]string{
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

					metrics.SetUnsafeModificationCount(0, common.JSONPatchCDIAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
						).To(Succeed())

						Expect(cdi.Spec.Config.FeatureGates).To(ContainElement("fg1"))
						Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.Global).To(BeEquivalentTo("50"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).To(BeEquivalentTo("75"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).To(BeEquivalentTo("25"))

					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.SetUnsafeModificationCount(5, common.JSONPatchCDIAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).ToNot(HaveOccurred())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res.Requeue).To(BeFalse())

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})

					metrics.SetUnsafeModificationCount(5, common.JSONPatchCDIAnnotationName)

					hco.Annotations = map[string]string{
						// Set bad json format (missing comma)
						common.JSONPatchKVAnnotationName: `[{`,
					}

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res.Requeue).To(BeFalse())
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
					hco.Annotations = map[string]string{
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

					metrics.SetUnsafeModificationCount(0, common.JSONPatchCNAOAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
						).To(Succeed())

						Expect(cna.Spec.KubeMacPool).ToNot(BeNil())
						Expect(cna.Spec.KubeMacPool.RangeStart).To(Equal("1.1.1.1.1.1"))
						Expect(cna.Spec.KubeMacPool.RangeEnd).To(Equal("5.5.5.5.5.5"))
						Expect(cna.Spec.ImagePullPolicy).To(BeEquivalentTo("Always"))
					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})
					metrics.SetUnsafeModificationCount(5, common.JSONPatchCNAOAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).ToNot(HaveOccurred())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res.Requeue).To(BeFalse())

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
					hco.Annotations = map[string]string{
						// Set bad json
						common.JSONPatchKVAnnotationName: `[{`,
					}
					metrics.SetUnsafeModificationCount(5, common.JSONPatchCNAOAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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

			Context("Detection of a tainted configuration for SSP", func() {

				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco.Annotations = map[string]string{
						common.JSONPatchSSPAnnotationName: `[
							{
								"op": "replace",
								"path": "/spec/templateValidator/replicas",
								"value": 5
							}
						]`,
					}

					metrics.SetUnsafeModificationCount(0, common.JSONPatchSSPAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
							Type:    hcov1beta1.ConditionTaintedConfiguration,
							Status:  metav1.ConditionTrue,
							Reason:  taintedConfigurationReason,
							Message: taintedConfigurationMessage,
						})))
					})

					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(1, common.JSONPatchSSPAnnotationName)
					})

					By("Verify that SSP was modified by the annotation", func() {
						ssp := operands.NewSSPWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: ssp.Name, Namespace: ssp.Namespace},
								ssp),
						).To(Succeed())

						Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
						Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(5)))
					})
				})

				It("Removes the TaintedConfiguration condition upon removal of such configuration", func() {
					hco.Status.Conditions = append(hco.Status.Conditions, metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					})
					metrics.SetUnsafeModificationCount(5, common.JSONPatchSSPAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					// Do the reconcile
					res, err := r.Reconcile(context.TODO(), request)
					Expect(err).ToNot(HaveOccurred())

					// Expecting "Requeue: false" since the conditions aren't empty
					Expect(res.Requeue).To(BeFalse())

					// Get the HCO
					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchSSPAnnotationName)
					})
				})

				It("Removes the TaintedConfiguration condition if the annotation is wrong", func() {
					hco.Annotations = map[string]string{
						// Set bad json
						common.JSONPatchSSPAnnotationName: `[{`,
					}
					metrics.SetUnsafeModificationCount(5, common.JSONPatchSSPAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					// Check conditions
					Expect(foundResource.Status.Conditions).To(Not(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
						Type:    hcov1beta1.ConditionTaintedConfiguration,
						Status:  metav1.ConditionTrue,
						Reason:  taintedConfigurationReason,
						Message: taintedConfigurationMessage,
					}))))
					By("verify that the metrics match to the annotation", func() {
						verifyUnsafeMetrics(0, common.JSONPatchSSPAnnotationName)
					})
				})
			})

			Context("Detection of a tainted configuration for all the annotations", func() {
				It("Raises a TaintedConfiguration condition upon detection of such configuration", func() {
					hco.Annotations = map[string]string{
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
						common.JSONPatchSSPAnnotationName: `[
							{
								"op": "replace",
								"path": "/spec/templateValidator/replicas",
								"value": 5
							}
						]`,
					}
					metrics.SetUnsafeModificationCount(0, common.JSONPatchKVAnnotationName)
					metrics.SetUnsafeModificationCount(0, common.JSONPatchCDIAnnotationName)
					metrics.SetUnsafeModificationCount(0, common.JSONPatchCNAOAnnotationName)
					metrics.SetUnsafeModificationCount(0, common.JSONPatchSSPAnnotationName)

					cl := commontestutils.InitClient([]client.Object{hcoNamespace, hco})
					r := initReconciler(cl, nil)

					By("Reconcile", func() {
						res, err := r.Reconcile(context.TODO(), request)
						Expect(err).ToNot(HaveOccurred())
						Expect(res).To(Equal(reconcile.Result{Requeue: true}))
					})

					foundResource := &hcov1beta1.HyperConverged{}
					Expect(
						cl.Get(context.TODO(),
							types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
							foundResource),
					).To(Succeed())

					By("Verify HC conditions", func() {
						Expect(foundResource.Status.Conditions).To(ContainElement(commontestutils.RepresentCondition(metav1.Condition{
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
						verifyUnsafeMetrics(1, common.JSONPatchSSPAnnotationName)
					})

					By("Verify that KV was modified by the annotation", func() {
						kv := operands.NewKubeVirtWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: kv.Name, Namespace: kv.Namespace},
								kv),
						).To(Succeed())

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
						).To(Succeed())

						Expect(cdi.Spec.Config.FeatureGates).To(ContainElement("fg1"))
						Expect(cdi.Spec.Config.FilesystemOverhead).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.Global).To(BeEquivalentTo("50"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass).ToNot(BeNil())
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["AAA"]).To(BeEquivalentTo("75"))
						Expect(cdi.Spec.Config.FilesystemOverhead.StorageClass["BBB"]).To(BeEquivalentTo("25"))

					})
					By("Verify that CNA was modified by the annotation", func() {
						cna := operands.NewNetworkAddonsWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: cna.Name, Namespace: cna.Namespace},
								cna),
						).To(Succeed())

						Expect(cna.Spec.KubeMacPool).ToNot(BeNil())
						Expect(cna.Spec.KubeMacPool.RangeStart).To(Equal("1.1.1.1.1.1"))
						Expect(cna.Spec.KubeMacPool.RangeEnd).To(Equal("5.5.5.5.5.5"))
						Expect(cna.Spec.ImagePullPolicy).To(BeEquivalentTo("Always"))
					})
					By("Verify that SSP was modified by the annotation", func() {
						ssp := operands.NewSSPWithNameOnly(hco)
						Expect(
							cl.Get(context.TODO(),
								types.NamespacedName{Name: ssp.Name, Namespace: ssp.Namespace},
								ssp),
						).To(Succeed())

						Expect(ssp.Spec.TemplateValidator.Replicas).ToNot(BeNil())
						Expect(*ssp.Spec.TemplateValidator.Replicas).To(Equal(int32(5)))
					})
				})
			})
		})

	})
})

func verifyUnsafeMetrics(expected int, annotation string) {
	count, err := metrics.GetUnsafeModificationsCount(annotation)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, count).To(BeEquivalentTo(expected))
}

func verifyHyperConvergedCRExistsMetricTrue() {
	hcExists, err := metrics.IsHCOMetricHyperConvergedExists()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, hcExists).To(BeTrue())
}

func verifyHyperConvergedCRExistsMetricFalse() {
	hcExists, err := metrics.IsHCOMetricHyperConvergedExists()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, hcExists).To(BeFalse())
}

func verifySystemHealthStatusHealthy(hco *hcov1beta1.HyperConverged) {
	ExpectWithOffset(1, hco.Status.SystemHealthStatus).To(Equal(systemHealthStatusHealthy))

	systemHealthStatusMetric, err := metrics.GetHCOMetricSystemHealthStatus()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, systemHealthStatusMetric).To(Equal(metrics.SystemHealthStatusHealthy))
}

func verifySystemHealthStatusError(hco *hcov1beta1.HyperConverged) {
	ExpectWithOffset(1, hco.Status.SystemHealthStatus).To(Equal(systemHealthStatusError))

	systemHealthStatusMetric, err := metrics.GetHCOMetricSystemHealthStatus()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	ExpectWithOffset(1, systemHealthStatusMetric).To(Equal(metrics.SystemHealthStatusError))
}

func searchInRelatedObjects(relatedObjects []corev1.ObjectReference, kind, name string) bool {
	for _, obj := range relatedObjects {
		if obj.Kind == kind && obj.Name == name {
			return true
		}
	}
	return false
}

func openshift2CdiSecProfile(hcProfile *openshiftconfigv1.TLSSecurityProfile) *cdiv1beta1.TLSSecurityProfile {
	var custom *cdiv1beta1.CustomTLSProfile
	if hcProfile.Custom != nil {
		custom = &cdiv1beta1.CustomTLSProfile{
			TLSProfileSpec: cdiv1beta1.TLSProfileSpec{
				Ciphers:       hcProfile.Custom.Ciphers,
				MinTLSVersion: cdiv1beta1.TLSProtocolVersion(hcProfile.Custom.MinTLSVersion),
			},
		}
	}

	return &cdiv1beta1.TLSSecurityProfile{
		Type:         cdiv1beta1.TLSProfileType(hcProfile.Type),
		Old:          (*cdiv1beta1.OldTLSProfile)(hcProfile.Old),
		Intermediate: (*cdiv1beta1.IntermediateTLSProfile)(hcProfile.Intermediate),
		Modern:       (*cdiv1beta1.ModernTLSProfile)(hcProfile.Modern),
		Custom:       custom,
	}
}
