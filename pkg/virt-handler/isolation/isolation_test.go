package isolation

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	gomock "github.com/golang/mock/gomock"
	mount "github.com/moby/sys/mountinfo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/kubevirt/pkg/util"
)

var _ = Describe("IsolationResult", func() {

	Context("Node IsolationResult", func() {

		isolationResult := NodeIsolationResult()

		It("Should have mounts", func() {
			mounts, err := isolationResult.Mounts(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounts).ToNot(BeNil())
			Expect(len(mounts)).ToNot(BeZero())
		})

		It("Should have root mounted", func() {
			mounted, err := isolationResult.IsMounted("/")
			Expect(err).ToNot(HaveOccurred())
			Expect(mounted).To(BeTrue())
		})

		It("Should resolve absolute paths with relative navigation", func() {
			mounted, err := isolationResult.IsMounted("/var/..")
			Expect(err).ToNot(HaveOccurred())
			Expect(mounted).To(BeTrue())
		})

		It("Should resolve relative paths", func() {
			_, err := isolationResult.IsMounted(".")
			Expect(err).ToNot(HaveOccurred())
		})

		It("Should resolve symlinks", func() {
			tmpDir, err := ioutil.TempDir("", "kubevirt")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tmpDir)

			symlinkPath := filepath.Join(tmpDir, "mysymlink")
			err = os.Symlink("/", symlinkPath)
			Expect(err).ToNot(HaveOccurred())

			mounted, err := isolationResult.IsMounted(symlinkPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounted).To(BeTrue())
		})

		It("Should regard a non-existent path as not mounted, not as an error", func() {
			mounted, err := isolationResult.IsMounted("/aasdfjhk")
			Expect(err).ToNot(HaveOccurred())
			Expect(mounted).To(BeFalse())
		})
	})

	Context("Container IsolationResult", func() {

		isolationResult := NewIsolationResult(os.Getpid(), os.Getppid(), "", nil)

		It("Should have mounts", func() {
			mounts, err := isolationResult.Mounts(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(mounts).ToNot(BeNil())
			Expect(len(mounts)).ToNot(BeZero())
		})
	})

	Context("Mountpoint handling", func() {

		var ctrl *gomock.Controller
		var mockIsolationResultNode *MockIsolationResult
		var mockIsolationResultContainer *MockIsolationResult

		BeforeEach(func() {
			ctrl = gomock.NewController(GinkgoT())

			mockIsolationResultNode = NewMockIsolationResult(ctrl)
			mockIsolationResultNode.EXPECT().
				Pid().
				Return(1).
				AnyTimes()
			mockIsolationResultNode.EXPECT().
				MountRoot().
				Return("/proc/1/root").
				AnyTimes()

			mockIsolationResultContainer = NewMockIsolationResult(ctrl)
			mockIsolationResultContainer.EXPECT().
				Pid().
				Return(2).
				AnyTimes()
		})

		AfterEach(func() {
			ctrl.Finish()
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
					expectedPathToRootOnNode: "/proc/1/root/var/lib/docker/overlay2/f15d9ce07df72e80d809aa99ab4a171f2f3636f65f0653e75db8ca0befd8ae02/merged",
				}, {
					mountDriver:              "devicemapper",
					hostMountInfoFile:        "devicemapper_host",
					launcherMountInfoFile:    "devicemapper_launcher",
					expectedPathToRootOnNode: "/proc/1/root/var/lib/docker/devicemapper/mnt/d0990551ba8254871a449b2ff0d9063061ae96a2c195d7a850b62f030eae1710/rootfs",
				}, {
					mountDriver:              "btrfs",
					hostMountInfoFile:        "btrfs_host",
					launcherMountInfoFile:    "btrfs_launcher",
					expectedPathToRootOnNode: "/proc/1/root/var/lib/containers/storage/btrfs/subvolumes/e9a94e2cde75c54834378d4835d4eda6bebb56b02068b9254780de6f9344ad0e",
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
						Expect(path).To(Equal(dataset.expectedPathToRootOnNode))
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

			It("Should match the correct device", func() {
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
				Expect(path).To(Equal("/proc/1/root/11"))
			})

			It("Should construct a valid path when the node mountpoint does not match the filesystem path", func() {
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
				Expect(path).To(Equal("/proc/1/root/other/location"))
			})

			It("Should find the longest match for a filesystem path", func() {
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
				Expect(path).To(Equal("/proc/1/root/long/filesystem"))
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
				Expect(path).To(HavePrefix("/proc/1/root/mymounts/"))
			})
		})
	})
})
