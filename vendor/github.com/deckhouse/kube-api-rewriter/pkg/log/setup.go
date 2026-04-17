/*
Copyright 2024 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package log

import (
	"io"
	"log/slog"
	"os"
	"strings"
)

type Format string

const (
	JSONLog   Format = "json"
	TextLog   Format = "text"
	PrettyLog Format = "pretty"
)

type Output string

const (
	Stdout  Output = "stdout"
	Stderr  Output = "stderr"
	Discard Output = "discard"
)

// Defaults
const (
	DefaultLogLevel       = slog.LevelInfo
	DefaultDebugLogFormat = PrettyLog
	DefaultLogFormat      = JSONLog
)

var DefaultLogOutput = os.Stdout

type Options struct {
	Level  string
	Format string
	Output string
}

func SetupDefaultLoggerFromEnv(opts Options) {
	handler := SetupHandler(opts)
	if handler != nil {
		slog.SetDefault(slog.New(handler))
	}
}

func SetupHandler(opts Options) slog.Handler {
	logLevel := detectLogLevel(opts.Level)
	logOutput := detectLogOutput(opts.Output)
	logFormat := detectLogFormat(opts.Format, logLevel)

	logHandlerOpts := &slog.HandlerOptions{Level: logLevel}
	switch logFormat {
	case TextLog:
		return slog.NewTextHandler(logOutput, logHandlerOpts)
	case JSONLog:
		return slog.NewJSONHandler(logOutput, logHandlerOpts)
	case PrettyLog:
		return NewPrettyHandler(logOutput, logHandlerOpts)
	}
	return nil
}

func detectLogLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	}
	return DefaultLogLevel
}

func detectLogFormat(format string, level slog.Level) Format {
	switch strings.ToLower(format) {
	case string(TextLog):
		return TextLog
	case string(JSONLog):
		return JSONLog
	case string(PrettyLog):
		return PrettyLog
	}
	if level == slog.LevelDebug {
		return DefaultDebugLogFormat
	}
	return DefaultLogFormat
}

func detectLogOutput(output string) io.Writer {
	switch strings.ToLower(output) {
	case string(Stdout):
		return os.Stdout
	case string(Stderr):
		return os.Stderr
	case string(Discard):
		return io.Discard
	}
	return DefaultLogOutput
}
