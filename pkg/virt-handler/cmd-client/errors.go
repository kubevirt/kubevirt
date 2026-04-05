/*
Copyright The KubeVirt Authors.
SPDX-License-Identifier: Apache-2.0
*/

package cmdclient

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"os"
	"syscall"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cmdv1 "kubevirt.io/kubevirt/pkg/handler-launcher-com/cmd/v1"
)

func IsUnimplemented(err error) bool {
	if grpcStatus, ok := status.FromError(err); ok {
		if grpcStatus.Code() == codes.Unimplemented {
			return true
		}
	}
	return false
}

func handleError(err error, cmdName string, response *cmdv1.Response) error {
	if IsDisconnected(err) {
		return err
	} else if IsUnimplemented(err) {
		return err
	} else if err != nil {
		msg := fmt.Sprintf("unknown error encountered sending command %s: %s", cmdName, err.Error())
		return fmt.Errorf("%s", msg)
	} else if response != nil && !response.Success {
		return fmt.Errorf("server error. command %s failed: %q", cmdName, response.Message)
	}
	return nil
}

func IsDisconnected(err error) bool {
	if err == nil {
		return false
	}

	if err == rpc.ErrShutdown || err == io.ErrUnexpectedEOF || err == io.EOF {
		return true
	}

	if opErr, ok := err.(*net.OpError); ok {
		if syscallErr, ok := opErr.Err.(*os.SyscallError); ok {
			// catches "connection reset by peer"
			if syscallErr.Err == syscall.ECONNRESET {
				return true
			}
		}
	}

	if grpcStatus, ok := status.FromError(err); ok {

		// see https://github.com/grpc/grpc-go/blob/master/codes/codes.go
		switch grpcStatus.Code() {
		case codes.Canceled:
			// e.g. v1client connection closing
			return true
		}

	}

	return false
}
