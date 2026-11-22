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
    say ""
    say "=== $name === start（est ~$(fmt_dur $est)，timeout ${to}s）"
    say "CMD: $*"
    rm -f "$LOG_DIR/$name.rc"
    ( with_timeout "$to" "$@" >"$LOG_DIR/$name.log" 2>&1; \
      echo $? >"$LOG_DIR/$name.rc" ) &
    local wpid=$!
    while kill -0 "$wpid" 2>/dev/null; do
        sleep 5
        if kill -0 "$wpid" 2>/dev/null; then
            say "  [$name] running $(( $(date +%s) - t0 ))s"
            tail -n 1 "$LOG_DIR/$name.log" 2>/dev/null | sed 's/^/      /'
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
        tail -5 "$LOG_DIR/$name.log"; fail=$((fail+1))
    else
        say "FAIL ${name} rc=${rc}（实际 $(fmt_dur ${el})，log ${LOG_DIR}/${name}.log）"
        tail -5 "$LOG_DIR/$name.log"; fail=$((fail+1))
    fi
}
skip_if() { say "SKIP ${1}（${2}）"; skip=$((skip+1)); }

# macOS has no GNU `timeout`: prefer timeout/gtimeout, else emulate with a
# background watchdog. Returns 124 on timeout (same as GNU timeout).
with_timeout() {           # with_timeout <sec> <cmd...>
    local to="$1"; shift
    if command -v timeout >/dev/null 2>&1; then
        timeout -k 5 "$to" "$@"; return $?
    fi
    if command -v gtimeout >/dev/null 2>&1; then
        gtimeout -k 5 "$to" "$@"; return $?
    fi
    "$@" &
    local pid=$!
    ( sleep "$to"; kill -TERM "$pid" 2>/dev/null; sleep 2; kill -KILL "$pid" 2>/dev/null ) &
    local watchdog=$!
    wait "$pid"; local rc=$?
    kill "$watchdog" 2>/dev/null
    wait "$watchdog" 2>/dev/null
    if [ $rc -ge 128 ] && ! kill -0 "$pid" 2>/dev/null; then return 124; fi
    return $rc
}

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
with_timeout 60 env CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 "$BIN" >"$LOG_DIR/overflow.log" 2>&1
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
