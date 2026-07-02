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

// This file registers the "-v" verbosity flag on flag.CommandLine.
// Build with -tags nokubevirtlogflag to disable this registration,
// e.g. when another package in the binary already registers "-v".

//go:build !nokubevirtlogflag

package log

import goflag "flag"

func init() {
	// "the practical default level is V(2)"
	// see https://github.com/kubernetes/community/blob/master/contributors/devel/logging.md
	goflag.IntVar(&defaultVerbosity, "v", defaultVerbosityLevel, "log level for V logs")
}
