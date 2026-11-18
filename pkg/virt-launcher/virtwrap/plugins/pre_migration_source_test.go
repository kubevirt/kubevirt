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
	"errors"
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

// The file lives in package plugins, so its wrapper constructors must not reuse the
// production helper names.
func testOptionalString(value string) *pluginsv1alpha1.OptionalString {
	return &pluginsv1alpha1.OptionalString{Kind: &pluginsv1alpha1.OptionalString_Value{Value: value}}
}

func testEmptyString() *pluginsv1alpha1.OptionalString {
	return &pluginsv1alpha1.OptionalString{}
}

func testOptionalUint64(value uint64) *pluginsv1alpha1.OptionalUint64 {
	return &pluginsv1alpha1.OptionalUint64{Kind: &pluginsv1alpha1.OptionalUint64_Value{Value: value}}
}

func testEmptyUint64() *pluginsv1alpha1.OptionalUint64 {
	return &pluginsv1alpha1.OptionalUint64{}
}

func testOptionalInt32(value int32) *pluginsv1alpha1.OptionalInt32 {
	return &pluginsv1alpha1.OptionalInt32{Kind: &pluginsv1alpha1.OptionalInt32_Value{Value: value}}
}

func testEmptyInt32() *pluginsv1alpha1.OptionalInt32 {
	return &pluginsv1alpha1.OptionalInt32{}
}

