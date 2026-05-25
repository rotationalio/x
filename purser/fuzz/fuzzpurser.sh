#!/usr/bin/env bash
# Run all purser/fuzz targets (or one named target). Execute from anywhere in the repo.
#
# Usage:
#   ./purser/fuzz/fuzzpurser.sh                         # 30s per target, 2m test timeout
#   ./purser/fuzz/fuzzpurser.sh 3s                      # smoke: 3s per target
#   ./purser/fuzz/fuzzpurser.sh FuzzParseKeyID          # single target
#   FUZZTIME=10s TIMEOUT=5m ./purser/fuzz/fuzzpurser.sh # override timeout
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
pkg="./purser/fuzz"

targets=(
	FuzzMeta_unmarshal
	FuzzSealed_unmarshal
	FuzzParseOpenWire
	FuzzParseKeyID_agreesWithUnmarshal
	FuzzParseKeyID
	FuzzRegistry_ParseKeyID
)

fuzztime="${FUZZTIME:-30s}"
timeout="${TIMEOUT:-2m}"

if [[ $# -ge 1 && "${1:-}" =~ ^[0-9] ]]; then
	fuzztime="$1"
	shift
fi

if [[ "$fuzztime" =~ ^[0-9]+s$ ]]; then
	sec="${fuzztime%s}"
	if [[ -z "${TIMEOUT:-}" && "$sec" -le 10 ]]; then
		timeout="$((sec * 4))s"
	fi
fi

run_target() {
	local target="$1"
	echo "Fuzzing ${target} (fuzztime=${fuzztime}, timeout=${timeout})..."
	(
		cd "$root"
		go test -timeout="$timeout" -run '^$' -fuzz="${target}\$" -fuzztime="$fuzztime" "$pkg"
	)
	echo "Completed ${target}."
}

cd "$root"

if [[ $# -ge 1 ]]; then
	run_target "$1"
	exit 0
fi

for target in "${targets[@]}"; do
	run_target "$target"
done

echo "All fuzz targets finished."
