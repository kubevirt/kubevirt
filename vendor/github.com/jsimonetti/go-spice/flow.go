package spice

import (
	"net"
	"sync"
	"time"
)

// flow is a connection pipe to couple tenant to compute connections
type flow struct {
	tenant  net.Conn
	compute net.Conn
	timeout time.Duration
	bufSize int
}

// newFlow returns a new flow
func newFlow(tenant net.Conn, compute net.Conn) *flow {
	flow := &flow{
		tenant:  tenant,
		compute: compute,
		timeout: 10 * time.Second,
		bufSize: 4096 * 16,
	}
	return flow
}

// SetTimeout will set the write deadlines of the connections
func (f *flow) SetTimeout(timeout time.Duration) {
	f.timeout = timeout
}

// SetBufferSize will set the buffer size for the connection reads
func (f *flow) SetBufferSize(size int) {
	f.bufSize = size
}

// Pipe will start piping the connections together
func (f *flow) Pipe() error {
	f.pipe(f.compute, f.tenant)
	return nil
}

func (f *flow) pipe(src, dst net.Conn) (sent, received int64) {
	if src == nil || dst == nil {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		sent = f.pipeAndClose(src, dst)
		wg.Done()
	}()
	go func() {
		received = f.pipeAndClose(dst, src)
		wg.Done()
	}()
	wg.Wait()
	return
}

func (f *flow) pipeAndClose(src, dst net.Conn) (copied int64) {
	defer dst.Close()
	buf := make([]byte, f.bufSize)
	for {
		n, err := src.Read(buf)
		copied += int64(n)
		if n > 0 {
			dst.SetWriteDeadline(time.Now().Add(f.timeout))
			if _, err := dst.Write(buf[0:n]); err != nil {
				break
			}
		}
		if err != nil {
			break
		}
	}
	return
}
