package virthandler

import (
	"bytes"
	goerror "errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/log"

	cmdclient "kubevirt.io/kubevirt/pkg/virt-handler/cmd-client"
)

type UnixDomainSocketError struct {
	socket string
}

func (e *UnixDomainSocketError) Error() string {
	return fmt.Sprintf("Error connection refused from socket %s", e.socket)
}

func passtRepairInternal(passtLivbirtDir string) error {
	var err error
	cmd := exec.Command("passt-repair", passtLivbirtDir)

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	cmd.Stdout = os.Stdout

	ll(passtLivbirtDir)
	if err = cmd.Start(); err != nil {
		fmt.Printf("ERROR: could not start passt-repair: %q", err)
		return err

	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	timeout := time.After(time.Minute)
	for {
		select {
		case <-timeout:
			cmd.Process.Kill()
			return goerror.New("timed out waiting for passt-repair to complete with " + passtLivbirtDir)
		case err = <-done:
			if err != nil {
				repairErrorString := stderrBuf.String()
				fmt.Printf("DEBUG: passt-repair %s,%q, %q\n", passtLivbirtDir, err, repairErrorString)
				if strings.Contains(repairErrorString, "Connection refused") ||
					strings.Contains(repairErrorString, "No such file or directory") {
					return &UnixDomainSocketError{socket: passtLivbirtDir}
				}
			}
			return nil
		}
	}
}

func passtRepair(vmi *v1.VirtualMachineInstance) {
	const passtLibvirtDirRelativeToLauncherSock = "../../libvirt-runtime/qemu/run/passt/"

	laucherSock, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		log.Log.Object(vmi).Error("failed to find launcher cmd socket on host for pod " + err.Error())
	}
	passtLibvirtDir := filepath.Join(laucherSock, passtLibvirtDirRelativeToLauncherSock)

	passtRepairInternal(passtLibvirtDir)

	//for err = passtRepairInternal(passtLibvirtDir); goerror.As(err, &unixDomainSocketError) && count < 100; count++ {
	//	fmt.Printf("DEBUG: passt-repair %s, waiting. count=%d\n", socket, count)
	//	time.Sleep(10 * time.Second)
	//	err = passtRepairInternal(passtLibvirtDir)
	//}
}

func ll(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return
	}
	fmt.Printf("DEBUG: ll\n")
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Println("Error getting info:", err)
			continue
		}

		modTime := info.ModTime().Format("Jan _2 15:04")
		mode := info.Mode()
		size := info.Size()
		name := entry.Name()

		fmt.Printf("%s %10d %s %s\n", mode, size, modTime, name)
	}
}
