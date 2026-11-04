#!/bin/sh
# Link the existing GBE package into a private build tree: no forked supervisor.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
stage="$root/tools/fence-corpus/.cache/adapter"
mkdir -p "$stage/tools/go-by-example/gbe" "$stage/docs/go-by-example"
# Keep the existing resolver/build algorithm; expose its authenticated SDK path.
awk '{ print; if ($0 == "GO=\"${GOROOT_PINNED}/bin/${go_name}\"") print "printf '\''%s\\n'\'' \"${GO}\" > \"${ROOT}/sdk-path\"" }' \
  "$root/tools/go-by-example/build.sh" > "$stage/tools/go-by-example/build.sh"
for src in "$root"/tools/go-by-example/gbe/*; do
  ln -sf "$src" "$stage/tools/go-by-example/gbe/$(basename "$src")"
done
ln -sf "$root/tools/fence-corpus/adapter.go" "$stage/tools/go-by-example/gbe/fence_adapter.go"
ln -sf "$root/docs/go-by-example/toolchain.tsv" "$stage/docs/go-by-example/toolchain.tsv"
bash "$stage/tools/go-by-example/build.sh"
