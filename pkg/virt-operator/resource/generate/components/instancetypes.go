package components

import (
	"bytes"
	_ "embed"
	"io"

	"k8s.io/apimachinery/pkg/util/yaml"

	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
)

//go:embed data/common-clusterinstancetypes-bundle.yaml
var clusterInstancetypesBundle []byte

//go:embed data/common-clusterpreferences-bundle.yaml
var clusterPreferencesBundle []byte

func NewClusterInstancetypes() ([]*instancetypev1beta1.VirtualMachineClusterInstancetype, error) {
	return decodeResources[instancetypev1beta1.VirtualMachineClusterInstancetype](clusterInstancetypesBundle)
}

func NewClusterPreferences() ([]*instancetypev1beta1.VirtualMachineClusterPreference, error) {
	return decodeResources[instancetypev1beta1.VirtualMachineClusterPreference](clusterPreferencesBundle)
}

type clusterType interface {
	instancetypev1beta1.VirtualMachineClusterInstancetype | instancetypev1beta1.VirtualMachineClusterPreference
}

func decodeResources[C clusterType](b []byte) ([]*C, error) {
	decoder := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(b), 1024)
	var bundle []*C
	for {
		bundleResource := new(C)
		err := decoder.Decode(bundleResource)
		if err == io.EOF {
			return bundle, nil
		}
		if err != nil {
			return nil, err
		}
		bundle = append(bundle, bundleResource)
	}
}
