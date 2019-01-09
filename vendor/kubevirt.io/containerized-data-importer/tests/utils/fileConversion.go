package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/ulikunitz/xz"

	"kubevirt.io/containerized-data-importer/pkg/image"
)

var formatTable = map[string]func(string, string) (string, error){
	image.ExtGz:    toGz,
	image.ExtXz:    toXz,
	image.ExtTar:   toTar,
	image.ExtQcow2: toQcow2,
	"":             toNoop,
}

// FormatTestData accepts the path of a single file (srcFile) and attempts to generate an output
// file in the format defined by targetFormats (e.g. ".tar", ".gz" will produce a .tar.gz formatted file).  The output file is written to the directory in `tgtDir`.
// returns:
//		(string) Path of output file
//		(error)  Errors that occur during formatting
func FormatTestData(srcFile, tgtDir string, targetFormats ...string) (string, error) {
	var err error
	for _, tf := range targetFormats {
		f, ok := formatTable[tf]
		if !ok {
			return "", errors.Errorf("format extension %q not recognized", tf)
		}
		// invoke conversion func
		srcFile, err = f(srcFile, tgtDir)
		if err != nil {
			return "", errors.Wrap(err, "could not format test data")
		}
	}
	return srcFile, nil
}

func toTar(src, tgtDir string) (string, error) {
	return ArchiveFiles(src, tgtDir, src)
}

// ArchiveFiles creates a tar file that archives the given source files.
func ArchiveFiles(targetFile, tgtDir string, sourceFilesNames ...string) (string, error) {
	tgtFile, tgtPath, _ := createTargetFile(targetFile, tgtDir, image.ExtTar)
	defer tgtFile.Close()

	w := tar.NewWriter(tgtFile)
	defer w.Close()

	for _, src := range sourceFilesNames {
		srcFile, err := os.Open(src)
		if err != nil {
			return "", errors.Wrapf(err, "Error opening file %s", src)
		}
		defer srcFile.Close()

		srcFileInfo, err := srcFile.Stat()
		if err != nil {
			return "", errors.Wrapf(err, "Error stating file %s", src)
		}

		hdr, err := tar.FileInfoHeader(srcFileInfo, "")
		if err != nil {
			return "", errors.Wrapf(err, "Error generating tar file header for %s", src)
		}

		err = w.WriteHeader(hdr)
		if err != nil {
			return "", errors.Wrapf(err, "Error writing tar header to %s", tgtPath)
		}

		_, err = io.Copy(w, srcFile)
		if err != nil {
			return "", errors.Wrapf(err, "Error writing to file %s", tgtPath)
		}
	}

	return tgtPath, nil
}

func toGz(src, tgtDir string) (string, error) {
	tgtFile, tgtPath, _ := createTargetFile(src, tgtDir, image.ExtGz)
	defer tgtFile.Close()

	w := gzip.NewWriter(tgtFile)
	defer w.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return "", errors.Wrapf(err, "Error opening file %s", src)
	}
	defer srcFile.Close()

	_, err = io.Copy(w, srcFile)
	if err != nil {
		return "", errors.Wrapf(err, "Error writing to file %s", tgtPath)
	}
	return tgtPath, nil
}

func toXz(src, tgtDir string) (string, error) {
	tgtFile, tgtPath, _ := createTargetFile(src, tgtDir, image.ExtXz)
	defer tgtFile.Close()

	w, err := xz.NewWriter(tgtFile)
	if err != nil {
		return "", errors.Wrapf(err, "Error getting xz writer for file %s", tgtPath)
	}
	defer w.Close()

	srcFile, err := os.Open(src)
	if err != nil {
		return "", errors.Wrapf(err, "Error opening file %s", src)
	}
	defer srcFile.Close()

	_, err = io.Copy(w, srcFile)
	if err != nil {
		return "", errors.Wrapf(err, "Error writing to file %s", tgtPath)
	}
	return tgtPath, nil
}

func toQcow2(srcfile, tgtDir string) (string, error) {
	base := strings.TrimSuffix(filepath.Base(srcfile), ".iso")
	tgt := filepath.Join(tgtDir, base+image.ExtQcow2)
	args := []string{"convert", "-f", "raw", "-O", "qcow2", srcfile, tgt}

	if err := doCmdAndVerifyFile(tgt, "qemu-img", args...); err != nil {
		return "", err
	}
	return tgt, nil
}

func toNoop(src, tgtDir string) (string, error) {
	return copyIfNotPresent(src, tgtDir)
}

func doCmdAndVerifyFile(tgt, cmd string, args ...string) error {
	if err := doCmd(cmd, args...); err != nil {
		return err
	}
	if _, err := os.Stat(tgt); err != nil {
		return errors.Wrapf(err, "Failed to stat file %q", tgt)
	}
	return nil
}

func doCmd(osCmd string, osArgs ...string) error {
	cmd := exec.Command(osCmd, osArgs...)
	cout, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrapf(err, "OS command `%s %v` errored: %v\nStdout/Stderr: %s", osCmd, strings.Join(osArgs, " "), err, string(cout))
	}
	return nil
}

// copyIfNotPresent checks for the src file in the tgtDir.  If it is not there, it attempts to copy it from src to tgtdir.
// If a copy is performed, the path to the copy is returned.
// If the file already exists, the original src string is returned.
func copyIfNotPresent(src, tgtDir string) (string, error) {
	ret := filepath.Join(tgtDir, filepath.Base(src))
	_, err := os.Stat(ret)
	if err != nil && !os.IsNotExist(err) {
		return "", errors.Wrap(err, "Unexpected error stating file")
	}
	if os.IsNotExist(err) {
		if err = doCmd("cp", src, ret); err != nil {
			return "", err
		}
	}
	return ret, nil
}

// createTargetFile is a simple helper to create a file with the provided extension in the target directory.
// returns a pointer to the new file, path to the new file, or an error. It is the responsibility of the caller to
// close the file.
func createTargetFile(src, tgtDir, ext string) (*os.File, string, error) {
	tgt := filepath.Join(tgtDir, filepath.Base(src)+ext)
	tgtFile, err := os.Create(tgt)
	if err != nil {
		return nil, "", errors.Wrap(err, "Error creating file")
	}
	return tgtFile, tgt, nil
}
