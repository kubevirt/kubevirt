package safepath

import (
	"container/list"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	"golang.org/x/sys/unix"
)

// JoinAndResolveWithRelativeRoot joins an absolute relativeRoot base path with
// additional elements which have to be kept below the relativeRoot base.
// Relative and absolute links will be resolved relative to the provided rootBase
// and can not escape it.
func JoinAndResolveWithRelativeRoot(rootBase string, elems ...string) (*Path, error) {
	// ensure that rootBase is absolute
	if !filepath.IsAbs(rootBase) {
		return nil, fmt.Errorf("basepath is not absolute: %q", rootBase)
	}

	path := pathRoot
	fifo := newLimitedFifo(256)
	for i := len(elems) - 1; i >= 0; i-- {
		if err := fifo.push(strings.Split(filepath.Clean(elems[i]), pathSeparator)); err != nil {
			return nil, err
		}
	}

	for !fifo.empty() {
		child := fifo.pop()
		var link string
		var err error

		path, link, err = advance(rootBase, path, child)
		if err != nil {
			return nil, err
		}
		if link != "" {
			if err := fifo.push(strings.Split(link, pathSeparator)); err != nil {
				return nil, err
			}
		}
	}

	// Assert that the result is indeed a clean path in the expected format
	// at this point in time.
	finalPath := newPath(rootBase, path)
	fd, err := OpenAtNoFollow(finalPath)
	if err != nil {
		return nil, err
	}
	_ = fd.Close()

	return finalPath, nil
}

type fifo struct {
	ops    uint
	store  *list.List
	maxOps uint
}

func (f *fifo) push(pathElements []string) error {
	for i := len(pathElements) - 1; i >= 0; i-- {
		if f.ops > f.maxOps {
			return fmt.Errorf("more than %v path elements evaluated", f.maxOps)
		}
		if pathElements[i] == "" {
			continue
		}
		f.ops++
		f.store.PushFront(pathElements[i])
	}
	return nil
}

func (f *fifo) pop() string {
	if val := f.store.Front(); val != nil {
		f.store.Remove(val)
		return val.Value.(string)
	}
	return ""
}

func (f *fifo) empty() bool {
	return f.store.Len() == 0
}

// newLimitedFifo creates a fifo with a maximum enqueue limit to
// avoid abuse on filepath operations.
func newLimitedFifo(maxOps uint) *fifo {
	return &fifo{
		store:  list.New(),
		maxOps: maxOps,
	}
}

// OpenAtNoFollow safely opens a filedescriptor to a path relative to
// rootBase. Any symlink encountered will be treated as invalid and the operation will be aborted.
// This works best together with a path first resolved with JoinAndResolveWithRelativeRoot
// which can resolve relative paths and symlinks.
func OpenAtNoFollow(path *Path) (file *File, err error) {
	fd, err := open(path.rootBase)
	if err != nil {
		return nil, fmt.Errorf("failed opening path %v: %w", path, err)
	}
	for _, child := range strings.Split(filepath.Clean(path.relativePath), pathSeparator) {
		if child == "" {
			continue
		}
		newfd, err := openat(fd, child)
		_ = syscall.Close(fd) // always close the parent after the lookup
		if err != nil {
			return nil, fmt.Errorf("failed opening %s for path %v: %w", child, path, err)
		}
		fd = newfd
	}
	return &File{fd: fd, path: path}, nil
}

