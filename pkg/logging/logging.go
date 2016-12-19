package logging

import (
	"errors"
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"k8s.io/client-go/1.5/pkg/api/meta"
	k8sruntime "k8s.io/client-go/1.5/pkg/runtime"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

type logLevel int

const (
	DEBUG    logLevel = iota
	INFO     logLevel = iota
	WARNING  logLevel = iota
	ERROR    logLevel = iota
	CRITICAL logLevel = iota
)

type LoggableObject interface {
	meta.ObjectMetaAccessor
	k8sruntime.Object
}

type FilteredLogger struct {
	logContext            *log.Context
	filterLevel           logLevel
	currentLogLevel       logLevel
	verbosityLevel        int
	currentVerbosityLevel int
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
		logContext:            log.NewContext(logger).With("component", defaultComponent),
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
		log := MakeLogger(logger).With("component", component)
		loggers[component] = log
	}
	return loggers[component]
}

func DefaultLogger() *FilteredLogger {
	return Logger(defaultComponent)
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

func (l FilteredLogger) Log(params ...interface{}) error {
	// messages should be logged if any of these conditions are met:
	// The log filtering level is debug
	// The log filtering level is info and verbosity checks match
	// The log message priority is warning or higher
	force := (l.filterLevel == DEBUG) || (l.currentLogLevel >= WARNING)

	if force || (l.filterLevel == INFO &&
		(l.currentLogLevel == l.filterLevel) &&
		(l.currentVerbosityLevel <= l.verbosityLevel)) {
		logParams := make([]interface{}, 0)

		if l.currentVerbosityLevel > 1 {
			_, fileName, lineNumber, _ := runtime.Caller(1)
			logParams = append(logParams, "filename", filepath.Base(fileName))
			logParams = append(logParams, "line", lineNumber)
		}
		logParams = append(logParams, params...)
		return l.logContext.Log(logParams...)
	}
	return nil
}

func (l FilteredLogger) Object(obj LoggableObject) *FilteredLogger {

	name := obj.GetObjectMeta().GetName()
	uid := obj.GetObjectMeta().GetUID()
	kind := obj.GetObjectKind()

	logParams := make([]interface{}, 0)
	logParams = append(logParams, "name", name)
	logParams = append(logParams, "kind", kind)
	logParams = append(logParams, "uid", uid)

	l.logContext.With(logParams)
	return &l
}

func (l *FilteredLogger) With(obj ...interface{}) *FilteredLogger {
	l.logContext.With(obj...)
	return l
}

func (l *FilteredLogger) WithPrefix(obj ...interface{}) *FilteredLogger {
	l.logContext.WithPrefix(obj...)
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
