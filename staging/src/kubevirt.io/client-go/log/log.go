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
	goflag "flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

const (
	libvirtTimestampFormat = "2006-01-02 15:04:05.999-0700"
	logTimestampFormat     = "2006-01-02T15:04:05.000000Z"
)

type LogLevel int32

const (
	INFO LogLevel = iota
	WARNING
	ERROR
	FATAL
)

var LogLevelNames = map[LogLevel]string{
	INFO:    "info",
	WARNING: "warning",
	ERROR:   "error",
	FATAL:   "fatal",
}

var lock sync.Mutex

type LoggableObject interface {
	metav1.ObjectMetaAccessor
	k8sruntime.Object
}

type FilteredVerbosityLogger struct {
	filteredLogger FilteredLogger
}

type FilteredLogger struct {
	logger                logr.Logger
	component             string
	filterLevel           LogLevel
	currentLogLevel       LogLevel
	verbosityLevel        int
	currentVerbosityLevel int
	err                   error
	contextValues         []interface{}
}

var Log = DefaultLogger()

func init() {
	// "the practical default level is V(2)"
	// see https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md
	goflag.IntVar(&defaultVerbosity, "v", 2, "log level for V logs")
}

func InitializeLogging(comp string) {
	defaultComponent = comp
	Log = DefaultLogger()
	goflag.CommandLine.Set("component", comp)
	goflag.CommandLine.Set("logtostderr", "true")
}

func MakeLogger(logger logr.Logger) *FilteredLogger {
	defaultLogLevel := INFO
	defaultCurrentVerbosity := 2

	return &FilteredLogger{
		logger:                logger,
		component:             defaultComponent,
		filterLevel:           defaultLogLevel,
		currentLogLevel:       defaultLogLevel,
		verbosityLevel:        defaultVerbosity,
		currentVerbosityLevel: defaultCurrentVerbosity,
	}
}

var loggers = make(map[string]*FilteredLogger)
var defaultComponent = ""
var defaultVerbosity = 0

func defaultLogr() logr.Logger {
	return funcr.NewJSON(
		func(obj string) { fmt.Fprintln(os.Stderr, obj) },
		funcr.Options{
			LogTimestamp:    true,
			LogCaller:       funcr.All,
			TimestampFormat: logTimestampFormat,
			Verbosity:       10,
		},
	)
}

func createLogger(component string) {
	lock.Lock()
	defer lock.Unlock()
	_, ok := loggers[component]
	if ok == false {
		logger := defaultLogr()
		log := MakeLogger(logger)
		log.component = component
		loggers[component] = log
	}
}

func Logger(component string) *FilteredLogger {
	_, ok := loggers[component]
	if ok == false {
		createLogger(component)
	}
	return loggers[component]
}

func DefaultLogger() *FilteredLogger {
	return Logger(defaultComponent)
}

// SetIOWriter redirects log output to the given writer. Intended for testing.
func (l *FilteredLogger) SetIOWriter(w io.Writer) {
	l.logger = funcr.NewJSON(
		func(obj string) { fmt.Fprintln(w, obj) },
		funcr.Options{
			LogTimestamp:    true,
			LogCaller:       funcr.All,
			TimestampFormat: logTimestampFormat,
			Verbosity:       10,
		},
	)
}

func (l *FilteredLogger) SetLogger(logger logr.Logger) *FilteredLogger {
	l.logger = logger
	return l
}

func (l FilteredLogger) msg(msg interface{}) {
	l.log("msg", msg)
}

func (l FilteredLogger) msgf(msg string, args ...interface{}) {
	l.log("msg", fmt.Sprintf(msg, args...))
}

func (l FilteredLogger) Log(params ...interface{}) error {
	l.log(params...)
	return nil
}

func (l FilteredLogger) log(params ...interface{}) {
	// messages should be logged if any of these conditions are met:
	// The log filtering level is info and verbosity checks match
	// The log message priority is warning or higher
	if l.currentLogLevel >= WARNING || (l.filterLevel == INFO &&
		(l.currentLogLevel == l.filterLevel) &&
		(l.currentVerbosityLevel <= l.verbosityLevel)) {

		// Extract "msg" from params (params are key-value pairs)
		msg := ""
		kvs := make([]interface{}, 0, len(params)+4)
		for i := 0; i < len(params)-1; i += 2 {
			if key, ok := params[i].(string); ok && key == "msg" {
				msg = fmt.Sprintf("%v", params[i+1])
			} else {
				kvs = append(kvs, params[i], params[i+1])
			}
		}
		if len(params)%2 != 0 {
			kvs = append(kvs, params[len(params)-1])
		}

		kvs = append(kvs, "level", LogLevelNames[l.currentLogLevel], "component", l.component)

		logger := l.logger
		if len(l.contextValues) > 0 {
			logger = logger.WithValues(l.contextValues...)
		}
		if l.err != nil {
			logger = logger.WithValues("reason", l.err)
		}

		switch l.currentLogLevel {
		case ERROR, FATAL:
			logger.Error(l.err, msg, kvs...)
		default:
			logger.Info(msg, kvs...)
		}
	}
}

func (l FilteredVerbosityLogger) Log(params ...interface{}) error {
	l.filteredLogger.log(params...)
	return nil
}

func (l FilteredVerbosityLogger) V(level int) *FilteredVerbosityLogger {
	if level >= 0 {
		l.filteredLogger.currentVerbosityLevel = level
	}
	return &l
}

func (l FilteredVerbosityLogger) Info(msg string) {
	l.filteredLogger.Level(INFO).log("msg", msg)
}

