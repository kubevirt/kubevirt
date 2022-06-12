package scp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	// DefaultFilePerm holds default permission bits for transferred files.
	DefaultFilePerm = os.FileMode(0644)
	// DefaultDirPerm holds default permission bits for transferred directories.
	DefaultDirPerm = os.FileMode(0755)

	// ErrNoTransferOption indicate a non-nil TransferOption should be provided.
	ErrNoTransferOption = errors.New("scp: TransferOption is not provided")

	// DirectoryPreReads sets the num of pre-read files/directories for recursively transferring a directory.
	// Set it larger may speedup the transfer with lots of small files.
	// Do not set it too large or you will exceed the max open files limit.
	DirectoryPreReads = 10
)

// FileTransferOption holds the transfer options for file.
type FileTransferOption struct {
	// Context for the file transfer.
	// Can be both set with Timeout.
	// Default: no context
	Context context.Context
	// Timeout for transferring the file.
	// Can be both set with Context.
	// Default: 0 (Means no timeout)
	Timeout time.Duration
	// The permission bits for transferred file.
	// Override "PreserveProp" if specified.
	// Default: 0644
	Perm os.FileMode
	// Preserve modification times and permission bits from the original file.
	// Only valid for file transfer.
	// Default: false
	PreserveProp bool
	// Limits the used bandwidth, specified in Kbit/s.
	// Default: 0 (Means no limit)
	// TODO: not implemented yet
	SpeedLimit int64
}

// KnownSize is intended for reader whose size is already known before reading.
type KnownSize interface {
	// return num in bytes
	Size() int64
}

// CopyFileToRemote copies a local file to remote location.
// It will automatically close the file after read.
func (c *Client) CopyFileToRemote(localFile string, remoteLoc string, opt *FileTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	f, err := os.Open(localFile)
	if err != nil {
		return fmt.Errorf("scp: %v", err)
	}
	defer f.Close()

	return c.CopyToRemote(f, remoteLoc, opt)
}

// CopyToRemote copies content from reader to remoteTarget.
// The reader must implement "KnownSize" interface except *os.File.
//
// Currently, it supports following readers:
//   - *os.File
//   - *strings.Reader
//   - *bytes.Reader
// Note that the last part of remoteTarget will be used as filename if unspecified.
//
// It's CALLER'S responsibility to CLOSE the file if an *os.File is supplied.
func (c *Client) CopyToRemote(reader io.Reader, remoteTarget string, opt *FileTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	var size int64
	var fileName, remotePath string
	var mtime, atime *time.Time
	var perm os.FileMode = DefaultFilePerm
	if opt.Perm != 0 {
		perm = opt.Perm
	}

	switch r := reader.(type) {
	case *os.File:
		stat, err := r.Stat()
		if err != nil {
			return fmt.Errorf("scp: error getting file stat %v", err)
		}
		size = stat.Size()
		fileName = stat.Name()
		remotePath = remoteTarget
		if opt.PreserveProp {
			mt, at := stat.ModTime(), time.Now()
			mtime, atime = &mt, &at
			if opt.Perm == 0 {
				perm = stat.Mode()
			}
		}
	default:
		if ks, ok := reader.(KnownSize); ok {
			size = ks.Size()
			fileName = filepath.Base(remoteTarget)
			// ToSlash guarantees "coping from Windows to *unix" works as expected
			remotePath = filepath.ToSlash(filepath.Dir(remoteTarget))
		} else {
			return fmt.Errorf("scp: reader does not implement KnownSize interface")
		}
	}

	session, stream, reusableErrCh, err := c.prepareTransfer(remoteServerOption{Mode: scpLocalToRemote}, remotePath)
	if err != nil {
		return err
	}
	defer session.Close()
	defer stream.Close()

	job := sendJob{
		Type:         file,
		Size:         size,
		Reader:       reader,
		Destination:  fileName,
		Perm:         perm,
		AccessTime:   atime,
		ModifiedTime: mtime,
	}

	finished := make(chan struct{})
	go c.sendToRemote(nil, job, stream, finished, reusableErrCh)

	stopFn, timer := setupTimeout(opt.Timeout)
	defer stopFn()

	select {
	case <-setupContext(opt.Context):
		return opt.Context.Err()
	case <-timer:
		return fmt.Errorf("scp: timeout sending file to remote")
	case err = <-reusableErrCh:
		// remote scp server automatically exits on error
		return fmt.Errorf("scp: %v", err)
	case <-finished:
		c.sendToRemote(nil, exitJob, stream, nil, reusableErrCh)
	}

	return nil
}

