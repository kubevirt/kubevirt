package resolver

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strings"

	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	opregistry "github.com/operator-framework/operator-registry/pkg/registry"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type CatalogKey struct {
	Name      string
	Namespace string
}

func (k *CatalogKey) String() string {
	return fmt.Sprintf("%s/%s", k.Name, k.Namespace)
}

type APISet map[opregistry.APIKey]struct{}

func EmptyAPISet() APISet {
	return map[opregistry.APIKey]struct{}{}
}

func (s APISet) PopAPIKey() *opregistry.APIKey {
	for a := range s {
		api := &opregistry.APIKey{
			Group:   a.Group,
			Version: a.Version,
			Kind:    a.Kind,
			Plural:  a.Plural,
		}
		delete(s, a)
		return api
	}
	return nil
}

func GVKStringToProvidedAPISet(gvksStr string) APISet {
	set := make(APISet)
	// TODO: Should we make gvk strings lowercase to avoid issues with user set gvks?
	gvks := strings.Split(strings.Replace(gvksStr, " ", "", -1), ",")
	for _, gvkStr := range gvks {
		gvk, _ := schema.ParseKindArg(gvkStr)
		if gvk != nil {
			set[opregistry.APIKey{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind}] = struct{}{}
		}
	}

	return set
}

func APIKeyToGVKString(key opregistry.APIKey) string {
	// TODO: Add better validation of GVK
	return strings.Join([]string{key.Kind, key.Version, key.Group}, ".")
}

func APIKeyToGVKHash(key opregistry.APIKey) (string, error) {
	hash := fnv.New64a()
	if _, err := hash.Write([]byte(APIKeyToGVKString(key))); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum64()), nil
}

func (s APISet) String() string {
	gvkStrs := make([]string, len(s))
	i := 0
	for api := range s {
		// TODO: Only add valid GVK strings
		gvkStrs[i] = APIKeyToGVKString(api)
		i++
	}
	sort.Strings(gvkStrs)

	return strings.Join(gvkStrs, ",")
}

// TODO: Generalize set logic and make an abstraction for sets to implemement to feed into it.

// Union returns the union of the APISet and the given list of APISets
func (s APISet) Union(sets ...APISet) APISet {
	union := make(APISet)
	for api := range s {
		union[api] = struct{}{}
	}
	for _, set := range sets {
		for api := range set {
			union[api] = struct{}{}
		}
	}

	return union
}

// Intersection returns the intersection of the APISet and the given list of APISets
func (s APISet) Intersection(sets ...APISet) APISet {
	intersection := make(APISet)
	for _, set := range sets {
		for api := range set {
			if _, ok := s[api]; ok {
				intersection[api] = struct{}{}
			}
		}
	}

	return intersection
}

func (s APISet) Difference(set APISet) APISet {
	difference := make(APISet).Union(s)
	for api := range set {
		if _, ok := difference[api]; ok {
			delete(difference, api)
		}
	}

	return difference
}

// IsSubset returns true if the APISet is a subset of the given one
func (s APISet) IsSubset(set APISet) bool {
	for api := range s {
		if _, ok := set[api]; !ok {
			return false
		}
	}

	return true
}

// StripPlural returns the APISet with the Plural field of all APIKeys removed
func (s APISet) StripPlural() APISet {
	set := make(APISet)
	for api := range s {
		set[opregistry.APIKey{Group: api.Group, Version: api.Version, Kind: api.Kind}] = struct{}{}
	}

	return set
}

type APIOwnerSet map[opregistry.APIKey]OperatorSurface

func EmptyAPIOwnerSet() APIOwnerSet {
	return map[opregistry.APIKey]OperatorSurface{}
}

type OperatorSet map[string]OperatorSurface

func EmptyOperatorSet() OperatorSet {
	return map[string]OperatorSurface{}
}

// Snapshot returns a new set, pointing to the same values
func (o OperatorSet) Snapshot() OperatorSet {
	out := make(map[string]OperatorSurface)
	for key, val := range o {
		out[key] = val
	}
	return out
}

type APIMultiOwnerSet map[opregistry.APIKey]OperatorSet

func EmptyAPIMultiOwnerSet() APIMultiOwnerSet {
	return map[opregistry.APIKey]OperatorSet{}
}

func (s APIMultiOwnerSet) PopAPIKey() *opregistry.APIKey {
	for a := range s {
		api := &opregistry.APIKey{
			Group:   a.Group,
			Version: a.Version,
			Kind:    a.Kind,
			Plural:  a.Plural,
		}
		delete(s, a)
		return api
	}
	return nil
}

