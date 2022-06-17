package util

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

	"kubevirt.io/client-go/log"

	"kubevirt.io/kubevirt/pkg/util"
)

type VmmWrapper struct {
	user             uint32
	apiSocketPath    string
	eventMonitorConn net.Conn
	cmd              *exec.Cmd
}

func NewCloudHvWrapper(nonRoot bool) *VmmWrapper {
	if nonRoot {
		return &VmmWrapper{
			user: util.NonRootUID,
		}
	}
	return &VmmWrapper{
		user: util.RootUser,
	}
}

func (l *VmmWrapper) CreateCloudHvApiSocket(virtSharedDir string) (string, error) {
	dirPath, err := os.MkdirTemp(virtSharedDir, "")
	if err != nil {
		return "", err
	}

	l.apiSocketPath = filepath.Join(dirPath, "ch.sock")

	return l.apiSocketPath, nil
}

func (l *VmmWrapper) StartCloudHv(stopChan chan struct{}) (err error) {
	//	chPath := "/usr/bin/cloud-hypervisor"
	chPath, err := exec.LookPath("cloud-hypervisor")
	if err != nil {
		return err
	}

	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return err
	}

	f := os.NewFile(uintptr(fds[0]), "")
	defer f.Close()
	eventMonitorConn, err := net.FileConn(f)
	if err != nil {
		return err
	}
	l.eventMonitorConn = eventMonitorConn

	args := []string{
		"--api-socket", l.apiSocketPath,
		"--event-monitor", fmt.Sprintf("fd=%d", fds[1]),
		"-vv",
	}
	cmd := exec.Command(chPath, args...)
	if l.user != 0 {
		log.Log.Infof("Non root user (%d) => applying specific caps", l.user)
		cmd.SysProcAttr = &syscall.SysProcAttr{
			AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE, unix.CAP_SYS_PTRACE},
		}
	}

	// Connect cloud-hypervisor's stderr to our own stdout in order to see
	// the logs in the container logs
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout

	u, err := user.Current()
	if err != nil {
		return err
	}
	log.Log.Infof("Running Cloud Hypervisor with user (uid=%s,gid=%s)", u.Uid, u.Gid)

	err = cmd.Start()
	if err != nil {
		log.Log.Reason(err).Error("failed to start cloud-hypervisor")
		return err
	}

	go func() {
		select {
		case <-stopChan:
			log.Log.Info("Received stopChan -> killing process")
			cmd.Process.Kill()
		}
	}()

	l.cmd = cmd

	return nil
}

func (l *VmmWrapper) WaitCloudHvProcess() error {
	if l.cmd == nil {
		return fmt.Errorf("Missing Cmd to wait for")
	}

	return l.cmd.Wait()
}

func (l *VmmWrapper) EventMonitorConn() net.Conn {
	return l.eventMonitorConn
}
