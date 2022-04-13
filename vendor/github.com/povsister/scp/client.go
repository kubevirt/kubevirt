package scp

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/ssh"
)

var (
	// ErrNoClientOption indicates a non-nil ClientOption should be provided
	ErrNoClientOption = errors.New("scp: ClientOption is not provided")
)

// Client has the "golang.org/x/crypto/ssh/Client" embedded,
// so it can be used as normal SSH client with additional SCP features.
type Client struct {
	// The underlying ssh client
	*ssh.Client
	// Option for the scp client
	scpOpt *ClientOption
	// a flag to indicate whether it's root user
	rootUser *bool
}

// ClientOption contains several configurable options for SCP client.
type ClientOption struct {
	// Use sudo to run remote scp server.
	// Default: false.
	Sudo bool
	// The scp remote server executable file.
	// Default: "scp".
	//
	// If your scp command is not in the default path,
	// specify it as "/path/to/scp".
	RemoteBinary string
}

// applies the default values if not set
func (o *ClientOption) applyDefault() {
	if len(o.RemoteBinary) == 0 {
		o.RemoteBinary = "scp"
	}
}

// NewClient returns a SSH client with SCP capability.
//
// The serverAddr should be in "host" or "host:port" format.
// If no port is supplied, the default port 22 will be used.
//
// IPv6 serverAddr must be enclosed in square brackets, as in "[::1]" or "[::1]:22"
func NewClient(serverAddr string, sshCfg *ssh.ClientConfig, scpOpt *ClientOption) (*Client, error) {
	c, err := dialServer(serverAddr, sshCfg)
	if err != nil {
		return nil, err
	}

	return newClient(c, scpOpt)
}

// NewClientFromExistingSSH returns a SSH client with SCP capability.
// It reuse the existing SSH connection without dialing
func NewClientFromExistingSSH(existing *ssh.Client, scpOpt *ClientOption) (*Client, error) {
	return newClient(existing, scpOpt)
}

func newClient(ssh *ssh.Client, opt *ClientOption) (*Client, error) {
	if opt == nil {
		return nil, ErrNoClientOption
	}

	opt.applyDefault()
	return &Client{Client: ssh, scpOpt: opt}, nil
}

// dial SSH to given server addr
func dialServer(addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	ep, err := addDefaultPort(addr)
	if err != nil {
		return nil, err
	}
	return ssh.Dial("tcp", ep, cfg)
}

// add default port 22 if needed
func addDefaultPort(addr string) (string, error) {
	_, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.Contains(err.Error(), "missing port") {
			newAddr := net.JoinHostPort(strings.Trim(addr, "[]"), "22")
			// Test the addr again
			if _, _, err = net.SplitHostPort(newAddr); err == nil {
				return newAddr, nil
			}
		}
		return "", fmt.Errorf("error parsing serverAddr: %s", err)
	}
	return addr, nil
}

func (c *Client) isRootUser() bool {
	if c.rootUser != nil {
		// short path
		return *c.rootUser
	}
	result := c.User() == "root"
	c.rootUser = &result
	return *c.rootUser
}
