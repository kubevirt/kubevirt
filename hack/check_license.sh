#!/bin/bash
#
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
# Copyright 2023 Red Hat, Inc.
#

git remote add upstream https://github.com/kubevirt/kubevirt.git > /dev/null 2>&1
git fetch upstream > /dev/null

diff_output=$(git diff upstream/main --name-only --diff-filter=A)
missing_license=false

while IFS= read -r file; do
  if [ -f "$file" ] && ! grep -q "http://www.apache.org/licenses/LICENSE-2.0" "$file"; then
    echo "Missing license header: $file"
    missing_license=true
  fi
done <<< "$diff_output"

if [ "$missing_license" = true ]; then
  echo "License headers missing. Please add the license header to the listed files."
  exit 1
elif [ -z "$diff_output" ]; then
  echo "No new files added."
  exit 0
else
  echo "All added files have license headers."
  exit 0
fi

