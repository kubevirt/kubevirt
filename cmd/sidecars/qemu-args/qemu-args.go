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
 *
 */

package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/pflag"

	vmSchema "kubevirt.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	qemuArgsAnnotation = "qemu.vm.kubevirt.io/additionalArgs"
	qemuv1NS           = "http://libvirt.org/schemas/domain/qemu/1.0"
)

func onDefineDomain(vmiJSON, domainXML []byte) (string, error) {
	vmiSpec := vmSchema.VirtualMachineInstance{}
	if err := json.Unmarshal(vmiJSON, &vmiSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal given VMI spec: %s %s", err, string(vmiJSON))
	}

	domainSpec := api.DomainSpec{}
	if err := xml.Unmarshal(domainXML, &domainSpec); err != nil {
		return "", fmt.Errorf("failed to unmarshal given Domain spec: %s %s", err, string(domainXML))
	}

	annotations := vmiSpec.GetAnnotations()
	qemuArgsString, found := annotations[qemuArgsAnnotation]
	if !found {
		return string(domainXML), nil
	}

	if domainSpec.QEMUCmd == nil {
		domainSpec.QEMUCmd = &api.Commandline{}
	}

	qemuArgs := strings.Fields(qemuArgsString)
	for _, arg := range qemuArgs {
		domainSpec.QEMUCmd.QEMUArg = append(domainSpec.QEMUCmd.QEMUArg, api.Arg{Value: arg})
	}

	if len(domainSpec.QEMUCmd.QEMUArg) > 0 {
		domainSpec.XmlNS = qemuv1NS
	}
	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		return "", fmt.Errorf("failed to marshal new Domain spec: %s %+v", err, domainSpec)
	}

	return string(newDomainXML), nil
}

func main() {
	var vmiJSON, domainXML string
	pflag.StringVar(&vmiJSON, "vmi", "", "VMI to change in JSON format")
	pflag.StringVar(&domainXML, "domain", "", "Domain spec in XML format")
	pflag.Parse()

	logger := log.New(os.Stderr, "qemu-args", log.Ldate)
	if vmiJSON == "" || domainXML == "" {
		logger.Printf("Bad input vmi=%d, domain=%d", len(vmiJSON), len(domainXML))
		os.Exit(1)
	}

	newDomainXML, err := onDefineDomain([]byte(vmiJSON), []byte(domainXML))
	if err != nil {
		logger.Printf("onDefineDomain failed: %s", err)
		panic(err)
	}
	fmt.Println(newDomainXML)
}
