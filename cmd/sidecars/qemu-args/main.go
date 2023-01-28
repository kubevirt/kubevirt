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

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"

	"github.com/spf13/pflag"

	vmSchema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	qemuCommandLineArgsAnnotation = "libvirt.vm.kubevirt.io/qemuArgs"
	NS_SCHEMA_QEMU_1_0            = "http://libvirt.org/schemas/domain/qemu/1.0"
)

var logger *log.Logger

func mergeQEMUCmd(current, annotations *api.Commandline) api.Commandline {
	// Use map for simplicity
	mapEnv := make(map[string]string)
	mapArg := make(map[string]int)

	// Populate map with current
	for _, env := range current.QEMUEnv {
		mapEnv[env.Name] = env.Value
	}
	for index, arg := range current.QEMUArg {
		mapArg[arg.Value] = index
	}

	// Here we might overwrite existing enviroment values.
	for _, env := range annotations.QEMUEnv {
		if value, exists := mapEnv[env.Name]; exists && value != env.Value {
			logger.Printf("Overwriting enviroment variable '%s': From '%s' to '%s'",
				env.Name, value, env.Value)
		}
		mapEnv[env.Name] = env.Value
	}

	// Here we might warn if arg already exists.
	for _, arg := range annotations.QEMUArg {
		if _, exists := mapArg[arg.Value]; exists {
			logger.Printf("Argument was already set: %s", arg.Value)
		} else {
			mapArg[arg.Value] = len(mapArg)
		}
	}

	// Converge
	ret := api.Commandline{}
	for name, value := range mapEnv {
		ret.QEMUEnv = append(ret.QEMUEnv, api.Env{Name: name, Value: value})
	}

	// Keep the arg order, in case it matters
	ret.QEMUArg = make([]api.Arg, len(mapArg))
	for value, index := range mapArg {
		ret.QEMUArg[index].Value = value
	}
	return ret
}

func onDefineDomain(vmiJSON, domainXML []byte) ([]byte, error) {
	vmiSpec := vmSchema.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		logger.Printf("Failed to unmarshal given VMI spec: %s due %s", vmiJSON, err)
		return nil, err
	}

	// Same struct as api.Commandline but with a nicer name for JSON. User can set env's name and
	// values, or arg's multiple values in a json format. See:
	// https://libvirt.org/drvqemu.html#pass-through-of-arbitrary-qemu-commands
	qemuCmd := struct {
		QEMUEnv []api.Env `json:"env,omitempty"`
		QEMUArg []api.Arg `json:"arg,omitempty"`
	}{}

	annotations := vmiSpec.GetAnnotations()
	if qemuArgs, found := annotations[qemuCommandLineArgsAnnotation]; !found {
		logger.Println("No command line arguments provided. Returning original domain spec.")
		return domainXML, nil
	} else if err := json.Unmarshal([]byte(qemuArgs), &qemuCmd); err != nil {
		logger.Printf("Failed to unmarshal qemu args: %s due %s", qemuArgs, err)
		return domainXML, err
	}

	domainSpec := api.DomainSpec{}
	if err := xml.Unmarshal(domainXML, &domainSpec); err != nil {
		logger.Printf("Failed to unmarshal given domain spec: %s due %s", domainXML, err)
		return nil, err
	}

	// Merge sidecar annotations with existing QEMUCmd.
	// Should Warn if overwrite an existing argument or environment variable
	apiCmd := api.Commandline(qemuCmd)
	if domainSpec.QEMUCmd != nil {
		apiCmd = mergeQEMUCmd(domainSpec.QEMUCmd, &apiCmd)
	}
	domainSpec.QEMUCmd = &apiCmd

	// Set namespace in order to be accepted by libvirt
	domainSpec.XmlNS = NS_SCHEMA_QEMU_1_0

	if newDomainXML, err := xml.Marshal(domainSpec); err != nil {
		logger.Printf("Failed to marshal updated domain spec: %+v due %s", domainSpec, err)
		return nil, err
	} else {
		logger.Println("Successfully updated original domain spec with requested command line arguments")
		return newDomainXML, nil
	}
}

func main() {
	var vmiJSON, domainXML string
	pflag.StringVar(&vmiJSON, "vmi", "", "VMI to change in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "Domain spec in XML format")
	pflag.Parse()

	logger = log.New(os.Stderr, "qemu-args", log.Ldate)
	if vmiJSON == "" || domainXML == "" {
		logger.Printf("Bad input vmi=%d, domain=%d", len(vmiJSON), len(domainXML))
		os.Exit(1)
	}

	result, err := onDefineDomain([]byte(vmiJSON), []byte(domainXML))
	if err != nil {
		panic(err)
	}
	fmt.Println(string(result))
}
