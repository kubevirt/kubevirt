package mapper

import (
	"reflect"

	"github.com/jeevatkm/go-model"
)

func AddConversion(inPtr interface{}, outPtr interface{}) {
	// Copy content of concrete instances
	inType := reflect.TypeOf(inPtr).Elem()
	outType := reflect.TypeOf(outPtr).Elem()
	addStructConversion(inType, outType)
	addStructConversion(outType, inType)
}

func addStructConversion(inType reflect.Type, outType reflect.Type) {
	model.AddConversionByType(inType, outType, func(in reflect.Value) (reflect.Value, error) {
		out := reflect.New(outType).Interface()
		errs := model.Copy(out, in.Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out).Elem(), errs[0]
		}
		return reflect.ValueOf(out).Elem(), nil
	})
}

func addStructPtrConversion(inType reflect.Type, outType reflect.Type) {
	model.AddConversionByType(inType, outType, func(in reflect.Value) (reflect.Value, error) {
		out := reflect.New(outType.Elem()).Interface()
		errs := model.Copy(out, in.Elem().Interface())
		if len(errs) > 0 {
			return reflect.ValueOf(out), errs[0]
		}
		return reflect.ValueOf(out), nil
	})
}

func AddPtrConversion(inPtrPtr interface{}, outPtrPtr interface{}) {
	// Copy content of pointers
	inType := reflect.TypeOf(inPtrPtr).Elem()
	outType := reflect.TypeOf(outPtrPtr).Elem()
	addStructPtrConversion(inType, outType)
	addStructPtrConversion(outType, inType)
}
