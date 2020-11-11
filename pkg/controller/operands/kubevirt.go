package operands

import (
	"errors"
	"fmt"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kubevirtDefaultNetworkInterfaceValue = "masquerade"
)

type kubevirtHandler genericOperand

func newKubevirtHandler(Client client.Client, Scheme *runtime.Scheme) *kubevirtHandler {
	handler := &kubevirtHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirt",
		isCr:                   true,
		removeExistingOwner:    false,
		setControllerReference: true,
		getFullCr: func(hc *hcov1beta1.HyperConverged) runtime.Object {
			return NewKubeVirt(hc)
		},
		getEmptyCr: func() runtime.Object { return &kubevirtv1.KubeVirt{} },
		getConditions: func(cr runtime.Object) []conditionsv1.Condition {
			return translateKubeVirtConds(cr.(*kubevirtv1.KubeVirt).Status.Conditions)
		},
		checkComponentVersion: func(cr runtime.Object) bool {
			found := cr.(*kubevirtv1.KubeVirt)
			return checkComponentVersion(hcoutil.KubevirtVersionEnvV, found.Status.ObservedKubeVirtVersion)
		},
		getObjectMeta: func(cr runtime.Object) *metav1.ObjectMeta {
			return &cr.(*kubevirtv1.KubeVirt).ObjectMeta
		},
	}

	handler.updateCr = handler.updateCrImp

	return handler
}

func (h *kubevirtHandler) updateCrImp(req *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	virt, ok1 := required.(*kubevirtv1.KubeVirt)
	found, ok2 := exists.(*kubevirtv1.KubeVirt)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to KubeVirt")
	}
	if !reflect.DeepEqual(found.Spec, virt.Spec) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt's Spec to its opinionated values")
		}
		virt.Spec.DeepCopyInto(&found.Spec)
		err := h.Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (h *kubevirtHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	gh := (*genericOperand)(h)
	return gh.ensure(req)
}

func NewKubeVirt(hc *hcov1beta1.HyperConverged, opts ...string) *kubevirtv1.KubeVirt {
	spec := kubevirtv1.KubeVirtSpec{
		UninstallStrategy: kubevirtv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist,
		Infra:             hcoConfig2KvConfig(hc.Spec.Infra),
		Workloads:         hcoConfig2KvConfig(hc.Spec.Workloads),
	}

	return &kubevirtv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + hc.Name,
			Labels:    getLabels(hc),
			Namespace: getNamespace(hc.Namespace, opts),
		},
		Spec: spec,
	}
}

func hcoConfig2KvConfig(hcoConfig hcov1beta1.HyperConvergedConfig) *kubevirtv1.ComponentConfig {
	if hcoConfig.NodePlacement != nil {
		kvConfig := &kubevirtv1.ComponentConfig{}
		kvConfig.NodePlacement = &kubevirtv1.NodePlacement{}

		if hcoConfig.NodePlacement.Affinity != nil {
			kvConfig.NodePlacement.Affinity = &corev1.Affinity{}
			hcoConfig.NodePlacement.Affinity.DeepCopyInto(kvConfig.NodePlacement.Affinity)
		}

		if hcoConfig.NodePlacement.NodeSelector != nil {
			kvConfig.NodePlacement.NodeSelector = make(map[string]string)
			for k, v := range hcoConfig.NodePlacement.NodeSelector {
				kvConfig.NodePlacement.NodeSelector[k] = v
			}
		}

		for _, hcoTolr := range hcoConfig.NodePlacement.Tolerations {
			kvTolr := corev1.Toleration{}
			hcoTolr.DeepCopyInto(&kvTolr)
			kvConfig.NodePlacement.Tolerations = append(kvConfig.NodePlacement.Tolerations, kvTolr)
		}

		return kvConfig
	}
	return nil
}

type kvConfigHandler genericOperand

func newKvConfigHandler(Client client.Client, Scheme *runtime.Scheme) *kvConfigHandler {
	handler := &kvConfigHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirt Config",
		isCr:                   false,
		removeExistingOwner:    false,
		setControllerReference: true,
		getFullCr: func(hc *hcov1beta1.HyperConverged) runtime.Object {
			return NewKubeVirtConfigForCR(hc, hc.Namespace)
		},
		getEmptyCr: func() runtime.Object { return &corev1.ConfigMap{} },
		getObjectMeta: func(cr runtime.Object) *metav1.ObjectMeta {
			return &cr.(*corev1.ConfigMap).ObjectMeta
		},
	}

	handler.updateCr = handler.updateCrImp

	return handler
}

func (h *kvConfigHandler) updateCrImp(req *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	kubevirtConfig, ok1 := required.(*corev1.ConfigMap)
	found, ok2 := exists.(*corev1.ConfigMap)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to ConfigMap")
	}

	if req.UpgradeMode {
		changed := false
		// only virtconfig.SmbiosConfigKey, virtconfig.MachineTypeKey, virtconfig.SELinuxLauncherTypeKey,
		// virtconfig.FeatureGatesKey and virtconfig.UseEmulationKey are going to be manipulated
		// and only on HCO upgrades.
		// virtconfig.MigrationsConfigKey is going to be removed if set in the past (only during upgrades).
		// TODO: This is going to change in the next HCO release where the whole configMap is going
		// to be continuously reconciled
		for _, k := range []string{
			virtconfig.FeatureGatesKey,
			virtconfig.SmbiosConfigKey,
			virtconfig.MachineTypeKey,
			virtconfig.SELinuxLauncherTypeKey,
			virtconfig.UseEmulationKey,
			virtconfig.MigrationsConfigKey,
		} {
			if found.Data[k] != kubevirtConfig.Data[k] {
				req.Logger.Info(fmt.Sprintf("Updating %s on existing KubeVirt config", k))
				found.Data[k] = kubevirtConfig.Data[k]
				changed = true
			}
		}
		for _, k := range []string{virtconfig.MigrationsConfigKey} {
			_, ok := found.Data[k]
			if ok {
				req.Logger.Info(fmt.Sprintf("Deleting %s on existing KubeVirt config", k))
				delete(found.Data, k)
				changed = true
			}
		}

		if changed {
			err := h.Client.Update(req.Ctx, found)
			if err != nil {
				req.Logger.Error(err, "Failed updating the kubevirt config map")
				return false, false, err
			}
			return true, false, nil
		}
	}
	return false, false, nil
}

