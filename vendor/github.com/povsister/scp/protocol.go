package scp

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	// DebugMode controls the debug output.
	// If true, the debug information of remote scp server
	// will be printed in Stderr.
	DebugMode = false
)

type sessionStream struct {
	In     io.WriteCloser
	Out    io.Reader
	ErrOut io.Reader
}

func (s *sessionStream) Close() error {
	return s.In.Close()
}

func (c *Client) sessionAndStream() (*ssh.Session, *sessionStream, error) {
	s, err := c.NewSession()
	if err != nil {
		return nil, nil, err
	}
	ss := &sessionStream{}
	for _, f := range []func() error{
		func() (err error) {
			ss.In, err = s.StdinPipe()
			return
		},
		func() (err error) {
			ss.Out, err = s.StdoutPipe()
			return
		},
		func() (err error) {
			ss.ErrOut, err = s.StderrPipe()
			return
		},
	} {
		if err = f(); err != nil {
			return nil, nil, err
		}
	}
	return s, ss, nil
}

// represents the remote scp server working mode
type scpServerMode string

const (
	// Used like "scp user@remote.server:~/something ./"
	scpRemoteToLocal scpServerMode = "f"
	// Used like "scp ./something user@remote.server:~/"
	scpLocalToRemote scpServerMode = "t"
)

type remoteServerOption struct {
	Mode      scpServerMode
	Recursive bool
	Preserve  bool // preserve time and modes
}

func (c *Client) launchScpServerOnRemote(o remoteServerOption, s *ssh.Session, st *sessionStream, remotePath string, readyCh chan<- struct{}, errCh chan<- error) {
	remoteExec := c.scpOpt.RemoteBinary
	if c.scpOpt.Sudo && !c.isRootUser() {
		remoteExec = fmt.Sprintf("sudo %s", c.scpOpt.RemoteBinary)
	}

	flags := []string{"-q", fmt.Sprintf("-%s", o.Mode)}
	if o.Recursive {
		flags = append(flags, "-r")
	}
	if o.Preserve {
		flags = append(flags, "-p")
	}
	if DebugMode {
		flags = append(flags, "-v")
	}
	cmd := fmt.Sprintf("%s %s '%s'", remoteExec, strings.Join(flags, " "), remotePath)
	err := s.Start(cmd)
	if err != nil {
		errCh <- fmt.Errorf("error executing command %q on remote: %s", cmd, err)
		return
	}
	<-remoteServerReady(o.Mode, st)
	close(readyCh)
	err = s.Wait()
	if err != nil {
		errCh <- fmt.Errorf("unexpected remote scp server failure: %s", err)
		return
	}
}

func remoteServerReady(mode scpServerMode, s *sessionStream) <-chan struct{} {
	ch := make(chan struct{})
	switch mode {
	case scpLocalToRemote:
		go func() {
			defer func() {
				// does not care about the panic msg.
				recover()
				close(ch)
			}()
			// read the first OK response from remote receiver server
			checkResponse(s)
		}()
	case scpRemoteToLocal:
		// for remote sending server.
		// It doe not send back any thing until
		// the first OK response is received.
		close(ch)
	default:
		panicf("programmer error: unknown scpServerMode %q", mode)
	}
	return ch
}

type transferType string

const (
	// indicate a timestamp
	timestamp transferType = "timestamp"
	// indicate a file transfer
	file transferType = "file"
	// indicate a directory transfer
	directory transferType = "directory"
	// exit the scp server (at the root directory)
	// or back to the previous directory (equals to "cd ..")
	exit transferType = "exit"
)

type sendJob struct {
	Type         transferType
	Size         int64
	Reader       io.Reader // the content reader
	Destination  string    // must be file or directory name. Path is not supported in send
	Perm         os.FileMode
	AccessTime   *time.Time // can be nil
	ModifiedTime *time.Time // must be both set or nil with atime
	close        bool       // close the reader after using it. internal usage.
}

var (
	// represent a "E" signal
	exitJob = sendJob{Type: exit}
)

