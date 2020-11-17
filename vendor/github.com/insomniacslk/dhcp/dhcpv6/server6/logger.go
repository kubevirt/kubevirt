package server6

import (
	"github.com/insomniacslk/dhcp/dhcpv6"
)

// Logger is a handler which will be used to output logging messages
type Logger interface {
	// PrintMessage print _all_ DHCP messages
	PrintMessage(prefix string, message *dhcpv6.Message)

	// Printf is use to print the rest debugging information
	Printf(format string, v ...interface{})
}

// EmptyLogger prints nothing
type EmptyLogger struct{}

// Printf is just a dummy function that does nothing
func (e EmptyLogger) Printf(format string, v ...interface{}) {}

// PrintMessage is just a dummy function that does nothing
func (e EmptyLogger) PrintMessage(prefix string, message *dhcpv6.Message) {}

// Printfer is used for actual output of the logger. For example *log.Logger is a Printfer.
type Printfer interface {
	// Printf is the function for logging output. Arguments are handled in the manner of fmt.Printf.
	Printf(format string, v ...interface{})
}

// ShortSummaryLogger is a wrapper for Printfer to implement interface Logger.
// DHCP messages are printed in the short format.
type ShortSummaryLogger struct {
	// Printfer is used for actual output of the logger
	Printfer
}

// Printf prints a log message as-is via predefined Printfer
func (s ShortSummaryLogger) Printf(format string, v ...interface{}) {
	s.Printfer.Printf(format, v...)
}

// PrintMessage prints a DHCP message in the short format via predefined Printfer
func (s ShortSummaryLogger) PrintMessage(prefix string, message *dhcpv6.Message) {
	s.Printf("%s: %s", prefix, message)
}

// DebugLogger is a wrapper for Printfer to implement interface Logger.
// DHCP messages are printed in the long format.
type DebugLogger struct {
	// Printfer is used for actual output of the logger
	Printfer
}

// Printf prints a log message as-is via predefined Printfer
func (d DebugLogger) Printf(format string, v ...interface{}) {
	d.Printfer.Printf(format, v...)
}

// PrintMessage prints a DHCP message in the long format via predefined Printfer
func (d DebugLogger) PrintMessage(prefix string, message *dhcpv6.Message) {
	d.Printf("%s: %s", prefix, message.Summary())
}
