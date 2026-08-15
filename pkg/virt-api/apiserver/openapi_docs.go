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

package apiserver

import (
	"net/http"
	"strings"

	"k8s.io/kube-openapi/pkg/validation/spec"
)

type subresourceDoc struct {
	description   string
	responseCodes []int
}

func subresourceOperationDocs() map[string]subresourceDoc {
	const (
		vmi = "/namespaces/{namespace}/virtualmachineinstances/{name}"
		vm  = "/namespaces/{namespace}/virtualmachines/{name}"
	)
	badReq := []int{http.StatusBadRequest}
	badReqNotFound := []int{http.StatusBadRequest, http.StatusNotFound}
	serverErr := []int{http.StatusInternalServerError}
	cancelEvac := []int{http.StatusBadRequest, http.StatusNotFound, http.StatusInternalServerError}

	docs := map[string]subresourceDoc{
		// VirtualMachine subresources.
		vm + "/start":              {"Start a VirtualMachine object.", badReqNotFound},
		vm + "/stop":               {"Stop a VirtualMachine object.", badReqNotFound},
		vm + "/restart":            {"Restart a VirtualMachine object.", badReqNotFound},
		vm + "/migrate":            {"Migrate a running VirtualMachine to another node.", badReqNotFound},
		vm + "/addvolume":          {"Add a volume and disk to a running Virtual Machine.", badReq},
		vm + "/removevolume":       {"Removes a volume and disk from a running Virtual Machine.", badReq},
		vm + "/memorydump":         {"Dumps a VirtualMachineInstance memory.", serverErr},
		vm + "/removememorydump":   {"Remove memory dump association.", serverErr},
		vm + "/objectgraph":        {"Get graph of objects related to a Virtual Machine", nil},
		vm + "/expand-spec":        {"Get VirtualMachine object with expanded instancetype and preference.", []int{http.StatusNotFound, http.StatusInternalServerError}},
		vm + "/evacuate":           {"Cancel evacuation Virtual Machine", cancelEvac},
		vm + "/evacuate/{path}":    {"Cancel evacuation Virtual Machine", cancelEvac},
		vm + "/portforward":        {"Open a websocket connection forwarding traffic to the running VMI for the specified VirtualMachine and port.", nil},
		vm + "/portforward/{path}": {"Open a websocket connection forwarding traffic to the running VMI for the specified VirtualMachine and port.", nil},

		// VirtualMachineInstance subresources.
		vmi + "/addvolume":           {"Add a volume and disk to a running Virtual Machine Instance", badReq},
		vmi + "/removevolume":        {"Removes a volume and disk from a running Virtual Machine Instance", badReq},
		vmi + "/backup":              {"Initiate a VirtualMachineInstance backup.", badReqNotFound},
		vmi + "/redefine-checkpoint": {"Redefine a checkpoint for a VirtualMachineInstance.", badReqNotFound},
		vmi + "/freeze":              {"Freeze a VirtualMachineInstance object.", serverErr},
		vmi + "/unfreeze":            {"Unfreeze a VirtualMachineInstance object.", serverErr},
		vmi + "/pause":               {"Pause a VirtualMachineInstance object.", badReqNotFound},
		vmi + "/unpause":             {"Unpause a VirtualMachineInstance object.", badReqNotFound},
		vmi + "/reset":               {"Reset a VirtualMachineInstance object.", serverErr},
		vmi + "/softreboot":          {"Soft reboot a VirtualMachineInstance object.", serverErr},
		vmi + "/guestosinfo":         {"Get guest agent os information", nil},
		vmi + "/userlist":            {"Get list of active users via guest agent", nil},
		vmi + "/filesystemlist":      {"Get list of active filesystems on guest machine via guest agent", nil},
		vmi + "/objectgraph":         {"Get graph of objects related to a Virtual Machine Instance", nil},
		vmi + "/evacuate":            {"Cancel evacuation Virtual Machine Instance", cancelEvac},
		vmi + "/evacuate/{path}":     {"Cancel evacuation Virtual Machine Instance", cancelEvac},
		vmi + "/console":             {"Open a websocket connection to a serial console on the specified VirtualMachineInstance.", nil},
		vmi + "/vnc":                 {"Open a websocket connection to connect to VNC on the specified VirtualMachineInstance.", nil},
		vmi + "/vnc/{path}":          {"Open a websocket connection to connect to VNC on the specified VirtualMachineInstance.", nil},
		vmi + "/usbredir":            {"Open a websocket connection to connect to USB device on the specified VirtualMachineInstance.", nil},
		vmi + "/vsock":               {"Open a websocket connection forwarding traffic to the specified VirtualMachineInstance and port via VSOCK.", nil},
		vmi + "/portforward":         {"Open a websocket connection forwarding traffic to the specified VirtualMachineInstance and port.", nil},
		vmi + "/portforward/{path}":  {"Open a websocket connection forwarding traffic to the specified VirtualMachineInstance and port.", nil},
		vmi + "/sev":                 {"Handle SEV (AMD Secure Encrypted Virtualization) requests for a Virtual Machine.", nil},
		vmi + "/sev/{path}":          {"Handle SEV (AMD Secure Encrypted Virtualization) requests for a Virtual Machine.", nil},

		// Cluster-level routes already carry a description via ClusterLevelRoutes;
		// only their responses need restoring here (they lose their documented
		// success response because the go-restful route declares no body type).
		"/namespaces/{namespace}/expand-vm-spec": {"", []int{http.StatusOK, http.StatusBadRequest, http.StatusInternalServerError}},
		"/guestfs":                               {"", []int{http.StatusOK, http.StatusBadRequest}},
		"/healthz":                               {"", []int{http.StatusOK, http.StatusInternalServerError}},
	}
	return docs
}

// documentSubresourceOperations restores the original descriptions and error
// responses on the generated subresource operations.
func documentSubresourceOperations(swagger *spec.Swagger) {
	if swagger.Paths == nil {
		return
	}
	docs := subresourceOperationDocs()
	for path, item := range swagger.Paths.Paths {
		suffix, ok := subresourceSuffix(path)
		if !ok {
			continue
		}
		doc, ok := docs[suffix]
		if !ok {
			continue
		}
		for _, op := range []*spec.Operation{item.Get, item.Put, item.Post, item.Delete, item.Patch} {
			if op == nil {
				continue
			}
			if doc.description != "" {
				op.Description = doc.description
			}
			for _, code := range doc.responseCodes {
				addStringResponse(op, code)
			}
		}
	}
}

const subresourceAPIPrefix = "/apis/subresources.kubevirt.io/"

// subresourceSuffix strips the group/version prefix from a served path so it
// can be matched against subresourceOperationDocs regardless of the API
// version. It returns false for paths outside the subresources API group.
func subresourceSuffix(path string) (string, bool) {
	if !strings.HasPrefix(path, subresourceAPIPrefix) {
		return "", false
	}
	rest := path[len(subresourceAPIPrefix):]
	slash := strings.IndexByte(rest, '/')
	if slash < 0 {
		return "", false
	}
	return rest[slash:], true
}

func addStringResponse(op *spec.Operation, code int) {
	if op.Responses == nil {
		op.Responses = &spec.Responses{}
	}
	if op.Responses.StatusCodeResponses == nil {
		op.Responses.StatusCodeResponses = map[int]spec.Response{}
	}
	if _, exists := op.Responses.StatusCodeResponses[code]; exists {
		return
	}
	op.Responses.StatusCodeResponses[code] = stringResponse(code)
}
