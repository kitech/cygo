# main2 — corona 长时间多场景 soak 测试

`main.c` 是最小 demo；`main2.c` 是长跑/混沌测试驱动，用于在分钟~小时级别
暴露协程调度、channel、netpoller、GC 与信号路径上的偶发问题，并在**没有
调用栈**的情况下仍能定位“死在哪个场景哪一步”。

## 构建

```bash
# CMake
cmake -B build && cmake --build build --target corona2

# Zig
cd corona-c && zig build corona2      # 产物 zig-out/bin/corona2
cd corona-c && zig build run2         # 直接运行（默认参数）
```

## 运行参数（环境变量）

| 变量 | 默认 | 含义 |
|---|---|---|
| `CRN2_SECONDS` | 3600 | 总运行时长（秒）|
| `CRN2_SEED` | time() | 随机种子（复现用）|
| `CRN2_SCENARIOS` | 全部 | 逗号分隔，如 `s1,s5,s21`（也接受完整名 `s5_closestorm`）|
| `CRN2_STATS_SEC` | 30 | 统计输出间隔 |
| `CRN2_DNS` | 0 | 启用 DNS 场景（s10）|
| `CRN2_DEEP` | 0 | 深递归栈探针（s17 加深）|
| `CRN2_OVERFLOW` | 0 | 蓄意触发 fiber 栈保护页（s18）|
| `CRN2_ABORT_ON_STALL` | 0 | 场景心跳停滞时主动 abort（拿 core）|
| `CRN2_DESTROY_ON_EXIT` | 0 | 结束时调用 `crn_destroy`（实验性）|

## 场景

| 编号 | 场景 | 覆盖能力 |
|---|---|---|
| s1 | ping-pong | 无缓冲 chan rendezvous、resume 竞态 |
| s2 | pipeline | 多级 stage、close 传播/EOF |
| s3 | worker pool | fan-out/fan-in、jobs 关闭后退出 |
| s4 | select 超时 | `caseDefault` + 轮询超时 |
| s5 | close 风暴 | close 唤醒、double-close、close 后 send/recv |
| s6 | mutex 竞争 | lock/trylock/unlock、计数校验 |
| s7 | cond 变量 | timedwait/signal/broadcast（小规模）|
| s8 | socketpair | 非阻塞 EAGAIN→yield、握手 |
| s9 | TCP 回环 | accept/connect/read/write |
| s10 | DNS | `crn_getaddrinfo` + `gethostbyname_r` |
| s11 | timer 风暴 | usleep/nanosleep 与并发 sleeper |
| s12 | fiber churn | post/exit、stats 校验 |
| s13 | GC 压力 | malloc/calloc/realloc/uncollectable/finalizer |
| s14 | 调度 churn | `crn_sched` |
| s15 | 管道/eventfd/poll | poll 超时与读写 |
| s16 | specifics | fiber 级 TLS |
| s17 | 栈探针 | `crn_fiber_stackaddr_cur`、递归深度 |
| s18 | 蓄意溢出 | altstack handler 取证（默认关）|
| s19 | chan-of-chans | 通过 chan 传递 chan 指针 |
| s20 | 信号量 | 缓冲 chan 当计数信号量 |
| s21 | select 公平性 | 双 chan 选择分布统计 |
| s22 | close 状态机 | 关闭后 send/recv/len/cap 语义 |
| s23 | fd 巡检 | `/proc/self/fd` 计数趋势（Linux）|
| s24 | epoll 直用 | epoll_create/epoll_wait + eventfd |
| s25 | select | fd_set + timeval 超时 |
| s26 | 文件 IO | open/read/write/fopen 钩子 |
| s27 | 优雅退出 | SIGTERM/SIGINT→协作退出（由主循环实现）|

## 验证命令

