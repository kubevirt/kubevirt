package disk

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"kubevirt.io/kubevirt/pkg/util"
)

func CreateIsoImage(iso string, volID string, files []string) error {
	if volID == "" {
		volID = "cfgdata"
	}

	isoStaging := fmt.Sprintf("%s.staging", iso)

	var args []string
	args = append(args, "-output")
	args = append(args, isoStaging)
	args = append(args, "-follow-links")
	args = append(args, "-volid")
	args = append(args, volID)
	args = append(args, "-joliet")
	args = append(args, "-rock")
	args = append(args, "-graft-points")
	args = append(args, "-partition_cyl_align")
	args = append(args, "on")
	args = append(args, files...)

	isoBinary := "xorrisofs"

	// #nosec No risk for attacket injection. Parameters are predefined strings
	cmd := exec.Command(isoBinary, args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	err = os.Rename(isoStaging, iso)

	return err
}

func CreateEmptyIsoImage(iso string, size int64) error {
	isoStaging := fmt.Sprintf("%s.staging", iso)

	f, err := os.Create(isoStaging)
	if err != nil {
		return fmt.Errorf("failed to create empty iso: '%s'", isoStaging)
	}
	err = util.WriteBytes(f, 0, size)
	if err != nil {
		return err
	}
	util.CloseIOAndCheckErr(f, &err)
	if err != nil {
		return err
	}
	err = os.Rename(isoStaging, iso)

	return err
}

func GetFilesLayoutForISO(dirPath string) ([]string, error) {
	var filesPath []string
	files, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		fileName := file.Name()
		filesPath = append(filesPath, fileName+"="+filepath.Join(dirPath, fileName))
	}
	return filesPath, nil
}
