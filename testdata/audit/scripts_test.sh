#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
work=$(mktemp -d "$root/testdata/audit/.work.XXXXXX")
mkdir -p "$work/tool/scripts" "$work/tool/bin" "$work/project" "$work/mock" "$work/bins" "$work/empty"
cp "$root/scripts/"*.sh "$work/tool/scripts/"
printf 'module audit\n' > "$work/project/go.mod"
cat > "$work/mock/go" <<'EOF'
#!/usr/bin/env bash
if [[ "$1" == list ]]; then
    if [[ "$SCENARIO" == discovery ]]; then exit 2; fi
    echo example.org/audit
elif [[ "$1" == test ]]; then
    case "$SCENARIO" in
        clean) exit 0 ;;
        compile) echo 'build failed' >&2; exit 1 ;;
        fatal) echo '{"Output":"fatal error: concurrent map writes\n"}'; exit 1 ;;
    esac
fi
EOF
cat > "$work/tool/bin/fuzz" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$CALLS"
exit "${FUZZ_RC:-0}"
EOF
chmod +x "$work/mock/go" "$work/tool/bin/fuzz"
export PATH="$work/mock:$PATH"
export CALLS="$work/calls.txt"
expect() {
    local want=$1 rc=0
    shift
    "$@" > "$work/last.log" 2>&1 || rc=$?
    if [[ "$rc" != "$want" ]]; then
        cat "$work/last.log"
        echo "expected exit $want, got $rc: $*" >&2
        exit 1
    fi
}
for script in "$root/scripts/"*.sh; do bash -n "$script"; done
for scenario in clean compile fatal discovery; do
    export SCENARIO=$scenario
    rc=1
    [[ "$scenario" == clean ]] && rc=0
    expect "$rc" bash "$work/tool/scripts/run_project_allRaceTest.sh" \
        --project-path "$work/project" --output-dir "$work/$scenario"
done
grep -q 'fatal error:' "$work/fatal/all_panics.txt"
cd "$work"
printf '#!/bin/sh\n' > bins/one
printf '#!/bin/sh\n' > bins/two
chmod +x bins/*
expect 0 bash tool/scripts/run_full.sh --bin-dir bins --out-dir out
[[ $(wc -l < "$CALLS") == 2 ]]
grep -q -- "$work/bins/one" "$CALLS"
export FUZZ_RC=7
expect 1 bash tool/scripts/run_full.sh --bin-dir bins --out-dir out
[[ $(wc -l < "$CALLS") == 4 ]]
expect 2 bash tool/scripts/run_full.sh --bin-dir bins --out-dir out --timeout -1
echo retained > out/allpanic.txt
expect 1 bash tool/scripts/run_full.sh --bin-dir empty --out-dir out
grep -q retained out/allpanic.txt
mkdir -p projects/DEMO/demo projects/DEMO/demoF projects/DEMO/demoG
expect 1 bash tool/scripts/do_run_full.sh DEMO --gopie-root tool --projects-root projects --bin-dir-f bins --bin-dir-g bins
echo 'PASS: shell syntax, discovery/build failures, fatal reports, relative paths, continuation, validation, empty input, shared binary rejection'
