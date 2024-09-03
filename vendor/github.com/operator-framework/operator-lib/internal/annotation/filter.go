// Copyright 2021 The Operator-SDK Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package annotation contains event handler and predicate builders for annotations.
// There are two types of builders:
//
// - Falsy builders result in objects being queued if the annotation is not present OR contains a falsy value.
// - Truthy builders are the falsy complement: objects will be enqueued if the annotation is present AND contains a truthy value.
//
// Truthiness/falsiness is determined by Go's strconv.ParseBool().
package annotation

import (
	"context"
	"fmt"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Options configures a filter.
type Options struct {
	Log logr.Logger

	// Internally set.
	truthy bool
}

// NewFalsyPredicate returns a predicate that passes objects
// that do not have annotation with key string key or whose value is falsy.
func NewFalsyPredicate[T client.Object](key string, opts Options) (predicate.TypedPredicate[T], error) {
	opts.truthy = false
	return newFilter[T](key, opts)
}

// NewFalsyEventHandler returns an event handler that enqueues objects
// that do not have annotation with key string key or whose value is falsy.
func NewFalsyEventHandler[T client.Object](key string, opts Options) (handler.TypedEventHandler[T, reconcile.Request], error) {
	opts.truthy = false
	return newEventHandler[T](key, opts)
}

// NewTruthyPredicate returns a predicate that passes objects
// that do have annotation with key string key and whose value is truthy.
func NewTruthyPredicate[T client.Object](key string, opts Options) (predicate.TypedPredicate[T], error) {
	opts.truthy = true
	return newFilter[T](key, opts)
}

// NewTruthyEventHandler returns an event handler that enqueues objects
// that do have annotation with key string key and whose value is truthy.
func NewTruthyEventHandler[T client.Object](key string, opts Options) (handler.TypedEventHandler[T, reconcile.Request], error) {
	opts.truthy = true
	return newEventHandler[T](key, opts)
}

func defaultOptions(opts *Options) {
	if opts.Log.GetSink() == nil {
		opts.Log = logf.Log
	}
}

// newEventHandler returns a filter for use as an event handler.
func newEventHandler[T client.Object](key string, opts Options) (handler.TypedEventHandler[T, reconcile.Request], error) {
	f, err := newFilter[T](key, opts)
	if err != nil {
		return nil, err
	}

	f.hdlr = &handler.TypedEnqueueRequestForObject[T]{}
	return handler.TypedFuncs[T, reconcile.Request]{
		CreateFunc: func(ctx context.Context, evt event.TypedCreateEvent[T], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if f.Create(evt) {
				f.hdlr.Create(ctx, evt, q)
			}
		},
		UpdateFunc: func(ctx context.Context, evt event.TypedUpdateEvent[T], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if f.Update(evt) {
				f.hdlr.Update(ctx, evt, q)
			}
		},
		DeleteFunc: func(ctx context.Context, evt event.TypedDeleteEvent[T], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if f.Delete(evt) {
				f.hdlr.Delete(ctx, evt, q)
			}
		},
		GenericFunc: func(ctx context.Context, evt event.TypedGenericEvent[T], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			if f.Generic(evt) {
				f.hdlr.Generic(ctx, evt, q)
			}
		},
	}, nil
}

// newFilter returns a filter for use as a predicate.
func newFilter[T client.Object](key string, opts Options) (*filter[T], error) {
	defaultOptions(&opts)

	// Make sure the annotation key and eventual value are valid together.
	if err := validateAnnotation(key, opts.truthy); err != nil {
		return nil, err
	}

	f := filter[T]{}
	f.key = key
	// Falsy filters return true in all cases except when the annotation is present and true.
	// Truthy filters only return true when the annotation is present and true.
	f.ret = !opts.truthy
	f.log = opts.Log.WithName("pause")
	return &f, nil
}

func validateAnnotation(key string, truthy bool) error {
	fldPath := field.NewPath("metadata", "annotations")
	annotation := map[string]string{key: fmt.Sprintf("%v", truthy)}
	return validation.ValidateAnnotations(annotation, fldPath).ToAggregate()
}

// filter implements a filter for objects with a truthy "paused" annotation (see Key).
// When this annotation is removed or value does not evaluate to "true",
// the controller will see events from these objects again.
type filter[T client.Object] struct {
	key  string
	ret  bool
	log  logr.Logger
	hdlr *handler.TypedEnqueueRequestForObject[T]
}

// Create implements predicate.Predicate.Create().
func (f *filter[T]) Create(evt event.TypedCreateEvent[T]) bool {
	var obj client.Object = evt.Object
	if obj == nil {
		f.log.Error(nil, "CreateEvent received with no object", "event", evt)
		return f.ret
	}
	return f.run(obj)
}

// Update implements predicate.Predicate.Update().
func (f *filter[T]) Update(evt event.TypedUpdateEvent[T]) bool {
	var newObj client.Object = evt.ObjectNew
	if newObj != nil {
		return f.run(newObj)
	}

	var oldObj client.Object = evt.ObjectOld
	if oldObj != nil {
		return f.run(oldObj)
	}

	if f.hdlr == nil {
		f.log.Error(nil, "UpdateEvent received with no metadata", "event", evt)
	}
	return f.ret
}

// Delete implements predicate.Predicate.Delete().
func (f *filter[T]) Delete(evt event.TypedDeleteEvent[T]) bool {
	var obj client.Object = evt.Object
	if obj == nil {
		if f.hdlr == nil {
			f.log.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		}
		return f.ret
	}
	return f.run(obj)
}

// Generic implements predicate.Predicate.Generic().
func (f *filter[T]) Generic(evt event.TypedGenericEvent[T]) bool {
	var obj client.Object = evt.Object
	if obj == nil {
		if f.hdlr == nil {
			f.log.Error(nil, "GenericEvent received with no metadata", "event", evt)
		}
		return f.ret
	}
	return f.run(obj)
}

func (f *filter[T]) run(obj client.Object) bool {
	annotations := obj.GetAnnotations()
	if len(annotations) == 0 {
		return f.ret
	}
	annoStr, hasAnno := annotations[f.key]
	if !hasAnno {
		return f.ret
	}
	annoBool, err := strconv.ParseBool(annoStr)
	if err != nil {
		f.log.Error(err, "Bad annotation value", "key", f.key, "value", annoStr)
		return f.ret
	}
	// If the filter is falsy (f.ret == true) and value is false, then the object passes the filter.
	// If the filter is truthy (f.ret == false) and value is true, then the object passes the filter.
	return !annoBool == f.ret
}
