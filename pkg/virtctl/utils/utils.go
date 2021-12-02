package utils

import (
	"fmt"
	"io"
	"os"
	"os/signal"

	"golang.org/x/crypto/ssh/terminal"
)

// AttachConsole attaches stdin and stdout to the console
// in -> stdinWriter | stdinReader -> console
// out <- stdoutReader | stdoutWriter <- console
func AttachConsole(stdinReader, stdoutReader *io.PipeReader, stdinWriter, stdoutWriter *io.PipeWriter, message string, resChan <-chan error) (err error) {
	stopChan := make(chan struct{}, 1)
	writeStop := make(chan error)
	readStop := make(chan error)
	if terminal.IsTerminal(int(os.Stdin.Fd())) {
		state, err := terminal.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("Make raw terminal failed: %s", err)
		}
		defer terminal.Restore(int(os.Stdin.Fd()), state)
	}
	fmt.Fprint(os.Stderr, message)

	in := os.Stdin
	out := os.Stdout

	go func() {
		interrupt := make(chan os.Signal, 1)
		signal.Notify(interrupt, os.Interrupt)
		<-interrupt
		close(stopChan)
	}()

	go func() {
		_, err := io.Copy(out, stdoutReader)
		readStop <- err
	}()

	go func() {
		defer close(writeStop)
		buf := make([]byte, 1024, 1024)
		for {
			// reading from stdin
			n, err := in.Read(buf)
			if err != nil && err != io.EOF {
				writeStop <- err
				return
			}
			if n == 0 && err == io.EOF {
				return
			}

			// the escape sequence
			if buf[0] == 29 {
				return
			}
			// Writing out to the console connection
			_, err = stdinWriter.Write(buf[0:n])
			if err == io.EOF {
				return
			}
		}
	}()

	select {
	case <-stopChan:
	case err = <-readStop:
	case err = <-writeStop:
	case err = <-resChan:
	}

	return err
}
