package operands

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	conditionsv1 "github.com/openshift/custom-resource-status/conditions/v1"
	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	kubevirtv1 "kubevirt.io/client-go/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/pkg/apis/hco/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/controller/common"
	"github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
)

const (
	SELinuxLauncherType = "virt_launcher.process"
)

// env vars
const (
	kvmEmulationEnvName = "KVM_EMULATION"
	smbiosEnvName       = "SMBIOS"
	machineTypeEnvName  = "MACHINETYPE"
)

var (
	useKVMEmulation = false
)

func init() {
	kvmEmulationStr, varExists := os.LookupEnv(kvmEmulationEnvName)
	if varExists {
		isKVMEmulation, err := strconv.ParseBool(strings.ToLower(kvmEmulationStr))
		useKVMEmulation = err == nil && isKVMEmulation
	}

	mandatoryKvFeatureGates = getMandatoryKvFeatureGates(useKVMEmulation)
}

// KubeVirt hard coded FeatureGates
// These feature gates are set by HCO in the KubeVirt CR and can't be modified by the end user.
const (
	// indicates that we support turning on DataVolume workflows. This means using DataVolumes in the VM and VMI
	// definitions. There was a period of time where this was in alpha and needed to be explicility enabled.
	// It also means that someone is using KubeVirt with CDI. So by not enabling this feature gate, someone can safely
	// use kubevirt without CDI and know that users of kubevirt will not be able to post VM/VMIs that use CDI workflows
	// that aren't available to them
	kvDataVolumesGate = "DataVolumes"

	// Enable Single-root input/output virtualization
	kvSRIOVGate = "SRIOV"

	// Enables VMIs to be live migrated. Without this, migrations are not possible and will be blocked
	kvLiveMigrationGate = "LiveMigration"

	// Enables the CPUManager feature gate to label the nodes which have the Kubernetes CPUManager running. VMIs that
	// require dedicated CPU resources will automatically be scheduled on the labeled nodes
	kvCPUManagerGate = "CPUManager"

	// Enables schedule VMIs according to their CPU model
	kvCPUNodeDiscoveryGate = "CPUNodeDiscovery"

	// Enables the alpha offline snapshot functionality
	kvSnapshotGate = "Snapshot"

	// Allow attaching a data volume to a running VMI
	kvHotplugVolumesGate = "HotplugVolumes"

	// Allow assigning GPU and vGPU devices to virtual machines
	kvGPUGate = "GPU"

	// Allow assigning host devices to virtual machines
	kvHostDevicesGate = "HostDevices"
)

var (
	hardCodeKvFgs = []string{
		kvDataVolumesGate,
		kvSRIOVGate,
		kvLiveMigrationGate,
		kvCPUManagerGate,
		kvCPUNodeDiscoveryGate,
		kvSnapshotGate,
		kvHotplugVolumesGate,
		kvGPUGate,
		kvHostDevicesGate,
	}

	// holds a list of mandatory KubeVirt feature gates. Some of them are the hard coded feature gates and some of
	// them are added according to conditions; e.g. if SSP is deployed.
	mandatoryKvFeatureGates []string
)

// These KubeVirt feature gates are automatically enabled in KubeVirt if SSP is deployed
const (
	// Support migration for VMs with host-model CPU mode
	kvWithHostModelCPU = "WithHostModelCPU"

	// Enable HyperV strict host checking for HyperV enlightenments
	kvHypervStrictCheck = "HypervStrictCheck"
)

var (
	sspConditionKvFgs = []string{
		kvWithHostModelCPU,
		kvHypervStrictCheck,
	}
)

// KubeVirt feature gates that are exposed in HCO API
const (
	kvWithHostPassthroughCPU = "WithHostPassthroughCPU"
	kvSRIOVLiveMigration     = "SRIOVLiveMigration"
)

// CPU Plugin default values
var (
	hardcodedObsoleteCPUModels = []string{
		"486",
		"pentium",
		"pentium2",
		"pentium3",
		"pentiumpro",
		"coreduo",
		"n270",
		"core2duo",
		"Conroe",
		"athlon",
		"phenom",
		"qemu64",
		"qemu32",
		"kvm64",
		"kvm32",
	}
)

// ************  KubeVirt Handler  **************
type kubevirtHandler genericOperand

func newKubevirtHandler(Client client.Client, Scheme *runtime.Scheme) *kubevirtHandler {
	return &kubevirtHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirt",
		removeExistingOwner:    false,
		setControllerReference: true,
		hooks:                  &kubevirtHooks{},
	}
}

type kubevirtHooks struct {
	cache *kubevirtv1.KubeVirt
}

func (h *kubevirtHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	if h.cache == nil {
		kv, err := NewKubeVirt(hc)
		if err != nil {
			return nil, err
		}
		h.cache = kv
	}
	return h.cache, nil
}

