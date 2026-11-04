#!/bin/sh
# Link the existing GBE package into a private build tree: no forked supervisor.
set -eu
root=$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)
stage="$root/tools/fence-corpus/.cache/adapter"
mkdir -p "$stage/tools/go-by-example/gbe" "$stage/docs/go-by-example"
# Keep the resolver and digest check. Its bashy fallback assumes go.exe;
# correct only that generated platform spelling for a PATH without host Go.
# Also expose the authenticated SDK path to the adapter.
awk '
  $0 == "  bootstrap_go=\"${bootstrap_root}/bin/go.exe\"" {
    print "  bootstrap_go=\"${bootstrap_root}/bin/go\""
    print "  [ \"${os}\" = windows ] && bootstrap_go=\"${bootstrap_go}.exe\""
    next
  }
  { print }
  $0 == "GO=\"${GOROOT_PINNED}/bin/${go_name}\"" {
    print "printf '\''%s\\n'\'' \"${GO}\" > \"${ROOT}/sdk-path\""
  }
' "$root/tools/go-by-example/build.sh" > "$stage/tools/go-by-example/build.sh"
for src in "$root"/tools/go-by-example/gbe/*; do
  ln -sf "$src" "$stage/tools/go-by-example/gbe/$(basename "$src")"
done
for src in "$root"/tools/fence-corpus/*.go; do
  ln -sf "$src" "$stage/tools/go-by-example/gbe/fence_$(basename "$src")"
done
ln -sf "$root/docs/go-by-example/toolchain.tsv" "$stage/docs/go-by-example/toolchain.tsv"
bash "$stage/tools/go-by-example/build.sh"
