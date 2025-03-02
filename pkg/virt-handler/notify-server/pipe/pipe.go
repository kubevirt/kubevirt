package pipe

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"kubevirt.io/client-go/log"

	diskutils "kubevirt.io/kubevirt/pkg/ephemeral-disk-utils"
	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
)

// InjectNotify injects the domain-notify.sock into the VMI pod and listens for connections
func InjectNotify(ctx context.Context, logger *log.FilteredLogger, pod isolation.IsolationResult, virtShareDir string, nonRoot bool) (chan net.Conn, error) {
	root, err := pod.MountRoot()
	if err != nil {
		return nil, err
	}
	socketDir, err := root.AppendAndResolveWithRelativeRoot(virtShareDir)
	if err != nil {
		return nil, err
	}

	listener, err := safepath.ListenUnixNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return nil, fmt.Errorf("failed to create unix socket for proxy service: %w", err)
	}
	socketPath, err := safepath.JoinNoFollow(socketDir, "domain-notify-pipe.sock")
	if err != nil {
		return nil, err
	}

	if nonRoot {
		err := diskutils.DefaultOwnershipManager.SetFileOwnership(socketPath)
		if err != nil {
			return nil, fmt.Errorf("unable to change ownership for domain notify: %w", err)
		}
	}

	fdChan := make(chan net.Conn, 100)
	// Pass connections
	go func(listener net.Listener, fdChan chan net.Conn) {
		defer close(fdChan)
		for {
			fd, err := listener.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					// As Accept blocks, closing it is our mechanism to exit this loop
					return
				}
				logger.Reason(err).Error("Domain pipe accept error encountered.")
				// keep listening until stop invoked
				time.Sleep(1 * time.Second)
			}
			fdChan <- fd
		}
	}(listener, fdChan)

	go func() {
		<-ctx.Done()
		logger.Infof("closing notify pipe listener")
		if err := listener.Close(); err != nil {
			logger.Infof("failed closing notify pipe listener: %v", err)
		}
		logger.Infof("closed notify pipe listener")
	}()

	return fdChan, nil
}
