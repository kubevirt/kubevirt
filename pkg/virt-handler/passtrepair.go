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
	ll(passtLivbirtDir)
	socketFile, err := findSocketRepairFile(passtLivbirtDir)
	if err != nil {
		return err
	}
	passtRepairArg := passtLivbirtDir
	if socketFile != "" {
		passtRepairArg = socketFile
	}
	fmt.Printf("DEBUG: calling passt-repair with arg %q\n", passtRepairArg)

	cmd := exec.Command("passt-repair", passtRepairArg)
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	cmd.Stdout = os.Stdout

	if err = cmd.Start(); err != nil {
		fmt.Printf("ERROR: could not start passt-repair: %q\n", err)
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
				fmt.Printf("DEBUG: passt-repair done err:<%q>, %s\n", err, repairErrorString)
				ll(passtLivbirtDir)
				os.Stdout.Sync()
				if strings.Contains(repairErrorString, "Connection refused") ||
					strings.Contains(repairErrorString, "No such file or directory") {
					return &UnixDomainSocketError{socket: passtLivbirtDir}
				}
			}
			return nil
		}
	}
}
func findSocketRepairFile(dirPath string) (string, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		fileName := entry.Name()
		fmt.Printf("DEBUG: findSocketRepairFile type %v\n", entry.Type().String())
		if strings.HasSuffix(fileName, ".socket.repair") {
			fullPath := filepath.Join(dirPath, fileName)
			return fullPath, nil
		}
	}
	return "", nil
}

func passtRepair(vmi *v1.VirtualMachineInstance) {
	const passtLibvirtDirRelativeToLauncherSock = "../../libvirt-runtime/qemu/run/passt/"

	laucherSock, err := cmdclient.FindSocketOnHost(vmi)
	if err != nil {
		log.Log.Object(vmi).Error("failed to find launcher cmd socket on host for pod " + err.Error())
	}
	passtLibvirtDir := filepath.Join(laucherSock, passtLibvirtDirRelativeToLauncherSock)

	err = passtRepairInternal(passtLibvirtDir)
	if err != nil {
		fmt.Printf("DEBUG: passt-repair returned error , %q\n", err)
	}

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
