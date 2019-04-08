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
 * Copyright 2019 SAP SE
 *
 */

package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"os"

	libvirt "github.com/libvirt/libvirt-go"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"

	vmSchema "kubevirt.io/kubevirt/pkg/api/v1"
	"kubevirt.io/kubevirt/pkg/hooks"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha2 "kubevirt.io/kubevirt/pkg/hooks/v1alpha2"
	"kubevirt.io/kubevirt/pkg/log"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/cli"
)

type infoServer struct {
	Version string
}

type monList []cephMon

type cephMon struct {
	Host string `json:"host"`
	Port string `json:"port,omitempty"`
}

type attachmentList []attachment

type attachment struct {
	Pool   string `json:"pool"`
	Volume string `json:"volume"`
	Device string `json:"device"`
	Bus    string `json:"bus,omitempty"`
}

type cephConfig struct {
	User        string
	Key         string
	Mons        []cephMon
	attachments []attachment
}

func (s infoServer) Info(ctx context.Context, params *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	log.Log.Info("Hook's Info method has been called")

	return &hooksInfo.InfoResult{
		Name: "rbd-hotplug",
		Versions: []string{
			hooksV1alpha2.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnSyncVMI,
				Priority: 0,
			},
			&hooksInfo.HookPoint{
				Name:     hooksInfo.OnDefineDomainHookPointName,
				Priority: 1,
			},
		},
	}, nil
}

const (
	cephMonDefaultPort = "6789"
	cephUser           = "rbd-hotplug.vm.kubevirt.io/user"

	// TODO move out to a k8s secret
	cephSecret      = "rbd-hotplug.vm.kubevirt.io/secret"
	cephMons        = "rbd-hotplug.vm.kubevirt.io/monitors"
	cephAttachments = "rbd-hotplug.vm.kubevirt.io/attachments"

	secretXML = "<secret ephemeral='no' private='no'><usage type='ceph'><name>client.vmimages secret</name></usage></secret>"
)

type v1alpha2Server struct{}

func (s v1alpha2Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha2.PreCloudInitIsoParams) (*hooksV1alpha2.PreCloudInitIsoResult, error) {
	return nil, nil
}

func getAllConfiguredAnnotations(annotations map[string]string) (*cephConfig, error) {
	cconf := cephConfig{}
	if user, found := annotations[cephUser]; !found {
		return nil, fmt.Errorf("%s is not specifed in annotation", cephUser)
	} else {
		cconf.User = user
	}
	if key, found := annotations[cephSecret]; !found {
		return nil, fmt.Errorf("%s is not specifed in annotation", cephSecret)
	} else {
		cconf.Key = key
	}
	if mons, found := annotations[cephMons]; !found {
		return nil, fmt.Errorf("%s is not specifed in annotation", cephMons)
	} else {
		monitors := make(monList, 0)
		if err := json.Unmarshal([]byte(mons), &monitors); err != nil {
			return nil, err
		}
		cconf.Mons = monitors
	}
	if pvcs, found := annotations[cephAttachments]; found {
		att := make(attachmentList, 0)
		if err := json.Unmarshal([]byte(pvcs), &att); err != nil {
			return nil, err
		}
		cconf.attachments = att
	}
	return &cconf, nil
}

func getAllDomainDisks(dom cli.VirDomain) ([]api.Disk, error) {
	xmlstr, err := dom.GetXMLDesc(0)
	if err != nil {
		return nil, err
	}

	var newSpec api.DomainSpec
	err = xml.Unmarshal([]byte(xmlstr), &newSpec)
	if err != nil {
		return nil, err
	}

	return newSpec.Devices.Disks, nil
}

func buildDisksXML(pvc attachment, user string, mons monList) api.Disk {
	// build device xml
	sec := api.DiskSecret{Type: "ceph", Usage: "client.vmimages secret"}
	auth := api.DiskAuth{Username: user, Secret: &sec}
	driver := api.DiskDriver{Name: "qemu", Type: "raw", Cache: "writeback"}
	var hosts []api.DiskSourceHost
	for _, cephMon := range mons {
		if cephMon.Port == "" {
			cephMon.Port = cephMonDefaultPort
		}
		host := api.DiskSourceHost{Name: cephMon.Host, Port: cephMon.Port}
		hosts = append(hosts, host)
	}
	targetName := fmt.Sprintf("%s/%s", pvc.Pool, pvc.Volume)
	source := api.DiskSource{Protocol: "rbd", Name: targetName, Host: hosts}
	target := api.DiskTarget{Bus: pvc.Bus, Device: pvc.Device}
	disk := api.Disk{Type: "network", Device: "disk", Auth: &auth, Driver: &driver, Source: source, Target: target}
	return disk
}

func defineSecret(conn *libvirt.Connect, key string) error {
	sec, err := conn.LookupSecretByUsage(libvirt.SECRET_USAGE_TYPE_CEPH, "client.vmimages secret")
	if err != nil {
		log.Log.Info("Secret not defined. Define secret and set key to libvirt")
		sec, err = conn.SecretDefineXML(secretXML, 0)
		if err != nil {
			return err
		}
	}

	rawKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return err
	}
	sec.SetValue(rawKey, 0)
	return nil
}

