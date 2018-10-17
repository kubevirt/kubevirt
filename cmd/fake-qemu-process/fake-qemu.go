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
 * Copyright 2017 Red Hat, Inc.
 *
 */

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"
)

func main() {
	uuid := pflag.String("uuid", "", "some fake arg")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt,
		syscall.SIGTERM,
	)

	pflag.Parse()
	fmt.Printf("Started fake qemu process with uuid %s\n", *uuid)

	timeout := time.After(60 * time.Second)
	select {
	case <-timeout:
	case <-c:
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("Exit fake qemu process\n")
}