// DirTransferOption holds the transfer options for directory.
type DirTransferOption struct {
	// Context for the directory transfer.
	// Can be both set with Timeout.
	// Default: no context
	Context context.Context
	// Timeout for transferring the whole directory.
	// Can be both set with Context.
	// Default: 0 (means no timeout)
	Timeout time.Duration
	// Preserve modification times and modes from the original file/directory.
	// Default: false
	PreserveProp bool
	// Limits the used bandwidth, specified in Kbit/s.
	// Default: 0 (Means no limit)
	// TODO: not implemented yet
	SpeedLimit int64
}

// CopyDirToRemote recursively copies a directory to remoteDir.
func (c *Client) CopyDirToRemote(localDir string, remoteDir string, opt *DirTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	dir, err := os.Open(localDir)
	if err != nil {
		return fmt.Errorf("scp: error opening local dir: %v", err)
	}

	o := remoteServerOption{
		Mode:      scpLocalToRemote,
		Recursive: true,
	}
	session, stream, reusableErrCh, err := c.prepareTransfer(o, remoteDir)
	if err != nil {
		return err
	}
	defer session.Close()
	defer stream.Close()

	cancelSend, jobCh := traverse(opt.Context, dir, opt, reusableErrCh)
	defer cancelSend() // ensure no goroutine leak
	finished := make(chan struct{})
	go c.sendToRemote(cancelSend, jobCh, stream, finished, reusableErrCh)

	stopFn, timer := setupTimeout(opt.Timeout)
	defer stopFn()

	select {
	case <-setupContext(opt.Context):
		return opt.Context.Err()
	case <-timer:
		return fmt.Errorf("scp: timeout recursively sending directory to remote")
	case err = <-reusableErrCh:
		return fmt.Errorf("scp: %v", err)
	case <-finished:
		// don't call exitJob.
		// Because it's generated by traverse automatically.
	}

	return nil
}

// traverse iterates files and directories of fd in specific order.
// Return a chan for jobs.
// The fd will be automatically closed after read.
func traverse(parentCtx context.Context, fd *os.File, opt *DirTransferOption, errCh chan error) (context.CancelFunc, <-chan sendJob) {
	jobCh := make(chan sendJob, DirectoryPreReads)

	pCtx := context.TODO()
	if parentCtx != nil {
		pCtx = parentCtx
	}
	ctx, cancel := context.WithCancel(pCtx)

	go traverseDir(ctx, true, fd, opt, jobCh, errCh)

	return cancel, jobCh
}

func traverseDir(ctx context.Context, rootDir bool, dir *os.File, opt *DirTransferOption, jobCh chan sendJob, errCh chan error) {
	if rootDir {
		defer close(jobCh)
	}

	readFn := func() ([]os.FileInfo, os.FileInfo) {
		defer dir.Close()

		curDirStat, err := dir.Stat()
		if err != nil {
			errCh <- fmt.Errorf("error getting dir stat: %v", err)
			return nil, nil
		}
		list, err := dir.Readdir(-1)
		if err != nil {
			errCh <- fmt.Errorf("error traverse dir: %v", err)
			return nil, nil
		}
		return list, curDirStat
	}
	list, curDirStat := readFn()
	if list == nil || curDirStat == nil {
		return
	}

	deliverDir(ctx, curDirStat, opt, jobCh)

	var subDirs []os.FileInfo
	for i := range list {
		if ctx.Err() != nil {
			return
		}
		fStat := list[i]
		// transfer files first
		if !fStat.IsDir() {
			fd, err := os.Open(filepath.Join(dir.Name(), fStat.Name()))
			if err != nil {
				errCh <- fmt.Errorf("error opening file: %v", err)
				return
			}
			deliverFile(ctx, fd, fStat, opt, jobCh)
		} else {
			subDirs = append(subDirs, fStat)
		}
	}

	// traverse sub dirs
	for i := range subDirs {
		if ctx.Err() != nil {
			return
		}
		dirStat := subDirs[i]
		fd, err := os.Open(filepath.Join(dir.Name(), dirStat.Name()))
		if err != nil {
			errCh <- fmt.Errorf("error opening sub dir: %v", err)
			return
		}
		// recursively transfer the dirs
		traverseDir(ctx, false, fd, opt, jobCh, errCh)
	}

	select {
	case jobCh <- exitJob:
		// exit current directory
	case <-ctx.Done():
		return
	}

}

