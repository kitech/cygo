#!/bin/bash
# main2.sh — 顺序执行 main2 的全部验证项
#   ./main2.sh            默认：构建→冒烟→过滤→栈探针→溢出取证→DNS→fd→valgrind
#   ./main2.sh quick      只跑冒烟+过滤+栈探针（~3 分钟）
#   ./main2.sh long       追加 30 分钟长跑
#   ./main2.sh all        追加 1 小时目标复现（CRN2_DNS=1）
set -u
cd "$(dirname "$0")"

MODE="${1:-default}"
LOG_DIR="${LOG_DIR:-/tmp/opencode/main2}"
mkdir -p "$LOG_DIR"

BIN="${BIN:-./zig-out/bin/corona2}"
[ -x "$BIN" ] || BIN="./build/corona2"
[ -x "$BIN" ] || BIN="../build/corona2"

pass=0; fail=0; skip=0
script_start=$(date +%s)

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

echo "================================================================"
echo " main2.sh mode=$MODE  $MODE_DESC"
echo " log dir      : $LOG_DIR"
echo " estimated    : ~$(fmt_dur $TOTAL_EST)（不含构建；超时上限另计）"
echo " started at   : $(date '+%F %T')"
echo "================================================================"

run() {                      # run <name> <est_sec> <timeout_sec> <cmd...>
    local name="$1"; local est="$2"; local to="$3"; shift 3
    local t0=$(date +%s)
    echo
    echo "=== $name ===（est ~$(fmt_dur $est)，timeout ${to}s）"
    echo "CMD: $*"
    timeout "$to" "$@" >"$LOG_DIR/$name.log" 2>&1
    local rc=$?
    local el=$(( $(date +%s) - t0 ))
    if [ $rc -eq 0 ]; then
        echo "PASS $name（实际 $(fmt_dur $el)）"; pass=$((pass+1))
    elif [ $rc -eq 124 ]; then
        echo "FAIL $name（timeout ${to}s，实际 $(fmt_dur $el)）"
        tail -3 "$LOG_DIR/$name.log"; fail=$((fail+1))
    else
        echo "FAIL $name rc=$rc（实际 $(fmt_dur $el)，log $LOG_DIR/$name.log）"
        tail -5 "$LOG_DIR/$name.log"; fail=$((fail+1))
    fi
}
skip_if() { echo "SKIP $1（$2）"; skip=$((skip+1)); }

# ---------------------------------------------------------------- build
echo
echo "=== build ===（est ~$(fmt_dur 20)）"
if [ ! -x "$BIN" ]; then
    if command -v zig >/dev/null 2>&1; then
        echo "CMD: zig build corona2"
        zig build corona2 || { echo "FAIL build"; exit 1; }
        BIN="./zig-out/bin/corona2"
    elif [ -d build ] || [ -d ../build ]; then
        echo "CMD: cmake --build build --target corona2"
        cmake --build build --target corona2 2>/dev/null || cmake --build ../build --target corona2 || { echo "FAIL build"; exit 1; }
        [ -x ./build/corona2 ] && BIN="./build/corona2" || BIN="../build/corona2"
    else
        echo "CMD: cmake -B build && cmake --build build --target corona2"
        cmake -B build >/dev/null && cmake --build build --target corona2 || { echo "FAIL build"; exit 1; }
        BIN="./build/corona2"
    fi
else
    echo "SKIP build（已存在 $BIN）"
fi
echo "BIN: $BIN"

# 1) 冒烟
run smoke 50 120 env CRN2_SECONDS=45 CRN2_SCENARIOS=s1,s2,s4,s5,s6 "$BIN"

# 2) 场景过滤：close 状态机 + select 公平性
run filter 50 120 env CRN2_SECONDS=45 CRN2_SCENARIOS=s5,s21,s22 "$BIN"

# 3) 栈探针
run deep 65 300 env CRN2_DEEP=1 CRN2_SECONDS=60 CRN2_SCENARIOS=s17 "$BIN"

if [ "$MODE" = "quick" ]; then
    echo
    echo "== summary: pass=$pass fail=$fail skip=$skip mode=$MODE 总耗时 $(fmt_dur $(( $(date +%s) - script_start ))) =="
    exit $(( fail > 0 ))
fi

# 4) 蓄意栈溢出：期望 rc!=0 且日志含 SO- 标记（handler 生效）
echo
echo "=== overflow ===（est ~$(fmt_dur 10)，timeout 60s）"
echo "CMD: env CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 $BIN"
t0=$(date +%s)
timeout 60 env CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 "$BIN" >"$LOG_DIR/overflow.log" 2>&1
rc=$?
if [ $rc -ne 0 ] && [ $rc -ne 124 ] && grep -q "^SO-" "$LOG_DIR/overflow.log"; then
    echo "PASS overflow（handler captured SO- markers，rc=$rc，实际 $(fmt_dur $(( $(date +%s) - t0 )))）"
    pass=$((pass+1))
else
    echo "FAIL overflow rc=$rc（log $LOG_DIR/overflow.log）"
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

echo
echo "== summary: pass=$pass fail=$fail skip=$skip mode=$MODE =="
echo "== estimated ~$(fmt_dur $TOTAL_EST)，实际 $(fmt_dur $(( $(date +%s) - script_start ))) =="
exit $(( fail > 0 ))
