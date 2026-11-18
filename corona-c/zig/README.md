# corona-c · Zig 构建

与 `CMakeLists.txt` / `cxrt.cmake` 等价的 Zig 构建入口。**不改动任何 C 源码**，
CMake 路径仍然可用。

要求：Zig >= 0.17.0（开发机实测 0.17.1）。

## 快速开始

```bash
cd corona-c
zig build                 # crn 静态库 + corona 可执行 + crnreport 工具
zig build corona          # 只构建可执行
zig build tests           # 构建 14 个测试二进制（zig-out/bin/tXX_*）
zig build check           # 构建并依次运行全部测试
zig build check-str       # 运行 vendored zigstr 自带测试
zig build bdwgc-zig       # 仅编译 bdwgc-zig 绑定（不被引用、不链接）
zig build run             # 运行 corona 示例程序
zig build -Dnetpoller=epoll   # 切换 netpoller 后端
zig build -Doptimize=Debug    # 默认即 Debug
```

产物在 `zig-out/bin/`、`zig-out/lib/`；缓存 `.zig-cache/`。

## 编译速度优先（默认）

- C 侧 `-O0`、无 `-g`、无 `-fstack-protector-all` 插桩、`-pipe`
- 默认步骤只构建 `crn` + `corona` + `crnreport`；测试与 bdwgc-zig 需显式步骤
- 需要 C 调试符号 / 栈保护：`zig build -Dcdebug=true`
- 需要性能压测：`zig build -Doptimize=ReleaseFast`
- 并行度按 CPU 数，也可显式 `zig build -j8`

## Zig C-ABI 源码（`zigsrc/`）

`corona-c/zigsrc/corona_zig.zig` 导出 C ABI 函数（`crn_zig_*`），编进 `libcrn.a` 与
`libcrn.so`，C 侧通过 `zigsrc/corona_zig.h` 调用：

| 函数 | 说明 |
|---|---|
| `crn_zig_hello()` | 返回静态字符串 |
| `crn_zig_add(a,b)` | 整数加 |
| `crn_zig_fib(n)` | 迭代 fibonacci |
| `crn_zig_sum(ptr,len)` | i32 数组求和（i64） |
| `crn_zig_is_sorted(ptr,len)` | 是否非递减 |
| `crn_zig_banner(buf,cap)` | 写 `corona-zig/<ver>`，返回长度 |

- Zig 构建：`zig_demo = b.addObject(...pic=true)`，object 同时加入 `crn`/`crn_sh`
- CMake 构建：`cxrt.cmake` 中 `add_custom_command` 调用
  `zig build-obj -fPIC -O Debug -femit-bin=... corona_zig.zig`，
  产物作为 `EXTERNAL_OBJECT` 加入 `crn_objlib` 并显式并入 `crn_st`/`crn_sh`
- 机器上没有 zig 时仅告警，C-only 构建继续可用
- 验证：`corona-c/tests/t15_zigdemo.c`（`make -C corona-c/tests t15_zigdemo`）

## macOS：Zig 对象与 clang 链接

CMake 用 `zig build-obj` 生成 `crn_zig_demo.o`，再由 clang/ld 链接。系统链接器
**不提供 Zig 的 compiler-rt**，Debug 模式下对象会引用 `___zig_probe_stack`
（ziglang/zig#6817），导致 `Undefined symbols`。当前处理：

- `-fno-stack-check`：去掉该引用（对象内只有 zarray 等小帧函数）
- `-target <arch>-macos.<CMAKE_OSX_DEPLOYMENT_TARGET>`：消除
  “object built for newer macOS version” 警告，并对齐部署目标

若需保留栈探测，改为用 `zig build-lib -static -fcompiler-rt` 产出静态库并链接
（`cxrt.cmake` 注释中保留该备选命令）。Zig 自身构建路径（`zig build`）不受影响：
动态库/可执行由 Zig 链接，compiler-rt 自动包含。

### `_XOPEN_SOURCE` 与 BSD 类型（`u_char/u_int`）

`_XOPEN_SOURCE=600`（`ucontext_t` 必需）会让 Apple SDK 隐藏 BSD 扩展类型，
`iohook.c` 经 `iohook.h -> resolv.h -> nameser.h` 会报
`unknown type name 'u_char'`。修复：Apple 平台同时定义 `_DARWIN_C_SOURCE`
（`cxrt.cmake`、`build.zig`、`tests/Makefile` 均已加，Linux 不受影响）。

## 与 CMake 的对应关系

| CMake | Zig |
|---|---|
| `add_library(crn_objlib ...)` + `crn_st` | `crn` 静态库（`addLibrary`）→ `zig-out/lib/libcrn.a` |
| `add_library(crn_sh SHARED ...)` | `crn_sh` 共享库（同源集，PIC，链接 cxrt + 系统库）→ `zig-out/lib/libcrn.so` |
| `add_library(cxrt STATIC src/cxrtbase.c ...)` | `cxrt` 静态库 → `zig-out/lib/libcxrt.a` |
| `add_executable(corona main.c)` + `target_link_libraries(corona crn cxrt LIBGC)` | `corona` 可执行，链接 `crn` + `cxrt` + 系统 `gc/event/event_pthreads/sigsegv/dl/pthread` |
| `-Wl,--export-dynamic` | `exe.rdynamic = true` |
| `corona_c_flags`（`-DNRDEBUG -DCORO_* ...`） | `cflags` 常量，逐项一致 |
| `# netpoller_ev.c / netpoller_epoll.c` 注释切换 | `-Dnetpoller=libevent\|epoll` |
| `tests/Makefile` | `zig build tests` / `zig build check` |

## 依赖包（vendored，`zig/vendor/`）

采用 **path 依赖**：本机无法直连 GitHub/codeberg，`git+https` 依赖会让 `zig build` 挂起。

| 包 | 用途 | 版本 |
|---|---|---|
| `zig-string` (zigstr) | `crnreport` 的字符串格式化（真实使用） | master, 0.10.0 |
| `bdwgc-zig` | **只编译不引用**：`zig build bdwgc-zig` 通过 `addObject` 编译其 `bdwgc` 模块，不 import、不 link | master, 0.1.0 |
| `bdwgc` | bdwgc-zig 依赖的 GC C 源码 | `635112b` |
| `translate-c` | bdwgc-zig 生成绑定的工具 | `0da7a16c` |
| `aro` | translate-c 的 C 解析器 | `d0c8c4d9` |

各目录下 `UPSTREAM.md` 记录来源 URL、commit、本地补丁（仅把 `url/hash` 改成 `path`）
以及用 `gh-proxy.com` 前缀重新拉取的方法。

注意：`corona` 链接的是**系统 libgc**；vendored `bdwgc` 只为 bdwgc-zig 的编译服务。

## 已知问题

- `t07_mutex` 偶发挂起（底层多机 resume 竞态）；CI 可用
  `cd tests && RETRIES=2 ./runtests.sh` 复核。
- `zig build` 默认不构建 bdwgc-zig，避免首次构建耗时过长。
- `build.zig.zon` 的 `.fingerprint` 为占位值；首次 `zig build` 若报
  "invalid fingerprint"，按提示替换为它给出的值即可（仅本地信任标记）。

