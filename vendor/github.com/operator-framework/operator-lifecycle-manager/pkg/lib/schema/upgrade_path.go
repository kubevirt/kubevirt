package schema

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/ghodss/yaml"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

// Files is a map of files.
type Files map[string][]byte

// Glob searches the `/manifests` directory for files matching the pattern and returns them.
func Glob(pattern string) (Files, error) {
	matching := map[string][]byte{}
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}

	for _, name := range files {
		bytes, err := ioutil.ReadFile(name)
		if err != nil {
			return nil, err
		}
		matching[name] = bytes
	}

	return matching, nil
}

// CheckUpgradePath checks that every ClusterServiceVersion in a package directory has a valid `spec.replaces` field.
func CheckUpgradePath(packageDir string) error {
	replaces := map[string]string{}
	csvFiles, err := Glob(filepath.Join(packageDir, "**.clusterserviceversion.yaml"))
	if err != nil {
		return err
	}

	for _, bytes := range csvFiles {
		jsonBytes, err := yaml.YAMLToJSON(bytes)
		if err != nil {
			return err
		}
		var csv v1alpha1.ClusterServiceVersion
		err = json.Unmarshal(jsonBytes, &csv)
		if err != nil {
			return err
		}
		replaces[csv.ObjectMeta.Name] = csv.Spec.Replaces
	}

	for replacing, replaced := range replaces {
		fmt.Printf("%s -> %s\n", replaced, replacing)

		if _, ok := replaces[replaced]; replaced != "" && !ok {
			err := fmt.Errorf("%s should replace %s, which does not exist", replacing, replaced)
			return err
		}
	}
	return nil
}