// it accepts a single "sendJob" or "<-chan sendJob"
func (c *Client) sendToRemote(cancel context.CancelFunc, jobs interface{}, stream *sessionStream, finished chan<- struct{}, errCh chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			if cancel != nil {
				// cancel the traverse goroutine on error
				cancel()
			}
			// empty the chan and close the fd in buffer
			if jobCh, ok := jobs.(<-chan sendJob); ok {
				for {
					j, ok := <-jobCh
					if !ok {
						break
					}
					if j.close && j.Reader != nil {
						if rc, ok := j.Reader.(io.ReadCloser); ok {
							_ = rc.Close()
						}
					}
				}
			}
			errCh <- fmt.Errorf("%v", r)
		}
	}()

	setupDebug(stream.ErrOut)

	switch js := jobs.(type) {
	case sendJob:
		handleSend(js, stream)
	case <-chan sendJob:
		for {
			j, ok := <-js
			if !ok {
				// jobCh closed
				break
			}
			handleSend(j, stream)
		}
	default:
		panicf("programmer error: unknown type %T", jobs)
	}

	if finished != nil {
		close(finished)
	}
}

func handleSend(j sendJob, stream *sessionStream) {
	switch j.Type {
	case file:
		// close if required
		if j.close {
			if rc, ok := j.Reader.(io.ReadCloser); ok {
				defer rc.Close()
			}
		}
		// set timestamp for the next coming file
		if j.AccessTime != nil && j.ModifiedTime != nil {
			sendTimestamp(j, stream)
		}
		// send signal
		_, err := fmt.Fprintf(stream.In, "C0%o %d %s\n", j.Perm, j.Size, j.Destination)
		if err != nil {
			panicf("error sending signal C: %s", err)
		}
		checkResponse(stream)
		// send file
		_, err = io.Copy(stream.In, j.Reader)
		if err != nil {
			panicf("error sending file %q: %s", j.Destination, err)
		}
		_, err = fmt.Fprint(stream.In, "\x00")
		if err != nil {
			panicf("error finishing file %q: %s", j.Destination, err)
		}
		checkResponse(stream)

	case directory:
		if j.AccessTime != nil && j.ModifiedTime != nil {
			sendTimestamp(j, stream)
		}
		// size is always 0 for directory
		_, err := fmt.Fprintf(stream.In, "D0%o 0 %s\n", j.Perm, j.Destination)
		if err != nil {
			panicf("error sending signal D: %s", err)
		}
		checkResponse(stream)

	case exit:
		_, err := fmt.Fprintf(stream.In, "E\n")
		if err != nil {
			panicf("error sending signal E: %s", err)
		}
		checkResponse(stream)
	default:
		panicf("programmer error: unknown transferType %q", j.Type)
	}
}

func sendTimestamp(j sendJob, stream *sessionStream) {
	_, err := fmt.Fprintf(stream.In, "T%d 0 %d 0\n", j.ModifiedTime.Unix(), j.AccessTime.Unix())
	if err != nil {
		panicf("error sending signal T: %s", err)
	}
	checkResponse(stream)
}

func setupDebug(errReader io.Reader) {
	if errReader == nil {
		return
	}
	if DebugMode {
		go io.Copy(os.Stderr, errReader)
	} else {
		go io.Copy(ioutil.Discard, errReader)
	}
}

type receiveJob struct {
	Type      transferType
	Writer    io.Writer
	Path      string
	Perm      os.FileMode
	close     bool // close writer
	recursive bool // recursive receive
}

func (c *Client) receiveFromRemote(job receiveJob, stream *sessionStream, finished chan<- struct{}, errCh chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			if job.close {
				if rc, ok := job.Writer.(io.WriteCloser); ok {
					_ = rc.Close()
				}
			}
			errCh <- fmt.Errorf("%v", r)
		}
	}()

	setupDebug(stream.ErrOut)

	switch job.Type {
	case file:
		handleReceive(job, stream)
	case directory:
		handleReceive(job, stream)
	default:
		panicf("programmer error: unsupported receive type %q", job.Type)
	}

	if finished != nil {
		close(finished)
	}
}

