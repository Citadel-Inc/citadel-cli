#!/usr/bin/env bash
# Enforce a minimum statement coverage per Go package (default 75%).
# Writes a Markdown table to $GITHUB_STEP_SUMMARY when running in Actions.
# Usage: ./scripts/check-package-coverage.sh
# Env: COVERAGE_MIN (default 75), COVERAGE_SKIP_REGEX (optional egrep -v pattern for packages)

set -euo pipefail

MIN="${COVERAGE_MIN:-75}"
SKIP="${COVERAGE_SKIP_REGEX:-}"
MODULE="github.com/Rethunk-Tech/citadel-cli"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT

go test ./... -cover -covermode=atomic -count=1 >"$tmp" 2>&1 || {
	cat "$tmp"
	exit 1
}

fail=0
summary_rows=()

while IFS= read -r line; do
	[[ "$line" =~ coverage:[[:space:]]+[0-9.]+% ]] || continue
	pkg="$(awk '{print $2}' <<<"$line")"
	if [[ -n "$SKIP" ]] && grep -Eq "$SKIP" <<<"$pkg"; then
		continue
	fi
	pct="$(sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' <<<"$line")"
	if awk -v p="$pct" -v m="$MIN" 'BEGIN { exit !(p + 0 >= m + 0) }'; then
		gate="✅"
	else
		echo "coverage gate: ${pkg} ${pct}% < ${MIN}%" >&2
		fail=1
		gate="❌"
	fi
	short=".${pkg#"${MODULE}"}"
	summary_rows+=("| \`${short}\` | ${pct}% | ${gate} |")
done <"$tmp"

if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
	{
		echo "## Coverage (min ${MIN}%)"
		echo ""
		echo "| Package | Coverage | Gate |"
		echo "|---------|----------|------|"
		for row in "${summary_rows[@]}"; do
			echo "$row"
		done
	} >>"$GITHUB_STEP_SUMMARY"
fi

if [[ "$fail" -ne 0 ]]; then
	echo "check-package-coverage: FAILED (minimum ${MIN}% per package)" >&2
	exit 1
fi

echo "check-package-coverage: OK (all packages >= ${MIN}%)"
