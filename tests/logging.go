package tests

import (
	"fmt"
	"regexp"

	"kubevirt.io/kubevirt/pkg/log"
)

var originalLogger *log.FilteredLogger

type logEvent map[string]interface{}

var logEvents []logEvent

type CapturedLogger struct {
}

func (c CapturedLogger) Log(keyvals ...interface{}) error {
	event := logEvent{}
	key := ""
	// Since keyvals is a sequence of two-tuples key, val, key, val...
	// alternate between tracking the key and value...
	for _, val := range keyvals {
		if key == "" {
			key = val.(string)
		} else {
			event[key] = val
			key = ""
		}
	}
	logEvents = append(logEvents, event)
	return nil
}

// Return all logged messages ( map[string]interface{} ) in case
// direct tests on the logs needs to be performed (e.g. count number of events)
func (c CapturedLogger) GetLogs() []logEvent {
	return logEvents
}

// Pattern is a regex.
// Iterate over each log message and compare the "msg" field
// Returns true if any lines match
func (c CapturedLogger) ContainsMsg(pattern string) bool {
	for _, event := range logEvents {
		logMsg := fmt.Sprintf("%v", event["msg"])
		match, err := regexp.MatchString(pattern, logMsg)
		if err != nil {
			panic(fmt.Errorf("unexpected error when matching regex: %v", err))
		}

		if match {
			return true
		}
	}
	return false
}

// Call this function to replace the default stdio logger with an object
// that records each log message in a slice. You must call ResetLogging
// when done in order to ensure unit-test isolation.
func EnableCapturedLogging() CapturedLogger {
	logger := CapturedLogger{}
	originalLogger = log.DefaultLogger()
	logEvents = []logEvent{}

	capLogger := log.MakeLogger(logger)
	capLogger.SetLogLevel(log.DEBUG)
	log.SetDefaultLogger(capLogger)
	return logger
}

func ResetLogging() {
	log.SetDefaultLogger(originalLogger)
}
