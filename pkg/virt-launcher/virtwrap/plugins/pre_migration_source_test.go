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

package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"libvirt.org/go/libvirt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	pluginv1alpha1 "kubevirt.io/api/plugin/v1alpha1"

	pluginsv1alpha1 "kubevirt.io/kubevirt/pkg/hooks/plugins/v1alpha1"
)

// exposedParams maps every libvirt.DomainMigrateParameters value field exposed to
// PreMigrationSource hooks to the proto field it converts to. deniedParams lists
// the fields deliberately not exposed, with the reason. Together they must cover
// every field of libvirt.DomainMigrateParameters (the XxxSet presence flags follow
// their value field), so a libvirt-go release adding a field fails the
// exhaustiveness test below and forces an explicit decision.
var exposedParams = map[string]string{
	"URI":                          "Uri",
	"DestName":                     "DestName",
	"DestXML":                      "DestXml",
	"PersistXML":                   "PersistXml",
	"Bandwidth":                    "Bandwidth",
	"ParallelConnections":          "ParallelConnections",
	"MigrateDisks":                 "MigrateDisks",
	"MigrateDisksDetectZeroesList": "MigrateDisksDetectZeroesList",
	"DisksURI":                     "DisksUri",
	"Compression":                  "Compression",
	"GraphicsURI":                  "GraphicsUri",
	"ListenAddress":                "ListenAddress",
	"DisksPort":                    "DisksPort",
	"CompressionMTLevel":           "CompressionMtLevel",
	"CompressionMTThreads":         "CompressionMtThreads",
	"CompressionMTDThreads":        "CompressionMtDthreads",
	"CompressionXBZRLECache":       "CompressionXbzrleCache",
	"AutoConvergeInitial":          "AutoConvergeInitial",
	"AutoConvergeIncrement":        "AutoConvergeIncrement",
	"TLSDestination":               "TlsDestination",
	"CompressionZlibLevel":         "CompressionZlibLevel",
	"CompressionZstdLevel":         "CompressionZstdLevel",
	"BandwidthAvailSwitchover":     "BandwidthAvailSwitchover",
	"MigrateDisksTargetZero":       "MigrateDisksTargetZero",
}

var deniedParams = map[string]string{
	"MigrateDisksDetectZeroes": "deprecated upstream: wrong type and unused",
}

// fakeHookSidecar is a programmable LauncherHookService test double. The legacy
// protoc-gen-go used by this repo generates no Unimplemented server stub, so the
// hook points the tests do not program must be stubbed out explicitly.
type fakeHookSidecar struct {
	preMigrationSource func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error)
}

func (s *fakeHookSidecar) GuestDefinition(context.Context, *pluginsv1alpha1.GuestDefinitionRequest) (*pluginsv1alpha1.GuestDefinitionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented in this test double")
}

func (s *fakeHookSidecar) PreBoot(context.Context, *pluginsv1alpha1.PreBootRequest) (*pluginsv1alpha1.PreBootResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented in this test double")
}

func (s *fakeHookSidecar) PreMigrationSource(ctx context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
	return s.preMigrationSource(ctx, req)
}