func (h *kvConfigHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	gh := (*genericOperand)(h)
	res := gh.ensure(req)

	return res
}

type kvPriorityClassHandler genericOperand

func (h *kvPriorityClassHandler) updateCrImp(req *common.HcoRequest, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	pc, ok1 := required.(*schedulingv1.PriorityClass)
	found, ok2 := exists.(*schedulingv1.PriorityClass)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to PriorityClass")
	}

	// at this point we found the object in the cache and we check if something was changed
	if (pc.Name == found.Name) && (pc.Value == found.Value) && (pc.Description == found.Description) {
		return false, false, nil
	}

	if req.HCOTriggered {
		req.Logger.Info("Updating existing KubeVirt's Spec to new opinionated values")
	} else {
		req.Logger.Info("Reconciling an externally updated KubeVirt's Spec to its opinionated values")
	}

	// something was changed but since we can't patch a priority class object, we remove it
	err := h.Client.Delete(req.Ctx, found, &client.DeleteOptions{})
	if err != nil {
		return false, false, err
	}

	// create the new object
	err = h.Client.Create(req.Ctx, pc, &client.CreateOptions{})
	if err != nil {
		return false, false, err
	}

	return true, !req.HCOTriggered, nil
}

func (h *kvPriorityClassHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	gh := (*genericOperand)(h)
	res := gh.ensure(req)

	return res
}

func newKvPriorityClassHandler(Client client.Client, Scheme *runtime.Scheme) *kvPriorityClassHandler {
	handler := &kvPriorityClassHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirt PriorityClass",
		isCr:                   false,
		removeExistingOwner:    false,
		setControllerReference: false,
		getFullCr: func(hc *hcov1beta1.HyperConverged) runtime.Object {
			return NewKubeVirtPriorityClass(hc)
		},
		getEmptyCr: func() runtime.Object { return &schedulingv1.PriorityClass{} },
		getObjectMeta: func(cr runtime.Object) *metav1.ObjectMeta {
			return &cr.(*schedulingv1.PriorityClass).ObjectMeta
		},
	}
	handler.updateCr = handler.updateCrImp

	return handler
}

func NewKubeVirtPriorityClass(hc *hcov1beta1.HyperConverged) *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-cluster-critical",
			Labels: getLabels(hc),
		},
		// 1 billion is the highest value we can set
		// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/#priorityclass
		Value:         1000000000,
		GlobalDefault: false,
		Description:   "This priority class should be used for KubeVirt core components only.",
	}
}

// translateKubeVirtConds translates list of KubeVirt conditions to a list of custom resource
// conditions.
func translateKubeVirtConds(orig []kubevirtv1.KubeVirtCondition) []conditionsv1.Condition {
	translated := make([]conditionsv1.Condition, len(orig))

	for i, origCond := range orig {
		translated[i] = conditionsv1.Condition{
			Type:    conditionsv1.ConditionType(origCond.Type),
			Status:  origCond.Status,
			Reason:  origCond.Reason,
			Message: origCond.Message,
		}
	}

	return translated
}

func NewKubeVirtConfigForCR(cr *hcov1beta1.HyperConverged, namespace string) *corev1.ConfigMap {
	labels := map[string]string{
		hcoutil.AppLabel: cr.Name,
	}
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-config",
			Labels:    labels,
			Namespace: namespace,
		},
		// only virtconfig.SmbiosConfigKey, virtconfig.MachineTypeKey, virtconfig.SELinuxLauncherTypeKey,
		// virtconfig.FeatureGatesKey and virtconfig.UseEmulationKey are going to be manipulated
		// and only on HCO upgrades.
		// virtconfig.MigrationsConfigKey is going to be removed if set in the past (only during upgrades).
		// TODO: This is going to change in the next HCO release where the whole configMap is going
		// to be continuously reconciled
		Data: map[string]string{
			virtconfig.FeatureGatesKey:        "DataVolumes,SRIOV,LiveMigration,CPUManager,CPUNodeDiscovery,Sidecar,Snapshot",
			virtconfig.SELinuxLauncherTypeKey: "virt_launcher.process",
			virtconfig.NetworkInterfaceKey:    kubevirtDefaultNetworkInterfaceValue,
		},
	}
	val, ok := os.LookupEnv("SMBIOS")
	if ok && val != "" {
		cm.Data[virtconfig.SmbiosConfigKey] = val
	}
	val, ok = os.LookupEnv("MACHINETYPE")
	if ok && val != "" {
		cm.Data[virtconfig.MachineTypeKey] = val
	}
	val, ok = os.LookupEnv("KVM_EMULATION")
	if ok && val != "" {
		cm.Data[virtconfig.UseEmulationKey] = val
	}
	return cm
}