// deliver a directory transfer
func deliverDir(ctx context.Context, stat os.FileInfo, opt *DirTransferOption, jobCh chan sendJob) {
	j := sendJob{
		Type:        directory,
		Destination: stat.Name(),
		Perm:        DefaultDirPerm,
	}
	if opt.PreserveProp {
		// directory permission bit not available on windows
		if runtime.GOOS != "windows" {
			j.Perm = stat.Mode()
		}
		mt, at := stat.ModTime(), time.Now()
		j.ModifiedTime, j.AccessTime = &mt, &at
	}

	select {
	case jobCh <- j:
		// queue the dir job
	case <-ctx.Done():
		return
	}
}

// deliver a file transfer job.
// close the fd automatically.
func deliverFile(ctx context.Context, fd *os.File, stat os.FileInfo, opt *DirTransferOption, jobCh chan sendJob) {
	j := sendJob{
		Type:        file,
		Size:        stat.Size(),
		Reader:      fd,
		Destination: stat.Name(),
		Perm:        DefaultFilePerm,
		close:       true,
	}
	if opt.PreserveProp {
		j.Perm = stat.Mode()
		mt, at := stat.ModTime(), time.Now()
		j.ModifiedTime, j.AccessTime = &mt, &at
	}
	select {
	case jobCh <- j:
		// queue the file job
	case <-ctx.Done():
		return
	}
}

// helper func to setup a timeout timer.
// 0 means no timeout and the chan will block forever.
func setupTimeout(dur time.Duration) (func(), <-chan time.Time) {
	if dur == 0 {
		return func() {}, make(chan time.Time)
	}
	t := time.NewTimer(dur)
	return func() { t.Stop() }, t.C
}

// helper func to return the ctx.Done() if possible.
// It will return a chan that never close if ctx is nil.
func setupContext(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return make(chan struct{})
	}
	return ctx.Done()
}

// prepare for the transfer. Including setup session/stream and run remote scp command
func (c *Client) prepareTransfer(o remoteServerOption, remotePath string) (*ssh.Session, *sessionStream, chan error, error) {
	session, stream, err := c.sessionAndStream()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("scp: error creating ssh session %v", err)
	}

	errCh := make(chan error, 3)
	serverReady := make(chan struct{})

	go c.launchScpServerOnRemote(o, session, stream, remotePath, serverReady, errCh)

	t := time.NewTimer(10 * time.Second)
	defer t.Stop()
	select {
	case <-t.C:
		return nil, nil, nil, fmt.Errorf("scp: timeout starting remote scp server")
	case err = <-errCh:
		return nil, nil, nil, fmt.Errorf("scp: %v", err)
	case <-serverReady:
	}

	return session, stream, errCh, nil
}

// CopyFromRemote copies a remote file into buffer.
//
// Note that "PreserveProp" and "Perm" option does not take effect in this case.
func (c *Client) CopyFromRemote(remoteFile string, buffer io.Writer, opt *FileTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	if buffer == nil {
		return fmt.Errorf("scp: buffer can not be nil")
	}

	return c.copyFromRemote(remoteFile, "", buffer, opt)
}

