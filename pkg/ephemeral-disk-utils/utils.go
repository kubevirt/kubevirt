/*
 * This file is part of the kubevirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2017 Red Hat, Inc.
 *
 */

package ephemeraldiskutils

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"

	"kubevirt.io/kubevirt/pkg/logging"
)

func RemoveFile(path string) error {
	err := os.Remove(path)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("failed to remove cloud-init temporary data file %s", path))
		return err
	}
	return nil
}
func FileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	exists := false

	if err == nil {
		exists = true
	} else if os.IsNotExist(err) {
		err = nil
	}
	return exists, err
}
func Md5CheckSum(filePath string) ([]byte, error) {
	var result []byte

	file, err := os.Open(filePath)
	if err != nil {
		return result, err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)

	if err != nil {
		return result, err
	}

	result = hash.Sum(result)
	return result, nil
}

func SetFileOwnership(username string, file string) error {
	usrObj, err := user.Lookup(username)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to look up username %s", username))
		return err
	}

	uid, err := strconv.Atoi(usrObj.Uid)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to find uid for username %s", username))
		return err
	}

	gid, err := strconv.Atoi(usrObj.Gid)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unable to find gid for username %s", username))
		return err
	}

	return os.Chown(file, uid, gid)
}

func FilesAreEqual(path1 string, path2 string) (bool, error) {
	exists, err := FileExists(path1)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unexpected error encountered while attempting to determine if %s exists", path1))
		return false, err
	} else if exists == false {
		return false, nil
	}

	exists, err = FileExists(path2)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("unexpected error encountered while attempting to determine if %s exists", path2))
		return false, err
	} else if exists == false {
		return false, nil
	}

	sum1, err := Md5CheckSum(path1)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("calculating md5 checksum failed for %s", path1))
		return false, err
	}
	sum2, err := Md5CheckSum(path2)
	if err != nil {
		logging.DefaultLogger().V(2).Error().Reason(err).Msg(fmt.Sprintf("calculating md5 checksum failed for %s", path2))
		return false, err
	}

	return bytes.Equal(sum1, sum2), nil
}
