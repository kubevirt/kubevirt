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
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	libvirtTimestampFormat = "2006-01-02 15:04:05.999-0700"
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

type FilteredLogger struct {
	logContext            *log.Context
	component             string
	filterLevel           LogLevel
	currentLogLevel       LogLevel
	verbosityLevel        int
	currentVerbosityLevel int
	err                   error
}

var Log = DefaultLogger()

func InitializeLogging(comp string) {
	defaultComponent = comp
	Log = DefaultLogger()
	glog.CopyStandardLogTo(LogLevelNames[INFO])
	goflag.CommandLine.Set("component", comp)
	goflag.CommandLine.Set("logtostderr", "true")
}

func getDefaultVerbosity() int {
	if verbosityFlag := flag.Lookup("v"); verbosityFlag != nil {
		defaultVerbosity, _ := strconv.Atoi(verbosityFlag.Value.String())
		return defaultVerbosity
	} else {
		// "the practical default level is V(2)"
		// see https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md
		return 2
	}
}

// Wrap a go-kit logger in a FilteredLogger. Not cached
func MakeLogger(logger log.Logger) *FilteredLogger {
	defaultLogLevel := INFO

	defaultVerbosity = getDefaultVerbosity()
	// This verbosity will be used for info logs without setting a custom verbosity level
	defaultCurrentVerbosity := 2

	return &FilteredLogger{
		logContext:            log.NewContext(logger),
		component:             defaultComponent,
		filterLevel:           defaultLogLevel,
		currentLogLevel:       defaultLogLevel,
		verbosityLevel:        defaultVerbosity,
		currentVerbosityLevel: defaultCurrentVerbosity,
	}
}

type NullLogger struct{}

func (n NullLogger) Log(params ...interface{}) error { return nil }

var loggers = make(map[string]*FilteredLogger)
var defaultComponent = ""
var defaultVerbosity = 0

func createLogger(component string) {
	lock.Lock()
	defer lock.Unlock()
	_, ok := loggers[component]
	if ok == false {
		logger := log.NewJSONLogger(os.Stderr)
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

// SetIOWriter is meant to be used for testing. "log" and "glog" logs are sent to /dev/nil.
// KubeVirt related log messages will be sent to this writer
func (l *FilteredLogger) SetIOWriter(w io.Writer) {
	l.logContext = log.NewContext(log.NewJSONLogger(w))
	goflag.CommandLine.Set("logtostderr", "false")
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

func (l FilteredLogger) msg(msg interface{}) {
	l.log(3, "msg", msg)
}

func (l FilteredLogger) msgf(msg string, args ...interface{}) {
	l.log(3, "msg", fmt.Sprintf(msg, args...))
}

func (l FilteredLogger) Log(params ...interface{}) error {
	return l.log(2, params...)
}

func (l FilteredLogger) log(skipFrames int, params ...interface{}) error {
	// messages should be logged if any of these conditions are met:
	// The log filtering level is info and verbosity checks match
	// The log message priority is warning or higher
	if l.currentLogLevel >= WARNING || (l.filterLevel == INFO &&
		(l.currentLogLevel == l.filterLevel) &&
		(l.currentVerbosityLevel <= l.verbosityLevel)) {
		now := time.Now().UTC()
		_, fileName, lineNumber, _ := runtime.Caller(skipFrames)
		logParams := make([]interface{}, 0, 8)

		logParams = append(logParams,
			"level", LogLevelNames[l.currentLogLevel],
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
func (l FilteredLogger) Key(key string, kind string) *FilteredLogger {
	if key == "" {
		return &l

	}
	name, namespace, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return &l
	}
	logParams := make([]interface{}, 0)
	if namespace != "" {
		logParams = append(logParams, "namespace", namespace)
	}
	logParams = append(logParams, "name", name)
	logParams = append(logParams, "kind", kind)
	l.With(logParams...)
	return &l
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

	l.With(logParams...)
	return &l
}

func (l FilteredLogger) ObjectRef(obj *v1.ObjectReference) *FilteredLogger {

	if obj == nil {
		return &l
	}

	logParams := make([]interface{}, 0)
	if obj.Namespace != "" {
		logParams = append(logParams, "namespace", obj.Namespace)
	}
	logParams = append(logParams, "name", obj.Name)
	logParams = append(logParams, "kind", obj.Kind)
	logParams = append(logParams, "uid", obj.UID)

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
func (l FilteredLogger) V(level int) *FilteredLogger {
	if level >= 0 {
		l.currentVerbosityLevel = level
	}
	return &l
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
		logger.logContext.Log(
			"level", "info",
			"timestamp", now.Format("2006-01-02T15:04:05.000000Z"),
			"component", logger.component,
			"subcomponent", "libvirt",
			"msg", line,
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

	// check if we really got a position
	isPos := false
	if split := strings.Split(pos, ":"); len(split) == 2 {
		if _, err := strconv.Atoi(split[1]); err == nil {
			isPos = true
		}
	}

	if !isPos {
		msg = strings.TrimSpace(fragments[3] + ": " + fragments[4])
		logger.logContext.Log(
			"level", severity,
			"timestamp", t.Format("2006-01-02T15:04:05.000000Z"),
			"component", logger.component,
			"subcomponent", "libvirt",
			"thread", thread,
			"msg", msg,
		)
	} else {
		logger.logContext.Log(
			"level", severity,
			"timestamp", t.Format("2006-01-02T15:04:05.000000Z"),
			"pos", pos,
			"component", logger.component,
			"subcomponent", "libvirt",
			"thread", thread,
			"msg", msg,
		)
	}
}
