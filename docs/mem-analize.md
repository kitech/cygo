已核实几处关键代码，确认存在。以下是 corona-c 子项目的内存问题审查结论（仅分析，未改动任何文件；如需修复请切换到 build 模式）。
高危问题（可崩溃 / 内存破坏 / 死锁）
1. coronagc.c:84 — GC_MALLOC 后直接 memset(ptr,0,size)，未检查 NULL（已核实）。OOM 时直接空指针写。同文件 realloc 等多处同样。
2. futex.c:127（已核实）— crn_mutex_unlock 唤醒等待者时传的是解锁者自己的 mygr->id/mcid，而 crn_procer_resume_one 校验 grid 不匹配就静默 return → 等待者永远不被唤醒（死锁）。应为 wtgr->id, wtgr->mcid。且 assert(rv2==CC_OK) 在队列为空时（CC_ERR_OUT_OF_RANGE）直接 abort。
3. hchan.c:98-103（已核实）— crn_hchan_close 的生命周期严重错误：
- bzero(hc, sizeof(crn_hchan)) 把 pthread_mutex_t lock 清零后再 pmutex_unlock → UB；
- 但所有 park 在 send/recv 上的 fiber 的 hclock = &hc->lock、以及 hcdt->wokehc 都还指向这个结构 → 被唤醒的 fiber 恢复执行即 use-after-free / 对已清零的锁操作；
- chan_dispose(hc->c) 销毁其他 fiber 可能正阻塞的 cond/mutex；
- crn_gc_free(hc) 目前实际是 no-op（见下），掩盖了问题但语义完全失真。
4. coronagc.c:104-112 — crn_gc_free 默认什么都不做（no-op），而所有调用方（hchan close、chan_dispose、datstu 等）都按“已释放”处理，"free 后置零/复用"语义全部失真。这是多个 UAF 问题的根因之一。
5. hselect.c:89-90 / 99-100（已核实） — szqueue_remove 队列空时返回 NULL，未检查就 hcdt->gr → 空指针解引用（chan 无等待者时 select 段错误）。对比 hchan.c:68 是有检查的，此处漏了。
6. hselect.c:153-165（已核实） — pass 3 中 cas 被显式置为 nilptr 后循环里直接用 cas->hcdt->wokeby，第一次迭代必然空指针崩溃。应为 sk->hcdt。
7. netpoller_epoll.c:72, 247+ — np->evfds[999] 固定数组，用系统任意 fd 直接索引，无边界检查，fd≥999 即越界写（内存破坏）。hashtable 字段存在但被注释掉未用。
8. netpoller_epoll.c:393-394 + 421-427 — epoll_ctl(ADD) 失败路径：evdata_free(d) 后 d2->dr/np->evfds[fd] 仍指向已 free 的 d，之后回调读它 → UAF。writefd 同样。
中危
- snprintf 返回值误用（多处越界写）：corona.c:409-421（crn_array_tostr_int）、corona_util.c:219-221（crn_simlog）、rxilog.c:208/277（log_log_nofmt）——len += vsnprintf(...) 累积的是“应有长度”，超过缓冲区后 sizeof(buf)-len-1 变负转 size_t 成巨值 → 越界写/读。
- netpoller_event.c:270-277 — 自建 addrinfo 链按 sizeof(struct sockaddr)(16B) 分配拷贝，但 IPv6 ai_addrlen=28 → 堆越界读。
- iohook2.c — getaddrinfo 返回的 res 链从未 freeaddrinfo（循环遍历到尾后对 NULL 调 free）→ 每次 DNS 解析真实泄漏一条 addrinfo 链（非 GC 内存，GC 兜不住）。
- netpoller_event.c:286-322 — DNS 回调写 fiber 栈变量（*resout2 = resout）：resume_prechk 校验与写回之间 fiber 可能已 finish 并 munmap 栈 → 写已 munmap 内存。结果应放 GC 堆 box 而非 fiber 栈。
- corona.c:304/342/1403 — GC_add_roots 与 GC_remove_roots 范围不匹配（remove 用了不同的 ssze 和 1 字节范围），根集单调增长。
- corona.c:119 — inuseids 从未 add（被注释），fiber id 复用导致 netpoller 残留事件可能命中新 fiber（UAF 窗口）；建议加 generation/seqno 校验。
- netpoller_ev.c:150-155 — YIELD_TYPE_SLEEP 缺 break，fallthrough 双重注册定时器/双重 resume；MSLEEP 落入 default 把毫秒数当 fd 用。
- netpoller_epoll.c:54-56 — 时间比较 tv_sec*CRN_USEC 用 int 乘法，>~2146 秒溢出，排序错乱可能引发 evdata 双重 free。
- hchan.c 等待队列 szqueue_init(256) 满时 assert(...) 直接 abort 而非阻塞；close 与并发 select 的 szqueue 竞态。
- chan.c:444-449 — srand/rand 进程级全局，与调度器互相踩踏（select 随机性被污染）。
- futex.c:49 — pcond_timedwait 可能在 initHook 前调用未初始化的函数指针。
低危 / 其他
- iohook.c:899-904：pselect 超时计算把 bts.tv_sec 加了两次（第二处应为 timeout->tv_sec）→ 定时错误。
- iohook.c:681, 909：VLA long tfds[nevts+1] 无上限 → 潜在栈溢出。
- libco/sjlj.c:79-81：co_derive 失败路径 co_delete 会 free 调用者提供的内存 → 潜在 double-free（ucontext.c 的做法是对的）。
- corona.c:1853-1890：crn_get_stats 内层 i 遮蔽外层、arr 从不 destroy；crn_destroy 不停线程。
- functrace.c:46：assert(1==2) 每次进入函数即触发（该文件一旦链接即崩）。
- 可见性类 data race：corona.c:1393 GC 线程非原子读 gr->state；hchan close 中 bzero 破坏 closed 原子性。
基本干净的文件
atomic.c、gcsp_corrector.c、hookbyplt.c、szqueue.c、libco/amd64.c、libco/aarch64.c、libco/ucontext.c、libco/libco.c、coro.c（上游 libcoro，mmap 栈 guard page 配对正确）、datstu.c 基本干净。
建议修复优先级
1. 先修必崩/必死锁的：H5（futex 唤醒死锁）、H1/H2（select 空指针）、H9（GC_MALLOC NULL）、H7（epoll 失败路径 UAF）、H6（evfds 越界）。
2. 设计层：明确 crn_gc_free 所有权（M10/H3 的根因）、重构 crn_hchan_close 生命周期（不 bzero/不 free，交给 finalizer）、fiber 销毁 vs 回调竞态加 generation。
3. 缓冲区类：统一 clamp snprintf 返回值；addrinfo 按 addrlen 分配；iohook2 补 freeaddrinfo。
4. 补 break（netpoller_ev）、修 pselect 超时、GC_add/remove_roots 配对。
