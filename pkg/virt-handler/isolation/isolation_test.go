package isolation

import (
	"fmt"
	"os"
	"path/filepath"

	gomock "github.com/golang/mock/gomock"
	mount "github.com/moby/sys/mountinfo"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/safepath"
	"kubevirt.io/kubevirt/pkg/unsafepath"

	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("IsolationResult", func() {
	Context("Node IsolationResult", func() {
		isolationResult := NodeIsolationResult()

		It("Should have mounts", func() {
			mounts, err := isolationResult.Mounts(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounts).ToNot(BeEmpty())
		})

		It("Should have root mounted", func() {
			root, err := safepath.NewPathNoFollow("/")
			Expect(err).ToNot(HaveOccurred())
			mounted, err := IsMounted(root)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounted).To(BeTrue())
		})
	})

	Context("Container IsolationResult", func() {
		isolationResult := NewIsolationResult(os.Getpid(), os.Getppid())

		It("Should have mounts", func() {
			mounts, err := isolationResult.Mounts(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounts).ToNot(BeEmpty())
		})
	})

	Context("Mountpoint handling", func() {
		var ctrl *gomock.Controller
		var mockIsolationResultNode *MockIsolationResult
		var mockIsolationResultContainer *MockIsolationResult
		var tmpDir string

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())
			tmpDir = GinkgoT().TempDir()
			root, err := safepath.JoinAndResolveWithRelativeRoot(filepath.Join("/proc/self/root", tmpDir))
			Expect(err).ToNot(HaveOccurred())

			mockIsolationResultNode = NewMockIsolationResult(ctrl)
			mockIsolationResultNode.EXPECT().
				Pid().
				Return(1).
				AnyTimes()
			mockIsolationResultNode.EXPECT().
				MountRoot().
				Return(root, nil).
				AnyTimes()

			mockIsolationResultContainer = NewMockIsolationResult(ctrl)
			mockIsolationResultContainer.EXPECT().
				Pid().
				Return(2).
				AnyTimes()
		})

		Context("Using real world sampled host and launcher mountinfo data", func() {
			type mountInfoData struct {
				mountDriver              string
				hostMountInfoFile        string
				launcherMountInfoFile    string
				expectedPathToRootOnNode string
			}

			mountInfoDataList := []mountInfoData{
				{
					mountDriver:              "overlay",
					hostMountInfoFile:        "overlay_host",
					launcherMountInfoFile:    "overlay_launcher",
					expectedPathToRootOnNode: "/var/lib/docker/overlay2/f15d9ce07df72e80d809aa99ab4a171f2f3636f65f0653e75db8ca0befd8ae02/merged",
				}, {
					mountDriver:              "devicemapper",
					hostMountInfoFile:        "devicemapper_host",
					launcherMountInfoFile:    "devicemapper_launcher",
					expectedPathToRootOnNode: "/var/lib/docker/devicemapper/mnt/d0990551ba8254871a449b2ff0d9063061ae96a2c195d7a850b62f030eae1710/rootfs",
				}, {
					mountDriver:              "btrfs",
					hostMountInfoFile:        "btrfs_host",
					launcherMountInfoFile:    "btrfs_launcher",
					expectedPathToRootOnNode: "/var/lib/containers/storage/btrfs/subvolumes/e9a94e2cde75c54834378d4835d4eda6bebb56b02068b9254780de6f9344ad0e",
				},
			}

			getMountsFrom := func(file string, f mount.FilterFunc) []*mount.Info {
				in, err := os.Open(filepath.Join("testdata", "mountinfo", file))
				Expect(err).NotTo(HaveOccurred())
				defer util.CloseIOAndCheckErr(in, nil)

				mounts, err := mount.GetMountsFromReader(in, f)
				Expect(err).ToNot(HaveOccurred())
				return mounts
			}

			for _, dataset := range mountInfoDataList {
				Context(fmt.Sprintf("Using storage driver %v", dataset.mountDriver), func() {
					BeforeEach(func() {
						Expect(os.MkdirAll(filepath.Join(tmpDir, dataset.expectedPathToRootOnNode), os.ModePerm)).To(Succeed())
						mockIsolationResultNode.EXPECT().
							Mounts(gomock.Any()).
							DoAndReturn(func(f mount.FilterFunc) ([]*mount.Info, error) {
								return getMountsFrom(dataset.hostMountInfoFile, f), nil
							}).
							AnyTimes()
						mockIsolationResultContainer.EXPECT().
							Mounts(gomock.Any()).
							DoAndReturn(func(f mount.FilterFunc) ([]*mount.Info, error) {
								return getMountsFrom(dataset.launcherMountInfoFile, f), nil
							}).
							AnyTimes()
					})

					It("Should detect the root mount point for a container", func() {
						rootMount, err := MountInfoRoot(mockIsolationResultContainer)
						Expect(err).ToNot(HaveOccurred())
						Expect(rootMount).ToNot(BeNil())
						Expect(rootMount.Mountpoint).To(Equal("/"))
					})

					It("Should detect the full path to the root mount point of a container on the node", func() {
						path, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
						Expect(err).ToNot(HaveOccurred())
						Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root", tmpDir, dataset.expectedPathToRootOnNode)))
					})
				})
			}
		})

		Context("Using simulated mount data", func() {
			rootMountPoint := &mount.Info{
				Major:      1,
				Minor:      1,
				Mountpoint: "/",
				Root:       "/",
			}

			initMountsMock := func(m *MockIsolationResult, mounts []*mount.Info) {
				m.EXPECT().
					Mounts(gomock.Any()).
					DoAndReturn(func(f mount.FilterFunc) ([]*mount.Info, error) {
						result := []*mount.Info{}
						for _, mi := range mounts {
							skip, stop := f(mi)
							if !skip {
								result = append(result, mi)
							}
							if stop {
								break
							}
						}
						return result, nil
					}).
					AnyTimes()
			}

			It("Should detect the root mountpoint", func() {
				initMountsMock(mockIsolationResultContainer, []*mount.Info{
					{
						Major:      2,
						Minor:      1,
						Mountpoint: "/notroot",
						Root:       "/",
					},
					rootMountPoint,
				})

				rootMount, err := MountInfoRoot(mockIsolationResultContainer)
				Expect(err).ToNot(HaveOccurred())
				Expect(rootMount).To(Equal(rootMountPoint))
			})

			It("Should find the correct nfs mount, based on source", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/match"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/target",
						Root:       "/",
						Source:     "somehost:/somepath",
						FSType:     "nfs4",
					},
					rootMountPoint,
				})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/nomatch1",
						Root:       "/",
						Source:     "somehost:/someotherpath",
						FSType:     "nfs4",
					},
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/nomatch2",
						Root:       "/",
						Source:     "somehost:/somestrangepath",
						FSType:     "nfs4",
					},
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/match",
						Root:       "/",
						Source:     "somehost:/somepath",
						FSType:     "nfs4",
					},
					rootMountPoint,
				})
				path, err := ParentPathForMount(mockIsolationResultNode, mockIsolationResultContainer, "somehost:/somepath", "/target")
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root", tmpDir, "/match")))
			})

			It("Should find the longest root, if major and minor match", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/match/something"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/target",
						Root:       "/root/something",
					},
					rootMountPoint,
				})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/nomatch1",
						Root:       "/",
					},
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/match",
						Root:       "/root",
					},
					{
						Major:      200,
						Minor:      123,
						Mountpoint: "/nomatch2",
						Root:       "/",
					},
					rootMountPoint,
				})
				path, err := ParentPathForMount(mockIsolationResultNode, mockIsolationResultContainer, "", "/target")
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root", tmpDir, "/match/something")))
			})

			It("Should match the correct device", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/12"), os.ModePerm)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/21"), os.ModePerm)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/11"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{rootMountPoint})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      1,
						Minor:      2,
						Mountpoint: "/12",
						Root:       "/",
					}, {
						Major:      2,
						Minor:      1,
						Mountpoint: "/21",
						Root:       "/",
					}, {
						Major:      1,
						Minor:      1,
						Mountpoint: "/11",
						Root:       "/",
					},
				})

				path, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root", tmpDir, "/11")))
			})

			It("Should construct a valid path when the node mountpoint does not match the filesystem path", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/some/path"), os.ModePerm)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/other/location"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/",
						Root:       "/some/path",
					},
				})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/other/location",
						Root:       "/some/path",
					},
				})

				path, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root", tmpDir, "/other/location")))
			})

			It("Should find the longest match for a filesystem path", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/some/path/quite/deeply/located/on/the/filesystem"), os.ModePerm)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/long/filesystem"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/",
						Root:       "/some/path/quite/deeply/located/on/the/filesystem",
					},
				})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/short",
						Root:       "/some",
					}, {
						Major:      1,
						Minor:      1,
						Mountpoint: "/long",
						Root:       "/some/path/quite/deeply/located/on/the",
					}, {
						Major:      1,
						Minor:      1,
						Mountpoint: "/medium",
						Root:       "/some/path/quite",
					},
				})

				path, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(Equal(filepath.Join("/proc/self/root/", tmpDir, "long/filesystem")))
			})

			It("Should fail when the device does not exist", func() {
				initMountsMock(mockIsolationResultContainer, []*mount.Info{rootMountPoint})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      2,
						Minor:      1,
						Mountpoint: "/",
						Root:       "/",
					},
				})

				_, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).To(HaveOccurred())
			})

			It("Should fail if the target filesystem path is not mounted", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/other/path"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{rootMountPoint})
				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/",
						Root:       "/other/path",
					},
				})

				_, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).To(HaveOccurred())
			})

			It("Should not fail for duplicate mountpoints", func() {
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/mymounts/first"), os.ModePerm)).To(Succeed())
				Expect(os.MkdirAll(filepath.Join(tmpDir, "/mymounts/second"), os.ModePerm)).To(Succeed())
				initMountsMock(mockIsolationResultContainer, []*mount.Info{rootMountPoint})

				initMountsMock(mockIsolationResultNode, []*mount.Info{
					{
						Major:      1,
						Minor:      1,
						Mountpoint: "/mymounts/first",
						Root:       "/",
					}, {
						Major:      1,
						Minor:      1,
						Mountpoint: "/mymounts/second",
						Root:       "/",
					},
				})

				path, err := ParentPathForRootMount(mockIsolationResultNode, mockIsolationResultContainer)
				Expect(err).ToNot(HaveOccurred())
				Expect(unsafepath.UnsafeAbsolute(path.Raw())).To(HavePrefix(filepath.Join("/proc/self/root", tmpDir, "/mymounts/")))
			})
		})
	})
})