func (h kubevirtHooks) getEmptyCr() client.Object                          { return &kubevirtv1.KubeVirt{} }
func (h kubevirtHooks) postFound(*common.HcoRequest, runtime.Object) error { return nil }
func (h kubevirtHooks) getConditions(cr runtime.Object) []conditionsv1.Condition {
	return translateKubeVirtConds(cr.(*kubevirtv1.KubeVirt).Status.Conditions)
}
func (h kubevirtHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*kubevirtv1.KubeVirt)
	return checkComponentVersion(hcoutil.KubevirtVersionEnvV, found.Status.ObservedKubeVirtVersion)
}
func (h kubevirtHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*kubevirtv1.KubeVirt).ObjectMeta
}
func (h *kubevirtHooks) reset() {
	h.cache = nil
}

func (h *kubevirtHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	virt, ok1 := required.(*kubevirtv1.KubeVirt)
	found, ok2 := exists.(*kubevirtv1.KubeVirt)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to KubeVirt")
	}
	if !reflect.DeepEqual(found.Spec, virt.Spec) ||
		!reflect.DeepEqual(found.Labels, virt.Labels) {
		if req.HCOTriggered {
			req.Logger.Info("Updating existing KubeVirt's Spec to new opinionated values")
		} else {
			req.Logger.Info("Reconciling an externally updated KubeVirt's Spec to its opinionated values")
		}
		util.DeepCopyLabels(&virt.ObjectMeta, &found.ObjectMeta)
		virt.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func NewKubeVirt(hc *hcov1beta1.HyperConverged, opts ...string) (*kubevirtv1.KubeVirt, error) {
	config, err := getKVConfig(hc)
	if err != nil {
		return nil, err
	}

	kvCertConfig := hcoCertConfig2KvCertificateRotateStrategy(hc.Spec.CertConfig)

	spec := kubevirtv1.KubeVirtSpec{
		UninstallStrategy:           kubevirtv1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist,
		Infra:                       hcoConfig2KvConfig(hc.Spec.Infra),
		Workloads:                   hcoConfig2KvConfig(hc.Spec.Workloads),
		Configuration:               *config,
		CertificateRotationStrategy: *kvCertConfig,
		WorkloadUpdateStrategy:      hcWorkloadUpdateStrategyToKv(hc.Spec.WorkloadUpdateStrategy),
	}

	kv := NewKubeVirtWithNameOnly(hc, opts...)
	kv.Spec = spec

	if err := applyPatchToSpec(hc, common.JSONPatchKVAnnotationName, kv); err != nil {
		return nil, err
	}

	return kv, nil
}

func hcWorkloadUpdateStrategyToKv(hcObject *hcov1beta1.HyperConvergedWorkloadUpdateStrategy) kubevirtv1.KubeVirtWorkloadUpdateStrategy {
	kvObject := kubevirtv1.KubeVirtWorkloadUpdateStrategy{}
	if hcObject != nil {
		if hcObject.BatchEvictionInterval != nil {
			kvObject.BatchEvictionInterval = new(metav1.Duration)
			*kvObject.BatchEvictionInterval = *hcObject.BatchEvictionInterval
		}

		if hcObject.BatchEvictionSize != nil {
			kvObject.BatchEvictionSize = new(int)
			*kvObject.BatchEvictionSize = *hcObject.BatchEvictionSize
		}

		if size := len(hcObject.WorkloadUpdateMethods); size > 0 {
			kvObject.WorkloadUpdateMethods = make([]kubevirtv1.WorkloadUpdateMethod, size)
			for i, updateMethod := range hcObject.WorkloadUpdateMethods {
				kvObject.WorkloadUpdateMethods[i] = kubevirtv1.WorkloadUpdateMethod(updateMethod)
			}
		}
	}

	return kvObject
}

func getKVConfig(hc *hcov1beta1.HyperConverged) (*kubevirtv1.KubeVirtConfiguration, error) {
	devConfig, err := getKVDevConfig(hc)
	if err != nil {
		return nil, err
	}

	kvLiveMigration, err := hcLiveMigrationToKv(hc.Spec.LiveMigrationConfig)
	if err != nil {
		return nil, err
	}

	obsoleteCPUs, minCPUModel := getObsoleteCPUConfig(hc.Spec.ObsoleteCPUs)

	config := &kubevirtv1.KubeVirtConfiguration{
		DeveloperConfiguration: devConfig,
		SELinuxLauncherType:    SELinuxLauncherType,
		NetworkConfiguration: &kubevirtv1.NetworkConfiguration{
			NetworkInterface: string(kubevirtv1.MasqueradeInterface),
		},
		MigrationConfiguration: kvLiveMigration,
		PermittedHostDevices:   toKvPermittedHostDevices(hc.Spec.PermittedHostDevices),
		ObsoleteCPUModels:      obsoleteCPUs,
		MinCPUModel:            minCPUModel,
	}

	if smbiosConfig, ok := os.LookupEnv(smbiosEnvName); ok {
		if smbiosConfig = strings.TrimSpace(smbiosConfig); smbiosConfig != "" {
			config.SMBIOSConfig = &kubevirtv1.SMBiosConfiguration{}
			err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(smbiosConfig), 1024).Decode(config.SMBIOSConfig)
			if err != nil {
				return nil, err
			}
		}
	}

	if val, ok := os.LookupEnv(machineTypeEnvName); ok {
		if val = strings.TrimSpace(val); val != "" {
			config.MachineType = val
		}
	}

	return config, nil
}

