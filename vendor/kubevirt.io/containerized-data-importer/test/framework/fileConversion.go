package framework

import (
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
	"kubevirt.io/containerized-data-importer/pkg/image"
)

var formatTable = map[string]func(string) (string, error){
	image.ExtGz:    transformGz,
	image.ExtXz:    transformXz,
	image.ExtTar:   transformTar,
	image.ExtQcow2: transformQcow2,
	"":             transformNoop,
}

// create file based on targetFormat extensions and return created file's name.
// note: intermediate files are removed.
func FormatTestData(srcFile string, targetFormats ...string) (string, error) {
	outFile := srcFile
	var err error
	var prevFile string

	for _, tf := range targetFormats {
		f, ok := formatTable[tf]
		if !ok {
			return "", errors.Errorf("format extension %q not recognized", tf)
		}
		if len(tf) == 0 {
			continue
		}
		// invoke conversion func
		outFile, err = f(outFile)
		if prevFile != srcFile {
			os.Remove(prevFile)
		}
		if err != nil {
			return "", errors.Wrap(err, "could not format test data")
		}
		prevFile = outFile
	}
	return outFile, nil
}

func transformFile(srcFile, outfileName, osCmd string, osArgs ...string) (string, error) {
	cmd := exec.Command(osCmd, osArgs...)
	cout, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "OS command %s %v errored with output: %v", osCmd, strings.Join(osArgs, " "), string(cout))
	}
	finfo, err := os.Stat(outfileName)
	if err != nil {
		return "", errors.Wrapf(err, "error stat-ing file")
	}
	return finfo.Name(), nil
}

func transformTar(srcFile string) (string, error) {
	args := []string{"-cf", srcFile + image.ExtTar, srcFile}
	return transformFile(srcFile, srcFile+image.ExtTar, "tar", args...)
}

func transformGz(srcFile string) (string, error) {
	return transformFile(srcFile, srcFile+image.ExtGz, "gzip", "-k", srcFile)
}

func transformXz(srcFile string) (string, error) {
	return transformFile(srcFile, srcFile+image.ExtXz, "xz", "-k", srcFile)
}

func transformQcow2(srcfile string) (string, error) {
	outFile := strings.Replace(srcfile, ".iso", image.ExtQcow2, 1)
	args := []string{"convert", "-f", "raw", "-O", "qcow2", srcfile, outFile}
	return transformFile(srcfile, outFile, "qemu-img", args...)
}

func transformNoop(srcFile string) (string, error) {
	return srcFile, nil
}
