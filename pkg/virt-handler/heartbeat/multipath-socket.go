package heartbeat

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/storage/reservation"
	"kubevirt.io/kubevirt/pkg/virt-handler/heartbeat/filewatcher"
	"kubevirt.io/kubevirt/pkg/virt-handler/isolation"
	virt_chroot "kubevirt.io/kubevirt/pkg/virt-handler/virt-chroot"
)

const (
	procRootRun = "/proc/1/root/run"
	socket      = "multipathd.socket"
)

//go:generate mockgen -source $GOFILE -package=$GOPACKAGE -destination=generated_mock_$GOFILE
type mounter interface {
	Mount(sourcePath, targetPath *safepath.Path, ro bool) *exec.Cmd
	Umount(path *safepath.Path) *exec.Cmd
	IsMounted(mountPoint *safepath.Path) (isMounted bool, err error)
}

type mountManager struct{}

func (m *mountManager) Mount(sourcePath, targetPath *safepath.Path, ro bool) *exec.Cmd {
	return virt_chroot.MountChroot(sourcePath, targetPath, ro)
}

func (m *mountManager) Umount(path *safepath.Path) *exec.Cmd {
	return virt_chroot.UmountChroot(path)
}

func (m *mountManager) IsMounted(mountPoint *safepath.Path) (isMounted bool, err error) {
	return isolation.IsMounted(mountPoint)
}

type MonitorMultipathSocket struct {
	Watcher             *filewatcher.FileWatcher
	MultipathSocketPath string
	HostDir             string
	Mounter             mounter
}

func NewMonitorMultipathSocket() *MonitorMultipathSocket {
	multipathSocketPath := filepath.Join(procRootRun, socket)
	return &MonitorMultipathSocket{
		Watcher:             filewatcher.New(multipathSocketPath, 1*time.Second),
		MultipathSocketPath: multipathSocketPath,
		HostDir:             reservation.GetPrHelperHostSocketDir(),
		Mounter:             &mountManager{},
	}
}

func (m *MonitorMultipathSocket) Run() {
	sPath, err := m.createSPath()
	if err != nil {
		return
	}

	// Remove old bind mount if there was one
	if err := m.unmount(sPath); err != nil {
		return
	}
	// Try to mount existing socket and ignore errors
	_ = m.mount(sPath)

	m.Watcher.Run()
	for {
		select {
		case event := <-m.Watcher.Events:
			switch event {
			case filewatcher.Create:
				if err := m.mount(sPath); err != nil {
					continue
				}
			case filewatcher.Remove:
				if err := m.unmount(sPath); err != nil {
					continue
				}
			case filewatcher.InoChange:
				if err := m.unmount(sPath); err != nil {
					continue
				}
				if err := m.mount(sPath); err != nil {
					continue
				}
			}
		case err := <-m.Watcher.Errors:
			if err != nil {
				log.Log.Reason(err).Errorf("Failed monitoring multipath socket")
				m.Watcher.Close()
				return
			}
		}
	}
}

func (m *MonitorMultipathSocket) Stop() {
	m.Watcher.Close()
}

func (m *MonitorMultipathSocket) createSPath() (*safepath.Path, error) {
	if _, err := os.Stat(m.HostDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err := os.MkdirAll(m.HostDir, 0o700); err != nil {
			log.Log.Reason(err).Errorf("failed creating persistent reservation directory %s", m.HostDir)
			return nil, err
		}
		log.Log.Infof("Created persistent reservation directory %s", m.HostDir)
	}

	path := filepath.Join(m.HostDir, socket)
	file, err := os.Create(path)
	if err != nil {
		log.Log.Reason(err).Errorf("failed creating persistent reservation socket %s", path)
		return nil, err
	}
	if err := file.Close(); err != nil {
		log.Log.Reason(err).Errorf("failed closing persistent reservation socket %s", path)
		return nil, err
	}
	log.Log.Infof("Created persistent reservation directory socket %s", path)

	sPath, err := safepath.JoinAndResolveWithRelativeRoot("/", path)
	if err != nil {
		log.Log.Reason(err).Error("failed to create the safepath for the multipath socket mount")
		return nil, err
	}

	return sPath, nil
}

func (m *MonitorMultipathSocket) unmount(sPath *safepath.Path) error {
	if isMounted, err := m.Mounter.IsMounted(sPath); err == nil && isMounted {
		if out, err := m.Mounter.Umount(sPath).CombinedOutput(); err != nil {
			log.Log.Reason(err).Errorf("failed to umount persistent reservation directory %s: %s", sPath, string(out))
			return err
		}
		log.Log.Infof("Removed bind mount for the multipath socket")
	}

	return nil
}

func (m *MonitorMultipathSocket) mount(sPath *safepath.Path) error {
	sMultipathSocket, err := safepath.JoinAndResolveWithRelativeRoot(m.MultipathSocketPath)
	if err != nil {
		log.Log.Reason(err).Errorf("failed to create the safepath for the multipath socket")
		return err
	}

	if out, err := m.Mounter.Mount(sMultipathSocket, sPath, false).CombinedOutput(); err != nil {
		log.Log.Reason(err).Errorf("failed to create the multipath socket bind mount %s: %s", sPath, string(out))
		return err
	}

	log.Log.Infof("Created bind mount for the multipath socket")
	return nil
}
