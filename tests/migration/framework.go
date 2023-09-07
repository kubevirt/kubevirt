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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package migration

import (
	. "github.com/onsi/ginkgo/v2"

	"kubevirt.io/kubevirt/tests/decorators"
)

const describeName = "[rfe_id:393][crit:high][vendor:cnv-qe@redhat.com][level:system][sig-compute] "

func SIGMigrationDescribe(text string, args ...interface{}) bool {
	return Describe(describeName+text, decorators.SigComputeMigrations, decorators.SigCompute, args)
}

func FSIGMigrationDescribe(text string, args ...interface{}) bool {
	return FDescribe(describeName+text, decorators.SigComputeMigrations, decorators.SigCompute, args)
}

func PSIGMigrationDescribe(text string, args ...interface{}) bool {
	return PDescribe(describeName+text, decorators.SigComputeMigrations, decorators.SigCompute, args)
}
