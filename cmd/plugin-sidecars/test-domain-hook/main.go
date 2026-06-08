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

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	libvirtxml "libvirt.org/go/libvirtxml"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

type server struct {
	cpuVendor string
}

func (s *server) MutateDomain(_ context.Context, req *pluginsv1alpha1.MutateDomainRequest) (*pluginsv1alpha1.MutateDomainResponse, error) {
	domain := &libvirtxml.Domain{}
	if err := domain.Unmarshal(string(req.Domain)); err != nil {
		return nil, fmt.Errorf("unmarshal domain: %w", err)
	}

	var smbiosSysInfo *libvirtxml.DomainSysInfo
	for i := range domain.SysInfo {
		if domain.SysInfo[i].SMBIOS != nil {
			smbiosSysInfo = &domain.SysInfo[i]
			break
		}
	}
	if smbiosSysInfo == nil {
		domain.SysInfo = append(domain.SysInfo, libvirtxml.DomainSysInfo{
			SMBIOS: &libvirtxml.DomainSysInfoSMBIOS{},
		})
		smbiosSysInfo = &domain.SysInfo[len(domain.SysInfo)-1]
	}
	if smbiosSysInfo.SMBIOS.System == nil {
		smbiosSysInfo.SMBIOS.System = &libvirtxml.DomainSysInfoSystem{}
	}
	vendorValue := fmt.Sprintf("KubeVirt-Plugin-CPU-%s", s.cpuVendor)
	var filtered []libvirtxml.DomainSysInfoEntry
	for _, entry := range smbiosSysInfo.SMBIOS.System.Entry {
		if entry.Name != "manufacturer" {
			filtered = append(filtered, entry)
		}
	}
	smbiosSysInfo.SMBIOS.System.Entry = append(filtered,
		libvirtxml.DomainSysInfoEntry{Name: "manufacturer", Value: vendorValue})

	xml, err := domain.Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal domain: %w", err)
	}
	return &pluginsv1alpha1.MutateDomainResponse{Domain: []byte(xml)}, nil
}

type errorServer struct{}

func (s *errorServer) MutateDomain(_ context.Context, _ *pluginsv1alpha1.MutateDomainRequest) (*pluginsv1alpha1.MutateDomainResponse, error) {
	return nil, status.Errorf(codes.Internal, "intentional test error")
}

func detectCPUVendor() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "Unknown"
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "vendor_id") {
			parts := strings.SplitN(scanner.Text(), ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "Unknown"
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("usage: test-domain-hook-sidecar <socket-path> | --sleep | --error <socket-path>")
	}
	if os.Args[1] == "--sleep" {
		log.Println("Sleep mode: blocking forever without creating socket")
		select {}
	}

	errorMode := os.Args[1] == "--error"
	var socketPath string
	if errorMode {
		if len(os.Args) < 3 {
			log.Fatal("usage: test-domain-hook-sidecar --error <socket-path>")
		}
		socketPath = os.Args[2]
	} else {
		socketPath = os.Args[1]
	}

	if err := os.MkdirAll(filepath.Dir(socketPath), 0755); err != nil {
		log.Fatalf("mkdir: %v", err)
	}
	os.RemoveAll(socketPath)

	lis, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	s := grpc.NewServer()
	if errorMode {
		pluginsv1alpha1.RegisterDomainHookServiceServer(s, &errorServer{})
	} else {
		pluginsv1alpha1.RegisterDomainHookServiceServer(s, &server{cpuVendor: detectCPUVendor()})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		s.GracefulStop()
	}()

	log.Printf("Listening on %s (error mode: %v)", socketPath, errorMode)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
