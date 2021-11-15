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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"

	v1 "kubevirt.io/client-go/apis/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const (
	PROFILER_START = "start"
	PROFILER_STOP  = "stop"
	PROFILER_DUMP  = "dump"
)

const (
	defaultOutputDir = "cluster-profiler-results"
)

func writeResultsToDisk(dir string, results *v1.ClusterProfilerResults) error {
	os.RemoveAll(dir)
	err := os.Mkdir(dir, 0744)
	if err != nil {
		return err
	}

	for key, val := range results.ComponentResults {
		componentDir := filepath.Join(dir, key)

		err := os.Mkdir(componentDir, 0744)
		if err != nil {
			return err
		}

		for pprofKey, pprofBytes := range val.PprofData {
			filePath := filepath.Join(componentDir, pprofKey)
			err = ioutil.WriteFile(filePath, pprofBytes, 0644)
			if err != nil {
				return err
			}
		}
	}

	log.Printf("SUCCESS: Dumped PProf results for KubeVirt control plane to [%s]\n", dir)

	return nil
}

func main() {

	var cmd string
	var outputDir string

	clientConfig := kubecli.DefaultClientConfig(flag.CommandLine)

	flag.StringVar(&cmd, "cmd", "", "The profiler command, start|stop|dump")
	flag.StringVar(&outputDir, "output-dir", defaultOutputDir, "The directory to store the profiler results in.")
	flag.Parse()

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		log.Fatalf("Cannot obtain KubeVirt client: %v", err)
	}

	switch cmd {
	case PROFILER_START:
		err := virtClient.ClusterProfiler().Start()
		if err != nil {
			log.Fatalf("Error cluster profiler %s: %v", cmd, err)
		}
		log.Print("SUCCESS: started cpu profiling KubeVirt control plane")
	case PROFILER_STOP:
		err := virtClient.ClusterProfiler().Stop()
		if err != nil {
			log.Fatalf("Error cluster profiler %s: %v", cmd, err)
		}
		log.Print("SUCCESS: stopped cpu profiling KubeVirt control plane")
	case PROFILER_DUMP:
		results, err := virtClient.ClusterProfiler().Dump()
		if err != nil {
			log.Fatalf("Error cluster profiler %s: %v", cmd, err)
		}

		err = writeResultsToDisk(outputDir, results)
		if err != nil {
			panic(err)
		}
	default:
		if cmd == "" {
			log.Fatalf("--cmd must be set. Valid values are [start|stop|dump]")
		} else {
			log.Fatalf("unknown profiler --cmd value, [%s]. must be of type start|stop|dump", cmd)
		}

	}

}
