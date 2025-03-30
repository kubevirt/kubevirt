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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package watcher

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"time"

	"kubevirt.io/kubevirt/tests/framework/kubevirt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"

	"kubevirt.io/client-go/log"
)

type ProcessFunc func(event *v1.Event) (done bool)

type ObjectEventWatcher struct {
	object                 runtime.Object
	timeout                *time.Duration
	resourceVersion        string
	startType              startType
	warningPolicy          WarningsPolicy
	dontFailOnMissingEvent bool
}

func New(object runtime.Object) *ObjectEventWatcher {
	return &ObjectEventWatcher{object: object, startType: invalidWatch}
}

func (w *ObjectEventWatcher) Timeout(duration time.Duration) *ObjectEventWatcher {
	w.timeout = &duration
	return w
}

func (w *ObjectEventWatcher) SetWarningsPolicy(wp WarningsPolicy) *ObjectEventWatcher {
	w.warningPolicy = wp
	return w
}

/*
SinceNow sets a watch starting point for events, from the moment on the connection to the apiserver
was established.
*/
func (w *ObjectEventWatcher) SinceNow() *ObjectEventWatcher {
	w.startType = watchSinceNow
	return w
}

/*
SinceWatchedObjectResourceVersion takes the resource version of the runtime object which is watched,
and takes it as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceWatchedObjectResourceVersion() *ObjectEventWatcher {
	w.startType = watchSinceWatchedObjectUpdate
	return w
}

/*
SinceObjectResourceVersion takes the resource version of the passed in runtime object and takes it
as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceObjectResourceVersion(object runtime.Object) *ObjectEventWatcher {
	var err error
	w.startType = watchSinceObjectUpdate
	w.resourceVersion, err = meta.NewAccessor().ResourceVersion(object)
	Expect(err).ToNot(HaveOccurred())
	return w
}

/*
SinceResourceVersion sets the passed in resourceVersion as the starting point for all events to watch for.
*/
func (w *ObjectEventWatcher) SinceResourceVersion(rv string) *ObjectEventWatcher {
	w.resourceVersion = rv
	w.startType = watchSinceResourceVersion
	return w
}

func (w *ObjectEventWatcher) Watch(ctx context.Context, processFunc ProcessFunc, watchedDescription string) {
	Expect(w.startType).ToNot(Equal(invalidWatch))
	resourceVersion := ""

	switch w.startType {
	case watchSinceNow:
		resourceVersion = ""
	case watchSinceObjectUpdate, watchSinceResourceVersion:
		resourceVersion = w.resourceVersion
	case watchSinceWatchedObjectUpdate:
		var err error
		resourceVersion, err = meta.NewAccessor().ResourceVersion(w.object)
		Expect(err).ToNot(HaveOccurred())
	}

	cli := kubevirt.Client()

	f := processFunc

	objectRefOption := func(obj *v1.ObjectReference) []interface{} {
		logParams := make([]interface{}, 0)
		if obj == nil {
			return logParams
		}

		if obj.Namespace != "" {
			logParams = append(logParams, "namespace", obj.Namespace)
		}
		logParams = append(logParams, "name", obj.Name)
		logParams = append(logParams, "kind", obj.Kind)
		logParams = append(logParams, "uid", obj.UID)

		return logParams
	}

	if w.warningPolicy.FailOnWarnings {
		f = func(event *v1.Event) bool {
			msg := fmt.Sprintf("Event(%#v): type: '%v' reason: '%v' %v", event.InvolvedObject, event.Type, event.Reason, event.Message)
			if !w.warningPolicy.shouldIgnoreWarning(event) {
				ExpectWithOffset(1, event.Type).NotTo(Equal(string(WarningEvent)), "Unexpected Warning event received: %s,%s: %s", event.InvolvedObject.Name, event.InvolvedObject.UID, event.Message)
			}
			log.Log.With(objectRefOption(&event.InvolvedObject)).Info(msg)

			return processFunc(event)
		}
	} else {
		f = func(event *v1.Event) bool {
			if event.Type == string(WarningEvent) {
				log.Log.With(objectRefOption(&event.InvolvedObject)).Reason(fmt.Errorf("warning event received")).Error(event.Message)
			} else {
				log.Log.With(objectRefOption(&event.InvolvedObject)).Infof(event.Message)
			}
			return processFunc(event)
		}
	}

	var selector []string
	objectMeta := w.object.(metav1.ObjectMetaAccessor)
	name := objectMeta.GetObjectMeta().GetName()
	namespace := objectMeta.GetObjectMeta().GetNamespace()
	uid := objectMeta.GetObjectMeta().GetUID()

	selector = append(selector, fmt.Sprintf("involvedObject.name=%v", name))
	if namespace != "" {
		selector = append(selector, fmt.Sprintf("involvedObject.namespace=%v", namespace))
	}
	if uid != "" {
		selector = append(selector, fmt.Sprintf("involvedObject.uid=%v", uid))
	}

	eventWatcher, err := cli.CoreV1().Events(v1.NamespaceAll).
		Watch(context.Background(), metav1.ListOptions{
			FieldSelector:   fields.ParseSelectorOrDie(strings.Join(selector, ",")).String(),
			ResourceVersion: resourceVersion,
		})
	if err != nil {
		panic(err)
	}
	defer eventWatcher.Stop()
	done := make(chan struct{})

	go func() {
		defer GinkgoRecover()
		for watchEvent := range eventWatcher.ResultChan() {
			if watchEvent.Type != watch.Error {
				event := watchEvent.Object.(*v1.Event)
				if f(event) {
					close(done)
					break
				}
			} else {
				switch watchEvent.Object.(type) {
				case *metav1.Status:
					status := watchEvent.Object.(*metav1.Status)
					// api server sometimes closes connections to Watch() client command
					// ignore this error, because it will reconnect automatically
					if status.Message != "an error on the server (\"unable to decode an event from the watch stream: http2: response body closed\") has prevented the request from succeeding" {
						Fail(fmt.Sprintf("unexpected error event: %v", errors.FromObject(watchEvent.Object)))
					}
				default:
					Fail(fmt.Sprintf("unexpected error event: %v", errors.FromObject(watchEvent.Object)))
				}
			}
		}
	}()

	if w.timeout != nil {
		select {
		case <-done:
		case <-ctx.Done():
		case <-time.After(*w.timeout):
			if !w.dontFailOnMissingEvent {
				Fail(fmt.Sprintf("Waited for %v seconds on the event stream to match a specific event: %s", w.timeout.Seconds(), watchedDescription), 1)
			}
		}
	} else {
		select {
		case <-ctx.Done():
		case <-done:
		}
	}
}

