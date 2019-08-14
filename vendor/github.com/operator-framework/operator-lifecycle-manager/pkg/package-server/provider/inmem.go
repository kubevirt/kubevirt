package provider

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ghodss/yaml"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	operatorsv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/queueinformer"
	packagev1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/package-server/apis/packagemanifest/v1alpha1"
)

const (
	// ConfigMapPackageName is the key for package ConfigMap data
	ConfigMapPackageName = "packages"

	// ConfigMapCSVName is the key for CSV ConfigMap data
	ConfigMapCSVName = "clusterServiceVersions"
)

type packageKey struct {
	catalogSourceName      string
	catalogSourceNamespace string
	packageName            string
}

// InMemoryProvider syncs and provides PackageManifests from the cluster in an in-memory cache
type InMemoryProvider struct {
	*queueinformer.Operator

	mu        sync.RWMutex
	manifests map[packageKey]packagev1alpha1.PackageManifest
}

// NewInMemoryProvider returns a pointer to a new InMemoryProvider instance
func NewInMemoryProvider(informers []cache.SharedIndexInformer, queueOperator *queueinformer.Operator) *InMemoryProvider {
	// instantiate the in-mem provider
	prov := &InMemoryProvider{
		Operator:  queueOperator,
		manifests: make(map[packageKey]packagev1alpha1.PackageManifest),
	}

	// register CatalogSource informers.
	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "catalogsources")
	queueInformers := queueinformer.New(
		queue,
		informers,
		prov.syncCatalogSource,
		nil,
		"catsrc",
	)
	for _, informer := range queueInformers {
		prov.RegisterQueueInformer(informer)
	}

	return prov
}

// parsePackageManifestsFromConfigMap returns a list of PackageManifests from a given ConfigMap
func parsePackageManifestsFromConfigMap(cm *corev1.ConfigMap, catalogSourceName, catalogSourceNamespace string) ([]packagev1alpha1.PackageManifest, error) {
	cmName := cm.GetName()
	logger := log.WithFields(log.Fields{
		"Action": "Load ConfigMap",
		"name":   cmName,
	})

	found := false
	csvs := make(map[string]operatorsv1alpha1.ClusterServiceVersion)
	csvListYaml, ok := cm.Data[ConfigMapCSVName]
	if ok {
		logger.Debug("ConfigMap contains CSVsf")
		csvListJSON, err := yaml.YAMLToJSON([]byte(csvListYaml))
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", cmName, err)
			return nil, fmt.Errorf("error loading CSV list yaml from ConfigMap %s: %s", cmName, err)
		}

		var parsedCSVList []operatorsv1alpha1.ClusterServiceVersion
		err = json.Unmarshal([]byte(csvListJSON), &parsedCSVList)
		if err != nil {
			log.Debugf("Load ConfigMap     -- ERROR %s : error=%s", cmName, err)
			return nil, fmt.Errorf("error parsing CSV list (json) from ConfigMap %s: %s", cmName, err)
		}

		for _, csv := range parsedCSVList {
			found = true

			// TODO: add check for invalid CSV definitions
			log.Debugf("found csv %s", csv.GetName())
			csvs[csv.GetName()] = csv
		}
	}

	manifests := []packagev1alpha1.PackageManifest{}
	packageListYaml, ok := cm.Data[ConfigMapPackageName]
	if ok {
		logger.Debug("ConfigMap contains packages")
		packageListJSON, err := yaml.YAMLToJSON([]byte(packageListYaml))
		if err != nil {
			logger.Debugf("ERROR: %s", err)
			return nil, fmt.Errorf("error loading package list yaml from ConfigMap %s: %s", cmName, err)
		}

		var parsedStatuses []packagev1alpha1.PackageManifestStatus
		err = json.Unmarshal([]byte(packageListJSON), &parsedStatuses)
		if err != nil {
			logger.Debugf("ERROR: %s", err)
			return nil, fmt.Errorf("error parsing package list (json) from ConfigMap %s: %s", cmName, err)
		}

		for _, status := range parsedStatuses {
			found = true

			// add the name and namespace of the CatalogSource
			manifest := packagev1alpha1.PackageManifest{
				ObjectMeta: metav1.ObjectMeta{
					Name:      status.PackageName,
					Namespace: cm.GetNamespace(),
					Labels:    map[string]string{},
				},
				Status: status,
			}

			manifest.Status.CatalogSourceName = catalogSourceName
			manifest.Status.CatalogSourceNamespace = catalogSourceNamespace

			// add all PackageChannel CSVDescriptions
			for i, channel := range manifest.Status.Channels {
				csv, ok := csvs[channel.CurrentCSVName]
				if !ok {
					return nil, fmt.Errorf("packagemanifest %s references non-existent csv %s", manifest.Status.PackageName, channel.CurrentCSVName)
				}

				manifest.Status.Channels[i].CurrentCSVDesc = packagev1alpha1.CreateCSVDescription(&csv)

				// set the Provider
				if manifest.Status.DefaultChannelName != "" && csv.GetName() == manifest.Status.DefaultChannelName || i == 0 {
					manifest.Status.Provider = packagev1alpha1.AppLink{
						Name: csv.Spec.Provider.Name,
						URL:  csv.Spec.Provider.URL,
					}

					// add Provider as a label
					manifest.ObjectMeta.Labels["provider"] = manifest.Status.Provider.Name
					manifest.ObjectMeta.Labels["provider-url"] = manifest.Status.Provider.URL
				}
			}

			// set CatalogSource labels
			manifest.ObjectMeta.Labels["catalog"] = manifest.Status.CatalogSourceName
			manifest.ObjectMeta.Labels["catalog-namespace"] = manifest.Status.CatalogSourceNamespace

			log.Debugf("retrieved packagemanifest %s", manifest.GetName())
			manifests = append(manifests, manifest)
		}
	}

	if !found {
		logger.Debug("ERROR: No valid resource found")
		return nil, fmt.Errorf("error parsing ConfigMap %s: no valid resources found", cmName)
	}

	return manifests, nil
}

