package commonTestUtils

import (
	"context"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis"
	sspopv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis"
	vmimportv1beta1 "github.com/kubevirt/vm-import-operator/pkg/apis/v2v/v1beta1"
	consolev1 "github.com/openshift/api/console/v1"
	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	sdkapi "kubevirt.io/controller-lifecycle-operator-sdk/pkg/sdk/api"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	. "github.com/onsi/gomega"
)

// Name and Namespace of our primary resource
const (
	Name             = "kubevirt-hyperconverged"
	Namespace        = "kubevirt-hyperconverged"
	Conversion_image = "quay.io/kubevirt/kubevirt-v2v-conversion:v2.0.0"
	Vmware_image     = "quay.io/kubevirt/kubevirt-vmware:v2.0.0"
)

var (
	TestLogger  = logf.Log.WithName("controller_hyperconverged")
	TestRequest = reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      Name,
			Namespace: Namespace,
		},
	}
)

func NewHco() *hcov1beta1.HyperConverged {
	return &hcov1beta1.HyperConverged{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: Namespace,
		},
		Spec: hcov1beta1.HyperConvergedSpec{},
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

func NewHyperConvergedConfig() *sdkapi.NodePlacement {
	seconds1, seconds2 := int64(1), int64(2)
	return &sdkapi.NodePlacement{
		NodeSelector: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
								{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
							},
							MatchFields: []corev1.NodeSelectorRequirement{
								{Key: "key1", Operator: "operator1", Values: []string{"value11, value12"}},
								{Key: "key2", Operator: "operator2", Values: []string{"value21, value22"}},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{Key: "key1", Operator: "operator1", Value: "value1", Effect: "effect1", TolerationSeconds: &seconds1},
			{Key: "key2", Operator: "operator2", Value: "value2", Effect: "effect2", TolerationSeconds: &seconds2},
		},
	}
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
		sspopv1.AddToScheme,
		vmimportv1beta1.AddToScheme,
		consolev1.AddToScheme,
		monitoringv1.AddToScheme,
		extv1.AddToScheme,
	} {
		Expect(f(testScheme)).To(BeNil())
	}

	return testScheme
}
