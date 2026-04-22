package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"google.golang.org/grpc"

	vmschema "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"
	hooksInfo "kubevirt.io/kubevirt/pkg/hooks/info"
	hooksV1alpha3 "kubevirt.io/kubevirt/pkg/hooks/v1alpha3"

	"kubevirt.io/kubevirt/cmd/sidecars/network-bpf-bridge-binding/bpfattach"
	"kubevirt.io/kubevirt/cmd/sidecars/network-bpf-bridge-binding/callback"
	"kubevirt.io/kubevirt/cmd/sidecars/network-bpf-bridge-binding/domain"
	"kubevirt.io/kubevirt/cmd/sidecars/network-bpf-bridge-binding/netsetup"
)

type InfoServer struct {
	Version string
}

func (s InfoServer) Info(_ context.Context, _ *hooksInfo.InfoParams) (*hooksInfo.InfoResult, error) {
	return &hooksInfo.InfoResult{
		Name: "network-bpf-bridge-binding",
		Versions: []string{
			s.Version,
		},
		HookPoints: []*hooksInfo.HookPoint{
			{Name: hooksInfo.OnDefineDomainHookPointName, Priority: 0},
			{Name: hooksInfo.ShutdownHookPointName, Priority: 0},
		},
	}, nil
}

type V1alpha3Server struct {
	Done      chan struct{}
	BPFObj    string
	TapName   string
	VethLocal string
	VethPeer  string

	once     sync.Once
	setup    *setupResult
	setupErr error
}

type setupResult struct {
	tapIdx, vethIdx int
}

func (s *V1alpha3Server) ensureNetAndBPF() error {
	s.once.Do(func() {
		obj := s.BPFObj
		if obj == "" {
			obj = filepath.Join("/opt", "network-bpf-bridge-binding", "bpf_bridge.o")
		}
		if _, err := os.Stat(obj); err != nil {
			s.setupErr = fmt.Errorf("BPF object %q: %w", obj, err)
			return
		}
		tap := s.TapName
		if tap == "" {
			tap = netsetup.DefaultTapName
		}
		vL := s.VethLocal
		if vL == "" {
			vL = netsetup.DefaultVethLocal
		}
		vP := s.VethPeer
		if vP == "" {
			vP = netsetup.DefaultVethPeerName
		}

		tapIdx, vethIdx, err := netsetup.EnsureBridgeWiring(tap, vL, vP)
		if err != nil {
			s.setupErr = fmt.Errorf("net setup: %w", err)
			return
		}
		if err := bpfattach.Attach(obj, tap, vL, tapIdx, vethIdx); err != nil {
			s.setupErr = fmt.Errorf("bpf attach: %w", err)
			return
		}
		s.setup = &setupResult{tapIdx: tapIdx, vethIdx: vethIdx}
		log.Log.Infof("bpf-bridge-binding: tap %s idx %d, veth %s idx %d", tap, tapIdx, vL, vethIdx)
	})
	return s.setupErr
}

func (s *V1alpha3Server) OnDefineDomain(_ context.Context, params *hooksV1alpha3.OnDefineDomainParams) (*hooksV1alpha3.OnDefineDomainResult, error) {
	if err := s.ensureNetAndBPF(); err != nil {
		return nil, err
	}

	vmi := &vmschema.VirtualMachineInstance{}
	if err := json.Unmarshal(params.GetVmi(), vmi); err != nil {
		return nil, fmt.Errorf("failed to unmarshal VMI: %v", err)
	}

	tap := s.TapName
	if tap == "" {
		tap = netsetup.DefaultTapName
	}

	cfg, err := domain.NewNetworkConfigurator(vmi.Spec.Domain.Devices.Interfaces, vmi.Spec.Networks, tap)
	if err != nil {
		return nil, err
	}

	newDomainXML, err := callback.OnDefineDomain(params.GetDomainXML(), cfg)
	if err != nil {
		return nil, err
	}

	return &hooksV1alpha3.OnDefineDomainResult{
		DomainXML: newDomainXML,
	}, nil
}

func (s *V1alpha3Server) PreCloudInitIso(_ context.Context, params *hooksV1alpha3.PreCloudInitIsoParams) (*hooksV1alpha3.PreCloudInitIsoResult, error) {
	return &hooksV1alpha3.PreCloudInitIsoResult{
		CloudInitData: params.GetCloudInitData(),
	}, nil
}

func (s *V1alpha3Server) Shutdown(_ context.Context, _ *hooksV1alpha3.ShutdownParams) (*hooksV1alpha3.ShutdownResult, error) {
	tap := s.TapName
	if tap == "" {
		tap = netsetup.DefaultTapName
	}
	vL := s.VethLocal
	if vL == "" {
		vL = netsetup.DefaultVethLocal
	}
	bpfattach.Detach(tap, vL)
	log.Log.Info("Shutdown bpf-bridge network binding")
	select {
	case s.Done <- struct{}{}:
	default:
	}
	return &hooksV1alpha3.ShutdownResult{}, nil
}

func waitForShutdown(server *grpc.Server, errChan <-chan error, shutdownChan <-chan struct{}) {
	signalStopChan := make(chan os.Signal, 1)
	signal.Notify(signalStopChan, os.Interrupt,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)
	var err error
	select {
	case sig := <-signalStopChan:
		log.Log.Infof("bpf-bridge sidecar received signal: %s", sig.String())
	case err = <-errChan:
		log.Log.Reason(err).Error("Failed to run grpc server")
	case <-shutdownChan:
		log.Log.Info("Exiting")
	}

	if err == nil {
		server.GracefulStop()
	}
}

func Serve(server *grpc.Server, socket net.Listener, shutdownChan <-chan struct{}) {
	errChan := make(chan error)
	go func() {
		errChan <- server.Serve(socket)
	}()

	waitForShutdown(server, errChan, shutdownChan)
}
