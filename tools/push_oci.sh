#!/bin/bash

set -e
set -u
set -o pipefail

label=$1

for target in $(bazel query 'kind("^oci_push\b", //...)' 2> /dev/null); do
  bazel run --config=release --embed_label="${label}" "${target}"
done
