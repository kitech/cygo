#!/bin/bash
# main2.sh — 顺序执行 main2 的全部验证项
#   ./main2.sh            默认：构建→冒烟→过滤→栈探针→溢出取证→DNS→fd→valgrind
#   ./main2.sh quick      只跑冒烟+过滤+栈探针（~3 分钟）
#   ./main2.sh long       追加 30 分钟长跑
#   ./main2.sh all        追加 1 小时目标复现（CRN2_DNS=1）
set -u
cd "$(dirname "$0")"

MODE="${1:-default}"

# Android/Termux: /tmp is read-only. Prefer LOG_DIR, then $TMPDIR (Termux),
# then $PREFIX/tmp, then a repo-local directory.
if [ -z "${LOG_DIR:-}" ]; then
    if [ -d /tmp ] && [ -w /tmp ]; then
        LOG_DIR=/tmp/opencode/main2
    elif [ -n "${TMPDIR:-}" ] && [ -d "${TMPDIR}" ]; then
        LOG_DIR="${TMPDIR}/main2"
    elif [ -n "${PREFIX:-}" ] && [ -d "${PREFIX}/tmp" ]; then
        LOG_DIR="${PREFIX}/tmp/main2"
    else
        LOG_DIR="$(pwd)/main2-logs"
    fi
fi
if ! mkdir -p "$LOG_DIR" 2>/dev/null; then
    LOG_DIR="$(pwd)/main2-logs"
    mkdir -p "$LOG_DIR" || { echo "FAIL: cannot create log dir"; exit 1; }
fi
# main2.c s26 uses this for its file-io scenario
export CRN2_TMPDIR="${CRN2_TMPDIR:-$LOG_DIR}"

# ------------------------------------------------------- timeout forensics
# 超时后先抓现场再杀：/proc 快照 + gdb 全线程栈，最后才 TERM/KILL。
export CRN2_PTRACE="${CRN2_PTRACE:-1}"   # 允许 gdb 以兄弟进程身份 attach
BT_ON_TIMEOUT="${BT_ON_TIMEOUT:-1}"      # 0: 不抓现场，退回普通 timeout
BT_BUDGET="${BT_BUDGET:-20}"             # gdb 最多占用（秒）
BT_KILL_GRACE="${BT_KILL_GRACE:-5}"      # TERM 与 KILL 之间的宽限（秒）

BIN="${BIN:-./zig-out/bin/corona2}"
[ -x "$BIN" ] || BIN="./build/corona2"
[ -x "$BIN" ] || BIN="../build/corona2"

pass=0; fail=0; skip=0
script_start=$(date +%s)

say() { echo "[$(date '+%H:%M:%S')] $*"; }

fmt_dur() { # seconds -> 1m23s
    local s=$1
    if [ $s -ge 60 ]; then printf '%dm%02ds' $((s/60)) $((s%60)); else printf '%ds' $s; fi
}

# ---------------------------------------------------------------- estimates
case "$MODE" in
    bt-selftest) TOTAL_EST=0; MODE_DESC="bt-selftest（取证链路自检）" ;;
    quick)   TOTAL_EST=180;  MODE_DESC="quick（冒烟/过滤/栈探针）" ;;
    long)    TOTAL_EST=2050; MODE_DESC="long（默认项 + 30 分钟长跑）" ;;
    all)     TOTAL_EST=5700; MODE_DESC="all（long + 1 小时目标复现）" ;;
    *)       TOTAL_EST=520;  MODE_DESC="default（完整默认项）" ;;
esac

say "================================================================"
say "main2.sh mode=$MODE  $MODE_DESC"
say "log dir      : $LOG_DIR"
say "estimated    : ~$(fmt_dur $TOTAL_EST)（不含构建；超时上限另计）"
say "started at   : $(date '+%F %T')"
say "================================================================"

