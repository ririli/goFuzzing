#!/usr/bin/env bash
# 串行实验队列的配置与公共助手，被 prepare_copy.sh / serial_chain.sh / serial_tick.sh 共享。
#
# 口径（2026-09-30 与用户确认）：
#   - 8 个项目全部纳入，小项目在前，grpc 垫后；每个项目先 goroutine(G) 后 function(F)。
#   - G/F 副本一律从原始副本重新 cp -a；旧副本用 mv 留档，不做任何 rm -rf。
#   - 多 module 项目按「产品模块」口径：只移走没有测试包的嵌套 module，其余 module 都接线插桩。
#   - 长跑被中断后按二进制续跑（run_full.sh --resume）。

GOPIE_ROOT="${GOPIE_ROOT:-/home/riri/projects/gopie}"
PROJECTS_ROOT="${PROJECTS_ROOT:-/home/riri/realProjects}"
EXP_ROOT="${EXP_ROOT:-/home/riri/exp/serial}"
RETIRED_ROOT="${RETIRED_ROOT:-/home/riri/retired}"
CHAIN_UNIT="${CHAIN_UNIT:-gopie-serial-chain}"

# 队列顺序："项目目录名:副本基础名"
SERIAL_TARGETS=(
    "GORILLA:websocket"
    "GIN:gin"
    "GORUMS:gorums"
    "FIBER:fiber"
    "BEEGO:beego"
    "ETCD:etcd"
    "PROMETHEUS:prometheus"
    "GRPC:grpc"
)
# 每个项目内部的颗粒度顺序
SERIAL_MODES=(G F)

# 统一时间参数（见 AGENTS.md 实验时间标准）
RUN_TIMEOUT=300
RUN_FUZZ_TIME=1800
RUN_RECOVER_TIMEOUT=200

# 需要从 G/F 副本物理移走的嵌套 module（实测测试包数为 0 或与并发实验无关）。
# 只列出相对副本根的路径；移走的前提是该目录下确实有 go.mod。
declare -A PRUNE_DIRS=(
    [websocket]=""
    [gin]=""
    [beego]=""
    [fiber]=""
    [gorums]="examples"
    [etcd]="tools/mod tools/rw-heatmaps tools/testgrid-analysis"
    [prometheus]="compliance documentation/examples/remote_storage internal/tools web/ui/mantine-ui/src/promql/tools"
    [grpc]="cmd/protoc-gen-go-grpc examples gcp/observability interop/observability interop/xds security/advancedtls security/advancedtls/examples stats/opencensus test/tools"
)

log_ts() { date '+%Y-%m-%d %H:%M:%S'; }

