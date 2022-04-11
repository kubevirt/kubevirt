package hyperconverged

import (
	"context"
	"fmt"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	consolev1 "github.com/openshift/api/console/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	operatorsapiv2 "github.com/operator-framework/api/pkg/operators/v2"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/commonTestUtils"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/operands"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
)

// Mock TestRequest to simulate Reconcile() being called on an event for a watched resource
var (
	request = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
)

func initReconciler(client client.Client, old *ReconcileHyperConverged) *ReconcileHyperConverged {
	s := commonTestUtils.GetScheme()
	eventEmitter := commonTestUtils.NewEventEmitterMock()
	ci := commonTestUtils.ClusterInfoMock{}
	operandHandler := operands.NewOperandHandler(client, s, ci, eventEmitter)
	upgradeMode := false
	firstLoop := true
	upgradeableCondition := newStubOperatorCondition()
	if old != nil {
		upgradeMode = old.upgradeMode
		firstLoop = old.firstLoop
		upgradeableCondition = old.upgradeableCondition
	}
	// Create a ReconcileHyperConverged object with the scheme and fake client
	return &ReconcileHyperConverged{
		client:               client,
		scheme:               s,
		operandHandler:       operandHandler,
		eventEmitter:         eventEmitter,
		firstLoop:            firstLoop,
		ownVersion:           version.Version,
		upgradeMode:          upgradeMode,
		upgradeableCondition: upgradeableCondition,
	}
}

type stubCondition struct {
	condition *metav1.Condition
}

func newStubOperatorCondition() hcoutil.Condition {
	cond := &stubCondition{}

	return cond
}

