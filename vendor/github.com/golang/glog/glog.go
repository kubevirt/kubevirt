// Go support for leveled logs, analogous to https://code.google.com/p/google-glog/
//
// Copyright 2013 Google Inc. All Rights Reserved.
// Copyright 2018 The KubeVirt Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package reimplements github.com/golang/glog and logs to a structured logger.
package glog

import (
	"bytes"
	"flag"
	"fmt"
	stdLog "log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	log2 "github.com/go-kit/kit/log"
)

type logLevel int32

const (
	infoLevel logLevel = iota
	warningLevel
	errorLevel
	fatalLevel
)

var logLevelNames = map[logLevel]string{
	infoLevel:    "info",
	warningLevel: "warning",
	errorLevel:   "error",
	fatalLevel:   "fatal",
}

var glogVerbosity string
var glogComponent string
var toStderr bool
var logger log2.Logger

func init() {
	flag.StringVar(&glogVerbosity, "v", "2", "log level for V logs")
	flag.StringVar(&glogComponent, "component", "", "logger component")
	flag.BoolVar(&toStderr, "logtostderr", false, "log to standard error instead of files")
	logger = log2.NewJSONLogger(os.Stderr)
}

func severityByName(s string) (logLevel, bool) {
	s = strings.ToLower(s)
	for i, name := range logLevelNames {
		if name == s {
			return logLevel(i), true
		}
	}
	return 0, false
}

// OutputStats tracks the number of output lines and bytes written.
type OutputStats struct {
	lines int64
	bytes int64
}

// Lines returns the number of lines written.
func (s *OutputStats) Lines() int64 {
	return atomic.LoadInt64(&s.lines)
}

// Bytes returns the number of bytes written.
func (s *OutputStats) Bytes() int64 {
	return atomic.LoadInt64(&s.bytes)
}

// Stats tracks the number of lines of output and number of bytes
// per severity level. Values must be read with atomic.LoadInt64.
var Stats struct {
	Info, Warning, Error OutputStats
}

// level is exported because it appears in the arguments to V and is
// the type of the glogVerbosity flag, which can be set programmatically.
// It's a distinct type because we want to discriminate it from logType.
// Variables of type level are only changed under logging.mu.
// The -glogVerbosity flag is read only with atomic ops, so the state of the logging
// module is consistent.

// level is treated as a sync/atomic int32.

// level specifies a level of glogVerbosity for V logs. *level implements
// flag.Value; the -glogVerbosity flag is of type level and should be modified
// only through the flag.Value interface.
type Level int32

// get returns the value of the level.
func (l *Level) get() Level {
	return Level(atomic.LoadInt32((*int32)(l)))
}

