#!/usr/bin/env bash
# GoPie 串行实验驱动：按队列顺序（每个项目先 goroutine 后 function）逐步骤执行
# 「重建副本+插桩+编译」->「run_full --resume 长跑」，全部完成后落 ALL_DONE。
#
# 常驻运行（由 serial_tick.sh 用 systemd-run --user 拉起，小时级监督只负责重启与汇报）：
#   systemd-run --user --unit=gopie-serial-chain --collect /usr/bin/bash /home/riri/projects/gopie/scripts/serial_chain.sh
#
# 幂等性：
#   - state/<step>.prepared 表示该副本已插桩并编译好二进制（重跑不重复插桩，插桩不幂等）
#   - state/<step>.done     表示该步骤长跑已把全部二进制跑完（run_full 写了 PROJECT_COMPLETE）
#   - 单个二进制级别的续跑由 run_full.sh --resume 的 .done 标记负责
# 失败策略：任一步骤准备或长跑未正常收尾，写 HALTED 原因并退出，等人工确认，绝不自动往下跑。

set -uo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
# shellcheck source=/dev/null
source "$script_dir/serial_queue.sh"

mkdir -p "$EXP_ROOT/state" "$EXP_ROOT/logs"
chain_log="$EXP_ROOT/logs/chain.log"

if ! require_roots; then
    echo "$(log_ts) ROOT_GUARD rejected chain start" | tee -a "$chain_log"
    printf '%s root guard rejected\n' "$(log_ts)" >"$EXP_ROOT/HALTED"
    exit 1
fi
note "$chain_log" "[ROOTS] gopie=$GOPIE_ROOT projects=$PROJECTS_ROOT exp=$EXP_ROOT retired=$RETIRED_ROOT"

# 单实例锁：上一轮孤儿进程没退干净时绝不允许第二个实例并写同一批结果
command -v flock >/dev/null 2>&1 || {
    echo "$(log_ts) FLOCK_MISSING: refuse to start without a single-instance lock" | tee -a "$chain_log"
    printf '%s flock missing\n' "$(log_ts)" >"$EXP_ROOT/HALTED"
    exit 1
}
exec 9>"$EXP_ROOT/chain.lock"
if ! flock -n 9; then
    echo "$(log_ts) ANOTHER_INSTANCE_RUNNING lock=$EXP_ROOT/chain.lock" | tee -a "$chain_log"
    exit 0
fi

if [[ -f "$EXP_ROOT/ALL_DONE" ]]; then
    note "$chain_log" "ALL_DONE already present, nothing to do"
    exit 0
fi
rm -f -- "$EXP_ROOT/HALTED"
note "$chain_log" "[CHAIN START] pid=$$ targets=${#SERIAL_TARGETS[@]} steps=$(queue_steps | wc -l)"

total_steps=$(queue_steps | wc -l)
step_index=0
for entry in $(queue_steps); do
    step_index=$((step_index + 1))
    project_dir=${entry%%:*}
    rest=${entry#*:}
    copy_name=${rest%%:*}
    mode=${rest##*:}
    step=$(step_id "$copy_name" "$mode")
    gran=$(granularity_of "$mode")

    if [[ -f "$EXP_ROOT/state/${step}.done" ]]; then
        note "$chain_log" "[$step_index/$total_steps] SKIP $step (already done)"
        continue
    fi

    resolve_paths "$project_dir" "$copy_name" "$mode" || {
        printf '%s prepare failed for %s: path check\n' "$(log_ts)" "$step" >"$EXP_ROOT/HALTED"
        note "$chain_log" "HALTED at $step: path check failed"
        exit 1
    }
    out_dir="$RESOLVED_COPY/gopieRes"
    bins_dir="$GOPIE_ROOT/testbins/$step"

    note "$chain_log" "[$step_index/$total_steps] START $step gran=$gran copy=$RESOLVED_COPY"

    if ! bash "$script_dir/prepare_copy.sh" --project-dir "$project_dir" --copy-name "$copy_name" --mode "$mode" \
        >>"$EXP_ROOT/logs/prepare_${step}.out.log" 2>&1; then
        printf '%s prepare_copy.sh failed for %s, see %s\n' "$(log_ts)" "$step" "$EXP_ROOT/logs/prep_${step}.log" >"$EXP_ROOT/HALTED"
        note "$chain_log" "HALTED at $step: PREPARE_FAIL"
        exit 1
    fi
    note "$chain_log" "[$step_index/$total_steps] PREPARE_OK $step"

    if [[ ! -d "$bins_dir" ]]; then
        printf '%s no bins dir for %s: %s\n' "$(log_ts)" "$step" "$bins_dir" >"$EXP_ROOT/HALTED"
        note "$chain_log" "HALTED at $step: missing $bins_dir"
        exit 1
    fi

    run_log="$EXP_ROOT/logs/run_${step}.log"
    note "$chain_log" "[$step_index/$total_steps] RUN $step -> $out_dir (log: $run_log)"
    bash "$GOPIE_ROOT/scripts/run_full.sh" \
        --bin-dir "$bins_dir" \
        --out-dir "$out_dir" \
        --granularity "$gran" \
        --timeout "$RUN_TIMEOUT" \
        --fuzz-time "$RUN_FUZZ_TIME" \
        --recover-timeout "$RUN_RECOVER_TIMEOUT" \
        --resume >"$run_log" 2>&1
    note "$chain_log" "[$step_index/$total_steps] run_full returned rc=$? for $step"

    if [[ -f "$out_dir/PROJECT_COMPLETE" ]]; then
        date '+%Y-%m-%d %H:%M:%S' >"$EXP_ROOT/state/${step}.done"
        note "$chain_log" "[$step_index/$total_steps] END $step (STEP_DONE)"
    else
        printf '%s run_full did not finish %s, see %s\n' "$(log_ts)" "$step" "$run_log" >"$EXP_ROOT/HALTED"
        note "$chain_log" "HALTED at $step: PROJECT_COMPLETE missing"
        exit 1
    fi
done

date '+%Y-%m-%d %H:%M:%S' >"$EXP_ROOT/ALL_DONE"
note "$chain_log" "[CHAIN ALL_DONE] all $total_steps steps finished"
