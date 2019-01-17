package utils

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// RunGoCLICommand egecutes gocli with given args
func RunGoCLICommand(cliPath string, args ...string) error {
	var outBuf, errBuf bytes.Buffer
	cmd := exec.Command(cliPath, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		wd, _ := os.Getwd()
		fmt.Fprintf(os.Stderr, "Working dir: %s\n", wd)
		fmt.Fprintf(os.Stderr, "GoCLI standard output\n%s\n", outBuf.String())
		fmt.Fprintf(os.Stderr, "GoCLI error output\n%s\n", errBuf.String())
		return err
	}

	capture := false
	returnBuf := bytes.NewBuffer(nil)
	scanner := bufio.NewScanner(&outBuf)
	for scanner.Scan() {
		t := scanner.Text()
		if !capture {
			if strings.Contains(t, "Connected to tcp://192.168.66.") {
				capture = true
			}
			continue
		}
		_, err = returnBuf.Write([]byte(t))
		if err != nil {
			return err
		}
	}
	return nil
}
