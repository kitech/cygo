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

## 与 CMake 的对应关系

| CMake | Zig |
|---|---|
| `add_library(crn_objlib ...)` + `crn_st` | `crn` 静态库（`addLibrary`） |
| `add_executable(corona main.c)` + `target_link_libraries(corona crn cxrt LIBGC)` | `corona` 可执行，链接 `crn` + 系统 `gc/event/event_pthreads/sigsegv/dl/pthread` |
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

