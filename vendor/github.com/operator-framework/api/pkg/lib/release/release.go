package release

import (
	"encoding/json"
	"slices"
	"strings"

	semver "github.com/blang/semver/v4"
)

// +k8s:openapi-gen=true
// OperatorRelease is a wrapper around a slice of semver.PRVersion which supports correct
// marshaling to YAML and JSON.
// +kubebuilder:validation:Type=string
// +kubebuilder:validation:MaxLength=20
// +kubebuilder:validation:XValidation:rule="self.matches('^[0-9A-Za-z-]+(\\\\.[0-9A-Za-z-]+)*$')",message="release version must be composed of dot-separated identifiers containing only alphanumerics and hyphens"
// +kubebuilder:validation:XValidation:rule="!self.split('.').exists(x, x.matches('^0[0-9]+$'))",message="numeric identifiers in release version must not have leading zeros"
type OperatorRelease struct {
	Release []semver.PRVersion `json:"-"`
}

// DeepCopyInto creates a deep-copy of the Version value.
func (v *OperatorRelease) DeepCopyInto(out *OperatorRelease) {
	out.Release = slices.Clone(v.Release)
}

// MarshalJSON implements the encoding/json.Marshaler interface.
func (v OperatorRelease) MarshalJSON() ([]byte, error) {
	segments := []string{}
	for _, segment := range v.Release {
		segments = append(segments, segment.String())
	}
	return json.Marshal(strings.Join(segments, "."))
}

// UnmarshalJSON implements the encoding/json.Unmarshaler interface.
func (v *OperatorRelease) UnmarshalJSON(data []byte) (err error) {
	var versionString string

	if err = json.Unmarshal(data, &versionString); err != nil {
		return
	}

	segments := strings.Split(versionString, ".")
	for _, segment := range segments {
		release, err := semver.NewPRVersion(segment)
		if err != nil {
			return err
		}
		v.Release = append(v.Release, release)
	}

	return nil
}

// OpenAPISchemaType is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
//
// See: https://github.com/kubernetes/kube-openapi/tree/master/pkg/generators
func (_ OperatorRelease) OpenAPISchemaType() []string { return []string{"string"} }

// OpenAPISchemaFormat is used by the kube-openapi generator when constructing
// the OpenAPI spec of this type.
// "semver" is not a standard openapi format but tooling may use the value regardless
func (_ OperatorRelease) OpenAPISchemaFormat() string { return "semver" }

func (r OperatorRelease) String() string {
	segments := []string{}
	for _, segment := range r.Release {
		segments = append(segments, segment.String())
	}
	return strings.Join(segments, ".")
}
