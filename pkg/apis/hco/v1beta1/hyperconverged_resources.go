package v1beta1

import (
	"fmt"
	sdkapi "github.com/kubevirt/controller-lifecycle-operator-sdk/pkg/sdk/api"
	corev1 "k8s.io/api/core/v1"
	"os"

	networkaddonsshared "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/shared"
	networkaddonsv1 "github.com/kubevirt/cluster-network-addons-operator/pkg/apis/networkaddonsoperator/v1"
	networkaddonsnames "github.com/kubevirt/cluster-network-addons-operator/pkg/names"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	sspv1 "github.com/kubevirt/kubevirt-ssp-operator/pkg/apis/kubevirt/v1"
	consolev1 "github.com/openshift/api/console/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	cdiv1beta1 "kubevirt.io/containerized-data-importer/pkg/apis/core/v1beta1"
)

func (r *HyperConverged) getNamespace(defaultNamespace string, opts []string) string {
	if len(opts) > 0 {
		return opts[0]
	}
	return defaultNamespace
}

func (r *HyperConverged) getLabels() map[string]string {
	hcoName := HyperConvergedName

	if r.Name != "" {
		hcoName = r.Name
	}

	return map[string]string{
		hcoutil.AppLabel: hcoName,
	}
}

func (r *HyperConverged) NewKubeVirt(opts ...string) *kubevirtv1.KubeVirt {
	return &kubevirtv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + r.Name,
			Labels:    r.getLabels(),
			Namespace: r.getNamespace(r.Namespace, opts),
		},
		Spec: kubevirtv1.KubeVirtSpec{
			UninstallStrategy: kubevirtv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist,
		},
		// TODO: propagate NodePlacement
	}
}

func (r *HyperConverged) NewCDI(opts ...string) *cdiv1beta1.CDI {
	uninstallStrategy := cdiv1beta1.CDIUninstallStrategyBlockUninstallIfWorkloadsExist

	spec := cdiv1beta1.CDISpec{
		UninstallStrategy: &uninstallStrategy,
	}

	if r.Spec.Infra.NodePlacement != nil {
		hcoConfig2CdiPlacement(r.Spec.Infra.NodePlacement, &spec.Infra)
	}
	if r.Spec.Workloads.NodePlacement != nil {
		hcoConfig2CdiPlacement(r.Spec.Workloads.NodePlacement, &spec.Workloads)
	}

	return &cdiv1beta1.CDI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdi-" + r.Name,
			Labels:    r.getLabels(),
			Namespace: r.getNamespace(hcoutil.UndefinedNamespace, opts),
		},
		Spec: spec,
		// TODO: propagate NodePlacement
	}
}

func hcoConfig2CdiPlacement(hcoPlacement *sdkapi.NodePlacement, cdiPlacement *cdiv1beta1.NodePlacement) {
	if hcoPlacement != nil {
		if hcoPlacement.Affinity != nil {
			cdiPlacement.Affinity = &corev1.Affinity{}
			hcoPlacement.Affinity.DeepCopyInto(cdiPlacement.Affinity)
		}

		if hcoPlacement.NodeSelector != nil {
			cdiPlacement.NodeSelector = make(map[string]string)
			for k, v := range hcoPlacement.NodeSelector {
				cdiPlacement.NodeSelector[k] = v
			}
		}

		for _, hcoTolr := range hcoPlacement.Tolerations {
			cdiTolr := corev1.Toleration{}
			hcoTolr.DeepCopyInto(&cdiTolr)
			cdiPlacement.Tolerations = append(cdiPlacement.Tolerations, cdiTolr)
		}
	}
}

func (r *HyperConverged) NewNetworkAddons(opts ...string) *networkaddonsv1.NetworkAddonsConfig {

	cnaoSpec := networkaddonsshared.NetworkAddonsConfigSpec{
		Multus:      &networkaddonsshared.Multus{},
		LinuxBridge: &networkaddonsshared.LinuxBridge{},
		Ovs:         &networkaddonsshared.Ovs{},
		NMState:     &networkaddonsshared.NMState{},
		KubeMacPool: &networkaddonsshared.KubeMacPool{},
	}

	cnaoInfra := hcoConfig2CnaoPlacement(r.Spec.Infra.NodePlacement)
	cnaoWorkloads := hcoConfig2CnaoPlacement(r.Spec.Workloads.NodePlacement)
	if cnaoInfra != nil || cnaoWorkloads != nil {
		cnaoSpec.PlacementConfiguration = &networkaddonsshared.PlacementConfiguration{
			Infra:     cnaoInfra,
			Workloads: cnaoWorkloads,
		}
	}

	return &networkaddonsv1.NetworkAddonsConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      networkaddonsnames.OPERATOR_CONFIG,
			Labels:    r.getLabels(),
			Namespace: r.getNamespace(hcoutil.UndefinedNamespace, opts),
		},
		Spec: cnaoSpec,
	}
}

func hcoConfig2CnaoPlacement(hcoConf *sdkapi.NodePlacement) *networkaddonsshared.Placement {
	if hcoConf == nil {
		return nil
	}
	empty := true
	cnaoPlacement := &networkaddonsshared.Placement{}
	if hcoConf.Affinity != nil {
		empty = false
		hcoConf.Affinity.DeepCopyInto(&cnaoPlacement.Affinity)
	}

	for _, hcoTol := range hcoConf.Tolerations {
		empty = false
		cnaoTol := corev1.Toleration{}
		hcoTol.DeepCopyInto(&cnaoTol)
		cnaoPlacement.Tolerations = append(cnaoPlacement.Tolerations, cnaoTol)
	}

	if len(hcoConf.NodeSelector) > 0 {
		empty = false
		cnaoPlacement.NodeSelector = make(map[string]string)
		for k, v := range hcoConf.NodeSelector {
			cnaoPlacement.NodeSelector[k] = v
		}
	}

	if empty {
		return nil
	}
	return cnaoPlacement
}

func (r *HyperConverged) NewKubeVirtCommonTemplateBundle(opts ...string) *sspv1.KubevirtCommonTemplatesBundle {
	return &sspv1.KubevirtCommonTemplatesBundle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "common-templates-" + r.Name,
			Labels:    r.getLabels(),
			Namespace: r.getNamespace(hcoutil.OpenshiftNamespace, opts),
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
			Labels: r.getLabels(),
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}

func (r *HyperConverged) NewConsoleCLIDownload() *consolev1.ConsoleCLIDownload {
	kv := os.Getenv(hcoutil.KubevirtVersionEnvV)
	url := fmt.Sprintf("https://github.com/kubevirt/kubevirt/releases/%s", kv)
	text := fmt.Sprintf("KubeVirt %s release downloads", kv)

	if val, ok := os.LookupEnv("VIRTCTL_DOWNLOAD_URL"); ok && val != "" {
		url = val
	}

	if val, ok := os.LookupEnv("VIRTCTL_DOWNLOAD_TEXT"); ok && val != "" {
		text = val
	}

	return &consolev1.ConsoleCLIDownload{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "virtctl-clidownloads-" + r.Name,
			Labels: r.getLabels(),
		},

		Spec: consolev1.ConsoleCLIDownloadSpec{
			Description: "The virtctl client is a supplemental command-line utility for managing virtualization resources from the command line.",
			DisplayName: "virtctl - KubeVirt command line interface",
			Links: []consolev1.CLIDownloadLink{
				{
					Href: url,
					Text: text,
				},
			},
		},
	}
}
