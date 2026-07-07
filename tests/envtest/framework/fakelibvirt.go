package framework

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"google.golang.org/grpc"
	"k8s.io/client-go/tools/cache"

	virtv1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
	"kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/info"
	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/arch"
	convertertypes "kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/converter/types"
	grpcutil "kubevirt.io/kubevirt/pkg/util/net/grpc"
)

// FakeDomain stores the domain XML and state.
type FakeDomain struct {
	Name string
	XML  string
}

// FakeLibvirt runs a gRPC server implementing the virt-launcher
// cmd-server protocol. When SyncVirtualMachine is called, the real
// converter produces domain XML which is stored in memory.
//
// When enabled via WithFakeLibvirt(), a VMI watcher automatically
// calls SyncVirtualMachine when VMIs reach the Scheduled phase,
// so tests can simply wait for the domain to appear.
type FakeLibvirt struct {
	mu      sync.RWMutex
	domains map[string]*FakeDomain

	socketPath  string
	socketDir   string
	grpcServer  *grpc.Server
	vmiInformer cache.SharedIndexInformer
	virtClient  kubecli.KubevirtClient
	handlerReg  cache.ResourceEventHandlerRegistration

	synced   map[string]bool
	syncedMu sync.Mutex
}

func newFakeLibvirt(socketDir string, vmiInformer cache.SharedIndexInformer, virtClient kubecli.KubevirtClient) *FakeLibvirt {
	return &FakeLibvirt{
		domains:     make(map[string]*FakeDomain),
		synced:      make(map[string]bool),
		socketDir:   socketDir,
		socketPath:  filepath.Join(socketDir, "fake-launcher.sock"),
		vmiInformer: vmiInformer,
		virtClient:  virtClient,
	}
}

// LookupDomain returns the domain with the given name, or nil.
func (f *FakeLibvirt) LookupDomain(name string) *FakeDomain {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.domains[name]
}

func (f *FakeLibvirt) defineDomain(name, domainXML string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.domains[name] = &FakeDomain{Name: name, XML: domainXML}
}

// Start creates the gRPC server and registers a VMI watcher that
// automatically calls SyncVirtualMachine when VMIs reach Scheduled.
func (f *FakeLibvirt) Start() error {
	sock, err := grpcutil.CreateSocket(f.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create fake launcher socket: %w", err)
	}

	f.grpcServer = grpc.NewServer()
	cmdv1.RegisterCmdServer(f.grpcServer, &fakeCmdServer{libvirt: f})
	info.RegisterCmdInfoServer(f.grpcServer, &fakeInfoServer{})

	go f.grpcServer.Serve(sock)

	reg, err := f.vmiInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { f.onVMIEvent(obj) },
		UpdateFunc: func(_, obj interface{}) { f.onVMIEvent(obj) },
	})
	if err != nil {
		return fmt.Errorf("failed to add VMI event handler: %w", err)
	}
	f.handlerReg = reg
	return nil
}

func (f *FakeLibvirt) Stop() {
	if f.handlerReg != nil {
		f.vmiInformer.RemoveEventHandler(f.handlerReg)
	}
	if f.grpcServer != nil {
		f.grpcServer.Stop()
	}
	os.Remove(f.socketPath)
	os.RemoveAll(f.socketDir)
}

func (f *FakeLibvirt) onVMIEvent(obj interface{}) {
	vmi, ok := obj.(*virtv1.VirtualMachineInstance)
	if !ok || vmi.Status.Phase != virtv1.Scheduled {
		return
	}

	key := vmi.Namespace + "/" + vmi.Name
	f.syncedMu.Lock()
	if f.synced[key] {
		f.syncedMu.Unlock()
		return
	}
	f.synced[key] = true
	f.syncedMu.Unlock()

	go f.syncVMI(vmi)
}

func (f *FakeLibvirt) syncVMI(vmi *virtv1.VirtualMachineInstance) {
	client, err := cmdclient.NewClient(f.socketPath)
	if err != nil {
		return
	}
	defer client.Close()
	client.SyncVirtualMachine(vmi, nil)
}

// fakeCmdServer implements cmdv1.CmdServer. SyncVirtualMachine runs
// the real converter; all other methods return success no-ops.
type fakeCmdServer struct {
	libvirt *FakeLibvirt
}

func (s *fakeCmdServer) SyncVirtualMachine(_ context.Context, request *cmdv1.VMIRequest) (*cmdv1.Response, error) {
	response := &cmdv1.Response{Success: true}

	var vmi virtv1.VirtualMachineInstance
	if err := json.Unmarshal(request.Vmi.VmiJson, &vmi); err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("failed to unmarshal VMI: %v", err)
		return response, nil
	}

	domain := &api.Domain{}
	c := &convertertypes.ConverterContext{
		AllowEmulation: true,
		SMBios:         &cmdv1.SMBios{},
		Architecture:   arch.NewConverter("amd64"),
	}
	if err := converter.Convert_v1_VirtualMachineInstance_To_api_Domain(&vmi, domain, c); err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("failed to convert VMI to domain: %v", err)
		return response, nil
	}

	api.NewDefaulter(c.Architecture.GetArchitecture()).SetObjectDefaults_Domain(domain)

	domainXML, err := xml.MarshalIndent(domain.Spec, "", "  ")
	if err != nil {
		response.Success = false
		response.Message = fmt.Sprintf("failed to marshal domain XML: %v", err)
		return response, nil
	}

	s.libvirt.defineDomain(domain.Spec.Name, string(domainXML))
	return response, nil
}

