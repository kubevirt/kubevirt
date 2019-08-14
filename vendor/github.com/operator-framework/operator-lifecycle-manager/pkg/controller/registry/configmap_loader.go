package registry

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	"k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

const (
	ConfigMapCRDName     = "customResourceDefinitions"
	ConfigMapCSVName     = "clusterServiceVersions"
	ConfigMapPackageName = "packages"
)

// ConfigMapCatalogResourceLoader loads a ConfigMap of resources into the in-memory catalog
type ConfigMapCatalogResourceLoader struct {
	namespace string
	opClient  operatorclient.ClientInterface
}

func NewConfigMapCatalogResourceLoader(namespace string, opClient operatorclient.ClientInterface) ConfigMapCatalogResourceLoader {
	return ConfigMapCatalogResourceLoader{
		namespace: namespace,
		opClient:  opClient,
	}
}

func (d *ConfigMapCatalogResourceLoader) LoadCatalogResources(catalog *InMem, configMapName string) error {
	log.Debugf("Load ConfigMap     -- BEGIN %s", configMapName)

	cm, err := d.opClient.KubernetesInterface().CoreV1().ConfigMaps(d.namespace).Get(configMapName, metav1.GetOptions{})
	if err != nil {
		log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
		return fmt.Errorf("error loading catalog from ConfigMap %s: %s", configMapName, err)
	}
	return d.LoadCatalogResourcesFromConfigMap(catalog, cm)
}

func (d *ConfigMapCatalogResourceLoader) LoadCatalogResourcesFromConfigMap(catalog *InMem, cm *v1.ConfigMap) error {
	configMapName := cm.GetName()
	found := false
	crdListYaml, ok := cm.Data[ConfigMapCRDName]
	if ok {
		crdListJson, err := yaml.YAMLToJSON([]byte(crdListYaml))
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error loading CRD list yaml from ConfigMap %s: %s", configMapName, err)
		}

		var parsedCRDList []v1beta1.CustomResourceDefinition
		err = json.Unmarshal([]byte(crdListJson), &parsedCRDList)
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error parsing CRD list (json) from ConfigMap %s: %s", configMapName, err)
		}

		for _, crd := range parsedCRDList {
			found = true
			catalog.SetCRDDefinition(crd)
		}
	}

	csvListYaml, ok := cm.Data[ConfigMapCSVName]
	if ok {
		csvListJson, err := yaml.YAMLToJSON([]byte(csvListYaml))
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error loading CSV list yaml from ConfigMap %s: %s", configMapName, err)
		}

		var parsedCSVList []v1alpha1.ClusterServiceVersion
		err = json.Unmarshal([]byte(csvListJson), &parsedCSVList)
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error parsing CSV list (json) from ConfigMap %s: %s", configMapName, err)
		}

		for _, csv := range parsedCSVList {
			found = true
			catalog.setCSVDefinition(csv)
		}
	}

	packageListYaml, ok := cm.Data[ConfigMapPackageName]
	if ok {
		log.Debug("Load ConfigMap      -- ConfigMap contains packages")
		packageListJson, err := yaml.YAMLToJSON([]byte(packageListYaml))
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error loading package list yaml from ConfigMap %s: %s", configMapName, err)
		}

		var parsedPackageManifests []PackageManifest
		err = json.Unmarshal([]byte(packageListJson), &parsedPackageManifests)
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", configMapName, err)
			return fmt.Errorf("error parsing package list (json) from ConfigMap %s: %s", configMapName, err)
		}
		for _, packageManifest := range parsedPackageManifests {
			found = true
			if err := catalog.AddPackageManifest(packageManifest); err != nil {
				return err
			}
		}
		log.Debugf("Load ConfigMap      -- Found packages: %v", catalog.packages)
	}

	if !found {
		log.Debugf("Load ConfigMap     -- ERROR %s : no resources found", configMapName)
		return fmt.Errorf("error parsing ConfigMap %s: no valid resources found", configMapName)
	}

	log.Debugf("Load ConfigMap     -- OK    %s", configMapName)
	return nil
}
