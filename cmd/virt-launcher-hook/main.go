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
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	conntrackHookSock = "/var/run/kubevirt/sockets/conntrack-hook.sock"
	socketTimeout     = 1 * time.Second

	qemuConfPath             = "/etc/libvirt/qemu.conf"
	envSharedFilesystemPaths = "SHARED_FILESYSTEM_PATHS"
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

	if operation == "migrate" && subOperation == "begin" {
		runMigrateBeginHook(os.Stdin, os.Stdout)
		return
	}

	// "started begin" fires on the destination after migration data transfer
	// completes but before VM resumes. Used to gate conntrack injection.
	if operation == "started" && subOperation == "begin" {
		if err := waitForConntrackSync(); err != nil {
			log.Printf("conntrack sync error: %v", err)
		}
		return
	}
}

// runMigrateBeginHook reads the original XML from stdin, attempts to inject
// per-disk seclabel relabel='no' for shared filesystem PVCs, and writes either
func runMigrateBeginHook(in io.Reader, out io.Writer) {
	sharedPaths := getSharedFilesystemPaths()
	if len(sharedPaths) == 0 {
		if _, err := io.Copy(out, in); err != nil {
			log.Printf("passthrough copy failed: %v", err)
		}
		return
	}

	xmlBytes, err := io.ReadAll(in)
	if err != nil {
		log.Printf("read stdin failed: %v", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("seclabel injection panicked: %v; passing through original XML", r)
			_, _ = out.Write(xmlBytes)
		}
	}()

	if err := injectSharedDiskSeclabels(xmlBytes, sharedPaths, out); err != nil {
		log.Printf("seclabel injection error: %v; passing through original XML", err)
		_, _ = out.Write(xmlBytes)
	}
}

// sourceLineRE matches a libvirt disk <source file="X"/> or <source file="X"></source>
// (single-quoted variants too) with no child elements — required so we never
// double-inject a seclabel into a source that already has one.
var sourceLineRE = regexp.MustCompile(`<source\s+file=["']([^"']+)["']\s*((?:/>)|(?:></source>))`)

func injectSharedDiskSeclabels(xmlBytes []byte, sharedPaths []string, out io.Writer) error {
	matches := sourceLineRE.FindAllSubmatchIndex(xmlBytes, -1)
	if len(matches) == 0 {
		_, err := out.Write(xmlBytes)
		return err
	}

	var buf bytes.Buffer
	buf.Grow(len(xmlBytes) + len(matches)*64)

	last := 0
	for _, m := range matches {
		matchStart, matchEnd := m[0], m[1]
		path := string(xmlBytes[m[2]:m[3]])

		buf.Write(xmlBytes[last:matchStart])
		if pathInShared(path, sharedPaths) {
			fmt.Fprintf(&buf, `<source file="%s"><seclabel model='dac' relabel='no'/></source>`, path)
		} else {
			buf.Write(xmlBytes[matchStart:matchEnd])
		}
		last = matchEnd
	}
	buf.Write(xmlBytes[last:])

	_, err := out.Write(buf.Bytes())
	return err
}

func getSharedFilesystemPaths() []string {
	var raw []string
	if env := os.Getenv(envSharedFilesystemPaths); env != "" {
		raw = splitNonEmpty(env, ":")
	} else {
		raw = parseSharedFilesystemsFromQemuConf(qemuConfPath)
	}
	for i, p := range raw {
		raw[i] = filepath.Clean(p)
	}
	return raw
}

func splitNonEmpty(s, sep string) []string {
	var out []string
	for part := range strings.SplitSeq(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

var sharedFsConfRE = regexp.MustCompile(`(?m)^\s*shared_filesystems\s*=\s*\[\s*([^\]]*)\]`)

func parseSharedFilesystemsFromQemuConf(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	m := sharedFsConfRE.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(m[1], ",") {
		part = strings.Trim(strings.TrimSpace(part), `"'`)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func pathInShared(path string, sharedPaths []string) bool {
	p := filepath.Clean(path)
	for _, sp := range sharedPaths {
		if p == sp || strings.HasPrefix(p, sp+string(filepath.Separator)) {
			return true
		}
	}
	return false
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
