/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/* TODO:
Taken from https://github.com/kubernetes/kubernetes/blob/1889a6ef52eb18b08e24843577c5b9d3b9a65daa/pkg/controller/controller_utils_test.go
As soon as expectations become available in client-go or apimachinery, delete this.
*/

package controller

import (
	"fmt"
	"sync"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
)

const namespaceKubevirt = "kubevirt"

// ValidSecurityContextWithContainerDefaults creates a valid security context provider based on
// empty container defaults.  Used for testing.
func ValidSecurityContextWithContainerDefaults() *v1.SecurityContext {
	priv := false
	return &v1.SecurityContext{
		Capabilities: &v1.Capabilities{},
		Privileged:   &priv,
	}
}

// NewFakeControllerExpectationsLookup creates a fake store for PodExpectations.
func NewFakeControllerExpectationsLookup(ttl time.Duration) (*ControllerExpectations, *clock.FakeClock) {
	fakeTime := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	fakeClock := clock.NewFakeClock(fakeTime)
	ttlPolicy := &cache.TTLPolicy{TTL: ttl, Clock: fakeClock}
	ttlStore := cache.NewFakeExpirationStore(
		ExpKeyFunc, nil, ttlPolicy, fakeClock)
	return &ControllerExpectations{ttlStore, ""}, fakeClock
}

func newReplicationController(replicas int) *v1.ReplicationController {

	rc := &v1.ReplicationController{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1"},
		ObjectMeta: metav1.ObjectMeta{
			UID:             uuid.NewUUID(),
			Name:            "foobar",
			Namespace:       namespaceKubevirt,
			ResourceVersion: "18",
		},
		Spec: v1.ReplicationControllerSpec{
			Replicas: func() *int32 { i := int32(replicas); return &i }(),
			Selector: map[string]string{"foo": "bar"},
			Template: &v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"name": "foo",
						"type": "production",
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Image:                  "foo/bar",
							TerminationMessagePath: v1.TerminationMessagePathDefault,
							ImagePullPolicy:        v1.PullIfNotPresent,
							SecurityContext:        ValidSecurityContextWithContainerDefaults(),
						},
					},
					RestartPolicy: v1.RestartPolicyAlways,
					DNSPolicy:     v1.DNSDefault,
					NodeSelector: map[string]string{
						"baz": "blah",
					},
				},
			},
		},
	}
	return rc
}

// create count pods with the given phase for the given rc (same selectors and kubevirtNamespace), and add them to the store.
func newPodList(store cache.Store, count int, status v1.PodPhase, rc *v1.ReplicationController) *v1.PodList {
	pods := []v1.Pod{}
	for i := 0; i < count; i++ {
		newPod := v1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("pod%d", i),
				Labels:    rc.Spec.Selector,
				Namespace: rc.Namespace,
			},
			Status: v1.PodStatus{Phase: status},
		}
		if store != nil {
			store.Add(&newPod)
		}
		pods = append(pods, newPod)
	}
	return &v1.PodList{
		Items: pods,
	}
}

func TestControllerExpectations(t *testing.T) {
	ttl := 30 * time.Second
	e, fakeClock := NewFakeControllerExpectationsLookup(ttl)
	// In practice we can't really have add and delete expectations since we only either create or
	// delete replicas in one rc pass, and the rc goes to sleep soon after until the expectations are
	// either fulfilled or timeout.
	adds, dels := 10, 30
	rc := newReplicationController(1)

	// RC fires off adds and deletes at apiserver, then sets expectations
	rcKey, err := KeyFunc(rc)
	if err != nil {
		t.Errorf("Couldn't get key for object %#v: %v", rc, err)
	}
	e.SetExpectations(rcKey, adds, dels)
	var wg sync.WaitGroup
	for i := 0; i < adds+1; i++ {
		wg.Add(1)
		go func() {
			// In prod this can happen either because of a failed create by the rc
			// or after having observed a create via informer
			e.CreationObserved(rcKey)
			wg.Done()
		}()
	}
	wg.Wait()

	// There are still delete expectations
	if e.SatisfiedExpectations(rcKey) {
		t.Errorf("Rc will sync before expectations are met")
	}
	for i := 0; i < dels+1; i++ {
		wg.Add(1)
		go func() {
			e.DeletionObserved(rcKey)
			wg.Done()
		}()
	}
	wg.Wait()

	// Expectations have been surpassed
	if podExp, exists, err := e.GetExpectations(rcKey); err == nil && exists {
		add, del := podExp.GetExpectations()
		if add != -1 || del != -1 {
			t.Errorf("Unexpected pod expectations %#v", podExp)
		}
	} else {
		t.Errorf("Could not get expectations for rc, exists %v and err %v", exists, err)
	}
	if !e.SatisfiedExpectations(rcKey) {
		t.Errorf("Expectations are met but the rc will not sync")
	}

	// Next round of rc sync, old expectations are cleared
	e.SetExpectations(rcKey, 1, 2)
	if podExp, exists, err := e.GetExpectations(rcKey); err == nil && exists {
		add, del := podExp.GetExpectations()
		if add != 1 || del != 2 {
			t.Errorf("Unexpected pod expectations %#v", podExp)
		}
	} else {
		t.Errorf("Could not get expectations for rc, exists %v and err %v", exists, err)
	}

	// Expectations have expired because of ttl
	fakeClock.Step(ttl + 1)
	if !e.SatisfiedExpectations(rcKey) {
		t.Errorf("Expectations should have expired but didn't")
	}
}

func TestUIDExpectations(t *testing.T) {
	uidExp := NewUIDTrackingControllerExpectations(NewControllerExpectations())
	rcList := []*v1.ReplicationController{
		newReplicationController(2),
		newReplicationController(1),
		newReplicationController(0),
		newReplicationController(5),
	}
	rcToPods := map[string][]string{}
	rcKeys := []string{}
	for i := range rcList {
		rc := rcList[i]
		rcName := fmt.Sprintf("rc-%v", i)
		rc.Name = rcName
		rc.Spec.Selector[rcName] = rcName
		podList := newPodList(nil, 5, v1.PodRunning, rc)
		rcKey, err := KeyFunc(rc)
		if err != nil {
			t.Fatalf("Couldn't get key for object %#v: %v", rc, err)
		}
		rcKeys = append(rcKeys, rcKey)
		rcPodNames := []string{}
		for i := range podList.Items {
			p := &podList.Items[i]
			p.Name = fmt.Sprintf("%v-%v", p.Name, rc.Name)
			rcPodNames = append(rcPodNames, PodKey(p))
		}
		rcToPods[rcKey] = rcPodNames
		uidExp.ExpectDeletions(rcKey, rcPodNames)
	}
	for i := range rcKeys {
		j := rand.Intn(i + 1)
		rcKeys[i], rcKeys[j] = rcKeys[j], rcKeys[i]
	}
	for _, rcKey := range rcKeys {
		if uidExp.SatisfiedExpectations(rcKey) {
			t.Errorf("Controller %v satisfied expectations before deletion", rcKey)
		}
		for _, p := range rcToPods[rcKey] {
			uidExp.DeletionObserved(rcKey, p)
		}
		if !uidExp.SatisfiedExpectations(rcKey) {
			t.Errorf("Controller %v didn't satisfy expectations after deletion", rcKey)
		}
		uidExp.DeleteExpectations(rcKey)
		if uidExp.GetUIDs(rcKey) != nil {
			t.Errorf("Failed to delete uid expectations for %v", rcKey)
		}
	}
}
