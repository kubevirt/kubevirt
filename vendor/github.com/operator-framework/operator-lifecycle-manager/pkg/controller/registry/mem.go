package registry

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/equality"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/lib/operatorclient"
)

// InMem - catalog source implementation that stores the data in memory in golang maps
var _ Source = &InMem{}

type InMem struct {
	// map ClusterServiceVersion name to their resource definition
	clusterservices map[string]v1alpha1.ClusterServiceVersion

	// map ClusterServiceVersions by name to metadata for the CSV that replaces it
	replaces map[string][]CSVMetadata

	// map CRD to their full definition
	crds map[CRDKey]v1beta1.CustomResourceDefinition

	// map CRD to the names of the CSVs that own them
	crdOwners map[CRDKey][]string

	// map package name to their full manifest
	packages map[string]PackageManifest

	// map from CSV name to the package channel(s) that contain it.
	csvPackageChannels map[string][]packageAndChannel
}

type packageAndChannel struct {
	packageRef PackageManifest
	channelRef PackageChannel
}

func NewInMemoryFromDirectory(directory string) (*InMem, error) {
	log.Infof("loading catalog from directory: %s", directory)
	loader := DirectoryCatalogResourceLoader{NewInMem()}
	if err := loader.LoadCatalogResources(directory); err != nil {
		return nil, err
	}
	return loader.Catalog, nil
}

func NewInMemoryFromConfigMap(cmClient operatorclient.ClientInterface, namespace, cmName string) (*InMem, error) {
	log.Infof("loading catalog from a configmap: %s", cmName)
	loader := ConfigMapCatalogResourceLoader{namespace, cmClient}
	catalog := NewInMem()
	if err := loader.LoadCatalogResources(catalog, cmName); err != nil {
		return nil, err
	}
	return catalog, nil
}

// NewInMem returns a ptr to a new InMem instance
func NewInMem() *InMem {
	return &InMem{
		clusterservices:    map[string]v1alpha1.ClusterServiceVersion{},
		replaces:           map[string][]CSVMetadata{},
		crds:               map[CRDKey]v1beta1.CustomResourceDefinition{},
		crdOwners:          map[CRDKey][]string{},
		packages:           map[string]PackageManifest{},
		csvPackageChannels: map[string][]packageAndChannel{},
	}
}

// SetCRDDefinition sets the full resource definition of a CRD in the stored map
// only sets a new definition if one is not already set
func (m *InMem) SetCRDDefinition(crd v1beta1.CustomResourceDefinition) error {
	key := CRDKey{
		Kind:    crd.Spec.Names.Kind,
		Name:    crd.GetName(),
		Version: crd.Spec.Version,
	}
	if old, exists := m.crds[key]; exists && !equality.Semantic.DeepEqual(crd, old) {
		return fmt.Errorf("invalid CRD : definition for CRD %s already set", crd.GetName())
	}
	m.crds[key] = crd
	return nil
}

// FindReplacementCSVForPackageNameUnderChannel returns the CSV that replaces the CSV with the
// matching CSV name, within the package and channel specified.
func (m *InMem) FindReplacementCSVForPackageNameUnderChannel(packageName string, channelName string, csvName string) (*v1alpha1.ClusterServiceVersion, error) {
	latestCSV, err := m.FindCSVForPackageNameUnderChannel(packageName, channelName)
	if err != nil {
		return nil, err
	}

	if latestCSV.GetName() == csvName {
		return nil, fmt.Errorf("Channel is already up-to-date")
	}

	// Walk backwards over the `replaces` field until we find the CSV with the specified name.
	var currentCSV = latestCSV
	var nextCSV *v1alpha1.ClusterServiceVersion = nil
	for currentCSV != nil {
		if currentCSV.GetName() == csvName {
			return nextCSV, nil
		}

		nextCSV = currentCSV
		replacesName := currentCSV.Spec.Replaces
		currentCSV = nil

		if replacesName != "" {
			replacesCSV, err := m.FindCSVByName(replacesName)
			if err != nil {
				return nil, err
			}

			currentCSV = replacesCSV
		}
	}

	return nil, fmt.Errorf("Could not find matching replacement for CSV `%s` in package `%s` for channel `%s`", csvName, packageName, channelName)
}

