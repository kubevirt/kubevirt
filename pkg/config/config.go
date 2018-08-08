package config

import (
	"fmt"
	"strings"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"

	v12 "kubevirt.io/kubevirt/pkg/api/v1"
)

const configName = "kube-system/kubevirt-config"
const LabelNetworkPrefix = "network.kubevirt.io/"

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
	config, err := c.getConfig()
	if err != nil {
		return false, err
	}
	return config.IsUseEmulation(), nil
}

func (c *ClusterConfig) GetLabelNetworksForNode(node *v1.Node) (map[string]*v12.LabelNetworkDefinition, error) {
	config, err := c.getConfig()
	if err != nil {
		return nil, err
	}
	res := map[string]*v12.LabelNetworkDefinition{}

	for n, networks := range config.Networks {
		if networks.LabelNetwork != nil {
			if match, def := matchNode(node, networks.LabelNetwork.Definitions); match {
				res[LabelNetworkPrefix+n] = def
			}
		}
	}
	return res, nil
}

func (c *ClusterConfig) GetNotMatchingLabelNetworksOnNode(node *v1.Node) ([]string, error) {
	matching, err := c.GetLabelNetworksForNode(node)
	if err != nil {
		return nil, err
	}

	mismatch := []string{}

	for k, _ := range node.Labels {
		if strings.HasPrefix(k, LabelNetworkPrefix) {
			if _, ok := matching[k]; !ok {
				mismatch = append(mismatch, k)
			}
		}
	}
	return mismatch, nil
}

func (c *ClusterConfig) GetMissingLabelNetworksOnNode(node *v1.Node) ([]string, error) {
	matching, err := c.GetLabelNetworksForNode(node)
	if err != nil {
		return nil, err
	}

	missing := []string{}

	for k, _ := range matching {
		if strings.HasPrefix(k, LabelNetworkPrefix) {
			if _, ok := node.Labels[k]; !ok {
				missing = append(missing, k)
			}
		}
	}
	return missing, nil
}

func (c *ClusterConfig) getConfig() (*v12.KubeVirtConfig, error) {
	obj, exists, err := c.store.GetByKey(configName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return &v12.KubeVirtConfig{}, nil
	}
	return obj.(*v12.KubeVirtConfig), nil
}

func matchNode(node *v1.Node, definitions []v12.LabelNetworkDefinition) (bool, *v12.LabelNetworkDefinition) {
	for _, def := range definitions {
		if labels.SelectorFromSet(labels.Set(def.NodeSelector)).Matches(labels.Set(node.Labels)) {
			return true, &def
		}
	}
	return false, nil
}

func (c *ClusterConfig) GetImagePullPolicy() (policy v1.PullPolicy, err error) {
	policy = v1.PullIfNotPresent // Default if not specified
	config, err := c.getConfig()
	if err != nil {
		return
	}
	if config.Developer == nil || config.Developer.ImagePullPolicy == "" {
		return
	}
	value := config.Developer.ImagePullPolicy
	switch value {
	case "Always":
		policy = v1.PullAlways
	case "Never":
		policy = v1.PullNever
	case "IfNotPresent":
		policy = v1.PullIfNotPresent
	default:
		err = fmt.Errorf("Invalid ImagePullPolicy in ConfigMap: %s", value)
	}
	return
}
