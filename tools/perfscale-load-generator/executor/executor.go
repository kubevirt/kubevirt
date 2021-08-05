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
	"time"

	"golang.org/x/time/rate"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
)

const scenarioLabel = "virt-load-generator-scenario"

// Executor contains the information required to execute a job
type Executor struct {
	Start  time.Time
	End    time.Time
	Config config.ConfigSpec
	// Limits the number of workers to QPS and Burst
	limiter *rate.Limiter
}

// NewExecutorList Returns a executor
func NewExecutorList(conf config.ConfigSpec) *Executor {
	return &Executor{
		Config:  conf,
		limiter: rate.NewLimiter(rate.Limit(conf.Scenario.QPS), conf.Scenario.Burst),
	}
}

func (e Executor) Run() {
	e.Start = time.Now().UTC()
	var namespace string

	for i := 1; i <= e.Config.Scenario.IterationCount; i++ {
		log.Log.V(2).Infof("Creating %d VMIs from iteration %d", e.Config.Scenario.VMICount, i)

		if e.Config.Scenario.NamespacedIterations || i == 1 {
			namespace = fmt.Sprintf("%s-%d", e.Config.Scenario.Namespace, i)
			if err := CreateNamespaces(e.Config.ClientSet, namespace, e.Config.Scenario.UUID); err != nil {
				log.Log.V(2).Error(err.Error())
				continue
			}
			log.Log.V(2).Infof("Created namespace %s", namespace)
		}

		e.createVMI(namespace, i)

		if e.Config.Scenario.IterationVMIWait {
			log.Log.V(2).Infof("Waiting %s for VMIs in namespace %v to be in the Running phase", e.Config.Scenario.MaxWaitTimeout, namespace)
			if err := WaitForRunningVMIs(e.Config.ClientSet, namespace, e.Config.Scenario.UUID, e.Config.Scenario.VMICount, e.Config.Scenario.MaxWaitTimeout); err != nil {
				log.Log.V(2).Errorf("Failed to create VMIs: %v", err)
			} else {
				log.Log.V(2).Infof("%d VMIs were sucessfully created in namespace %v in %v", e.Config.Scenario.VMICount, namespace, time.Since(e.Start))
			}
		}

		DeleteVMIs(e.Config.ClientSet, namespace, e.Config.Scenario.UUID, e.limiter)

		if e.Config.Scenario.IterationWaitForDeletion {
			log.Log.V(2).Infof("Waiting %s for VMIs in namespace %v to be deleted", e.Config.Scenario.MaxWaitTimeout, namespace)
			if err := WaitForDeleteVMIs(e.Config.ClientSet, namespace, e.Config.Scenario.UUID); err != nil {
				log.Log.V(2).Errorf("Failed to delete VMIs: %v", err)
			} else {
				log.Log.V(2).Infof("All VMIs were deleted")
			}
		}

		if e.Config.Scenario.IterationInterval > 0 {
			log.Log.V(2).Infof("Sleeping for %v between interations", e.Config.Scenario.IterationInterval)
			time.Sleep(e.Config.Scenario.IterationInterval)
		}

		if e.Config.Scenario.IterationCleanup {
			log.Log.V(2).Infof("Clean up all created namespaces")
			WaitForDeleteVMIs(e.Config.ClientSet, namespace, e.Config.Scenario.UUID)
			if err := CleanupNamespaces(e.Config.ClientSet, e.Config.Scenario.UUID); err != nil {
				log.Log.V(2).Errorf("Error cleaning up namespaces: %v", err)
			}
		}
	}
	e.End = time.Now().UTC()
	log.Log.V(2).Infof("Scenario Startup Time: %v", e.Start)
	log.Log.V(2).Infof("Scenario End Time: %v", e.End)
}

func (e Executor) createVMI(ns string, i int) {
	for j := 1; j <= e.Config.Scenario.VMICount; j++ {
		name := fmt.Sprintf("%s-%d-%d", e.Config.Scenario.Name, i, j)
		CreateVMI(e.Config.ClientSet, name, ns, e.Config.Scenario.UUID, e.Config.Scenario.VMISpec)
		e.limiter.Wait(context.TODO())
	}
}
