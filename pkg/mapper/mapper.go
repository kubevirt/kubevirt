/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package mapper

import (
	"reflect"

	"github.com/jeevatkm/go-model"
	"k8s.io/client-go/pkg/util/errors"
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
		err := Copy(out, in.Interface())
		return reflect.ValueOf(out).Elem(), err
	})
}

func addStructPtrConversion(inType reflect.Type, outType reflect.Type) {
	model.AddConversionByType(inType, outType, func(in reflect.Value) (reflect.Value, error) {
		out := reflect.New(outType.Elem()).Interface()
		err := Copy(out, in.Elem().Interface())
		return reflect.ValueOf(out), err
	})
}

func AddPtrConversion(inPtrPtr interface{}, outPtrPtr interface{}) {
	// Copy content of pointers
	inType := reflect.TypeOf(inPtrPtr).Elem()
	outType := reflect.TypeOf(outPtrPtr).Elem()
	addStructPtrConversion(inType, outType)
	addStructPtrConversion(outType, inType)
}

func Copy(dst, src interface{}) error {
	return errors.NewAggregate(model.Copy(dst, src))
}
