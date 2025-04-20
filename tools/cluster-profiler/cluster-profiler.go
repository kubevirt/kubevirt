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
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	flag "github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"
)

const errorClusterProfilerFmt = "Error cluster profiler %s: %v"

const (
	PROFILER_START = "start"
	PROFILER_STOP  = "stop"
	PROFILER_DUMP  = "dump"
)

const (
	defaultOutputDir    = "cluster-profiler-results"
	defaultDumpPageSize = 10
)

func prepareDir(dir string, reuseOutputDir bool) error {
	if _, err := os.Stat(dir); err == nil {
		if !reuseOutputDir {
			oldResultsDstDir := fmt.Sprintf("%s-old-%s", dir, rand.String(4))
			log.Printf("Moving already existing %q => %q\n", dir, oldResultsDstDir)
			if err := os.Rename(dir, oldResultsDstDir); err != nil {
				return err
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.MkdirAll(dir, 0744)
}

func writeResultsToDisk(dir string, results *v1.ClusterProfilerResults) error {
	for key, val := range results.ComponentResults {
		componentDir := filepath.Join(dir, key)

		err := os.Mkdir(componentDir, 0744)
		if err != nil {
			return err
		}

		for pprofKey, pprofBytes := range val.PprofData {
			filePath := filepath.Join(componentDir, pprofKey)
			err = os.WriteFile(filePath, pprofBytes, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	var (
		cmd           string
		outputDir     string
		continueToken string

		labelSelector  string
		pageSize       int
		reuseOutputDir bool
	)

	clientConfig := kubecli.DefaultClientConfig(flag.CommandLine)

	flag.StringVar(&cmd, "cmd", "", "The profiler command, start|stop|dump")
	flag.StringVar(&outputDir, "output-dir", defaultOutputDir, "The directory to store the profiler results in.")
	flag.IntVar(&pageSize, "page-size", defaultDumpPageSize, "Page size used for fetching profile results. Works only with dump command")
	flag.StringVar(&continueToken, "continue", "", "Token to be used to continue fetching profiles")
	flag.BoolVar(&reuseOutputDir, "reuse-output-dir", false, "Use output-dir even if exists and is not empty")

	// NOTE: To profile specific kubevirt component (for example virt-api) use `kubevirt.io=virt-operator` label selector.
	flag.StringVar(&labelSelector, "l", "", "Label selector for limiting pods to fetch the profiler results from. Works only with 'dump' command. kubectl LIST label selector format expected")

	flag.Parse()

	if cmd != PROFILER_DUMP && len(labelSelector) > 0 {
		log.Fatalf("labelSelector can only be used with 'dump' command")
	}

	if pageSize <= 0 {
		log.Fatalf("page-size has to be larger than 0; got %d", pageSize)
	}
	if len(labelSelector) > 0 {
		if _, err := labels.Parse(labelSelector); err != nil {
			log.Fatalf("failed to parse label selector: %v", err)
		}
	}

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(clientConfig)
	if err != nil {
		log.Fatalf("Cannot obtain KubeVirt client: %v", err)
	}

	switch cmd {
	case PROFILER_START:
		err := virtClient.ClusterProfiler().Start()
		if err != nil {
			log.Fatalf(errorClusterProfilerFmt, cmd, err)
		}
		log.Print("SUCCESS: started cpu profiling KubeVirt control plane")
	case PROFILER_STOP:
		err := virtClient.ClusterProfiler().Stop()
		if err != nil {
			log.Fatalf(errorClusterProfilerFmt, cmd, err)
		}
		log.Print("SUCCESS: stopped cpu profiling KubeVirt control plane")
	case PROFILER_DUMP:
		err := fetchAndSaveClusterProfilerResults(virtClient, pageSize, labelSelector, outputDir, continueToken, reuseOutputDir)
		if err != nil {
			log.Fatalf(errorClusterProfilerFmt, cmd, err)
		}
	default:
		if cmd == "" {
			log.Fatalf("--cmd must be set. Valid values are [start|stop|dump]")
		} else {
			log.Fatalf("unknown profiler --cmd value, [%s]. must be of type start|stop|dump", cmd)
		}
	}
}

func fetchAndSaveClusterProfilerResults(c kubecli.KubevirtClient, pageSize int, labelSelector, outputDir, continueToken string, reuseOutputDir bool) error {
	if err := prepareDir(outputDir, reuseOutputDir); err != nil {
		return err
	}

	var (
		req = &v1.ClusterProfilerRequest{
			PageSize:      int64(pageSize),
			LabelSelector: labelSelector,
			Continue:      continueToken,
		}

		result            *v1.ClusterProfilerResults
		lastContinueToken string
		counter           int
		err               error
	)

	for {
		fmt.Printf("\rFetching in progress. Downloaded so far: %d ", counter)
		result, err = c.ClusterProfiler().Dump(req)
		if err != nil {
			break
		}

		if len(result.ComponentResults) == 0 {
			break
		}

		err = writeResultsToDisk(outputDir, result)
		if err != nil {
			break
		}

		counter += len(result.ComponentResults)
		if result.Continue == "" {
			break
		}

		lastContinueToken = result.Continue
		req.Continue = result.Continue
	}

	if err == nil {
		log.Printf("\rSUCCESS: Dumped PProf %d results for KubeVirt control plane to [%s]\n", counter, outputDir)
		return nil
	}
	return fmt.Errorf("%v\nContinue token from last successful profiles fetch: %q\n", err, lastContinueToken)
}
