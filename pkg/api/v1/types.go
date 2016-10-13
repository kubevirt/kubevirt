package v1

import (
	"github.com/rmohr/go-model"
	"github.com/satori/go.uuid"
	"reflect"
)

type VM struct {
	Name         string            `json:"name" xml:"name" valid:"required"`
	UUID         string            `json:"uuid" xml:"uuid" valid:"uuid"`
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
}

func init() {
	model.AddConversion((*uuid.UUID)(nil), (*string)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(in.Interface().(uuid.UUID).String()), nil
	})
	model.AddConversion((*string)(nil), (*uuid.UUID)(nil), func(in reflect.Value) (reflect.Value, error) {
		return reflect.ValueOf(uuid.FromStringOrNil(in.String())), nil
	})
}
