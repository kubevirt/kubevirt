package main

import (
	"errors"
	"testing"
)

type fakeExecClient struct {
	exitCode int
	stdOut   string
	err      error
}

func (f fakeExecClient) Exec(string, string, []string, int32) (int, string, error) {
	return f.exitCode, f.stdOut, f.err
}

type fakeGuestPingClient struct {
	err error
}

func (f fakeGuestPingClient) GuestPing(string, int32) error {
	return f.err
}

func TestRunGuestAgentProbeReturnsFailureOnPingError(t *testing.T) {
	client := fakeGuestPingClient{err: errors.New("guest agent is unavailable")}

	if exitCode := runGuestAgentProbe(client, "domain", 1); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunGuestAgentProbeReturnsSuccess(t *testing.T) {
	if exitCode := runGuestAgentProbe(fakeGuestPingClient{}, "domain", 1); exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
}

func TestRunExecProbeReturnsFailureOnExecutionError(t *testing.T) {
	client := fakeExecClient{
		exitCode: -1,
		err:      errors.New("guest agent is unavailable"),
	}

	if exitCode := runExecProbe(client, "domain", "command", nil, 1); exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
}

func TestRunExecProbeReturnsGuestExitCode(t *testing.T) {
	client := fakeExecClient{exitCode: 42}

	if exitCode := runExecProbe(client, "domain", "command", nil, 1); exitCode != 42 {
		t.Fatalf("expected guest exit code 42, got %d", exitCode)
	}
}