// FindCSVForPackageNameUnderChannel finds the CSV referenced by the specified channel under the
// package with the specified name.
func (m *InMem) FindCSVForPackageNameUnderChannel(packageName string, channelName string) (*v1alpha1.ClusterServiceVersion, error) {
	packageManifest, ok := m.packages[packageName]
	if !ok {
		return nil, fmt.Errorf("Unknown package %s", packageName)
	}

	for _, channel := range packageManifest.Channels {
		if channel.Name == channelName {
			return m.FindCSVByName(channel.CurrentCSVName)
		}
	}

	return nil, fmt.Errorf("Unknown channel %s in package %s", channelName, packageName)
}

// addPackageManifest adds a new package manifest to the in memory catalog.
func (m *InMem) AddPackageManifest(pkg PackageManifest) error {
	if len(pkg.PackageName) == 0 {
		return fmt.Errorf("Empty package name")
	}

	// Make sure that each channel name is unique and that the referenced CSV exists.
	channelMap := make(map[string]bool, len(pkg.Channels))
	for _, channel := range pkg.Channels {
		if _, exists := channelMap[channel.Name]; exists {
			return fmt.Errorf("Channel %s declared twice in package manifest", channel.Name)
		}

		channelMap[channel.Name] = true

		currentCSV, err := m.FindCSVByName(channel.CurrentCSVName)
		if err != nil {
			return fmt.Errorf("Missing CSV with name %s", channel.CurrentCSVName)
		}

		// For each of the CSVs in the full replacement chain, add an entry to the package map.
		csvChain, err := m.fullCSVReplacesHistory(currentCSV)
		if err != nil {
			return err
		}

		for _, csv := range csvChain {
			if _, ok := m.csvPackageChannels[csv.GetName()]; !ok {
				m.csvPackageChannels[csv.GetName()] = []packageAndChannel{}
			}

			m.csvPackageChannels[csv.GetName()] = append(m.csvPackageChannels[csv.GetName()], packageAndChannel{
				packageRef: pkg,
				channelRef: channel,
			})
		}
	}

	// Make sure the default channel name matches a real channel, if given.
	if pkg.DefaultChannelName != "" {
		if _, exists := channelMap[pkg.DefaultChannelName]; !exists {
			return fmt.Errorf("Invalid default channel %s", pkg.DefaultChannelName)
		}
	}

	m.packages[pkg.PackageName] = pkg
	return nil
}

// fullCSVHistory returns the full set of CSVs in the `replaces` history, starting at the given CSV.
func (m *InMem) fullCSVReplacesHistory(csv *v1alpha1.ClusterServiceVersion) ([]v1alpha1.ClusterServiceVersion, error) {
	if csv.Spec.Replaces == "" {
		return []v1alpha1.ClusterServiceVersion{*csv}, nil
	}

	replaced, err := m.FindCSVByName(csv.Spec.Replaces)
	if err != nil {
		return []v1alpha1.ClusterServiceVersion{}, err
	}

	replacedChain, err := m.fullCSVReplacesHistory(replaced)
	if err != nil {
		return []v1alpha1.ClusterServiceVersion{}, err
	}

	return append(replacedChain, *csv), nil
}

// setOrReplaceCRDDefinition overwrites any existing definition with the same name
func (m *InMem) setOrReplaceCRDDefinition(crd v1beta1.CustomResourceDefinition) {
	m.crds[CRDKey{
		Kind:    crd.Spec.Names.Kind,
		Name:    crd.GetName(),
		Version: crd.Spec.Version,
	}] = crd
}

