#!/usr/bin/env bash
# 清理 fuzz 长跑在 /tmp 遗留的临时目录。
#
#   bash scripts/clean_tmp.sh                 # 只试算，不删
#   bash scripts/clean_tmp.sh --run           # 真的删
#   bash scripts/clean_tmp.sh --run --min-age-min 30
#
# 背景：fuzz 会反复 SIGKILL 被测测试进程，Go 的 TMPDIR（go-build*、Test*、
# etcd-integration*、raftexample*）来不及回收就在 /tmp 里堆积。2026-10-03 跑
# etcdG 的 tests/* 时一小时漏了 40G，/tmp 累计 47G，继续下去会撑满磁盘并杀掉长跑。
#
# 安全边界：只碰 /tmp 顶层、只碰下面列出的模式、且只碰 mtime 老于 --min-age-min 的条目
# （正在写入的临时目录 mtime 必然是新的，因此不会被删）。路径不是 /tmp 直接拒绝执行。

set -uo pipefail

tmp_root=/tmp
min_age_min=60
do_run=false

while (($# > 0)); do
    case "$1" in
        --run) do_run=true; shift ;;
        --tmp-root) tmp_root=${2:?missing value for --tmp-root}; shift 2 ;;
        --min-age-min) min_age_min=${2:?missing value for --min-age-min}; shift 2 ;;
        *) echo "ERROR: unknown argument: $1" >&2; exit 2 ;;
    esac
done

[[ "$tmp_root" == /tmp ]] || { echo "REFUSE: tmp_root must be /tmp, got $tmp_root" >&2; exit 2; }
[[ "$min_age_min" =~ ^[0-9]+$ ]] || { echo "REFUSE: --min-age-min must be a number" >&2; exit 2; }

mapfile -t victims < <(
    find "$tmp_root" -maxdepth 1 -type d \
        \( -name 'go-build*' -o -name 'Test*' -o -name 'etcd-integration*' -o -name 'raftexample*' \) \
        -mmin +"$min_age_min" 2>/dev/null | sort
)

if ((${#victims[@]} == 0)); then
    echo "CLEAN_TMP nothing to reclaim (min_age_min=$min_age_min)"
    exit 0
fi

size_lines=$(du -shc "${victims[@]}" 2>/dev/null | tail -1)
echo "CLEAN_TMP candidates=${#victims[@]} size=$size_lines min_age_min=$min_age_min"

if ! $do_run; then
    echo "CLEAN_TMP DRY (add --run to delete)"
    printf '  %s\n' "${victims[@]}" | head -20
    exit 0
fi

find "$tmp_root" -maxdepth 1 -type d \
    \( -name 'go-build*' -o -name 'Test*' -o -name 'etcd-integration*' -o -name 'raftexample*' \) \
    -mmin +"$min_age_min" -exec rm -rf {} + 2>/dev/null
echo "CLEAN_TMP DONE rc=$?"
echo "CLEAN_TMP after $(du -sh "$tmp_root" 2>/dev/null | tail -1)"