```bash
# 0) 构建（任选其一）
cmake -B build && cmake --build build --target corona2
cd corona-c && zig build corona2

# 1) 短跑冒烟（60s，核心场景，适合每次改动后）
CRN2_SECONDS=60 CRN2_SCENARIOS=s1,s2,s4,s5,s6 ./build/corona2

# 2) 全量长跑（30 分钟）
CRN2_SECONDS=1800 ./build/corona2

# 3) 高压栈用量（深递归场景）
CRN2_DEEP=1 CRN2_SECONDS=600 ./build/corona2

# 4) 目标复现：macOS 长跑 + DNS（配合 core dump）
ulimit -c unlimited
sudo sysctl -w kern.corefile=/cores/core.%P      # macOS
CRN2_SECONDS=3600 CRN2_DNS=1 ./corona2
# 崩溃后：lldb ./corona2 /cores/core.*
#        thread apply all bt / register read

# 5) 恶意栈溢出取证（验证 altstack handler 能否抓到现场）
CRN2_OVERFLOW=1 CRN2_SCENARIOS=s18 ./corona2

# 6) 场景过滤：只跑 select 公平性 / close 状态机
CRN2_SECONDS=300 CRN2_SCENARIOS=s5,s21 ./corona2

# 7) 稳定性辅助
CRN2_SECONDS=900 CRN2_STATS_SEC=10 ./corona2 | tee soak.log
grep -E "STALL|LEAK|FAIL|panic" soak.log

# 8) 内存/泄漏巡检（Linux）
valgrind --leak-check=full --error-exitcode=99 \
  env CRN2_SECONDS=120 CRN2_SCENARIOS=s1,s5 ./corona2

# 9) 长跑后统计趋势（脚本化建议）
grep "^STATS" soak.log | awk '{print $2, $5, $6}'   # 时间戳 / fiber_totcnt / fiber_totmem
```

一键顺序执行全部验证：

```bash
./main2.sh              # 默认：构建→冒烟→过滤→溢出取证→DNS→fd→valgrind（可用时）
./main2.sh quick        # 只跑冒烟+过滤（~2 分钟）
./main2.sh long         # 追加 30 分钟长跑
./main2.sh all          # 追加 1 小时目标复现（CRN2_DNS=1）
```

脚本特性：

- 开跑前打印 **mode / 日志目录 / 预估总耗时 / 开始时间**
- 每一步打印 **完整命令（含 env 变量）/ 预估耗时 / timeout 上限**
- 结束后打印 PASS/FAIL/SKIP 汇总与 **预估 vs 实际耗时**

日志输出到 `/tmp/opencode/main2/`（可用 `LOG_DIR=` 覆盖）。

## 崩溃取证设计

1. **进度槽**：每个场景每轮把 `{iter, op, heartbeat_ms}` 写入原子槽；
   信号处理器只调用 `write(2)` 输出 `SO-SLOT ...` / `SO-SIGNAL ...` →
   **即使没有栈也知道死在哪个场景哪一步**。
2. **watchdog**：心跳停滞超过 120s 打印 `STALL`；`CRN2_ABORT_ON_STALL=1`
   时主动 abort 以生成 core。
3. **altstack + SIGSEGV/SIGBUS/SIGABRT/SIGILL/SIGFPE**：主线程安装，并通过
   `crn_set_thread_createcb` 在**每个 machine 线程**入口再安装一次，覆盖
   “fiber 栈踩保护页”这类无栈现场。
4. **统计趋势**：周期输出
   `STATS t=.. fibers=.. act=.. mem=.. maxstk=.. mch=.. fin=..`。

## 已知边界

- s7（cond）基于 pthread 原语，仅做小规模单 mutex 的 wait/signal；失败只记录。
- s9/s10 依赖 loopback 与名称解析，受限环境会跳过或仅告警。
- s18 需要信号处理器生效；默认关闭（`CRN2_OVERFLOW=1` 开启）。
- s23 仅 Linux（读取 `/proc/self/fd`）。
- s27 的 `crn_destroy` 目前只置空全局指针，结束时调用可能留下悬挂线程，默认关闭。
