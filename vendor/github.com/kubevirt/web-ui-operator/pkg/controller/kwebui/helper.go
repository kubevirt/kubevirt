package kwebui

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

func Unique() string {
	b := make([]byte, 5)
	if _, err := rand.Read(b); err != nil {
		return "abcde"
	}
	return fmt.Sprintf("%X", b)
}

func pipeToLog(pipe io.ReadCloser, name string) {
	buf := make([]byte, 1024, 1024)
	for {
		n, err := pipe.Read(buf[:])
		if n > 0 {
			LogPerLine(name, string(buf[:n]))
		}
		if err != nil {
			if err != io.EOF {
				log.Error(err,  fmt.Sprintf("%s read error", name))
			}
			return
		}
	}
}

func RunCommand(cmd string, args []string, env []string, anonymArgs []string) error {
	command := exec.Command(cmd, args...)
	command.Env = append(os.Environ(), env...)
	stdoutIn,_ := command.StdoutPipe()
	stderrIn,_ := command.StderrPipe()

	err := command.Start()
	if err != nil {
		log.Error(err, fmt.Sprintf("Execution failed: %s %s", cmd, strings.Join(anonymArgs," ")))
		return err
	}
	go pipeToLog(stdoutIn, "stdout")
	go pipeToLog(stderrIn, "stdout")
	err = command.Wait()
	if err != nil {
		log.Error(err, fmt.Sprintf("Execution failed (wait): %s %s", cmd, strings.Join(anonymArgs," ")))
		return err
	}
	return nil
}

func LogPerLine(header string, out string) {
	for _,line := range strings.Split(out, "\n") {
		log.Info(fmt.Sprintf("%s: %s", header, line))
	}
}

func Def(s string, other string, defVal string) string {
	if s == "" {
		if other == "" {
			return defVal
		}
		return other
	}
	return s
}

func RemoveFile(name string) {
	err := os.Remove(name)
	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to remove file: %s", name))
	}
}

func AfterLast(value string, a string) string {
	pos := strings.LastIndex(value, a)
	if pos == -1 {
		return ""
	}
	adjustedPos := pos + len(a)
	if adjustedPos >= len(value) {
		return ""
	}
	return value[adjustedPos:]
}
