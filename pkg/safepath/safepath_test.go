package safepath

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"syscall"

	"kubevirt.io/kubevirt/pkg/unsafepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type pathBuilder struct {
	segments     [][]string
	relativeRoot string
	systemRoot   string
}

// Path adds a new path segment to the final path to construct
func (p *pathBuilder) Path(path string) *pathBuilder {
	p.segments = append(p.segments, []string{path})
	return p
}

// Link adds a path segemtn and a link location to the final path to construct.
// The link target can be an absolute or a relative path.
func (p *pathBuilder) Link(path string, target string) *pathBuilder {
	p.segments = append(p.segments, []string{path, target})
	return p
}

// new returns a new path builder with the given relative root prefix.
func new(root string) *pathBuilder {
	return &pathBuilder{segments: [][]string{}, relativeRoot: root}
}

// RelativeRoot returns the final full relative root path.
// Must be called after Builder() to be valid.
func (p *pathBuilder) RelativeRoot() string {
	return filepath.Join(p.systemRoot, p.relativeRoot)
}

// SystemRoot returns the emulated system root path, where the
// RelativeRoot path is a child of.
// Must be called after Builder() to be valid.
func (p *pathBuilder) SystemRoot() string {
	return p.systemRoot
}

// Build the defined path. Absolute links are prefixed wit the SystemRoot which
// will be the base of a ginkgo managed tmp directory.
func (p *pathBuilder) Build() (string, error) {
	p.systemRoot = GinkgoT().TempDir()
	relativeRoot := filepath.Join(p.systemRoot, p.relativeRoot)
	parent := relativeRoot
	if err := os.MkdirAll(parent, os.ModePerm); err != nil {
		return "", err
	}
	for _, elem := range p.segments {
		parent = filepath.Join(parent, elem[0])
		if len(elem) == 2 {
			link := elem[1]
			if err := os.Symlink(link, parent); err != nil {
				return "", err
			}
		} else {
			if err := os.MkdirAll(parent, os.ModePerm); err != nil {
				return "", err
			}
		}
	}

	relativePath := ""
	for _, elem := range p.segments {
		relativePath = filepath.Join(relativePath, elem[0])
	}

	return relativePath, nil
}

