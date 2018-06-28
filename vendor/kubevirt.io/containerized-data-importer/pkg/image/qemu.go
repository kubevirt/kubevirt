package image

import (
	"bytes"
	"io"
	"os/exec"

	"github.com/pkg/errors"
)

// magic number strings, needed to detect and test qcow2 files
var QCOW2MagicStr = []byte{'Q', 'F', 'I', 0xfb}
var QCOW2MagicStrSize = len(QCOW2MagicStr)

// Return the magic number which is contained in the 1st `QCOW2MagicStrSize` bytes of
// the passed in file. If the file is too small then an empty magic string is returned.
// Error is returned if a non-eof io error occurs.
func GetMagicNumber(f io.Reader) ([]byte, error) {
	buff := make([]byte, QCOW2MagicStrSize)
	cnt, err := f.Read(buff)
	if cnt < QCOW2MagicStrSize {
		return nil, nil
	}
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "unable to read byte buffer")
	}
	return buff, nil
}

func MatchQcow2MagicNum(match []byte) bool {
	return bytes.HasPrefix(match, QCOW2MagicStr)
}

func ConvertQcow2ToRaw(src, dest string) error {
	cmd := exec.Command("qemu-img", "convert", "-f", "qcow2", "-O", "raw", src, dest)
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(err, "could not convert qcow2 image to raw")
	}
	return nil
}