func (l FilteredVerbosityLogger) Infof(msg string, args ...interface{}) {
	l.filteredLogger.Level(INFO).log("msg", fmt.Sprintf(msg, args...))
}

func (l FilteredVerbosityLogger) Object(obj LoggableObject) *FilteredVerbosityLogger {
	l.filteredLogger = *l.filteredLogger.Object(obj)
	return &l
}

func (l FilteredVerbosityLogger) Reason(err error) *FilteredVerbosityLogger {
	l.filteredLogger.err = err
	return &l
}

func (l FilteredVerbosityLogger) Verbosity(level int) bool {
	return l.filteredLogger.Verbosity(level)
}

func (l FilteredLogger) Object(obj LoggableObject) *FilteredLogger {
	name := obj.GetObjectMeta().GetName()
	namespace := obj.GetObjectMeta().GetNamespace()
	uid := obj.GetObjectMeta().GetUID()
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	logParams := make([]interface{}, 0)
	if namespace != "" {
		logParams = append(logParams, "namespace", namespace)
	}
	logParams = append(logParams, "name", name)
	logParams = append(logParams, "kind", kind)
	logParams = append(logParams, "uid", uid)

	l.contextValues = append(append([]interface{}{}, l.contextValues...), logParams...)
	return &l
}

func (l FilteredLogger) With(obj ...interface{}) *FilteredLogger {
	l.contextValues = append(append([]interface{}{}, l.contextValues...), obj...)
	return &l
}

func (l *FilteredLogger) SetLogLevel(filterLevel LogLevel) error {
	if (filterLevel >= INFO) && (filterLevel <= FATAL) {
		l.filterLevel = filterLevel
		return nil
	}
	return fmt.Errorf("Log level %d does not exist", filterLevel)
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
func (l FilteredLogger) V(level int) *FilteredVerbosityLogger {
	if level >= 0 {
		l.currentVerbosityLevel = level
	}
	return &FilteredVerbosityLogger{
		filteredLogger: l,
	}
}

func (l FilteredLogger) Verbosity(level int) bool {
	return l.currentVerbosityLevel >= level
}

func (l FilteredLogger) Reason(err error) *FilteredLogger {
	l.err = err
	return &l
}

func (l FilteredLogger) Level(level LogLevel) *FilteredLogger {
	l.currentLogLevel = level
	return &l
}

func (l FilteredLogger) Info(msg string) {
	l.Level(INFO).msg(msg)
}

func (l FilteredLogger) Infof(msg string, args ...interface{}) {
	l.Level(INFO).msgf(msg, args...)
}

func (l FilteredLogger) Warning(msg string) {
	l.Level(WARNING).msg(msg)
}

func (l FilteredLogger) Warningf(msg string, args ...interface{}) {
	l.Level(WARNING).msgf(msg, args...)
}

func (l FilteredLogger) Error(msg string) {
	l.Level(ERROR).msg(msg)
}

func (l FilteredLogger) Errorf(msg string, args ...interface{}) {
	l.Level(ERROR).msgf(msg, args...)
}

func (l FilteredLogger) Critical(msg string) {
	l.Level(FATAL).msg(msg)
	panic(msg)
}

func (l FilteredLogger) Criticalf(msg string, args ...interface{}) {
	l.Level(FATAL).msgf(msg, args...)
}

func LogLibvirtLogLine(logger *FilteredLogger, line string) {
	if len(strings.TrimSpace(line)) == 0 {
		return
	}

	fragments := strings.SplitN(line, ": ", 5)
	if len(fragments) < 4 {
		now := time.Now()
		logger.logger.Info(line,
			"subcomponent", "libvirt",
			"component", logger.component,
			"timestamp", now.Format(logTimestampFormat),
		)
		return
	}
	severity := strings.ToLower(strings.TrimSpace(fragments[2]))

	if severity == "debug" {
		severity = "info"
	}

	t, err := time.Parse(libvirtTimestampFormat, strings.TrimSpace(fragments[0]))
	if err != nil {
		fmt.Println(err)
		return
	}
	thread := strings.TrimSpace(fragments[1])
	pos := strings.TrimSpace(fragments[3])
	msg := strings.TrimSpace(fragments[4])

	if strings.Contains(msg, "unable to execute QEMU agent command") {
		if logger.verbosityLevel < 4 {
			return
		}
		severity = LogLevelNames[WARNING]
	}

	kvs := []interface{}{
		"level", severity,
		"timestamp", t.Format(logTimestampFormat),
		"component", logger.component,
		"subcomponent", "libvirt",
		"thread", thread,
	}

	isPos := false
	if split := strings.Split(pos, ":"); len(split) == 2 {
		if _, err := fmt.Sscanf(split[1], "%d", new(int)); err == nil {
			isPos = true
		}
	}

	if isPos {
		kvs = append(kvs, "pos", pos)
	} else {
		msg = strings.TrimSpace(fragments[3] + ": " + fragments[4])
	}

	logger.logger.Info(msg, kvs...)
}

var qemuLogLines = ""

func LogQemuLogLine(logger *FilteredLogger, line string) {
	if len(strings.TrimSpace(line)) == 0 {
		return
	}

	if strings.HasSuffix(line, "\\") {
		qemuLogLines += line
		return
	}

	if len(qemuLogLines) > 0 {
		line = qemuLogLines + line
		qemuLogLines = ""
	}

	now := time.Now()
	logger.logger.Info(line,
		"level", "info",
		"timestamp", now.Format(logTimestampFormat),
		"component", logger.component,
		"subcomponent", "qemu",
	)
}
