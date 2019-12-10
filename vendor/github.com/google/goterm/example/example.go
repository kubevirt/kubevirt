// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/google/goterm/term"
)

// Constants for IO and greeting.
const (
	file    = "/tmp/goscript-"               // file the base filename of the logfile
	bufSz   = 8192                           // BUFSZ size of the buffers used for IO
	Welcome = "Examplescript up and running" // Welcome Welcome message printed when the application starts
)

// A version of the UNIX command "script" with no errchecking buffered IO or cool features
func main() {
	// Get PTYs up
	pty, _ := term.OpenPTY()
	defer pty.Close()
	// Save the current Stdin attributes
	backupTerm, _ := term.Attr(os.Stdin)
	// Copy attributes
	myTerm := backupTerm
	// Change the Stdin term to RAW so we get everything
	myTerm.Raw()
	myTerm.Set(os.Stdin)
	// Set the backup attributes on our PTY slave
	backupTerm.Set(pty.Slave)
	// Make sure we'll get the attributes back when exiting
	defer backupTerm.Set(os.Stdin)
	// Get the snooping going
	go Snoop(pty)
	// Handle changes in termsize
	sig := make(chan os.Signal, 2)
	// Notify if window size changes or shell dies
	signal.Notify(sig, syscall.SIGWINCH, syscall.SIGCLD)
	// Start up the slaveshell
	cmd := exec.Command(os.Getenv("SHELL"), "")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pty.Slave, pty.Slave, pty.Slave
	cmd.Args = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true}
	cmd.Start()
	// Get the initial winsize
	myTerm.Winsz(os.Stdin)
	myTerm.Winsz(pty.Slave)
	// If the termsize changes , propagate to our PTY
	for {
		switch <-sig {
		case syscall.SIGWINCH:
			myTerm.Winsz(os.Stdin)
			myTerm.Setwinsz(pty.Slave)
		default:
			return
		}
	}
}

// Snoop gets the script file up and running and kicks of the reader and writer functions
func Snoop(pty *term.PTY) {
	// Just something that might be a bit uniqe
	pid := os.Getpid()
	pidcol, _ := term.NewColor256(strconv.Itoa(pid), strconv.Itoa(pid%256), "")
	greet := fmt.Sprintln("\n", term.Green(Welcome), " pid:", pidcol,
		" file:", term.Yellow(file+strconv.Itoa(pid)+"\n"))
	// Our logfile
	file, _ := os.Create(file + strconv.Itoa(pid))
	os.Stdout.Write([]byte(greet))
	go reader(pty.Master, file)
	go writer(pty.Master)
}

// reader reads from master and writes to file and stdout
func reader(master *os.File, log *os.File) {
	var buf = make([]byte, bufSz)
	defer func() {
		log.Sync()
		log.Close()
	}()
	for {
		nr, _ := master.Read(buf)
		os.Stdout.Write(buf[:nr])
		log.Write(buf[:nr])
	}
}

// writer reads from stdin and writes to master
func writer(master *os.File) {
	var buf = make([]byte, bufSz)
	for {
		nr, _ := os.Stdin.Read(buf)
		master.Write(buf[:nr])
	}
}
