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
 * Copyright 2021 IBM, Inc.
 *
 */

package watcher

import (
	"fmt"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
	kvv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"
	objUtils "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
)

const (
	informerTimeout = time.Minute
)

type ObjListWatcher struct {
	sync.Mutex

	// runningObjs is a set to avoid recount duplicated events of an object
	runningObjs            map[string]bool
	desiredObjRunningCount int
	updateChannel          chan bool

	virtCli      kubecli.KubevirtClient
	listOpts     metav1.ListOptions
	ResourceKind string

	informer    cache.SharedInformer
	stopChannel chan struct{}
}

func NewObjListWatcher(virtCli kubecli.KubevirtClient, resourceKind string, objCount int, listOpts metav1.ListOptions) *ObjListWatcher {
	w := &ObjListWatcher{
		stopChannel:            make(chan struct{}),
		updateChannel:          make(chan bool),
		runningObjs:            make(map[string]bool),
		virtCli:                virtCli,
		ResourceKind:           resourceKind,
		desiredObjRunningCount: objCount,
		listOpts:               listOpts,
	}

	labelSelector, err := labels.Parse(w.listOpts.LabelSelector)
	if err != nil {
		panic(err)
	}
	objListWatcher := controller.NewListWatchFromClient(
		virtCli.RestClient(),
		w.ResourceKind,
		k8sv1.NamespaceAll,
		fields.Everything(),
		labelSelector)

	w.informer = cache.NewSharedInformer(objListWatcher, nil, 0)
	w.stopChannel = make(chan struct{})
	w.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(oldObj, newObj interface{}) {
			w.handleUpdate(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			w.handleDeleted(obj)
		},
	})

	return w
}

func (w *ObjListWatcher) Run() {
	go w.informer.Run(w.stopChannel)
	timeoutCh := make(chan struct{})
	timeoutTimer := time.AfterFunc(informerTimeout, func() {
		close(timeoutCh)
	})
	defer timeoutTimer.Stop()
	if !cache.WaitForCacheSync(timeoutCh, w.informer.HasSynced) {
		panic(fmt.Errorf("watcher timed out waiting for caches to sync"))
	}
	// waiting 1s to prevent the watcher miss events from the test
	time.Sleep(1 * time.Second)
}

func (w *ObjListWatcher) Stop() {
	close(w.stopChannel)
}

func (w *ObjListWatcher) handleUpdate(obj interface{}) {
	switch w.ResourceKind {
	case objUtils.VMIResource:
		vmi, ok := obj.(*kvv1.VirtualMachineInstance)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachineInstance obj: %v", w.ResourceKind)
		}
		if vmi.Status.Phase == kvv1.Running {
			w.addObj(w.getID(vmi))
		}
		if vmi.Status.Phase == kvv1.Succeeded || vmi.Status.Phase == kvv1.Failed {
			log.Log.Errorf("VirtualMachineInstance obj: %s is a final state, waiting for garbage collection", fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name))
		}

	case objUtils.VMResource:
		vm, ok := obj.(*kvv1.VirtualMachine)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachine obj: %v", w.ResourceKind)
		}
		if vm.Status.Ready {
			w.addObj(w.getID(vm))
		}

	case objUtils.VMIReplicaSetResource:
		replicaSet, ok := obj.(*kvv1.VirtualMachineInstanceReplicaSet)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachineInstanceReplicaSet obj: %v", w.ResourceKind)
		}
		w.addObj(w.getID(replicaSet))

	default:
		log.Log.Errorf("Watcher does not support object type %s", w.ResourceKind)
		return
	}
}

