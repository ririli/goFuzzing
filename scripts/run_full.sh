#!/usr/bin/env bash
# Recursively run GoPie full fuzzing for Linux test binaries.

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  bash scripts/run_full.sh --bin-dir DIR --out-dir DIR [options]

Options:
  --bin-dir DIR             Directory containing Linux test binaries
  --out-dir DIR             Directory for logs and aggregated reports
  --granularity MODE        goroutine (default) or function
  --timeout SECONDS         Per-execution timeout (default: 60)
  --fuzz-time SECONDS       Per-test fuzzing session limit (default: 0, unlimited)
  --recover-timeout SECONDS Recovery timeout (default: 200)
  --resume                  Skip binaries that already have a completion marker in <out-dir>/.done
  -h, --help                Show this help
EOF
}

bin_dir="testbins/nonblocking"
out_dir=""
granularity="goroutine"
timeout_seconds=60
fuzz_time_seconds=0
recover_timeout_seconds=200
resume=false

while (($# > 0)); do
    case "$1" in
        --bin-dir) bin_dir=${2:?missing value for --bin-dir}; shift 2 ;;
        --out-dir) out_dir=${2:?missing value for --out-dir}; shift 2 ;;
        --granularity) granularity=${2:?missing value for --granularity}; shift 2 ;;
        --timeout) timeout_seconds=${2:?missing value for --timeout}; shift 2 ;;
        --fuzz-time) fuzz_time_seconds=${2:?missing value for --fuzz-time}; shift 2 ;;
        --recover-timeout) recover_timeout_seconds=${2:?missing value for --recover-timeout}; shift 2 ;;
        --resume) resume=true; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$out_dir" ]]; then
    echo "ERROR: --out-dir is required" >&2
    usage >&2
    exit 2
fi
if [[ "$granularity" != "goroutine" && "$granularity" != "function" ]]; then
    echo "ERROR: --granularity must be goroutine or function" >&2
    exit 2
fi

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
root_dir=$(cd -- "$script_dir/.." && pwd)
fuzz_bin="$root_dir/bin/fuzz"
for value in "$timeout_seconds" "$recover_timeout_seconds"; do
    [[ "$value" =~ ^[1-9][0-9]*$ ]] || { echo "ERROR: timeouts must be positive integers" >&2; exit 2; }
done
[[ "$fuzz_time_seconds" =~ ^(0|[1-9][0-9]*)$ ]] || { echo "ERROR: --fuzz-time must be a nonnegative integer" >&2; exit 2; }

if [[ ! -d "$bin_dir" ]]; then
    echo "ERROR: binary directory not found: $bin_dir" >&2
    exit 1
fi
if [[ ! -x "$fuzz_bin" ]]; then
    echo "ERROR: Linux fuzz executable not found: $fuzz_bin" >&2
    echo "Build it first: go build -o ./bin ./cmd/..." >&2
    exit 1
fi

mkdir -p -- "$out_dir"
bin_dir=$(realpath -- "$bin_dir")
out_dir=$(realpath -- "$out_dir")
done_dir="$out_dir/.done"

mapfile -d '' bins < <(find "$bin_dir" -type f -perm /111 -print0 | sort -z)
if ((${#bins[@]} == 0)); then
    echo "No executable files found in $bin_dir"
    exit 1
fi

if $resume && [[ -f "$out_dir/PROJECT_COMPLETE" ]]; then
    echo "Already complete: $out_dir/PROJECT_COMPLETE"
    echo "All ${#bins[@]} binaries were processed in a previous run; drop PROJECT_COMPLETE to force a re-run."
    exit 0
fi

if $resume; then
    # keep aggregated reports from previous (interrupted) passes
    mkdir -p -- "$done_dir"
else
    rm -rf -- "$done_dir"
    mkdir -p -- "$done_dir"
    rm -f -- "$out_dir/allpanic.txt" "$out_dir/alldatarace.txt" "$out_dir/PROJECT_COMPLETE"
fi

echo "BinDir: $bin_dir"
echo "OutDir: $out_dir"
echo "Granularity: $granularity, Timeout: ${timeout_seconds}s, FuzzTime: ${fuzz_time_seconds}s, RecoverTimeout: ${recover_timeout_seconds}s"
echo "Resume: $resume"
echo "Found ${#bins[@]} binaries"
echo

project_start=$(date +%s)
ok=0
failed=0
skipped=0

for bin in "${bins[@]}"; do
    relative=${bin#"$bin_dir"/}
    safe_name=${relative//\//__}
    out_file="$out_dir/${safe_name}.txt"

    if $resume && [[ -f "$done_dir/$safe_name" ]]; then
        skipped=$((skipped + 1))
        echo "[$((ok + failed + skipped))/${#bins[@]}] SKIP (done) $relative"
        continue
    fi

    index=$((ok + failed + skipped + 1))
    echo "[$index/${#bins[@]}] $relative -> $out_file"

    rc=0
    "$fuzz_bin" \
        --task full \
        --path "$bin" \
        --granularity "$granularity" \
        --timeout "$timeout_seconds" \
        --fuzztime "$fuzz_time_seconds" \
        --recovertimeout "$recover_timeout_seconds" \
        --outdir "$out_dir" >"$out_file" 2>&1 || rc=$?
    if ((rc == 0)); then
        ok=$((ok + 1))
    else
        failed=$((failed + 1))
        echo "  WARNING: exit code $rc"
    fi
    date '+%Y-%m-%d %H:%M:%S rc='"$rc" >"$done_dir/$safe_name"
done

elapsed=$(( $(date +%s) - project_start ))
elapsed_text=$(printf '%d.%02d:%02d:%02d' "$((elapsed / 86400))" "$((elapsed / 3600 % 24))" "$((elapsed / 60 % 60))" "$((elapsed % 60))")
race_file="$out_dir/alldatarace.txt"
printf 'completed: %s\nbin_dir: %s\nout_dir: %s\ngranularity: %s\nbinaries: %s\nok: %s\nfailed: %s\nresumed_skipped: %s\nelapsed: %s\n' \
    "$(date '+%Y-%m-%d %H:%M:%S')" "$bin_dir" "$out_dir" "$granularity" "${#bins[@]}" "$ok" "$failed" "$skipped" "$elapsed_text" >"$out_dir/PROJECT_COMPLETE"

echo
echo "Done: $ok ok, $failed failed, $skipped resumed, ${#bins[@]} total"
echo "Aggregated reports: $out_dir/allpanic.txt / $race_file"
echo "Completion marker: $out_dir/PROJECT_COMPLETE"
echo "Project total elapsed: $elapsed_text"

((failed == 0))