func (nc *stubCondition) Set(_ context.Context, status metav1.ConditionStatus, reason, message string) error {
	nc.condition = &metav1.Condition{
		Type:    operatorsapiv2.Upgradeable,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	return nil
}

func (nc *stubCondition) validate(status metav1.ConditionStatus, reason, message string) {
	ExpectWithOffset(2, nc.condition).ToNot(BeNil())
	ExpectWithOffset(2, nc.condition.Status).To(Equal(status))
	ExpectWithOffset(2, nc.condition.Reason).To(Equal(reason))
	ExpectWithOffset(2, nc.condition.Message).To(ContainSubstring(message))
}

func validateOperatorCondition(r *ReconcileHyperConverged, status metav1.ConditionStatus, reason, message string) {
	cond := r.upgradeableCondition.(*stubCondition)
	cond.validate(status, reason, message)
}

type BasicExpected struct {
	namespace            *corev1.Namespace
	hco                  *hcov1beta1.HyperConverged
	pc                   *schedulingv1.PriorityClass
	kv                   *kubevirtcorev1.KubeVirt
	cdi                  *cdiv1beta1.CDI
	cna                  *networkaddonsv1.NetworkAddonsConfig
	ssp                  *sspv1beta1.SSP
	mService             *corev1.Service
	serviceMonitor       *monitoringv1.ServiceMonitor
	promRule             *monitoringv1.PrometheusRule
	cliDownload          *consolev1.ConsoleCLIDownload
	cliDownloadsRoute    *routev1.Route
	cliDownloadsService  *corev1.Service
	virtioWinConfig      *corev1.ConfigMap
	virtioWinRole        *rbacv1.Role
	virtioWinRoleBinding *rbacv1.RoleBinding
	hcoCRD               *apiextensionsv1.CustomResourceDefinition
	consolePluginDeploy  *appsv1.Deployment
	consolePluginSvc     *corev1.Service
	consolePlugin        *consolev1alpha1.ConsolePlugin
	consoleConfig        *operatorv1.Console
}

func (be BasicExpected) toArray() []runtime.Object {
	return []runtime.Object{
		be.namespace,
		be.hco,
		be.pc,
		be.kv,
		be.cdi,
		be.cna,
		be.ssp,
		be.mService,
		be.serviceMonitor,
		be.promRule,
		be.cliDownload,
		be.cliDownloadsRoute,
		be.cliDownloadsService,
		be.virtioWinConfig,
		be.virtioWinRole,
		be.virtioWinRoleBinding,
		be.hcoCRD,
		be.consolePluginDeploy,
		be.consolePluginSvc,
		be.consolePlugin,
		be.consoleConfig,
	}
}

func (be BasicExpected) initClient() *commonTestUtils.HcoTestClient {
	return commonTestUtils.InitClient(be.toArray())
}

func getBasicDeployment() *BasicExpected {

	res := &BasicExpected{}

	hco := &hcov1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{hcoutil.AppLabel: name},
		},
		Spec: hcov1beta1.HyperConvergedSpec{},
		Status: hcov1beta1.HyperConvergedStatus{
			Conditions: []metav1.Condition{
				{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  metav1.ConditionTrue,
					Reason:  common.ReconcileCompleted,
					Message: common.ReconcileCompletedMessage,
				},
			},
			Versions: []hcov1beta1.Version{
				{
					Name:    hcoVersionName,
					Version: version.Version,
				},
			},
		},
	}
	res.hco = hco

	components.GetOperatorCR().Spec.DeepCopyInto(&res.hco.Spec)

	res.namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: hco.Namespace,
			Annotations: map[string]string{
				hcoutil.OpenshiftNodeSelectorAnn: "",
			},
		},
	}

	res.pc = operands.NewKubeVirtPriorityClass(hco)
	res.mService = operands.NewMetricsService(hco)
	res.serviceMonitor = operands.NewServiceMonitor(hco, namespace)
	res.promRule = operands.NewPrometheusRule(hco, namespace)

	expectedKV, err := operands.NewKubeVirt(hco, namespace)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	expectedKV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/kubevirts/%s", expectedKV.Namespace, expectedKV.Name)
	expectedKV.Status.Conditions = []kubevirtcorev1.KubeVirtCondition{
		{
			Type:   kubevirtcorev1.KubeVirtConditionAvailable,
			Status: corev1.ConditionTrue,
		},
		{
			Type:   kubevirtcorev1.KubeVirtConditionProgressing,
			Status: corev1.ConditionFalse,
		},
		{
			Type:   kubevirtcorev1.KubeVirtConditionDegraded,
			Status: corev1.ConditionFalse,
		},
	}
	res.kv = expectedKV

	expectedCDI, err := operands.NewCDI(hco)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
	expectedCDI.Status.Conditions = getGenericCompletedConditions()
	res.cdi = expectedCDI

	expectedCNA, err := operands.NewNetworkAddons(hco)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
	expectedCNA.Status.Conditions = getGenericCompletedConditions()
	res.cna = expectedCNA

	expectedSSP, _, err := operands.NewSSP(hco)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	expectedSSP.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedSSP.Namespace, expectedSSP.Name)
	expectedSSP.Status.Conditions = getGenericCompletedConditions()
	res.ssp = expectedSSP

	expectedCliDownload := operands.NewConsoleCLIDownload(hco)
	expectedCliDownload.SelfLink = fmt.Sprintf("/apis/console.openshift.io/v1/consoleclidownloads/%s", expectedCliDownload.Name)
	res.cliDownload = expectedCliDownload

	expectedCliDownloadsRoute := operands.NewCliDownloadsRoute(hco)
	expectedCliDownloadsRoute.SelfLink = fmt.Sprintf("/apis/route.openshift.io/v1/namespaces/%s/routes/%s", expectedCliDownloadsRoute.Namespace, expectedCliDownloadsRoute.Name)
	res.cliDownloadsRoute = expectedCliDownloadsRoute

	expectedCliDownloadsService := operands.NewCliDownloadsService(hco)
	expectedCliDownloadsService.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/services/%s", expectedCliDownloadsService.Namespace, expectedCliDownloadsService.Name)
	res.cliDownloadsService = expectedCliDownloadsService

	expectedVirtioWinConfig, err := operands.NewVirtioWinCm(hco)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	res.virtioWinConfig = expectedVirtioWinConfig

	expectedVirtioWinRole := operands.NewVirtioWinCmReaderRole(hco)
	res.virtioWinRole = expectedVirtioWinRole

	expectedVirtioWinRoleBinding := operands.NewVirtioWinCmReaderRoleBinding(hco)
	res.virtioWinRoleBinding = expectedVirtioWinRoleBinding

	expectedConsolePluginDeployment, err := operands.NewKvUiPluginDeplymnt(hco)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	res.consolePluginDeploy = expectedConsolePluginDeployment

	expectedConsolePluginService := operands.NewKvUiPluginSvc(hco)
	res.consolePluginSvc = expectedConsolePluginService

	expectedConsolePlugin := operands.NewKvConsolePlugin(hco)
	res.consolePlugin = expectedConsolePlugin

	expectedConsoleConfig := &operatorv1.Console{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Console",
			APIVersion: "operator.openshift.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
	}
	res.consoleConfig = expectedConsoleConfig

	hcoCrd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hyperconvergeds.hco.kubevirt.io",
		},
	}
	res.hcoCRD = hcoCrd

	return res
}

// returns the HCO after reconcile, and the returned requeue
func doReconcile(cl client.Client, hco *hcov1beta1.HyperConverged, old *ReconcileHyperConverged) (*hcov1beta1.HyperConverged, *ReconcileHyperConverged, bool) {
	r := initReconciler(cl, old)

	r.firstLoop = false
	r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
	if r.ownVersion == "" {
		r.ownVersion = version.Version
	}

	res, err := r.Reconcile(context.TODO(), request)
	Expect(err).ToNot(HaveOccurred())

	foundResource := &hcov1beta1.HyperConverged{}
	Expect(
		cl.Get(context.TODO(),
			types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
			foundResource),
	).ToNot(HaveOccurred())

	return foundResource, r, res.Requeue
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

func checkAvailability(hco *hcov1beta1.HyperConverged, expected metav1.ConditionStatus) {
	found := false
	for _, cond := range hco.Status.Conditions {
		if cond.Type == hcov1beta1.ConditionAvailable {
			found = true
			ExpectWithOffset(1, cond.Status).To(Equal(expected))
			break
		}
	}

	if !found {
		Fail(fmt.Sprintf(`Can't find 'Available' condition; %v`, hco.Status.Conditions))
	}
}
