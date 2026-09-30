#!/usr/bin/env bash
# 小时级监督入口：由 Windows 侧的 Qoder 自动化每小时调用一次
#   WSL_UTF8=1 wsl.exe -- bash /home/riri/projects/gopie/scripts/serial_tick.sh
# 职责只有三件：保证串行驱动活着（WSL 重启后拉回来）、打印 STATUS、全部完成时报 ALL_DONE。
# 真正的实验逻辑在 serial_chain.sh / prepare_copy.sh / run_full.sh --resume 里。

set -uo pipefail

# --report-only：只打印状态，不拉起驱动（人工检查队列时用）
report_only=false
if [[ "${1:-}" == "--report-only" ]]; then
    report_only=true
    shift
fi

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=/dev/null
source "$script_dir/serial_queue.sh"
export_systemd_user_env
if ! require_roots; then
    echo "STATUS_AT $(log_ts)"
    echo "RESULT HALTED"
    echo "HALTED_REASON ROOT_GUARD rejected: partial root override without GOPIE_SANDBOX"
    exit 1
fi

mkdir -p "$EXP_ROOT/state" "$EXP_ROOT/logs"
status_log="$EXP_ROOT/logs/tick.log"

steps_done=0
steps_total=0
cur_project="" cur_copy="" cur_mode=""
declare -a PENDING=()
while IFS=: read -r pd pc pm; do
    steps_total=$((steps_total + 1))
    if [[ -f "$EXP_ROOT/state/$(step_id "$pc" "$pm").done" ]]; then
        steps_done=$((steps_done + 1))
    elif [[ -z "$cur_project" ]]; then
        cur_project=$pd cur_copy=$pc cur_mode=$pm
    else
        PENDING+=("$pc$pm")
    fi
done < <(queue_steps)

echo "STATUS_AT $(log_ts)"
echo "QUEUE_PROGRESS $steps_done/$steps_total steps finished"

if [[ -f "$EXP_ROOT/ALL_DONE" ]]; then
    echo "RESULT ALL_DONE (finished $(cat "$EXP_ROOT/ALL_DONE"))"
    exit 0
fi

if [[ -f "$EXP_ROOT/HALTED" ]]; then
    echo "RESULT HALTED"
    echo "HALTED_REASON $(cat "$EXP_ROOT/HALTED")"
    echo "NOT auto-retrying; inspect the log above, then: rm -f $EXP_ROOT/HALTED  (人工确认后才做)"
    exit 0
fi

if [[ -z "$cur_project" ]]; then
    echo "RESULT ALL_DONE (no pending step)"
    exit 0
fi

cur_step=$(step_id "$cur_copy" "$cur_mode")
resolve_paths "$cur_project" "$cur_copy" "$cur_mode" || {
    echo "RESULT HALTED"
    echo "HALTED_REASON path check failed for $cur_step"
    exit 0
}

# 驱动是否活着：systemd 单元 + 实际进程双查（transient 单元扛不过 WSL 重启）
unit_active=$(systemctl --user is-active "$CHAIN_UNIT" 2>/dev/null)
[[ -n "$unit_active" ]] || unit_active=unknown
fuzz_alive=$(pgrep -f 'bin/fuzz --task full' >/dev/null 2>&1 && echo yes || echo no)
prep_alive=$(pgrep -f 'scripts/serial_chain.sh' >/dev/null 2>&1 && echo yes || echo no)

if [[ "$unit_active" != "active" && "$prep_alive" != "yes" ]]; then
    if $report_only; then
        echo "ACTION SKIPPED_LAUNCH (--report-only)"
    else
        echo "ACTION RELAUNCH serial_chain (unit was: $unit_active)"
        # systemd-run 不继承调用方环境，必须显式把根路径带过去，否则会静默回落到默认值
        systemd-run --user --unit="$CHAIN_UNIT" --collect \
            --setenv="GOPIE_SANDBOX=${GOPIE_SANDBOX:-}" \
            --setenv="GOPIE_ROOT=$GOPIE_ROOT" \
            --setenv="PROJECTS_ROOT=$PROJECTS_ROOT" \
            --setenv="EXP_ROOT=$EXP_ROOT" \
            --setenv="RETIRED_ROOT=$RETIRED_ROOT" \
            /usr/bin/bash "$script_dir/serial_chain.sh" \
            >>"$EXP_ROOT/logs/chain.out.log" 2>&1 || echo "ACTION RELAUNCH_FAILED (see $EXP_ROOT/logs/chain.out.log)"
        unit_active=$(systemctl --user is-active "$CHAIN_UNIT" 2>/dev/null)
        [[ -n "$unit_active" ]] || unit_active=unknown
    fi
else
    echo "ACTION KEEP (chain already running; unit=$unit_active, fuzz_process=$fuzz_alive)"
fi

echo "CURRENT_STEP $cur_step granularity=$(granularity_of "$cur_mode") copy=$RESOLVED_COPY"
echo "PENDING_STEPS ${PENDING[*]:-none}"

out_dir="$RESOLVED_COPY/gopieRes"
bins_dir="$GOPIE_ROOT/testbins/$cur_step"
if [[ -d "$bins_dir" ]]; then
    total_bins=$(find "$bins_dir" -type f -perm /111 2>/dev/null | wc -l)
else
    total_bins=0
fi
done_bins=$(find "$out_dir/.done" -maxdepth 1 -type f 2>/dev/null | wc -l)
echo "STEP_PREPARED $([[ -f "$EXP_ROOT/state/$cur_step.prepared" ]] && echo yes || echo 'no (still instrumenting/building)')"
echo "BINARY_PROGRESS $done_bins/$total_bins"

# ETA：用 .done 标记的 mtime 跨度估算平均单二进制耗时
if ((done_bins >= 2)); then
    mapfile -t mt < <(find "$out_dir/.done" -maxdepth 1 -type f -printf '%T@\n' 2>/dev/null | sort -n)
    first=${mt[0]%%.*}
    last=${mt[-1]%%.*}
    span=$((last - first))
    if ((span > 0 && done_bins >= 2)); then
        avg=$((span / (done_bins - 1)))
        remain=$(( (total_bins - done_bins) * avg ))
        printf 'ETA_STEP_REMAINING %dh%02dm (avg %ds/binary, %d of %d done)\n' \
            "$((remain / 3600))" "$((remain % 3600 / 60))" "$avg" "$done_bins" "$total_bins"
    fi
fi

panic_total=$(find "$PROJECTS_ROOT" -path '*/gopieRes/allpanic.txt' -exec cat {} + 2>/dev/null | grep -c 'panic' || true)
race_total=$(find "$PROJECTS_ROOT" -path '*/gopieRes/alldatarace.txt' -exec cat {} + 2>/dev/null | grep -c 'DATA RACE' || true)
echo "FINDINGS_TOTAL panics_lines=$panic_total datarace_lines=$race_total"
echo "CHAIN_LOG $EXP_ROOT/logs/chain.log"
echo "RUN_LOG $EXP_ROOT/logs/run_${cur_step}.log"
echo "HOST uptime=$(uptime -p 2>/dev/null | sed 's/^up //') disk_free=$(df -h / | awk 'NR==2{print $4}')"
tail -3 "$EXP_ROOT/logs/chain.log" 2>/dev/null | sed 's/^/CHAIN_TAIL /'
