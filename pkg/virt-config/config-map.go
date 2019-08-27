/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017, 2018 Red Hat, Inc.
 *
 */

package virtconfig

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	extv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/cache"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	clientutil "kubevirt.io/client-go/util"
)

const (
	configMapName             = "kubevirt-config"
	FeatureGatesKey           = "feature-gates"
	EmulatedMachinesKey       = "emulated-machines"
	MachineTypeKey            = "machine-type"
	useEmulationKey           = "debug.useEmulation"
	ImagePullPolicyKey        = "dev.imagePullPolicy"
	MigrationsConfigKey       = "migrations"
	CpuModelKey               = "default-cpu-model"
	CpuRequestKey             = "cpu-request"
	MemoryOvercommitKey       = "memory-overcommit"
	LessPVCSpaceTolerationKey = "pvc-tolerate-less-space-up-to-percent"
	NodeSelectorsKey          = "node-selectors"
	NetworkInterfaceKey       = "default-network-interface"
	PermitSlirpInterface      = "permitSlirpInterface"
	NodeDrainTaintDefaultKey  = "kubevirt.io/drain"
)

type ConfigModifiedFn func()

func getConfigMap() *k8sv1.ConfigMap {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	var cfgMap *k8sv1.ConfigMap
	err = wait.PollImmediate(time.Second*1, time.Second*10, func() (bool, error) {

		namespace, curErr := clientutil.GetNamespace()
		if err != nil {
			return false, err
		}

		cfgMap, curErr = virtClient.CoreV1().ConfigMaps(namespace).Get(configMapName, metav1.GetOptions{})

		if curErr != nil {
			if errors.IsNotFound(curErr) {
				logger := log.DefaultLogger()
				logger.Infof("%s ConfigMap does not exist. Using defaults.", configMapName)
				cfgMap = &k8sv1.ConfigMap{}
				return true, nil
			}
			return false, curErr
		}

		return true, nil
	})

	if err != nil {
		panic(err)
	}

	return cfgMap
}

// NewClusterConfig represents the `kubevirt-config` config map. It can be used to live-update
// values if the config changes. The config update works like this:
// 1. Check if the config exists. If it does not exist, return the default config
// 2. Check if the config got updated. If so, try to parse and return it
// 3. In case of errors or no updates (resource version stays the same), it returns the values from the last good config
func NewClusterConfig(configMapInformer cache.SharedIndexInformer, crdInformer cache.SharedIndexInformer, namespace string) *ClusterConfig {

	c := &ClusterConfig{
		configMapInformer: configMapInformer,
		crdInformer:       crdInformer,
		lock:              &sync.Mutex{},
		namespace:         namespace,
		lastValidConfig:   defaultClusterConfig(),
		defaultConfig:     defaultClusterConfig(),
	}

	c.configMapInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.configAddedDeleted,
		DeleteFunc: c.configAddedDeleted,
		UpdateFunc: c.configUpdated,
	})

	c.crdInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.crdAddedDeleted,
		DeleteFunc: c.crdAddedDeleted,
		UpdateFunc: c.crdUpdated,
	})

	return c
}

func (c *ClusterConfig) configAddedDeleted(obj interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		go c.configModifiedCallback()
	}
}
func (c *ClusterConfig) configUpdated(old, cur interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		go c.configModifiedCallback()
	}
}

func isDataVolumeCrd(crd *extv1beta1.CustomResourceDefinition) bool {
	if crd.Spec.Names.Kind == "DataVolume" {
		return true
	}

	return false

}

func (c *ClusterConfig) crdAddedDeleted(obj interface{}) {
	crd := obj.(*extv1beta1.CustomResourceDefinition)
	if !isDataVolumeCrd(crd) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		go c.configModifiedCallback()
	}
}

func (c *ClusterConfig) crdUpdated(old, cur interface{}) {
	c.crdAddedDeleted(cur)
}

