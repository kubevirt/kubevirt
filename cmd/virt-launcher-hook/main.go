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
 *
 */

// Libvirt hook binary for qemu.
// Installed at /etc/libvirt/hooks/qemu.
// Called by libvirt with: domain_name operation sub-operation [extra]
// Domain XML is passed via stdin.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	conntrackHookSock = "/var/run/kubevirt/sockets/conntrack-hook.sock"
	socketTimeout     = 1 * time.Second
)

func main() {
	// Write error logs to container stderr
	if f, err := os.OpenFile("/proc/1/fd/2", os.O_WRONLY|os.O_APPEND, 0); err == nil {
		log.SetOutput(f)
	}
	log.SetFlags(0)
	log.SetPrefix("qemu-hook: ")

	if len(os.Args) < 4 {
		log.Printf("expected at least 3 args, got %d", len(os.Args)-1)
		os.Exit(0)
	}

	operation := os.Args[2]
	subOperation := os.Args[3]

	// "started begin" fires on the destination after migration data transfer
	// completes but before VM resumes. Used to gate conntrack injection.
	if operation == "started" && subOperation == "begin" {
		if err := waitForConntrackSync(); err != nil {
			log.Printf("conntrack sync error: %v", err)
		}
	}
}

func waitForConntrackSync() error {
	if _, err := os.Stat(conntrackHookSock); err != nil {
		return nil
	}

	conn, err := net.DialTimeout("unix", conntrackHookSock, socketTimeout)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(socketTimeout))

	if _, err := conn.Write([]byte("wait\n")); err != nil {
		return fmt.Errorf("failed to send wait: %w", err)
	}

	response, err := readResponse(conn)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}
	if response != "ok" {
		return fmt.Errorf("unexpected response: %s", response)
	}

	return nil
}

func readResponse(conn net.Conn) (string, error) {
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("no response")
}
