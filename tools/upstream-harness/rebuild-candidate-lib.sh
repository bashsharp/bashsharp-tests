#!/bin/bash
# Shared helpers for rebuild-candidate.sh and its shell tests.

# Extract the bashy go build command emitted by `make -n build-bashy` and
# redirect the Makefile's temporary output path to the candidate's stable binary.
derive_bashy_real_build_command() {
	local line cmd
	line=$1
	cmd=$(printf '%s\n' "$line" | grep -Eo 'go build -trimpath -ldflags "[^"]*" -o ("?\$\$tmp"?|"?\$tmp"?|"?\$out"?|[^ ]+) ./cmd/bashy' | head -1 || true)
	test -n "$cmd" || return 1
	cmd=${cmd//'"$$tmp"'/bin/bashy.real}
	cmd=${cmd//'$$tmp'/bin/bashy.real}
	cmd=${cmd//'"$tmp"'/bin/bashy.real}
	cmd=${cmd//'$tmp'/bin/bashy.real}
	cmd=${cmd//'"$out"'/bin/bashy.real}
	cmd=${cmd//'$out'/bin/bashy.real}
	printf '%s\n' "$cmd"
}