func defaultClusterConfig() *Config {
	parallelOutboundMigrationsPerNodeDefault := ParallelOutboundMigrationsPerNodeDefault
	parallelMigrationsPerClusterDefault := ParallelMigrationsPerClusterDefault
	bandwithPerMigrationDefault := resource.MustParse(BandwithPerMigrationDefault)
	nodeDrainTaintDefaultKey := NodeDrainTaintDefaultKey
	allowAutoConverge := MigrationAllowAutoConverge
	progressTimeout := MigrationProgressTimeout
	completionTimeoutPerGiB := MigrationCompletionTimeoutPerGiB
	cpuRequestDefault := resource.MustParse(DefaultCPURequest)
	emulatedMachinesDefault := strings.Split(DefaultEmulatedMachines, ",")
	nodeSelectorsDefault, _ := parseNodeSelectors(DefaultNodeSelectors)
	defaultNetworkInterface := DefaultNetworkInterface
	return &Config{
		ResourceVersion: "0",
		ImagePullPolicy: DefaultImagePullPolicy,
		UseEmulation:    DefaultUseEmulation,
		MigrationConfig: &MigrationConfig{
			ParallelMigrationsPerCluster:      &parallelMigrationsPerClusterDefault,
			ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNodeDefault,
			BandwidthPerMigration:             &bandwithPerMigrationDefault,
			NodeDrainTaintKey:                 &nodeDrainTaintDefaultKey,
			ProgressTimeout:                   &progressTimeout,
			CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
			UnsafeMigrationOverride:           DefaultUnsafeMigrationOverride,
			AllowAutoConverge:                 allowAutoConverge,
		},
		MachineType:            DefaultMachineType,
		CPURequest:             cpuRequestDefault,
		MemoryOvercommit:       DefaultMemoryOvercommit,
		EmulatedMachines:       emulatedMachinesDefault,
		LessPVCSpaceToleration: DefaultLessPVCSpaceToleration,
		NodeSelectors:          nodeSelectorsDefault,
		NetworkInterface:       defaultNetworkInterface,
		PermitSlirpInterface:   DefaultPermitSlirpInterface,
	}
}

type Config struct {
	ResourceVersion        string
	UseEmulation           bool
	MigrationConfig        *MigrationConfig
	ImagePullPolicy        k8sv1.PullPolicy
	MachineType            string
	CPUModel               string
	CPURequest             resource.Quantity
	MemoryOvercommit       int
	EmulatedMachines       []string
	FeatureGates           string
	LessPVCSpaceToleration int
	NodeSelectors          map[string]string
	NetworkInterface       string
	PermitSlirpInterface   bool
}

type MigrationConfig struct {
	ParallelOutboundMigrationsPerNode *uint32            `json:"parallelOutboundMigrationsPerNode,omitempty"`
	ParallelMigrationsPerCluster      *uint32            `json:"parallelMigrationsPerCluster,omitempty"`
	BandwidthPerMigration             *resource.Quantity `json:"bandwidthPerMigration,omitempty"`
	NodeDrainTaintKey                 *string            `json:"nodeDrainTaintKey,omitempty"`
	ProgressTimeout                   *int64             `json:"progressTimeout,omitempty"`
	CompletionTimeoutPerGiB           *int64             `json:"completionTimeoutPerGiB,omitempty"`
	UnsafeMigrationOverride           bool               `json:"unsafeMigrationOverride"`
	AllowAutoConverge                 bool               `json:"allowAutoConverge"`
}

type ClusterConfig struct {
	configMapInformer                cache.SharedIndexInformer
	crdInformer                      cache.SharedIndexInformer
	namespace                        string
	lock                             *sync.Mutex
	lastValidConfig                  *Config
	defaultConfig                    *Config
	lastInvalidConfigResourceVersion string
	configModifiedCallback           ConfigModifiedFn
}

func (c *ClusterConfig) SetConfigModifiedCallback(cb ConfigModifiedFn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.configModifiedCallback = cb
	go c.configModifiedCallback()
}

