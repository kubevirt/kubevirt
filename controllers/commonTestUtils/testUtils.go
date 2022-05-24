package commonTestUtils

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
	openshiftconfigv1 "github.com/openshift/api/config/v1"
	consolev1 "github.com/openshift/api/console/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	imagev1 "github.com/openshift/api/image/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	routev1 "github.com/openshift/api/route/v1"
	csvv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	"github.com/kubevirt/hyperconverged-cluster-operator/api"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"
	ttov1alpha1 "github.com/kubevirt/tekton-tasks-operator/api/v1alpha1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/api"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
)

// Name and Namespace of our primary resource
const (
	Name           = "kubevirt-hyperconverged"
	Namespace      = "kubevirt-hyperconverged"
	VirtioWinImage = "quay.io/kubevirt/virtio-container-disk:v2.0.0"
)

var (
	TestLogger  = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)).WithName("controller_hyperconverged")
	TestRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      Name,
			Namespace: Namespace,
		},
	}
)

func NewHco() *hcov1beta1.HyperConverged {
	hco := components.GetOperatorCR()
	hco.ObjectMeta.Namespace = Namespace
	return hco
}

func NewHcoNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: Namespace,
		},
	}
}

func NewReq(inst *hcov1beta1.HyperConverged) *common.HcoRequest {
	return &common.HcoRequest{
		Request:      TestRequest,
		Logger:       TestLogger,
		Conditions:   common.NewHcoConditions(),
		Ctx:          context.TODO(),
		Instance:     inst,
		HCOTriggered: true,
	}
}

func getNodePlacement(num1, num2 int64) *sdkapi.NodePlacement {
	var (
		key1 = fmt.Sprintf("key%d", num1)
		key2 = fmt.Sprintf("key%d", num2)

		val1 = fmt.Sprintf("value%d", num1)
		val2 = fmt.Sprintf("value%d", num2)

		operator1 = corev1.NodeSelectorOperator(fmt.Sprintf("operator%d", num1))
		operator2 = corev1.NodeSelectorOperator(fmt.Sprintf("operator%d", num2))

		effect1 = corev1.TaintEffect(fmt.Sprintf("effect%d", num1))
		effect2 = corev1.TaintEffect(fmt.Sprintf("effect%d", num2))

		firstVal1  = fmt.Sprintf("value%d1", num1)
		secondVal1 = fmt.Sprintf("value%d2", num1)
		firstVal2  = fmt.Sprintf("value%d1", num2)
		secondVal2 = fmt.Sprintf("value%d2", num2)
	)
	return &sdkapi.NodePlacement{
		NodeSelector: map[string]string{
			key1: val1,
			key2: val2,
		},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: key1, Operator: operator1, Values: []string{firstVal1, secondVal1}},
								{Key: key2, Operator: operator2, Values: []string{firstVal2, secondVal2}},
							},
							MatchFields: []corev1.NodeSelectorRequirement{
								{Key: key1, Operator: operator1, Values: []string{firstVal1, secondVal1}},
								{Key: key2, Operator: operator2, Values: []string{firstVal2, secondVal2}},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{Key: key1, Operator: corev1.TolerationOperator(operator1), Value: val1, Effect: effect1, TolerationSeconds: &num1},
			{Key: key2, Operator: corev1.TolerationOperator(operator2), Value: val2, Effect: effect2, TolerationSeconds: &num2},
		},
	}
}

func NewNodePlacement() *sdkapi.NodePlacement {
	return getNodePlacement(1, 2)
}

func NewOtherNodePlacement() *sdkapi.NodePlacement {
	return getNodePlacement(3, 4)
}

var testScheme *runtime.Scheme

func GetScheme() *runtime.Scheme {
	if testScheme != nil {
		return testScheme
	}

	testScheme = scheme.Scheme

	for _, f := range []func(*runtime.Scheme) error{
		api.AddToScheme,
		kubevirtcorev1.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddonsv1.AddToScheme,
		sspv1beta1.AddToScheme,
		ttov1alpha1.AddToScheme,
		consolev1.AddToScheme,
		monitoringv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		routev1.Install,
		imagev1.Install,
		consolev1alpha1.Install,
		operatorv1.Install,
		openshiftconfigv1.Install,
	} {
		Expect(f(testScheme)).ToNot(HaveOccurred())
	}

	return testScheme
}

// RepresentCondition - returns a GomegaMatcher useful for comparing conditions
func RepresentCondition(expected metav1.Condition) gomegatypes.GomegaMatcher {
	return &RepresentConditionMatcher{
		expected: expected,
	}
}

type RepresentConditionMatcher struct {
	expected metav1.Condition
}

// Match - compares two conditions
// two conditions are the same if they have the same type, status, reason, and message
func (matcher *RepresentConditionMatcher) Match(actual interface{}) (success bool, err error) {
	actualCondition, ok := actual.(metav1.Condition)
	if !ok {
		return false, fmt.Errorf("RepresentConditionMatcher expects a Condition")
	}

	if matcher.expected.Type != actualCondition.Type {
		return false, nil
	}
	if matcher.expected.Status != actualCondition.Status {
		return false, nil
	}
	if matcher.expected.Reason != actualCondition.Reason {
		return false, nil
	}
	if matcher.expected.Message != actualCondition.Message {
		return false, nil
	}
	return true, nil
}

func (matcher *RepresentConditionMatcher) FailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nto match the condition\n\t%#v", actual, matcher.expected)
}

func (matcher *RepresentConditionMatcher) NegatedFailureMessage(actual interface{}) (message string) {
	return fmt.Sprintf("Expected\n\t%#v\nnot to match the condition\n\t%#v", actual, matcher.expected)
}