func getObsoleteCPUConfig(hcObsoleteCPUConf *hcov1beta1.HyperConvergedObsoleteCPUs) (map[string]bool, string) {
	obsoleteCPUModels := make(map[string]bool)
	for _, cpu := range hardcodedObsoleteCPUModels {
		obsoleteCPUModels[cpu] = true
	}
	minCPUModel := ""

	if hcObsoleteCPUConf != nil {
		for _, cpu := range hcObsoleteCPUConf.CPUModels {
			obsoleteCPUModels[cpu] = true
		}

		minCPUModel = hcObsoleteCPUConf.MinCPUModel
	}

	return obsoleteCPUModels, minCPUModel
}

func toKvPermittedHostDevices(permittedDevices *hcov1beta1.PermittedHostDevices) *kubevirtv1.PermittedHostDevices {
	if permittedDevices == nil {
		return nil
	}

	return &kubevirtv1.PermittedHostDevices{
		PciHostDevices:  toKvPciHostDevices(permittedDevices.PciHostDevices),
		MediatedDevices: toKvMediatedDevices(permittedDevices.MediatedDevices),
	}
}

func toKvPciHostDevices(hcoPciHostdevices []hcov1beta1.PciHostDevice) []kubevirtv1.PciHostDevice {
	if len(hcoPciHostdevices) > 0 {
		pciHostDevices := make([]kubevirtv1.PciHostDevice, 0, len(hcoPciHostdevices))
		for _, hcoPciHostDevice := range hcoPciHostdevices {
			if !hcoPciHostDevice.Disabled {
				pciHostDevices = append(pciHostDevices, kubevirtv1.PciHostDevice{
					PCIVendorSelector:        hcoPciHostDevice.PCIDeviceSelector,
					ResourceName:             hcoPciHostDevice.ResourceName,
					ExternalResourceProvider: hcoPciHostDevice.ExternalResourceProvider,
				})
			}
		}

		return pciHostDevices
	}
	return nil
}

func toKvMediatedDevices(hcoMediatedDevices []hcov1beta1.MediatedHostDevice) []kubevirtv1.MediatedHostDevice {
	if len(hcoMediatedDevices) > 0 {
		mediatedDevices := make([]kubevirtv1.MediatedHostDevice, 0, len(hcoMediatedDevices))
		for _, hcoMediatedHostDevice := range hcoMediatedDevices {
			if !hcoMediatedHostDevice.Disabled {
				mediatedDevices = append(mediatedDevices, kubevirtv1.MediatedHostDevice{
					MDEVNameSelector:         hcoMediatedHostDevice.MDEVNameSelector,
					ResourceName:             hcoMediatedHostDevice.ResourceName,
					ExternalResourceProvider: hcoMediatedHostDevice.ExternalResourceProvider,
				})
			}
		}

		return mediatedDevices
	}
	return nil
}

func hcLiveMigrationToKv(lm hcov1beta1.LiveMigrationConfigurations) (*kubevirtv1.MigrationConfiguration, error) {
	var bandwidthPerMigration *resource.Quantity = nil
	if lm.BandwidthPerMigration != nil {
		bandwidthPerMigrationObject, err := resource.ParseQuantity(*lm.BandwidthPerMigration)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the LiveMigrationConfig.bandwidthPerMigration field; %w", err)
		}
		bandwidthPerMigration = &bandwidthPerMigrationObject
	}

	return &kubevirtv1.MigrationConfiguration{
		BandwidthPerMigration:             bandwidthPerMigration,
		CompletionTimeoutPerGiB:           lm.CompletionTimeoutPerGiB,
		ParallelOutboundMigrationsPerNode: lm.ParallelOutboundMigrationsPerNode,
		ParallelMigrationsPerCluster:      lm.ParallelMigrationsPerCluster,
		ProgressTimeout:                   lm.ProgressTimeout,
	}, nil
}