func ChmodAtNoFollow(path *Path, mode os.FileMode) error {
	f, err := OpenAtNoFollow(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return os.Chmod(f.SafePath(), mode)
}

func ChownAtNoFollow(path *Path, uid, gid int) error {
	f, err := OpenAtNoFollow(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return os.Chown(f.SafePath(), uid, gid)
}

func ChpermAtNoFollow(path *Path, uid, gid int, mode os.FileMode) error {
	// first set the ownership, to avoid that someone may change back the file mode
	// after we set it. This is necessary if the file got somehow created without
	// the right owners, maybe with malicious intent.
	if err := ChownAtNoFollow(path, uid, gid); err != nil {
		return err
	}
	if err := ChmodAtNoFollow(path, mode); err != nil {
		return err
	}
	return nil
}

func MkdirAtNoFollow(path *Path, dirName string, mode os.FileMode) error {
	if err := isSingleElement(dirName); err != nil {
		return err
	}
	f, err := OpenAtNoFollow(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := unix.Mkdirat(f.fd, dirName, uint32(mode)); err != nil {
		return fmt.Errorf("failed making the directory %v: %w", path, err)
	}
	return nil
}

// TouchAtNoFollow safely touches a file relative to
// rootBase. The additional elements form the relative path. Any symlink
// encountered will be treated as invalid and the operation will be aborted.
// This works best together with a path first resolved with JoinAndResolveWithRelativeRoot
// which can resolve relative paths to their real path without symlinks.
// If the target file exists already, the function will fail.
func TouchAtNoFollow(path *Path, fileName string, mode os.FileMode) (err error) {
	if err := isSingleElement(fileName); err != nil {
		return err
	}
	parent, err := OpenAtNoFollow(path)
	if err != nil {
		return err
	}
	defer parent.Close()
	fd, err := touchat(parent.fd, fileName, uint32(mode))
	if err != nil {
		return err
	}
	_ = syscall.Close(fd)
	return nil
}

func MknodAtNoFollow(path *Path, fileName string, mode os.FileMode, dev uint64) (err error) {
	if err := isSingleElement(fileName); err != nil {
		return err
	}
	parent, err := OpenAtNoFollow(path)
	if err != nil {
		return err
	}
	defer parent.Close()
	return mknodat(parent.fd, fileName, uint32(mode), dev)
}

func StatAtNoFollow(path *Path) (os.FileInfo, error) {
	pathFd, err := OpenAtNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer pathFd.Close()
	return os.Stat(pathFd.SafePath())
}

func GetxattrNoFollow(path *Path, attr string) ([]byte, error) {
	var ret []byte
	pathFd, err := OpenAtNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer pathFd.Close()
	size, err := syscall.Getxattr(pathFd.SafePath(), attr, ret)
	if err != nil {
		return nil, err
	}
	ret = make([]byte, size)
	_, err = syscall.Getxattr(pathFd.SafePath(), attr, ret)
	if err != nil {
		return nil, err
	}

	return ret[:len(ret)-1], nil
}

type File struct {
	fd   int
	path *Path
}

func (f *File) Close() error {
	return syscall.Close(f.fd)
}

func (f *File) String() string {
	return f.Path().String()
}

// SafePath returns a path pointing to the associated file descriptor.
// It is safe to reuse this path without additional checks. The kernel
// will ensure that this path always points to the resolved file.
// To operate on the file just use os.Open and related calls.
func (f *File) SafePath() string {
	return path(f.fd)
}

func (f *File) Path() *Path {
	return f.path
}

// Path is a path which was at the time of creation a real path
// re
type Path struct {
	rootBase     string
	relativePath string
}

// Raw returns an "unsafe" path. It's properties are not safe to use without certain precautions.
// It exposes no access functions. All access happens via functions in the "unsafepath" package.
func (p *Path) Raw() *unsafepath.Path {
	return unsafepath.New(p.rootBase, p.relativePath)
}

func (p *Path) IsRoot() bool {
	return unsafepath.UnsafeAbsolute(p.Raw()) == pathRoot
}

// AppendAndResolveWithRelativeRoot returns a new path with the passed elements resolve relative
// to the current absolute path.
func (p *Path) AppendAndResolveWithRelativeRoot(relativeRootElems ...string) (*Path, error) {
	tmpPath, err := JoinAndResolveWithRelativeRoot(unsafepath.UnsafeAbsolute(p.Raw()), relativeRootElems...)
	if err != nil {
		return nil, err
	}

	newPath := newPath(p.rootBase, filepath.Join(p.relativePath, tmpPath.relativePath))
	fd, err := OpenAtNoFollow(newPath)
	if err != nil {
		return nil, err
	}
	_ = fd.Close()

	return newPath, err
}

func (p *Path) String() string {
	return fmt.Sprintf("root: %v, relative: %v", p.rootBase, p.relativePath)
}

// ExecuteNoFollow opens the file in question and provides the file descriptor path as safePath string.
// This safePath string can be (re)opened with normal os.* operations. The file descriptor path is
// managed by the kernel and there is no way to inject malicious symlinks.
func (p *Path) ExecuteNoFollow(callback func(safePath string) error) error {
	f, err := OpenAtNoFollow(p)
	if err != nil {
		return err
	}
	defer f.Close()
	return callback(f.SafePath())
}

// DirNoFollow returns the parent directory of the safepath.Path as safepath.Path.
func (p *Path) DirNoFollow() (*Path, error) {
	if len(p.relativePath) == 0 {
		return nil, fmt.Errorf("already at relative root, can't get parent")
	}
	newPath := newPath(p.rootBase, filepath.Dir(p.relativePath))
	return newPath, nil
}

// Base returns the basename of the relative untrusted part of the safepath.
func (p *Path) Base() (string, error) {
	if len(p.relativePath) == 0 {
		return "", fmt.Errorf("already at relative root, can't get parent")
	}
	return filepath.Base(p.relativePath), nil
}

func newPath(rootBase, relativePath string) *Path {
	return &Path{
		rootBase:     rootBase,
		relativePath: filepath.Join("/", relativePath),
	}
}

// NewFileNoFollow assumes that a real path to a file is given. It will validate that
// the file is indeed absolute by doing the following checks:
//   - ensure that the path is absolute
//   - ensure that the path does not container relative path elements
//   - ensure that no symlinks are provided
//
// It will return the opened file which contains a link to a safe-to-use path
// to the file, which can't be tampered with. To operate on the file just use os.Open and related calls.
func NewFileNoFollow(path string) (*File, error) {
	if filepath.Clean(path) != path || !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path %q must be absolute and must not contain relative elements", path)
	}
	p := newPath("/", path)
	return OpenAtNoFollow(p)
}

// NewPathNoFollow is a convenience method to get out of a supposedly link-free path a safepath.Path.
// If there is a symlink included the command will fail.
func NewPathNoFollow(path string) (*Path, error) {
	fd, err := NewFileNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	return fd.Path(), nil
}

// JoinNoFollow joins the root path with the given additional path.
// If the additional path element is not a real path (like containing symlinks), it fails.
func JoinNoFollow(rootPath *Path, path string) (*Path, error) {
	if filepath.Clean(path) != path || path == "" {
		return nil, fmt.Errorf("path %q must not contain relative elements and must not be empty", path)
	}
	p := newPath(unsafepath.UnsafeAbsolute(rootPath.Raw()), path)
	f, err := OpenAtNoFollow(p)
	if err != nil {
		return nil, err
	}
	return f.Path(), f.Close()
}

func isSingleElement(path string) error {
	cleanedPath := filepath.Clean(path)
	if cleanedPath != path || strings.ContainsAny(path, pathSeparator) {
		return fmt.Errorf("path %q must be a single non-relative path segment", path)
	}
	switch path {
	case "", "..", ".":
		return fmt.Errorf("path %q must be a single non-relative path segment", path)
	default:
		return nil
	}
}

// UnlinkAtNoFollow allows deleting the specified file or directory (directory must be empty to succeed).
func UnlinkAtNoFollow(path *Path) error {
	parent, err := path.DirNoFollow()
	if err != nil {
		return err
	}
	basename, err := path.Base()
	if err != nil {
		return nil
	}
	info, err := StatAtNoFollow(path)
	if err != nil {
		return err
	}
	fd, err := OpenAtNoFollow(parent)
	if err != nil {
		return err
	}
	defer fd.Close()

	options := 0
	if info.IsDir() {
		// if dir is empty we can delete it with AT_REMOVEDIR
		options = unix.AT_REMOVEDIR
	}
	if err = unlinkat(fd.fd, basename, options); err != nil {
		return fmt.Errorf("failed unlinking path %v: %w", path, err)
	}
	return nil
}

// ListenUnixNoFollow safely creates a socket in user-owned path
// Since there exists no socketat on unix, first a safe delete is performed,
// then the socket is created.
func ListenUnixNoFollow(socketDir *Path, socketName string) (net.Listener, error) {
	if err := isSingleElement(socketName); err != nil {
		return nil, err
	}

	addr, err := net.ResolveUnixAddr("unix", filepath.Join(unsafepath.UnsafeAbsolute(socketDir.Raw()), socketName))
	if err != nil {
		return nil, err
	}

	socketPath, err := JoinNoFollow(socketDir, socketName)
	if err == nil {
		// This ensures that we don't allow unlinking arbitrary files
		if err := UnlinkAtNoFollow(socketPath); err != nil {
			return nil, fmt.Errorf("failed unlinking socket %v: %w", socketPath, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return nil, err
	}

	// Ensure that the socket path is a real path
	// this does not 100% remove the chance of
	// having a socket created at the wrong place, but it makes it unlikely
	_, err = JoinNoFollow(socketDir, socketName)
	if err != nil {
		return nil, err
	}
	return listener, nil
}
