#!/usr/bin/env bash
# 对 Go 项目的所有测试包运行 go test -race，并汇总 race 与 panic。

set -uo pipefail

usage() {
    cat <<'EOF'
Usage:
  bash scripts/run_project_allRaceTest.sh --project-path DIR [options]

Options:
  --project-path DIR       被测 Go 项目根目录（必填）
  --output-dir DIR         结果目录（默认: ./race_results/<项目>_<时间戳>）
  --count N                每个测试函数执行次数（默认: 1）
  --timeout-minutes N      每个包的超时分钟数（默认: 5）
  -h, --help               显示帮助
EOF
}

project_path=""
output_dir=""
count=1
timeout_minutes=5

while (($# > 0)); do
    case "$1" in
        --project-path) project_path=${2:?missing value for --project-path}; shift 2 ;;
        --output-dir) output_dir=${2:?missing value for --output-dir}; shift 2 ;;
        --count) count=${2:?missing value for --count}; shift 2 ;;
        --timeout-minutes) timeout_minutes=${2:?missing value for --timeout-minutes}; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
    esac
done

if [[ -z "$project_path" ]]; then
    echo "ERROR: --project-path is required" >&2
    usage >&2
    exit 2
fi
if [[ ! "$count" =~ ^[1-9][0-9]*$ ]]; then
    echo "ERROR: --count must be a positive integer" >&2
    exit 2
fi
if [[ ! "$timeout_minutes" =~ ^[1-9][0-9]*$ ]]; then
    echo "ERROR: --timeout-minutes must be a positive integer" >&2
    exit 2
fi
if ! project_path=$(realpath -- "$project_path") || [[ ! -d "$project_path" ]]; then
    echo "ERROR: project path not found: $project_path" >&2
    exit 1
fi
if ! command -v go >/dev/null 2>&1; then
    echo "ERROR: go was not found in PATH" >&2
    exit 1
fi

project_name=$(basename -- "$project_path")
if [[ -z "$output_dir" ]]; then
    output_dir="$PWD/race_results/${project_name}_$(date '+%Y%m%d_%H%M%S')"
fi
mkdir -p -- "$output_dir/package_logs"
output_dir=$(realpath -- "$output_dir")

echo "============================================"
echo "  Go Race Detector Batch Tester"
echo "============================================"
echo "  Project  : $project_path"
echo "  Output   : $output_dir"
echo "  Count    : $count"
echo "  Timeout  : $timeout_minutes min/package"
echo "============================================"
echo

global_start=$(date +%s)
echo "[1/4] Discovering modules and test packages..."

declare -a module_dirs=()
while IFS= read -r -d '' mod_file; do
    module_dirs+=("$(dirname -- "$mod_file")")
done < <(find "$project_path" \
    -type d \( -name vendor -o -name testdata \) -prune -o \
    -type f -name go.mod -print0 | sort -z)
