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

//go:build !windows && !excludenative

package ssh

import (
	"encoding/binary"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

func setupTerminal() (func(), error) {
	fd := int(os.Stdin.Fd())

	state, err := term.MakeRaw(fd)
	if err != nil {
		return nil, err
	}

	return func() { term.Restore(fd, state) }, nil
}

func requestPty(session *ssh.Session) error {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	if err := session.RequestPty(
		os.Getenv("TERM"),
		h, w,
		ssh.TerminalModes{},
	); err != nil {
		return err
	}

	go resizeSessionOnWindowChange(session, os.Stdin.Fd())

	return nil
}

// resizeSessionOnWindowChange watches for SIGWINCH and refreshes the session with the new window size
func resizeSessionOnWindowChange(session *ssh.Session, _ uintptr) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGWINCH)

	for range sigs {
		session.SendRequest("window-change", false, windowSizePayloadFor())
	}
}

func windowSizePayloadFor() []byte {
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return buildWindowSizePayload(80, 24)
	}

	return buildWindowSizePayload(w, h)
}

func buildWindowSizePayload(width, height int) []byte {
	size := make([]byte, 16)
	binary.BigEndian.PutUint32(size, uint32(width))
	binary.BigEndian.PutUint32(size[4:], uint32(height))
	return size
}
