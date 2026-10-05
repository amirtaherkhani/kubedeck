#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CAPTURE_FILE="$(mktemp)"
OUTPUT_FILE="$(mktemp)"

cleanup() {
  rm -f "${CAPTURE_FILE}" "${OUTPUT_FILE}"
}

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

assert_has_arg() {
  local actual="$1"
  local expected="$2"

  [[ " ${actual} " == *" ${expected} "* ]] || fail \
    "expected Helm arguments to contain ${expected}; got: ${actual}"
}

assert_lacks_arg() {
  local actual="$1"
  local unexpected="$2"

  [[ " ${actual} " != *" ${unexpected} "* ]] || fail \
    "expected Helm arguments to omit ${unexpected}; got: ${actual}"
}

assert_contains() {
  local actual="$1"
  local expected="$2"

  [[ "${actual}" == *"${expected}"* ]] || fail \
    "expected output to contain ${expected}; got: ${actual}"
}

helm() {
  if [[ "${1:-}" == "upgrade" && "${2:-}" == "--help" ]]; then
    if [[ "${HELM_HELP_SUPPORTS_ROLLBACK:-true}" == "true" ]]; then
      printf '%s\n' '      --rollback-on-failure   roll back on failure'
    fi
    return 0
  fi

  printf '%s\n' "$*" > "${CAPTURE_FILE}"
}

kubectl() {
  if [[ "${1:-}" == "config" && "${2:-}" == "current-context" ]]; then
    printf '%s\n' 'rancher-desktop'
    return 0
  fi

  fail "unexpected kubectl invocation: $*"
}

export CAPTURE_FILE
export -f fail helm kubectl
trap cleanup EXIT INT TERM

cd "${REPO_ROOT}"

test_custom_image_apply_omits_automatic_rollback() {
  HELM_AUTO_ROLLBACK=false make --no-print-directory \
    helm-apply RELEASE=kubedeck > "${OUTPUT_FILE}"
  local safe_args
  safe_args="$(<"${CAPTURE_FILE}")"

  assert_has_arg "${safe_args}" '--wait'
  assert_has_arg "${safe_args}" '--wait-for-jobs'
  assert_has_arg "${safe_args}" '--cleanup-on-fail'
  assert_lacks_arg "${safe_args}" '--atomic'
  assert_lacks_arg "${safe_args}" '--rollback-on-failure'
}

test_default_apply_prefers_rollback_on_failure() {
  : > "${CAPTURE_FILE}"
  (
    unset HELM_AUTO_ROLLBACK
    ./scripts/platform-helm.sh apply kubedeck > "${OUTPUT_FILE}"
  )
  assert_has_arg "$(<"${CAPTURE_FILE}")" '--rollback-on-failure'
}

test_default_apply_falls_back_to_atomic() {
  : > "${CAPTURE_FILE}"
  HELM_AUTO_ROLLBACK=true HELM_HELP_SUPPORTS_ROLLBACK=false \
    ./scripts/platform-helm.sh apply kubedeck > "${OUTPUT_FILE}"
  assert_has_arg "$(<"${CAPTURE_FILE}")" '--atomic'
}

test_invalid_auto_rollback_setting_is_rejected() {
  if HELM_AUTO_ROLLBACK=invalid ./scripts/platform-helm.sh apply kubedeck \
    > "${OUTPUT_FILE}" 2>&1; then
    fail 'invalid HELM_AUTO_ROLLBACK value was accepted'
  fi
  assert_contains "$(<"${OUTPUT_FILE}")" 'HELM_AUTO_ROLLBACK must be true or false'
}

test_custom_image_apply_omits_automatic_rollback
test_default_apply_prefers_rollback_on_failure
test_default_apply_falls_back_to_atomic
test_invalid_auto_rollback_setting_is_rejected

printf '%s\n' 'PASS: platform Helm rollback policy'