var _ = Describe("safepath", func() {

	DescribeTable("should prevent an escape via", func(builder *pathBuilder, expectedPath string) {
		path, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		constructedPath, err := JoinAndResolveWithRelativeRoot(builder.RelativeRoot(), path)
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeAbsolute(constructedPath.Raw())).To(Equal(filepath.Join(builder.RelativeRoot(), expectedPath)))
	},
		Entry("an absolute link to root subdirectory", new("/var/lib/rel/root").Path("link/back/to").Link("link", "/link"),
			"/link",
		),
		Entry("an absolute link to root", new("/var/lib/rel/root").Path("link/back/to").Link("link", "/"),
			"/",
		),
		Entry("a relative link", new("/var/lib/rel/root").Path("link/back/to").Link("var", "../../../../../"),
			"/",
		),
	)

	DescribeTable("should be able to", func(builder *pathBuilder, expectedPath string) {
		path, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		constructedPath, err := JoinAndResolveWithRelativeRoot(builder.RelativeRoot(), path)
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeAbsolute(constructedPath.Raw())).To(Equal(filepath.Join(builder.RelativeRoot(), expectedPath)))
	},
		Entry("handle relative paths by cutting off at the relative root", new("/var/lib/rel/root").Path("link/back/to").Path("../../../../../"),
			`/`,
		),
		Entry("handle relative legitimate paths", new("/var/lib/rel/root").Path("link/back/to").Path("../../"),
			`/link`,
		),
		Entry("handle legitimate paths with relative symlinks", new("/var/lib/rel/root").Path("link/back/to").Link("test", "../../"),
			`/link`,
		),
		Entry("handle multiple legitimate symlink redirects", new("/var/lib/rel/root").Path("link/back/to").Link("test", "../../").Path("b/c").Link("yeah", "../"),
			`/link/b`,
		),
	)

	It("should detect self-referencing links", func() {
		builder := new("/var/lib/rel/root").Path("link/back/to").Link("test", "../test")
		path, err := builder.Build()
		Expect(err).ToNot(HaveOccurred())
		_, err = JoinAndResolveWithRelativeRoot(builder.RelativeRoot(), path)
		Expect(err).To(HaveOccurred())
	})

	It("should follow a sequence of linked links", func() {
		root := GinkgoT().TempDir()
		relativeRoot := filepath.Join(root, "testroot")
		path := "some/path/to/follow"
		Expect(os.MkdirAll(filepath.Join(relativeRoot, path, "test3", "test4"), os.ModePerm)).To(Succeed())
		Expect(os.Symlink("test3", filepath.Join(relativeRoot, path, "test2"))).To(Succeed())
		Expect(os.Symlink("test2", filepath.Join(relativeRoot, path, "test1"))).To(Succeed())
		// try to reach the test4 directory over the test1 link
		pp, err := JoinAndResolveWithRelativeRoot(relativeRoot, path, "/test1/test4")
		Expect(err).ToNot(HaveOccurred())
		// don't use join to avoid any clean operations
		Expect(unsafepath.UnsafeAbsolute(pp.Raw())).To(Equal(relativeRoot + "/some/path/to/follow/test3/test4"))
	})

	It("should detect too many redirects", func() {
		root := GinkgoT().TempDir()
		relativeRoot := filepath.Join(root, "testroot")
		path := "some/path/to/follow"
		Expect(os.MkdirAll(filepath.Join(relativeRoot, path, "test3", "test4"), os.ModePerm)).To(Succeed())
		Expect(os.Symlink("test3", filepath.Join(relativeRoot, path, "test100"))).To(Succeed())
		for i := 101; i < 401+50; i++ {
			Expect(os.Symlink(fmt.Sprintf("test%d", i-1), filepath.Join(relativeRoot, path, fmt.Sprintf("test%d", i)))).To(Succeed())
		}

		// try to reach the test4 directory over the test1 link
		_, err := JoinAndResolveWithRelativeRoot(relativeRoot, path, "/test435/test4")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("more than 256 path elements evaluated"))
	})

	It("should not resolve symlinks in the root path", func() {
		root := GinkgoT().TempDir()
		relativeRoot := filepath.Join(root, "testroot")
		path := "some/path/to/follow"
		Expect(os.MkdirAll(filepath.Join(relativeRoot, path, "test3", "test4"), os.ModePerm)).To(Succeed())
		Expect(os.Symlink("test3", filepath.Join(relativeRoot, path, "test2"))).To(Succeed())
		Expect(os.Symlink("test2", filepath.Join(relativeRoot, path, "test1"))).To(Succeed())
		// include the symlink in the root path
		pp, err := JoinAndResolveWithRelativeRoot(filepath.Join(relativeRoot, path, "test1"), "test4")
		Expect(err).ToNot(HaveOccurred())
		// don't use join to avoid any clean operations
		Expect(unsafepath.UnsafeAbsolute(pp.Raw())).To(Equal(relativeRoot + "/some/path/to/follow/test1/test4"))
	})

	It("should create a socket repeatedly the safe way", func() {
		root, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		l, err := ListenUnixNoFollow(root, "my.sock")
		Expect(err).ToNot(HaveOccurred())
		l.Close()
		l, err = ListenUnixNoFollow(root, "my.sock")
		Expect(err).ToNot(HaveOccurred())
		l.Close()
	})

	It("should open a safepath and provide its filedescriptor path with execute", func() {
		root, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		Expect(os.MkdirAll(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "test"), os.ModePerm)).To(Succeed())

		Expect(root.ExecuteNoFollow(func(safePath string) error {
			Expect(safePath).To(ContainSubstring("/proc/self/fd/"))
			_, err := os.Stat(filepath.Join(safePath, "test"))
			Expect(err).ToNot(HaveOccurred())
			return nil
		})).To(Succeed())
	})

	It("should create a child directory", func() {
		root, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		Expect(MkdirAtNoFollow(root, "test", os.ModePerm)).To(Succeed())
		_, err = os.Stat(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "test"))
		Expect(err).ToNot(HaveOccurred())
	})

	It("should set owner and file permissions", func() {
		root, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		u, err := user.Current()
		Expect(err).ToNot(HaveOccurred())
		uid, err := strconv.Atoi(u.Uid)
		Expect(err).ToNot(HaveOccurred())
		gid, err := strconv.Atoi(u.Gid)
		Expect(err).ToNot(HaveOccurred())
		Expect(ChpermAtNoFollow(root, uid, gid, 0777)).To(Succeed())
		stat, err := StatAtNoFollow(root)
		Expect(err).ToNot(HaveOccurred())
		Expect(stat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(gid)))
		Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(uid)))
		Expect(stat.Mode() & 0777).To(Equal(fs.FileMode(0777)))
		Expect(ChpermAtNoFollow(root, uid, gid, 0770)).To(Succeed())
		stat, err = StatAtNoFollow(root)
		Expect(err).ToNot(HaveOccurred())
		Expect(stat.Mode() & 0777).To(Equal(fs.FileMode(0770)))
		Expect(stat.Sys().(*syscall.Stat_t).Gid).To(Equal(uint32(gid)))
		Expect(stat.Sys().(*syscall.Stat_t).Uid).To(Equal(uint32(uid)))
	})

	It("should unlink files and directories", func() {
		root, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		Expect(TouchAtNoFollow(root, "test", os.ModePerm)).To(Succeed())
		Expect(MkdirAtNoFollow(root, "testdir", os.ModePerm)).To(Succeed())
		_, err = os.Stat(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "test"))
		Expect(err).ToNot(HaveOccurred())
		_, err = os.Stat(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "testdir"))
		Expect(err).ToNot(HaveOccurred())
		p, err := JoinNoFollow(root, "test")
		Expect(err).ToNot(HaveOccurred())
		dir, err := JoinNoFollow(root, "testdir")
		Expect(err).ToNot(HaveOccurred())
		Expect(UnlinkAtNoFollow(p)).To(Succeed())
		Expect(UnlinkAtNoFollow(dir)).To(Succeed())
		_, err = os.Stat(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "test"))
		Expect(err).To(HaveOccurred())
		_, err = os.Stat(filepath.Join(unsafepath.UnsafeAbsolute(root.Raw()), "testdir"))
		Expect(err).To(HaveOccurred())
	})

	It("should return base and relative paths correctly", func() {
		baseDir := GinkgoT().TempDir()
		root, err := JoinAndResolveWithRelativeRoot(baseDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(MkdirAtNoFollow(root, "test", os.ModePerm)).To(Succeed())
		child, err := JoinNoFollow(root, "test")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRoot(child.Raw())).To(Equal(baseDir))
		Expect(unsafepath.UnsafeRelative(child.Raw())).To(Equal("/test"))
	})

	It("should append new relative root components to the relative path", func() {
		baseDir := GinkgoT().TempDir()
		root, err := JoinAndResolveWithRelativeRoot(baseDir)
		Expect(err).ToNot(HaveOccurred())
		Expect(MkdirAtNoFollow(root, "test", os.ModePerm)).To(Succeed())
		child, err := root.AppendAndResolveWithRelativeRoot("test")
		Expect(err).ToNot(HaveOccurred())
		Expect(unsafepath.UnsafeRoot(child.Raw())).To(Equal(baseDir))
		Expect(unsafepath.UnsafeRelative(child.Raw())).To(Equal("/test"))
	})

	It("should detect absolute root", func() {
		p, err := NewPathNoFollow("/")
		Expect(err).ToNot(HaveOccurred())
		Expect(p.IsRoot()).To(BeTrue())
		tmpDir, err := JoinAndResolveWithRelativeRoot("/", GinkgoT().TempDir())
		Expect(err).ToNot(HaveOccurred())
		Expect(tmpDir.IsRoot()).To(BeFalse())
	})

	It("should be possible to use os.ReadDir on a safepath", func() {
		baseDir := GinkgoT().TempDir()
		root, err := JoinAndResolveWithRelativeRoot(baseDir)
		Expect(err).ToNot(HaveOccurred())

		Expect(os.MkdirAll(filepath.Join(baseDir, "test"), os.ModePerm)).To(Succeed())

		var files []os.DirEntry
		Expect(root.ExecuteNoFollow(func(safePath string) (err error) {
			files, err = os.ReadDir(safePath)
			return err
		})).To(Succeed())
		Expect(files).To(HaveLen(1))
	})
})
