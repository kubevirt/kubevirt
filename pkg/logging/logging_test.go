package logging

import (
	"errors"
	"kubevirt.io/kubevirt/pkg/api"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var logCalled bool = false
var logParams []interface{} = make([]interface{}, 0)

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
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
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

func TestDebugMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.Debug().Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "DEBUG", "Logged line was not DEBUG level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestInfoMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.Info().Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "INFO", "Logged line was not INFO level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestWarningMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.Warning().Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "WARNING", "Logged line was not WARNING level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestErrorMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.Error().Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "ERROR", "Logged line was not ERROR level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestCriticalMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.Critical().Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "CRITICAL", "Logged line was not CRITICAL level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestObject(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	vm := api.VM{}
	log.Object(&vm).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "INFO", "Logged line was not of level INFO")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[4].(string) == "pos", "Logged line was not pos")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	assert(t, logEntry[8].(string) == "name", "Logged line did not contain object name")
	assert(t, logEntry[10].(string) == "kind", "Logged line did not contain object kind")
	assert(t, logEntry[12].(string) == "uid", "Logged line did not contain UUID")
	tearDown()
}

func TestError(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	err := errors.New("Test error")
	log.Error().Log(err)
	assert(t, logCalled, "Error was not logged via .Log()")

	logCalled = false
	log.Msg(err)
	assert(t, logCalled, "Error was not logged via .Msg()")

	logCalled = false
	// using more than one parameter in format string
	log.Msgf("[%d] %s", 1, err)
	assert(t, logCalled, "Error was not logged via .Msgf()")

	logCalled = false
	log.Msgf("%s", err)
	assert(t, logCalled, "Error was not logged via .Msgf()")
	tearDown()
}

func TestMultipleLevels(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	// change levels more than once
	log.Info().Debug().Info().Msg("test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == "INFO", "Logged line was not of level INFO")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[4].(string) == "pos", "Logged line was not pos")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	assert(t, logEntry[8].(string) == "msg", "Logged line did not contain message header")
	assert(t, logEntry[9].(string) == "test", "Logged line did not contain message")
	tearDown()
}

func TestLogVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.SetVerbosityLevel(2)
	log.V(2).Log("msg", "test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	assert(t, strings.HasPrefix(logEntry[5].(string), "logging_test.go"), "Logged line referenced wrong module")
	tearDown()
}

func TestMsgVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.SetVerbosityLevel(2)
	log.V(2).Msg("test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	assert(t, strings.HasPrefix(logEntry[5].(string), "logging_test.go"), "Logged line referenced wrong module")
	tearDown()
}

func TestMsgfVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(DEBUG)
	log.SetVerbosityLevel(2)
	log.V(2).Msgf("%s", "test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	assert(t, strings.HasPrefix(logEntry[5].(string), "logging_test.go"), "Logged line referenced wrong module")
	tearDown()
}
