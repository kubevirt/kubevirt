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

//go:build !coverage_e2e

package coveragehttp

// Start is a no-op in production (non-coverage) builds. Daemons call it
// unconditionally at startup; the coverage HTTP server only exists in binaries
// built with the "coverage_e2e" tag (hack/bazel-build-images.sh --build-cover).
func Start(component string) {}
