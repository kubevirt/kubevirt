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
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"sync"

	extv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
)

const (
	NodeDrainTaintDefaultKey = "kubevirt.io/drain"
)

type ConfigModifiedFn func()

// NewClusterConfig is a wrapper of NewClusterConfigWithCPUArch with default cpuArch.
func NewClusterConfig(crdInformer cache.SharedIndexInformer,
	kubeVirtInformer cache.SharedIndexInformer,
	namespace string) *ClusterConfig {
	return NewClusterConfigWithCPUArch(
		crdInformer,
		kubeVirtInformer,
		namespace,
		runtime.GOARCH,
	)
}

// NewClusterConfigWithCPUArch represents the `kubevirt-config` config map. It can be used to live-update
// values if the config changes. The config update works like this:
// 1. Check if the config exists. If it does not exist, return the default config
// 2. Check if the config got updated. If so, try to parse and return it
// 3. In case of errors or no updates (resource version stays the same), it returns the values from the last good config
func NewClusterConfigWithCPUArch(crdInformer cache.SharedIndexInformer,
	kubeVirtInformer cache.SharedIndexInformer,
	namespace, cpuArch string) *ClusterConfig {

	defaultConfig := defaultClusterConfig(cpuArch)

	c := &ClusterConfig{
		crdInformer:      crdInformer,
		kubeVirtInformer: kubeVirtInformer,
		cpuArch:          cpuArch,
		lock:             &sync.Mutex{},
		namespace:        namespace,
		lastValidConfig:  defaultConfig,
		defaultConfig:    defaultConfig,
	}

	c.crdInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.crdAddedDeleted,
		DeleteFunc: c.crdAddedDeleted,
		UpdateFunc: c.crdUpdated,
	})

	c.kubeVirtInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.configAddedDeleted,
		UpdateFunc: c.configUpdated,
	})

	return c
}

func (c *ClusterConfig) configAddedDeleted(_ interface{}) {
	go c.GetConfig()
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		for _, callback := range c.configModifiedCallback {
			go callback()
		}
	}
}

func (c *ClusterConfig) configUpdated(_, _ interface{}) {
	go c.GetConfig()
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		for _, callback := range c.configModifiedCallback {
			go callback()
		}
	}
}

func isDataVolumeCrd(crd *extv1.CustomResourceDefinition) bool {
	return crd.Spec.Names.Kind == "DataVolume"
}

func isDataSourceCrd(crd *extv1.CustomResourceDefinition) bool {
	return crd.Spec.Names.Kind == "DataSource"
}

func (c *ClusterConfig) crdAddedDeleted(obj interface{}) {
	go c.GetConfig()
	crd := obj.(*extv1.CustomResourceDefinition)
	if !isDataVolumeCrd(crd) && !isDataSourceCrd(crd) {
		return
	}

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.configModifiedCallback != nil {
		for _, callback := range c.configModifiedCallback {
			go callback()

		}
	}
}

func (c *ClusterConfig) crdUpdated(_, cur interface{}) {
	c.crdAddedDeleted(cur)
}