func (w *ObjListWatcher) handleDeleted(obj interface{}) {
	switch w.ResourceKind {
	case objUtils.VMResource:
		vm, ok := obj.(*kvv1.VirtualMachine)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachine obj: %v", w.ResourceKind)
			return
		}
		isGarbageCollected, err := w.isGarbageCollected(vm)
		if err != nil {
			log.Log.Errorf("Could not get VirtualMachine: %v, err: %#v", fmt.Sprintf("%s/%s", vm.Namespace, vm.Name), err)
		}
		if !isGarbageCollected {
			log.Log.Errorf("VirtualMachine obj: %v has not been garbage collected yet", fmt.Sprintf("%s/%s", vm.Namespace, vm.Name))
			return
		}
		w.deleteObj(w.getID(vm))

	case objUtils.VMIReplicaSetResource:
		replicaSet, ok := obj.(*kvv1.VirtualMachineInstanceReplicaSet)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachineInstanceReplicaSet obj: %v", w.ResourceKind)
			return
		}
		isGarbageCollected, err := w.isGarbageCollected(replicaSet)
		if err != nil {
			log.Log.Errorf("Could not get VirtualMachineInstanceReplicaSet: %v, err: %#v", fmt.Sprintf("%s/%s", replicaSet.Namespace, replicaSet.Name), err)
		}
		if !isGarbageCollected {
			log.Log.Errorf("VirtualMachineInstanceReplicaSet obj: %v has not been garbage collected yet", fmt.Sprintf("%s/%s", replicaSet.Namespace, replicaSet.Name))
			return
		}
		w.deleteObj(w.getID(replicaSet))
	case objUtils.VMIResource:
		vmi, ok := obj.(*kvv1.VirtualMachineInstance)
		if !ok {
			log.Log.Errorf("Could not convert VirtualMachineInstance obj: %v", w.ResourceKind)
			return
		}
		isGarbageCollected, err := w.isGarbageCollected(vmi)
		if err != nil {
			log.Log.Errorf("Could not get VirtualMachineInstance: %v, err: %#v", fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name), err)
		}
		if !isGarbageCollected {
			log.Log.Errorf("VirtualMachineInstance obj: %v has not been garbage collected yet", fmt.Sprintf("%s/%s", vmi.Namespace, vmi.Name))
			return
		}
		w.deleteObj(w.getID(vmi))

	default:
		return
	}
}

func (w *ObjListWatcher) WaitRunning(timeout time.Duration) error {
	if w.getRunningCount() == w.desiredObjRunningCount {
		return nil
	}
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timeout after %s waiting for objects be running", timeout)

		case <-w.updateChannel:
			count := w.getRunningCount()
			if count == w.desiredObjRunningCount {
				return nil
			}
			log.Log.V(6).Infof("Waiting %d %v to be Running", (w.desiredObjRunningCount - count), w.ResourceKind)
		}
	}
}

func (w *ObjListWatcher) WaitDeletion(timeout time.Duration) error {
	runningCount := w.getRunningCount()
	if runningCount == 0 || runningCount == w.desiredObjRunningCount {
		return nil
	}
	for {
		select {
		case <-time.After(timeout):
			return fmt.Errorf("timeout after %s waiting for objects be deleted", timeout)

		case <-w.updateChannel:
			count := w.getRunningCount()
			if count == 0 {
				return nil
			}
			log.Log.V(6).Infof("Still %d %v waiting to be Garbage Collected", count, w.ResourceKind)
		}
	}
}

func (w *ObjListWatcher) addObj(id string) {
	w.Lock()
	w.runningObjs[id] = true
	w.Unlock()
	w.updateChannel <- true
}

func (w *ObjListWatcher) deleteObj(id string) {
	w.Lock()
	delete(w.runningObjs, id)
	w.Unlock()
	w.updateChannel <- true
}

func (w *ObjListWatcher) getRunningCount() int {
	var count int
	w.Lock()
	count = len(w.runningObjs)
	w.Unlock()
	return count
}

func (w *ObjListWatcher) isGarbageCollected(obj interface{}) (bool, error) {
	objKey, err := controller.KeyFunc(obj)
	if err != nil {
		log.Log.Errorf("Error getting obj key for %s err: %v", obj, err)
		return false, err
	}
	_, exists, err := w.informer.GetStore().GetByKey(objKey)
	switch {
	case err != nil:
		log.Log.Errorf("Error getting obj %s from cache err: %v", obj, err)
		return false, err
	case !exists:
		log.Log.Errorf("Obj %s not found in cache", objKey)
		return true, nil
	default:
		log.Log.Errorf("Obj %s found in cache", objKey)
		return false, nil
	}
}

func (w *ObjListWatcher) getID(obj metav1.Object) string {
	return fmt.Sprintf("%s/%s/%s", w.ResourceKind, obj.GetNamespace(), obj.GetName())
}
