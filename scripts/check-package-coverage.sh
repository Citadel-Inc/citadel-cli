#!/usr/bin/env bash
# Enforce a minimum statement coverage per Go package (default 75%).
# Usage: ./scripts/check-package-coverage.sh
# Env: COVERAGE_MIN (default 75), COVERAGE_SKIP_REGEX (optional egrep -v pattern for packages)

set -euo pipefail

MIN="${COVERAGE_MIN:-75}"
SKIP="${COVERAGE_SKIP_REGEX:-}"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

go test ./... -cover -covermode=atomic -count=1 >"$tmp" 2>&1 || {
	cat "$tmp"
	exit 1
}

fail=0
while IFS= read -r line; do
	[[ "$line" =~ coverage:[[:space:]]+[0-9.]+% ]] || continue
	pkg="$(awk '{print $2}' <<<"$line")"
	if [[ -n "$SKIP" ]] && grep -Eq "$SKIP" <<<"$pkg"; then
		continue
	fi
	pct="$(sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' <<<"$line")"
	if awk -v p="$pct" -v m="$MIN" 'BEGIN { exit !(p + 0 >= m + 0) }'; then
		continue
	fi
	echo "coverage gate: ${pkg} ${pct}% < ${MIN}%" >&2
	fail=1
done <"$tmp"

if [[ "$fail" -ne 0 ]]; then
	echo "check-package-coverage: FAILED (minimum ${MIN}% per package)" >&2
	exit 1
fi

echo "check-package-coverage: OK (all packages >= ${MIN}%)"