func defaultClusterConfig(cpuArch string) *v1.KubeVirtConfiguration {
	parallelOutboundMigrationsPerNodeDefault := ParallelOutboundMigrationsPerNodeDefault
	parallelMigrationsPerClusterDefault := ParallelMigrationsPerClusterDefault
	bandwithPerMigrationDefault := resource.MustParse(BandwithPerMigrationDefault)
	nodeDrainTaintDefaultKey := NodeDrainTaintDefaultKey
	allowAutoConverge := MigrationAllowAutoConverge
	allowPostCopy := MigrationAllowPostCopy
	defaultUnsafeMigrationOverride := DefaultUnsafeMigrationOverride
	progressTimeout := MigrationProgressTimeout
	completionTimeoutPerGiB := MigrationCompletionTimeoutPerGiB
	cpuRequestDefault := resource.MustParse(DefaultCPURequest)
	nodeSelectorsDefault, _ := parseNodeSelectors(DefaultNodeSelectors)
	defaultNetworkInterface := DefaultNetworkInterface
	defaultMemBalloonStatsPeriod := DefaultMemBalloonStatsPeriod
	SmbiosDefaultConfig := &v1.SMBiosConfiguration{
		Family:       SmbiosConfigDefaultFamily,
		Manufacturer: SmbiosConfigDefaultManufacturer,
		Product:      SmbiosConfigDefaultProduct,
	}
	supportedQEMUGuestAgentVersions := strings.Split(strings.TrimRight(SupportedGuestAgentVersions, ","), ",")
	DefaultOVMFPath, DefaultMachineType, emulatedMachinesDefault := getCPUArchSpecificDefault(cpuArch)
	defaultDiskVerification := &v1.DiskVerification{
		MemoryLimit: resource.NewScaledQuantity(DefaultDiskVerificationMemoryLimitMBytes, resource.Mega),
	}

	return &v1.KubeVirtConfiguration{
		ImagePullPolicy: DefaultImagePullPolicy,
		DeveloperConfiguration: &v1.DeveloperConfiguration{
			UseEmulation:           DefaultAllowEmulation,
			MemoryOvercommit:       DefaultMemoryOvercommit,
			LessPVCSpaceToleration: DefaultLessPVCSpaceToleration,
			MinimumReservePVCBytes: DefaultMinimumReservePVCBytes,
			NodeSelectors:          nodeSelectorsDefault,
			CPUAllocationRatio:     DefaultCPUAllocationRatio,
			DiskVerification:       defaultDiskVerification,
			LogVerbosity: &v1.LogVerbosity{
				VirtAPI:        DefaultVirtAPILogVerbosity,
				VirtOperator:   DefaultVirtOperatorLogVerbosity,
				VirtController: DefaultVirtControllerLogVerbosity,
				VirtHandler:    DefaultVirtHandlerLogVerbosity,
				VirtLauncher:   DefaultVirtLauncherLogVerbosity,
			},
		},
		MigrationConfiguration: &v1.MigrationConfiguration{
			ParallelMigrationsPerCluster:      &parallelMigrationsPerClusterDefault,
			ParallelOutboundMigrationsPerNode: &parallelOutboundMigrationsPerNodeDefault,
			NodeDrainTaintKey:                 &nodeDrainTaintDefaultKey,
			BandwidthPerMigration:             &bandwithPerMigrationDefault,
			ProgressTimeout:                   &progressTimeout,
			CompletionTimeoutPerGiB:           &completionTimeoutPerGiB,
			UnsafeMigrationOverride:           &defaultUnsafeMigrationOverride,
			AllowAutoConverge:                 &allowAutoConverge,
			AllowPostCopy:                     &allowPostCopy,
		},
		MachineType:      DefaultMachineType,
		CPURequest:       &cpuRequestDefault,
		EmulatedMachines: emulatedMachinesDefault,
		NetworkConfiguration: &v1.NetworkConfiguration{
			NetworkInterface:                  defaultNetworkInterface,
			PermitSlirpInterface:              pointer.BoolPtr(DefaultPermitSlirpInterface),
			PermitBridgeInterfaceOnPodNetwork: pointer.BoolPtr(DefaultPermitBridgeInterfaceOnPodNetwork),
		},
		SMBIOSConfig:                SmbiosDefaultConfig,
		SELinuxLauncherType:         DefaultSELinuxLauncherType,
		SupportedGuestAgentVersions: supportedQEMUGuestAgentVersions,
		OVMFPath:                    DefaultOVMFPath,
		MemBalloonStatsPeriod:       &defaultMemBalloonStatsPeriod,
		APIConfiguration: &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{RateLimiter: &v1.RateLimiter{TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
				QPS:   DefaultVirtAPIQPS,
				Burst: DefaultVirtAPIBurst,
			}}},
		},
		ControllerConfiguration: &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{RateLimiter: &v1.RateLimiter{TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
				QPS:   DefaultVirtControllerQPS,
				Burst: DefaultVirtControllerBurst,
			}}},
		},
		HandlerConfiguration: &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{RateLimiter: &v1.RateLimiter{TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
				QPS:   DefaultVirtHandlerQPS,
				Burst: DefaultVirtHandlerBurst,
			}}},
		},
		WebhookConfiguration: &v1.ReloadableComponentConfiguration{
			RestClient: &v1.RESTClientConfiguration{RateLimiter: &v1.RateLimiter{TokenBucketRateLimiter: &v1.TokenBucketRateLimiter{
				QPS:   DefaultVirtWebhookClientQPS,
				Burst: DefaultVirtWebhookClientBurst,
			}}},
		},
	}
}

type ClusterConfig struct {
	crdInformer                      cache.SharedIndexInformer
	kubeVirtInformer                 cache.SharedIndexInformer
	namespace                        string
	cpuArch                          string
	lock                             *sync.Mutex
	lastValidConfig                  *v1.KubeVirtConfiguration
	defaultConfig                    *v1.KubeVirtConfiguration
	lastInvalidConfigResourceVersion string
	lastValidConfigResourceVersion   string
	configModifiedCallback           []ConfigModifiedFn
}

