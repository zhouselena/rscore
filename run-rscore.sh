#!/usr/bin/env bash
set -eux pipefail

# Build the rscore binary.
bash build.sh

if [[ ! -x "./bin/rscore" ]]; then
  echo "Error: ./bin/rscore not found or not executable. Please run 'bash build.sh' first." >&2
  exit 1
else
  echo "Found ./bin/rscore"
fi

./bin/rscore
