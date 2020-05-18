package nodelabeller

import (
	"encoding/xml"
	"io/ioutil"
)

const (
	usableNo string = "no"
)

var domCapabilitiesFilePath = nodeLabellerVolumePath + "/virsh_domcapabilities.xml"

// getCPUInfo retrieves xml data from file and parse them.
// Output of this function is slice of usable cpu models and features.
// Only models with tag usable yes will be used.
func (n *NodeLabeller) getCPUInfo() ([]string, map[string]bool, error) {
	hostDomCapabilities := HostDomCapabilities{}
	err := getStructureFromXMLFile(domCapabilitiesFilePath, &hostDomCapabilities)
	if err != nil {
		return nil, nil, err
	}

	obsoleteCPUsx86 := n.clusterConfig.GetObsoleteCPUs()

	basicFeaturesMap := make(map[string]bool)
	cpus := make([]string, 0)
	features := make(map[string]bool)
	var newFeatures map[string]bool

	for _, mode := range hostDomCapabilities.CPU.Mode {
		if mode.Vendor.Name != "" {
			minCPU := n.clusterConfig.GetMinCPU()
			var err error
			basicFeaturesMap, err = parseFeatures(basicFeaturesMap, minCPU)
			if err != nil {
				return nil, nil, err
			}
		}

		for _, model := range mode.Model {
			if _, ok := obsoleteCPUsx86[model.Name]; ok || model.Usable == usableNo || model.Usable == "" {
				continue
			}

			newFeatures, err = parseFeatures(basicFeaturesMap, model.Name)
			if err != nil {
				return nil, nil, err
			}

			features = unionMap(features, newFeatures)

			cpus = append(cpus, model.Name)
		}
	}
	return cpus, features, nil
}

//parseFeatures loads features from file and returns only new features which are not in basic features
func parseFeatures(basicFeatures map[string]bool, cpuName string) (map[string]bool, error) {
	features, err := loadFeatures(cpuName)
	if err != nil {
		return nil, err
	}
	return subtractMap(features, basicFeatures), nil
}

//LoadFeatures loads features for given cpu name
func loadFeatures(cpuModelName string) (map[string]bool, error) {
	if cpuModelName == "" {
		return map[string]bool{}, nil
	}

	cpuFeatures := FeatureModel{}
	cpuFeaturepath := getPathCPUFefatures(cpuModelName)
	err := getStructureFromXMLFile(cpuFeaturepath, &cpuFeatures)
	if err != nil {
		return nil, err
	}

	features := make(map[string]bool)
	for _, f := range cpuFeatures.Model.Features {
		features[f.Name] = true
	}
	return features, nil
}

//getPathCPUFefatures creates path where folder with cpu models is
func getPathCPUFefatures(name string) string {
	return nodeLabellerVolumePath + "/cpu_map/" + "x86_" + name + ".xml"
}

func unionMap(a, b map[string]bool) map[string]bool {
	unionMap := make(map[string]bool)
	for feature := range a {
		unionMap[feature] = true
	}
	for feature := range b {
		unionMap[feature] = true
	}
	return unionMap
}

func subtractMap(a, b map[string]bool) map[string]bool {
	new := make(map[string]bool)
	for k := range a {
		if _, ok := b[k]; !ok {
			new[k] = true
		}
	}
	return new
}

func convertStringSliceToMap(s []string) map[string]bool {
	result := make(map[string]bool)
	for _, v := range s {
		result[v] = true
	}
	return result
}

//GetStructureFromXMLFile load data from xml file and unmarshals them into given structure
//Given structure has to be pointer
func getStructureFromXMLFile(path string, structure interface{}) error {
	rawFile, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	//unmarshal data into structure
	err = xml.Unmarshal(rawFile, structure)
	if err != nil {
		return err
	}
	return nil
}