func (m *InMemoryProvider) syncCatalogSource(obj interface{}) error {
	// assert as CatalogSource
	catsrc, ok := obj.(*operatorsv1alpha1.CatalogSource)
	if !ok {
		log.Debugf("wrong type: %#v", obj)
		return fmt.Errorf("casting catalog source failed")
	}

	var manifests []packagev1alpha1.PackageManifest

	// handle by sourceType
	switch catsrc.Spec.SourceType {
	case "internal":
		// get the CatalogSource's ConfigMap
		cm, err := m.OpClient.KubernetesInterface().CoreV1().ConfigMaps(catsrc.GetNamespace()).Get(catsrc.Spec.ConfigMap, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get catalog config map %s when updating status: %s", catsrc.Spec.ConfigMap, err)
		}

		// parse PackageManifest from ConfigMap
		manifests, err = parsePackageManifestsFromConfigMap(cm, catsrc.GetName(), catsrc.GetNamespace())
		if err != nil {
			return fmt.Errorf("failed to load package manifest from config map %s", cm.GetName())
		}

	default:
		return fmt.Errorf("catalog source %s in namespace %s source type %s not recognized", catsrc.GetName(), catsrc.GetNamespace(), catsrc.Spec.SourceType)
	}

	// update manifests
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, manifest := range manifests {
		key := packageKey{
			catalogSourceName:      manifest.Status.CatalogSourceName,
			catalogSourceNamespace: manifest.Status.CatalogSourceNamespace,
			packageName:            manifest.GetName(),
		}

		if pm, ok := m.manifests[key]; ok {
			// use existing CreationTimestamp
			manifest.CreationTimestamp = pm.ObjectMeta.CreationTimestamp
		} else {
			// set CreationTimestamp if first time seeing the PackageManifest
			manifest.CreationTimestamp = metav1.NewTime(time.Now())
		}

		log.Debugf("storing packagemanifest at %+v", key)
		m.manifests[key] = manifest
	}

	return nil
}

// ListPackageManifests implements PackageManifestProvider.ListPackageManifests()
func (m *InMemoryProvider) ListPackageManifests(namespace string) (*packagev1alpha1.PackageManifestList, error) {
	manifestList := &packagev1alpha1.PackageManifestList{}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.manifests) > 0 {
		var matching []packagev1alpha1.PackageManifest
		for _, manifest := range m.manifests {
			if manifest.GetNamespace() == namespace {
				// tack on the csv spec for each channel
				matching = append(matching, manifest)
			}
		}

		manifestList.Items = matching
	}

	return manifestList, nil
}

// GetPackageManifest implements PackageManifestProvider.GetPackageManifest(...)
func (m *InMemoryProvider) GetPackageManifest(namespace, name string) (*packagev1alpha1.PackageManifest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var manifest packagev1alpha1.PackageManifest
	for key, pm := range m.manifests {
		if key.packageName == name && key.catalogSourceNamespace == namespace {
			manifest = pm
		}
	}

	return &manifest, nil
}
