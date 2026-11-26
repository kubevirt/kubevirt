package main

import (
	"bytes"
	"testing"
)

func TestWriteExecOutputKeepsStreamsSeparate(t *testing.T) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	writeExecOutput(&stdOut, &stdErr, "probe output\n", "probe error\n")

	if stdOut.String() != "probe output\n" {
		t.Fatalf("unexpected stdout: %q", stdOut.String())
	}
	if stdErr.String() != "probe error\n" {
		t.Fatalf("unexpected stderr: %q", stdErr.String())
	}
}

func TestWriteExecOutputDoesNotAddNewlines(t *testing.T) {
	var stdOut bytes.Buffer
	var stdErr bytes.Buffer

	writeExecOutput(&stdOut, &stdErr, "output", "error")

	if stdOut.String() != "output" {
		t.Fatalf("unexpected stdout: %q", stdOut.String())
	}
	if stdErr.String() != "error" {
		t.Fatalf("unexpected stderr: %q", stdErr.String())
	}
}