run() {                      # run <name> <est_sec> <timeout_sec> <cmd...>
    local name="$1"; local est="$2"; local to="$3"; shift 3
    local t0=$(date +%s)
    local hard=$(( to + BT_BUDGET + BT_KILL_GRACE + 20 ))
    say ""
    say "=== $name === start（est ~$(fmt_dur $est)，timeout ${to}s）"
    say "CMD: $*"
    rm -f "$LOG_DIR/$name.rc" "$LOG_DIR/$name.bt" "$LOG_DIR/$name.proc"
    ( with_timeout "$name" "$to" "$@" >"$LOG_DIR/$name.log" 2>&1; \
      echo $? >"$LOG_DIR/$name.rc" ) &
    local wpid=$!
    while kill -0 "$wpid" 2>/dev/null; do
        sleep 5
        if kill -0 "$wpid" 2>/dev/null; then
            say "  [$name] running $(( $(date +%s) - t0 ))s"
            tail -n 1 "$LOG_DIR/$name.log" 2>/dev/null | sed 's/^/      /'
        fi
        if [ $(( $(date +%s) - t0 )) -gt "$hard" ]; then
            say "  [$name] forensics wedged（>${hard}s），强制结束"
            kill -KILL "$wpid" 2>/dev/null
            break
        fi
    done
    wait "$wpid" 2>/dev/null
    local rc=$(cat "$LOG_DIR/$name.rc" 2>/dev/null)
    rm -f "$LOG_DIR/$name.rc"
    [ -n "$rc" ] || rc=1
    local el=$(( $(date +%s) - t0 ))
    if [ "$rc" -eq 0 ]; then
        say "PASS ${name}（实际 $(fmt_dur ${el})）"; pass=$((pass+1))
    elif [ "$rc" -eq 124 ]; then
        say "FAIL ${name}（timeout ${to}s，实际 $(fmt_dur ${el})）"
        grep -a -E 'main2 start|DONE|controller stalled' "$LOG_DIR/$name.log" 2>/dev/null \
            | tail -3 | sed 's/^/    /'
        tail -5 "$LOG_DIR/$name.log" 2>/dev/null | sed 's/^/    /'
        if [ -f "$LOG_DIR/$name.bt" ]; then
            say "    bt  : $LOG_DIR/$name.bt"
            grep -a -E '^[ *]*[0-9]+ +Thread|^#[0-9]+ ' "$LOG_DIR/$name.bt" \
                | head -40 | sed 's/^/    /'
        fi
        if [ -f "$LOG_DIR/$name.proc" ]; then
            say "    proc: $LOG_DIR/$name.proc"
        fi
        fail=$((fail+1))
    else
        say "FAIL ${name} rc=${rc}（实际 $(fmt_dur ${el})，log ${LOG_DIR}/${name}.log）"
        tail -5 "$LOG_DIR/$name.log"; fail=$((fail+1))
    fi
}
skip_if() { say "SKIP ${1}（${2}）"; skip=$((skip+1)); }

# ---- 超时取证：先抓现场，再 TERM/KILL；返回码仍是 124（同 GNU timeout）。
alive() {                   # alive <pid>：Linux 僵尸态视为已退出
    kill -0 "$1" 2>/dev/null || return 1
    if [ -r "/proc/$1/status" ]; then
        grep -q '^State:.*Z' "/proc/$1/status" 2>/dev/null && return 1
    fi
    return 0
}

pick_target() {             # pick_target <pid>：valgrind 场景取真正的被测进程
    local c
    if ! command -v pgrep >/dev/null 2>&1; then echo "$1"; return; fi
    c=$(pgrep -P "$1" 2>/dev/null | head -n1)
    if [ -n "$c" ]; then echo "$c"; else echo "$1"; fi
}

kill_tree() {               # kill_tree <pid>：先子后父
    local c
    if command -v pgrep >/dev/null 2>&1; then
        for c in $(pgrep -P "$1" 2>/dev/null); do kill_tree "$c"; done
    fi
    kill -KILL "$1" 2>/dev/null
    return 0
}

proc_dump() {               # proc_dump <pid> <out>
    local tgt="$1" out="$2" t cg
    {
        echo "== $(date '+%F %T') pid=$tgt"
        echo "== yama_ptrace_scope: $(cat /proc/sys/kernel/yama/ptrace_scope 2>/dev/null || echo n/a)"
        echo "== threads: $(ls /proc/$tgt/task 2>/dev/null | wc -l)"
        echo "== cgroup: $(tr '\n' ' ' < "/proc/$tgt/cgroup" 2>/dev/null)"
        for cg in $(awk -F: '{print $3}' "/proc/$tgt/cgroup" 2>/dev/null); do
            echo "   freeze($cg): $(cat "/sys/fs/cgroup$cg/cgroup.freeze" 2>/dev/null || echo n/a)"
        done
        for t in /proc/$tgt/task/*; do
            [ -d "$t" ] || continue
            echo "-- tid=${t##*/} comm=$(cat "$t/comm" 2>/dev/null) wchan=$(cat "$t/wchan" 2>/dev/null)"
            grep -E '^(State|SigBlk|SigPnd|ShdPnd|SigCgt)' "$t/status" 2>/dev/null | sed 's/^/     /'
            awk '{printf "     stat: state=%s utime=%s stime=%s\n", $3, $14, $15}' "$t/stat" 2>/dev/null
        done
    } >"$out" 2>&1
}