func (s APIMultiOwnerSet) PopAPIRequirers() OperatorSet {
	requirers := EmptyOperatorSet()
	for a := range s {
		for key, op := range s[a] {
			requirers[key] = op
		}
		delete(s, a)
		return requirers
	}
	return nil
}

type OperatorSourceInfo struct {
	Package     string
	Channel     string
	StartingCSV string
	Catalog     CatalogKey
}

func (i *OperatorSourceInfo) String() string {
	return fmt.Sprintf("%s/%s in %s/%s", i.Package, i.Channel, i.Catalog.Name, i.Catalog.Namespace)
}

var ExistingOperator = OperatorSourceInfo{"", "", "", CatalogKey{"", ""}}

// OperatorSurface describes the API surfaces provided and required by an Operator.
type OperatorSurface interface {
	ProvidedAPIs() APISet
	RequiredAPIs() APISet
	Identifier() string
	Replaces() string
	SourceInfo() *OperatorSourceInfo
	Bundle() *opregistry.Bundle
}

type Operator struct {
	name         string
	replaces     string
	providedAPIs APISet
	requiredAPIs APISet
	bundle       *opregistry.Bundle
	sourceInfo   *OperatorSourceInfo
}

var _ OperatorSurface = &Operator{}

func NewOperatorFromBundle(bundle *opregistry.Bundle, startingCSV string, sourceKey CatalogKey) (*Operator, error) {
	csv, err := bundle.ClusterServiceVersion()
	if err != nil {
		return nil, err
	}
	providedAPIs, err := bundle.ProvidedAPIs()
	if err != nil {
		return nil, err
	}
	requiredAPIs, err := bundle.RequiredAPIs()
	if err != nil {
		return nil, err
	}
	return &Operator{
		name:         csv.GetName(),
		replaces:     csv.Spec.Replaces,
		providedAPIs: providedAPIs,
		requiredAPIs: requiredAPIs,
		bundle:       bundle,
		sourceInfo: &OperatorSourceInfo{
			Package:     bundle.Package,
			Channel:     bundle.Channel,
			StartingCSV: startingCSV,
			Catalog:     sourceKey,
		},
	}, nil
}

func NewOperatorFromCSV(csv *v1alpha1.ClusterServiceVersion) (*Operator, error) {
	providedAPIs := EmptyAPISet()
	for _, crdDef := range csv.Spec.CustomResourceDefinitions.Owned {
		parts := strings.SplitN(crdDef.Name, ".", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("error parsing crd name: %s", crdDef.Name)
		}
		providedAPIs[opregistry.APIKey{Plural: parts[0], Group: parts[1], Version: crdDef.Version, Kind: crdDef.Kind}] = struct{}{}
	}
	for _, api := range csv.Spec.APIServiceDefinitions.Owned {
		providedAPIs[opregistry.APIKey{Group: api.Group, Version: api.Version, Kind: api.Kind, Plural: api.Name}] = struct{}{}
	}

	requiredAPIs := EmptyAPISet()
	for _, crdDef := range csv.Spec.CustomResourceDefinitions.Required {
		parts := strings.SplitN(crdDef.Name, ".", 2)
		if len(parts) < 2 {
			return nil, fmt.Errorf("error parsing crd name: %s", crdDef.Name)
		}
		requiredAPIs[opregistry.APIKey{Plural: parts[0], Group: parts[1], Version: crdDef.Version, Kind: crdDef.Kind}] = struct{}{}
	}
	for _, api := range csv.Spec.APIServiceDefinitions.Required {
		requiredAPIs[opregistry.APIKey{Group: api.Group, Version: api.Version, Kind: api.Kind, Plural: api.Name}] = struct{}{}
	}

	return &Operator{
		name:         csv.GetName(),
		replaces:     csv.Spec.Replaces,
		providedAPIs: providedAPIs,
		requiredAPIs: requiredAPIs,
		sourceInfo:   &ExistingOperator,
	}, nil
}

func (o *Operator) ProvidedAPIs() APISet {
	return o.providedAPIs
}

func (o *Operator) RequiredAPIs() APISet {
	return o.requiredAPIs
}

func (o *Operator) Identifier() string {
	return o.name
}

func (o *Operator) Replaces() string {
	return o.replaces
}

func (o *Operator) Package() string {
	return o.bundle.Package
}

func (o *Operator) SourceInfo() *OperatorSourceInfo {
	return o.sourceInfo
}

func (o *Operator) Bundle() *opregistry.Bundle {
	return o.bundle
}
