/*
 * This file is part of the KubeVirt project
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
 * Copyright 2020 Red Hat, Inc.
 *
 */

package util

import (
	"fmt"
	"io"
	"os"

	"kubevirt.io/client-go/log"
)

// CloseIOAndCheckErr closes the file and check the returned error.
// If there was an error a log messages will be printed.
// If a valid address (not nil) is passed in  err the function will also update the error
// Note: to update the error the calling funtion need to use named returned variable (If called as defer function)
func CloseIOAndCheckErr(c io.Closer, err *error) {
	if ferr := c.Close(); ferr != nil {
		log.DefaultLogger().Reason(ferr).Error("Error when closing file")
		// Update the calling error only in case there wasn't a different error already
		if err != nil && *err == nil {
			*err = ferr
		}
	}
}

func WriteBytes(f *os.File, c byte, n int64) error {
	var err error
	var i, total int64
	buf := make([]byte, 1<<12)

	for i = 0; i < 1<<12; i++ {
		buf[i] = c
	}

	for i = 0; i < n>>12; i++ {
		x, err := f.Write(buf)
		total += int64(x)
		if err != nil {
			return err
		}
	}

	x, err := f.Write(buf[:n&(1<<12-1)])
	total += int64(x)
	if err != nil {
		return err
	}
	if total != n {
		return fmt.Errorf("wrote %d bytes instead of %d", total, n)
	}

	return nil
}
