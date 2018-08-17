package image

import (
	"os/exec"

	"github.com/pkg/errors"
)

func ConvertQcow2ToRaw(src, dest string) error {
	cmd := exec.Command("qemu-img", "convert", "-f", "qcow2", "-O", "raw", src, dest)
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "could not convert qcow2 image to raw")
	}
	return nil
}