// setConfig parses the provided config map and updates the provided config.
// Default values in the provided config stay in tact.
func setConfig(config *Config, configMap *k8sv1.ConfigMap) error {

	// set revision
	config.ResourceVersion = configMap.ResourceVersion

	// set migration options
	rawConfig := strings.TrimSpace(configMap.Data[MigrationsConfigKey])
	if rawConfig != "" {
		// only sets values if they were specified, default values stay intact
		err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(rawConfig), 1024).Decode(config.MigrationConfig)
		if err != nil {
			return fmt.Errorf("failed to parse migration config: %v", err)
		}
	}

	// set image pull policy
	policy := strings.TrimSpace(configMap.Data[ImagePullPolicyKey])
	switch policy {
	case "":
		// keep the default
	case "Always":
		config.ImagePullPolicy = k8sv1.PullAlways
	case "Never":
		config.ImagePullPolicy = k8sv1.PullNever
	case "IfNotPresent":
		config.ImagePullPolicy = k8sv1.PullIfNotPresent
	default:
		return fmt.Errorf("invalid dev.imagePullPolicy in config: %v", policy)
	}

	// set if emulation is used
	useEmulation := strings.TrimSpace(configMap.Data[useEmulationKey])
	switch useEmulation {
	case "":
		// keep the default
	case "true":
		config.UseEmulation = true
	case "false":
		config.UseEmulation = false
	default:
		return fmt.Errorf("invalid debug.useEmulation in config: %v", useEmulation)
	}

	// set machine type
	if machineType := strings.TrimSpace(configMap.Data[MachineTypeKey]); machineType != "" {
		config.MachineType = machineType
	}

	if cpuModel := strings.TrimSpace(configMap.Data[CpuModelKey]); cpuModel != "" {
		config.CPUModel = cpuModel
	}

	if cpuRequest := strings.TrimSpace(configMap.Data[CpuRequestKey]); cpuRequest != "" {
		config.CPURequest = resource.MustParse(cpuRequest)
	}

	if memoryOvercommit := strings.TrimSpace(configMap.Data[MemoryOvercommitKey]); memoryOvercommit != "" {
		if value, err := strconv.Atoi(memoryOvercommit); err == nil && value > 0 {
			config.MemoryOvercommit = value
		} else {
			return fmt.Errorf("Invalid memoryOvercommit in ConfigMap: %s", memoryOvercommit)
		}
	}

	if emulatedMachines := strings.TrimSpace(configMap.Data[EmulatedMachinesKey]); emulatedMachines != "" {
		vals := strings.Split(emulatedMachines, ",")
		for i := range vals {
			vals[i] = strings.TrimSpace(vals[i])
		}
		config.EmulatedMachines = vals
	}

	if featureGates := strings.TrimSpace(configMap.Data[FeatureGatesKey]); featureGates != "" {
		config.FeatureGates = featureGates
	}

	if toleration := strings.TrimSpace(configMap.Data[LessPVCSpaceTolerationKey]); toleration != "" {
		if value, err := strconv.Atoi(toleration); err != nil || value < 0 || value > 100 {
			return fmt.Errorf("Invalid lessPVCSpaceToleration in ConfigMap: %s", toleration)
		} else {
			config.LessPVCSpaceToleration = value
		}
	}

	if nodeSelectors := strings.TrimSpace(configMap.Data[NodeSelectorsKey]); nodeSelectors != "" {
		if selectors, err := parseNodeSelectors(nodeSelectors); err != nil {
			return err
		} else {
			config.NodeSelectors = selectors
		}
	}

	// disable slirp
	permitSlirp := strings.TrimSpace(configMap.Data[PermitSlirpInterface])
	switch permitSlirp {
	case "":
		// keep the default
	case "true":
		config.PermitSlirpInterface = true
	case "false":
		config.PermitSlirpInterface = false
	default:
		return fmt.Errorf("invalid value for permitSlirpInterfaces in config: %v", permitSlirp)
	}

	// set default network interface
	iface := strings.TrimSpace(configMap.Data[NetworkInterfaceKey])
	switch iface {
	case "":
		// keep the default
	case string(v1.BridgeInterface), string(v1.SlirpInterface), string(v1.MasqueradeInterface):
		config.NetworkInterface = iface
	default:
		return fmt.Errorf("invalid default-network-interface in config: %v", iface)
	}
	return nil
}

// getConfig returns the latest valid parsed config map result, or updates it
// if a newer version is available.
// XXX Rework this, to happen mostly in informer callbacks.
// This will also allow us then to react to config changes and e.g. restart some controllers
func (c *ClusterConfig) getConfig() (config *Config) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if obj, exists, err := c.configMapInformer.GetStore().GetByKey(c.namespace + "/" + configMapName); err != nil {
		log.DefaultLogger().Reason(err).Errorf("Error loading the cluster config from cache, falling back to last good resource version '%s'", c.lastValidConfig.ResourceVersion)
		return c.lastValidConfig
	} else if !exists {
		return c.defaultConfig
	} else {
		configMap := obj.(*k8sv1.ConfigMap)
		if c.lastValidConfig.ResourceVersion == configMap.ResourceVersion ||
			c.lastInvalidConfigResourceVersion == configMap.ResourceVersion {
			return c.lastValidConfig
		}
		config := defaultClusterConfig()
		if err := setConfig(config, configMap); err != nil {
			c.lastInvalidConfigResourceVersion = configMap.ResourceVersion
			log.DefaultLogger().Reason(err).Errorf("Invalid cluster config with resource version '%s', falling back to last good resource version '%s'", configMap.ResourceVersion, c.lastValidConfig.ResourceVersion)
			return c.lastValidConfig
		}
		log.DefaultLogger().Infof("Updating cluster config to resource version '%s'", configMap.ResourceVersion)
		c.lastValidConfig = config
		return c.lastValidConfig
	}
}

func (c *ClusterConfig) HasDataVolumeAPI() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	objects := c.crdInformer.GetStore().List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1beta1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			if isDataVolumeCrd(crd) {
				return true
			}
		}
	}
	return false
}

func parseNodeSelectors(str string) (map[string]string, error) {
	nodeSelectors := make(map[string]string)
	for _, s := range strings.Split(strings.TrimSpace(str), "\n") {
		v := strings.Split(s, "=")
		if len(v) != 2 {
			return nil, fmt.Errorf("Invalid node selector: %s", s)
		}
		nodeSelectors[v[0]] = v[1]
	}
	return nodeSelectors, nil
}
