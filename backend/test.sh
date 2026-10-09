#!/bin/sh
set -u

module=$(go list -m -f '{{.Path}}') || exit 1
packages=$(go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...) || exit 1
[ -n "$packages" ] || { echo 'No test packages found'; exit 0; }
output=$(mktemp)
trap 'rm -f "$output"' EXIT
set +e
go test -json -count=1 $packages >"$output"
status=$?
set -e
risq_prefix=$(printf '%s\n' "$packages" | sed "s#^$module/##; s#/tests/.*#/#" | head -n 1)
awk -v module="$module" -v risq="$risq_prefix" '
function field(name, line, value) { match(line, quote name quote ":" quote "[^" quote "]*"); value=substr(line, RSTART+length(name)+4, RLENGTH-length(name)-4); return value }
function number(name, line, value) { match(line, quote name quote ":[0-9.]+"); value=substr(line, RSTART+length(name)+3, RLENGTH-length(name)-3); return value }
function label(path) { sub("^" module "/?", "", path); if (risq != "") sub("^" risq, "", path); return path }
BEGIN { quote=sprintf("%c", 34); printf "%-42s %8s %10s\n", "NAME", "TESTS", "TIME" }
/"Action":"pass"/ && /"Test":"/ { package=field("Package", $0); passed[package]++ }
/("Action":"pass"|"Action":"fail")/ && /"Package":"/ && !/"Test":"/ { package=field("Package", $0); time=number("Elapsed", $0); printf "%-42s %8d %9ss\n", label(package), passed[package]+0, time+0 }
' "$output"
if [ "$status" -ne 0 ]; then
    echo
    echo 'Test failures:'
    sed -n 's/.*"Output":"\(.*\)".*/\1/p' "$output" | sed 's/\\n/\n/g; s/\\"/"/g; s/\\\\/\\/g'
    exit 1
fi
echo
echo 'All Tests Pass'