// findServiceConflicts collates a list of errors from conflicting catalog entries
func (m *InMem) findServiceConflicts(csv v1alpha1.ClusterServiceVersion) []error {
	errs := []error{}

	// validate owned CRDs
	for _, crdReq := range csv.Spec.CustomResourceDefinitions.Owned {
		key := CRDKey{
			Kind:    crdReq.Kind,
			Name:    crdReq.Name,
			Version: crdReq.Version,
		}
		// validate crds have definitions stored
		if _, ok := m.crds[key]; !ok {
			errs = append(errs, fmt.Errorf("missing definition for owned CRD %v", key))
		}
	}
	return errs
}

// addService is a helper fn to register a new service into the catalog
// will error if `safe` is true and conflicts are found
func (m *InMem) addService(csv v1alpha1.ClusterServiceVersion, safe bool) error {
	name := csv.GetName()

	// find and log any conflicts; return with error if in `safe` mode
	if conflicts := m.findServiceConflicts(csv); len(conflicts) > 0 {
		log.Debugf("found conflicts for CSV %s: %v", name, conflicts)
		if safe {
			return fmt.Errorf("cannot add CSV %s safely: %v", name, conflicts)
		}
	}

	// add service
	m.clusterservices[name] = csv

	// register it as replacing CSV from its spec, if any
	if csv.Spec.Replaces != "" {
		if _, ok := m.replaces[csv.Spec.Replaces]; !ok {
			m.replaces[csv.Spec.Replaces] = []CSVMetadata{}
		}

		m.replaces[csv.Spec.Replaces] = append(m.replaces[csv.Spec.Replaces], CSVMetadata{
			Name:    name,
			Version: csv.Spec.Version.String(),
		})
	}

	// register its crds
	for _, crd := range csv.Spec.CustomResourceDefinitions.Owned {
		key := CRDKey{
			Name:    crd.Name,
			Version: crd.Version,
			Kind:    crd.Kind,
		}

		if m.crdOwners[key] == nil {
			m.crdOwners[key] = []string{}
		}

		m.crdOwners[key] = append(m.crdOwners[key], name)
	}
	return nil
}

// setCSVDefinition registers a new service into the catalog
// will return error if any conflicts exist
func (m *InMem) setCSVDefinition(csv v1alpha1.ClusterServiceVersion) error {
	return m.addService(csv, true)
}

// AddOrReplaceService registers service into the catalog, overwriting any existing values
func (m *InMem) AddOrReplaceService(csv v1alpha1.ClusterServiceVersion) {
	_ = m.addService(csv, false)
}

// removeService is a helper fn to delete a service from the catalog
func (m *InMem) removeService(name string) error {
	csv, exists := m.clusterservices[name]
	if !exists {
		return fmt.Errorf("not found: ClusterServiceVersion %s", name)
	}

	delete(m.clusterservices, name)
	if csv.Spec.Replaces != "" {
		delete(m.replaces, csv.Spec.Replaces)
	}

	return nil
}

// FindCSVByName looks up the CSV with the given name.
func (m *InMem) FindCSVByName(name string) (*v1alpha1.ClusterServiceVersion, error) {
	csv, exists := m.clusterservices[name]
	if !exists {
		return nil, fmt.Errorf("not found: ClusterServiceVersion %s", name)
	}

	return &csv, nil
}

// FindReplacementCSVForName looks up any CSV in the catalog that replaces the given CSV, if any.
func (m *InMem) FindReplacementCSVForName(name string) (*v1alpha1.ClusterServiceVersion, error) {
	csvMetadata, ok := m.replaces[name]
	if !ok {
		return nil, fmt.Errorf("not found: ClusterServiceVersion that replaces %s", name)
	}

	if len(csvMetadata) < 1 {
		return nil, fmt.Errorf("not found: ClusterServiceVersion that replaces %s", name)
	}

	return m.FindCSVByName(csvMetadata[0].Name)
}

// AllPackages returns all package manifests in the catalog
func (m *InMem) AllPackages() map[string]PackageManifest {
	return m.packages
}

