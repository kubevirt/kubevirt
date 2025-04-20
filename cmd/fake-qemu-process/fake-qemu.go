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
 * Copyright The KubeVirt Authors.
 *
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

func main() {
	uuid := pflag.String("uuid", "", "the UUID of the fake qemu process")
	pidFile := pflag.String("pidfile", "", "the path of the PID file to create")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGTERM,
	)

	pflag.Parse()
	fmt.Printf("Started fake qemu process with uuid %s and pidfile %s\n", *uuid, *pidFile)

	if *pidFile != "" {
		pid := os.Getpid()
		err := os.WriteFile(*pidFile, []byte(strconv.Itoa(pid)), 0644)
		if err != nil {
			fmt.Printf("Could not write to PID file %s: %v\n", *pidFile, err)
			os.Exit(1)
		}
	}

	timeout := time.After(60 * time.Second)
	select {
	case <-timeout:
	case <-c:
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Exit fake qemu process\n")
}
