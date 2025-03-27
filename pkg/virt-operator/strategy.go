package virt_operator

import (
	"context"
	"fmt"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/log"

	install "kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/install"
	operatorutil "kubevirt.io/kubevirt/pkg/virt-operator/util"
)

func (c *KubeVirtController) getCachedInstallStrategy(config *operatorutil.KubeVirtDeploymentConfig, generation int64) (*install.Strategy, bool) {
	cachedValue := c.latestStrategy.Load()
	if cachedValue == nil {
		return nil, false
	}
	cachedEntry, ok := cachedValue.(strategyCacheEntry)
	if !ok {
		return nil, ok
	}

	if cachedEntry.key == fmt.Sprintf(installStrategyKeyTemplate, config.GetDeploymentID(), generation) {
		return cachedEntry.value, true
	}
	return nil, false
}

func (c *KubeVirtController) cacheInstallStrategy(cachedEntry *install.Strategy, config *operatorutil.KubeVirtDeploymentConfig, generation int64) {
	c.latestStrategy.Store(strategyCacheEntry{key: fmt.Sprintf(installStrategyKeyTemplate, config.GetDeploymentID(), generation), value: cachedEntry})
}

func (c *KubeVirtController) deleteAllInstallStrategy() error {
	for _, obj := range c.stores.InstallStrategyConfigMapCache.List() {
		configMap, ok := obj.(*k8sv1.ConfigMap)
		if ok && configMap.DeletionTimestamp == nil {
			err := c.clientset.CoreV1().ConfigMaps(configMap.Namespace).Delete(context.Background(), configMap.Name, metav1.DeleteOptions{})
			if err != nil {
				log.Log.Errorf("Failed to delete configmap %+v: %v", configMap, err)
				return err
			}
		}
	}

	// reset the cached strategy
	c.latestStrategy.Store(strategyCacheEntry{})
	return nil
}
