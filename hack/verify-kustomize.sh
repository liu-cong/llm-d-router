#!/usr/bin/env bash

# Copyright 2026 The llm-d Authors.
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

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
KUSTOMIZE_ROOT="${SCRIPT_ROOT}/config/kustomize"
TEMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "${TEMP_DIR}" || true
}
trap cleanup EXIT

TARGETS=(
  "${KUSTOMIZE_ROOT}/base/common"
  "${KUSTOMIZE_ROOT}/base/gateway"
  "${KUSTOMIZE_ROOT}/base/standalone"
)

for overlay in "${KUSTOMIZE_ROOT}/overlays/"*; do
  if [[ -d "${overlay}" ]]; then
    TARGETS+=("${overlay}")
  fi
done

for target in "${TARGETS[@]}"; do
  rel_path="${target#"${SCRIPT_ROOT}/"}"
  out_file="${TEMP_DIR}/$(echo "${rel_path}" | tr '/' '_').yaml"
  echo "Building ${rel_path}..."
  kubectl kustomize "${target}" > "${out_file}"
  if [[ ! -s "${out_file}" ]]; then
    echo "ERROR: rendered output for ${rel_path} is empty"
    exit 1
  fi
done

# Verify nameReference propagation in the full-featured overlay.
full_out="${TEMP_DIR}/config_kustomize_overlays_full-featured.yaml"
for expected in \
  "full-router-inference-pool" \
  "full-router-epp" \
  "full-router-inference-gateway-sa-metrics-reader-secret" \
  "--ha-enable-leader-election=true" \
  "--tracing=true"; do
  if ! grep -q -- "${expected}" "${full_out}"; then
    echo "ERROR: expected '${expected}' in rendered full-featured overlay"
    exit 1
  fi
done

echo "All Kustomize bases and overlays built and verified."
