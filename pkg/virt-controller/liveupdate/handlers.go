/*
 * This file is part of the KubeVirt project
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
 * Copyright 2024 The KubeVirt Authors.
 *
 */

package liveupdate

import (
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	v1 "kubevirt.io/api/core/v1"
)

func fieldExists(obj interface{}, path string) bool {
	t := reflect.TypeOf(obj)
	path = strings.Trim(path, "/")

	for _, field := range strings.Split(path, "/") {
		elem, found := t.FieldByName(field)
		if !found {
			return false
		}

		t = elem.Type

		if t.Kind() == reflect.Pointer {
			t = t.Elem()
		}
	}

	return true
}

func getField(vm *v1.VirtualMachine, path string) (reflect.Value, error) {
	path = strings.Trim(path, "/")
	v := reflect.ValueOf(vm).Elem()

	for _, field := range strings.Split(path, "/") {
		v = v.FieldByName(field)

		if v.Kind() == reflect.Pointer {
			v = v.Elem()
		}

		if !v.IsValid() {
			return reflect.Value{}, fmt.Errorf("field '%s' not found", path)
		}
	}
	return v, nil
}

func copyFields(dstVM, srcVM *v1.VirtualMachine, pathList []string) error {
	for _, path := range pathList {
		value, err := getField(srcVM, path)
		if err != nil {
			return err
		}
		dst, err := getField(dstVM, path)
		if err != nil {
			return err
		}

		if !dst.CanSet() {
			return fmt.Errorf("field '%s' of destination object cannot be set", path)
		}
		dst.Set(value)
	}
	return nil
}

type LiveUpdateHandler interface {
	// GetManagedFields() should return an array containing the paths
	// that must be filtered out when comparing a new VM spec
	// with its last seen version.
	//
	// Every string in an array must be a forward-slash separated
	// path of the field that has to be filtered out.
	// Example: to filter out .spec.template.spec.domain.memory.guest you should supply
	// '/Spec/Template/Spec/Domain/Memory/Guest'.
	// Every element of the path is a Go struct field name.
	GetManagedFields() []string
	// HandleLiveUpdate() should implement the live-update operation
	// the function receives the current VM and VMI objects as parameters
	HandleLiveUpdate(vm *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error
}

type LiveUpdater struct {
	handlers []LiveUpdateHandler
}

func (up *LiveUpdater) RegisterHandlers(handlers ...LiveUpdateHandler) error {
	for _, handler := range handlers {
		for _, field := range handler.GetManagedFields() {
			if !fieldExists(v1.VirtualMachine{}, field) {
				return fmt.Errorf("cannot register live-update handler, field '%s' does not exist", field)
			}
		}

		up.handlers = append(up.handlers, handler)
	}
	return nil
}

func (up *LiveUpdater) FilterUpdatableFields(vm, oldVM *v1.VirtualMachine) error {
	for _, handler := range up.handlers {
		if err := copyFields(vm, oldVM, handler.GetManagedFields()); err != nil {
			return err
		}
	}
	return nil
}

func (up *LiveUpdater) HandleLiveUpdates(vm, oldVM *v1.VirtualMachine, vmi *v1.VirtualMachineInstance) error {
	for _, handler := range up.handlers {
		for _, field := range handler.GetManagedFields() {
			oldValue, err := getField(oldVM, field)
			if err != nil {
				return err
			}

			newValue, err := getField(vm, field)
			if err != nil {
				return err
			}

			if !equality.Semantic.DeepEqual(oldValue.Interface(), newValue.Interface()) {
				// only call one handler at a time
				return handler.HandleLiveUpdate(vm, vmi)
			}
		}
	}

	return nil
}
