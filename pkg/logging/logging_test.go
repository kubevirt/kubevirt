package logging

import (
	"path/filepath"
	"runtime"
	"testing"
)

var logCalled bool = false
var logParams []interface{} = nil

type MockLogger struct {
}

func (l MockLogger) Log(params ...interface{}) error {
	logCalled = true
	logParams = append(logParams, params)
	return nil
}

func assert(t *testing.T, condition bool, failMessage string) {
	if !condition {
		_, filePath, lineNumber, _ := runtime.Caller(1)
		fileName := filepath.Base(filePath)
		t.Fatalf("[%s:%d] %s", fileName, lineNumber, failMessage)
	}
}

func compareLog(logLine []interface{}, referenceLine []string) bool {
	if len(logLine) == len(referenceLine) {
		for i, _ := range logLine {
			if logLine[i].(string) != referenceLine[i] {
				return false
			}
		}
		return true
	}
	return false
}

func setUp() {
	defaultComponent = "test"
	logCalled = false
	logParams = make([]interface{}, 0)
}

func tearDown() {

}

func TestDefaultLogLevels(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.Log("default level message")
	assert(t, logCalled, "defualt loglevel should have been info")
	tearDown()
}

// Simply a self-check test as most tests will depend on this behavior working
// Enforces that unit tests are run in isolated contexts
func TestMockLogger(t *testing.T) {
	setUp()
	l := MockLogger{}
	assert(t, !logCalled, "Test Case was not correctly initialized")
	assert(t, len(logParams) == 0, "logParams was not reset")
	assert(t, compareLog(logParams, []string{}), "logParams was not reset")
	l.Log("test", "message")
	assert(t, logCalled, "MockLogger was not called")
	tearDown()
}

func TestBadLevel(t *testing.T) {
	setUp()
	l := Logger("test")
	error := l.SetLogLevel(10)
	assert(t, error != nil, "Allowed to set illegal log level")
	assert(t, l.filterLevel != 10, "Allowed to set illegal log level")
	tearDown()
}

func TestGoodLevel(t *testing.T) {
	setUp()
	l := Logger("test")
	error := l.SetLogLevel(INFO)
	assert(t, error == nil, "Unable to set log level")
	tearDown()
}

func TestComponent(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.Log("foo", "bar")

	assert(t, len(logParams) == 1, "Expected 1 log line")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[1].(string) == "test", "Component was not logged")
	tearDown()
}

func TestDebugCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	assert(t, log.filterLevel == INFO, "Unable to set log level")

	log = log.Debug()
	log.Log("This is a debug message")
	assert(t, !logCalled, "Debug log entry should not have been recorded")

	log = log.Info()
	log.Log("This is an info message")
	assert(t, logCalled, "Info log entry should have been recorded")
	tearDown()
}

func TestInfoCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(WARNING)
	assert(t, log.filterLevel == WARNING, "Unable to set log level")

	log = log.Debug()
	log.Log("This is a debug message")
	assert(t, !logCalled, "Debug log entry should not have been recorded")

	log = log.Info()
	log.Log("This is an info message")
	assert(t, !logCalled, "Info log entry should not have been recorded")

	log = log.Warning()
	log.Log("This is a warning message")
	assert(t, logCalled, "Warning log entry should have been recorded")
	tearDown()
}

func TestVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	if err := log.SetVerbosityLevel(2); err != nil {
		t.Fatal("Unexpected error setting verbosity")
	}
	log.Log("this is a verbosity level 0 message")
	assert(t, logCalled, "Log entry (V=0) should have been recorded")

	logCalled = false
	log = log.V(3)
	log.Log("This is a verbosity level 3 message")
	assert(t, !logCalled, "Log entry (V=3) should not have been recorded")

	// this call should be ignored. repeat last test to prove it
	log = log.V(-1)
	log.Log("This is a verbosity level 3 message")
	assert(t, !logCalled, "Log entry (V=3) should not have been recorded")

	log.V(2).Log("This is a verbosity level 2 message")
	assert(t, logCalled, "Log entry (V=2) should have been recorded")

	// once again, this call should do nothing.
	log = log.V(-1)
	log.Log("This is a verbosity level 2 message")
	assert(t, logCalled, "Log entry (V=2) should have been recorded")
	tearDown()
}

func TestNegativeVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	err := log.SetVerbosityLevel(-1)
	assert(t, err != nil, "Requesting a negative verbosity should not have been allowed")
	tearDown()
}

func TestCachedLoggers(t *testing.T) {
	setUp()
	logger := MockLogger{}
	log := Logger("test")
	log.SetLogger(logger)

	// set a value on this log class
	log.SetLogLevel(ERROR)
	// obtain a new filtered logger and prove it reflects that same log level

	log2 := Logger("test")

	assert(t, log.filterLevel == ERROR, "Log object was not correctly filtered")
	assert(t, log2.filterLevel == ERROR, "Log object was not cached")
	tearDown()
}

func TestWarningCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})

	log.Warning().Log("message", "test warning message")
	assert(t, logCalled, "Warning level message should have been recorded")

	log.Error().Log("error", "test error message")
	assert(t, logCalled, "Error level message should have been recorded")
	tearDown()
}

func TestLogConcurrency(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	// create a new log object from the previous one.
	log2 := log.Warning()
	assert(t, log.currentLogLevel != log2.currentLogLevel, "log and log2 should not have the same log level")
	assert(t, log.currentLogLevel == INFO, "Calling Warning() did not create a new log object")
	tearDown()
}
