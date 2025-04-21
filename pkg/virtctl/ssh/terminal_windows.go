//go:build windows && !excludenative

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

package ssh

import (
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/windows"
	"golang.org/x/term"
)

func setupTerminal() (func(), error) {
	fdIn := int(os.Stdin.Fd())
	fdOut := int(os.Stdout.Fd())
	handleIn := windows.Handle(fdIn)
	handleOut := windows.Handle(fdOut)

	modeIn := uint32(0)
	if err := windows.GetConsoleMode(handleIn, &modeIn); err != nil {
		return nil, err
	}

	modeOut := uint32(0)
	if err := windows.GetConsoleMode(handleOut, &modeOut); err != nil {
		return nil, err
	}

	// Set the same modes as PowerShell/openssh-portable
	// See https://github.com/PowerShell/openssh-portable/blob/latestw_all/contrib/win32/win32compat/console.c#L129
	// For Windows console modes see also https://docs.microsoft.com/en-us/windows/console/setconsolemode
	// Disable unwanted modes
	newModeIn := modeIn &^ (windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT |
		windows.ENABLE_PROCESSED_INPUT | windows.ENABLE_MOUSE_INPUT)
	// Enable wanted modes
	newModeIn |= (windows.ENABLE_WINDOW_INPUT | windows.ENABLE_VIRTUAL_TERMINAL_INPUT)
	newModeOut := modeOut | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING |
		windows.DISABLE_NEWLINE_AUTO_RETURN

	if err := windows.SetConsoleMode(handleIn, newModeIn); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(handleOut, newModeOut); err != nil {
		// Try to restore saved input modes
		windows.SetConsoleMode(handleIn, modeIn)
		return nil, err
	}

	return func() {
		// Restore to initially saved modes
		windows.SetConsoleMode(handleIn, modeIn)
		windows.SetConsoleMode(handleOut, modeOut)
	}, nil
}

func requestPty(session *ssh.Session) error {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		return err
	}

	// Do the same as PowerShell/openssh-portable
	// See https://github.com/PowerShell/openssh-portable/blob/latestw_all/contrib/win32/win32compat/wmain_common.c#L58
	term := os.Getenv("TERM")
	if term == "" {
		term = "xterm-256color"
	}

	if err := session.RequestPty(
		term,
		h, w,
		ssh.TerminalModes{},
	); err != nil {
		return err
	}

	return nil
}
