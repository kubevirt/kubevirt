package notifymanager

import (
	"context"
	"fmt"
	"net"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	"kubevirt.io/kubevirt/pkg/virt-handler/notify-server/pipe"
	"kubevirt.io/kubevirt/pkg/vmitrait"
)

func NewPipeManager(podIsolationDetector isolation.PodIsolationDetector, virtShareDir string) *pipeManager {
	return &pipeManager{
		podIsolationDetector: podIsolationDetector,
		virtShareDir:         virtShareDir,
	}
}

type pipeManager struct {
	podIsolationDetector isolation.PodIsolationDetector
	virtShareDir         string
}

func (m *pipeManager) StartDomainNotify(domainPipeStopChan <-chan struct{}, vmi *v1.VirtualMachineInstance) error {
	return m.startDomainNotifyPipe(domainPipeStopChan, vmi)
}

func (m *pipeManager) startDomainNotifyPipe(domainPipeStopChan <-chan struct{}, vmi *v1.VirtualMachineInstance) error {
	res, err := m.podIsolationDetector.Detect(vmi)
	if err != nil {
		return fmt.Errorf("failed to detect isolation for launcher pod when setting up notify pipe: %v", err)
	}

	listener, err := pipe.InjectNotify(res, m.virtShareDir, vmitrait.IsNonRoot(vmi))
	if err != nil {
		return err
	}
	ctx := contextFromChan(domainPipeStopChan)
	handleDomainNotifyPipe(ctx, listener, m.virtShareDir, vmi)

	return nil
}

func handleDomainNotifyPipe(ctx context.Context, ln net.Listener, virtShareDir string, vmi *v1.VirtualMachineInstance) {
	logger := log.Log.Object(vmi)
	fdChan := pipe.ChanFromListener(ctx, logger, ln)

	// Process new connections
	// exit when stop encountered
	go pipe.Pipe(ctx, fdChan, func(conn net.Conn) {
		pipe.Proxy(logger, conn, pipe.NewConnectToNotifyFunc(virtShareDir))
	})
}

func contextFromChan(c <-chan struct{}) context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-c
		cancel()
	}()
	return ctx
}
