package hyperconverged

import (
	sspv1 "github.com/MarSik/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	networkaddons "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcov1alpha1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	cdi "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
	kubevirt "kubevirt.io/kubevirt/pkg/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const UndefinedNamespace string = ""
const OpenshiftNamespace string = "openshift"

// The set of resources managed by the HCO
func (r *ReconcileHyperConverged) getAllResources(cr *hcov1alpha1.HyperConverged, request reconcile.Request) []runtime.Object {
	return []runtime.Object{
		newKubeVirtConfigForCR(cr, request.Namespace),
		newKubeVirtForCR(cr, request.Namespace),
		newCDIForCR(cr, UndefinedNamespace),
		newNetworkAddonsForCR(cr, UndefinedNamespace),
		newKubeVirtCommonTemplateBundleForCR(cr, OpenshiftNamespace),
		newKubeVirtNodeLabellerBundleForCR(cr, request.Namespace),
		newKubeVirtTemplateValidatorForCR(cr, request.Namespace),
	}
}

func newKubeVirtConfigForCR(cr *hcov1alpha1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-config",
			Labels:    labels,
			Namespace: namespace,
		},
		Data: map[string]string{
			"feature-gates": "DataVolumes,SRIOV,LiveMigration,CPUManager,CPUNodeDiscovery",
		},
	}
}

// newKubeVirtForCR returns a KubeVirt CR
func newKubeVirtForCR(cr *hcov1alpha1.HyperConverged, namespace string) *kubevirt.KubeVirt {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &kubevirt.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

// newCDIForCr returns a CDI CR
func newCDIForCR(cr *hcov1alpha1.HyperConverged, namespace string) *cdi.CDI {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &cdi.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

// newNetworkAddonsForCR returns a NetworkAddonsConfig CR
func newNetworkAddonsForCR(cr *hcov1alpha1.HyperConverged, namespace string) *networkaddons.NetworkAddonsConfig {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &networkaddons.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkaddonsnames.OPERATOR_CONFIG,
			Labels:    labels,
			Namespace: namespace,
		},
		Spec: networkaddons.NetworkAddonsConfigSpec{
			Multus:      &networkaddons.Multus{},
			LinuxBridge: &networkaddons.LinuxBridge{},
			KubeMacPool: &networkaddons.KubeMacPool{},
		},
	}
}

func newKubeVirtCommonTemplateBundleForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtCommonTemplatesBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtCommonTemplatesBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-templates-" + cr.Name,
			Labels:    labels,
			Namespace: OpenshiftNamespace,
		},
	}
}

func newKubeVirtNodeLabellerBundleForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtNodeLabellerBundle {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtNodeLabellerBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "node-labeller-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

func newKubeVirtTemplateValidatorForCR(cr *hcov1alpha1.HyperConverged, namespace string) *sspv1.KubevirtTemplateValidator {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &sspv1.KubevirtTemplateValidator{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "template-validator-" + cr.Name,
			Labels:    labels,
			Namespace: namespace,
		},
	}
}

