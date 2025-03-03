#!/bin/sh -l
# Copyright 2023 The Authors (see AUTHORS file)
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

if [ -z "${GITHUB_OUTPUT+x}" ]; then
  echo "required environment variable \$GITHUB_OUTPUT is not set"
  exit 126
fi

cd /scancode-toolkit || exit 127

./scancode "/github/workspace/${1}" \
  --json "/github/workspace/scancode.json" \
  --csv "/github/workspace/scancode.csv" \
  --license \
  --package \
  --copyright \
  --license-score "70"

echo "json=scancode.json" >> "${GITHUB_OUTPUT}"
echo "csv=scancode.csv" >> "${GITHUB_OUTPUT}"
