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

	kvv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/controller"

	"k8s.io/client-go/tools/cache"

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
}

func (w *ObjListWatcher) Stop() {
	close(w.stopChannel)
}

func (w *ObjListWatcher) handleUpdate(obj interface{}) {
	switch w.ResourceKind {
	case objUtils.VMIResource:
		vmi, ok := obj.(*kvv1.VirtualMachineInstance)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachineInstance obj: %v", w.ResourceKind)
		}
		if vmi.Status.Phase == kvv1.Running {
			w.addObj(string(vmi.GetUID()))
		}

	case objUtils.VMResource:
		vm, ok := obj.(*kvv1.VirtualMachine)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachine obj: %v", w.ResourceKind)
		}
		if vm.Status.Ready {
			w.addObj(string(vm.GetUID()))
		}

	case objUtils.VMIReplicaSetResource:
		replicaSet, ok := obj.(*kvv1.VirtualMachineInstanceReplicaSet)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachineInstanceReplicaSet obj: %v", w.ResourceKind)
		}
		w.addObj(string(replicaSet.GetUID()))

	default:
		log.Log.V(2).Errorf("Watcher does not support object type %s", w.ResourceKind)
		return
	}
}

func (w *ObjListWatcher) handleDeleted(obj interface{}) {
	switch w.ResourceKind {
	case objUtils.VMIResource:
		vmi, ok := obj.(*kvv1.VirtualMachineInstance)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachineInstance obj: %v", w.ResourceKind)
		}
		w.deleteObj(string(vmi.GetUID()))

	case objUtils.VMResource:
		vm, ok := obj.(*kvv1.VirtualMachine)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachine obj: %v", w.ResourceKind)
		}
		w.deleteObj(string(vm.GetUID()))

	case objUtils.VMIReplicaSetResource:
		replicaSet, ok := obj.(*kvv1.VirtualMachineInstanceReplicaSet)
		if !ok {
			log.Log.V(2).Errorf("Could not convert VirtualMachineInstanceReplicaSet obj: %v", w.ResourceKind)
		}
		w.deleteObj(string(replicaSet.GetUID()))

	default:
		log.Log.V(2).Errorf("Watcher does not support object type %s", w.ResourceKind)
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
	if w.getRunningCount() == 0 {
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
			log.Log.V(6).Infof("Still %d Running %v", count, w.ResourceKind)
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
