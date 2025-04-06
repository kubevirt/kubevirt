package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var containerTag = ""

const resultsDir = "/tmp/sonobuoy/results"

func main() {
	err := execute()
	if err != nil {
		fmt.Printf("Failed to execute conformance suite: %v\n", err)
		os.Exit(1)
	}

	const writeFilePerms = 0o666
	err = os.WriteFile(fmt.Sprintf("%s/done", resultsDir), []byte(strings.Join([]string{resultsDir}, "\n")), writeFilePerms)
	if err != nil {
		fmt.Printf("Failed to notify sonobuoy that I am done: %v\n", err)
		os.Exit(1)
	}
}

func execute() error {
	args := []string{"--container-tag", containerTag, "--junit-output", fmt.Sprintf("%s/junit.xml", resultsDir)}
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
		args = append(args, "--ginkgo.focus", "\\[Conformance\\]")
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
	cmd.Env = append(cmd.Env, fmt.Sprintf("ARTIFACTS=%s", resultsDir))
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	return cmd.Run()
}
