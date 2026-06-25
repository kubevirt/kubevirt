/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package log

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
)

var logCalled bool = false

type logEntry struct {
	level         int
	msg           string
	err           error
	keysAndValues []interface{}
}

var logEntries []logEntry

type MockLogSink struct {
	values []interface{}
}

func (m *MockLogSink) Init(_ logr.RuntimeInfo) {}

func (m *MockLogSink) Enabled(_ int) bool { return true }

func (m *MockLogSink) Info(level int, msg string, keysAndValues ...interface{}) {
	logCalled = true
	all := append(append([]interface{}{}, m.values...), keysAndValues...)
	logEntries = append(logEntries, logEntry{level: level, msg: msg, keysAndValues: all})
}

func (m *MockLogSink) Error(err error, msg string, keysAndValues ...interface{}) {
	logCalled = true
	all := append(append([]interface{}{}, m.values...), keysAndValues...)
	logEntries = append(logEntries, logEntry{err: err, msg: msg, keysAndValues: all})
}

func (m *MockLogSink) WithValues(keysAndValues ...interface{}) logr.LogSink {
	newValues := append(append([]interface{}{}, m.values...), keysAndValues...)
	return &MockLogSink{values: newValues}
}

func (m *MockLogSink) WithName(_ string) logr.LogSink {
	return m
}

func mockLogger() logr.Logger {
	return logr.New(&MockLogSink{})
}

func assert(t *testing.T, condition bool, failMessage string) {
	t.Helper()
	if !condition {
		t.Fatalf("%s", failMessage)
	}
}

func setUp() {
	defaultComponent = "test"
	logCalled = false
	logEntries = nil
}

func tearDown() {}

func lastEntry() logEntry {
	return logEntries[len(logEntries)-1]
}

func hasKey(e logEntry, key string) bool {
	for i := 0; i < len(e.keysAndValues)-1; i += 2 {
		if k, ok := e.keysAndValues[i].(string); ok && k == key {
			return true
		}
	}
	return false
}

func getValue(e logEntry, key string) interface{} {
	for i := 0; i < len(e.keysAndValues)-1; i += 2 {
		if k, ok := e.keysAndValues[i].(string); ok && k == key {
			return e.keysAndValues[i+1]
		}
	}
	return nil
}

func TestDefaultLogLevels(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.Log("msg", "default level message")
	assert(t, logCalled, "default loglevel should have been info")
	tearDown()
}

func TestMockLogger(t *testing.T) {
	setUp()
	assert(t, !logCalled, "Test Case was not correctly initialized")
	assert(t, len(logEntries) == 0, "logEntries was not reset")
	l := MakeLogger(mockLogger())
	l.Log("msg", "test message")
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
	log := MakeLogger(mockLogger())
	log.Log("foo", "bar")

	assert(t, len(logEntries) == 1, "Expected 1 log line")
	e := lastEntry()
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	tearDown()
}

func TestWith(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())

	log.With("arg1", "val1").Log("foo1", "bar1")
	log.With("arg2", "val2").Log("foo2", "bar2")

	assert(t, len(logEntries) == 2, "Expected 2 log lines")

	e := logEntries[0]
	assert(t, getValue(e, "arg1").(string) == "val1", "Custom With() field was not logged")

	e = logEntries[1]
	assert(t, getValue(e, "arg2").(string) == "val2", "Custom With() was not logged")
	tearDown()
}

func TestInfoCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(WARNING)
	assert(t, log.filterLevel == WARNING, "Unable to set log level")

	log = log.Level(INFO)
	log.Log("msg", "This is an info message")
	assert(t, !logCalled, "Info log entry should not have been recorded")

	log = log.Level(WARNING)
	log.Log("msg", "This is a warning message")
	assert(t, logCalled, "Warning log entry should have been recorded")
	tearDown()
}

func TestVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())

	assert(t, log.verbosityLevel == 2, "Default verbosity should be 2")

	if err := log.SetVerbosityLevel(3); err != nil {
		t.Fatal("Unexpected error setting verbosity")
	}
	log.Log("msg", "this is a verbosity level 2 message")
	assert(t, logCalled, "Log entry (V=2) should have been recorded")

	logCalled = false
	vLog := log.V(4)
	vLog.Log("msg", "This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")

	logCalled = false
	vLog = vLog.V(-1)
	vLog.Log("msg", "This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")

	logCalled = false
	vLog.V(3).Log("msg", "This is a verbosity level 3 message")
	assert(t, logCalled, "Log entry (V=3) should have been recorded")

	logCalled = false
	vLog = vLog.V(-1)
	vLog.Log("msg", "This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")
	tearDown()
}

func TestNegativeVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	err := log.SetVerbosityLevel(-1)
	assert(t, err != nil, "Requesting a negative verbosity should not have been allowed")
	tearDown()
}

func TestCachedLoggers(t *testing.T) {
	setUp()
	log := Logger("test")
	log.SetLogger(mockLogger())

	log.SetLogLevel(ERROR)
	log2 := Logger("test")

	assert(t, log.filterLevel == ERROR, "Log object was not correctly filtered")
	assert(t, log2.filterLevel == ERROR, "Log object was not cached")
	tearDown()
}

func TestWarningCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())

	log.Level(WARNING).Log("msg", "test warning message")
	assert(t, logCalled, "Warning level message should have been recorded")

	log.Level(ERROR).Log("msg", "test error message")
	assert(t, logCalled, "Error level message should have been recorded")
	tearDown()
}

func TestLogConcurrency(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log2 := log.Level(WARNING)
	assert(t, log.currentLogLevel != log2.currentLogLevel, "log and log2 should not have the same log level")
	assert(t, log.currentLogLevel == INFO, "Calling Warning() did not create a new log object")
	tearDown()
}

func TestInfoMessage(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	log.Level(INFO).Log("msg", "test message")
	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[INFO], "Logged line was not info level")
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	assert(t, e.msg == "test message", "Message was not logged")
	tearDown()
}

func TestWarningMessage(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(WARNING)
	log.Level(WARNING).Log("msg", "test message")
	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[WARNING], "Logged line was not warning level")
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	tearDown()
}

func TestErrorMessage(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(ERROR)
	log.Level(ERROR).Log("msg", "test message")
	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[ERROR], "Logged line was not error level")
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	tearDown()
}

func TestCriticalMessage(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(FATAL)
	log.Level(FATAL).Log("msg", "test message")
	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[FATAL], "Logged line was not fatal level")
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	tearDown()
}

func TestError(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	err := errors.New("Test error")
	log.Level(ERROR).Log("msg", err)
	assert(t, logCalled, "Error was not logged via .Log()")

	logCalled = false
	log.msg(err)
	assert(t, logCalled, "Error was not logged via .msg()")

	logCalled = false
	log.msgf("[%d] %s", 1, err)
	assert(t, logCalled, "Error was not logged via .msgf()")

	logCalled = false
	log.msgf("%s", err)
	assert(t, logCalled, "Error was not logged via .msgf()")
	tearDown()
}

func TestMultipleLevels(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	log.Level(WARNING).Level(INFO).Level(WARNING).msg("test")
	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[WARNING], "Logged line was not of level warning")
	assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
	assert(t, e.msg == "test", "Message was not logged")
	tearDown()
}

func TestLogVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	log.SetVerbosityLevel(2)

	assert(t, log.Verbosity(2), "Verbosity should match")
	assert(t, log.Verbosity(1), "Actual verbosity is higher")
	assert(t, !log.Verbosity(3), "Verbosity is lower")

	assert(t, log.V(2).Verbosity(2), "Verbosity should match")
	assert(t, log.V(2).Verbosity(1), "Actual verbosity is higher")
	assert(t, !log.V(2).Verbosity(3), "Verbosity is lower")

	log.V(2).Log("msg", "test")
	assert(t, logCalled, "V(2) log should have been recorded")
	tearDown()
}

