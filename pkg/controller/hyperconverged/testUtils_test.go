package hyperconverged

import (
	"context"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/operands"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	"github.com/kubevirt/hyperconverged-cluster-operator/version"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/commonTestUtils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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

func initReconciler(client client.Client) *ReconcileHyperConverged {
	s := commonTestUtils.GetScheme()
	operandHandler := operands.NewOperandHandler(client, s, true, &eventEmitterMock{})
	// Create a ReconcileHyperConverged object with the scheme and fake client
	return &ReconcileHyperConverged{
		client:             client,
		scheme:             s,
		operandHandler:     operandHandler,
		eventEmitter:       &eventEmitterMock{},
		cliDownloadHandler: &operands.CLIDownloadHandler{Client: client, Scheme: s},
		firstLoop:          true,
	}
}

type BasicExpected struct {
	hco                  *hcov1beta1.HyperConverged
	pc                   *schedulingv1.PriorityClass
	kvConfig             *corev1.ConfigMap
	kvStorageConfig      *corev1.ConfigMap
	kvStorageRole        *rbacv1.Role
	kvStorageRoleBinding *rbacv1.RoleBinding
	kv                   *kubevirtv1.KubeVirt
	cdi                  *cdiv1beta1.CDI
	cna                  *networkaddonsv1.NetworkAddonsConfig
	kvCtb                *sspv1.KubevirtCommonTemplatesBundle
	kvNlb                *sspv1.KubevirtNodeLabellerBundle
	kvTv                 *sspv1.KubevirtTemplateValidator
	vmi                  *vmimportv1beta1.VMImportConfig
	kvMtAg               *sspv1.KubevirtMetricsAggregation
	imsConfig            *corev1.ConfigMap
}

func (be BasicExpected) toArray() []runtime.Object {
	return []runtime.Object{
		be.hco,
		be.pc,
		be.kvConfig,
		be.kvStorageConfig,
		be.kvStorageRole,
		be.kvStorageRoleBinding,
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

func (be BasicExpected) initClient() *commonTestUtils.HcoTestClient {
	return commonTestUtils.InitClient(be.toArray())
}

func getBasicDeployment() *BasicExpected {

	res := &BasicExpected{}

	hco := &hcov1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: hcov1beta1.HyperConvergedSpec{},
		Status: hcov1beta1.HyperConvergedStatus{
			Conditions: []conditionsv1.Condition{
				{
					Type:    hcov1beta1.ConditionReconcileComplete,
					Status:  corev1.ConditionTrue,
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

	res.pc = hco.NewKubeVirtPriorityClass()
	// These are all of the objects that we expect to "find" in the client because
	// we already created them in a previous reconcile.
	expectedKVConfig := operands.NewKubeVirtConfigForCR(hco, namespace)
	expectedKVConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVConfig.Namespace, expectedKVConfig.Name)
	res.kvConfig = expectedKVConfig

	expectedKVStorageConfig := operands.NewKubeVirtStorageConfigForCR(hco, namespace)
	expectedKVStorageConfig.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/configmaps/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageConfig = expectedKVStorageConfig
	expectedKVStorageRole := operands.NewKubeVirtStorageRoleForCR(hco, namespace)
	expectedKVStorageRole.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/roles/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageRole = expectedKVStorageRole

	expectedKVStorageRoleBinding := operands.NewKubeVirtStorageRoleBindingForCR(hco, namespace)
	expectedKVStorageRoleBinding.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/rolebindings/%s", expectedKVStorageConfig.Namespace, expectedKVStorageConfig.Name)
	res.kvStorageRoleBinding = expectedKVStorageRoleBinding

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
	res.kv = expectedKV

	expectedCDI := hco.NewCDI()
	expectedCDI.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cdis/%s", expectedCDI.Namespace, expectedCDI.Name)
	expectedCDI.Status.Conditions = getGenericCompletedConditions()
	res.cdi = expectedCDI

	expectedCNA := hco.NewNetworkAddons()
	expectedCNA.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/cnas/%s", expectedCNA.Namespace, expectedCNA.Name)
	expectedCNA.Status.Conditions = getGenericCompletedConditions()
	res.cna = expectedCNA

	expectedKVCTB := hco.NewKubeVirtCommonTemplateBundle()
	expectedKVCTB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/ctbs/%s", expectedKVCTB.Namespace, expectedKVCTB.Name)
	expectedKVCTB.Status.Conditions = getGenericCompletedConditions()
	res.kvCtb = expectedKVCTB

	expectedKVNLB := operands.NewKubeVirtNodeLabellerBundleForCR(hco, namespace)
	expectedKVNLB.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/nlb/%s", expectedKVNLB.Namespace, expectedKVNLB.Name)
	expectedKVNLB.Status.Conditions = getGenericCompletedConditions()
	res.kvNlb = expectedKVNLB

	expectedKVTV := operands.NewKubeVirtTemplateValidatorForCR(hco, namespace)
	expectedKVTV.ObjectMeta.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/tv/%s", expectedKVTV.Namespace, expectedKVTV.Name)
	expectedKVTV.Status.Conditions = getGenericCompletedConditions()
	res.kvTv = expectedKVTV

	expectedVMI := operands.NewVMImportForCR(hco)
	expectedVMI.SelfLink = fmt.Sprintf("/apis/v1/namespaces/%s/vmimportconfigs/%s", expectedVMI.Namespace, expectedVMI.Name)
	expectedVMI.Status.Conditions = getGenericCompletedConditions()
	res.vmi = expectedVMI

	kvMtAg := operands.NewKubeVirtMetricsAggregationForCR(hco, namespace)
	kvMtAg.Status.Conditions = getGenericCompletedConditions()
	res.kvMtAg = kvMtAg

	res.imsConfig = operands.NewIMSConfigForCR(hco, namespace)
	res.imsConfig.Data["v2v-conversion-image"] = commonTestUtils.Conversion_image
	res.imsConfig.Data["kubevirt-vmware-image"] = commonTestUtils.Vmware_image

	return res
}

// returns the HCO after reconcile, and the returned requeue
func doReconcile(cl client.Client, hco *hcov1beta1.HyperConverged) (*hcov1beta1.HyperConverged, bool) {
	r := initReconciler(cl)

	r.ownVersion = os.Getenv(hcoutil.HcoKvIoVersionName)
	if r.ownVersion == "" {
		r.ownVersion = version.Version
	}

	res, err := r.Reconcile(request)
	Expect(err).To(BeNil())

	foundResource := &hcov1beta1.HyperConverged{}
	Expect(
		cl.Get(context.TODO(),
			types.NamespacedName{Name: hco.Name, Namespace: hco.Namespace},
			foundResource),
	).To(BeNil())

	return foundResource, res.Requeue
}

type eventEmitterMock struct{}

func (eventEmitterMock) Init(_ context.Context, _ manager.Manager, _ hcoutil.ClusterInfo, _ logr.Logger) {
}

func (eventEmitterMock) EmitEvent(_ runtime.Object, _, _, _ string) {
}

func (eventEmitterMock) UpdateClient(_ context.Context, _ client.Reader, _ logr.Logger) {
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

func checkHcoReady() (bool, error) {
	_, err := os.Stat(ready.FileName)

	if err == nil {
		return true, nil
	} else if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func checkAvailability(hco *hcov1beta1.HyperConverged, expected corev1.ConditionStatus) {
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
