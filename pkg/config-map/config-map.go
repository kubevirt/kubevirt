package configmap

import (
	"os"
)

func SetLocalDirectory(path string) error {
	// create main directory
	err := os.Mkdir(path, 0755)
	if err != nil {
		return err
	}
	return nil
}

/*
1. create configmap
2. add new volume type
3. add config map to template ! <first step>
4. create img based on path (with -graft-points)
5. add img disk to disks
6? am i able to mount it somewhere? /mnt/config-map
7? am i able to create symbolic links from /mnt/config-map/etc/* to /etc
*/
