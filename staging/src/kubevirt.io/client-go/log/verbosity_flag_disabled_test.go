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

//go:build nokubevirtlogflag

package log

import (
	goflag "flag"
	"testing"
)

func TestVerbosityFlagNotRegisteredWithBuildTag(t *testing.T) {
	if goflag.CommandLine.Lookup("v") != nil {
		t.Fatal("\"v\" flag must not be registered on flag.CommandLine when nokubevirtlogflag tag is set")
	}
}

func TestDefaultVerbosityCorrectWhenFlagRegistrationDisabled(t *testing.T) {
	log := MakeLogger(MockLogger{})
	if log.verbosityLevel != 2 {
		t.Fatalf("Default verbosity should be 2 when flag registration is disabled via nokubevirtlogflag, got %d", log.verbosityLevel)
	}
}
