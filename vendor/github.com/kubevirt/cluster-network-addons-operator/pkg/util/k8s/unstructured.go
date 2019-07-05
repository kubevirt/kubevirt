package k8s

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

// ToUnstructured convers an arbitrary object (which MUST obey the
// k8s object conventions) to an Unstructured
func ToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert to unstructured (marshal)")
	}
	u := &unstructured.Unstructured{}
	if err := json.Unmarshal(b, u); err != nil {
		return nil, errors.Wrapf(err, "failed to convert to unstructured (unmarshal)")
	}
	return u, nil
}

// UnstructuredFromYaml creates an unstructured object from a raw yaml string
func UnstructuredFromYaml(obj string) *unstructured.Unstructured {
	buf := bytes.NewBufferString(obj)
	decoder := yaml.NewYAMLOrJSONDecoder(buf, 4096)

	u := unstructured.Unstructured{}
	err := decoder.Decode(&u)
	if err != nil {
		panic(fmt.Sprintf("failed to parse test yaml: %v", err))
	}

	return &u
}
