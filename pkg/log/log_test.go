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
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sv1 "k8s.io/api/core/v1"

	"kubevirt.io/kubevirt/pkg/api/v1"
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
	assert(t, logCalled, "default loglevel should have been info")
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

func TestInfoCutoff(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(WARNING)
	assert(t, log.filterLevel == WARNING, "Unable to set log level")

	log = log.Level(INFO)
	log.Log("This is an info message")
	assert(t, !logCalled, "Info log entry should not have been recorded")

	log = log.Level(WARNING)
	log.Log("This is a warning message")
	assert(t, logCalled, "Warning log entry should have been recorded")
	tearDown()
}

func TestVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})

	assert(t, log.verbosityLevel == 2, "Default verbosity should be 2")

	if err := log.SetVerbosityLevel(3); err != nil {
		t.Fatal("Unexpected error setting verbosity")
	}
	log.Log("this is a verbosity level 2 message")
	assert(t, logCalled, "Log entry (V=2) should have been recorded")

	logCalled = false
	log = log.V(4)
	log.Log("This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")

	// this call should be ignored. repeat last test to prove it
	logCalled = false
	log = log.V(-1)
	log.Log("This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")

	logCalled = false
	log.V(3).Log("This is a verbosity level 3 message")
	assert(t, logCalled, "Log entry (V=3) should have been recorded")

	// once again, this call should do nothing.
	logCalled = false
	log = log.V(-1)
	log.Log("This is a verbosity level 4 message")
	assert(t, !logCalled, "Log entry (V=4) should not have been recorded")
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

	log.Level(WARNING).Log("message", "test warning message")
	assert(t, logCalled, "Warning level message should have been recorded")

	log.Level(ERROR).Log("error", "test error message")
	assert(t, logCalled, "Error level message should have been recorded")
	tearDown()
}

func TestLogConcurrency(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	// create a new log object from the previous one.
	log2 := log.Level(WARNING)
	assert(t, log.currentLogLevel != log2.currentLogLevel, "log and log2 should not have the same log level")
	assert(t, log.currentLogLevel == INFO, "Calling Warning() did not create a new log object")
	tearDown()
}

func TestInfoMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	log.Level(INFO).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[INFO], "Logged line was not INFO level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestWarningMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(WARNING)
	log.Level(WARNING).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[WARNING], "Logged line was not WARNING level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestErrorMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(ERROR)
	log.Level(ERROR).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[ERROR], "Logged line was not ERROR level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestCriticalMessage(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(CRITICAL)
	log.Level(CRITICAL).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[CRITICAL], "Logged line was not CRITICAL level")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	tearDown()
}

func TestObject(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	vm := v1.VirtualMachineInstance{ObjectMeta: v12.ObjectMeta{Namespace: "test"}}
	log.Object(&vm).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[INFO], "Logged line was not of level INFO")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[4].(string) == "pos", "Logged line was not pos")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	assert(t, logEntry[8].(string) == "namespace", "Logged line did not contain object namespace")
	assert(t, logEntry[10].(string) == "name", "Logged line did not contain object name")
	assert(t, logEntry[12].(string) == "kind", "Logged line did not contain object kind")
	assert(t, logEntry[14].(string) == "uid", "Logged line did not contain UUID")
	tearDown()
}

func TestObjectRef(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	vmRef := &k8sv1.ObjectReference{
		Kind:      "test",
		Name:      "test",
		Namespace: "test",
		UID:       "test",
	}
	log.ObjectRef(vmRef).Log("test", "message")
	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[INFO], "Logged line was not of level INFO")
	assert(t, logEntry[2].(string) == "timestamp", "Logged line is not expected format")
	assert(t, logEntry[4].(string) == "pos", "Logged line was not pos")
	assert(t, logEntry[6].(string) == "component", "Logged line is not expected format")
	assert(t, logEntry[7].(string) == "test", "Component was not logged")
	assert(t, logEntry[8].(string) == "namespace", "Logged line did not contain object namespace")
	assert(t, logEntry[10].(string) == "name", "Logged line did not contain object name")
	assert(t, logEntry[12].(string) == "kind", "Logged line did not contain object kind")
	assert(t, logEntry[14].(string) == "uid", "Logged line did not contain UUID")
	tearDown()
}

func TestError(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	err := errors.New("Test error")
	log.Level(ERROR).Log(err)
	assert(t, logCalled, "Error was not logged via .Log()")

	logCalled = false
	log.msg(err)
	assert(t, logCalled, "Error was not logged via .msg()")

	logCalled = false
	// using more than one parameter in format string
	log.msgf("[%d] %s", 1, err)
	assert(t, logCalled, "Error was not logged via .msgf()")

	logCalled = false
	log.msgf("%s", err)
	assert(t, logCalled, "Error was not logged via .msgf()")
	tearDown()
}

func TestMultipleLevels(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	// change levels more than once
	log.Level(WARNING).Level(INFO).Level(WARNING).msg("test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[0].(string) == "level", "Logged line did not have level entry")
	assert(t, logEntry[1].(string) == logLevelNames[WARNING], "Logged line was not of level WARNING")
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
	log.SetLogLevel(INFO)
	log.SetVerbosityLevel(2)
	log.V(2).Log("msg", "test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	assert(t, strings.HasPrefix(logEntry[5].(string), "log_test.go"), "Logged line referenced wrong module")
	tearDown()
}

func TestMsgVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	log.SetVerbosityLevel(2)
	log.V(2).msg("test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	tearDown()
}

func TestMsgfVerbosity(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.SetLogLevel(INFO)
	log.SetVerbosityLevel(2)
	log.V(2).msgf("%s", "test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[4].(string) == "pos", "Logged line did not contain pos")
	tearDown()
}

func TestErrWithMsgf(t *testing.T) {
	setUp()
	log := MakeLogger(MockLogger{})
	log.Reason(fmt.Errorf("testerror")).msgf("%s", "test")

	logEntry := logParams[0].([]interface{})
	assert(t, logEntry[8].(string) == "reason", "Logged line did not contain message header")
	assert(t, logEntry[9].(error).Error() == "testerror", "Logged line did not contain message header")
	assert(t, logEntry[10].(string) == "msg", "Logged line did not contain message header")
	assert(t, logEntry[11].(string) == "test", "Logged line did not contain message")
	tearDown()
}
