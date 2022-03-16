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
 * Copyright 2018 Red Hat, Inc.
 *
 */

package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/components"
	"kubevirt.io/kubevirt/pkg/virt-operator/resource/generate/rbac"
	"kubevirt.io/kubevirt/tools/util"
)

const (
	featureGatesPlaceholder  = "FeatureGatesPlaceholder"
	infraReplicasPlaceholder = 255
)

func generateKubeVirtCR(namespace *string, imagePullPolicy v1.PullPolicy, featureGatesFlag *string, infraReplicasFlag *string) {
	var featureGates string
	if strings.HasPrefix(*featureGatesFlag, "{{") {
		featureGates = featureGatesPlaceholder
	} else {
		featureGates = *featureGatesFlag
	}
	var infraReplicas uint8
	if strings.HasPrefix(*infraReplicasFlag, "{{") {
		infraReplicas = infraReplicasPlaceholder
	} else {
		val, err := strconv.ParseUint(*infraReplicasFlag, 10, 8)
		if err != nil {
			panic(err)
		}
		infraReplicas = uint8(val)
	}
	var buf bytes.Buffer
	util.MarshallObject(components.NewKubeVirtCR(*namespace, imagePullPolicy, featureGates, infraReplicas), &buf)
	cr := buf.String()
	// When creating a template, we need to add code to iterate over the feature-gates slice variable.
	// util.MarshallObject(), called above, uses yaml.Marshall(), which can only generate valid yaml.
	// However, the template syntax to iterate over an array variable is not valid yaml.
	// Since most templated values are strings, this is not usually a problem, as "{{.Variable}}" is a valid string.
	// At this point (again when creating a template), the value of featureGates looks like:
	//      featureGates:
	//      - FeatureGatesPlaceholder
	// however we want to treat the variable (".FeatureGates" here) as a slice and iterate over it (with a special case for empty list):
	//      featureGates:{{if .FeatureGates}}
	//      {{- range .FeatureGates}}
	//      - {{.}}
	//      {{- end}}{{else}} []{{end}}
	// The replace call below will transform the former into the latter, keeping the intendation ($1)
	if strings.HasPrefix(*featureGatesFlag, "{{") {
		featureGatesVar := strings.TrimPrefix(*featureGatesFlag, "{{")
		featureGatesVar = strings.TrimSuffix(featureGatesVar, "}}")
		re := regexp.MustCompile(`(?m)featureGates:\n([ \t]+)- ` + featureGatesPlaceholder)
		cr = re.ReplaceAllString(cr, `featureGates:{{if `+featureGatesVar+`}}
$1{{- range `+featureGatesVar+`}}
$1- {{.}}
$1{{- end}}{{else}} []{{end}}`)
	}
	// Same idea as above, but simpler. infra.replicas is a uint8.
	// However, when creating a template, we want its value to be something like "{{.InfraReplicas}}", which is not a uint8.
	// Therefore, the value was substituted for a placeholder above (255). Replacing with the templated value now.
	if strings.HasPrefix(*infraReplicasFlag, "{{") {
		cr = strings.Replace(cr, fmt.Sprintf("replicas: %d", infraReplicasPlaceholder), "replicas: "+*infraReplicasFlag, 1)
	}

	fmt.Print(cr)
}

func main() {
	resourceType := flag.String("type", "", "Type of resource to generate. kv | kv-cr | operator-rbac | priorityclass")
	namespace := flag.String("namespace", "kube-system", "Namespace to use.")
	pullPolicy := flag.String("pullPolicy", "IfNotPresent", "ImagePullPolicy to use.")
	featureGates := flag.String("featureGates", "", "Feature gates to enable.")
	infraReplicas := flag.String("infraReplicas", "2", "Number of replicas for virt-controller and virt-api")

	flag.Parse()

	imagePullPolicy := v1.PullPolicy(*pullPolicy)

	switch *resourceType {
	case "kv":
		kv, err := components.NewKubeVirtCrd()
		if err != nil {
			panic(fmt.Errorf("This should not happen, %v", err))
		}
		util.MarshallObject(kv, os.Stdout)
	case "kv-cr":
		generateKubeVirtCR(namespace, imagePullPolicy, featureGates, infraReplicas)
	case "operator-rbac":
		all := rbac.GetAllOperator(*namespace)
		for _, r := range all {
			util.MarshallObject(r, os.Stdout)
		}
	case "priorityclass":
		priorityClass := components.NewKubeVirtPriorityClassCR()
		util.MarshallObject(priorityClass, os.Stdout)
	default:
		panic(fmt.Errorf("unknown resource type %s", *resourceType))
	}
}
