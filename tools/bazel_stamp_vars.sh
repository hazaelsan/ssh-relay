#!/bin/bash

set -e
set -u
set -o pipefail

BUILD_TIMESTAMP="${BUILD_TIMESTAMP:-$(date +%s)}"
BUILD_SCM_HAS_LOCAL_CHANGES=$(
  git status --untracked-files=no --porcelain > /dev/null 2>&1 &&
    echo yes || echo no
)

cat << EOF
BUILD_SCM_BRANCH $(git symbolic-ref --short HEAD)
BUILD_SCM_COMMIT_SHA $(git rev-parse HEAD)
BUILD_SCM_HAS_LOCAL_CHANGES ${BUILD_SCM_HAS_LOCAL_CHANGES}
BUILD_TIMESTAMP_ISO8601 $(date --utc --date "@${BUILD_TIMESTAMP}" +'%Y-%m-%dT%H:%M:%SZ')
EOF