func TestErrWithMsgf(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.Reason(fmt.Errorf("testerror")).msgf("%s", "test")

	e := lastEntry()
	assert(t, getValue(e, "reason").(error).Error() == "testerror", "Reason was not logged")
	assert(t, e.msg == "test", "Message was not logged")
	tearDown()
}

func TestObjectLogging(t *testing.T) {
	tests := []struct {
		name      string
		configure func(log *FilteredLogger)
		logFunc   func(log *FilteredLogger, vm *v1.VirtualMachineInstance)
	}{
		{
			name: "simple",
			logFunc: func(log *FilteredLogger, vm *v1.VirtualMachineInstance) {
				log.Object(vm).Log("msg", "test message")
			},
		},
		{
			name: "change verbosity",
			configure: func(log *FilteredLogger) {
				_ = log.SetVerbosityLevel(6)
			},
			logFunc: func(log *FilteredLogger, vm *v1.VirtualMachineInstance) {
				log.Level(INFO).V(3).Object(vm).Log("msg", "test message")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setUp()
			log := MakeLogger(mockLogger())
			log.SetLogLevel(INFO)
			if tc.configure != nil {
				tc.configure(log)
			}

			vm := v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{Namespace: "test"}}
			tc.logFunc(log, &vm)

			e := lastEntry()
			assert(t, getValue(e, "level").(string) == LogLevelNames[INFO], "Logged line was not of level info")
			assert(t, getValue(e, "component").(string) == "test", "Component was not logged")
			assert(t, hasKey(e, "namespace"), "Logged line did not contain object namespace")
			assert(t, hasKey(e, "name"), "Logged line did not contain object name")
			assert(t, hasKey(e, "kind"), "Logged line did not contain object kind")
			assert(t, hasKey(e, "uid"), "Logged line did not contain UUID")
			tearDown()
		})
	}
}

func TestObjectContextLeakage(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	if err := log.SetVerbosityLevel(0); err != nil {
		t.Fatal(err)
	}

	vm := v1.VirtualMachineInstance{
		ObjectMeta: v12.ObjectMeta{Name: "leak-test"},
	}

	for range 10 {
		log.V(3).Object(&vm).Info("noop")
		log.Object(&vm).V(3).Info("noop")
	}

	assert(t, !logCalled, "Expected no log output during loop")
	assert(t, len(logEntries) == 0, fmt.Sprintf("Expected no log entries during loop, got %d entries", len(logEntries)))

	log.Error("post-check")
	assert(t, logCalled, "Expected post-check log to reach logger")
	assert(t, len(logEntries) == 1, fmt.Sprintf("Expected 1 log entry, got %d", len(logEntries)))
	e := lastEntry()
	for i := 0; i < len(e.keysAndValues)-1; i += 2 {
		if s, ok := e.keysAndValues[i+1].(string); ok {
			assert(t, s != "leak-test", fmt.Sprintf("Unexpected object context on base logger: %v", e.keysAndValues))
		}
	}

	tearDown()
}

func TestInfofVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(mockLogger())
	log.SetLogLevel(INFO)
	log.SetVerbosityLevel(2)

	logCalled = false
	log.V(3).Infof("This should not be logged: verbosity %d", 3)
	assert(t, !logCalled, "V(3).Infof() should not log when verbosity level is 2")

	logCalled = false
	log.V(2).Infof("This should be logged: verbosity %d", 2)
	assert(t, logCalled, "V(2).Infof() should log when verbosity level is 2")

	e := lastEntry()
	assert(t, getValue(e, "level").(string) == LogLevelNames[INFO], "Logged line was not INFO level")

	warningLog := log.Level(WARNING)
	logCalled = false
	logEntries = nil
	warningLog.V(3).Infof("This should not be logged: verbosity %d", 3)
	assert(t, !logCalled, "V(3).Infof() should not log when verbosity level is 2, even after Level(WARNING)")

	tearDown()
}