var _ = Describe("PreMigrationSource hooks", func() {
	const testFlags = libvirt.DomainMigrateFlags(0b1010)

	var vmi *v1.VirtualMachineInstance

	BeforeEach(func() {
		vmi = &v1.VirtualMachineInstance{ObjectMeta: metav1.ObjectMeta{Name: "test-vmi", Namespace: "default"}}
	})

	baseParams := func() *libvirt.DomainMigrateParameters {
		return &libvirt.DomainMigrateParameters{
			URI: "unix:///migration-uri", URISet: true,
			DestName: "dest-name", DestNameSet: true,
			DestXML: "<domain>dest</domain>", DestXMLSet: true,
		}
	}

	migrationSidecarPlugin := func(name, socketPath string, failureStrategy pluginv1alpha1.FailureStrategy) pluginv1alpha1.Plugin {
		return pluginv1alpha1.Plugin{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Spec: pluginv1alpha1.PluginSpec{
				LauncherHooks: []pluginv1alpha1.LauncherHook{
					{
						Sidecar:         &pluginv1alpha1.SidecarLauncherHook{SocketPath: socketPath, PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookPreMigrationSource}},
						FailureStrategy: failureStrategy,
					},
				},
			},
		}
	}

	// startSidecar serves a real LauncherHookService on a unix socket under
	// socketBaseDir - callSidecar always dials a real socket, so the fake has to
	// be a real listener - and returns the socket path for pluginName.
	startSidecar := func(socketBaseDir, pluginName string, handler func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error)) string {
		pluginDir := filepath.Join(socketBaseDir, pluginName)
		Expect(os.MkdirAll(pluginDir, 0755)).To(Succeed())
		socketPath := filepath.Join(pluginDir, "hook.sock")

		listener, err := net.Listen("unix", socketPath)
		Expect(err).NotTo(HaveOccurred())

		server := grpc.NewServer()
		pluginsv1alpha1.RegisterLauncherHookServiceServer(server, &fakeHookSidecar{preMigrationSource: handler})
		go func() { _ = server.Serve(listener) }()
		DeferCleanup(server.Stop)

		return socketPath
	}

	testOpts := func(socketBaseDir string) sidecarCallOptions {
		return sidecarCallOptions{
			socketBaseDir: socketBaseDir,
			readiness:     readinessBudget{deadline: time.Now().Add(time.Second), total: time.Second},
			socketReady:   osStatSocketReady,
		}
	}

	Context("hook selection", func() {
		It("returns the configuration unchanged when there are no plugins", func() {
			flags, params, err := applyPreMigrationSourceHooks(context.Background(), nil, vmi, testFlags, baseParams(), testOpts(""))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags))
			Expect(params).To(Equal(baseParams()))
		})

		It("returns the configuration unchanged when no plugin declares a PreMigrationSource hook", func() {
			plugin := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "guest-definition-only"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Sidecar: &pluginv1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/none/hook.sock", PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookGuestDefinition}}},
					},
				},
			}

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{plugin}, vmi, testFlags, baseParams(), testOpts(""))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags))
			Expect(params).To(Equal(baseParams()))
		})
	})

	Context("hook execution", func() {
		It("sends the VMI and the current configuration, and applies the response", func() {
			socketBaseDir := GinkgoT().TempDir()
			var capturedRequest *pluginsv1alpha1.PreMigrationSourceRequest
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				capturedRequest = req
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfig{
							Flags:  uint64(testFlags) | 1<<5,
							Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: &pluginsv1alpha1.OptionalString{Value: "zstd"}},
						},
					},
				}, nil
			})

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())

			expectedVMIJSON, err := json.Marshal(vmi)
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedRequest.Vmi).To(Equal(expectedVMIJSON))

			requestConfig, ok := capturedRequest.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceRequest_Libvirt)
			Expect(ok).To(BeTrue())
			Expect(requestConfig.Libvirt.Flags).To(Equal(uint64(testFlags)))
			Expect(requestConfig.Libvirt.Params.Uri).NotTo(BeNil())
			Expect(requestConfig.Libvirt.Params.Uri.Value).To(Equal("unix:///migration-uri"))
			Expect(requestConfig.Libvirt.Params.DestName.Value).To(Equal("dest-name"))

			Expect(flags).To(Equal(testFlags | 1<<5))
			Expect(params.Compression).To(Equal("zstd"))
			Expect(params.CompressionSet).To(BeTrue())
			Expect(params.URI).To(Equal(""), "the response replaces the whole configuration")
			Expect(params.URISet).To(BeFalse(), "the response replaces the whole configuration")
		})

		It("chains hooks so each receives the previous hook's output", func() {
			socketBaseDir := GinkgoT().TempDir()
			var secondRequestConfig *pluginsv1alpha1.LibvirtMigrationConfig
			firstSocketPath := startSidecar(socketBaseDir, "a-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				requestConfig := req.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceRequest_Libvirt)
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfig{Flags: requestConfig.Libvirt.Flags, Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: &pluginsv1alpha1.OptionalString{Value: "zstd"}}},
					},
				}, nil
			})
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				secondRequestConfig = req.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceRequest_Libvirt).Libvirt
				return &pluginsv1alpha1.PreMigrationSourceResponse{MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{Libvirt: secondRequestConfig}}, nil
			})

			_, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyFail),
				migrationSidecarPlugin("a-plugin", firstSocketPath, pluginv1alpha1.FailureStrategyFail),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())

			Expect(secondRequestConfig.Params.Compression).NotTo(BeNil())
			Expect(secondRequestConfig.Params.Compression.Value).To(Equal("zstd"), "the second hook must receive the first hook's output")
			Expect(params.Compression).To(Equal("zstd"))
		})

		It("keeps earlier modifications when a later Ignore-strategy hook fails", func() {
			socketBaseDir := GinkgoT().TempDir()
			firstSocketPath := startSidecar(socketBaseDir, "a-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfig{
							Flags:  uint64(testFlags),
							Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: &pluginsv1alpha1.OptionalString{Value: "zstd"}},
						},
					},
				}, nil
			})
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})

			_, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("a-plugin", firstSocketPath, pluginv1alpha1.FailureStrategyFail),
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyIgnore),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Compression).To(Equal("zstd"))
		})

		It("fails the run when a Fail-strategy hook fails", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})

			_, _, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(MatchError(ContainSubstring("boom")))
		})

		It("rejects a response with an unset migration configuration variant", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{}, nil
			})

			_, _, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(MatchError(ContainSubstring("unsupported migration configuration variant")))
		})
	})

	Context("libvirt conversion", func() {
		DescribeTable("round-trips the libvirt migration configuration",
			func(params libvirt.DomainMigrateParameters) {
				flags := libvirt.DomainMigrateFlags(0xdeadbeef)
				backFlags, backParams := libvirtConfigFromProto(libvirtConfigToProto(flags, &params))
				Expect(backFlags).To(Equal(flags))
				Expect(*backParams).To(Equal(params))
			},
			Entry("with every exposed field set to a distinct sentinel value", libvirt.DomainMigrateParameters{
				URI: "unix:///uri", URISet: true,
				DestName: "dest-name", DestNameSet: true,
				DestXML: "<domain>dest</domain>", DestXMLSet: true,
				PersistXML: "<domain>persist</domain>", PersistXMLSet: true,
				Bandwidth: 1234, BandwidthSet: true,
				ParallelConnections: 7, ParallelConnectionsSet: true,
				MigrateDisks: []string{"vda", "vdb"}, MigrateDisksSet: true,
				MigrateDisksDetectZeroesList: []string{"vda"}, MigrateDisksDetectZeroesSet: true,
				DisksURI: "unix:///disks", DisksURISet: true,
				Compression: "zstd", CompressionSet: true,
				GraphicsURI: "spice:///graphics", GraphicsURISet: true,
				ListenAddress: "0.0.0.0", ListenAddressSet: true,
				DisksPort: 49152, DisksPortSet: true,
				CompressionMTLevel: 1, CompressionMTLevelSet: true,
				CompressionMTThreads: 2, CompressionMTThreadsSet: true,
				CompressionMTDThreads: 3, CompressionMTDThreadsSet: true,
				CompressionXBZRLECache: 4096, CompressionXBZRLECacheSet: true,
				AutoConvergeInitial: 30, AutoConvergeInitialSet: true,
				AutoConvergeIncrement: 31, AutoConvergeIncrementSet: true,
				TLSDestination: "tls://destination", TLSDestinationSet: true,
				CompressionZlibLevel: 6, CompressionZlibLevelSet: true,
				CompressionZstdLevel: 9, CompressionZstdLevelSet: true,
				BandwidthAvailSwitchover: 5678, BandwidthAvailSwitchoverSet: true,
				MigrateDisksTargetZero: []string{"vdc"}, MigrateDisksTargetZeroSet: true,
			}),
			Entry("with no field set", libvirt.DomainMigrateParameters{}),
		)

		It("exposes or denies every libvirt migration parameter field", func() {
			paramsType := reflect.TypeOf(libvirt.DomainMigrateParameters{})
			for i := 0; i < paramsType.NumField(); i++ {
				field := paramsType.Field(i)
				_, exposed := exposedParams[field.Name]
				_, denied := deniedParams[field.Name]
				if strings.HasSuffix(field.Name, "Set") {
					valueField := strings.TrimSuffix(field.Name, "Set")
					_, valueExposed := exposedParams[valueField]
					_, valueDenied := deniedParams[valueField]
					Expect(valueExposed || valueDenied).To(BeTrue(), "presence flag %s must follow a classified value field", field.Name)
					continue
				}
				Expect(exposed && denied).To(BeFalse(), "%s is both exposed and denied", field.Name)
				Expect(exposed || denied).To(BeTrue(), "%s is neither exposed nor denied: decide whether PreMigrationSource hooks may set it", field.Name)
			}
		})

		It("maps every exposed libvirt parameter to a proto field", func() {
			protoParamsType := reflect.TypeOf(pluginsv1alpha1.LibvirtDomainMigrateParameters{})
			protoFields := 0
			for i := 0; i < protoParamsType.NumField(); i++ {
				if protoParamsType.Field(i).IsExported() {
					protoFields++
				}
			}
			Expect(protoFields).To(Equal(len(exposedParams)), "every proto field must be the target of exactly one exposed libvirt field")
			for libvirtField, protoField := range exposedParams {
				_, ok := protoParamsType.FieldByName(protoField)
				Expect(ok).To(BeTrue(), "exposed libvirt field %s maps to missing proto field %s", libvirtField, protoField)
			}
		})
	})
})