func (c *Client) copyFromRemote(remoteFile, localFile string, lw io.Writer, opt *FileTransferOption) error {
	o := remoteServerOption{
		Mode:     scpRemoteToLocal,
		Preserve: opt.PreserveProp,
	}
	session, stream, reusableErrCh, err := c.prepareTransfer(o, remoteFile)
	if err != nil {
		return err
	}
	defer session.Close()
	defer stream.Close()

	finished := make(chan struct{})
	j := receiveJob{
		Type:   file,
		Path:   localFile,
		Writer: lw,
		Perm:   opt.Perm,
		close:  len(localFile) != 0,
	}
	go c.receiveFromRemote(j, stream, finished, reusableErrCh)

	stopFn, timer := setupTimeout(opt.Timeout)
	defer stopFn()

	select {
	case <-setupContext(opt.Context):
		return opt.Context.Err()
	case <-timer:
		return fmt.Errorf("scp: timeout receiving file from remote")
	case err = <-reusableErrCh:
		return fmt.Errorf("scp: %v", err)
	case <-finished:
		// do nothing
	}
	return nil
}

// CopyFileFromRemote copies a remoteFile as localFile.
//
// If localFile does not exist, it will be automatically created.
// If localFile already exists, it will be truncated for writing.
// If localFile is a directory, the name of remoteFile will be used.
//
// For example:
//   - CopyFileFromRemote("/remote/file1", "/local/fileNotExist", &FileTransferOption)
//     - Result: "/remote/file1" -> "/local/file2"
//       The "fileNotExist" will be created.
//
//   - CopyFileFromRemote("/remote/file1", "/local/fileExist", &FileTransferOption)
//     - Result: "/remote/file1" -> "/local/fileExist"
//       The "fileExist" will be truncated for writing.
//
//   - CopyFileFromRemote("/remote/file1", "/local/dir", &FileTransferOption)
//     - Result: "/remote/file1" -> "/local/dir/file1"
//       The "file1" will be used as filename and stored under "/local/dir" directory.
//       Note that "/local/dir" must exist in this case.
func (c *Client) CopyFileFromRemote(remoteFile, localFile string, opt *FileTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	remoteFilename := filepath.Base(remoteFile)

	if stat, err := os.Stat(localFile); err != nil {
		if pStat, err := os.Stat(filepath.Dir(localFile)); err != nil {
			return fmt.Errorf("scp: %s", err)
		} else {
			if !pStat.IsDir() {
				return fmt.Errorf("scp: %s no such file or directory", localFile)
			}
		}
	} else {
		if stat.IsDir() {
			localFile = filepath.Join(localFile, remoteFilename)
		}
	}

	return c.copyFromRemote(remoteFile, localFile, nil, opt)
}

// CopyDirFromRemote recursively copies a remote directory into local directory.
// The localDir must exist before copying.
//
// For example:
//   - CopyDirFromRemote("/remote/dir1", "/local/dir2", &DirTransferOption{})
//     - Results: "remote/dir1/<contents>" -> "/local/dir2/<contents>"
func (c *Client) CopyDirFromRemote(remoteDir, localDir string, opt *DirTransferOption) error {
	if opt == nil {
		return ErrNoTransferOption
	}

	if stat, err := os.Stat(localDir); err != nil {
		return fmt.Errorf("scp: %s", err)
	} else {
		if !stat.IsDir() {
			return fmt.Errorf("scp: %s is not a directory", localDir)
		}
	}

	o := remoteServerOption{
		Mode:      scpRemoteToLocal,
		Recursive: true,
		Preserve:  opt.PreserveProp,
	}
	session, stream, reusableErrCh, err := c.prepareTransfer(o, remoteDir)
	if err != nil {
		return err
	}
	defer session.Close()
	defer stream.Close()

	finished := make(chan struct{})
	j := receiveJob{
		Type:      directory,
		Path:      localDir,
		recursive: true,
	}
	go c.receiveFromRemote(j, stream, finished, reusableErrCh)

	stopFn, timer := setupTimeout(opt.Timeout)
	defer stopFn()

	select {
	case <-setupContext(opt.Context):
		return opt.Context.Err()
	case <-timer:
		return fmt.Errorf("scp: timeout receiving directory from remote")
	case err = <-reusableErrCh:
		return fmt.Errorf("scp: %v", err)
	case <-finished:
		// do nothing
	}

	return nil
}
