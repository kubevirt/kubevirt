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
	"bytes"
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

const (
	usage = "usage: test-launcher-hook-sidecar <socket-path> | --sleep | --error [hook-point] <socket-path>"

	guestDefinitionHook    = "GuestDefinition"
	preBootHook            = "PreBoot"
	preMigrationSourceHook = "PreMigrationSource"

	// These strings are deliberately the same length. The test sidecar uses a
	// byte-for-byte replacement in the generated ISO so the resulting image
	// remains a valid ISO9660 filesystem without needing an ISO utility in the
	// small sidecar image.
	preBootOriginal = "VEP190_PREBOOT_BEFORE"
	preBootMutated  = "VEP190_PREBOOT_AFTER!"
)

type server struct {
	cpuVendor  string
	errorHooks map[string]bool
}

func (s *server) GuestDefinition(_ context.Context, req *pluginsv1alpha1.GuestDefinitionRequest) (*pluginsv1alpha1.GuestDefinitionResponse, error) {
	if err := s.errorFor(guestDefinitionHook); err != nil {
		return nil, err
	}

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
	return &pluginsv1alpha1.GuestDefinitionResponse{Domain: []byte(xml)}, nil
}

func (s *server) PreBoot(_ context.Context, req *pluginsv1alpha1.PreBootRequest) (*pluginsv1alpha1.PreBootResponse, error) {
	if err := s.errorFor(preBootHook); err != nil {
		return nil, err
	}

	isoPath := req.PreBootContext.ArtifactPaths[pluginsv1alpha1.PreBootArtifactCloudInitISO]
	if isoPath == "" {
		return nil, fmt.Errorf("PreBoot request has no cloudInitISO artifact")
	}

	iso, err := os.ReadFile(isoPath)
	if err != nil {
		return nil, fmt.Errorf("read cloud-init ISO %q: %w", isoPath, err)
	}
	if !bytes.Contains(iso, []byte(preBootOriginal)) {
		if bytes.Contains(iso, []byte(preBootMutated)) {
			// PreBoot may be retried after a later start step fails. Treat an
			// already-mutated artifact as success so this test sidecar is
			// idempotent, as a real artifact-mutating hook must be.
			log.Printf("PreBoot found cloud-init ISO %s already mutated", isoPath)
			return &pluginsv1alpha1.PreBootResponse{}, nil
		}
		return nil, fmt.Errorf("cloud-init ISO %q does not contain %q", isoPath, preBootOriginal)
	}
	mutated := bytes.Replace(iso, []byte(preBootOriginal), []byte(preBootMutated), 1)
	if err := os.WriteFile(isoPath, mutated, 0600); err != nil {
		return nil, fmt.Errorf("write cloud-init ISO %q: %w", isoPath, err)
	}

	log.Printf("PreBoot mutated cloud-init ISO %s: %s -> %s", isoPath, preBootOriginal, preBootMutated)
	return &pluginsv1alpha1.PreBootResponse{}, nil
}

func (s *server) PreMigrationSource(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
	if err := s.errorFor(preMigrationSourceHook); err != nil {
		return nil, err
	}
	requestConfig, ok := req.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceRequest_Libvirt)
	if !ok || requestConfig.Libvirt == nil {
		return nil, fmt.Errorf("unsupported migration configuration variant %T", req.MigrationConfig)
	}

	// VIR_MIGRATE_PARALLEL = 1<<17 (libvirt-domain.h); hardcoded to avoid linking cgo libvirt bindings into this sidecar image.
	const virMigrateParallel = 1 << 17

	// The response carries only the change: set_flags adds a bit to the flags KubeVirt
	// computed, so the flags KubeVirt always sets survive untouched.
	change := &pluginsv1alpha1.LibvirtMigrationConfigChanges{
		SetFlags: virMigrateParallel,
		Params:   &pluginsv1alpha1.LibvirtDomainMigrateParameters{ParallelConnections: &pluginsv1alpha1.OptionalInt32{Kind: &pluginsv1alpha1.OptionalInt32_Value{Value: 2}}},
	}

	log.Printf("PreMigrationSource mutated migration config: parallel connections=%d set_flags=%d", change.GetParams().GetParallelConnections().GetValue(), change.GetSetFlags())

	return &pluginsv1alpha1.PreMigrationSourceResponse{
		MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{Libvirt: change},
	}, nil
}

func (s *server) errorFor(hookPoint string) error {
	if s.errorHooks[hookPoint] {
		err := status.Errorf(codes.Internal, "intentional test error for %s", hookPoint)
		log.Print(err)
		return err
	}
	return nil
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
		log.Fatal(usage)
	}
	if os.Args[1] == "--sleep" {
		log.Println("Sleep mode: blocking forever without creating socket")
		select {}
	}

	socketPath := os.Args[1]
	errorHooks := map[string]bool{}
	if socketPath == "--error" {
		switch len(os.Args) {
		case 3:
			errorHooks[guestDefinitionHook], socketPath = true, os.Args[2]
		case 4:
			errorHooks[os.Args[2]], socketPath = true, os.Args[3]
		default:
			log.Fatal(usage)
		}
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
	pluginsv1alpha1.RegisterLauncherHookServiceServer(s, &server{
		cpuVendor:  detectCPUVendor(),
		errorHooks: errorHooks,
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM)
	go func() {
		<-sigCh
		s.GracefulStop()
	}()

	log.Printf("Listening on %s (error hooks: %v)", socketPath, errorHooks)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}
