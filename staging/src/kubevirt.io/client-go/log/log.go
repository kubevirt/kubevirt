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
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	context               []interface{}
	component             string
	filterLevel           LogLevel
	currentLogLevel       LogLevel
	verbosityLevel        int
	currentVerbosityLevel int
	err                   error
}

var Log = DefaultLogger()

var logFlags = goflag.NewFlagSet("kubevirt-log", goflag.ContinueOnError)

func init() {
	// "the practical default level is V(2)"
	// see https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md
	logFlags.IntVar(&defaultVerbosity, "v", 2, "log level for V logs")
}

// VerbosityFlag returns the "-v" verbosity flag for bridging into
// other flag sets (e.g. pflag). The flag is owned by a package-
// internal FlagSet so that importing this package never registers
// flags on flag.CommandLine and therefore cannot conflict with
// other packages that register a "-v" flag there.
//
// Example:
//
//	pflag.CommandLine.AddGoFlag(log.VerbosityFlag())
func VerbosityFlag() *goflag.Flag {
	f := logFlags.Lookup("v")
	if f == nil {
		panic("kubevirt.io/client-go/log: verbosity flag \"v\" not registered on internal FlagSet")
	}
	return f
}

func InitializeLogging(comp string) {
	defaultComponent = comp
	Log = DefaultLogger()
	goflag.CommandLine.Set("component", comp)
	goflag.CommandLine.Set("logtostderr", "true")
	log.SetOutput(stdlibAdapter{logger: Log})
}

// NewJSONLogger returns a logr.Logger which writes each log entry to w as a
// single JSON line. All of zap's own entry fields (level, timestamp, caller,
// message) are omitted; FilteredLogger injects its equivalents as ordinary
// key-value pairs so the emitted JSON shape stays under its control.
func NewJSONLogger(w io.Writer) logr.Logger {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     zapcore.OmitKey,
		LevelKey:       zapcore.OmitKey,
		TimeKey:        zapcore.OmitKey,
		NameKey:        zapcore.OmitKey,
		CallerKey:      zapcore.OmitKey,
		FunctionKey:    zapcore.OmitKey,
		StacktraceKey:  zapcore.OmitKey,
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), zapcore.AddSync(w), zapcore.DebugLevel)
	return zapr.NewLogger(zap.New(core))
}

