#!/usr/bin/env bash
# 并行启动 function 与 goroutine 两组 run_full 实验（后台长跑任务）。

set -euo pipefail

usage() {
    cat <<'EOF'
Usage:
  bash scripts/do_run_full.sh PROJECT_DIR [options]

Options:
  --gopie-root DIR         GoPie 目录（默认: /home/riri/projects/gopie）
  --projects-root DIR      被测项目根目录（默认: /home/riri/realProjects）
  --copy-name NAME         三个副本的基础名（默认: PROJECT_DIR 的小写形式）
  --bin-dir-f DIR          function 模式测试二进制目录
  --bin-dir-g DIR          goroutine 模式测试二进制目录
  --timeout SECONDS        单次执行超时（默认: 300）
  --fuzz-time SECONDS      单个测试 fuzz 总时长（默认: 1800）
  --recover-timeout SEC    panic 后恢复超时（默认: 200）
  -h, --help               显示帮助

目录约定:
  /home/riri/realProjects/PROJECT_DIR/COPY_NAME
  /home/riri/realProjects/PROJECT_DIR/COPY_NAME[F|G]

Example:
  bash scripts/do_run_full.sh BEEGO
  # 使用 BEEGO/beego、BEEGO/beegoF、BEEGO/beegoG 三个副本
EOF
}

project_dir_name=""
copy_name=""
gopie_root="/home/riri/projects/gopie"
projects_root="/home/riri/realProjects"
bin_dir_f=""
bin_dir_g=""
timeout_seconds=300
fuzz_time_seconds=1800
recover_timeout_seconds=200

while (($# > 0)); do
    case "$1" in
        --gopie-root) gopie_root=${2:?missing value for --gopie-root}; shift 2 ;;
        --projects-root) projects_root=${2:?missing value for --projects-root}; shift 2 ;;
        --copy-name) copy_name=${2:?missing value for --copy-name}; shift 2 ;;
        --bin-dir-f) bin_dir_f=${2:?missing value for --bin-dir-f}; shift 2 ;;
        --bin-dir-g) bin_dir_g=${2:?missing value for --bin-dir-g}; shift 2 ;;
        --timeout) timeout_seconds=${2:?missing value for --timeout}; shift 2 ;;
        --fuzz-time) fuzz_time_seconds=${2:?missing value for --fuzz-time}; shift 2 ;;
        --recover-timeout) recover_timeout_seconds=${2:?missing value for --recover-timeout}; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        -*) echo "ERROR: unknown argument: $1" >&2; usage >&2; exit 2 ;;
        *)
            if [[ -n "$project_dir_name" ]]; then
                echo "ERROR: PROJECT_DIR may only be specified once" >&2
                exit 2
            fi
            project_dir_name=$1
            shift
            ;;
    esac
done

if [[ -z "$project_dir_name" ]]; then
    echo "ERROR: PROJECT_DIR is required" >&2
    usage >&2
    exit 2
fi

if [[ ! -d "$gopie_root" ]]; then
    echo "ERROR: GoPie directory not found: $gopie_root" >&2
    exit 1
fi

copy_name=${copy_name:-"${project_dir_name,,}"}
bin_dir_f=${bin_dir_f:-"$gopie_root/testbins/${copy_name}F"}
bin_dir_g=${bin_dir_g:-"$gopie_root/testbins/${copy_name}G"}
project_dir="$projects_root/$project_dir_name"
out_dir_f="$project_dir/${copy_name}F/gopieRes"
out_dir_g="$project_dir/${copy_name}G/gopieRes"
run_script="$gopie_root/scripts/run_full.sh"

if [[ ! -f "$run_script" ]]; then
    echo "ERROR: run_full.sh not found: $run_script" >&2
    exit 1
fi
if [[ ! -d "$bin_dir_f" || ! -d "$bin_dir_g" ]]; then
    echo "ERROR: test binary directories must exist: $bin_dir_f and $bin_dir_g" >&2
    exit 1
fi
if [[ ! -d "$project_dir/$copy_name" || ! -d "$project_dir/${copy_name}F" || ! -d "$project_dir/${copy_name}G" ]]; then
    echo "ERROR: project copies $copy_name/${copy_name}F/${copy_name}G were not found under: $project_dir" >&2
    exit 1
fi

mkdir -p -- "$out_dir_f" "$out_dir_g"

nohup bash "$run_script" \
    --bin-dir "$bin_dir_f" \
    --out-dir "$out_dir_f" \
    --granularity function \
    --timeout "$timeout_seconds" \
    --fuzz-time "$fuzz_time_seconds" \
    --recover-timeout "$recover_timeout_seconds" \
    >"$gopie_root/runF.log" 2>"$gopie_root/runF.err.log" &
pid_f=$!
echo "run_full(F) launched: PID=$pid_f"

nohup bash "$run_script" \
    --bin-dir "$bin_dir_g" \
    --out-dir "$out_dir_g" \
    --granularity goroutine \
    --timeout "$timeout_seconds" \
    --fuzz-time "$fuzz_time_seconds" \
    --recover-timeout "$recover_timeout_seconds" \
    >"$gopie_root/runG.log" 2>"$gopie_root/runG.err.log" &
pid_g=$!
echo "run_full(G) launched: PID=$pid_g"

printf 'Logs: %s, %s\n' "$gopie_root/runF.log" "$gopie_root/runG.log"
