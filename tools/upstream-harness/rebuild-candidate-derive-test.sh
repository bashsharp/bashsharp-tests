#!/usr/bin/env bash
set -euo pipefail
script_dir=$(cd -- "$(dirname "$0")" && pwd)
# shellcheck source=tools/upstream-harness/rebuild-candidate-lib.sh
. "$script_dir/rebuild-candidate-lib.sh"

check_line() {
	local input got
	input=$1
	got=$(derive_bashy_real_build_command "$input")
	case $got in
	*' -o bin/bashy.real ./cmd/bashy') ;;
	*) printf 'derived command has wrong output path: %s\n' "$got" >&2; exit 1 ;;
	esac
	case $got in
	*'$'*) printf 'derived command still contains a dollar sign: %s\n' "$got" >&2; exit 1 ;;
	esac
}

check_line 'go build -trimpath -ldflags "-w" -o "$tmp" ./cmd/bashy'
check_line 'go build -trimpath -ldflags "-w" -o $tmp ./cmd/bashy'
check_line 'go build -trimpath -ldflags "-w" -o $out ./cmd/bashy'

echo 'rebuild candidate command derivation passed'