func getKVDevConfig(hc *hcov1beta1.HyperConverged) (*kubevirtv1.DeveloperConfiguration, error) {
	fgs := getKvFeatureGateList(&hc.Spec.FeatureGates)

	if len(fgs) > 0 || useKVMEmulation {
		return &kubevirtv1.DeveloperConfiguration{
			FeatureGates: fgs,
			UseEmulation: useKVMEmulation,
		}, nil
	}

	return nil, nil
}

func NewKubeVirtWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *kubevirtv1.KubeVirt {
	return &kubevirtv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentCompute),
			Namespace: getNamespace(hc.Namespace, opts),
		},
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

func getFeatureGateChecks(featureGates *hcov1beta1.HyperConvergedFeatureGates) []string {
	fgs := make([]string, 0, 2)

	if featureGates.WithHostPassthroughCPU {
		fgs = append(fgs, kvWithHostPassthroughCPU)
	}

	if featureGates.SRIOVLiveMigration {
		fgs = append(fgs, kvSRIOVLiveMigration)
	}

	return fgs
}

// ***********  KubeVirt Priority Class  ************
type kvPriorityClassHandler genericOperand

func newKvPriorityClassHandler(Client client.Client, Scheme *runtime.Scheme) *kvPriorityClassHandler {
	return &kvPriorityClassHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirtPriorityClass",
		removeExistingOwner:    false,
		setControllerReference: false,
		hooks:                  &kvPriorityClassHooks{},
	}
}

type kvPriorityClassHooks struct{}

func (h kvPriorityClassHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewKubeVirtPriorityClass(hc), nil
}
func (h kvPriorityClassHooks) getEmptyCr() client.Object                              { return &schedulingv1.PriorityClass{} }
func (h kvPriorityClassHooks) postFound(_ *common.HcoRequest, _ runtime.Object) error { return nil }
func (h kvPriorityClassHooks) getObjectMeta(cr runtime.Object) *metav1.ObjectMeta {
	return &cr.(*schedulingv1.PriorityClass).ObjectMeta
}

func (h *kvPriorityClassHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	pc, ok1 := required.(*schedulingv1.PriorityClass)
	found, ok2 := exists.(*schedulingv1.PriorityClass)
	if !ok1 || !ok2 {
		return false, false, errors.New("can't convert to PriorityClass")
	}

	// at this point we found the object in the cache and we check if something was changed
	if (pc.Name == found.Name) && (pc.Value == found.Value) &&
		(pc.Description == found.Description) && reflect.DeepEqual(pc.Labels, found.Labels) {
		return false, false, nil
	}

	if req.HCOTriggered {
		req.Logger.Info("Updating existing KubeVirt's Spec to new opinionated values")
	} else {
		req.Logger.Info("Reconciling an externally updated KubeVirt's Spec to its opinionated values")
	}

	// something was changed but since we can't patch a priority class object, we remove it
	err := Client.Delete(req.Ctx, found, &client.DeleteOptions{})
	if err != nil {
		return false, false, err
	}

	// create the new object
	err = Client.Create(req.Ctx, pc, &client.CreateOptions{})
	if err != nil {
		return false, false, err
	}

	return true, !req.HCOTriggered, nil
}

func NewKubeVirtPriorityClass(hc *hcov1beta1.HyperConverged) *schedulingv1.PriorityClass {
	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "scheduling.k8s.io/v1",
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   "kubevirt-cluster-critical",
			Labels: getLabels(hc, hcoutil.AppComponentCompute),
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

func getMandatoryKvFeatureGates(isKVMEmulation bool) []string {
	mandatoryFeatureGates := hardCodeKvFgs

	if !isKVMEmulation {
		mandatoryFeatureGates = append(mandatoryFeatureGates, sspConditionKvFgs...)
	}

	return mandatoryFeatureGates
}

// get list of feature gates or KV FG list
func getKvFeatureGateList(fgs *hcov1beta1.HyperConvergedFeatureGates) []string {
	checks := getFeatureGateChecks(fgs)
	res := make([]string, 0, len(checks)+len(mandatoryKvFeatureGates))
	res = append(res, mandatoryKvFeatureGates...)
	res = append(res, checks...)

	return res
}

func hcoCertConfig2KvCertificateRotateStrategy(hcoCertConfig hcov1beta1.HyperConvergedCertConfig) *kubevirtv1.KubeVirtCertificateRotateStrategy {
	return &kubevirtv1.KubeVirtCertificateRotateStrategy{
		SelfSigned: &kubevirtv1.KubeVirtSelfSignConfiguration{
			CA: &kubevirtv1.CertConfig{
				Duration:    hcoCertConfig.CA.Duration.DeepCopy(),
				RenewBefore: hcoCertConfig.CA.RenewBefore.DeepCopy(),
			},
			Server: &kubevirtv1.CertConfig{
				Duration:    hcoCertConfig.Server.Duration.DeepCopy(),
				RenewBefore: hcoCertConfig.Server.RenewBefore.DeepCopy(),
			},
		},
	}
}
