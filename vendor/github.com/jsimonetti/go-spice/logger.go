package spice

import "github.com/sirupsen/logrus"

// Logger is a logging adapter interface
type Logger interface {
	// Debug logs debugging messages.
	Debug(...interface{})
	// Info logs informational messages.
	Info(...interface{})
	// Error logs error messages.
	Error(...interface{})
	// WithFields creates a new Logger with the fields embedded
	WithFields(keyvals ...interface{}) Logger
	// WithError creates a new Logger with the error embedded
	WithError(err error) Logger
}

// adapter is a thin wrapper around the logrus logger that adapts it to
// the Logger interface.
type adapter struct {
	*logrus.Entry
}

// Adapt creates a Logger backed from a logrus Entry.
func Adapt(l *logrus.Entry) Logger {
	return &adapter{l}
}

func (a *adapter) WithFields(keyvals ...interface{}) Logger {
	fields := a.fields(keyvals...)
	return &adapter{a.Entry.WithFields(fields)}
}

func (a *adapter) WithError(err error) Logger {
	return &adapter{a.Entry.WithError(err)}
}

func (a *adapter) fields(keyvals ...interface{}) logrus.Fields {
	if len(keyvals)%2 != 0 {
		keyvals = append(keyvals, "MISSING")
	}

	fields := make(logrus.Fields)

	for i := 0; i < len(keyvals); i += 2 {
		k := keyvals[i].(string)
		v := keyvals[i+1]
		fields[k] = v
	}

	return fields
}
