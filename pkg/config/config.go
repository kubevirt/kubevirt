package config

import (
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

const configMapName = "kube-system/kubevirt-config"
const useEmulationKey = "debug.useEmulation"

func NewClusterConfig(configMapInformer cache.Store) *ClusterConfig {
	c := &ClusterConfig{
		store: configMapInformer,
	}
	return c
}

type ClusterConfig struct {
	store cache.Store
}

func (c *ClusterConfig) IsUseEmulation() (bool, error) {
	useEmulation := false
	obj, exists, err := c.store.GetByKey(configMapName)
	if err != nil {
		return false, err
	}
	if !exists {
		return useEmulation, nil
	}
	cm := obj.(*v1.ConfigMap)
	emu, ok := cm.Data[useEmulationKey]
	if ok {
		useEmulation = (strings.ToLower(emu) == "true")
	}
	return useEmulation, nil
}
