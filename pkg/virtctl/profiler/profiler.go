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

package profiler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/kubevirt/pkg/virtctl/templates"
)

const (
	PROFILER_START = "start"
	PROFILER_STOP  = "stop"
	PROFILER_DUMP  = "dump"
)

type ProfilerCommand struct {
	clientConfig clientcmd.ClientConfig
}

func newCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use:   fmt.Sprintf("profiler [start|stop|dump]"),
		Short: fmt.Sprintf("control plane profiler"),
		Args:  templates.ExactArgs("profiler", 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c := ProfilerCommand{clientConfig: clientConfig}
			return c.Run(args)
		},
	}
	cmd.SetUsageTemplate(templates.UsageTemplate())
	return cmd

}

func NewProfilerCommand(clientConfig clientcmd.ClientConfig) *cobra.Command {
	return newCommand(clientConfig)
}

func writeResultsToDisk(results *v1.ClusterProfilerResults) error {
	dir := "virtctl-cluster-profiler-results"
	os.RemoveAll(dir)
	err := os.Mkdir(dir, 0744)
	if err != nil {
		return err
	}

	globalAggregatedRequestCountFilePath := filepath.Join(dir, "global-aggregated-http-request-counts.json")
	globalAggregatedRequestCountMap := make(map[string]int)

	controllerAggregatedRequestCountFilePath := filepath.Join(dir, "virt-controller-aggregated-http-request-counts.json")
	controllerAggregatedRequestCountMap := make(map[string]int)

	handlerAggregatedRequestCountFilePath := filepath.Join(dir, "virt-handler-aggregated-http-request-counts.json")
	handlerAggregatedRequestCountMap := make(map[string]int)

	apiAggregatedRequestCountFilePath := filepath.Join(dir, "virt-api-aggregated-http-request-counts.json")
	apiAggregatedRequestCountMap := make(map[string]int)

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

		filePath := filepath.Join(componentDir, "http-request-counts.json")
		b, err := json.MarshalIndent(val.HTTPRequestCounts, "", "  ")
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(filePath, b, 0644)
		if err != nil {
			return err
		}

		for httpKey, count := range val.HTTPRequestCounts {

			curCount, ok := globalAggregatedRequestCountMap[httpKey]
			if ok {
				globalAggregatedRequestCountMap[httpKey] = curCount + count
			} else {
				globalAggregatedRequestCountMap[httpKey] = count
			}

			if strings.Contains(key, "virt-controller") {
				curCount, ok := controllerAggregatedRequestCountMap[httpKey]
				if ok {
					controllerAggregatedRequestCountMap[httpKey] = curCount + count
				} else {
					controllerAggregatedRequestCountMap[httpKey] = count
				}
			} else if strings.Contains(key, "virt-handler") {
				curCount, ok := handlerAggregatedRequestCountMap[httpKey]
				if ok {
					handlerAggregatedRequestCountMap[httpKey] = curCount + count
				} else {
					handlerAggregatedRequestCountMap[httpKey] = count
				}
			} else if strings.Contains(key, "virt-api") {
				curCount, ok := apiAggregatedRequestCountMap[httpKey]
				if ok {
					apiAggregatedRequestCountMap[httpKey] = curCount + count
				} else {
					apiAggregatedRequestCountMap[httpKey] = count
				}
			}
		}
	}

	b, err := json.MarshalIndent(globalAggregatedRequestCountMap, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(globalAggregatedRequestCountFilePath, b, 0644)
	if err != nil {
		return err
	}

	b, err = json.MarshalIndent(controllerAggregatedRequestCountMap, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(controllerAggregatedRequestCountFilePath, b, 0644)
	if err != nil {
		return err
	}

	b, err = json.MarshalIndent(handlerAggregatedRequestCountMap, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(handlerAggregatedRequestCountFilePath, b, 0644)
	if err != nil {
		return err
	}

	b, err = json.MarshalIndent(apiAggregatedRequestCountMap, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(apiAggregatedRequestCountFilePath, b, 0644)
	if err != nil {
		return err
	}

	fmt.Printf("SUCCESS: Results written to [%s]\n", dir)

	return nil
}

func (o *ProfilerCommand) Run(args []string) error {

	virtClient, err := kubecli.GetKubevirtClientFromClientConfig(o.clientConfig)
	if err != nil {
		return fmt.Errorf("Cannot obtain KubeVirt client: %v", err)
	}

	command := args[0]

	switch command {
	case PROFILER_START:
		err := virtClient.ClusterProfiler().Start()
		if err != nil {
			return fmt.Errorf("Error cluster profiler %s: %v", command, err)
		}
	case PROFILER_STOP:
		err := virtClient.ClusterProfiler().Stop()
		if err != nil {
			return fmt.Errorf("Error cluster profiler %s: %v", command, err)
		}
	case PROFILER_DUMP:
		results, err := virtClient.ClusterProfiler().Dump()
		if err != nil {
			return fmt.Errorf("Error cluster profiler %s: %v", command, err)
		}

		err = writeResultsToDisk(results)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown profiler command %s. must be of time start|stop|dump", command)

	}

	return nil
}