func handleReceive(recv receiveJob, stream *sessionStream) {
	var path []string
	if len(recv.Path) > 0 {
		path = append(path, recv.Path)
	}

	// signal the remote to start
	sendResponse(stream.In, statusOK)
	// a flag to indicate if first loop
	firstLoop := true

	for {
		j := readTransaction(stream)

		if !recv.recursive && j.Type == directory {
			sendResponse(stream.In, statusErr, "protocol error: directory received in non-recursive mode")
		}

		switch j.Type {
		case directory:
			// On the first loop. skip the root directory transfer
			if firstLoop {
				// The root directory should already exists
				setTimestamp(stream.In, recv.Path, j.ModifiedTime, j.AccessTime)
				sendResponse(stream.In, statusOK)
				firstLoop = false
				continue
			}
			path = append(path, j.Destination)
			toOpen := filepath.Join(path...)
			mkdir(stream.In, toOpen, j.Perm)
			setTimestamp(stream.In, toOpen, j.ModifiedTime, j.AccessTime)
			// confirm D command
			sendResponse(stream.In, statusOK)
		case file:
			// recursive recv
			if recv.recursive && len(path) >= 1 && recv.Writer == nil {
				toOpen := filepath.Join(append(path, j.Destination)...)
				fd := openFile(stream.In, toOpen, j.Perm)
				saveFile(fd, stream, j.Size, true)
				setTimestamp(stream.In, toOpen, j.ModifiedTime, j.AccessTime)
			} else {
				// single file transfer
				// write to buffer
				if recv.Writer == nil {
					// if buffer is not set. Means it's a file transfer.
					perm := j.Perm
					if recv.Perm != 0 {
						// overrides the file permission bits if needed
						perm = recv.Perm
					}
					recv.Writer = openFile(stream.In, recv.Path, perm)
				}
				saveFile(recv.Writer, stream, j.Size, recv.close)
				// if path is specified. Means its a file transfer.
				if len(recv.Path) > 0 {
					setTimestamp(stream.In, recv.Path, j.ModifiedTime, j.AccessTime)
				}
			}
			// confirm recv ok
			sendResponse(stream.In, statusOK)
			if !recv.recursive {
				return
			}
		case exit:
			sendResponse(stream.In, statusOK)
			if recv.recursive {
				if l := len(path); l >= 2 {
					// exit to parent directory
					path = path[0 : l-1]
				} else {
					// exit to root directory.
					// Means exit
					return
				}
			} else {
				// buffer or single file write. just exit
				return
			}
		default:
			sendResponse(stream.In, statusErr, "programmer error: unexpected receive transaction ", string(j.Type))
		}
		if firstLoop {
			firstLoop = false
		}
	}
}

func openFile(w io.Writer, path string, perm os.FileMode) *os.File {
	fd, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		sendResponse(w, statusErr, err.Error())
	}
	return fd
}

// mkdir makes the dir if not exist
func mkdir(w io.Writer, path string, perm os.FileMode) {
	if stat, err := os.Stat(path); err != nil {
		if err = os.Mkdir(path, perm); err != nil {
			sendResponse(w, statusErr, err.Error())
		}
	} else {
		if !stat.IsDir() {
			sendResponse(w, statusErr, path, "is not a directory")
		}
	}
}

// setTimestamp sets the atime and mtime for file/directory
func setTimestamp(w io.Writer, path string, mtime, atime *time.Time) {
	if mtime == nil || atime == nil {
		return
	}
	err := os.Chtimes(path, *atime, *mtime)
	if err != nil && w != nil {
		sendResponse(w, statusErr, fmt.Sprintf("error setting mtime and atime %s ", path), err.Error())
	}
}

// saveFile saves the file received to io.Writer.
// It will first confirm the command C and then start recv
func saveFile(w io.Writer, stream *sessionStream, size int64, close bool) {
	defer func() {
		if close {
			if rc, ok := w.(io.ReadCloser); ok {
				_ = rc.Close()
			}
		}
		if r := recover(); r != nil {
			sendResponse(w, statusErr, fmt.Sprintf("%s", r))
		}
	}()

	// confirm C command and starting transfer
	sendResponse(stream.In, statusOK)

	n, err := io.CopyN(w, stream.Out, size)
	if err != nil {
		panicf("error reading file from remote: %s", err)
	}
	if n != size {
		panicf("excepting file length %d, but got %d", size, n)
	}
	readDelimiter(stream.Out, 0, 1)
}

// read commands as a sendJob from remote
func readTransaction(stream *sessionStream) sendJob {
	// read the first command
	c := readCommand(stream)
	j := sendJob{}
	// read next if T command
	if c.Type == timestamp {
		sendResponse(stream.In, statusOK)
		next := readCommand(stream)
		switch next.Type {
		case file, directory:
			j.Type, j.Destination = next.Type, next.Destination
			j.ModifiedTime, j.AccessTime = c.Mtime, c.Atime
			j.Perm, j.Size = next.Perm, next.Size
		case timestamp:
			sendResponse(stream.In, statusErr, "protocol error: unexpected T after T")
		case exit:
			sendResponse(stream.In, statusErr, "protocol error: unexpected E after T")
		default:
			panic("programmer error: impossible switch case")
		}
	} else {
		j.Type, j.Destination = c.Type, c.Destination
		j.Perm, j.Size = c.Perm, c.Size
	}
	return j
}

