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
 * Copyright 2021 Nvidia
 *
 */

package steadyState

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

// SteadyStateLoadGenerator generates VMI workloads
type SteadyStateLoadGenerator struct {
	Workload   *config.Workload
	virtClient kubecli.KubevirtClient
	UUID       string
	objType    string
	firstLoop  bool
	churn      int
}

// NewSteadyStateLoadGenerator
func NewSteadyStateLoadGenerator(virtClient kubecli.KubevirtClient, workload *config.Workload) *SteadyStateLoadGenerator {
	return &SteadyStateLoadGenerator{
		virtClient: virtClient,
		Workload:   workload,
		firstLoop:  true,
		churn:      workload.Churn,
	}
}

// TODO: build helper function for createBurst so steady-state can call create burst or other types
//       of create during its create cycle
func (b *SteadyStateLoadGenerator) CreateWorkload() {
	var wg sync.WaitGroup

	obj := b.Workload.Object
	count := b.Workload.Count
	if !b.firstLoop {
		count = b.churn
	} else {
		uid, _ := uuid.NewUUID()
		b.UUID = uid.String()
	}

	for replica := 1; replica <= count; replica++ {
		log.Log.V(2).Infof("Replica %d of %d", replica, b.Workload.Count)
		templateData := objUtil.GenerateObjectTemplateData(obj, replica)

		newObject, err := objUtil.RenderObject(templateData, obj.ObjectTemplate)
		if err != nil {
			log.Log.Errorf("error rendering obj: %v", err)
		}
		config.AddLabels(newObject, b.UUID)
		if b.objType == "" {
			b.objType = objUtil.GetObjectResource(newObject)
		}

		if _, err := objUtil.CreateObject(b.virtClient, newObject); err != nil {
			log.Log.Errorf("error creating obj %s: %v", newObject.GroupVersionKind().Kind, err)
		}

		wg.Add(1)
		go func(newObject *unstructured.Unstructured) {
			defer wg.Done()
			b.Watch(newObject, false)
			log.Log.Infof("obj %s is available", newObject.GroupVersionKind().Kind)
		}(newObject)
	}
	wg.Wait()
}

func (b *SteadyStateLoadGenerator) DeleteWorkload() {
	obj := b.Workload.Object
	count := b.Workload.Count

	if flags.Delete {
		getObject, objType := objUtil.FindObject(b.virtClient, obj, count)
		b.objType = objType
		if getObject != nil {
			labels := getObject.GetLabels()
			jobUUID := labels[config.WorkloadLabel]
			log.Log.V(2).Infof("Deleting all workloads for job %s", jobUUID)
			objUtil.DeleteAllObjectsInNamespaces(b.virtClient, objType, config.GetListOpts(config.WorkloadLabel, jobUUID))
			b.Watch(getObject, flags.Delete)
		}
		log.Log.V(2).Infof("All workloads for job have been deleted")
		return
	}
	if b.firstLoop {
		b.firstLoop = false
	}
	count = b.churn
	var wg sync.WaitGroup
	for replica := 1; replica <= count; replica++ {
		templateData := objUtil.GenerateObjectTemplateData(obj, replica)
		newObject, err := objUtil.RenderObject(templateData, obj.ObjectTemplate)
		if err != nil {
			log.Log.Errorf("error rendering obj: %v", err)
		}

		log.Log.V(3).Infof("Deleting obj %s", newObject.GetName())
		objUtil.DeleteObject(b.virtClient, *newObject, b.objType, 0)

		wg.Add(1)
		go func(newObject *unstructured.Unstructured) {
			defer wg.Done()
			b.Watch(newObject, true)
			log.Log.Infof("obj %s was deleted", newObject.GroupVersionKind().Kind)
		}(newObject)
	}
	wg.Wait()
}

func (b *SteadyStateLoadGenerator) Watch(obj *unstructured.Unstructured, delete bool) {
	count := b.Workload.Count
	// TODO: break up firstLoop logic so it's clearer when we're creating churn and when we're deleting/creating
	if !b.firstLoop {
		if delete {
			// We expect b.churn fewer objects
			count = count - b.churn
		} else {
			// We expect b.churn objects to be created
			count = b.churn
		}
	}
	objWatcher := watcher.NewObjListWatcher(
		b.virtClient,
		b.objType,
		count,
		*config.GetListOpts(config.WorkloadLabel, b.UUID))
	objWatcher.Run()
	if delete {
		log.Log.Infof("Wait for obj(s) %s to be deleted", b.objType)
		objWatcher.WaitDeletion(b.Workload.Timeout.Duration)
	} else {
		log.Log.Infof("Wait for obj(s) %s to be available", b.objType)
		objWatcher.WaitRunning(b.Workload.Timeout.Duration)
	}
	objWatcher.Stop()
}

func (b *SteadyStateLoadGenerator) Wait() {
	time.Sleep(20 * time.Second)
}
