package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var containerTag = ""

func done(files []string) {
	err := os.WriteFile("/tmp/sonobuoy/results/done", []byte(strings.Join(files, "\n")), 0666)
	if err != nil {
		fmt.Printf("Failed to notify sonobuoy that I am done: %v\n", err)
	}
}

func main() {
	err := execute()
	done([]string{
		"/tmp/sonobuoy/results",
	})
	if err != nil {
		os.Exit(1)
	}
}

func execute() error {
	args := []string{}
	args = append(args, "--container-tag", containerTag)
	args = append(args, "--junit-output", "/tmp/sonobuoy/results/junit.xml")
	// additional conformance test overrides
	if value, exists := os.LookupEnv("E2E_SKIP"); exists {
		args = append(args, "--ginkgo.skip", value)
	} else {
		args = append(args, "--ginkgo.skip", "\\[Disruptive\\]")
	}

	if value, exists := os.LookupEnv("E2E_LABEL"); exists {
		args = append(args, "--ginkgo.label-filter", value)
	} else if value, exists := os.LookupEnv("E2E_FOCUS"); exists {
		args = append(args, "--ginkgo.focus", value)
	} else {
		args = append(args, "--ginkgo.label-filter", "(conformance)")
	}

	if value, exists := os.LookupEnv("CONTAINER_PREFIX"); exists {
		args = append(args, "--container-prefix", value)
	}
	if value, exists := os.LookupEnv("CONTAINER_TAG"); exists {
		args = append(args, "--container-tag", value)
	}

	args = append(args, "--config", "/conformance-config.json")

	cmd := exec.Command("/usr/bin/go_default_test", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "ARTIFACTS=/tmp/sonobuoy/results/")
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		fmt.Printf("command failed with %v\n", err)
		return err
	}
	return nil
}
