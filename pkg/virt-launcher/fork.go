package virtlauncher

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
	utilwait "k8s.io/apimachinery/pkg/util/wait"

	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/network/infraconfigurators"
)

// ForkAndMonitor itself to give qemu an extra grace period to properly terminate
// in case of virt-launcher crashes
func ForkAndMonitor(containerDiskDir string, runAsNonRoot bool) (int, error) {
	defer cleanupContainerDiskDirectory(containerDiskDir)
	defer terminateIstioProxy()
	index := 0
	if runAsNonRoot {
		index = 1
	}
	cmd := exec.Command(os.Args[index], append(os.Args[(index+1):], "--no-fork", "true")...)
	if runAsNonRoot {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			AmbientCaps: []uintptr{unix.CAP_NET_BIND_SERVICE},
		}
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Log.Reason(err).Error("failed to fork virt-launcher")
		return 1, err
	}

	exitStatus := make(chan syscall.WaitStatus, 10)
	sigs := make(chan os.Signal, 10)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGCHLD)
	go func() {
		for sig := range sigs {
			switch sig {
			case syscall.SIGCHLD:
				var wstatus syscall.WaitStatus
				wpid, err := syscall.Wait4(-1, &wstatus, syscall.WNOHANG, nil)
				if err != nil {
					log.Log.Reason(err).Errorf("Failed to reap process %d", wpid)
				}

				// there's a race between cmd.Wait() and syscall.Wait4 when
				// cleaning up the cmd's pid after it exits. This allows us
				// to detect the correct exit code regardless of which wait
				// wins the race.
				if wpid == cmd.Process.Pid {
					exitStatus <- wstatus
				}

			default:
				log.Log.V(3).Log("signalling virt-launcher to shut down")
				err := cmd.Process.Signal(syscall.SIGTERM)
				sig.Signal()
				if err != nil {
					log.Log.Reason(err).Errorf("received signal %s but can't signal virt-launcher to shut down", sig.String())
				}
			}
		}
	}()

	// wait for virt-launcher and collect the exit code
	exitCode := 0
	if err := cmd.Wait(); err != nil {
		select {
		case status := <-exitStatus:
			exitCode = int(status)
		default:
			exitCode = 1
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
			log.Log.Reason(err).Error("dirty virt-launcher shutdown")
		}

	}
	// give qemu some time to shut down in case it survived virt-handler
	// Most of the time we call `qemu-system=* binaries, but qemu-system-* packages
	// are not everywhere available where libvirt and qemu are. There we usually call qemu-kvm
	// which resides in /usr/libexec/qemu-kvm
	pid, _ := FindPid("qemu-system")
	qemuProcessCommandPrefix := "qemu-system"
	if pid <= 0 {
		pid, _ = FindPid("qemu-kvm")
		qemuProcessCommandPrefix = "qemu-kvm"
	}
	if pid > 0 {
		p, err := os.FindProcess(pid)
		if err != nil {
			return 1, err
		}
		// Signal qemu to shutdown
		err = p.Signal(syscall.SIGTERM)
		if err != nil {
			return 1, err
		}
		// Wait for 10 seconds for the qemu process to disappear
		err = utilwait.PollImmediate(1*time.Second, 10*time.Second, func() (bool, error) {
			pid, _ := FindPid(qemuProcessCommandPrefix)
			if pid == 0 {
				return true, nil
			}
			return false, nil
		})
		if err != nil {
			return 1, err
		}
	}
	return exitCode, nil
}

func cleanupContainerDiskDirectory(ephemeralDiskDir string) {
	// Cleanup the content of ephemeralDiskDir, to make sure that all containerDisk containers terminate
	err := removeContents(ephemeralDiskDir)
	if err != nil {
		log.Log.Reason(err).Errorf("could not clean up ephemeral disk directory: %s", ephemeralDiskDir)
	}
}

func removeContents(dir string) error {
	files, err := filepath.Glob(filepath.Join(dir, "*.sock"))
	if err != nil {
		return err
	}
	for _, file := range files {
		err = os.RemoveAll(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func terminateIstioProxy() {
	if istioProxyPresent() {
		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/quitquitquit", infraconfigurators.EnvoyMergedPrometheusTelemetryPort), "", nil)
		if err != nil || resp.StatusCode != http.StatusOK {
			log.Log.Error("Failed to request Istio proxy termination")
		}
	}
}

func istioProxyPresent() bool {
	resp, err := http.Get(fmt.Sprintf("http://localhost:%d/healthz/ready", infraconfigurators.EnvoyHealthCheckPort))
	if err != nil {
		return false
	}
	if resp.Header.Get("server") == "envoy" {
		return true
	}
	return false
}
