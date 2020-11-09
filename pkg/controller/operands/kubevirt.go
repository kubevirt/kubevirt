package operands

import (
	"fmt"
	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	objectreferencesv1 "github.com/openshift/custom-resource-status/objectreferences/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	virtconfig "kubevirt.io/kubevirt/pkg/virt-config"
	"os"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	kubevirtDefaultNetworkInterfaceValue = "masquerade"
)

type kubevirtHandler genericOperand

func (kv *kubevirtHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	virt := NewKubeVirt(req.Instance)
	res := NewEnsureResult(virt)
	if err := controllerutil.SetControllerReference(req.Instance, virt, kv.Scheme); err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(virt)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt")
	}

	res.SetName(key.Name)
	found := &kubevirtv1.KubeVirt{}
	err = kv.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating kubevirt")
			err = kv.Client.Create(req.Ctx, virt)
			if err == nil {
				return res.SetCreated().SetName(virt.Name)
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("KubeVirt already exists", "KubeVirt.Namespace", found.Namespace, "KubeVirt.Name", found.Name)

	if !reflect.DeepEqual(found.Spec, virt.Spec) {
		overwritten := false
		if req.HCOTriggered {
			req.Logger.Info("Updating existing VMimport's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt's Spec to its opinionated values")
			overwritten = true
		}
		virt.Spec.DeepCopyInto(&found.Spec)
		err = kv.Client.Update(req.Ctx, found)
		if err != nil {
			return res.Error(err)
		}
		if overwritten {
			res.SetOverwritten()
		}
		return res.SetUpdated()
	}

	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(kv.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

	// Handle KubeVirt resource conditions
	isReady := handleComponentConditions(req, "KubeVirt", translateKubeVirtConds(found.Status.Conditions))

	upgradeDone := req.ComponentUpgradeInProgress && isReady && checkComponentVersion(hcoutil.KubevirtVersionEnvV, found.Status.ObservedKubeVirtVersion)

	return res.SetUpgradeDone(upgradeDone)
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

func (kvc *kvConfigHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	kubevirtConfig := NewKubeVirtConfigForCR(req.Instance, req.Namespace)
	res := NewEnsureResult(kubevirtConfig)
	err := controllerutil.SetControllerReference(req.Instance, kubevirtConfig, kvc.Scheme)
	if err != nil {
		return res.Error(err)
	}

	key, err := client.ObjectKeyFromObject(kubevirtConfig)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for kubevirt config")
	}
	res.SetName(key.Name)

	found := &corev1.ConfigMap{}
	err = kvc.Client.Get(req.Ctx, key, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			req.Logger.Info("Creating kubevirt config")
			err = kvc.Client.Create(req.Ctx, kubevirtConfig)
			if err == nil {
				return res.SetCreated()
			}
		}
		return res.Error(err)
	}

	req.Logger.Info("KubeVirt config already exists", "KubeVirtConfig.Namespace", found.Namespace, "KubeVirtConfig.Name", found.Name)
	// Add it to the list of RelatedObjects if found
	objectRef, err := reference.GetReference(kvc.Scheme, found)
	if err != nil {
		return res.Error(err)
	}
	objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

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
			err = kvc.Client.Update(req.Ctx, found)
			if err != nil {
				req.Logger.Error(err, "Failed updating the kubevirt config map")
				return res.Error(err)
			}
		}
	}

	return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
}

type kvPriorityClassHandler genericOperand

func (kvpc *kvPriorityClassHandler) Ensure(req *common.HcoRequest) *EnsureResult {
	req.Logger.Info("Reconciling KubeVirt PriorityClass")
	pc := NewKubeVirtPriorityClass(req.Instance)
	res := NewEnsureResult(pc)
	key, err := client.ObjectKeyFromObject(pc)
	if err != nil {
		req.Logger.Error(err, "Failed to get object key for KubeVirt PriorityClass")
		return res.Error(err)
	}

	res.SetName(key.Name)
	found := &schedulingv1.PriorityClass{}
	err = kvpc.Client.Get(req.Ctx, key, found)

	if err != nil {
		if apierrors.IsNotFound(err) {
			// create the new object
			err = kvpc.Client.Create(req.Ctx, pc, &client.CreateOptions{})
			if err == nil {
				return res.SetCreated()
			}
		}

		return res.Error(err)
	}

	// at this point we found the object in the cache and we check if something was changed
	if pc.Name == found.Name && pc.Value == found.Value && pc.Description == found.Description {
		req.Logger.Info("KubeVirt PriorityClass already exists", "PriorityClass.Name", pc.Name)
		objectRef, err := reference.GetReference(kvpc.Scheme, found)
		if err != nil {
			req.Logger.Error(err, "failed getting object reference for found object")
			return res.Error(err)
		}
		objectreferencesv1.SetObjectReference(&req.Instance.Status.RelatedObjects, *objectRef)

		return res.SetUpgradeDone(req.ComponentUpgradeInProgress)
	}

	// something was changed but since we can't patch a priority class object, we remove it
	err = kvpc.Client.Delete(req.Ctx, found, &client.DeleteOptions{})
	if err != nil {
		return res.Error(err)
	}

	// create the new object
	err = kvpc.Client.Create(req.Ctx, pc, &client.CreateOptions{})
	if err != nil {
		return res.Error(err)
	}
	return res.SetUpdated()
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
