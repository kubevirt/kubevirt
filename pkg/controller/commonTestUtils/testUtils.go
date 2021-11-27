package commonTestUtils

import (
	"context"
	"fmt"

	consolev1 "github.com/openshift/api/console/v1"
	routev1 "github.com/openshift/api/route/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	kubevirtcorev1 "kubevirt.io/api/core/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer-api/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"

	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	gomegatypes "github.com/onsi/gomega/types"
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
		apis.AddToScheme,
		kubevirtcorev1.AddToScheme,
		cdiv1beta1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
		consolev1.AddToScheme,
		monitoringv1.AddToScheme,
		apiextensionsv1.AddToScheme,
		routev1.Install,
	} {
		Expect(f(testScheme)).To(BeNil())
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
