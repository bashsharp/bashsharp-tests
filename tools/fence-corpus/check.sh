#!/bin/sh
set -eu
here=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
bin=$(sh "$here/build.sh")
FENCE_CORPUS=1 GBE_ROOT=$(CDPATH= cd -- "$here/../.." && pwd) exec "$bin" fence-check "$@"
