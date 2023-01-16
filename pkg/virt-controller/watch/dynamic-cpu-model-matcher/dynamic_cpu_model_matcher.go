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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package watch

import (
	"fmt"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"

	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-controller/watch/topology"
)

var cpuModelToLaunchYear = map[string]int{
	"Cooperlake":                2020,
	"qemu64":                    -1,
	"qemu32":                    -1,
	"phenom":                    2007,
	"pentium3":                  1999,
	"pentium2":                  1997,
	"pentium":                   1993,
	"n270":                      2008,
	"kvm64":                     -1,
	"kvm32":                     -1,
	"coreduo":                   2006,
	"core2duo":                  2006,
	"athlon":                    1999,
	"Westmere-IBRS":             2010,
	"Westmere":                  2010,
	"Snowridge":                 2019,
	"Skylake-Server-noTSX-IBRS": 2016,
	"Skylake-Server-IBRS":       2016,
	"Skylake-Server":            2016,
	"Skylake-Client-noTSX-IBRS": 2015,
	"Skylake-Client-IBRS":       2015,
	"Skylake-Client":            2015,
	"SandyBridge-IBRS":          2011,
	"SandyBridge":               2011,
	"Penryn":                    2007,
	"Opteron_G5":                2012,
	"Opteron_G4":                2011,
	"Opteron_G3":                2009,
	"Opteron_G2":                2006,
	"Opteron_G1":                2004,
	"Nehalem-IBRS":              2008,
	"Nehalem":                   2008,
	"IvyBridge-IBRS":            2012,
	"IvyBridge":                 2012,
	"Icelake-Server-noTSX":      2019,
	"Icelake-Server":            2019,
	"Icelake-Client-noTSX":      2019,
	"Icelake-Client":            2019,
	"Haswell-noTSX-IBRS":        2013,
	"Haswell-noTSX":             2013,
	"Haswell-IBRS":              2013,
	"Haswell":                   2013,
	"EPYC-Rome":                 2019,
	"EPYC-Milan":                2021,
	"EPYC-IBPB":                 2017,
	"EPYC":                      2017,
	"Dhyana":                    2018,
	"Conroe":                    2006,
	"Cascadelake-Server-noTSX":  2019,
	"Cascadelake-Server":        2019,
	"Broadwell-noTSX-IBRS":      2014,
	"Broadwell-noTSX":           2014,
	"Broadwell-IBRS":            2014,
	"Broadwell":                 2014,
	"486":                       -1,
}

type DynamicCpuModelMatcher struct {
	nodeStore         cache.Store
	minimalLaunchYear int
}

func NewDynamicCpuModelMatcher(nodeStore cache.Store) *DynamicCpuModelMatcher {
	dcc := &DynamicCpuModelMatcher{
		minimalLaunchYear: 2007,
		nodeStore:         nodeStore,
	}
	return dcc
}

func (dcc *DynamicCpuModelMatcher) GetBestMatchModelForInitialNode(initialNodeName string, vmi *kubevirtv1.VirtualMachineInstance) (string, error) {
	initialNodeObj, exist, err := dcc.nodeStore.GetByKey(initialNodeName)
	initialNode := initialNodeObj.(*v1.Node)
	if !exist {
		return "", fmt.Errorf(fmt.Sprintf("vmiController can't determine preferred model for node:%v doesn't exist", initialNodeName))
	} else if err != nil {
		return "", fmt.Errorf(fmt.Sprintf("vmiController can't determine preferred model for node:%v got error:\n%v", initialNodeName, err.Error()))
	}
	vendorLabel := getNodeVendorLabel(initialNode)
	if vendorLabel == "" {
		return "", fmt.Errorf(fmt.Sprintf("DynamicConfigurationCalculator couldn't find vendor label for node: %v", initialNode.Name))
	}

	predicates := []topology.FilterPredicateFunc{
		withRespectToNodeVendorLabelFilerPredicate(vendorLabel),
		withRespectToVmiNodeLabelSelectorsFilerPredicate(vmi.Spec.NodeSelector),
	}
	if vmi.Spec.Affinity != nil && vmi.Spec.Affinity.NodeAffinity != nil && vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
		predicates = append(predicates, withRespectToVmiNodeAffinityFilerPredicate(vmi.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution))
	}

	nodes := topology.FilterNodesFromCache(dcc.nodeStore.List(), predicates...)

	numberOfNodes := len(nodes)
	thresholdSupport := getMedianNumber(2, numberOfNodes/3, 5)
	cpuModelCount := countCpuModelsSupportedByPotentialNodes(nodes)

	bestMatchCpuModel := getNewestSupportedModel(initialNode, cpuModelCount, thresholdSupport, dcc.minimalLaunchYear)
	if bestMatchCpuModel == "" {
		return "", fmt.Errorf(fmt.Sprintf("DynamicConfigurationCalculator couldn't find bestMatchModel for node: %v", initialNode.Name))
	}

	return bestMatchCpuModel, nil
}