func (c *ClusterConfig) SetConfigModifiedCallback(cb ConfigModifiedFn) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.configModifiedCallback = append(c.configModifiedCallback, cb)
	for _, callback := range c.configModifiedCallback {
		go callback()
	}
}

func setConfigFromKubeVirt(config *v1.KubeVirtConfiguration, kv *v1.KubeVirt) error {
	kvConfig := &kv.Spec.Configuration
	overrides, err := json.Marshal(kvConfig)
	if err != nil {
		return err
	}

	err = json.Unmarshal(overrides, &config)
	if err != nil {
		return err
	}

	return nil
}

// getCPUArchSpecificDefault get arch specific default config
func getCPUArchSpecificDefault(cpuArch string) (string, string, []string) {
	// get arch specific default config
	switch cpuArch {
	case "arm64":
		emulatedMachinesDefault := strings.Split(DefaultAARCH64EmulatedMachines, ",")
		return DefaultAARCH64OVMFPath, DefaultAARCH64MachineType, emulatedMachinesDefault
	case "ppc64le":
		emulatedMachinesDefault := strings.Split(DefaultPPC64LEEmulatedMachines, ",")
		return DefaultARCHOVMFPath, DefaultPPC64LEMachineType, emulatedMachinesDefault
	default:
		emulatedMachinesDefault := strings.Split(DefaultAMD64EmulatedMachines, ",")
		return DefaultARCHOVMFPath, DefaultAMD64MachineType, emulatedMachinesDefault
	}
}

// getConfig returns the latest valid parsed config map result, or updates it
// if a newer version is available.
// XXX Rework this, to happen mostly in informer callbacks.
// This will also allow us then to react to config changes and e.g. restart some controllers
func (c *ClusterConfig) GetConfig() (config *v1.KubeVirtConfiguration) {
	c.lock.Lock()
	defer c.lock.Unlock()

	kv := c.GetConfigFromKubeVirtCR()
	if kv == nil {
		return c.lastValidConfig
	}

	resourceVersion := kv.ResourceVersion

	// if there is a configuration config map present we should use its configuration
	// and ignore configuration in kubevirt
	if c.lastValidConfigResourceVersion == resourceVersion ||
		c.lastInvalidConfigResourceVersion == resourceVersion {
		return c.lastValidConfig
	}

	config = defaultClusterConfig(c.cpuArch)
	err := setConfigFromKubeVirt(config, kv)
	if err != nil {
		c.lastInvalidConfigResourceVersion = resourceVersion
		log.DefaultLogger().Reason(err).Errorf("Invalid cluster config using KubeVirt resource version '%s', falling back to last good resource version '%s'", resourceVersion, c.lastValidConfigResourceVersion)
		return c.lastValidConfig
	}

	log.DefaultLogger().Infof("Updating cluster config from KubeVirt to resource version '%s'", resourceVersion)
	c.lastValidConfigResourceVersion = resourceVersion
	c.lastValidConfig = config
	return c.lastValidConfig
}

func (c *ClusterConfig) GetConfigFromKubeVirtCR() *v1.KubeVirt {
	objects := c.kubeVirtInformer.GetStore().List()
	var kubeVirtName string
	for _, obj := range objects {
		if kv, ok := obj.(*v1.KubeVirt); ok && kv.DeletionTimestamp == nil {
			if kv.Status.Phase != "" {
				kubeVirtName = kv.Name
			}
		}
	}

	if kubeVirtName == "" {
		return nil
	}

	if obj, exists, err := c.kubeVirtInformer.GetStore().GetByKey(c.namespace + "/" + kubeVirtName); err != nil {
		log.DefaultLogger().Reason(err).Errorf("Error loading the cluster config from KubeVirt cache, falling back to last good resource version '%s'", c.lastValidConfigResourceVersion)
		return nil
	} else if !exists {
		// this path should not be possible
		return nil
	} else {
		return obj.(*v1.KubeVirt)
	}
}

func (c *ClusterConfig) HasDataSourceAPI() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	objects := c.crdInformer.GetStore().List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
			if isDataSourceCrd(crd) {
				return true
			}
		}
	}
	return false
}

func (c *ClusterConfig) HasDataVolumeAPI() bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	objects := c.crdInformer.GetStore().List()
	for _, obj := range objects {
		if crd, ok := obj.(*extv1.CustomResourceDefinition); ok && crd.DeletionTimestamp == nil {
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
