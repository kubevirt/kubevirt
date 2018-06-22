package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"

	"google.golang.org/grpc"

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksApi "kubevirt.io/kubevirt/pkg/hooks/v1alpha"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const name = "smbios"
const version = hooksApi.Version
const priority = 50

var hookPoints = [...]string{hooksApi.OnDefineDomainHookPointName}

const baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"

type hookServer struct{}

func newHookServer() *hookServer {
	return &hookServer{}
}

func (h *hookServer) Run() {
	socketPath := hooks.HookSocketsSharedDirectory + "/" + name + ".sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksApi.RegisterHookServer(server, hookServer{})
	server.Serve(socket)
}

func (h hookServer) Info(ctx context.Context, params *hooksApi.InfoParams) (*hooksApi.InfoResult, error) {
	for _, supportedVersion := range params.GetSupportedVersions() {
		if supportedVersion == hooksApi.Version {
			return &hooksApi.InfoResult{
				Name:       name,
				Version:    version,
				Priority:   priority,
				HookPoints: hookPoints[:],
			}, nil
		}
	}
	return nil, fmt.Errorf("No supported hook API version")
}

func (h hookServer) OnDefineDomain(ctx context.Context, params *hooksApi.OnDefineDomainParams) (*hooksApi.OnDefineDomainResult, error) {
	vmJSON := params.GetVm()
	vmSpec := vmSchema.VirtualMachine{}
	err := json.Unmarshal([]byte(vmJSON), &vmSpec)
	if err != nil {
		panic(err)
	}

	annotations := vmSpec.GetAnnotations()

	if _, found := annotations[baseBoardManufacturerAnnotation]; !found {
		return &hooksApi.OnDefineDomainResult{
			DomainXML: params.GetDomainXML(),
		}, nil
	}

	domainXML := params.GetDomainXML()
	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal([]byte(domainXML), &domainSpec)
	if err != nil {
		panic(err)
	}

	if domainSpec.OS.SMBios == nil {
		domainSpec.OS.SMBios = &domainSchema.SMBios{Mode: "sysinfo"}
	} else {
		domainSpec.OS.SMBios.Mode = "sysinfo"
	}

	if domainSpec.SysInfo == nil {
		domainSpec.SysInfo = &domainSchema.SysInfo{}
	}
	domainSpec.Type = "smbios"
	if baseBoardManufacturer, found := annotations[baseBoardManufacturerAnnotation]; found {
		domainSpec.SysInfo.BaseBoard = append(domainSpec.SysInfo.BaseBoard, domainSchema.Entry{
			Name:  "manufacturer",
			Value: baseBoardManufacturer,
		})
	}

	newDomainXMLRaw, err := xml.Marshal(domainSpec)
	if err != nil {
		panic(err)
	}

	newDomainXML := string(newDomainXMLRaw[:])

	return &hooksApi.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	newHookServer().Run()
}