func (w *ObjectEventWatcher) WaitFor(ctx context.Context, eventType EventType, reason interface{}) (e *v1.Event) {
	w.Watch(ctx, func(event *v1.Event) bool {
		if event.Type == string(eventType) && event.Reason == reflect.ValueOf(reason).String() {
			e = event
			return true
		}
		return false
	}, fmt.Sprintf("event type %s, reason = %s", string(eventType), reflect.ValueOf(reason).String()))
	return
}

func (w *ObjectEventWatcher) WaitNotFor(ctx context.Context, eventType EventType, reason interface{}) (e *v1.Event) {
	w.dontFailOnMissingEvent = true
	w.Watch(ctx, func(event *v1.Event) bool {
		if event.Type == string(eventType) && event.Reason == reflect.ValueOf(reason).String() {
			e = event
			Fail(fmt.Sprintf("Did not expect %s with reason %s", string(eventType), reflect.ValueOf(reason).String()), 1)
			return true
		}
		return false
	}, fmt.Sprintf("not happen event type %s, reason = %s", string(eventType), reflect.ValueOf(reason).String()))
	return
}

type EventType string

const (
	NormalEvent  EventType = "Normal"
	WarningEvent EventType = "Warning"
)

type WarningsPolicy struct {
	FailOnWarnings     bool
	WarningsIgnoreList []string
}

func (wp *WarningsPolicy) shouldIgnoreWarning(event *v1.Event) bool {
	if event.Type == string(WarningEvent) {
		for _, message := range wp.WarningsIgnoreList {
			if strings.Contains(event.Message, message) {
				return true
			}
		}
	}

	return false
}

type startType string

const (
	invalidWatch startType = "invalidWatch"
	// Watch since the moment a long poll connection is established
	watchSinceNow startType = "watchSinceNow"
	// Watch since the resourceVersion of the passed in runtime object
	watchSinceObjectUpdate startType = "watchSinceObjectUpdate"
	// Watch since the resourceVersion of the watched object
	watchSinceWatchedObjectUpdate startType = "watchSinceWatchedObjectUpdate"
	// Watch since the resourceVersion passed in to the builder
	watchSinceResourceVersion startType = "watchSinceResourceVersion"
)
