/*
 * This file is part of the kubevirt project
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

package logging

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

type logLevel int

const (
	DEBUG    logLevel = iota
	INFO     logLevel = iota
	WARNING  logLevel = iota
	ERROR    logLevel = iota
	CRITICAL logLevel = iota
)

var logLevelNames = map[logLevel]string{
	DEBUG:    "debug",
	INFO:     "info",
	WARNING:  "warning",
	ERROR:    "error",
	CRITICAL: "critical",
}

type LoggableObject interface {
	metav1.ObjectMetaAccessor
	k8sruntime.Object
}

type FilteredLogger struct {
	logContext            *log.Context
	component             string
	filterLevel           logLevel
	currentLogLevel       logLevel
	verbosityLevel        int
	currentVerbosityLevel int
	err                   error
}

func InitializeLogging(comp string) {
	flag.StringVar(&defaultComponent, "component", comp, "Default component for logs")
}

// Wrap a go-kit logger in a FilteredLogger. Not cached
func MakeLogger(logger log.Logger) *FilteredLogger {
	defaultLogLevel := INFO

	if verbosityFlag := flag.Lookup("v"); verbosityFlag != nil {
		defaultVerbosity, _ = strconv.Atoi(verbosityFlag.Value.String())
	} else {
		defaultVerbosity = 0
	}
	return &FilteredLogger{
		logContext:            log.NewContext(logger),
		component:             defaultComponent,
		filterLevel:           defaultLogLevel,
		currentLogLevel:       defaultLogLevel,
		verbosityLevel:        defaultVerbosity,
		currentVerbosityLevel: defaultVerbosity,
	}
}

type NullLogger struct{}

func (n NullLogger) Log(params ...interface{}) error { return nil }

var loggers = make(map[string]*FilteredLogger)
var defaultComponent = ""
var defaultVerbosity = 0

func Logger(component string) *FilteredLogger {
	if _, ok := loggers[component]; !ok {
		logger := log.NewLogfmtLogger(os.Stderr)
		log := MakeLogger(logger)
		log.component = component
		loggers[component] = log
	}
	return loggers[component]
}

func DefaultLogger() *FilteredLogger {
	return Logger(defaultComponent)
}

func (l *FilteredLogger) SetIOWriter(w io.Writer) *FilteredLogger {
	l.logContext = log.NewContext(log.NewLogfmtLogger(w))
	return l
}

func (l *FilteredLogger) SetLogger(logger log.Logger) *FilteredLogger {
	l.logContext = log.NewContext(logger)
	return l
}

type LogError struct {
	message string
}

func (e LogError) Error() string {
	return e.message
}

func (l FilteredLogger) Msg(msg interface{}) {
	l.log(2, "msg", msg)
}

func (l FilteredLogger) Msgf(msg string, args ...interface{}) {
	l.log(2, "msg", fmt.Sprintf(msg, args...))
}

func (l FilteredLogger) Log(params ...interface{}) error {
	return l.log(2, params...)
}

func (l FilteredLogger) log(skipFrames int, params ...interface{}) error {
	// messages should be logged if any of these conditions are met:
	// The log filtering level is debug
	// The log filtering level is info and verbosity checks match
	// The log message priority is warning or higher
	force := (l.filterLevel == DEBUG) || (l.currentLogLevel >= WARNING)

	if force || (l.filterLevel == INFO &&
		(l.currentLogLevel == l.filterLevel) &&
		(l.currentVerbosityLevel <= l.verbosityLevel)) {
		now := time.Now().UTC()
		_, fileName, lineNumber, _ := runtime.Caller(skipFrames)
		logParams := make([]interface{}, 0, 8)

		logParams = append(logParams,
			"level", logLevelNames[l.currentLogLevel],
			"timestamp", now.Format("2006-01-02T15:04:05.000000Z"),
			"pos", fmt.Sprintf("%s:%d", filepath.Base(fileName), lineNumber),
			"component", l.component,
		)
		if l.err != nil {
			l.logContext = l.logContext.With("reason", l.err)
		}
		return l.logContext.WithPrefix(logParams...).Log(params...)
	}
	return nil
}

func (l FilteredLogger) Object(obj LoggableObject) *FilteredLogger {

	name := obj.GetObjectMeta().GetName()
	uid := obj.GetObjectMeta().GetUID()
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	logParams := make([]interface{}, 0)
	logParams = append(logParams, "name", name)
	logParams = append(logParams, "kind", kind)
	logParams = append(logParams, "uid", uid)

	l.With(logParams...)
	return &l
}

func (l *FilteredLogger) With(obj ...interface{}) *FilteredLogger {
	l.logContext = l.logContext.With(obj...)
	return l
}

func (l *FilteredLogger) WithPrefix(obj ...interface{}) *FilteredLogger {
	l.logContext = l.logContext.WithPrefix(obj...)
	return l
}

func (l *FilteredLogger) SetLogLevel(filterLevel logLevel) error {
	if (filterLevel >= DEBUG) && (filterLevel <= CRITICAL) {
		l.filterLevel = filterLevel
		return nil
	}
	return errors.New(fmt.Sprintf("Log level %d does not exist", filterLevel))
}

func (l *FilteredLogger) SetVerbosityLevel(level int) error {
	if level >= 0 {
		l.verbosityLevel = level
	} else {
		return errors.New("Verbosity setting must not be negative")
	}
	return nil
}

// It would be consistent to return an error from this function, but
// a multi-value function would break the primary use case: log.V(2).Info()....
func (l FilteredLogger) V(level int) *FilteredLogger {
	if level >= 0 {
		l.currentVerbosityLevel = level
	}
	return &l
}

func (l FilteredLogger) Debug() *FilteredLogger {
	l.currentLogLevel = DEBUG
	return &l
}

func (l FilteredLogger) Info() *FilteredLogger {
	l.currentLogLevel = INFO
	return &l
}

func (l FilteredLogger) Warning() *FilteredLogger {
	l.currentLogLevel = WARNING
	return &l
}

func (l FilteredLogger) Error() *FilteredLogger {
	l.currentLogLevel = ERROR
	return &l
}

func (l FilteredLogger) Critical() *FilteredLogger {
	l.currentLogLevel = CRITICAL
	return &l
}

func (l FilteredLogger) Reason(err error) *FilteredLogger {
	l.err = err
	return &l
}
