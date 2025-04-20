/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright The KubeVirt Authors.
 */

package portforward

import (
	"errors"
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
)

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
		port.protocol = protocol[0]
		arg = protocol[1]
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
	port.remote = port.local

	if len(ports) > 1 {
		port.remote, err = strconv.Atoi(ports[1])
		if err != nil {
			return port, err
		}
	}

	return port, nil
}
