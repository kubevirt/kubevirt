package operands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/openshift/library-go/pkg/crypto"

	openshiftconfigv1 "github.com/openshift/api/config/v1"

	corev1 "k8s.io/api/core/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kubevirtcorev1 "kubevirt.io/api/core/v1"

	hcov1beta1 "github.com/kubevirt/hyperconverged-cluster-operator/api/v1beta1"
	"github.com/kubevirt/hyperconverged-cluster-operator/controllers/common"
	hcoutil "github.com/kubevirt/hyperconverged-cluster-operator/pkg/util"
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

	// Add downwardMetrics volume to expose a limited set of host metrics to guests
	kvDownwardMetricsGate = "DownwardMetrics"

	// Expand disks to the largest size
	kvExpandDisksGate = "ExpandDisks"

	// Allow automatic numa mapping on VMs with dedicated CPUs, if requested
	kvNUMA = "NUMA"

	// Export VMs to outside of the cluster
	kvVMExportGate = "VMExport"

	// Disable the installation and usage of the custom SELinux policy
	kvDisableCustomSELinuxPolicyGate = "DisableCustomSELinuxPolicy"
)

var (
	hardCodeKvFgs = []string{
		kvDataVolumesGate,
		kvSRIOVGate,
		kvCPUManagerGate,
		kvCPUNodeDiscoveryGate,
		kvSnapshotGate,
		kvHotplugVolumesGate,
		kvExpandDisksGate,
		kvGPUGate,
		kvHostDevicesGate,
		kvDownwardMetricsGate,
		kvNUMA,
		kvVMExportGate,
		kvDisableCustomSELinuxPolicyGate,
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
	kvNonRoot                = "NonRoot"
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

// KubeVirt containerDisk verification memory usage limit
var (
	kvDiskVerificationMemoryLimit, _ = resource.ParseQuantity("2G")
)

// ************  KubeVirt Handler  **************
type kubevirtHandler genericOperand

func newKubevirtHandler(Client client.Client, Scheme *runtime.Scheme) *kubevirtHandler {
	return &kubevirtHandler{
		Client:                 Client,
		Scheme:                 Scheme,
		crType:                 "KubeVirt",
		setControllerReference: true,
		hooks:                  &kubevirtHooks{},
	}
}

type kubevirtHooks struct {
	cache *kubevirtcorev1.KubeVirt
}

type rateLimits struct {
	Qps   float32
	Burst int
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

func (*kubevirtHooks) getEmptyCr() client.Object { return &kubevirtcorev1.KubeVirt{} }
func (*kubevirtHooks) getConditions(cr runtime.Object) []metav1.Condition {
	return translateKubeVirtConds(cr.(*kubevirtcorev1.KubeVirt).Status.Conditions)
}
func (*kubevirtHooks) checkComponentVersion(cr runtime.Object) bool {
	found := cr.(*kubevirtcorev1.KubeVirt)
	return checkComponentVersion(hcoutil.KubevirtVersionEnvV, found.Status.ObservedKubeVirtVersion)
}
func (h *kubevirtHooks) reset() {
	h.cache = nil
}

func (*kubevirtHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
	virt, ok1 := required.(*kubevirtcorev1.KubeVirt)
	found, ok2 := exists.(*kubevirtcorev1.KubeVirt)
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
		hcoutil.DeepCopyLabels(&virt.ObjectMeta, &found.ObjectMeta)
		virt.Spec.DeepCopyInto(&found.Spec)
		err := Client.Update(req.Ctx, found)
		if err != nil {
			return false, false, err
		}
		return true, !req.HCOTriggered, nil
	}
	return false, false, nil
}

func (*kubevirtHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

func NewKubeVirt(hc *hcov1beta1.HyperConverged, opts ...string) (*kubevirtcorev1.KubeVirt, error) {
	config, err := getKVConfig(hc)
	if err != nil {
		return nil, err
	}

	kvCertConfig := hcoCertConfig2KvCertificateRotateStrategy(hc.Spec.CertConfig)

	infrastructureHighlyAvailable := hcoutil.GetClusterInfo().IsInfrastructureHighlyAvailable()

	uninstallStrategy := kubevirtcorev1.KubeVirtUninstallStrategyBlockUninstallIfWorkloadsExist
	if hc.Spec.UninstallStrategy == hcov1beta1.HyperConvergedUninstallStrategyRemoveWorkloads {
		uninstallStrategy = kubevirtcorev1.KubeVirtUninstallStrategyRemoveWorkloads
	}

	spec := kubevirtcorev1.KubeVirtSpec{
		UninstallStrategy:           uninstallStrategy,
		Infra:                       hcoConfig2KvConfig(hc.Spec.Infra, infrastructureHighlyAvailable),
		Workloads:                   hcoConfig2KvConfig(hc.Spec.Workloads, true),
		Configuration:               *config,
		CertificateRotationStrategy: *kvCertConfig,
		WorkloadUpdateStrategy:      hcWorkloadUpdateStrategyToKv(&hc.Spec.WorkloadUpdateStrategy),
		ProductName:                 hcoutil.HyperConvergedCluster,
		ProductVersion:              os.Getenv(hcoutil.HcoKvIoVersionName),
		ProductComponent:            string(hcoutil.AppComponentCompute),
		ServiceMonitorNamespace:     getNamespace(hc.Namespace, opts),
	}

	kv := NewKubeVirtWithNameOnly(hc, opts...)
	kv.Spec = spec

	if err := applyPatchToSpec(hc, common.JSONPatchKVAnnotationName, kv); err != nil {
		return nil, err
	}

	return kv, nil
}

func hcoTuning2Kv(hc *hcov1beta1.HyperConverged) (*kubevirtcorev1.ReloadableComponentConfiguration, error) {
	var err error
	if hc.Spec.TuningPolicy == hcov1beta1.HyperConvergedAnnotationTuningPolicy {
		if annotation, ok := hc.Annotations[common.TuningPolicyAnnotationName]; ok {

			var rates rateLimits
			err := json.Unmarshal([]byte(annotation), &rates)
			if err != nil {
				return nil, err
			}

			if rates.Qps <= 0 {
				return nil, fmt.Errorf("qps parameter not found in annotation")
			}
			if rates.Burst <= 0 {
				return nil, fmt.Errorf("burst parameter not found in annotation")
			}

			return &kubevirtcorev1.ReloadableComponentConfiguration{
				RestClient: &kubevirtcorev1.RESTClientConfiguration{
					RateLimiter: &kubevirtcorev1.RateLimiter{
						TokenBucketRateLimiter: &kubevirtcorev1.TokenBucketRateLimiter{
							QPS:   rates.Qps,
							Burst: rates.Burst,
						},
					},
				},
			}, nil
		} else {
			err = fmt.Errorf("tuning policy set but annotation not present or wrong")
		}
	}
	return nil, err
}

func hcWorkloadUpdateStrategyToKv(hcObject *hcov1beta1.HyperConvergedWorkloadUpdateStrategy) kubevirtcorev1.KubeVirtWorkloadUpdateStrategy {
	kvObject := kubevirtcorev1.KubeVirtWorkloadUpdateStrategy{}
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
			kvObject.WorkloadUpdateMethods = make([]kubevirtcorev1.WorkloadUpdateMethod, size)
			for i, updateMethod := range hcObject.WorkloadUpdateMethods {
				kvObject.WorkloadUpdateMethods[i] = kubevirtcorev1.WorkloadUpdateMethod(updateMethod)
			}
		}
	}

	return kvObject
}

func getKVConfig(hc *hcov1beta1.HyperConverged) (*kubevirtcorev1.KubeVirtConfiguration, error) {
	devConfig, err := getKVDevConfig(hc)
	if err != nil {
		return nil, err
	}

	kvLiveMigration, err := hcLiveMigrationToKv(hc.Spec.LiveMigrationConfig)
	if err != nil {
		return nil, err
	}

	obsoleteCPUs, minCPUModel := getObsoleteCPUConfig(hc.Spec.ObsoleteCPUs)

	rateLimiter, err := hcoTuning2Kv(hc)
	if err != nil {
		return nil, err
	}

	config := &kubevirtcorev1.KubeVirtConfiguration{
		DeveloperConfiguration: devConfig,
		NetworkConfiguration: &kubevirtcorev1.NetworkConfiguration{
			NetworkInterface: string(kubevirtcorev1.MasqueradeInterface),
		},
		MigrationConfiguration:       kvLiveMigration,
		PermittedHostDevices:         toKvPermittedHostDevices(hc.Spec.PermittedHostDevices),
		MediatedDevicesConfiguration: toKvMediatedDevicesConfiguration(hc.Spec.MediatedDevicesConfiguration),
		ObsoleteCPUModels:            obsoleteCPUs,
		MinCPUModel:                  minCPUModel,
		TLSConfiguration:             hcTLSSecurityProfileToKv(hcoutil.GetClusterInfo().GetTLSSecurityProfile(hc.Spec.TLSSecurityProfile)),
		APIConfiguration:             rateLimiter,
		WebhookConfiguration:         rateLimiter,
		ControllerConfiguration:      rateLimiter,
		HandlerConfiguration:         rateLimiter,
	}

	if smbiosConfig, ok := os.LookupEnv(smbiosEnvName); ok {
		if smbiosConfig = strings.TrimSpace(smbiosConfig); smbiosConfig != "" {
			config.SMBIOSConfig = &kubevirtcorev1.SMBiosConfiguration{}
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

	if hc.Spec.DefaultCPUModel != nil {
		config.CPUModel = *hc.Spec.DefaultCPUModel
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

func toKvMediatedDevicesConfiguration(mdevsConfig *hcov1beta1.MediatedDevicesConfiguration) *kubevirtcorev1.MediatedDevicesConfiguration {
	if mdevsConfig == nil {
		return nil
	}

	return &kubevirtcorev1.MediatedDevicesConfiguration{
		MediatedDevicesTypes:    mdevsConfig.MediatedDevicesTypes,
		NodeMediatedDeviceTypes: toKvNodeMediatedDevicesConfiguration(mdevsConfig.NodeMediatedDeviceTypes),
	}
}

func toKvNodeMediatedDevicesConfiguration(hcoNodeMdevTypesConf []hcov1beta1.NodeMediatedDeviceTypesConfig) []kubevirtcorev1.NodeMediatedDeviceTypesConfig {
	if len(hcoNodeMdevTypesConf) > 0 {
		nodeMdevTypesConf := make([]kubevirtcorev1.NodeMediatedDeviceTypesConfig, 0, len(hcoNodeMdevTypesConf))
		for _, hcoNodeMdevTypeConf := range hcoNodeMdevTypesConf {
			nodeMdevTypesConf = append(nodeMdevTypesConf, kubevirtcorev1.NodeMediatedDeviceTypesConfig{
				NodeSelector:         hcoNodeMdevTypeConf.NodeSelector,
				MediatedDevicesTypes: hcoNodeMdevTypeConf.MediatedDevicesTypes,
			})
		}
		return nodeMdevTypesConf
	}

	return nil
}

func toKvPermittedHostDevices(permittedDevices *hcov1beta1.PermittedHostDevices) *kubevirtcorev1.PermittedHostDevices {
	if permittedDevices == nil {
		return nil
	}

	return &kubevirtcorev1.PermittedHostDevices{
		PciHostDevices:  toKvPciHostDevices(permittedDevices.PciHostDevices),
		MediatedDevices: toKvMediatedDevices(permittedDevices.MediatedDevices),
	}
}

func toKvPciHostDevices(hcoPciHostdevices []hcov1beta1.PciHostDevice) []kubevirtcorev1.PciHostDevice {
	if len(hcoPciHostdevices) > 0 {
		pciHostDevices := make([]kubevirtcorev1.PciHostDevice, 0, len(hcoPciHostdevices))
		for _, hcoPciHostDevice := range hcoPciHostdevices {
			if !hcoPciHostDevice.Disabled {
				pciHostDevices = append(pciHostDevices, kubevirtcorev1.PciHostDevice{
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

func toKvMediatedDevices(hcoMediatedDevices []hcov1beta1.MediatedHostDevice) []kubevirtcorev1.MediatedHostDevice {
	if len(hcoMediatedDevices) > 0 {
		mediatedDevices := make([]kubevirtcorev1.MediatedHostDevice, 0, len(hcoMediatedDevices))
		for _, hcoMediatedHostDevice := range hcoMediatedDevices {
			if !hcoMediatedHostDevice.Disabled {
				mediatedDevices = append(mediatedDevices, kubevirtcorev1.MediatedHostDevice{
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

func hcLiveMigrationToKv(lm hcov1beta1.LiveMigrationConfigurations) (*kubevirtcorev1.MigrationConfiguration, error) {
	var bandwidthPerMigration *resource.Quantity = nil
	if lm.BandwidthPerMigration != nil {
		bandwidthPerMigrationObject, err := resource.ParseQuantity(*lm.BandwidthPerMigration)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the LiveMigrationConfig.bandwidthPerMigration field; %w", err)
		}
		bandwidthPerMigration = &bandwidthPerMigrationObject
	}

	return &kubevirtcorev1.MigrationConfiguration{
		BandwidthPerMigration:             bandwidthPerMigration,
		CompletionTimeoutPerGiB:           lm.CompletionTimeoutPerGiB,
		ParallelOutboundMigrationsPerNode: lm.ParallelOutboundMigrationsPerNode,
		ParallelMigrationsPerCluster:      lm.ParallelMigrationsPerCluster,
		ProgressTimeout:                   lm.ProgressTimeout,
		Network:                           lm.Network,
		AllowAutoConverge:                 lm.AllowAutoConverge,
		AllowPostCopy:                     lm.AllowPostCopy,
	}, nil
}

func hcTLSProtocolVersionToKv(profileMinVersion openshiftconfigv1.TLSProtocolVersion) kubevirtcorev1.TLSProtocolVersion {
	switch profileMinVersion {
	case openshiftconfigv1.VersionTLS10:
		return kubevirtcorev1.VersionTLS10

	case openshiftconfigv1.VersionTLS11:
		return kubevirtcorev1.VersionTLS11

	case openshiftconfigv1.VersionTLS12:
		return kubevirtcorev1.VersionTLS12

	case openshiftconfigv1.VersionTLS13:
		return kubevirtcorev1.VersionTLS13

	default:
		return kubevirtcorev1.VersionTLS12
	}
}

func hcTLSSecurityProfileToKv(profile *openshiftconfigv1.TLSSecurityProfile) *kubevirtcorev1.TLSConfiguration {
	var profileCiphers []string
	var profileMinVersion openshiftconfigv1.TLSProtocolVersion

	if profile == nil {
		return nil
	}

	if profile.Custom != nil {
		profileCiphers = profile.Custom.TLSProfileSpec.Ciphers
		profileMinVersion = profile.Custom.TLSProfileSpec.MinTLSVersion
	} else {
		profileCiphers = openshiftconfigv1.TLSProfiles[profile.Type].Ciphers
		profileMinVersion = openshiftconfigv1.TLSProfiles[profile.Type].MinTLSVersion
	}

	return &kubevirtcorev1.TLSConfiguration{
		MinTLSVersion: hcTLSProtocolVersionToKv(profileMinVersion),
		Ciphers:       crypto.OpenSSLToIANACipherSuites(profileCiphers),
	}
}

func getKVDevConfig(hc *hcov1beta1.HyperConverged) (*kubevirtcorev1.DeveloperConfiguration, error) {
	devConf := &kubevirtcorev1.DeveloperConfiguration{
		DiskVerification: &kubevirtcorev1.DiskVerification{
			MemoryLimit: &kvDiskVerificationMemoryLimit,
		},
	}

	fgs := getKvFeatureGateList(&hc.Spec.FeatureGates)
	if len(fgs) > 0 {
		devConf.FeatureGates = fgs
	}
	if useKVMEmulation {
		devConf.UseEmulation = useKVMEmulation
	}
	if lv := hc.Spec.LogVerbosityConfig; lv != nil && lv.Kubevirt != nil {
		devConf.LogVerbosity = lv.Kubevirt.DeepCopy()
	}

	return devConf, nil
}

func NewKubeVirtWithNameOnly(hc *hcov1beta1.HyperConverged, opts ...string) *kubevirtcorev1.KubeVirt {
	return &kubevirtcorev1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt-" + hc.Name,
			Labels:    getLabels(hc, hcoutil.AppComponentCompute),
			Namespace: getNamespace(hc.Namespace, opts),
		},
	}
}

func hcoConfig2KvConfig(hcoConfig hcov1beta1.HyperConvergedConfig, infrastructureHighlyAvailable bool) *kubevirtcorev1.ComponentConfig {
	if hcoConfig.NodePlacement == nil && infrastructureHighlyAvailable {
		return nil
	}

	kvConfig := &kubevirtcorev1.ComponentConfig{}
	if !infrastructureHighlyAvailable {
		var singleReplica uint8 = 1
		kvConfig.Replicas = &singleReplica
	}

	if hcoConfig.NodePlacement != nil {
		kvConfig.NodePlacement = &kubevirtcorev1.NodePlacement{}

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
	}
	return kvConfig
}

func getFeatureGateChecks(featureGates *hcov1beta1.HyperConvergedFeatureGates) []string {
	fgs := make([]string, 0, 2)

	if featureGates.WithHostPassthroughCPU != nil && *featureGates.WithHostPassthroughCPU {
		fgs = append(fgs, kvWithHostPassthroughCPU)
	}

	if featureGates.NonRoot != nil && *featureGates.NonRoot {
		fgs = append(fgs, kvNonRoot)
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
		setControllerReference: false,
		hooks:                  &kvPriorityClassHooks{},
	}
}

type kvPriorityClassHooks struct{}

func (kvPriorityClassHooks) getFullCr(hc *hcov1beta1.HyperConverged) (client.Object, error) {
	return NewKubeVirtPriorityClass(hc), nil
}
func (kvPriorityClassHooks) getEmptyCr() client.Object { return &schedulingv1.PriorityClass{} }

func (kvPriorityClassHooks) updateCr(req *common.HcoRequest, Client client.Client, exists runtime.Object, required runtime.Object) (bool, bool, error) {
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
		req.Logger.Info("Updating existing PriorityClass's Spec to new opinionated values")
	} else {
		req.Logger.Info("Reconciling an externally updated PriorityClass's Spec to its opinionated values")
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

	pc.DeepCopyInto(found)

	return true, !req.HCOTriggered, nil
}

func (kvPriorityClassHooks) justBeforeComplete(_ *common.HcoRequest) { /* no implementation */ }

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
func translateKubeVirtConds(orig []kubevirtcorev1.KubeVirtCondition) []metav1.Condition {
	translated := make([]metav1.Condition, len(orig))

	for i, origCond := range orig {
		translated[i] = metav1.Condition{
			Type:    string(origCond.Type),
			Status:  metav1.ConditionStatus(origCond.Status),
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
	res := make([]string, 0, len(checks)+len(mandatoryKvFeatureGates)+1)
	res = append(res, mandatoryKvFeatureGates...)
	res = append(res, checks...)

	return res
}

func hcoCertConfig2KvCertificateRotateStrategy(hcoCertConfig hcov1beta1.HyperConvergedCertConfig) *kubevirtcorev1.KubeVirtCertificateRotateStrategy {
	return &kubevirtcorev1.KubeVirtCertificateRotateStrategy{
		SelfSigned: &kubevirtcorev1.KubeVirtSelfSignConfiguration{
			CA: &kubevirtcorev1.CertConfig{
				Duration:    hcoCertConfig.CA.Duration.DeepCopy(),
				RenewBefore: hcoCertConfig.CA.RenewBefore.DeepCopy(),
			},
			Server: &kubevirtcorev1.CertConfig{
				Duration:    hcoCertConfig.Server.Duration.DeepCopy(),
				RenewBefore: hcoCertConfig.Server.RenewBefore.DeepCopy(),
			},
		},
	}
}