type command struct {
	Type transferType
	// common field for file/directory
	Destination string
	Perm        os.FileMode
	// timestamp
	Mtime, Atime *time.Time
	// file
	Size int64
}

var (
	// the command signal length in byte
	commandLenByte = 1
	// the length limit for destination.
	// Typically the filename/directory is limited to 255 characters
	destinationLimit = 1<<8 - 1
	// the length limit for representing a file size in byte.
	// Using the length of MaxInt64 (9223372036854775807)
	fileSizeLenLimit = 19
)

func readCommand(stream *sessionStream) command {
	defer func() {
		if r := recover(); r != nil {
			errStr := fmt.Sprintf("%v", r)
			// if protocol error occurs.
			// Send it back to remote server
			if strings.HasPrefix(errStr, "protocol error") {
				// sendResponse continue panic on Err
				sendResponse(stream.In, statusErr, errStr)
			}
			// continue panic
			panic(errStr)
		}
	}()

	buf := make([]byte, commandLenByte)
	n, err := stream.Out.Read(buf)
	if err != nil {
		panicf("error receiving signal from remote scp server: %s", err)
	}
	if n != commandLenByte {
		panicf("expecting read %d byte, but got %d", commandLenByte, n)
	}

	var c command
	switch string(buf) {
	case "T":
		c.Type = timestamp
		c.Mtime = readUnixTimestamp(stream.Out)
		readDelimiter(stream.Out, ' ', 1)
		readDelimiter(stream.Out, '0', 1)
		readDelimiter(stream.Out, ' ', 1)
		c.Atime = readUnixTimestamp(stream.Out)
		readDelimiter(stream.Out, ' ', 1)
		readDelimiter(stream.Out, '0', 1)
		readDelimiter(stream.Out, '\n', 1)
	case "C":
		c.Type = file
		c.Perm = readPerm(stream.Out)
		readDelimiter(stream.Out, ' ', 1)
		c.Size = readFileSize(stream.Out)           // reads Delimiter ' ' as well
		c.Destination = readDestination(stream.Out) // reads Delimiter '\n' as well
	case "D":
		c.Type = directory
		c.Perm = readPerm(stream.Out)
		readDelimiter(stream.Out, ' ', 1)
		readDelimiter(stream.Out, '0', 1)
		readDelimiter(stream.Out, ' ', 1)
		c.Destination = readDestination(stream.Out)
	case "E":
		c.Type = exit
		readDelimiter(stream.Out, '\n', 1)
	case string(statusOK):
		// do nothing
	case string(statusErr), string(statusFatal):
		br := bufio.NewReader(stream.Out)
		reason, err := br.ReadString('\n')
		if err != nil {
			panicf("error reading failure reason from remote scp server: %s", err)
		}
		panic(trimErrMsg(reason))
	default:
		panicf("protocol error: unknown signal %s", buf)
	}

	return c
}

// send a response back to remote scp server.
// if the response is not statusOK, it will panic after send.
func sendResponse(w io.Writer, respType responseStatus, msg ...string) {
	var resp []byte
	if respType != statusOK {
		if len(msg) > 0 {
			for _, m := range msg {
				resp = append(resp, []byte(m)...)
			}
		}
		resp = append(resp, '\n')
	}

	header := []byte(respType)
	var err error
	if respType != statusOK {
		_, err = w.Write(append(header, append([]byte("scp: "), resp...)...))
	} else {
		_, err = w.Write(header)
	}
	if err != nil {
		panicf("error sending response to remote scp server: %s", err)
	}
	if respType != statusOK {
		panic(string(resp))
	}
}

func readUnixTimestamp(r io.Reader) *time.Time {
	unixTime := make([]byte, 10)
	n, err := r.Read(unixTime)
	if err != nil {
		panicf("error receiving timestamp from remote scp server: %s", err)
	}
	if n != 10 {
		panicf("expecting read %d byte, but got %d", 10, n)
	}
	unix, err := strconv.ParseInt(string(unixTime), 10, 64)
	if err != nil {
		panicf("protocol error: invalid timestamp %s %s", unixTime, err)
	}
	t := time.Unix(unix, 0)
	return &t
}