func getNewestSupportedModel(node *v1.Node, cpuModelCount map[string]int, thresholdSupport int, minimalLaunchYear int) string {
	bestMatchCpuModel := ""
	bestMatchCpuModelLaunchYear := 0
	for label, _ := range node.Labels {
		cpuModel := extractCpuModelFromLabel(label)
		if cpuModel != "" && isCPUModelValid(cpuModel, cpuModelCount, thresholdSupport, minimalLaunchYear) && bestMatchCpuModelLaunchYear < cpuModelToLaunchYear[cpuModel] {
			bestMatchCpuModelLaunchYear = cpuModelToLaunchYear[cpuModel]
			bestMatchCpuModel = cpuModel
		}

	}
	return bestMatchCpuModel
}
func isCPUModelValid(cpuModel string, cpuModelCount map[string]int, thresholdSupport int, minimalLaunchYear int) bool {
	return cpuModelCount[cpuModel] >= thresholdSupport && cpuModelToLaunchYear[cpuModel] >= minimalLaunchYear
}

func getNodeVendorLabel(node *v1.Node) string {
	for label, _ := range node.Labels {
		if strings.Contains(label, kubevirtv1.CPUModelVendorLabel) {
			return label
		}
	}
	return ""
}

func countCpuModelsSupportedByPotentialNodes(nodes []*v1.Node) map[string]int {
	modelCount := make(map[string]int)
	for _, node := range nodes {
		for cpuModelLabel, _ := range node.Labels {
			cpuModel := extractCpuModelFromLabel(cpuModelLabel)
			if cpuModel != "" {
				modelCount[cpuModel]++
			}
		}
	}
	return modelCount
}

func withRespectToNodeVendorLabelFilerPredicate(vendorLabel string) topology.FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if node == nil {
			return false
		}
		otherNodeVendorLabel := getNodeVendorLabel(node)
		if otherNodeVendorLabel == "" {
			log.Log.Infof(fmt.Sprintf("DynamicConfigurationCalculator couldn't find vendor label for node: %v", node.Name))
			return false
		}
		return otherNodeVendorLabel == vendorLabel
	}
}

func withRespectToVmiNodeLabelSelectorsFilerPredicate(nodeSelectors labels.Set) topology.FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if node == nil {
			return false
		}
		return labels.SelectorFromSet(nodeSelectors).Matches(labels.Set(node.Labels))
	}
}

func withRespectToVmiNodeAffinityFilerPredicate(nodeAffinity *v1.NodeSelector) topology.FilterPredicateFunc {
	return func(node *v1.Node) bool {
		if node == nil {
			return false
		}
		nodeSelector, _ := nodeaffinity.NewNodeSelector(nodeAffinity)
		return nodeSelector.Match(node)
	}
}

func getMedianNumber(firstNum int, secondNum int, thirdNum int) int {
	if firstNum <= secondNum && secondNum <= thirdNum {
		return secondNum
	} else if secondNum <= firstNum && firstNum <= thirdNum {
		return firstNum
	} else {
		return thirdNum
	}
}

func extractCpuModelFromLabel(label string) string {
	if strings.Contains(label, kubevirtv1.CPUModelLabel) {
		return strings.ReplaceAll(label, kubevirtv1.CPUModelLabel, "")
	}
	return ""
}
