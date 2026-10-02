#!/usr/bin/env bash
# 重建单个 G/F 副本并接线、插桩、编译测试二进制。
#
#   bash scripts/prepare_copy.sh --project-dir GORILLA --copy-name websocket --mode G
#
# 动作顺序（与 memory/AGENTS.md 的接线坑一致）：
#   留档旧副本 -> cp -a 原始副本 -> 移走嵌套 module -> 移走 go.work -> 每个 go.mod 加 replace
#   -> bin/fuzz --task inst -> 每个 go.mod 追加 require toolkit -> go mod tidy
#   -> bin/fuzz --task bins -> 落 <step>.prepared 标记
#
# 所有「删除」都是 mv 到 $RETIRED_ROOT，脚本里不出现 rm -rf。
# 已有 .prepared 标记时默认跳过；--force 重新做一遍。

set -uo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=/dev/null
source "$script_dir/serial_queue.sh"
require_roots || exit 1

project_dir=""
copy_name=""
mode=""
force=false
dry=false

while (($# > 0)); do
    case "$1" in
        --project-dir) project_dir=${2:?missing value for --project-dir}; shift 2 ;;
        --copy-name) copy_name=${2:?missing value for --copy-name}; shift 2 ;;
        --mode) mode=${2:?missing value for --mode}; shift 2 ;;
        --force) force=true; shift ;;
        --dry-run) dry=true; shift ;;
        *) echo "ERROR: unknown argument: $1" >&2; exit 2 ;;
    esac
done

[[ -n "$project_dir" && -n "$copy_name" && -n "$mode" ]] || {
    echo "Usage: prepare_copy.sh --project-dir NAME --copy-name NAME --mode G|F [--force] [--dry-run]" >&2
    exit 2
}
resolve_paths "$project_dir" "$copy_name" "$mode" || exit 1

step=$(step_id "$copy_name" "$mode")
gran=$(granularity_of "$mode") || exit 1
orig=$RESOLVED_ORIG
copy=$RESOLVED_COPY

mkdir -p "$EXP_ROOT/state" "$EXP_ROOT/logs" "$RETIRED_ROOT"
prep_log="$EXP_ROOT/logs/prep_${step}.log"
marker="$EXP_ROOT/state/${step}.prepared"

run() {
    if $dry; then
        printf 'DRY: %s\n' "$*"
        return 0
    fi
    "$@"
}

if [[ -f "$marker" ]] && ! $force; then
    echo "SKIP: $step already prepared ($marker)"
    exit 0
fi

echo "=== prepare $step ($(log_ts)) gran=$gran ===" | tee -a "$prep_log"
echo "orig=$orig copy=$copy" | tee -a "$prep_log"

# 1) 留档旧副本
if [[ -d "$copy" ]]; then
    stamp=$(date '+%Y%m%dT%H%M%S')
    dest="$RETIRED_ROOT/${step}-${stamp}"
    echo "retire: $copy -> $dest" | tee -a "$prep_log"
    if ! $dry; then
        mkdir -p -- "$RETIRED_ROOT"
        mv -- "$copy" "$dest" || { echo "RETIRE_FAIL rc=$?" | tee -a "$prep_log"; exit 1; }
    fi
fi

# 2) 从原始副本复制
echo "copy: $orig -> $copy" | tee -a "$prep_log"
if $dry; then
    echo "DRY: cp -a $orig $copy"
else
    cp -a -- "$orig" "$copy" || { echo "COPY_FAIL rc=$?" | tee -a "$prep_log"; exit 1; }
fi