const (
	RSName  = "hco-operator"
	podName = RSName + "-12345"
)

var ( // own resources
	csv = &csvv1alpha1.ClusterServiceVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ClusterServiceVersion",
			APIVersion: "operators.coreos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      RSName,
			Namespace: Namespace,
		},
	}

	deployment = &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      RSName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "operators.coreos.com/v1alpha1",
					Kind:       csvv1alpha1.ClusterServiceVersionKind,
					Name:       RSName,
					Controller: pointer.BoolPtr(true),
				},
			},
			UID: "1234567890",
		},
	}

	pod = &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "ReplicaSet",
					Name:       RSName,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
	}
)

// ClusterInfoMock mocks regular Openshift
type ClusterInfoMock struct{}

func (ClusterInfoMock) Init(_ context.Context, _ client.Client, _ logr.Logger) error {
	return nil
}
func (ClusterInfoMock) IsOpenshift() bool {
	return true
}
func (ClusterInfoMock) IsRunningLocally() bool {
	return false
}
func (ClusterInfoMock) IsManagedByOLM() bool {
	return true
}
func (ClusterInfoMock) IsControlPlaneHighlyAvailable() bool {
	return true
}
func (ClusterInfoMock) IsInfrastructureHighlyAvailable() bool {
	return true
}
func (ClusterInfoMock) GetDomain() string {
	return "domain"
}
func (c ClusterInfoMock) IsConsolePluginImageProvided() bool {
	return true
}
func (c ClusterInfoMock) GetPod() *corev1.Pod {
	return pod
}

func (c ClusterInfoMock) GetDeployment() *appsv1.Deployment {
	return deployment
}

func (c ClusterInfoMock) GetCSV() *csvv1alpha1.ClusterServiceVersion {
	return csv
}
func (ClusterInfoMock) GetTLSSecurityProfile(_ *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile {
	return &openshiftconfigv1.TLSSecurityProfile{
		Type:         openshiftconfigv1.TLSProfileIntermediateType,
		Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
	}
}
func (ClusterInfoMock) RefreshAPIServerCR(_ context.Context, _ client.Client) error {
	return nil
}

// ClusterInfoSNOMock mocks Openshift SNO
type ClusterInfoSNOMock struct{}

func (ClusterInfoSNOMock) Init(_ context.Context, _ client.Client, _ logr.Logger) error {
	return nil
}
func (ClusterInfoSNOMock) IsOpenshift() bool {
	return true
}
func (ClusterInfoSNOMock) IsRunningLocally() bool {
	return false
}
func (ClusterInfoSNOMock) IsManagedByOLM() bool {
	return true
}
func (ClusterInfoSNOMock) IsControlPlaneHighlyAvailable() bool {
	return false
}
func (ClusterInfoSNOMock) IsInfrastructureHighlyAvailable() bool {
	return false
}
func (ClusterInfoSNOMock) GetDomain() string {
	return "domain"
}
func (c ClusterInfoSNOMock) GetPod() *corev1.Pod {
	return pod
}

func (c ClusterInfoSNOMock) GetDeployment() *appsv1.Deployment {
	return deployment
}

func (c ClusterInfoSNOMock) GetCSV() *csvv1alpha1.ClusterServiceVersion {
	return csv
}
func (ClusterInfoSNOMock) GetTLSSecurityProfile(_ *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile {
	return &openshiftconfigv1.TLSSecurityProfile{
		Type:         openshiftconfigv1.TLSProfileIntermediateType,
		Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
	}
}
func (ClusterInfoSNOMock) RefreshAPIServerCR(_ context.Context, _ client.Client) error {
	return nil
}

func (ClusterInfoSNOMock) IsConsolePluginImageProvided() bool {
	return true
}

// ClusterInfoSRCPHAIMock mocks Openshift with SingleReplica ControlPlane and HighAvailable Infrastructure
type ClusterInfoSRCPHAIMock struct{}

func (ClusterInfoSRCPHAIMock) Init(_ context.Context, _ client.Client, _ logr.Logger) error {
	return nil
}
func (ClusterInfoSRCPHAIMock) IsOpenshift() bool {
	return true
}
func (ClusterInfoSRCPHAIMock) IsRunningLocally() bool {
	return false
}
func (ClusterInfoSRCPHAIMock) IsManagedByOLM() bool {
	return true
}
func (ClusterInfoSRCPHAIMock) IsControlPlaneHighlyAvailable() bool {
	return false
}
func (ClusterInfoSRCPHAIMock) IsInfrastructureHighlyAvailable() bool {
	return true
}
func (ClusterInfoSRCPHAIMock) GetPod() *corev1.Pod {
	return pod
}

func (ClusterInfoSRCPHAIMock) GetDeployment() *appsv1.Deployment {
	return deployment
}

func (ClusterInfoSRCPHAIMock) GetCSV() *csvv1alpha1.ClusterServiceVersion {
	return csv
}
func (ClusterInfoSRCPHAIMock) GetDomain() string {
	return "domain"
}
func (ClusterInfoSRCPHAIMock) IsConsolePluginImageProvided() bool {
	return true
}
func (ClusterInfoSRCPHAIMock) GetTLSSecurityProfile(_ *openshiftconfigv1.TLSSecurityProfile) *openshiftconfigv1.TLSSecurityProfile {
	return &openshiftconfigv1.TLSSecurityProfile{
		Type:         openshiftconfigv1.TLSProfileIntermediateType,
		Intermediate: &openshiftconfigv1.IntermediateTLSProfile{},
	}
}
func (ClusterInfoSRCPHAIMock) RefreshAPIServerCR(_ context.Context, _ client.Client) error {
	return nil
}
