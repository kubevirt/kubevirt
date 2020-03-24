// Copyright (c) 2018 Intel Corporation
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

package logging

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Level type
type Level uint32

// PanicLevel...MaxLevel indicates the logging level
const (
	PanicLevel Level = iota
	ErrorLevel
	VerboseLevel
	DebugLevel
	MaxLevel
	UnknownLevel
)

var loggingStderr bool
var loggingFp *os.File
var loggingLevel Level

const defaultTimestampFormat = time.RFC3339

func (l Level) String() string {
	switch l {
	case PanicLevel:
		return "panic"
	case VerboseLevel:
		return "verbose"
	case ErrorLevel:
		return "error"
	case DebugLevel:
		return "debug"
	}
	return "unknown"
}

func printf(level Level, format string, a ...interface{}) {
	header := "%s [%s] "
	t := time.Now()
	if level > loggingLevel {
		return
	}

	if loggingStderr {
		fmt.Fprintf(os.Stderr, header, t.Format(defaultTimestampFormat), level)
		fmt.Fprintf(os.Stderr, format, a...)
		fmt.Fprintf(os.Stderr, "\n")
	}

	if loggingFp != nil {
		fmt.Fprintf(loggingFp, header, t.Format(defaultTimestampFormat), level)
		fmt.Fprintf(loggingFp, format, a...)
		fmt.Fprintf(loggingFp, "\n")
	}
}

// Debugf prints logging if logging level >= debug
func Debugf(format string, a ...interface{}) {
	printf(DebugLevel, format, a...)
}

// Verbosef prints logging if logging level >= verbose
func Verbosef(format string, a ...interface{}) {
	printf(VerboseLevel, format, a...)
}

// Errorf prints logging if logging level >= error
func Errorf(format string, a ...interface{}) error {
	printf(ErrorLevel, format, a...)
	return fmt.Errorf(format, a...)
}

// Panicf prints logging plus stack trace. This should be used only for unrecoverble error
func Panicf(format string, a ...interface{}) {
	printf(PanicLevel, format, a...)
	printf(PanicLevel, "========= Stack trace output ========")
	printf(PanicLevel, "%+v", errors.New("Multus Panic"))
	printf(PanicLevel, "========= Stack trace output end ========")
}

// GetLoggingLevel gets current logging level
func GetLoggingLevel() Level {
	return loggingLevel
}

func getLoggingLevel(levelStr string) Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return DebugLevel
	case "verbose":
		return VerboseLevel
	case "error":
		return ErrorLevel
	case "panic":
		return PanicLevel
	}
	fmt.Fprintf(os.Stderr, "multus logging: cannot set logging level to %s\n", levelStr)
	return UnknownLevel
}

// SetLogLevel sets logging level
func SetLogLevel(levelStr string) {
	level := getLoggingLevel(levelStr)
	if level < MaxLevel {
		loggingLevel = level
	}
}

// SetLogStderr sets flag for logging stderr output
func SetLogStderr(enable bool) {
	loggingStderr = enable
}

// SetLogFile sets logging file
func SetLogFile(filename string) {
	if filename == "" {
		return
	}

	fp, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		loggingFp = nil
		fmt.Fprintf(os.Stderr, "multus logging: cannot open %s", filename)
	}
	loggingFp = fp
}

func init() {
	loggingStderr = true
	loggingFp = nil
	loggingLevel = PanicLevel
}
