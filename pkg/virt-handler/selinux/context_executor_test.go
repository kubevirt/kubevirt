package selinux

import (
	"fmt"
	"os"
	"os/exec"

	"go.uber.org/mock/gomock"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SELinux context executor", func() {
	var (
		ctrl     *gomock.Controller
		executor *MockExecutor
	)

	const (
		pid = 1234
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		executor = NewMockExecutor(ctrl)

		executor.
			EXPECT().
			CloseOnExec(gomock.All(gomock.Not(0), gomock.Not(1), gomock.Not(2))).
			Times(maxFDToCloseOnExec - minFDToCloseOnExec)
		executor.
			EXPECT().
			Run(gomock.Any()).
			Return(nil)
	})

	Context("with SELinux disabled", func() {
		BeforeEach(func() {
			executor.
				EXPECT().
				NewSELinux().
				Return(&SELinuxImpl{}, false, nil).
				Times(2)
		})

		It("should successfully execute a command", func() {
			ce, err := newContextExecutor(pid, &exec.Cmd{}, executor)
			Expect(err).ToNot(HaveOccurred())
			Expect(ce.desiredLabel).To(BeEmpty())
			Expect(ce.originalLabel).To(BeEmpty())
			Expect(ce.pid).To(Equal(pid))
			err = ce.Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with SELinux enabled", func() {
		const (
			desiredLabel  = "desiredLabel"
			originalLabel = "originalLabel"
		)

		BeforeEach(func() {
			executor.
				EXPECT().
				NewSELinux().
				Return(&SELinuxImpl{}, true, nil).
				Times(2)
			executor.
				EXPECT().
				FileLabel(fmt.Sprintf("/proc/%d/attr/current", pid)).
				Return(desiredLabel, nil)
			executor.
				EXPECT().
				FileLabel(fmt.Sprintf("/proc/%d/attr/current", os.Getpid())).
				Return(originalLabel, nil)
			executor.
				EXPECT().
				LockOSThread()
			executor.
				EXPECT().
				SetExecLabel(desiredLabel).
				Return(nil)
			executor.
				EXPECT().
				SetExecLabel(originalLabel).
				Return(nil)
			executor.
				EXPECT().
				UnlockOSThread()
		})

		It("should successfully execute a command", func() {
			ce, err := newContextExecutor(pid, &exec.Cmd{}, executor)
			Expect(err).ToNot(HaveOccurred())
			Expect(ce.desiredLabel).To(Equal(desiredLabel))
			Expect(ce.originalLabel).To(Equal(originalLabel))
			Expect(ce.pid).To(Equal(pid))
			err = ce.Execute()
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