func (s v1alpha2Server) OnSyncVMI(_ context.Context, params *hooksV1alpha2.OnSyncVMIParams) (*hooksV1alpha2.Empty, error) {
	log.Log.Info("Hook's OnSyncVMI callback method has been called ")
	vmiSpec := vmSchema.VirtualMachineInstance{}
	err := json.Unmarshal(params.GetVmi(), &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Error("Failed to unmarshal given VMI spec:")
		panic(err)
	}

	if vmiSpec.Status.Phase != "Running" {
		log.Log.Warningf("VM status is not running (status: %s). Cannot attach disks.", vmiSpec.Status.Phase)
		return &hooksV1alpha2.Empty{}, nil
	}
	currentAnnotation, err := getAllConfiguredAnnotations(vmiSpec.GetAnnotations())
	if err != nil {
		panic(err)
	}

	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	err = defineSecret(conn, currentAnnotation.Key)
	if err != nil {
		panic(err)
	}

	// get domain
	domain, err := conn.LookupDomainById(1)
	if err != nil {
		log.Log.Reason(err).Errorf("Cannot find domain %s", vmiSpec.Name)
		return &hooksV1alpha2.Empty{}, nil
	}

	disks, err := getAllDomainDisks(domain)
	if err != nil {
		log.Log.Reason(err).Error("Cannot get domain disks")
		return &hooksV1alpha2.Empty{}, nil
	}
	// check for disks that need to be detached
	for _, disk := range disks {
		if disk.Source.Protocol == "rbd" {
			toBeDeleted := true
			for _, pvc := range currentAnnotation.attachments {
				if disk.Target.Device == pvc.Device {
					toBeDeleted = false
					break
				}
			}
			if toBeDeleted {
				diskXML, err := xml.Marshal(&disk)
				if err != nil {
					log.Log.Reason(err).Errorf("Disk %s cannot be unmarshaled", disk.Target.Device)
					continue
				}
				log.Log.Infof("Detach (rbd) disk %s", disk.Target.Device)
				err = domain.DetachDevice(string(diskXML))
				if err != nil {
					log.Log.Reason(err).Error("Cannot detach disk")
				}
			}
		}
	}

	// attach disks
	for _, pvc := range currentAnnotation.attachments {
		// check if disk is already attached
		alreadyAttached := false
		for _, disk := range disks {
			if disk.Source.Protocol == "rbd" && disk.Target.Device == pvc.Device {
				// disk is already attached
				alreadyAttached = true
			}
		}
		if alreadyAttached {
			continue
		}

		disk := buildDisksXML(pvc, currentAnnotation.User, currentAnnotation.Mons)

		diskXML, err := xml.Marshal(&disk)
		if err != nil {
			log.Log.Reason(err).Errorf("Cannot marshal XML %s", disk.Target.Device)
			continue
		}

		log.Log.Infof("Attach (rbd) disk %s", disk.Target.Device)
		err = domain.AttachDevice(string(diskXML))
		if err != nil {
			log.Log.Reason(err).Error("Cannot attach disk")
		}
	}

	return &hooksV1alpha2.Empty{}, nil
}

func (s v1alpha2Server) OnDefineDomain(ctx context.Context, params *hooksV1alpha2.OnDefineDomainParams) (*hooksV1alpha2.OnDefineDomainResult, error) {
	log.Log.Info("Hook's OnDefineDomain callback method has been called")
	vmiSpec := vmSchema.VirtualMachineInstance{}
	vmiJSON := params.GetVmi()
	domainXML := params.GetDomainXML()
	err := json.Unmarshal(vmiJSON, &vmiSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given VMI spec: %s", vmiJSON)
		panic(err)
	}

	domainSpec := api.DomainSpec{}
	err = xml.Unmarshal(domainXML, &domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to unmarshal given domain spec: %s", domainXML)
		panic(err)
	}

	// replace the default lsi scsi controller with virtio-scsi which is way faster
	domainSpec.Devices.Controllers = append(domainSpec.Devices.Controllers, api.Controller{
		Type:  "scsi",
		Index: "0",
		Model: "virtio-scsi",
	})

	currentAnnotation, err := getAllConfiguredAnnotations(vmiSpec.GetAnnotations())
	if err != nil {
		panic(err)
	}

	for _, pvc := range currentAnnotation.attachments {
		disk := buildDisksXML(pvc, currentAnnotation.User, currentAnnotation.Mons)
		log.Log.Infof("Attach (rbd) disk %s during definition phase", disk.Target.Device)
		domainSpec.Devices.Disks = append(domainSpec.Devices.Disks, disk)
	}

	newDomainXML, err := xml.Marshal(domainSpec)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to marshal updated domain spec: %+v", domainSpec)
		panic(err)
	}
	conn, err := libvirt.NewConnect("qemu:///system")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	err = defineSecret(conn, currentAnnotation.Key)
	if err != nil {
		panic(err)
	}

	log.Log.Info("Successfully updated original domain spec with requested disk attributes")

	return &hooksV1alpha2.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func main() {
	log.InitializeLogging("rbd-hotplug-sidecar")

	var version string
	pflag.StringVar(&version, "version", "", "hook version to use")
	pflag.Parse()

	socketPath := hooks.HookSocketsSharedDirectory + "/rbd-hotplug.sock"
	socket, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("Failed to initialized socket on path: %s", socket)
		log.Log.Error("Check whether given directory exists and socket name is not already taken by other file")
		panic(err)
	}
	defer os.Remove(socketPath)

	server := grpc.NewServer([]grpc.ServerOption{}...)

	//hooksV1alpha1.Version,
	hooksInfo.RegisterInfoServer(server, infoServer{Version: version})
	hooksV1alpha2.RegisterCallbacksServer(server, v1alpha2Server{})
	log.Log.Infof("Starting hook server exposing 'info' and 'v1alpha1' services on socket %s", socketPath)

	server.Serve(socket)
}
