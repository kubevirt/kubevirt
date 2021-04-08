package commonTestUtils

import (
	"context"
	"fmt"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	consolev1 "github.com/openshift/api/console/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	sspv1beta1 "kubevirt.io/ssp-operator/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/components"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Name and Namespace of our primary resource
const (
	Name            = "kubevirt-hyperconverged"
	Namespace       = "kubevirt-hyperconverged"
	ConversionImage = "quay.io/kubevirt/kubevirt-v2v-conversion:v2.0.0"
	VmwareImage     = "quay.io/kubevirt/kubevirt-vmware:v2.0.0"
	VirtioWinImage  = "quay.io/kubevirt/virtio-container-disk:v2.0.0"
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
		cdiv1beta1.AddToScheme,
		networkaddons.AddToScheme,
		sspv1beta1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		consolev1.AddToScheme,
		monitoringv1.AddToScheme,
		apiextensionsv1.AddToScheme,
	} {
		Expect(f(testScheme)).To(BeNil())
	}

	return testScheme
}
