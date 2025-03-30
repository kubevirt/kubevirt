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
 * Copyright 2022 Nvidia
 *
 */

package steadyState

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

type SteadyStateLoadGenerator struct {
	Done <-chan time.Time
	UUID string
}

type SteadyStateJob struct {
	Workload      *config.Workload
	virtClient    kubecli.KubevirtClient
	UUID          string
	objType       string
	firstLoop     bool
	churn         int
	startedTime   time.Time
	minChurnSleep *config.Duration
	objIdx        int
	watchers      map[string]*watcher.ObjListWatcher
}

// NewSteadyStateJob
func newSteadyStateJob(virtClient kubecli.KubevirtClient, workload *config.Workload, uuid string) *SteadyStateJob {
	// minChurnSleep is an optinal config
	if workload.MinChurnSleep == nil {
		workload.MinChurnSleep = &config.Duration{Duration: config.DefaultMinSleepChurn}
	}
	if workload.MinChurnSleep.Duration < time.Microsecond {
		workload.MinChurnSleep = &config.Duration{Duration: config.DefaultMinSleepChurn}
	}
	return &SteadyStateJob{
		virtClient:    virtClient,
		Workload:      workload,
		firstLoop:     true,
		churn:         workload.Churn,
		startedTime:   time.Now(),
		minChurnSleep: workload.MinChurnSleep,
		UUID:          uuid,
		objIdx:        0,
		watchers:      map[string]*watcher.ObjListWatcher{},
	}
}

func (b *SteadyStateLoadGenerator) Delete(virtClient kubecli.KubevirtClient, workload *config.Workload) {
	ss := newSteadyStateJob(virtClient, workload, b.UUID)
	ss.DeleteWorkloads(ss.Workload.Count)
	ss.DeleteNamespaces()
	ss.stopAllWatchers()
	return
}

func (b *SteadyStateLoadGenerator) Run(virtClient kubecli.KubevirtClient, workload *config.Workload) {
	ss := newSteadyStateJob(virtClient, workload, b.UUID)
	// Ramp up phase, creating all objects, waiting before starting the steady state phase
	log.Log.V(1).Infof("Starting the Ramp Up Phase")
	ss.CreateWorkloads(ss.Workload.Count)

	// After the Ramp up phase, the steady-state phase will start, creating a churn of object in the cluster,
	// by deleting X objects and recreating them, until the test timeout.
	log.Log.V(1).Infof("Starting the Steady State Phase")
	for {
		select {
		case <-b.Done:
			// The steady state phase finishes when the test times out, and then the ramp down phase deleting all objs
			log.Log.V(1).Infof("Starting the Ramp Down Phase")
			ss.DeleteWorkloads(ss.Workload.Count)
			ss.DeleteNamespaces()
			ss.stopAllWatchers()
			return
		default:
			// Before deleting objetcs we wait Y seconds, which represents the object lifetime.
			ss.Wait()
			ss.DeleteWorkloads(ss.Workload.Churn)
			// Ramp up phase, creating all objects, waiting, and then deleting objects
			ss.CreateWorkloads(ss.Workload.Churn)
		}
	}
}

func (b *SteadyStateJob) CreateWorkloads(replicas int) {
	log.Log.V(1).Infof("SteadyState Load Generator CreateWorkloads")

	// The watcher must be created before to be able to watch all events related to the objs.
	// This is important because before creating each obj we need to create a obj watcher for each obj type.
	objSpec := b.Workload.Object
	objSample := renderObjSpecTemplate(objSpec, b.UUID)
	b.createWatcherIfNotExist(objSample)
	objUtil.CreateNamespaceIfNotExist(b.virtClient, objSample.GetNamespace(), config.WorkloadUUIDLabel, b.UUID)

	// Create all replicas
	for r := 1; r <= replicas; r++ {
		log.Log.V(2).Infof("Replica %d of %d with idx %d", r, replicas, b.objIdx)
		_, err := objUtil.CreateObjectReplica(b.virtClient, objSpec, &b.objIdx, b.UUID)
		if err != nil {
			continue
		}
	}

	// Wait all objects be Running
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Wait for obj(s) %s to be available", objType)
		objWatcher.WaitRunning(b.Workload.Timeout.Duration)
	}
}

func (b *SteadyStateJob) DeleteWorkloads(count int) {
	log.Log.V(1).Infof("SteadyState Load Generator DeleteWorkloads")
	objSpec := b.Workload.Object
	objSample := renderObjSpecTemplate(objSpec, b.UUID)
	b.createWatcherIfNotExist(objSample)

	// Delete objects that match labels up to the count
	objType := objUtil.GetObjectResource(objSample)
	labels := objSample.GetLabels()
	jobUUID := labels[config.WorkloadUUIDLabel]
	log.Log.V(2).Infof("Deleting %d objects for job %s", count, jobUUID)
	objUtil.DeleteNObjectsInNamespaces(b.virtClient, objType, config.GetListOpts(config.WorkloadUUIDLabel, jobUUID), count)

	// Wait all objects be Deleted. In the case of VMI, deleted means the succeded phase.
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Wait for obj(s) %s to be deleted", objType)
		objWatcher.WaitDeletion(b.Workload.Timeout.Duration)
	}

	log.Log.V(2).Infof("Deleted %d obj(s) for job %s", count, jobUUID)
}

func (b *SteadyStateJob) DeleteNamespaces() {
	log.Log.V(2).Infof("Clean up, deleting all created namespaces")
	objUtil.CleanupNamespaces(b.virtClient, 30*time.Minute, config.GetListOpts(config.WorkloadUUIDLabel, b.UUID))
	objUtil.WaitForDeleteNamespaces(b.virtClient, 30*time.Minute, *config.GetListOpts(config.WorkloadUUIDLabel, b.UUID))
}

func (b *SteadyStateJob) createWatcherIfNotExist(objSpec *unstructured.Unstructured) {
	objType := objUtil.GetObjectResource(objSpec)
	if _, exist := b.watchers[objType]; !exist {
		objWatcher := watcher.NewObjListWatcher(
			b.virtClient,
			objType,
			b.Workload.Count,
			*config.GetListOpts(config.WorkloadUUIDLabel, b.UUID))
		objWatcher.Run()
		b.watchers[objType] = objWatcher
	}
}

func (b *SteadyStateJob) Wait() {
	timePassed := time.Since(b.startedTime)
	remainingTime := b.Workload.Timeout.Duration - timePassed
	log.Log.V(3).Infof("Wait %f seconds before recreating %d objs. Remaining %f seconds to finish the test", b.minChurnSleep.Seconds(), b.Workload.Churn, remainingTime.Seconds())
	time.Sleep(b.minChurnSleep.Duration)
}

func (b *SteadyStateJob) stopAllWatchers() {
	for objType, objWatcher := range b.watchers {
		log.Log.Infof("Stopping obj %s watcher", objType)
		objWatcher.Stop()
	}
}

// Render objSpec template to be able to parse a sample of the object info
func renderObjSpecTemplate(objSpec *config.ObjectSpec, uuid string) *unstructured.Unstructured {
	var err error
	var obj *unstructured.Unstructured
	idx := 0
	if obj, err = objUtil.CreateObjectReplicaSpec(objSpec, &idx, uuid); err != nil {
		panic(err)
	}
	return obj
}
