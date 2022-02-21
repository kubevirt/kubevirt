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

package main

import (
	"time"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/api"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/burst"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/config"
	"kubevirt.io/kubevirt/tools/perfscale-load-generator/flags"
	steadyState "kubevirt.io/kubevirt/tools/perfscale-load-generator/steady-state"
)

func main() {
	log.Log.SetVerbosityLevel(flags.Verbosity)

	log.Log.V(1).Infof("Running Load Generator")

	workload := config.NewWorkload(flags.WorkloadConfigFile)
	client := config.NewKubevirtClient()
	if workload.Type == "" {
		workload.Type = config.Type
	}
	// Minimum 30s timeout
	if workload.Timeout.Duration <= time.Duration(30*time.Second) {
		workload.Timeout.Duration = config.Timeout
	}

	var lg api.LoadGenerator
	if workload.Type == "burst" {
		lg = burst.NewBurstLoadGenerator(client, workload)
	} else if workload.Type == "steady-state" {
		lg = steadyState.NewSteadyStateLoadGenerator(client, workload)
	} else {
		log.Log.V(1).Errorf("Load Generator doesn't have type %s", workload.Type)
		return
	}

	if flags.Delete {
		lg.DeleteWorkload()
	} else {
		timeout := time.After(workload.Timeout.Duration)
		working := true
		for working {
			select {
			case <-timeout:
				log.Log.V(1).Infof("Load Generator duration has timed out")
				working = false
			default:
			}

			log.Log.V(1).Infof("Load Generator CreateWorkload")
			lg.CreateWorkload()
			// Burst workload should only be run once and only call Create
			if workload.Type == "burst" {
				return
			}
			log.Log.V(1).Infof("Load Generator Wait")
			lg.Wait()
			log.Log.V(1).Infof("Load Generator DeleteWorkload")
			lg.DeleteWorkload()
			log.Log.V(1).Infof("Load Generator Wait")
			lg.Wait()
		}
	}
	return
}
