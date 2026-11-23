#!/bin/bash
set -euo pipefail

SCRIPT_PATH="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/circleci/output-force-deploy"
WORK_ROOT="$(mktemp -d)"
trap 'rm -rf "${WORK_ROOT}"' EXIT

failures=0

# Runs the script in a fresh directory. $1 is the manifest body ("none" for no manifest), $2 the config.yml body.
run_script() {
    local manifest="$1" config="$2" run_dir
    run_dir="$(mktemp -d "${WORK_ROOT}/run.XXXXXX")"
    mkdir -p "${run_dir}/manifest" "${run_dir}/.circleci"
    [ "${manifest}" = "none" ] || echo "${manifest}" > "${run_dir}/manifest/deploy-manifest.env"
    echo "${config}" > "${run_dir}/.circleci/config.yml"
    (cd "${run_dir}" && DEPLOY_MANIFEST_DIR="${run_dir}/manifest" CIRCLE_PROJECT_REPONAME=example CIRCLE_PROJECT_USERNAME=Clever bash "${SCRIPT_PATH}")
}

assert_contains() {
    if grep -qF -- "$2" <<<"$3"; then
        echo "PASS: $1"
    else
        echo "FAIL: $1"
        echo "  expected to contain: $2"
        failures=$((failures + 1))
    fi
}

assert_not_contains() {
    if grep -qF -- "$2" <<<"$3"; then
        echo "FAIL: $1"
        echo "  expected not to contain: $2"
        failures=$((failures + 1))
    else
        echo "PASS: $1"
    fi
}

output="$(run_script 'MANIFEST_SHA=abc
MANIFEST_APPS="app-a app-b"' "")"
assert_contains "manifest apps prefill force_deploy_apps" '"force_deploy_apps": "app-a app-b"' "${output}"

output="$(run_script 'MANIFEST_SHA=abc
MANIFEST_APPS=""' "")"
assert_contains "empty manifest app list says nothing to deploy" "nothing to force deploy" "${output}"
assert_not_contains "empty manifest app list prints no curl" "curl" "${output}"

output="$(run_script 'MANIFEST_SHA=abc' "  force_deploy_apps: { type: string }")"
assert_contains "manifest without app list falls back to config" '"force_deploy_apps": ""' "${output}"

output="$(run_script none "  force_deploy_apps: { type: string }")"
assert_contains "config declaring force_deploy_apps prints an empty list" '"force_deploy_apps": ""' "${output}"

output="$(run_script none "")"
assert_contains "legacy config prints force_deploy only" '"force_deploy": true' "${output}"
assert_not_contains "legacy config omits force_deploy_apps" "force_deploy_apps" "${output}"

if [ "${failures}" -ne 0 ]; then
    echo "${failures} assertion(s) failed"
    exit 1
fi
echo "all assertions passed"