// set sets the value of the level.
func (l *Level) set(val Level) {
	atomic.StoreInt32((*int32)(l), int32(val))
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Get is part of the flag.Value interface.
func (l *Level) Get() interface{} {
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	l.set(Level(v))
	return nil
}

// Flush flushes all pending log I/O.
func Flush() {
	// Does nothing, only here for compatibility
}

// CopyStandardLogTo arranges for messages written to the Go "log" package's
// default logs to also appear in the Google logs for the named and lower
// severities.  Subsequent changes to the standard log's default output location
// or format may break this behavior.
//
// Valid names are "infoLevel", "warningLevel", "errorLevel", and "fatalLevel".  If the name is not
// recognized, CopyStandardLogTo panics.
func CopyStandardLogTo(name string) {
	sev, ok := severityByName(name)
	if !ok {
		panic(fmt.Sprintf("log.CopyStandardLogTo(%q): unrecognized severity name", name))
	}
	// Set a log format that captures the user's file and line:
	//   d.go:23: message
	stdLog.SetFlags(stdLog.Lshortfile)
	stdLog.SetOutput(logBridge(sev))
}

// logBridge provides the Write method that enables CopyStandardLogTo to connect
// Go's standard logs to the logs provided by this package.
type logBridge logLevel

// Write parses the standard logging line and passes its components to the
// logger for severity(lb).
func (lb logBridge) Write(b []byte) (n int, err error) {
	var (
		file = "???"
		line = 1
		text string
	)
	// Split "d.go:23: message" into "d.go", "23", and "message".
	if parts := bytes.SplitN(b, []byte{':'}, 3); len(parts) != 3 || len(parts[0]) < 1 || len(parts[2]) < 1 {
		text = fmt.Sprintf("bad log format: %s", b)
	} else {
		file = string(parts[0])
		text = string(parts[2][1:]) // skip leading space
		line, err = strconv.Atoi(string(parts[1]))
		if err != nil {
			text = fmt.Sprintf("bad line number: %s", b)
			line = 1
		}
	}

	doLogPos(logLevel(lb), file, line, text)
	return len(b), nil
}

// Verbose is a boolean type that implements Infof (like Printf) etc.
// See the documentation of V for more information.
type Verbose bool

// V reports whether glogVerbosity at the call site is at least the requested level.
// The returned value is a boolean of type Verbose, which implements Info, Infoln
// and Infof. These methods will write to the Info log if called.
// Thus, one may write either
//	if glog.V(2) { glog.Info("log this") }
// or
//	glog.V(2).Info("log this")
// The second form is shorter but the first is cheaper if logging is off because it does
// not evaluate its arguments.
//
// Whether an individual call to V generates a log record depends on the setting of
// the -glogVerbosity and --vmodule flags; both are off by default. If the level in the call to
// V is at least the value of -glogVerbosity, or of -vmodule for the source file containing the
// call, the V call will log.
func V(level Level) Verbose {
	// This function tries hard to be cheap unless there's work to do.
	// The fast path is two atomic loads and compares.

	glogVerbosity, err := strconv.Atoi(glogVerbosity)
	if err != nil {
		Fatalf("Verbosity level is invalid: %v", err)
	}
	if glogVerbosity >= int(level) {
		return Verbose(true)
	}
	return Verbose(false)
}

// Info is equivalent to the global Info function, guarded by the value of glogVerbosity.
// See the documentation of V for usage.
func (v Verbose) Info(args ...interface{}) {
	if v {
		doLog(2, infoLevel, args...)
	}
}

// Infoln is equivalent to the global Infoln function, guarded by the value of glogVerbosity.
// See the documentation of V for usage.
func (v Verbose) Infoln(args ...interface{}) {
	if v {
		doLog(2, infoLevel, args...)
	}
}

// Infof is equivalent to the global Infof function, guarded by the value of glogVerbosity.
// See the documentation of V for usage.
func (v Verbose) Infof(format string, args ...interface{}) {
	if v {
		doLogf(2, infoLevel, format, args...)
	}
}

// Info logs to the infoLevel log.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Info(args ...interface{}) {
	doLog(2, infoLevel, args...)
}

// InfoDepth acts as Info but uses depth to determine which call frame to log.
// InfoDepth(0, "msg") is the same as Info("msg").
func InfoDepth(depth int, args ...interface{}) {
	doLog(2+depth, infoLevel, args...)
}

// Infoln logs to the infoLevel log.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Infoln(args ...interface{}) {
	doLog(2, infoLevel, args...)
}

// Infof logs to the infoLevel log.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Infof(format string, args ...interface{}) {
	doLogf(2, infoLevel, format, args...)
}

// Warning logs to the warningLevel and infoLevel logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Warning(args ...interface{}) {
	doLog(2, warningLevel, args...)
}

// WarningDepth acts as Warning but uses depth to determine which call frame to log.
// WarningDepth(0, "msg") is the same as Warning("msg").
func WarningDepth(depth int, args ...interface{}) {
	doLog(2+depth, warningLevel, args...)
}

// Warningln logs to the warningLevel and infoLevel logs.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Warningln(args ...interface{}) {
	doLog(2, warningLevel, args...)
}

// Warningf logs to the warningLevel and infoLevel logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Warningf(format string, args ...interface{}) {
	doLogf(2, warningLevel, format, args...)
}

// Error logs to the errorLevel, warningLevel, and infoLevel logs.
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Error(args ...interface{}) {
	doLog(2, errorLevel, args...)
}

