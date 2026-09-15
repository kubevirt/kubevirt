package driver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"time"
)

const (
	binary                   = "/usr/bin/passt"
	targetInterfaceParamName = "targetInterface"
)

func (d *Driver) executeBackendDevice(ctx context.Context) error {
	socketPath := path.Join(d.hostPath, "vhost.sock")
	ifaceName, exists := d.opaqueParams[targetInterfaceParamName]
	if !exists {
		return fmt.Errorf("parameter %s not exists", targetInterfaceParamName)
	}

	gwIP, err := ipv4Addr(ifaceName)
	if err != nil {
		return err
	}

	args := []string{
		"--vhost-user",
		"-s", socketPath,
		"--outbound-if4", ifaceName,
		"-g", gwIP,
		"-4",
		"--one-off",
		"-t", "5201",
	}

	cmd := exec.Command(binary, args...)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting %s: %v\n", binary, err)
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Poll until the socket file appears on the filesystem
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "Error checking socket path %s: %v\n", socketPath, err)
			return err
		}

		select {
		case <-ctx.Done():
			err := fmt.Errorf("timed out waiting for socket %s to be created", socketPath)
			fmt.Fprintln(os.Stderr, err)
			return err
		case <-time.After(50 * time.Millisecond):
		}
	}

	if err := os.Chown(socketPath, qemuUID, qemuGID); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to chown %s: %v\n", socketPath, err)
		return err
	}

	return nil
}

func ipv4Addr(ifaceName string) (string, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return "", fmt.Errorf("failed to find interface %s: %w", ifaceName, err)
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return "", fmt.Errorf("failed to get addresses for %s: %w", ifaceName, err)
	}
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		if ip := ipNet.IP.To4(); ip != nil && ip.IsGlobalUnicast() {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no IPv4 address found on %s", ifaceName)
}
