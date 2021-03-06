#!/usr/bin/env bash

# Copyright The Helm Authors.
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
set -euo pipefail
IFS=$'\n\t'

find_files() {
  find . -not \( \
    \( \
      -wholename './vendor' \
      -o -wholename './pkg/proto' \
      -o -wholename '*testdata*' \
      -o -wholename '*third_party*' \
    \) -prune \
  \) \
  \( -name '*.go' -o -name '*.sh' -o -name 'Dockerfile' \)
}

failed_license_header=($(grep -L 'Licensed under the Apache License, Version 2.0 (the "License");' $(find_files))) || :
if (( ${#failed_license_header[@]} > 0 )); then
  echo "Some source files are missing license headers."
  for f in "${failed_license_header[@]}"; do
    echo "  $f"
  done
  exit 1
fi

failed_copyright_header=($(grep -L 'Copyright The Helm Authors.' $(find_files))) || :
if (( ${#failed_copyright_header[@]} > 0 )); then
  echo "Some source files are missing the copyright header."
  for f in "${failed_copyright_header[@]}"; do
    echo "  $f"
  done
  exit 1
fi