gdb_dump() {                # gdb_dump <pid> <out>
    local -a gto=()
    if ! command -v gdb >/dev/null 2>&1; then
        echo "no gdb: Termux 上执行 pkg install gdb" >"$2"
        return 1
    fi
    if command -v timeout >/dev/null 2>&1; then
        gto=(timeout $(( BT_BUDGET + 5 )))
    elif command -v gtimeout >/dev/null 2>&1; then
        gto=(gtimeout $(( BT_BUDGET + 5 )))
    fi
    "${gto[@]+"${gto[@]}"}" gdb -q -nx -batch -p "$1" \
        -ex 'set pagination off' -ex 'set confirm off' \
        -ex 'info threads' \
        -ex 'thread apply all bt' \
        -ex 'p (int)GC_stop_count' \
        -ex 'p (int)GC_world_is_stopped' \
        -ex 'detach' >"$2" 2>&1
}

dump_on_timeout() {         # dump_on_timeout <name> <pid> <target> <to>
    local name="$1" pid="$2" tgt="$3" to="$4" base="$LOG_DIR/$1"
    echo "" >>"$LOG_DIR/$name.log"
    echo "### TIMEOUT ${to}s pid=$pid target=$tgt" >>"$LOG_DIR/$name.log"
    proc_dump "$tgt" "$base.proc"
    gdb_dump "$tgt" "$base.bt"
    echo "### forensics: $base.proc  $base.bt" >>"$LOG_DIR/$name.log"
}

with_timeout() {            # with_timeout <name> <sec> <cmd...>  -> 124 on timeout
    local name="$1" to="$2"; shift 2
    if [ "$BT_ON_TIMEOUT" != "1" ]; then
        if command -v timeout >/dev/null 2>&1; then
            timeout -k "$BT_KILL_GRACE" "$to" "$@"; return $?
        elif command -v gtimeout >/dev/null 2>&1; then
            gtimeout -k "$BT_KILL_GRACE" "$to" "$@"; return $?
        fi
    fi
    "$@" &
    local pid=$!
    local tgt=$(pick_target "$pid")
    local deadline=$(( $(date +%s) + to )) i=0
    while alive "$pid" && [ "$(date +%s)" -lt "$deadline" ]; do sleep 1; done
    if ! alive "$pid"; then
        wait "$pid" 2>/dev/null
        return $?
    fi

    say "[$name] TIMEOUT ${to}s → 先抓现场，再 TERM/KILL（pid=$pid target=$tgt）"
    dump_on_timeout "$name" "$pid" "$tgt" "$to"

    kill -TERM "$pid" 2>/dev/null
    while alive "$pid" && [ "$i" -lt "$BT_KILL_GRACE" ]; do sleep 1; i=$((i+1)); done
    if alive "$pid" || alive "$tgt"; then
        say "[$name] TERM 后仍存活（pid=$pid target=$tgt）→ SIGKILL"
        kill_tree "$pid"
        if [ "$tgt" != "$pid" ]; then kill_tree "$tgt"; fi
    fi
    wait "$pid" 2>/dev/null
    return 124
}

# 隐藏模式：./main2.sh bt-selftest —— 验证"超时→抓现场→杀掉"整条链路
if [ "$MODE" = "bt-selftest" ]; then
    say "=== bt-selftest ==="
    # BT_SELFTEST_CMD 可换成真实目标，例如：
    #   BT_SELFTEST_CMD='CRN2_PTRACE=1 CRN2_SECONDS=600 CRN2_SCENARIOS=s1 exec ./zig-out/bin/corona2'
    if [ -n "${BT_SELFTEST_CMD:-}" ]; then
        with_timeout selftest 5 sh -c "$BT_SELFTEST_CMD" >"$LOG_DIR/selftest.log" 2>&1
    else
        with_timeout selftest 5 sleep 60 >"$LOG_DIR/selftest.log" 2>&1
    fi
    rc=$?
    say "rc=$rc（期望 124）"
    ls -l "$LOG_DIR/selftest.proc" "$LOG_DIR/selftest.bt" 2>/dev/null
    head -20 "$LOG_DIR/selftest.proc" 2>/dev/null | sed 's/^/  /'
    grep -a -m3 -E 'nanosleep|sleep' "$LOG_DIR/selftest.bt" 2>/dev/null | sed 's/^/  /'
    exit $(( rc != 124 ))
fi