// ErrorDepth acts as Error but uses depth to determine which call frame to log.
// ErrorDepth(0, "msg") is the same as Error("msg").
func ErrorDepth(depth int, args ...interface{}) {
	doLog(2+depth, errorLevel, args...)
}

// Errorln logs to the errorLevel, warningLevel, and infoLevel logs.
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Errorln(args ...interface{}) {
	doLog(2, errorLevel, args...)
}

// Errorf logs to the errorLevel, warningLevel, and infoLevel logs.
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Errorf(format string, args ...interface{}) {
	doLogf(2, errorLevel, format, args...)
}

// Fatal logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Fatal(args ...interface{}) {
	doLog(2, fatalLevel, args...)
	os.Exit(255)
}

// FatalDepth acts as Fatal but uses depth to determine which call frame to log.
// FatalDepth(0, "msg") is the same as Fatal("msg").
func FatalDepth(depth int, args ...interface{}) {
	doLog(2+depth, fatalLevel, args...)
	os.Exit(255)
}

// Fatalln logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Println; a newline is appended if missing.
func Fatalln(args ...interface{}) {
	doLog(2, fatalLevel, args...)
	os.Exit(255)
}

// Fatalf logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs,
// including a stack trace of all running goroutines, then calls os.Exit(255).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Fatalf(format string, args ...interface{}) {
	doLogf(2, fatalLevel, format, args...)
	os.Exit(255)
}

// fatalNoStacks is non-zero if we are to exit without dumping goroutine stacks.
// It allows Exit and relatives to use the Fatal logs.
var fatalNoStacks uint32

// Exit logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs, then calls os.Exit(1).
// Arguments are handled in the manner of fmt.Print; a newline is appended if missing.
func Exit(args ...interface{}) {
	doLog(2, fatalLevel, args...)
	os.Exit(1)
}

// ExitDepth acts as Exit but uses depth to determine which call frame to log.
// ExitDepth(0, "msg") is the same as Exit("msg").
func ExitDepth(depth int, args ...interface{}) {
	doLog(2+depth, fatalLevel, args...)
	os.Exit(1)
}

// Exitln logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs, then calls os.Exit(1).
func Exitln(args ...interface{}) {
	doLog(2, fatalLevel, args...)
	os.Exit(1)
}

// Exitf logs to the fatalLevel, errorLevel, warningLevel, and infoLevel logs, then calls os.Exit(1).
// Arguments are handled in the manner of fmt.Printf; a newline is appended if missing.
func Exitf(format string, args ...interface{}) {
	doLogf(2, fatalLevel, format, args...)
	os.Exit(1)
}

func doLogf(skipFrames int, severity logLevel, format string, args ...interface{}) {
	if !toStderr {
		return
	}
	now := time.Now()
	_, fileName, lineNumber, _ := runtime.Caller(skipFrames)
	logger.Log(
		"level", logLevelNames[severity],
		"timestamp", now.Format("2006-01-02T15:04:05.000000Z"),
		"pos", fmt.Sprintf("%s:%d", filepath.Base(fileName), lineNumber),
		"component", glogComponent,
		"msg", fmt.Sprintf(format, args...),
	)
}

func doLog(skipFrames int, severity logLevel, args ...interface{}) {
	if !toStderr {
		return
	}
	now := time.Now()
	_, fileName, lineNumber, _ := runtime.Caller(skipFrames)
	logger.Log(
		"level", logLevelNames[severity],
		"timestamp", now.Format("2006-01-02T15:04:05.000000Z"),
		"pos", fmt.Sprintf("%s:%d", filepath.Base(fileName), lineNumber),
		"component", glogComponent,
		"msg", fmt.Sprint(args...),
	)
}

func doLogPos(severity logLevel, fileName string, lineNumber int, args ...interface{}) {
	if !toStderr {
		return
	}
	now := time.Now()
	logger.Log(
		"level", logLevelNames[severity],
		"timestamp", now.Format("2006-01-02T15:04:05.000000Z"),
		"pos", fmt.Sprintf("%s:%d", filepath.Base(fileName), lineNumber),
		"component", glogComponent,
		"msg", fmt.Sprint(args...),
	)
}
