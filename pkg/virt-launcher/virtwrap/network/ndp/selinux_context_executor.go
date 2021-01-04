package ndp

import (
	"fmt"
	"os"
	"runtime"

	"github.com/opencontainers/selinux/go-selinux"

	kvselinux "kubevirt.io/kubevirt/pkg/virt-handler/selinux"
)

type ConnectionMaker struct {
	desiredLabel  string
	ifaceName     string
	originalLabel string
	pid           int
}

func NewSELinuxAwareNDPConnectionMaker(pid int, listeningIfaceName string) (*ConnectionMaker, error) {
	desiredLabel, err := kvselinux.GetLabelForPID(pid)
	if err != nil {
		return nil, err
	}
	originalLabel, err := kvselinux.GetLabelForPID(os.Getpid())
	if err != nil {
		return nil, err
	}
	return &ConnectionMaker{
		pid:           pid,
		desiredLabel:  desiredLabel,
		ifaceName:     listeningIfaceName,
		originalLabel: originalLabel,
	}, nil
}

func (sce ConnectionMaker) Generate() (*NDPConnection, error) {
	if kvselinux.IsSELinuxEnabled() {
		if err := sce.setDesiredContext(); err != nil {
			return nil, err
		}
		defer sce.resetContext()
	}
	ndpConn, err := NewNDPConnection(sce.ifaceName)
	if err != nil {
		return nil, fmt.Errorf("failed to execute command in launcher namespace %d: %v", sce.pid, err)
	}
	return ndpConn, nil
}

func (sce ConnectionMaker) setDesiredContext() error {
	runtime.LockOSThread()
	if err := selinux.SetSocketLabel(sce.desiredLabel); err != nil {
		return fmt.Errorf("failed to switch selinux context to %s. Reason: %v", sce.desiredLabel, err)
	}
	return nil
}

func (sce ConnectionMaker) resetContext() error {
	defer runtime.UnlockOSThread()
	return selinux.SetSocketLabel(sce.originalLabel)
}
