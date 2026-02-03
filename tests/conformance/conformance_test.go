package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDoneFile(t *testing.T) {
	tmpDir, cleanup := setupTest(t, nil)
	defer cleanup()

	writeDoneFile()

	content, err := os.ReadFile(filepath.Join(tmpDir, "done"))
	if err != nil {
		t.Fatalf("failed to read done file: %v", err)
	}
	if string(content) != tmpDir {
		t.Errorf("done file content = %q, want %q", string(content), tmpDir)
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name         string
		executeFn    func() error
		wantExitCode int
	}{
		{
			name:         "success",
			executeFn:    func() error { return nil },
			wantExitCode: 0,
		},
		{
			name:         "failure",
			executeFn:    func() error { return errors.New("test failure") },
			wantExitCode: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTest(t, tt.executeFn)
			defer cleanup()

			exitCode := run()

			if exitCode != tt.wantExitCode {
				t.Errorf("run() = %d, want %d", exitCode, tt.wantExitCode)
			}
			if _, err := os.Stat(filepath.Join(tmpDir, "done")); os.IsNotExist(err) {
				t.Fatal("done file was not created")
			}
		})
	}
}

func TestRunPanic(t *testing.T) {
	tmpDir, cleanup := setupTest(t, func() error { panic("test panic") })
	defer cleanup()

	defer func() {
		if r := recover(); r != nil {
			if _, err := os.Stat(filepath.Join(tmpDir, "done")); os.IsNotExist(err) {
				t.Fatal("done file was not created on panic")
			}
		}
	}()

	_ = run()
	t.Fatal("expected panic did not occur")
}

func setupTest(t *testing.T, execFn func() error) (tmpDir string, cleanup func()) {
	tmpDir = t.TempDir()
	originalResultsDir := resultsDir
	originalExecuteFunc := executeFunc
	originalOutput := output
	resultsDir = tmpDir
	executeFunc = execFn
	output = io.Discard
	return tmpDir, func() {
		resultsDir = originalResultsDir
		executeFunc = originalExecuteFunc
		output = originalOutput
	}
}
