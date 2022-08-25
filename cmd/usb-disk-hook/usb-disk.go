package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	vmSchema "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha1 "kubevirt.io/kubevirt/pkg/hooks/v1alpha1"
	domainSchema "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
)

const (
	baseBoardManufacturerAnnotation = "smbios.vm.kubevirt.io/baseBoardManufacturer"
	onDefineDomainLoggingMessage    = "Hook's OnDefineDomain callback method has been called"

	hookName = "usb-disk"
	version  = "v1alpha1"
)

type usbDiskHookinfoServer struct{}

func (s usbDiskHookinfoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: hookName,
		Versions: []string{
			version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 0,
			},
		},
	}, nil
}

type usbDiskHookServer struct {
	diskName string
}

func (u usbDiskHookServer) OnDefineDomain(ctx context.Context, params *hooksV1alpha1.OnDefineDomainParams) (*hooksV1alpha1.OnDefineDomainResult, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	domainSpec := domainSchema.DomainSpec{}
	err := xml.Unmarshal(params.GetDomainXML(), &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", params.GetDomainXML())
		panic(err)
	}

	newDomainXML, err := onDefineDomain(domainSpec, u.diskName)
	if err != nil {
		return nil, err
	}

	return &hooksV1alpha1.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func onDefineDomain(domainSpec domainSchema.DomainSpec, diskName string) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	diskIdx := -1
	for i, curDisk := range domainSpec.Devices.Disks {
		if curDisk.Alias == nil {
			return nil, fmt.Errorf("disk alias is nil")
		}

		if strings.Contains(curDisk.Alias.GetName(), diskName) {
			diskIdx = i
			break
		}
	}

	if diskIdx == -1 {
		return nil, fmt.Errorf("disk with name %s does not exist", diskName)
	}

	domainSpec.Devices.Disks[diskIdx].Target.Bus = "usb"
	domainSpec.Devices.Disks[diskIdx].Model = ""

	usbControllerIdx := -1
	for i, controller := range domainSpec.Devices.Controllers {
		if controller.Type == "usb" {
			usbControllerIdx = i
			break
		}
	}

	if usbControllerIdx == -1 {
		domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers,
			domainSchema.Controller{
				Type:  "usb",
				Index: "0",
				Model: "qemu-xhci",
			},
		)
	} else {
		domainSpec.Devices.Controllers[usbControllerIdx].Model = "qemu-xhci"
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Infof("Successfully updated disk %s's bus to be of type usb", diskName)

	return newDomainXML, nil
}

func onDefineDomainOLD(vmiJSON []byte, domainXML []byte) ([]byte, error) {
	log.Log.Info(onDefineDomainLoggingMessage)

	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	annotations := vmiSpec.GetAnnotations()

	if _, found := annotations[baseBoardManufacturerAnnotation]; !found {
		log.Log.Info("SM BIOS hook sidecar was requested, but no attributes provided. Returning original domain spec")
		return domainXML, nil
	}

	domainSpec := domainSchema.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	domainSpec.OS.SMBios = &domainSchema.SMBios{Mode: "sysinfo"}

	if domainSpec.SysInfo == nil {
		domainSpec.SysInfo = &domainSchema.SysInfo{}
	}
	domainSpec.SysInfo.Type = "smbios"
	if baseBoardManufacturer, found := annotations[baseBoardManufacturerAnnotation]; found {
		domainSpec.SysInfo.BaseBoard = append(domainSpec.SysInfo.BaseBoard, domainSchema.Entry{
			Name:  "manufacturer",
			Value: baseBoardManufacturer,
		})
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested SMBIOS attributes")

	return newDomainXML, nil
}

func main() {
	log.InitializeLogging("usb-disk-hook")

	var diskName string
	pflag.StringVar(&diskName, "disk", "", "disk name to convert to usb bus type")
	pflag.Parse()

	socketPath := filepath.Join(hooks.HookSocketsSharedDirectory, fmt.Sprintf("%s.sock", hookName))
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	if diskName == "" {
		panic(fmt.Errorf("usage: \n        /usb-disk-hook --disk myAwesomeDisk"))
	}
	hooksInfo.RegisterInfoServer(server, usbDiskHookinfoServer{})
	hooksV1alpha1.RegisterCallbacksServer(server, usbDiskHookServer{diskName})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)
	server.Serve(socket)
}