func (s *fakeCmdServer) GetDomain(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainResponse, error) {
	s.libvirt.mu.RLock()
	defer s.libvirt.mu.RUnlock()
	for _, d := range s.libvirt.domains {
		domain := &api.Domain{}
		domain.Spec.Name = d.Name
		domain.Status.Status = api.Running
		domJSON, _ := json.Marshal(domain)
		return &cmdv1.DomainResponse{
			Response: &cmdv1.Response{Success: true},
			Domain:   string(domJSON),
		}, nil
	}
	return &cmdv1.DomainResponse{
		Response: &cmdv1.Response{Success: true},
	}, nil
}

func (s *fakeCmdServer) Ping(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.Response, error) {
	return &cmdv1.Response{Success: true}, nil
}

// No-op implementations for the rest of CmdServer
func (s *fakeCmdServer) PauseVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) UnpauseVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) FreezeVirtualMachine(_ context.Context, _ *cmdv1.FreezeRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) UnfreezeVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) ResetVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) SoftRebootVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) ShutdownVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) KillVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) DeleteVirtualMachine(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) MigrateVirtualMachine(_ context.Context, _ *cmdv1.MigrationRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) SyncMigrationTarget(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) CancelVirtualMachineMigration(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) SignalTargetPodCleanup(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) FinalizeVirtualMachineMigration(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) HotplugHostDevices(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) GetDomainStats(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DomainStatsResponse, error) { return &cmdv1.DomainStatsResponse{}, nil }
func (s *fakeCmdServer) GetGuestInfo(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestInfoResponse, error) { return &cmdv1.GuestInfoResponse{}, nil }
func (s *fakeCmdServer) GetUsers(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestUserListResponse, error) { return &cmdv1.GuestUserListResponse{}, nil }
func (s *fakeCmdServer) GetFilesystems(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.GuestFilesystemsResponse, error) { return &cmdv1.GuestFilesystemsResponse{}, nil }
func (s *fakeCmdServer) Exec(_ context.Context, _ *cmdv1.ExecRequest) (*cmdv1.ExecResponse, error) { return &cmdv1.ExecResponse{}, nil }
func (s *fakeCmdServer) GuestPing(_ context.Context, _ *cmdv1.GuestPingRequest) (*cmdv1.GuestPingResponse, error) { return &cmdv1.GuestPingResponse{}, nil }
func (s *fakeCmdServer) VirtualMachineMemoryDump(_ context.Context, _ *cmdv1.MemoryDumpRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) GetQemuVersion(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.QemuVersionResponse, error) { return &cmdv1.QemuVersionResponse{Version: "8.0.0"}, nil }
func (s *fakeCmdServer) SyncVirtualMachineCPUs(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) SyncVirtualMachineMemory(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) GetSEVInfo(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.SEVInfoResponse, error) { return &cmdv1.SEVInfoResponse{}, nil }
func (s *fakeCmdServer) GetLaunchMeasurement(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.LaunchMeasurementResponse, error) { return &cmdv1.LaunchMeasurementResponse{}, nil }
func (s *fakeCmdServer) InjectLaunchSecret(_ context.Context, _ *cmdv1.InjectLaunchSecretRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) GetDomainDirtyRateStats(_ context.Context, _ *cmdv1.EmptyRequest) (*cmdv1.DirtyRateStatsResponse, error) { return &cmdv1.DirtyRateStatsResponse{}, nil }
func (s *fakeCmdServer) GetScreenshot(_ context.Context, _ *cmdv1.VMIRequest) (*cmdv1.ScreenshotResponse, error) { return &cmdv1.ScreenshotResponse{}, nil }
func (s *fakeCmdServer) BackupVirtualMachine(_ context.Context, _ *cmdv1.BackupRequest) (*cmdv1.Response, error) { return &cmdv1.Response{Success: true}, nil }
func (s *fakeCmdServer) RedefineCheckpoint(_ context.Context, _ *cmdv1.RedefineCheckpointRequest) (*cmdv1.RedefineCheckpointResponse, error) { return &cmdv1.RedefineCheckpointResponse{}, nil }
func (s *fakeCmdServer) GetVMStats(_ context.Context, _ *cmdv1.VMStatsRequest) (*cmdv1.VMStatsResponse, error) { return &cmdv1.VMStatsResponse{}, nil }

type fakeInfoServer struct{}

func (s *fakeInfoServer) Info(_ context.Context, _ *info.CmdInfoRequest) (*info.CmdInfoResponse, error) {
	return &info.CmdInfoResponse{
		SupportedCmdVersions: []uint32{cmdv1.CmdVersion},
	}, nil
}