func testStringList(values ...string) *pluginsv1alpha1.StringList {
	return &pluginsv1alpha1.StringList{Values: values}
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
	// socketBaseDir - dialSidecar always dials a real socket, so the fake has to
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

	Context("fast-path arming selection", func() {
		It("reports whether PreMigrationSource hooks are selected from the plugin snapshot", func() {
			previous := GetPlugins()
			DeferCleanup(SetPlugins, previous)

			SetPlugins(nil)
			Expect(PreMigrationSourceHooksSelected(vmi)).To(BeFalse(), "no plugins means no hooks")

			guestDefinitionOnly := pluginv1alpha1.Plugin{
				ObjectMeta: metav1.ObjectMeta{Name: "guest-definition-only"},
				Spec: pluginv1alpha1.PluginSpec{
					LauncherHooks: []pluginv1alpha1.LauncherHook{
						{Sidecar: &pluginv1alpha1.SidecarLauncherHook{SocketPath: "/var/run/kubevirt-plugin/none/hook.sock", PermittedHooks: []pluginv1alpha1.LauncherHookPoint{pluginv1alpha1.LauncherHookGuestDefinition}}},
					},
				},
			}
			SetPlugins([]pluginv1alpha1.Plugin{guestDefinitionOnly})
			Expect(PreMigrationSourceHooksSelected(vmi)).To(BeFalse(), "a plugin without a PreMigrationSource hook does not arm the fast path")

			SetPlugins([]pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", "/var/run/kubevirt-plugin/test-plugin/hook.sock", pluginv1alpha1.FailureStrategyFail)})
			Expect(PreMigrationSourceHooksSelected(vmi)).To(BeTrue(), "a selected PreMigrationSource hook arms the fast path")

			brokenCondition := migrationSidecarPlugin("broken-condition", "/var/run/kubevirt-plugin/broken-condition/hook.sock", pluginv1alpha1.FailureStrategyFail)
			brokenCondition.Spec.LauncherHooks[0].Condition = "invalid ["
			SetPlugins([]pluginv1alpha1.Plugin{brokenCondition})
			Expect(PreMigrationSourceHooksSelected(vmi)).To(BeFalse(), "a failed selection runs no hooks, so the fast path stays unarmed")
		})
	})

	Context("hook execution", func() {
		It("sends the VMI and the current configuration, and applies the response's changes", func() {
			socketBaseDir := GinkgoT().TempDir()
			var capturedRequest *pluginsv1alpha1.PreMigrationSourceRequest
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				capturedRequest = req
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{
							SetFlags: 1 << 5,
							Params:   &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("zstd")},
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
			Expect(requestConfig.Libvirt.GetFlags()).To(Equal(uint64(testFlags)))
			Expect(requestConfig.Libvirt.Params.GetUri().GetValue()).To(Equal("unix:///migration-uri"))
			Expect(requestConfig.Libvirt.Params.GetDestName().GetValue()).To(Equal("dest-name"))

			Expect(flags).To(Equal(testFlags | 1<<5))
			Expect(params.Compression).To(Equal("zstd"))
			Expect(params.CompressionSet).To(BeTrue())
			Expect(params.URI).To(Equal("unix:///migration-uri"), "the response carries only its changes, so unset parameters are kept")
			Expect(params.URISet).To(BeTrue(), "the response carries only its changes, so unset parameters are kept")
		})

		It("chains hooks so each receives the previous hook's output", func() {
			socketBaseDir := GinkgoT().TempDir()
			var secondRequestConfig *pluginsv1alpha1.LibvirtMigrationConfig
			firstSocketPath := startSidecar(socketBaseDir, "a-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 5, Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("zstd")}},
					},
				}, nil
			})
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				secondRequestConfig = req.MigrationConfig.(*pluginsv1alpha1.PreMigrationSourceRequest_Libvirt).Libvirt
				return &pluginsv1alpha1.PreMigrationSourceResponse{}, nil
			})

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyFail),
				migrationSidecarPlugin("a-plugin", firstSocketPath, pluginv1alpha1.FailureStrategyFail),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())

			Expect(secondRequestConfig.GetFlags()).To(Equal(uint64(testFlags|1<<5)), "the second hook must receive the first hook's flag change")
			Expect(secondRequestConfig.Params.GetCompression()).NotTo(BeNil())
			Expect(secondRequestConfig.Params.GetCompression().GetValue()).To(Equal("zstd"), "the second hook must receive the first hook's output")
			Expect(flags).To(Equal(testFlags | 1<<5))
			Expect(params.Compression).To(Equal("zstd"))
		})

		It("keeps earlier modifications when a later Ignore-strategy hook fails", func() {
			socketBaseDir := GinkgoT().TempDir()
			firstSocketPath := startSidecar(socketBaseDir, "a-plugin", func(_ context.Context, req *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{
							SetFlags: 1 << 5,
							Params:   &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("zstd")},
						},
					},
				}, nil
			})
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("a-plugin", firstSocketPath, pluginv1alpha1.FailureStrategyFail),
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyIgnore),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags|1<<5), "an ignored failure keeps earlier flag changes")
			Expect(params.Compression).To(Equal("zstd"))
		})

		It("continues past an Ignore-strategy hook failure and applies the later hook's change", func() {
			socketBaseDir := GinkgoT().TempDir()
			failingSocketPath := startSidecar(socketBaseDir, "a-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("zstd")}},
					},
				}, nil
			})

			_, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("a-plugin", failingSocketPath, pluginv1alpha1.FailureStrategyIgnore),
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyFail),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(params.Compression).To(Equal("zstd"))
			Expect(params.CompressionSet).To(BeTrue())
		})

		It("stops at the first Fail-strategy hook", func() {
			socketBaseDir := GinkgoT().TempDir()
			failingSocketPath := startSidecar(socketBaseDir, "a-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})
			secondCalls := 0
			secondSocketPath := startSidecar(socketBaseDir, "b-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				secondCalls++
				return &pluginsv1alpha1.PreMigrationSourceResponse{}, nil
			})

			_, _, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{
				migrationSidecarPlugin("a-plugin", failingSocketPath, pluginv1alpha1.FailureStrategyFail),
				migrationSidecarPlugin("b-plugin", secondSocketPath, pluginv1alpha1.FailureStrategyFail),
			}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(MatchError(ContainSubstring("boom")))
			Expect(secondCalls).To(Equal(0), "the hook after the failing one must not run")
		})

		It("fails the run when a Fail-strategy hook fails", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return nil, fmt.Errorf("boom")
			})

			_, _, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(MatchError(ContainSubstring("boom")))
		})

		It("aborts without calling any sidecar when the context is canceled", func() {
			socketBaseDir := GinkgoT().TempDir()
			calls := 0
			socketPath := startSidecar(socketBaseDir, "a-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				calls++
				return &pluginsv1alpha1.PreMigrationSourceResponse{}, nil
			})

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			_, _, err := applyPreMigrationSourceHooks(ctx, []pluginv1alpha1.Plugin{migrationSidecarPlugin("a-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.Canceled)).To(BeTrue())
			Expect(calls).To(Equal(0))
		})

		It("changes nothing on an empty response", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{}, nil
			})

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags))
			Expect(params).To(Equal(baseParams()))
		})

		It("round-trips the tri-state encoding over a real sidecar connection", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{
							Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{
								Compression:  testOptionalString(""),
								Uri:          testEmptyString(),
								MigrateDisks: testStringList(),
							},
						},
					},
				}, nil
			})

			params := &libvirt.DomainMigrateParameters{
				URI: "unix:///migration-uri", URISet: true,
				Compression: "zstd", CompressionSet: true,
				MigrateDisks: []string{"vda"}, MigrateDisksSet: true,
			}

			flags, merged, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, params, testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags), "the response leaves the flags unset")
			Expect(merged.CompressionSet).To(BeTrue(), "a wrapper holding the empty string sets the parameter to empty")
			Expect(merged.Compression).To(Equal(""))
			Expect(merged.URISet).To(BeFalse(), "an empty wrapper removes the parameter")
			Expect(merged.MigrateDisksSet).To(BeFalse(), "an empty list removes the parameter")
		})
	})

	Context("libvirt conversion", func() {
		DescribeTable("round-trips the libvirt migration configuration",
			func(params libvirt.DomainMigrateParameters) {
				flags := libvirt.DomainMigrateFlags(0xdeadbeef)
				request := libvirtConfigToProto(flags, &params)
				Expect(request.Flags).To(Equal(uint64(flags)))
				mergedFlags, mergedParams, err := mergeLibvirtConfig(0, &libvirt.DomainMigrateParameters{}, &pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: request.Params})
				Expect(err).NotTo(HaveOccurred())
				Expect(mergedFlags).To(Equal(libvirt.DomainMigrateFlags(0)))
				Expect(*mergedParams).To(Equal(params))
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

		DescribeTable("merges a hook response into the current configuration",
			func(current libvirt.DomainMigrateParameters, response *pluginsv1alpha1.LibvirtMigrationConfigChanges, expected libvirt.DomainMigrateParameters, expectedFlags libvirt.DomainMigrateFlags) {
				flags, merged, err := mergeLibvirtConfig(testFlags, &current, response)
				Expect(err).NotTo(HaveOccurred())
				Expect(*merged).To(Equal(expected))
				Expect(flags).To(Equal(expectedFlags))
			},
			Entry("keeps a string parameter the response leaves unset",
				libvirt.DomainMigrateParameters{Compression: "zstd", CompressionSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				libvirt.DomainMigrateParameters{Compression: "zstd", CompressionSet: true},
				testFlags),
			Entry("overrides a string parameter",
				libvirt.DomainMigrateParameters{Compression: "zstd", CompressionSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("xbzrle")}},
				libvirt.DomainMigrateParameters{Compression: "xbzrle", CompressionSet: true},
				testFlags),
			Entry("sets a string parameter to the empty string",
				libvirt.DomainMigrateParameters{Compression: "zstd", CompressionSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testOptionalString("")}},
				libvirt.DomainMigrateParameters{Compression: "", CompressionSet: true},
				testFlags),
			Entry("removes a string parameter with an empty wrapper",
				libvirt.DomainMigrateParameters{Compression: "zstd", CompressionSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Compression: testEmptyString()}},
				libvirt.DomainMigrateParameters{},
				testFlags),
			Entry("keeps an int parameter the response leaves unset",
				libvirt.DomainMigrateParameters{ParallelConnections: 2, ParallelConnectionsSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				libvirt.DomainMigrateParameters{ParallelConnections: 2, ParallelConnectionsSet: true},
				testFlags),
			Entry("overrides an int parameter",
				libvirt.DomainMigrateParameters{ParallelConnections: 2, ParallelConnectionsSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{ParallelConnections: testOptionalInt32(4)}},
				libvirt.DomainMigrateParameters{ParallelConnections: 4, ParallelConnectionsSet: true},
				testFlags),
			Entry("sets an int parameter to zero",
				libvirt.DomainMigrateParameters{ParallelConnections: 2, ParallelConnectionsSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{ParallelConnections: testOptionalInt32(0)}},
				libvirt.DomainMigrateParameters{ParallelConnections: 0, ParallelConnectionsSet: true},
				testFlags),
			Entry("removes an int parameter with an empty wrapper",
				libvirt.DomainMigrateParameters{ParallelConnections: 2, ParallelConnectionsSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{ParallelConnections: testEmptyInt32()}},
				libvirt.DomainMigrateParameters{},
				testFlags),
			Entry("keeps a uint64 parameter the response leaves unset",
				libvirt.DomainMigrateParameters{Bandwidth: 1234, BandwidthSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				libvirt.DomainMigrateParameters{Bandwidth: 1234, BandwidthSet: true},
				testFlags),
			Entry("overrides a uint64 parameter",
				libvirt.DomainMigrateParameters{Bandwidth: 1234, BandwidthSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Bandwidth: testOptionalUint64(5678)}},
				libvirt.DomainMigrateParameters{Bandwidth: 5678, BandwidthSet: true},
				testFlags),
			Entry("sets a uint64 parameter to zero",
				libvirt.DomainMigrateParameters{Bandwidth: 1234, BandwidthSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Bandwidth: testOptionalUint64(0)}},
				libvirt.DomainMigrateParameters{Bandwidth: 0, BandwidthSet: true},
				testFlags),
			Entry("removes a uint64 parameter with an empty wrapper",
				libvirt.DomainMigrateParameters{Bandwidth: 1234, BandwidthSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{Bandwidth: testEmptyUint64()}},
				libvirt.DomainMigrateParameters{},
				testFlags),
			Entry("keeps a list parameter the response leaves unset",
				libvirt.DomainMigrateParameters{MigrateDisks: []string{"vda"}, MigrateDisksSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				libvirt.DomainMigrateParameters{MigrateDisks: []string{"vda"}, MigrateDisksSet: true},
				testFlags),
			Entry("overrides a list parameter",
				libvirt.DomainMigrateParameters{MigrateDisks: []string{"vda"}, MigrateDisksSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{MigrateDisks: testStringList("vdb", "vdc")}},
				libvirt.DomainMigrateParameters{MigrateDisks: []string{"vdb", "vdc"}, MigrateDisksSet: true},
				testFlags),
			Entry("removes a list parameter with an empty list",
				libvirt.DomainMigrateParameters{MigrateDisks: []string{"vda"}, MigrateDisksSet: true},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{Params: &pluginsv1alpha1.LibvirtDomainMigrateParameters{MigrateDisks: testStringList()}},
				libvirt.DomainMigrateParameters{},
				testFlags),
			Entry("keeps the flags the response leaves unset",
				libvirt.DomainMigrateParameters{},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				libvirt.DomainMigrateParameters{},
				testFlags),
			Entry("sets flag bits",
				libvirt.DomainMigrateParameters{},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 5},
				libvirt.DomainMigrateParameters{},
				testFlags|1<<5),
			Entry("clears flag bits",
				libvirt.DomainMigrateParameters{},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{ClearFlags: 0b0010},
				libvirt.DomainMigrateParameters{},
				testFlags&^0b0010),
			Entry("sets and clears different bits together",
				libvirt.DomainMigrateParameters{},
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 5, ClearFlags: 0b0010},
				libvirt.DomainMigrateParameters{},
				(testFlags|1<<5)&^0b0010),
			Entry("changes nothing on an empty response",
				*baseParams(),
				&pluginsv1alpha1.LibvirtMigrationConfigChanges{},
				*baseParams(),
				testFlags),
		)

		It("rejects a response that sets and clears the same flag bit", func() {
			_, _, err := mergeLibvirtConfig(testFlags, &libvirt.DomainMigrateParameters{}, &pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 4, ClearFlags: 1<<4 | 1<<5})
			Expect(err).To(MatchError(ContainSubstring("both set and cleared")))
		})

		It("routes an invalid flag change through the hook's failure strategy", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 5, ClearFlags: 1 << 5},
					},
				}, nil
			})

			flags, params, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyIgnore)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(flags).To(Equal(testFlags), "an ignored invalid change leaves the configuration intact")
			Expect(params).To(Equal(baseParams()))
		})

		It("fails the run when an invalid flag change comes from a Fail-strategy hook", func() {
			socketBaseDir := GinkgoT().TempDir()
			socketPath := startSidecar(socketBaseDir, "test-plugin", func(context.Context, *pluginsv1alpha1.PreMigrationSourceRequest) (*pluginsv1alpha1.PreMigrationSourceResponse, error) {
				return &pluginsv1alpha1.PreMigrationSourceResponse{
					MigrationConfig: &pluginsv1alpha1.PreMigrationSourceResponse_Libvirt{
						Libvirt: &pluginsv1alpha1.LibvirtMigrationConfigChanges{SetFlags: 1 << 5, ClearFlags: 1 << 5},
					},
				}, nil
			})

			_, _, err := applyPreMigrationSourceHooks(context.Background(), []pluginv1alpha1.Plugin{migrationSidecarPlugin("test-plugin", socketPath, pluginv1alpha1.FailureStrategyFail)}, vmi, testFlags, baseParams(), testOpts(socketBaseDir))
			Expect(err).To(MatchError(ContainSubstring("both set and cleared")))
		})

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
