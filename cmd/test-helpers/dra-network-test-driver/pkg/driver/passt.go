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
	targetPortsParamName     = "targetPorts"
	socketName               = "vhost.sock"
)

func executeBackendDevice(ctx context.Context, hostPath string, opaqueParams map[string]string) error {
	socketPath := path.Join(hostPath, socketName)
	ifaceName, exists := opaqueParams[targetInterfaceParamName]
	if !exists {
		return fmt.Errorf("required parameter %s is not specified", targetInterfaceParamName)
	}

	// passt runs in the node's network namespace, so every claim on a node must
	// forward a distinct set of ports. There is no sensible default.
	targetPorts, exists := opaqueParams[targetPortsParamName]
	if !exists {
		return fmt.Errorf("required parameter %s is not specified", targetPortsParamName)
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
		"-t", targetPorts,
	}

	cmd := exec.Command(binary, args...)

	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start %s: %w", binary, err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Poll until the socket file appears on the filesystem
	for {
		if _, err := os.Stat(socketPath); err == nil {
			break
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to check socket path %s: %w", socketPath, err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for socket %s to be created", socketPath)
		case <-time.After(50 * time.Millisecond):
		}
	}

	if err := os.Chown(socketPath, qemuUID, qemuGID); err != nil {
		return fmt.Errorf("failed to chown %s: %w", socketPath, err)
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
