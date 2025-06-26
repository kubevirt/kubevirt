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
 * Copyright The KubeVirt Authors
 *
 */

package testing

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/testing"
)

// PUT

type PutAction[T any] interface {
	testing.Action
	GetName() string
	GetOptions() T
}

type PutActionImpl[T any] struct {
	testing.ActionImpl
	Name    string
	Options T
}

func (a PutActionImpl[T]) GetName() string {
	return a.Name
}

func (a PutActionImpl[T]) GetOptions() T {
	return a.Options
}

func (a PutActionImpl[T]) DeepCopy() testing.Action {
	return PutActionImpl[T]{
		ActionImpl: a.ActionImpl.DeepCopy().(testing.ActionImpl),
		Name:       a.Name,
		Options:    a.Options,
	}
}

func NewRootPutAction[T any](resource schema.GroupVersionResource, name string, options T) PutActionImpl[T] {
	action := PutActionImpl[T]{}
	action.Verb = "put"
	action.Resource = resource
	action.Name = name
	action.Options = options

	return action
}

func NewPutAction[T any](resource schema.GroupVersionResource, namespace, name string, options T) PutActionImpl[T] {
	action := PutActionImpl[T]{}
	action.Verb = "put"
	action.Resource = resource
	action.Namespace = namespace
	action.Name = name
	action.Options = options

	return action
}

func NewRootPutSubresourceAction[T any](resource schema.GroupVersionResource, subresource, name string, options T) PutActionImpl[T] {
	action := PutActionImpl[T]{}
	action.Verb = "put"
	action.Resource = resource
	action.Subresource = subresource
	action.Name = name
	action.Options = options

	return action
}

func NewPutSubresourceAction[T any](resource schema.GroupVersionResource, namespace, subresource, name string, options T) PutActionImpl[T] {
	action := PutActionImpl[T]{}
	action.Verb = "put"
	action.Resource = resource
	action.Subresource = subresource
	action.Namespace = namespace
	action.Name = name
	action.Options = options

	return action
}

// GET

type GetAction[T any] interface {
	testing.Action
	GetName() string
	GetOptions() T
}

type GetActionImpl[T any] struct {
	testing.ActionImpl
	Name    string
	Options T
}

func (a GetActionImpl[T]) GetName() string {
	return a.Name
}

func (a GetActionImpl[T]) GetOptions() T {
	return a.Options
}

func (a GetActionImpl[T]) DeepCopy() testing.Action {
	return GetActionImpl[T]{
		ActionImpl: a.ActionImpl.DeepCopy().(testing.ActionImpl),
		Name:       a.Name,
		Options:    a.Options,
	}
}

func NewRootGetAction[T any](resource schema.GroupVersionResource, name string, options T) GetActionImpl[T] {
	action := GetActionImpl[T]{}
	action.Verb = "get"
	action.Resource = resource
	action.Name = name
	action.Options = options

	return action
}

func NewGetAction[T any](resource schema.GroupVersionResource, namespace, name string, options T) GetActionImpl[T] {
	action := GetActionImpl[T]{}
	action.Verb = "get"
	action.Resource = resource
	action.Namespace = namespace
	action.Name = name
	action.Options = options

	return action
}

func NewRootGetSubresourceAction[T any](resource schema.GroupVersionResource, subresource, name string, options T) GetActionImpl[T] {
	action := GetActionImpl[T]{}
	action.Verb = "get"
	action.Resource = resource
	action.Subresource = subresource
	action.Name = name
	action.Options = options

	return action
}

func NewGetSubresourceAction[T any](resource schema.GroupVersionResource, namespace, subresource, name string, options T) GetActionImpl[T] {
	action := GetActionImpl[T]{}
	action.Verb = "get"
	action.Resource = resource
	action.Subresource = subresource
	action.Namespace = namespace
	action.Name = name
	action.Options = options

	return action
}