# 3) 移走无测试包的嵌套 module（只移走确实含 go.mod 的目录）
prune_list=${PRUNE_DIRS[$copy_name]:-}
if [[ -n "$prune_list" ]]; then
    stamp=$(date '+%Y%m%dT%H%M%S')
    prune_dest="$RETIRED_ROOT/prune-${step}-${stamp}"
    for rel in $prune_list; do
        [[ -n "$rel" && "$rel" != "/" && "$rel" != *".."* ]] || { echo "SKIP bad prune path: $rel" | tee -a "$prep_log"; continue; }
        abs="$copy/$rel"
        if [[ -d "$abs" && -f "$abs/go.mod" ]]; then
            flat=${rel//\//_}
            echo "prune: $abs -> $prune_dest/$flat" | tee -a "$prep_log"
            if ! $dry; then
                mkdir -p -- "$prune_dest"
                mv -- "$abs" "$prune_dest/$flat" || echo "PRUNE_FAIL $rel rc=$?" | tee -a "$prep_log"
            fi
        else
            echo "prune skip (no go.mod): $abs" | tee -a "$prep_log"
        fi
    done
fi

# 3b) go.work 会让 workspace 覆盖整棵副本：剪掉 module 后 use 条目指向不存在的目录，
#     Go 连根 module 都编译不了（gorumsG 首轮 12/12 目录全挂在
#     "cannot load module examples listed in go.work file" -> 0 个测试二进制）。
#     本 harness 的接线模型是「每个 go.mod 各自 replace/require」，workspace 属多余概念，
#     所以把副本里的 go.work 一并 mv 留档，让副本退化成与其它 6 个项目一致的单 module 视图。
gw_scan_root=$copy
$dry && [[ ! -d "$copy" ]] && gw_scan_root=$orig
while IFS= read -r gw; do
    [[ -n "$gw" ]] || continue
    flat=${gw#"$gw_scan_root/"}
    flat=${flat//\//_}
    if $dry; then
        echo "DRY: retire go.work $gw -> $RETIRED_ROOT/gowork-${step}-*/$flat" | tee -a "$prep_log"
        continue
    fi
    gw_dest="$RETIRED_ROOT/gowork-${step}-$(date '+%Y%m%dT%H%M%S')"
    mkdir -p -- "$gw_dest"
    if mv -- "$gw" "$gw_dest/$flat"; then
        echo "retire go.work: $gw -> $gw_dest/$flat" | tee -a "$prep_log"
    else
        echo "GOWORK_RETIRE_FAIL $gw rc=$?" | tee -a "$prep_log"
    fi
done < <(find "$gw_scan_root" -name go.work -not -path '*/vendor/*' 2>/dev/null | sort)

# 4) 每个 module 先加 replace（插桩前）
scan_root=$copy
$dry && [[ ! -d "$copy" ]] && scan_root=$orig
mapfile -t modfiles < <(find "$scan_root" -name go.mod -not -path '*/vendor/*' 2>/dev/null | sed "s|^$scan_root|$copy|" | sort)
((${#modfiles[@]} > 0)) || { echo "NO go.mod under $scan_root" | tee -a "$prep_log"; exit 1; }
echo "modules: ${#modfiles[@]}" | tee -a "$prep_log"
for gm in "${modfiles[@]}"; do
    [[ -f "$gm" ]] || continue
    if ! grep -q '^replace toolkit =>' "$gm"; then
        printf '\nreplace toolkit => %s\n' "$GOPIE_ROOT" >>"$gm"
    fi
done

# 5) 插桩（原地覆盖，不幂等，所以只能对全新副本执行）
echo "=== instrument ($gran) ===" | tee -a "$prep_log"
(cd "$GOPIE_ROOT" && run ./bin/fuzz --task inst --path "$copy" --granularity "$gran") >>"$prep_log" 2>&1
echo "inst_rc=$?" | tee -a "$prep_log"

# 6) 插桩之后才追加 require，再 tidy
for gm in "${modfiles[@]}"; do
    dir=$(dirname "$gm")
    [[ -d "$dir" && -f "$gm" ]] || continue
    grep -q '^require toolkit v' "$gm" || printf '\nrequire toolkit v0.0.0\n' >>"$gm"
    echo "=== go mod tidy: $dir ===" | tee -a "$prep_log"
    (cd "$dir" && run go mod tidy) >>"$prep_log" 2>&1
    echo "tidy_rc=$? dir=$dir" | tee -a "$prep_log"
done

# 7) 编译测试二进制到 testbins/<step>；先移走上一轮遗留，避免陈旧二进制混进长跑
bins_dir="$GOPIE_ROOT/testbins/$step"
if [[ -d "$bins_dir" ]]; then
    stamp=$(date '+%Y%m%dT%H%M%S')
    echo "retire stale bins: $bins_dir -> $RETIRED_ROOT/bins-${step}-${stamp}" | tee -a "$prep_log"
    if ! $dry; then
        mkdir -p -- "$RETIRED_ROOT"
        mv -- "$bins_dir" "$RETIRED_ROOT/bins-${step}-${stamp}" || {
            echo "RETIRE_BINS_FAIL" | tee -a "$prep_log"
            exit 1
        }
    fi
fi
echo "=== bins -> $bins_dir ===" | tee -a "$prep_log"
if $dry; then
    echo "DRY: ./bin/fuzz --task bins --path $copy --output testbins/$step"
else
    mkdir -p -- "$bins_dir"
    (cd "$GOPIE_ROOT" && ./bin/fuzz --task bins --path "$copy" --output "testbins/$step") >>"$prep_log" 2>&1
    echo "bins_rc=$?" | tee -a "$prep_log"
    nb=$(find "$bins_dir" -type f -perm /111 2>/dev/null | wc -l)
    echo "binaries=$nb" | tee -a "$prep_log"
    ((nb > 0)) || { echo "PREPARE_FAIL: no test binaries built for $step" | tee -a "$prep_log"; exit 1; }
fi

$dry || { date '+%Y-%m-%d %H:%M:%S' >"$marker"; echo "PREPARE_OK $step" | tee -a "$prep_log"; }
