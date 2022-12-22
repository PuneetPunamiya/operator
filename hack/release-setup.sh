#!/usr/bin/env bash

set -u

SCRIPT_DIR=$(dirname $0)

source ${SCRIPT_DIR}/release.sh

declare -r TEKTONCD_OPERATOR_RELEASE_VERSION=${1}; shift
declare -r git_remote_name=${1:-origin}

checkout_release_branch ${git_remote_name} ${TEKTONCD_OPERATOR_RELEASE_VERSION}
set_version_label ${TEKTONCD_OPERATOR_RELEASE_VERSION}
commit_changes ${TEKTONCD_OPERATOR_RELEASE_VERSION}
push_release_branch ${git_remote_name} ${TEKTONCD_OPERATOR_RELEASE_VERSION}