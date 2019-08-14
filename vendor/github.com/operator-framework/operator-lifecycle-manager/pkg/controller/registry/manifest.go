package registry

import (
	"fmt"
	"io/ioutil"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"

	"encoding/json"

	"github.com/ghodss/yaml"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// LoadCRDFromFile is a utility function for loading the CRD schemas.
func LoadCRDFromFile(m *InMem, filepath string) (*v1beta1.CustomResourceDefinition, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to load CRD from file %s: %v", filepath, err)
	}
	crd := v1beta1.CustomResourceDefinition{}
	if _, _, err = scheme.Codecs.UniversalDecoder().Decode(data, nil, &crd); err != nil {
		return nil, fmt.Errorf("could not decode contents of file %s into CRD: %v", filepath, err)
	}
	if err = m.SetCRDDefinition(crd); err != nil {
		return nil, fmt.Errorf("unable to set CRD found in catalog: %v", err)
	}
	return &crd, nil
}

// LoadCSVFromFile is a utility function for loading CSV definitions
func LoadCSVFromFile(m *InMem, filepath string) (*v1alpha1.ClusterServiceVersion, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to load CSV from file %s: %v", filepath, err)
	}
	csv := v1alpha1.ClusterServiceVersion{}
	if _, _, err = scheme.Codecs.UniversalDecoder().Decode(data, nil, &csv); err != nil {
		return nil, fmt.Errorf("could not decode contents of file %s into CSV: %v", filepath, err)
	}
	if err = m.setCSVDefinition(csv); err != nil {
		return nil, fmt.Errorf("unable to set CSV found in catalog: %v", err)
	}
	return &csv, nil
}

// LoadPackageFromFile is a utility function for loading Package definitions
func LoadPackageFromFile(m *InMem, filepath string) (*PackageManifest, error) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("unable to load package from file %s: %v", filepath, err)
	}
	pkg := PackageManifest{}

	packageJson, err := yaml.YAMLToJSON(data)

	if err != nil {
		return nil, fmt.Errorf("error loading package yaml: %s", err)
	}

	err = json.Unmarshal([]byte(packageJson), &pkg)

	if err = m.AddPackageManifest(pkg); err != nil {
		return nil, fmt.Errorf("unable to set package found in catalog: %v", err)
	}
	return &pkg, nil
}
