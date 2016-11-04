package v1

import (
	"encoding/xml"
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	kubeapi "k8s.io/client-go/1.5/pkg/api"
	"k8s.io/client-go/1.5/pkg/api/unversioned"
	"kubevirt/core/pkg/api"
	"reflect"
)

func init() {
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
	VMs                  []VM `json:"items"`
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