# require_roots：路径一致性闸门。
# systemd-run --user 不会继承调用方的环境变量，曾经因此让一次沙箱自测静默回落到真实路径，
# 把 /home/riri/realProjects/GORILLA/websocketG 重新插桩了一遍。所以这里只允许两种布局：
#   1) 全部是真实默认路径（正式实验）
#   2) 显式 GOPIE_SANDBOX=/tmp/xxx，四个根全部由它派生（自测，且必须在 /tmp 下）
# 任何「只改了一个根」的混合状态一律拒绝运行。
require_roots() {
    if [[ -n "${GOPIE_SANDBOX:-}" ]]; then
        [[ "$GOPIE_SANDBOX" == /tmp/* ]] || {
            echo "ROOT_GUARD REJECT: GOPIE_SANDBOX must live under /tmp, got: $GOPIE_SANDBOX" >&2
            return 1
        }
        GOPIE_ROOT="$GOPIE_SANDBOX/gopie"
        PROJECTS_ROOT="$GOPIE_SANDBOX/proj"
        EXP_ROOT="$GOPIE_SANDBOX/exp"
        RETIRED_ROOT="$GOPIE_SANDBOX/retired"
        mkdir -p "$GOPIE_ROOT/scripts" "$GOPIE_ROOT/testbins" "$PROJECTS_ROOT" "$EXP_ROOT" "$RETIRED_ROOT"
        echo "ROOT_GUARD SANDBOX $GOPIE_SANDBOX"
        return 0
    fi
    local bad=""
    [[ "$GOPIE_ROOT" == "/home/riri/projects/gopie" ]] || bad+="GOPIE_ROOT=$GOPIE_ROOT "
    [[ "$PROJECTS_ROOT" == "/home/riri/realProjects" ]] || bad+="PROJECTS_ROOT=$PROJECTS_ROOT "
    [[ "$EXP_ROOT" == "/home/riri/exp/serial" ]] || bad+="EXP_ROOT=$EXP_ROOT "
    [[ "$RETIRED_ROOT" == "/home/riri/retired" ]] || bad+="RETIRED_ROOT=$RETIRED_ROOT "
    if [[ -n "$bad" ]]; then
        echo "ROOT_GUARD REJECT: partial override without GOPIE_SANDBOX -> $bad" >&2
        return 1
    fi
    return 0
}

# note <log-file> <消息...>：同时打到 stdout 和 chain log
note() {
    local logfile=$1
    shift
    printf '%s %s\n' "$(log_ts)" "$*" | tee -a "$logfile"
}

# step_id <copy> <mode> -> websocketG
step_id() { printf '%s%s' "$1" "$2"; }

# granularity_of <mode> -> goroutine|function
granularity_of() {
    case "$1" in
        G) echo goroutine ;;
        F) echo function ;;
        *) return 1 ;;
    esac
}

# 校验模式字母
mode_ok() {
    case "$1" in
        G | F) return 0 ;;
        *) return 1 ;;
    esac
}

# resolve_paths <project_dir> <copy> <mode>
# 设置 RESOLVED_ORIG / RESOLVED_COPY；名字严格校验后才拼路径，任何一条不满足就返回非 0。
resolve_paths() {
    local project_dir=$1 copy=$2 mode=$3
    [[ "$mode" == "G" || "$mode" == "F" ]] || { echo "BAD MODE: $mode (expect G or F)" >&2; return 1; }
    [[ "$project_dir" =~ ^[A-Z0-9_]+$ ]] || { echo "BAD PROJECT_DIR: $project_dir" >&2; return 1; }
    [[ "$copy" =~ ^[a-z][a-z0-9_]*$ ]] || { echo "BAD COPY NAME: $copy" >&2; return 1; }

    RESOLVED_ORIG="$PROJECTS_ROOT/$project_dir/$copy"
    RESOLVED_COPY="$PROJECTS_ROOT/$project_dir/$copy$mode"

    [[ "$RESOLVED_ORIG" != "$RESOLVED_COPY" ]] || { echo "COPY == ORIG: $RESOLVED_ORIG" >&2; return 1; }
    [[ -d "$RESOLVED_ORIG" && -d "$RESOLVED_ORIG/.git" ]] || {
        echo "ORIGINAL COPY MISSING OR NOT A GIT REPO: $RESOLVED_ORIG" >&2
        return 1
    }
    [[ ! -L "$RESOLVED_COPY" ]] || { echo "COPY IS A SYMLINK: $RESOLVED_COPY" >&2; return 1; }
    return 0
}

# wsl_env_for_systemd：从非登录 shell（wsl.exe 直接投递）里补齐用户级 systemd 的连接信息
export_systemd_user_env() {
    export XDG_RUNTIME_DIR="${XDG_RUNTIME_DIR:-/run/user/$(id -u)}"
    export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-unix:path=${XDG_RUNTIME_DIR}/bus}"
}

# queue_steps：把 SERIAL_TARGETS x SERIAL_MODES 展开成 step 列表（队列顺序）
queue_steps() {
    local target copy mode
    for target in "${SERIAL_TARGETS[@]}"; do
        copy=${target#*:}
        for mode in "${SERIAL_MODES[@]}"; do
            printf '%s:%s:%s\n' "${target%%:*}" "$copy" "$mode"
        done
    done
}
