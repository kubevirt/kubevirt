package spice

import (
	"context"
	"net"

	"github.com/sirupsen/logrus"
)

// Option is a functional option handler for Server.
type Option func(*Proxy) error

// SetOption runs a functional option against the server.
func (p *Proxy) SetOption(option Option) error {
	return option(p)
}

// WithLogger can be used to provide a custom logger.
// Defaults to a logrus implementation.
func WithLogger(log Logger) Option {
	return func(p *Proxy) error {
		p.log = log
		return nil
	}
}

// WithAuthenticator can be provided to implement custom authentication
// By default, "auth-less" no-op mode is enabled.
func WithAuthenticator(a Authenticator) Option {
	return func(p *Proxy) error {
		if err := a.Init(); err != nil {
			return err
		}
		p.authenticator[a.Method()] = a
		return nil
	}
}

// WithDialer can be used to provide a custom dialer to reach compute nodes
// the network is always of type 'tcp' and the computeAddress is the compute node
// computeAddress that is return by an Authenticator.
func WithDialer(dial func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(p *Proxy) error {
		p.dial = dial
		return nil
	}
}

func defaultDialer() func(context.Context, string, string) (net.Conn, error) {
	dialer := &net.Dialer{}

	return dialer.DialContext
}

func defaultLogger() Logger {
	return Adapt(logrus.New().WithField("app", "spiceProxy"))
}
