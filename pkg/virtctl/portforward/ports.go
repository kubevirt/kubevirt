package portforward

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type forwardedPort struct {
	local    int
	remote   int
	protocol string
}

func parsePorts(args []string) ([]forwardedPort, error) {
	ports := make([]forwardedPort, len(args))

	for i, arg := range args {
		forwardedPort, err := parsePort(arg)
		if err != nil {
			return ports, err
		}
		ports[i] = forwardedPort
	}

	return ports, nil
}

const (
	protocolTCP = "tcp"
	protocolUDP = "udp"
	minPort     = 1
	maxPort     = 65535
)

func isValidPort(port int) bool {
	return port >= minPort && port <= maxPort
}

func parsePort(arg string) (forwardedPort, error) {
	var (
		port = forwardedPort{
			// default to tcp
			protocol: protocolTCP,
		}
		err error
	)

	protocol := strings.Split(arg, "/")
	if len(protocol) > 1 {
		port.protocol = strings.ToLower(protocol[0])
		arg = protocol[1]
	}

	if port.protocol != protocolTCP && port.protocol != protocolUDP {
		return port, fmt.Errorf("unknown protocol: %s, expected tcp or udp", port.protocol)
	}

	if strings.Count(arg, ":") > 1 {
		return port, errors.New("invalid port, expected local[:remote] with at most two ports")
	}

	ports := strings.FieldsFunc(arg, func(r rune) bool {
		return r == ':'
	})
	if len(ports) < 1 {
		return port, errors.New("invalid port, missing local and/or remote port")
	}

	port.local, err = strconv.Atoi(ports[0])
	if err != nil {
		return port, err
	}
	if !isValidPort(port.local) {
		return port, fmt.Errorf("invalid port %d, must be between %d and %d", port.local, minPort, maxPort)
	}
	port.remote = port.local

	if len(ports) > 1 {
		port.remote, err = strconv.Atoi(ports[1])
		if err != nil {
			return port, err
		}
		if !isValidPort(port.remote) {
			return port, fmt.Errorf("invalid port %d, must be between %d and %d", port.remote, minPort, maxPort)
		}
	}

	return port, nil
}