// ListServices lists all versions of the service in the catalog
func (m *InMem) ListServices() ([]v1alpha1.ClusterServiceVersion, error) {
	services := []v1alpha1.ClusterServiceVersion{}
	for _, csv := range m.clusterservices {
		services = append(services, csv)
	}
	return services, nil
}

// ListLatestCSVsForCRD lists the latests versions of the service that manages the given CRD.
func (m *InMem) ListLatestCSVsForCRD(key CRDKey) ([]CSVAndChannelInfo, error) {
	// Find the names of the CSVs that own the CRD.
	ownerCSVNames, ok := m.crdOwners[key]
	if !ok {
		return nil, fmt.Errorf("not found: CRD %s", key)
	}

	// For each of the CSVs found, lookup the package channels that create that CSV somewhere along
	// the way. For each, we then filter to the latest CSV that creates the CRD, and return that.
	// This allows the caller to find the *latest* version of each channel, that will successfully
	// instantiate the require CRD.
	channelInfo := make([]CSVAndChannelInfo, 0, len(ownerCSVNames))
	added := map[string]bool{}

	for _, ownerCSVName := range ownerCSVNames {
		packageChannels, ok := m.csvPackageChannels[ownerCSVName]
		if !ok {
			// Note: legacy handling. To be removed once all CSVs are part of packages.
			latestCSV, err := m.findLatestCSVThatOwns(ownerCSVName, key)
			if err != nil {
				return nil, err
			}

			channelInfo = append(channelInfo, CSVAndChannelInfo{
				CSV:              latestCSV,
				Channel:          PackageChannel{},
				IsDefaultChannel: false,
			})
			continue
		}

		for _, packageChannel := range packageChannels {
			// Find the latest CSV in the channel that owns the CRD.
			latestCSV, err := m.findLatestCSVThatOwns(packageChannel.channelRef.CurrentCSVName, key)
			if err != nil {
				return nil, err
			}

			key := fmt.Sprintf("%s::%s", latestCSV.GetName(), packageChannel.channelRef.Name)
			if _, ok := added[key]; ok {
				continue
			}

			channelInfo = append(channelInfo, CSVAndChannelInfo{
				CSV:              latestCSV,
				Channel:          packageChannel.channelRef,
				IsDefaultChannel: packageChannel.channelRef.IsDefaultChannel(packageChannel.packageRef),
			})
			added[key] = true
		}
	}

	return channelInfo, nil
}

// findLatestCSVThatOwns returns the latest CSV in the chain of CSVs, starting at the CSV with the
// specified name, that owns the referenced CRD. For example, if given CSV `foobar-v1.2.0` in the
// a chain of foobar-v1.2.0 --(replaces)--> foobar-v1.1.0 --(replaces)--> foobar-v1.0.0 and
// `foobar-v1.1.0` is the latest that owns the CRD, it will be returned.
func (m *InMem) findLatestCSVThatOwns(csvName string, key CRDKey) (*v1alpha1.ClusterServiceVersion, error) {
	csv, err := m.FindCSVByName(csvName)
	if err != nil {
		return nil, err
	}

	// Check if the CSV owns the CRD.
	for _, crdReq := range csv.Spec.CustomResourceDefinitions.Owned {
		if crdReq.Name == key.Name {
			return csv, nil
		}
	}

	// Otherwise, check the CSV this CSV replaces.
	if csv.Spec.Replaces == "" {
		return nil, fmt.Errorf("Could not find owner for CRD %s", key.Name)
	}

	return m.findLatestCSVThatOwns(csv.Spec.Replaces, key)
}

// FindCRDByName looks up the full CustomResourceDefinition for the resource with the given name
func (m *InMem) FindCRDByKey(key CRDKey) (*v1beta1.CustomResourceDefinition, error) {
	crd, ok := m.crds[key]
	if !ok {
		return nil, fmt.Errorf("not found: CRD %s", key)
	}
	return &crd, nil
}
