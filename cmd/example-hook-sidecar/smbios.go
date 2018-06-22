package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net"
	"os"

	"google.golang.org/grpc"

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	hooks "kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"

type infoServer struct{}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	return &hooksInfo.InfoResult{
		Name: "smbios",
		Versions: []string{
			hooksV1alpha1.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 50,
			},
		},
	}, nil
}

type v1alpha1Server struct{}

func (s v1alpha1Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	vmJSON := params.GetVm()
	vmSpec := vmSchema.VirtualMachine{}
	err := json.Unmarshal([]byte(vmJSON), &vmSpec)
	if err != nil {
		panic(err)
	}

	annotations := vmSpec.GetAnnotations()

	if _, found := annotations[baseBoardManufacturerAnnotation]; !found {
		return &hooksV1alpha1.OnDefineDomainResult{
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

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	socketPath := hooks.HookSocketsSharedDirectory + "/smbios.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)
	hooksInfo.RegisterInfoServer(server, infoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, v1alpha1Server{})
	server.Serve(socket)
}
