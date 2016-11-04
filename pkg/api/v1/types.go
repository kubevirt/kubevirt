package v1

import (
	"encoding/xml"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"k8s.io/client-go/1.5/pkg/apimachinery/announced"
	"k8s.io/client-go/1.5/pkg/runtime"
	"kubevirt/core/pkg/api"
	"reflect"
)

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// GroupName is the group name use in this package
const GroupName = "kubevirt.io"

// GroupVersion is group version used to register these objects
var GroupVersion = unversioned.GroupVersion{Group: GroupName, Version: "v1alpha1"}

// Adds the list of known types to api.Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&VM{},
		&VMList{},
	)
	return nil
}

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:              GroupName,
			VersionPreferenceOrder: []string{GroupVersion.Version},
			ImportPrefix:           "kubevirt/core/pgk/api/v1/types",
		},
		announced.VersionToSchemeFunc{
			GroupVersion.Version: AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}

	model.AddConversion((*uuid.UUID)(nil), (*string)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(in.Interface().(uuid.UUID).String()), nil
	})
	model.AddConversion((*string)(nil), (*uuid.UUID)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(uuid.FromStringOrNil(in.String())), nil
	})
	model.AddConversion((*VMSpec)(nil), (*api.VMSpec)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := api.VMSpec{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
	model.AddConversion((*api.VMSpec)(nil), (*VMSpec)(nil), func(in reflect.Value) (reflect.Value, error) {
		out := VMSpec{}
		errs := model.Copy(&out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
}

type VM struct {
	unversioned.TypeMeta `json:",inline"`
	kubeapi.ObjectMeta   `json:"metadata,omitempty"`
	Spec                 VMSpec `json:"spec,omitempty" valid:"required"`
}

type VMList struct {
	unversioned.TypeMeta `json:",inline"`
	unversioned.ListMeta `json:"metadata,omitempty"`
	Items                []VM `json:"items"`
}

type VMSpec struct {
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

type Domain struct {
	Name         string   `json:"name" xml:"name" valid:"required"`
	UUID         string   `json:"uuid" xml:"uuid" valid:"uuid"`
	XMLName      xml.Name `xml:"domain"`
	RawDomain    []byte
	NodeSelector map[string]string `json:"-"`
}
