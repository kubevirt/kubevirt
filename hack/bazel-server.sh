# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright The KubeVirt Authors.
#

# Execute unit tests without relying on Bazel.
if [ "$KUBEVIRT_NO_BAZEL" = true ]; then
    sleep infinity
else
    BAZEL_PID=$(bazel info | grep server_pid | cut -d " " -f 2)
    while kill -0 $BAZEL_PID 2>/dev/null; do sleep 1; done
    # Might not be necessary, just to be sure that exec shutdowns always succeed
    # and are not killed by docker.
    sleep 1
fi
