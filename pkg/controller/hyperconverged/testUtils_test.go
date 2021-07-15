package hyperconverged

import (
	"context"
	"fmt"
	consolev1 "github.com/openshift/api/console/v1"
	"os"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
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
	operandHandler := operands.NewOperandHandler(client, s, true, eventEmitter)
	upgradeMode := false
	firstLoop := true
	if old != nil {
		upgradeMode = old.upgradeMode
		firstLoop = old.firstLoop
	}
	// Create a ReconcileHyperConverged object with the scheme and fake client
	return &ReconcileHyperConverged{
		client:         client,
		scheme:         s,
		operandHandler: operandHandler,
		eventEmitter:   eventEmitter,
		firstLoop:      firstLoop,
		ownVersion:     version.Version,
		upgradeMode:    upgradeMode,
	}
}

type BasicExpected struct {
	hco                  *hcov1beta1.HyperConverged
	pc                   *schedulingv1.PriorityClass
	kvStorageConfig      *corev1.ConfigMap
	kvStorageRole        *rbacv1.Role
	kvStorageRoleBinding *rbacv1.RoleBinding
	kv                   *kubevirtv1.KubeVirt
	cdi                  *cdiv1beta1.CDI
	cna                  *networkaddonsv1.NetworkAddonsConfig
	ssp                  *sspv1beta1.SSP
	vmi                  *vmimportv1beta1.VMImportConfig
	imsConfig            *corev1.ConfigMap
	mService             *corev1.Service
	serviceMonitor       *monitoringv1.ServiceMonitor
	promRule             *monitoringv1.PrometheusRule
	cliDownload          *consolev1.ConsoleCLIDownload
}

func (be BasicExpected) toArray() []runtime.Object {
	return []runtime.Object{
		be.hco,
		be.pc,
		be.kvStorageConfig,
		be.kvStorageRole,
		be.kvStorageRoleBinding,
		be.kv,
		be.cdi,
		be.cna,
		be.ssp,
		be.vmi,
		be.imsConfig,
		be.mService,
		be.serviceMonitor,
		be.promRule,
		be.cliDownload,
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
			Versions: hcov1beta1.Versions{
				{Name: hcoVersionName, Version: version.Version},
			},
		},
	}
	res.hco = hco

	res.pc = operands.NewKubeVirtPriorityClass(hco)
	res.mService = operands.NewMetricsService(hco, namespace)
	res.serviceMonitor = operands.NewServiceMonitor(hco, namespace)
	res.promRule = operands.NewPrometheusRule(hco, namespace)

	expectedKVStorageConfig := operands.NewKubeVirtStorageConfigForCR(hco, namespace)
	expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageConfig = expectedKVStorageConfig
	expectedKVStorageRole := operands.NewConfigReaderRoleForCR(hco, namespace)
	expectedKVStorageRole.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/roles/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageRole = expectedKVStorageRole

	expectedKVStorageRoleBinding := operands.NewConfigReaderRoleBindingForCR(hco, namespace)
	expectedKVStorageRoleBinding.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/rolebindings/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageRoleBinding = expectedKVStorageRoleBinding

	expectedKV, err := operands.NewKubeVirt(hco, namespace)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

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

	expectedSSP := operands.NewSSP(hco)
	expectedSSP.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedSSP.Namespace, expectedSSP.Name)
	expectedSSP.Status.Conditions = getGenericCompletedConditions()
	res.ssp = expectedSSP

	expectedVMI := operands.NewVMImportForCR(hco)
	expectedVMI.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedVMI.Namespace, expectedVMI.Name)
	expectedVMI.Status.Conditions = getGenericCompletedConditions()
	res.vmi = expectedVMI

	res.imsConfig, err = operands.NewIMSConfigForCR(hco, namespace)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	res.imsConfig.Data["v2v-conversion-image"] = commonTestUtils.ConversionImage
	res.imsConfig.Data["kubevirt-vmware-image"] = commonTestUtils.VmwareImage

	expectedCliDownload := operands.NewConsoleCLIDownload(hco)
	expectedCliDownload.SelfLink = fmt.Sprintf("/apis/console.openshift.io/v1/consoleclidownloads/%s", expectedCliDownload.Name)
	res.cliDownload = expectedCliDownload

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
	Expect(err).To(BeNil())

	foundResource := &hcov1beta1.HyperConverged{}
	Expect(
		cl.Get(context.TODO(),
			types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
			foundResource),
	).To(BeNil())

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

func checkHcoReady() bool {
	return hcoutil.IsReady()
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
