package v1alpha1

import (
	networkaddonsshared "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
	networkaddonsv1alpha1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1alpha1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1alpha1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1alpha1"
)

func (r *HyperConverged) getNamespace(defaultNamespace string, opts []string) string {
	if len(opts) > 0 {
		return opts[0]
	}
	return defaultNamespace
}

func (r *HyperConverged) getLabels(component hcoutil.AppComponent) map[string]string {
	hcoName := hcoutil.HyperConvergedName

	if r.Name != "" {
		hcoName = r.Name
	}

	return map[string]string{
		hcoutil.AppLabel:          hcoName,
		hcoutil.AppLabelManagedBy: hcoutil.OperatorName,
		hcoutil.AppLabelVersion:   hcoutil.GetHcoKvIoVersion(),
		hcoutil.AppLabelPartOf:    hcoutil.HyperConvergedCluster,
		hcoutil.AppLabelComponent: string(component),
	}
}

func (r *HyperConverged) NewKubeVirt(opts ...string) *kubevirtv1.KubeVirt {
	return &kubevirtv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + r.Name,
			Labels:    r.getLabels(hcoutil.AppComponentCompute),
			Namespace: r.getNamespace(r.Namespace, opts),
		},
		Spec: kubevirtv1.KubeVirtSpec{
			UninstallStrategy: kubevirtv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist,
		},
	}
}

func (r *HyperConverged) NewCDI(opts ...string) *cdiv1alpha1.CDI {
	uninstallStrategy := cdiv1alpha1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist
	return &cdiv1alpha1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-" + r.Name,
			Labels:    r.getLabels(hcoutil.AppComponentStorage),
			Namespace: r.getNamespace(hcoutil.UndefinedNamespace, opts),
		},
		Spec: cdiv1alpha1.CDISpec{
			UninstallStrategy: &uninstallStrategy,
		},
	}
}

func (r *HyperConverged) NewNetworkAddons(opts ...string) *networkaddonsv1alpha1.NetworkAddonsConfig {
	return &networkaddonsv1alpha1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkaddonsnames.OPERATOR_CONFIG,
			Labels:    r.getLabels(hcoutil.AppComponentNetwork),
			Namespace: r.getNamespace(hcoutil.UndefinedNamespace, opts),
		},
		Spec: networkaddonsshared.NetworkAddonsConfigSpec{
			Multus:      &networkaddonsshared.Multus{},
			LinuxBridge: &networkaddonsshared.LinuxBridge{},
			Ovs:         &networkaddonsshared.Ovs{},
			NMState:     &networkaddonsshared.NMState{},
			KubeMacPool: &networkaddonsshared.KubeMacPool{},
		},
	}
}

func (r *HyperConverged) NewKubeVirtPriorityClass() *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-cluster-critical",
			Labels: r.getLabels(hcoutil.AppComponentSchedule),
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}