# ---------------------------------------------------------------- build
build_t0=$(date +%s)
say ""
say "=== build === start（est ~$(fmt_dur 20)）"
if [ ! -x "$BIN" ]; then
    built=0
    if command -v zig >/dev/null 2>&1; then
        say "CMD: zig build corona2"
        if zig build corona2; then
            BIN="./zig-out/bin/corona2"
            built=1
        else
            say "zig build failed, falling back to CMake"
        fi
    fi
    if [ "$built" -eq 0 ]; then
        if [ -f ../build/CMakeCache.txt ]; then
            say "CMD: cmake --build ../build --target corona2"
            cmake --build ../build --target corona2 || { say "FAIL build"; exit 1; }
        else
            say "CMD: cmake -S .. -B ../build && cmake --build ../build --target corona2"
            cmake -S .. -B ../build >/dev/null || { say "FAIL build"; exit 1; }
            cmake --build ../build --target corona2 || { say "FAIL build"; exit 1; }
        fi
        BIN="../build/corona2"
    fi
else
    say "SKIP build（已存在 ${BIN}）"
fi
say "build done（$(fmt_dur $(( $(date +%s) - build_t0 )))）"
say "BIN: $BIN"
say "forensics: gdb=$(command -v gdb 2>/dev/null || echo none) pgrep=$(command -v pgrep 2>/dev/null || echo none) ptrace_scope=$(cat /proc/sys/kernel/yama/ptrace_scope 2>/dev/null || echo n/a)"

# 1) 冒烟
run smoke 50 120 env CRN2_SECONDS=45 CRN2_SCENARIOS=s1,s2,s4,s5,s6 "$BIN"

# 2) 场景过滤：close 状态机 + select 公平性
run filter 50 120 env CRN2_SECONDS=45 CRN2_SCENARIOS=s5,s21,s22 "$BIN"

# 3) 栈探针
run deep 65 300 env CRN2_DEEP=1 CRN2_SECONDS=60 CRN2_SCENARIOS=s17 "$BIN"

if [ "$MODE" = "quick" ]; then
    say ""
    say "== summary: pass=$pass fail=$fail skip=$skip mode=$MODE 总耗时 $(fmt_dur $(( $(date +%s) - script_start ))) =="
    exit $(( fail > 0 ))
fi

# 4) 蓄意栈溢出：期望 rc!=0 且日志含 SO- 标记（handler 生效）
say ""
say "=== overflow ===（est ~$(fmt_dur 10)，timeout 60s）"
say "CMD: env CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 $BIN"
t0=$(date +%s)
with_timeout overflow 60 env CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 "$BIN" >"$LOG_DIR/overflow.log" 2>&1
rc=$?
if [ $rc -ne 0 ] && [ $rc -ne 124 ] && grep -q "^SO-" "$LOG_DIR/overflow.log"; then
    say "PASS overflow（handler captured SO- markers，rc=${rc}，实际 $(fmt_dur $(( $(date +%s) - t0 )))）"
    pass=$((pass+1))
else
    say "FAIL overflow rc=${rc}（log ${LOG_DIR}/overflow.log）"
    tail -5 "$LOG_DIR/overflow.log"; fail=$((fail+1))
fi

# 5) DNS（受限环境跳过）
if command -v getent >/dev/null 2>&1 && getent hosts localhost >/dev/null 2>&1; then
    run dns 50 120 env CRN2_DNS=1 CRN2_SECONDS=45 CRN2_SCENARIOS=s10 "$BIN"
else
    skip_if dns "no resolver"
fi

# 6) fd 巡检（Linux）
if [ -d /proc/self/fd ]; then
    run fdcheck 50 120 env CRN2_SECONDS=45 CRN2_SCENARIOS=s23 "$BIN"
else
    skip_if fdcheck "not linux"
fi

# 7) valgrind（有则跑短场景）
if command -v valgrind >/dev/null 2>&1; then
    run valgrind 180 600 valgrind --leak-check=full --error-exitcode=99 \
        env CRN2_SECONDS=60 CRN2_SCENARIOS=s1,s5 "$BIN"
else
    skip_if valgrind "valgrind not installed"
fi

# 8) 长跑
if [ "$MODE" = "long" ] || [ "$MODE" = "all" ]; then
    run long 1830 2100 env CRN2_SECONDS=1800 CRN2_STATS_SEC=30 "$BIN"
fi

# 9) 目标复现（1 小时，DNS 开启）
if [ "$MODE" = "all" ]; then
    run soak1h 3630 3900 env CRN2_SECONDS=3600 CRN2_DNS=1 "$BIN"
fi

say ""
say "== summary: pass=$pass fail=$fail skip=$skip mode=$MODE =="
say "== estimated ~$(fmt_dur $TOTAL_EST)，实际 $(fmt_dur $(( $(date +%s) - script_start ))) =="
exit $(( fail > 0 ))