func readDelimiter(r io.Reader, sep byte, repeat int) {
	read := make([]byte, repeat)
	n, err := r.Read(read)
	if err != nil {
		panicf("error reading from remote scp server: %s", err)
	}
	if n != repeat {
		panicf("expecting read %d bytes, but got %d", 10, n)
	}
	for _, perByte := range read {
		if perByte != sep {
			panicf("protocol error: expecting delimiter %s but got %s", string(sep), read)
		}
	}
}

func readPerm(r io.Reader) os.FileMode {
	read := make([]byte, 4)
	n, err := r.Read(read)
	if err != nil {
		panicf("error reading mode from remote scp server: %s", err)
	}
	if n != 4 {
		panicf("expecting read %d byte, but got %d", 4, n)
	}
	mode, err := strconv.ParseInt(string(read), 8, 32)
	if err != nil {
		panicf("protocol error: invalid mode %s %s", read, err)
	}
	return os.FileMode(mode)
}

func readFileSize(r io.Reader) (result int64) {
	readByte := make([]byte, 1)
	var n, totalLen int
	var err error
	var read []byte
	for {
		if totalLen > fileSizeLenLimit {
			panicf("file size exceeding limit: %s", read)
		}
		n, err = r.Read(readByte)
		if err != nil {
			panicf("error reading file size from remote scp server: %s", err)
		}
		if n != 1 {
			panicf("expecting read %d byte, but got %d", 1, n)
		}
		totalLen += n
		if readByte[0] == ' ' {
			break
		} else if readByte[0] >= '0' && readByte[0] <= '9' {
			read = append(read, readByte[0])
		} else {
			panicf("protocol error: invalid file size num %s", readByte)
		}
	}

	if len(read) == 0 {
		panicf("protocol error: file size missing")
	}

	result, err = strconv.ParseInt(string(read), 10, 64)
	if err != nil {
		panicf("protocol error: invalid file size %s %s", read, err)
	}
	return
}

func readDestination(r io.Reader) string {
	readByte := make([]byte, 1)
	var n, totalLen int
	var err error
	var read []byte
	for {
		if totalLen > destinationLimit {
			panicf("file or directory name is too long: %s", read)
		}
		n, err = r.Read(readByte)
		if err != nil {
			panicf("error reading file or directory name from remote scp server: %s", err)
		}
		if n != 1 {
			panicf("expecting read %d byte, but got %d", 1, n)
		}
		totalLen += n
		if readByte[0] == '\n' {
			break
		} else {
			read = append(read, readByte[0])
		}
	}

	if len(read) == 0 {
		panicf("protocol error: empty file or directory name")
	}

	return string(read)
}

type responseStatus string

const (
	// There are 3 types of responses that the remote can send back:
	// OK, Error and Fatal
	//
	// The difference between Error and Fatal is that the connection is not closed by the remote.
	// However, a Error can indicate a file transfer failure (such as invalid destination directory)
	//
	// All responses except for the OK always have a message (although they can be empty)

	// Normal OK
	statusOK responseStatus = "\x00"
	// A failure operation
	statusErr responseStatus = "\x01"
	// Indicate a Fatal error, though no one actually use it.
	statusFatal responseStatus = "\x02"

	// The byte length for representing a status
	statusByteLen = 1
)

// check response from remote scp
// panic on error
func checkResponse(stream *sessionStream) {
	status := make([]byte, statusByteLen)
	n, err := stream.Out.Read(status)
	if err != nil {
		panicf("error reading server response status: %s", err)
	}
	if n != statusByteLen {
		panicf("expecting read %d byte, but got %d", statusByteLen, n)
	}

	st := responseStatus(status[0:statusByteLen])
	switch st {
	case statusErr, statusFatal:
		buf := bufio.NewReader(stream.Out)
		errMsg, err := buf.ReadString('\n')
		if err != nil {
			panicf("error reading server response message: %s", err)
		}
		panic(trimErrMsg(errMsg))
	case statusOK:
		// status OK, do nothing
	default:
		panicf("unknown server response status %s", st)
	}
}

func trimErrMsg(msg string) string {
	return strings.Trim(strings.TrimPrefix(msg, "scp:"), " ")
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