// Wrap a logr logger in a FilteredLogger. Not cached
func MakeLogger(logger logr.Logger) *FilteredLogger {
	defaultLogLevel := INFO

	// This verbosity will be used for info logs without setting a custom verbosity level
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

func createLogger(component string) {
	lock.Lock()
	defer lock.Unlock()
	_, ok := loggers[component]
	if ok == false {
		log := MakeLogger(NewJSONLogger(os.Stderr))
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
	l.logger = NewJSONLogger(w)
	goflag.CommandLine.Set("logtostderr", "false")
}

func (l *FilteredLogger) SetLogger(logger logr.Logger) *FilteredLogger {
	l.logger = logger
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
		logParams := make([]interface{}, 0, 8+len(l.context)+len(params)+1)

		logParams = append(logParams,
			"level", LogLevelNames[l.currentLogLevel],
			"timestamp", now.Format(logTimestampFormat),
			"pos", fmt.Sprintf("%s:%d", filepath.Base(fileName), lineNumber),
			"component", l.component,
		)
		logParams = append(logParams, l.context...)
		if l.err != nil {
			logParams = append(logParams, "reason", l.err)
		}
		logParams = append(logParams, params...)
		l.logger.Info("", sanitizeKeyVals(logParams)...)
	}
	return nil
}

// sanitizeKeyVals enforces the key-value pair contract of logr: an odd
// trailing key gets a placeholder value and non-string keys are stringified,
// both of which the previous go-kit backend tolerated.
func sanitizeKeyVals(keyVals []interface{}) []interface{} {
	if len(keyVals)%2 != 0 {
		keyVals = append(keyVals, "(MISSING)")
	}
	for i := 0; i < len(keyVals); i += 2 {
		if _, ok := keyVals[i].(string); !ok {
			keyVals[i] = fmt.Sprint(keyVals[i])
		}
	}
	return keyVals
}

func (l FilteredVerbosityLogger) Log(params ...interface{}) error {
	return l.filteredLogger.log(2, params...)
}

func (l FilteredVerbosityLogger) V(level int) *FilteredVerbosityLogger {
	if level >= 0 {
		l.filteredLogger.currentVerbosityLevel = level
	}
	return &l
}

func (l FilteredVerbosityLogger) Info(msg string) {
	l.filteredLogger.Level(INFO).log(2, "msg", msg)
}

func (l FilteredVerbosityLogger) Infof(msg string, args ...interface{}) {
	l.filteredLogger.Level(INFO).log(2, "msg", fmt.Sprintf(msg, args...))
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

	l.with(logParams...)
	return &l
}

func (l FilteredLogger) With(obj ...interface{}) *FilteredLogger {
	l.context = appendContext(l.context, obj)
	return &l
}

func (l *FilteredLogger) with(obj ...interface{}) *FilteredLogger {
	l.context = appendContext(l.context, obj)
	return l
}

// appendContext always allocates a fresh slice so that FilteredLogger copies
// sharing a backing array can never observe each other's appended values.
func appendContext(context []interface{}, obj []interface{}) []interface{} {
	merged := make([]interface{}, 0, len(context)+len(obj))
	merged = append(merged, context...)
	return append(merged, obj...)
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

// stdlibRegexp matches the configurable prefix of the standard library's log
// package: an optional date, an optional time and an optional file:line.
var stdlibRegexp = regexp.MustCompile(`(?s)^(?:[0-9]{4}/[0-9]{2}/[0-9]{2} )?(?:[0-9]{2}:[0-9]{2}:[0-9]{2}(?:\.[0-9]+)? )?(?:.+?:[0-9]+: )?(?P<msg>.*)`)

// stdlibAdapter redirects the standard library's global "log" package output
// into a FilteredLogger, stripping the date/time/file prefix so the message
// is not duplicated by the structured fields. It replaces the equivalent
// adapter that go-kit/log used to provide.
type stdlibAdapter struct {
	logger *FilteredLogger
}

func (a stdlibAdapter) Write(p []byte) (int, error) {
	msg := string(p)
	if match := stdlibRegexp.FindStringSubmatch(msg); match != nil {
		msg = match[stdlibRegexp.SubexpIndex("msg")]
	}
	a.logger.Log("msg", strings.TrimRight(msg, "\n"))
	return len(p), nil
}

func LogLibvirtLogLine(logger *FilteredLogger, line string) {

	if len(strings.TrimSpace(line)) == 0 {
		return
	}

	fragments := strings.SplitN(line, ": ", 5)
	if len(fragments) < 4 {
		now := time.Now()
		logger.logger.Info("",
			"level", "info",
			"timestamp", now.Format(logTimestampFormat),
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

	//TODO: implement proper behavior for unsupported GA commands
	// by either considering the GA version as unsupported or just don't
	// send commands which not supported
	if strings.Contains(msg, "unable to execute QEMU agent command") {
		if logger.verbosityLevel < 4 {
			return
		}

		severity = LogLevelNames[WARNING]
	}

	// check if we really got a position
	isPos := false
	if split := strings.Split(pos, ":"); len(split) == 2 {
		if _, err := strconv.Atoi(split[1]); err == nil {
			isPos = true
		}
	}

	if !isPos {
		msg = strings.TrimSpace(fragments[3] + ": " + fragments[4])
		logger.logger.Info("",
			"level", severity,
			"timestamp", t.Format(logTimestampFormat),
			"component", logger.component,
			"subcomponent", "libvirt",
			"thread", thread,
			"msg", msg,
		)
	} else {
		logger.logger.Info("",
			"level", severity,
			"timestamp", t.Format(logTimestampFormat),
			"pos", pos,
			"component", logger.component,
			"subcomponent", "libvirt",
			"thread", thread,
			"msg", msg,
		)
	}
}

var qemuLogLines = ""

func LogQemuLogLine(logger *FilteredLogger, line string) {

	if len(strings.TrimSpace(line)) == 0 {
		return
	}

	// Concat break lines to have full command in one log message
	if strings.HasSuffix(line, "\\") {
		qemuLogLines += line
		return
	}

	if len(qemuLogLines) > 0 {
		line = qemuLogLines + line
		qemuLogLines = ""
	}

	now := time.Now()
	logger.logger.Info("",
		"level", "info",
		"timestamp", now.Format(logTimestampFormat),
		"component", logger.component,
		"subcomponent", "qemu",
		"msg", line,
	)
}