if ((${#module_dirs[@]} == 0)); then
    module_dirs+=("$project_path")
fi
echo "  Found ${#module_dirs[@]} Go module(s)."

declare -a package_modules=()
declare -a packages=()
for module_dir in "${module_dirs[@]}"; do
    list_file=$(mktemp)
    if (cd -- "$module_dir" && go list -e -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' ./...) >"$list_file" 2>&1; then
        while IFS= read -r pkg; do
            [[ -z "$pkg" || "$pkg" == go:\ * || "$pkg" =~ [[:space:]] || "$pkg" == */vendor/* ]] && continue
            package_modules+=("$module_dir")
            packages+=("$pkg")
        done <"$list_file"
    else
        echo "  [WARN] go list failed in module '$module_dir'; skipping it." >&2
        sed 's/^/    /' "$list_file" >&2
    fi
    rm -f -- "$list_file"
done

total_packages=${#packages[@]}
if ((total_packages == 0)); then
    echo "[WARN] No test packages found in $project_path" >&2
    exit 0
fi
echo "  Found $total_packages test package(s)."
echo

echo "[2/4] Running go test -race on each package..."
results_file=$(mktemp)
trap 'rm -f -- "$results_file"' EXIT
race_file="$output_dir/all_race_warnings.txt"
panic_file="$output_dir/all_panics.txt"
: >"$race_file"
: >"$panic_file"

race_package_count=0
panic_package_count=0
total_races=0
total_panics=0
fail_count=0
pass_count=0
test_time_seconds=0

for ((i = 0; i < total_packages; i++)); do
    module_dir=${package_modules[$i]}
    pkg=${packages[$i]}
    relative_module=${module_dir#"$project_path"}
    relative_module=${relative_module#/}
    [[ -z "$relative_module" ]] && relative_module="."
    safe_name=$(printf '%s__%s' "$relative_module" "$pkg" | tr '/:' '__' | tr -cd '[:alnum:]_.-')
    package_log="$output_dir/package_logs/${safe_name}.json"

    printf '  [%d/%d] %s [%s] ' "$((i + 1))" "$total_packages" "$pkg" "$relative_module"
    package_start=$(date +%s)
    (cd -- "$module_dir" && go test -race -json -count "$count" -timeout "${timeout_minutes}m" "$pkg") >"$package_log" 2>&1
    rc=$?
    elapsed=$(( $(date +%s) - package_start ))
    test_time_seconds=$((test_time_seconds + elapsed))

    race_count=$(grep -c 'WARNING: DATA RACE' "$package_log" || true)
    panic_count=$(grep -Ec '(^|"Output":"[[:space:]]*)panic:' "$package_log" || true)
    result="PASS"
    if ((rc != 0)); then
        result="FAIL"
        fail_count=$((fail_count + 1))
    elif ((race_count == 0 && panic_count == 0)); then
        pass_count=$((pass_count + 1))
    fi

    if ((race_count > 0)); then
        race_package_count=$((race_package_count + 1))
        total_races=$((total_races + race_count))
        {
            printf '\n----------------------------------------\n  Package: %s\n  Full log: %s\n----------------------------------------\n' "$pkg" "$package_log"
            cat -- "$package_log"
        } >>"$race_file"
    fi
    if ((panic_count > 0)); then
        panic_package_count=$((panic_package_count + 1))
        total_panics=$((total_panics + panic_count))
        {
            printf '\n----------------------------------------\n  Package: %s\n  Full log: %s\n----------------------------------------\n' "$pkg" "$package_log"
            cat -- "$package_log"
        } >>"$panic_file"
    fi

    status="OK"
    ((rc != 0)) && status="[FAIL]"
    ((panic_count > 0)) && status="[PANIC]"
    ((race_count > 0)) && status="[RACE]"
    printf '%02d:%02d  %s (races=%d, panics=%d)\n' "$((elapsed / 60))" "$((elapsed % 60))" "$status" "$race_count" "$panic_count"
    printf '%s\t%s\t%d\t%d\t%d\n' "$pkg" "$result" "$race_count" "$panic_count" "$elapsed" >>"$results_file"
done

echo
echo "[3/4] Saving race and panic warnings..."
if ((total_races == 0)); then
    echo "(No race warnings detected)" >"$race_file"
fi
if ((total_panics == 0)); then
    echo "(No panics detected)" >"$panic_file"
fi

wall_time_seconds=$(( $(date +%s) - global_start ))
summary_file="$output_dir/summary.txt"
{
    echo "============================================"
    echo "  Go Race Detector Batch Test - Summary"
    echo "============================================"
    echo "  Project          : $project_path"
    echo "  Date             : $(date '+%Y-%m-%d %H:%M:%S')"
    echo "  Count per test   : $count"
    echo "  Timeout/pkg      : $timeout_minutes min"
    echo "  Total packages   : $total_packages"
    echo "  Pass (clean)     : $pass_count"
    echo "  Failed           : $fail_count"
    echo "  Race detected    : $race_package_count"
    echo "  Panic detected   : $panic_package_count"
    echo "  Total race blocks: $total_races"
    echo "  Total panic lines: $total_panics"
    echo "  Wall time         : $wall_time_seconds s"
    echo "  go test (sum)     : $test_time_seconds s"
    echo
    printf '%-64s %-8s %-8s %-8s %-8s\n' Package Result Races Panics Seconds
    while IFS=$'\t' read -r pkg result races panics elapsed; do
        printf '%-64s %-8s %-8s %-8s %-8s\n' "$pkg" "$result" "$races" "$panics" "$elapsed"
    done <"$results_file"
} >"$summary_file"

echo "[4/4] Summary written to $summary_file"
echo "  Race warnings: $race_file"
echo "  Panic warnings: $panic_file"
echo "  Per-package logs: $output_dir/package_logs"

if ((race_package_count > 0 || panic_package_count > 0)); then
    exit 1
fi
exit 0
