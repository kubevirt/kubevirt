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

package executor

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/time/rate"

	"github.com/google/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	objUtil "kubevirt.io/kubevirt/tools/perfscale-load-generator/object"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/watcher"
)

// objTypes is used to clean up the experiment to delete the objects
var objTypes []string

// Executor contains the information required to execute a job
type Executor struct {
	Start  time.Time
	End    time.Time
	Config config.TestConfig
	UUID   string
}

// NewExecutor Returns a executor
func NewExecutor(conf config.TestConfig) *Executor {
	uid, _ := uuid.NewUUID()
	ex := &Executor{
		Config: conf,
		UUID:   uid.String(),
	}
	ex.safeExit()
	return ex
}

func (e Executor) Run() {
	e.Start = time.Now().UTC()
	for _, workload := range e.Config.Workloads {
		// limits the number of object creationg throughput
		workloadLimiter := rate.NewLimiter(rate.Limit(workload.QPS), workload.Burst)

		for iteration := 1; iteration <= workload.IterationCount; iteration++ {
			log.Log.V(2).Infof("Starting iteration %d", iteration)
			objWatchers := []*watcher.ObjListWatcher{}
			objTypes = []string{}

			for _, obj := range workload.Objects {
				var err error
				var replicas []*unstructured.Unstructured
				if replicas, err = e.createObjectReplicaSpec(obj, iteration, workload.NamespacedIterations); err != nil {
					e.cleanUp()
					panic(fmt.Errorf("unexpected error creating replicas: %v", err))
				}

				objType := objUtil.GetObjectResource(replicas[0])
				objTypes = append(objTypes, objType)
				objWatcher := watcher.NewObjListWatcher(
					e.Config.Global.Client,
					objType,
					obj.Replicas,
					*e.getListOpts())
				objWatcher.Run()
				objWatchers = append(objWatchers, objWatcher)

				if err := e.createObjectReplicas(replicas, workloadLimiter); err != nil {
					panic(fmt.Errorf("%s", err))
				}
				log.Log.V(2).Infof("Iteration %d, created %d %s", iteration, len(replicas), objType)
			}

			if workload.IterationCreationWait {
				for _, objWatcher := range objWatchers {
					log.Log.V(2).Infof("Iteration %d, waiting all %s be running", iteration, objWatcher.ResourceKind)
					if err := objWatcher.WaitRunning(workload.MaxWaitTimeout.Duration); err != nil {
						e.cleanUp()
						panic(fmt.Errorf("unexpected error when waiting %s: %v", objWatcher.ResourceKind, err))
					}
				}
			}

			if workload.IterationCleanup {
				e.cleanUp()
				if workload.IterationDeletionWait {
					for _, objType := range objTypes {
						log.Log.V(2).Infof("Iteration %d, waiting all %s be deleted", iteration, objType)
						deletionWait(objWatchers, workload.MaxWaitTimeout.Duration)
					}
					log.Log.V(2).Infof("Iteration %d, waiting all namespaces be deleted", iteration)
					objUtil.WaitForDeleteNamespaces(e.Config.Global.Client, workload.MaxWaitTimeout.Duration, *e.getListOpts())
				}
			}

			stopAllWatchers(objWatchers)

			if workload.IterationInterval.Duration > 0 {
				log.Log.V(2).Infof("Sleeping for %s between interations", workload.IterationInterval.Duration)
				time.Sleep(workload.IterationInterval.Duration)
			}
		}

		if workload.WaitWhenFinished.Duration > 0 {
			log.Log.V(2).Infof("Sleeping for %s after the workload execution", workload.WaitWhenFinished.Duration)
			time.Sleep(workload.WaitWhenFinished.Duration)
		}
	}
	e.End = time.Now().UTC()
	log.Log.V(2).Infof("Benchmark Startup Time: %v", e.Start)
	log.Log.V(2).Infof("Benchmark End Time: %v", e.End)
}

// createObjectReplicas returns the last created object to provies information for wait and delete the objects
func (e Executor) createObjectReplicaSpec(obj *config.ObjectSpec, iteration int, namespacedIterations bool) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured
	for r := 1; r <= obj.Replicas; r++ {
		var newObject *unstructured.Unstructured
		var err error

		templateData := objUtil.GenerateObjectTemplateData(obj, iteration, r, namespacedIterations)
		if newObject, err = objUtil.RendereObject(templateData, obj.ObjectTemplate); err != nil {
			return nil, fmt.Errorf("error rendering obj: %v", err)
		}
		objUtil.AddLabels(newObject, e.UUID)
		objects = append(objects, newObject)
	}
	return objects, nil
}

func (e Executor) createObjectReplicas(objects []*unstructured.Unstructured, limiter *rate.Limiter) error {
	for _, obj := range objects {
		objUtil.CreateNamespace(e.Config.Global.Client, obj.GetNamespace(), config.WorkloadLabel, e.UUID)

		if _, err := objUtil.CreateObject(e.Config.Global.Client, obj); err != nil {
			return fmt.Errorf("error creating obj %s: %v", obj.GroupVersionKind().Kind, err)
		}

		// throttle the obj creation throughput
		limiter.Wait(context.TODO())
	}
	return nil
}

func (e Executor) getListOpts() *metav1.ListOptions {
	listOpts := metav1.ListOptions{}
	listOpts.LabelSelector = fmt.Sprintf("%s=%s", config.WorkloadLabel, e.UUID)
	return &listOpts
}

func deletionWait(objWatchers []*watcher.ObjListWatcher, timeout time.Duration) {
	for _, objWatcher := range objWatchers {
		if err := objWatcher.WaitDeletion(timeout); err != nil {
			panic(fmt.Errorf("unexpected error when waiting obj: %v", err))
		}
	}
}

func stopAllWatchers(objWatchers []*watcher.ObjListWatcher) {
	log.Log.V(2).Infof("Stopping all watchers")
	for _, objWatcher := range objWatchers {
		objWatcher.Stop()
	}
}

func (e Executor) cleanUp() {
	for _, objType := range objTypes {
		log.Log.V(2).Infof("Clean up, deleting all created %s", objType)
		objUtil.DeleteAllObjectsInNamespaces(e.Config.Global.Client, objType, e.getListOpts())
	}
	log.Log.V(2).Infof("Clean up, deleting all created namespaces")
	objUtil.CleanupNamespaces(e.Config.Global.Client, 30*time.Minute, e.getListOpts())
}

func (e Executor) safeExit() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Log.V(2).Errorf("unexpected crtl-c exit")
		e.cleanUp()
		os.Exit(1)
	}()
}
