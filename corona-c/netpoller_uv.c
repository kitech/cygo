/*
 * netpoller_uv.c — libuv backend for corona-c.
 *
 * 本文件不是默认后端，也未接入任何构建目标（不改 cxrt.cmake / build.zig）。
 * 目标：与默认的 netpoller_event.c（libevent）在"可观察行为"上等价，可互相替换。
 * 依赖：libuv >= 1.44（UV_DISCONNECT 自 1.9 起；本机实测 1.53.0）。
 *
 * 与 netpoller_event.c 的差异清单（逐条对应下文注释）：
 *
 *  [D1] 线程模型
 *       event 版：evthread_use_pthreads() 后，machine 线程可直接 event_add。
 *       uv  版：libuv 除 uv_async_send 外非线程安全，故所有 uv_* 调用都
 *               通过 uv_async_t + 互斥队列搬到 loop 线程执行。
 *
 *  [D2] 同一 fd 的多个等待者
 *       event 版：每个 yield 一个 struct event，libevent 内部 LIST_INSERT_HEAD
 *                串起来，同一 fd 的多个事件都会被回调（LIFO 顺序）。
 *       uv  版：libuv 明确禁止同一 fd 多个 active uv_poll（返回 UV_EEXIST），
 *               因此改成"每 fd 一个聚合 uv_poll + 等待者链表"，
 *               链表用头插复刻 LIFO 顺序。
 *
 *  [D3] 每次 IO 的 3.456s 超时
 *       event 版：event_add(evt, &tv{3,456000})，超时同样会回调并 resume。
 *       uv  版：uv_poll 没有超时参数，改为每个等待者配一个 uv_timer(3456ms)。
 *
 *  [D4] uv_poll_init 会强制 fd 进入 O_NONBLOCK
 *       event 版：event_add 不改 fd 标志。
 *       uv  版：uv_poll_init 内部调用 uv__nonblock(fd,1)（docs: 1.2.2 起）。
 *               这会破坏 corona 的协程 IO 语义（见函数内注释），
 *               因此 init 后必须用 fcntl(F_SETFL) 恢复原标志。
 *
 *  [D5] 定时器粒度
 *       event 版：epoll_wait 超时经 evutil_tv_to_msec_ 向上取整到 ms。
 *       uv  版：uv_timer 本身就是 ms，同样向上取整 -> 粒度为 ms 级，
 *               不是"需要自旋补偿的亚毫秒"，两边误差都在 ±1ms 内。
 *
 *  [D6] EV_CLOSED
 *       event 版：EV_CLOSED 在 Linux 走 EPOLLRDHUP，在 kqueue 走 EV_EOF。
 *       uv  版：UV_DISCONNECT 在 Linux 映射 POLLRDHUP，在 kqueue 由 EV_EOF
 *               映射（仅 AIX 不支持，本仓库不涉及）。
 *
 *  [D7] 句柄生命周期
 *       event 版：event_del + event_free 后立刻 resume，fd 可立即复用。
 *       uv  版：uv_poll_stop 是同步注销（fd 可立即复用），但 uv_close 的
 *               内存只能在 close_cb 里释放，所以先 stop 再 close 再 resume。
 *
 *  [D8] 错误上报
 *       event 版：EPOLLERR/HUP 都会触发 event 回调。
 *       uv  版：POLLERR 以 status<0（UV_EBADF）上报并停止该 handle；
 *               POLLHUP 会混入请求的事件位。两者都按"唤醒全部等待者"处理。
 *
 *  [D9] DNS（唯一无法做到实现级一致的项）
 *       event 版：evdns 自带解析器（读 /etc/resolv.conf + /etc/hosts）。
 *       uv  版：uv_getaddrinfo 在线程池里调系统 getaddrinfo（NSS：
 *               systemd-resolved / mDNS / LDAP / SSSD 等）。
 *               异步契约、/etc/hosts 行为一致；NSS 插件、search/ndots、
 *               超时重试、地址排序可能不同——这是设计差异，无法消除。
 *
 *  [D10] 主循环
 *       event 版：每轮 event_base_loopexit(586ms) + event_base_dispatch。
 *       uv  版：每轮重设 586ms 一次性 uv_timer + uv_run(UV_RUN_ONCE)。
 *
 *  [D11] 内存分配
 *       event 版：event_set_mem_functions(crn_gc_malloc, ...)。
 *       uv  版：libuv 无分配器钩子，内部用 libc malloc；
 *               我们自己的结构体用 crn_gc_* / crn_raw_*。
 *
 *  [D12] 普通文件
 *       event 版：Linux epoll ADD 对普通文件 EPERM（失败）；kqueue 可注册。
 *       uv  版：Linux uv__io_check_fd 返回 EPERM；kqueue 直接 UV_EINVAL。
 *               两者在 Linux 上一致；corona 的 read 只在 select 判"会阻塞"
 *               时才 yield，普通文件永远 ready，实际不会进入 netpoller。
 *
 *  [D13] 队列执行时机
 *       为规避 uv_poll_stop -> uv_close 的 double-invalidate 与 fd 复用竞态，
 *       所有 uv_* 的注册动作都在 uv_prepare 回调里执行（prepare 在 poll 之前、
 *       上一轮 close 之后），而不是放在 uv_async 回调里。async 只负责唤醒。
 */

#include <assert.h>
#include <stdbool.h>
#include <stdint.h>
#include <inttypes.h>
#include <errno.h>
#include <string.h>
#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>
#include <netdb.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>

#include <uv.h>

#include "coronagc.h"
#include "coronapriv.h"
#include "futex.h"
#include "yieldtypes.h"

// 日志与 gclock 辅助（与 netpoller_event.c 相同的 extern）
extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);
extern void crn_procer_resume_one(void* cbdata, int ytype, int grid, int mcid);
extern bool crn_procer_resume_prechk(void* gr_, int ytype, int grid, int mcid);

/*
 * [D3] 与 event 版 event_add(evt, {3,456000}) 对应的单次 IO 超时。
 */
#define CRN_UV_IO_TIMEOUT_MS 3456

/*
 * [D10] event 版 event_base_loopexit 的 586ms 周期。
 */
#define CRN_UV_HEARTBEAT_MS 586

typedef struct netpoller netpoller;

/*
 * evdata / evdata_new / evdata_free：
 * 保留与 netpoller_event.c 同名的非 static 符号（符号级兼容），
 * 便于将来只替换编译单元；二者只被本文件内部使用。
 */
typedef struct evdata {
    int evtyp;
    void* data; // fiber*
    int grid;
    int mcid;
    int ytype;
    int64_t fd; // fd or ns or hostname
    void** out; // dns 输出指针（保留字段以对齐 event 版结构）
    int* errcode;
    uint64_t deadline_ns;
    uint32_t seqno;
} evdata;

evdata* evdata_new(int evtyp, void* data) {
    assert(evtyp >= 0);
    evdata* d = crn_gc_malloc(sizeof(evdata));
    memset(d, 0, sizeof(evdata));
    d->evtyp = evtyp;
    d->data = data;
    return d;
}

void evdata_free(evdata* d) {
    crn_gc_free(d);
}

/*
 * [D2] 为什么不是一个 yield 一个 uv_poll：
 * libuv docs/poll："It is not okay to have multiple active poll handles for
 * the same socket"；源码里 uv_poll_init()/uv_poll_start() 在发现同一 fd
 * 已被别的 watcher 占用时返回 UV_EEXIST。
 * 而 event 版允许同一 fd 多个 struct event（libevent evmap 用链表聚合）。
 * 因此这里用"每 fd 一个聚合句柄 + 等待者链表"来达到同样效果。
 *
 * 链表用头插（prepend）：libevent evmap_io_add_ 是 LIST_INSERT_HEAD，
 * 所以同一 fd 的后注册事件先被回调；头插即复刻该 LIFO 顺序。
 */
typedef struct uvwaiter {
    struct uvwaiter* next;
    int want;                 // UV_READABLE / UV_WRITABLE
    int grid;
    int mcid;
    int ytype;
    void* data;               // fiber*
    evdata* d;
    np_token* tok;
    struct uvfd* fdref;
    uv_timer_t tmo;           // [D3] 每个等待者独立的 3456ms 超时
} uvwaiter;

typedef struct uvfd {
    int fd;
    int pevents;              // 当前 uv_poll_start 的聚合事件位
    int closing;              // [D7] uv_close 已发出、close_cb 未执行
    uv_poll_t poll;
    uvwaiter* waiters;        // 单链表，头插 => LIFO
    struct uvjob* deferred_head;     // [D7] 等待 close_cb 后重新建 handle 的 job
    struct uvjob* deferred_tail;
} uvfd;

/*
 * [D1] 跨线程投递的 job。machine 线程只入队，loop 线程在 uv_prepare 里消费。
 */
enum uvjobtyp {
    UVJOB_READ = 1,
    UVJOB_WRITE,
    UVJOB_TIMER,
    UVJOB_DNS,
    UVJOB_CANCEL, // 取消一批未决令牌（destroy/poll轮清理）
};

typedef struct uvjob {
    struct uvjob* next;
    int typ;
    int64_t fd;               // fd；timer 时为 ns（仅日志用途）
    uint64_t deadline_ns;     // timer 的绝对截止时间
    int ytype;
    int grid;
    int mcid;
    void* data;               // fiber*
    struct uvdns* dns;
    np_token* tok;            // 未决事件令牌（机器线程入列，loop 侧挂 obj）
    void** carr;              // UVJOB_CANCEL: np_token* 数组
    int ccnt;
} uvjob;

/*
 * [D9] DNS 请求容器（opaque 返回给 iohook2.c，只判 NULL）。
 * uv_getaddrinfo 的 req 是 request 而非 handle，完成回调后可直接释放本结构。
 */
typedef struct uvdns {
    uv_getaddrinfo_t req;
    int ytype;
    int grid;
    int mcid;
    void* data;               // fiber*
    void** out;               // struct addrinfo**
    int* errcode;
    char* node;               // 自己复制一份，uv_getaddrinfo 调用后即可释放
    np_token* tok;
} uvdns;

typedef struct uvtimer {
    uv_timer_t t;
    evdata* d;
    np_token* tok;
} uvtimer;

struct netpoller {
    uv_loop_t loop;
    uv_async_t async;         // [D1] 唯一跨线程安全的唤醒手段
    uv_prepare_t prepare;     // [D13] 真正的 uv 注册动作都在这里执行
    uv_timer_t heartbeat;     // [D10] 586ms 心跳
    uvfd** fds;               // fd -> 聚合句柄（可增长，不回收）
    size_t nfds;
    pmutex_t qmu;             // 保护 job 队列
    uvjob* qhead;
    uvjob* qtail;
};

static netpoller* gnpl__ = 0;

/* 前向声明：下面几个回调互相引用，C 需要先声明。 */
static void np_poll_cb(uv_poll_t* h, int status, int events);
static void np_waiter_free_cb(uv_handle_t* h);
static void np_uvtimer_free_cb(uv_handle_t* h);
static void np_fd_maybe_close(netpoller* np, uvfd* f);
static int np_start_io(uvjob* j);

/* ------------------------------------------------------------------ */
/* job 队列                                                            */
/* ------------------------------------------------------------------ */

/*
 * [D1] event 版在这里直接 event_add（libevent 线程安全）。
 * uv 版只入队 + uv_async_send（唯一允许从任意线程调用的 libuv API）。
 *
 * [gclock 差异] event 版在 machine 线程的 add 前后调 crn_pre/post_gclock_proc；
 * 这里真正的 uv 调用发生在 netpoller 线程（其 mcid 不在 3..5 的 safepoint
 * 检查范围内），所以 gclock 只包住 enqueue 这一小段，保持"machine 在钩子内"
 * 的语义，而不在 loop 线程上动作。
 */
static void np_enqueue(uvjob* j) {
    netpoller* np = gnpl__;
    assert(np != 0);
    crn_pre_gclock_proc(__func__);
    pthread_mutex_lock(&np->qmu);
    j->next = nilptr;
    if (np->qtail) {
        np->qtail->next = j;
    } else {
        np->qhead = j;
    }
    np->qtail = j;
    pthread_mutex_unlock(&np->qmu);
    crn_post_gclock_proc(__func__);
    uv_async_send(&np->async);
}

static uvjob* np_job_new(int typ, int64_t fd, int ytype, fiber* gr) {
    uvjob* j = crn_raw_malloc(sizeof(uvjob));
    if (j == nilptr) {
        return nilptr;
    }
    memset(j, 0, sizeof(*j));
    j->typ = typ;
    j->fd = fd;
    j->ytype = ytype;
    j->grid = gr->id;
    j->mcid = gr->mcid;
    j->data = gr;
    // 令牌在机器线程入列（fiber 自己的 machine = 单写者），随 job 发往
    // loop；loop 侧把 watcher 挂到 obj 上。DNS 的令牌在 netpoller_dnsresolv
    // 里创建（挂在 r->tok），这里不重复。
    if (typ != UVJOB_DNS) {
        np_token* tok = (np_token*)crn_gc_malloc(sizeof(np_token));
        if (tok != nilptr) {
            tok->kind = (typ == UVJOB_TIMER) ? NPT_TIMER : NPT_IO;
            tok->obj = nilptr;
            crn_npreg_add(gr, tok);
            j->tok = tok;
        }
    }
    return j;
}

/* ------------------------------------------------------------------ */
/* fd -> 聚合 poll 句柄                                                */
/* ------------------------------------------------------------------ */

/*
 * [D4] 关键且必须：uv_poll_init() 会把 fd 设成 O_NONBLOCK
 * （docs/poll: changed in 1.2.2 "the file descriptor is set to non-blocking
 * mode"；源码 uv_poll_init() 内 uv__nonblock(fd, 1)）。
 *
 * 这会直接打断 corona 的协程 IO：crn_fd_would_blocking() 开头就
 *     if (fd_is_nonblocking(fd)) return 0;        // iohook.c:222
 * 而 fd_is_nonblocking() 读的是内核 F_GETFL（iohookcb.c:65），
 * 于是所有 hooked read/recv/send/connect 都会跳过 yield 判断，
 * 直接把 EAGAIN 返回给应用（connect 还会跳过协程重连路径）。
 * event 版的 event_add 不做这个副作用，所以这里必须恢复原标志。
 *
 * 恢复必须走公开 fcntl()（不是 fcntl_f）：它的 F_SETFL 分支会先
 * hookcb_fd_set_nonblocking() 再 fcntl_f()（iohook.c:1139-1146），
 * 内核 O_NONBLOCK 与 corona 的 fdctx->isNonBlocking 一起归位。
 *
 * 恢复 O_NONBLOCK 对 uv_poll 自身安全：uv_poll 只把 fd 注册到
 * epoll/kqueue，不对 fd 做任何 I/O。
 * 残余风险：恢复前的几微秒内，若有其他线程并发使用同一 fd，
 * 会短暂看到非阻塞；正常使用（同一 fd 不跨线程并发 IO）无影响。
 */
static uvfd* np_fd_get(netpoller* np, int fd) {
    if (fd < 0) {
        return nilptr;
    }
    if ((size_t)fd >= np->nfds) {
        size_t ncap = np->nfds ? np->nfds : 64;
        while (ncap <= (size_t)fd) {
            ncap *= 2;
        }
        uvfd** nfds2 = crn_gc_realloc(np->fds, ncap * sizeof(uvfd*));
        if (nfds2 == nilptr) {
            return nilptr;
        }
        memset(nfds2 + np->nfds, 0, (ncap - np->nfds) * sizeof(uvfd*));
        np->fds = nfds2;
        np->nfds = ncap;
    }

    uvfd* f = np->fds[fd];
    if (f != nilptr) {
        return f;
    }

    /*
     * [D12] 普通文件：uv_poll_init 在 Linux 返回 UV_EPERM，在 kqueue 返回
     * UV_EINVAL。event 版在 Linux 也会因 epoll EPERM 失败（失败即丢弃、
     * 不 resume，行为一致）；kqueue 上 libevent 能注册普通文件，但 corona
     * 的 read 用 select 判阻塞，普通文件永远 ready，不会走到这里。
     */
    f = crn_raw_malloc(sizeof(uvfd));
    if (f == nilptr) {
        return nilptr;
    }
    f->fd = fd;
    f->pevents = 0;
    f->waiters = nilptr;

    int fl0 = fcntl(fd, F_GETFL);
    int rv = uv_poll_init(&np->loop, &f->poll, fd);
    if (rv != 0) {
        lwarn("uv_poll_init failed fd=%d rv=%d %s\n", fd, rv, uv_strerror(rv));
        crn_raw_free(f);
        return nilptr;
    }
    if (fl0 != -1) {
        fcntl(fd, F_SETFL, fl0);
    }
    f->poll.data = f;
    np->fds[fd] = f;
    return f;
}

static int np_fd_pevents(uvfd* f) {
    int ev = UV_DISCONNECT; // [D6] event 版每个 event 都带 EV_CLOSED
    for (uvwaiter* w = f->waiters; w != nilptr; w = w->next) {
        ev |= w->want;
    }
    return ev;
}

static int np_fd_apply(netpoller* np, uvfd* f) {
    (void)np;
    if (f->closing) {
        return 0;
    }
    int ev = np_fd_pevents(f);
    if (ev == f->pevents && f->pevents != 0) {
        return 0;
    }
    int prev = f->pevents;
    f->pevents = ev;
    int rv = uv_poll_start(&f->poll, ev, np_poll_cb);
    if (rv != 0) {
        /* 失败时恢复旧 mask：否则后续同 mask 的 apply 会被上面的
         * "ev == f->pevents" 短路，剩余等待者永远注册不上。
         * event 版每个 event 独立 event_add，没有这种粘住的状态。 */
        f->pevents = prev;
    }
    return rv;
}

/* ------------------------------------------------------------------ */
/* 唤醒 / 卸载                                                         */
/* ------------------------------------------------------------------ */

static int np_unlink(netpoller* np, uvfd* f, uvwaiter* w) {
    (void)np;
    if (f == nilptr) {
        return 0;
    }
    uvwaiter** pp = &f->waiters;
    while (*pp != nilptr) {
        if (*pp == w) {
            *pp = w->next;
            w->next = nilptr;
            return 1;
        }
        pp = &(*pp)->next;
    }
    return 0;
}

/*
 * [D7] 先 stop 再 close 再 resume：
 * uv_poll_stop() 是同步生效的（docs：effective immediately，连 pending
 * callback 都取消；源码 uv__io_stop 同步清 loop->watchers[fd] 并
 * EPOLL_CTL_DEL），因此 resume 后 fiber 立刻对同一 fd 再次 yield 不会
 * 和旧 handle 冲突，等价于 event 版 event_del+event_free 先于 resume。
 * 但 uv_close 之后 handle 内存只能在 close_cb 里释放（docs/handle 指南），
 * 所以 evdata 里的 grid/mcid/ytype/data 先拷出来，句柄内存延后释放。
 */
static void np_wake_waiter(netpoller* np, uvfd* f, uvwaiter* w) {
    (void)np;
    (void)f;
    // 先认领令牌：0=并发取消已接管（obj 由取消 job 释放），不碰 w；
    // 2=fiber 正在销毁，只做注销/释放，跳过唤醒
    int nprv = crn_npreg_claim(w->tok);
    if (nprv == 0) { return; }
    evdata* d = w->d;
    int grid = d->grid;
    int mcid = d->mcid;
    int ytype = d->ytype;
    void* data = d->data;

    uv_timer_stop(&w->tmo);
    uv_close((uv_handle_t*)&w->tmo, np_waiter_free_cb);
    w->d = nilptr;
    evdata_free(d);
    crn_npreg_done(w->tok);
    if (nprv == 1) {
        crn_procer_resume_one(data, ytype, grid, mcid);
    }
}

static void np_waiter_free_cb(uv_handle_t* h) {
    uvwaiter* w = (uvwaiter*)h->data;
    if (w != nilptr) {
        crn_raw_free(w);
    }
}

static void np_uvfd_free_cb(uv_handle_t* h) {
    uvfd* f = (uvfd*)h->data;
    if (f == nilptr) {
        return;
    }
    netpoller* np = gnpl__;

    /* [D7] 旧 handle 的 close_cb 到了，此时 uv__poll_close 的
     * uv__platform_invalidate_fd 已经执行完，才允许为同一 fd 建新 handle。
     * 先把 table 条目清掉，再把等待中的 job 重新走一遍 np_start_io。 */
    if (np != nilptr && f->fd >= 0 && (size_t)f->fd < np->nfds &&
        np->fds[f->fd] == f) {
        np->fds[f->fd] = nilptr;
    }

    uvjob* q = f->deferred_head;
    f->deferred_head = nilptr;
    f->deferred_tail = nilptr;
    crn_raw_free(f);

    while (q != nilptr) {
        uvjob* next = q->next;
        int own = np_start_io(q);
        if (own) {
            crn_raw_free(q);
        }
        q = next;
    }
}

/*
 * [D8] 与 event 版的事件分派对照：
 *  - status < 0 或 UV_DISCONNECT：对应 EPOLLERR/HUP/EV_CLOSED，唤醒全部；
 *  - UV_READABLE：唤醒所有读等待者（event 版所有 EV_READ 事件都回调）；
 *  - UV_WRITABLE：唤醒所有写等待者。
 * event 版回调根本不看 events（超时也 resume），所以这里也不把
 * "就绪但读不到数据"当错误；libuv 文档也说明 poll 可能假就绪，
 * 醒来后 iohook 会重新判断。
 *
 * 链表是单一链表（不是 rd/wr 两条），按 LIFO 顺序遍历唤醒，
 * 以复刻 libevent 同一个 LIST_INSERT_HEAD 链表的回调顺序。
 *
 * [D7] 顺序分三段，严格对齐 event 版 event_del+event_free 先于
 * crn_procer_resume_one：
 *   1) 只把命中的等待者摘到本地 woken 链；
 *   2) np_fd_maybe_close()（必要时 uv_poll_stop / 重算 mask）；
 *   3) 再 close 各等待者 timer 并 resume。
 * 这样 fiber 被 machine 线程恢复后立刻 close/复用同一 fd 时，
 * 旧 poll 已经注销，不会踩到 event 版没有的窗口。
 */
static void np_poll_dispatch(netpoller* np, uvfd* f, int status, int events) {
    int wakeall = (status < 0) || ((events & UV_DISCONNECT) != 0);
    int emask = events & (UV_READABLE | UV_WRITABLE);

    uvwaiter* woken = nilptr;
    uvwaiter* woken_tail = nilptr;

    /* 阶段 1：摘链，保持原 LIFO 顺序 */
    uvwaiter** pp = &f->waiters;
    while (*pp != nilptr) {
        uvwaiter* w = *pp;
        int hit = wakeall || ((emask & w->want) != 0);
        if (!hit) {
            pp = &w->next;
            continue;
        }
        *pp = w->next;
        w->next = nilptr;
        if (woken_tail != nilptr) {
            woken_tail->next = w;
        } else {
            woken = w;
        }
        woken_tail = w;
    }

    /* 阶段 2：先注销/更新 fd 的 poll */
    np_fd_maybe_close(np, f);

    /* 阶段 3：再唤醒 fiber */
    while (woken != nilptr) {
        uvwaiter* next = woken->next;
        np_wake_waiter(np, f, woken);
        woken = next;
    }
}

static void np_fd_maybe_close(netpoller* np, uvfd* f) {
    if (f == nilptr || f->closing) {
        return;
    }
    if (f->waiters != nilptr) {
        int rv = np_fd_apply(np, f);
        if (rv != 0) {
            lwarn("uv_poll_start failed fd=%d rv=%d %s\n", f->fd, rv, uv_strerror(rv));
        }
        return;
    }
    /*
     * [D7] 不立刻从 table 摘除，也不允许新 handle 在同一轮内创建：
     * uv_close 之后 uv__poll_close 还会再调一次 uv__platform_invalidate_fd，
     * 如果那时同一 fd 已经有新 handle 注册，会把新注册删掉。
     * 所以这里只 stop + close 并标记 closing；table 条目保留到 close_cb，
     * 期间到来的同 fd job 挂进 deferred，等 close_cb 里再建新 handle。
     */
    f->closing = 1;
    uv_poll_stop(&f->poll);
    uv_close((uv_handle_t*)&f->poll, np_uvfd_free_cb);
}

/* ------------------------------------------------------------------ */
/* loop 线程回调                                                       */
/* ------------------------------------------------------------------ */

static void np_io_timeout_cb(uv_timer_t* t) {
    netpoller* np = gnpl__;
    uvwaiter* w = (uvwaiter*)t->data;
    assert(np != 0);
    assert(w != nilptr);
    uvfd* f = w->fdref;
    if (np_unlink(np, f, w) == 0) {
        /* 已被 poll 回调唤醒并 close，定时器 callback 不应该再来。 */
        return;
    }
    /* [D7] 与 np_poll_dispatch 同序：先注销/更新 fd，再 resume。 */
    np_fd_maybe_close(np, f);
    np_wake_waiter(np, f, w);
}

static void np_poll_cb(uv_poll_t* h, int status, int events) {
    netpoller* np = gnpl__;
    uvfd* f = (uvfd*)h->data;
    assert(np != 0);
    assert(f != nilptr);
    np_poll_dispatch(np, f, status, events);
}

static void np_timer_cb(uv_timer_t* t) {
    uvtimer* ut = (uvtimer*)t->data;
    // 先认领令牌：0=取消已接管；2=销毁中跳过唤醒
    int nprv = crn_npreg_claim(ut->tok);
    if (nprv == 0) { return; }
    evdata* d = ut->d;
    int grid = d->grid;
    int mcid = d->mcid;
    int ytype = d->ytype;
    void* data = d->data;

    ut->d = nilptr;
    uv_close((uv_handle_t*)t, np_uvtimer_free_cb);
    evdata_free(d);
    crn_npreg_done(ut->tok);
    if (nprv == 1) {
        crn_procer_resume_one(data, ytype, grid, mcid);
    }
}

static void np_uvtimer_free_cb(uv_handle_t* h) {
    uvtimer* ut = (uvtimer*)h->data;
    if (ut != nilptr) {
        crn_raw_free(ut);
    }
}

/*
 * [D9] errcode 翻译（必须）：event 版通过 EVUTIL_EAI_* 返回的是平台
 * EAI_*（实测 glibc/OpenBSD: EAI_NONAME=-2；FreeBSD/NetBSD: =8）。
 * libuv 返回固定负值 UV_EAI_*（uv/errno.h 为 -3000..-3014），
 * 且 EAI_SYSTEM 会被映射成 -errno。不翻译会让 hooked getaddrinfo
 * 的返回码与默认后端不同。
 */
static int np_eai_from_uv(int status) {
    if (status == 0) {
        return 0;
    }
    switch (status) {
#ifdef EAI_CANCELED
    case UV_ECANCELED: return EAI_CANCELED;
#endif
#ifdef EAI_ADDRFAMILY
    case UV_EAI_ADDRFAMILY: return EAI_ADDRFAMILY;
#endif
#ifdef EAI_AGAIN
    case UV_EAI_AGAIN: return EAI_AGAIN;
#endif
#ifdef EAI_BADFLAGS
    case UV_EAI_BADFLAGS: return EAI_BADFLAGS;
#endif
#ifdef EAI_BADHINTS
    case UV_EAI_BADHINTS: return EAI_BADHINTS;
#endif
#ifdef EAI_CANCELED
    case UV_EAI_CANCELED: return EAI_CANCELED;
#endif
#ifdef EAI_FAIL
    case UV_EAI_FAIL: return EAI_FAIL;
#endif
#ifdef EAI_FAMILY
    case UV_EAI_FAMILY: return EAI_FAMILY;
#endif
#ifdef EAI_MEMORY
    case UV_EAI_MEMORY: return EAI_MEMORY;
#endif
#if defined(EAI_NODATA) && (!defined(EAI_NONAME) || EAI_NODATA != EAI_NONAME)
    case UV_EAI_NODATA: return EAI_NODATA;
#endif
#ifdef EAI_NONAME
    case UV_EAI_NONAME: return EAI_NONAME;
#endif
#ifdef EAI_OVERFLOW
    case UV_EAI_OVERFLOW: return EAI_OVERFLOW;
#endif
#ifdef EAI_PROTOCOL
    case UV_EAI_PROTOCOL: return EAI_PROTOCOL;
#endif
#ifdef EAI_SERVICE
    case UV_EAI_SERVICE: return EAI_SERVICE;
#endif
#ifdef EAI_SOCKTYPE
    case UV_EAI_SOCKTYPE: return EAI_SOCKTYPE;
#endif
    default:
        break;
    }
#ifdef EAI_SYSTEM
    return EAI_SYSTEM;
#elif defined(EAI_FAIL)
    return EAI_FAIL;
#else
    return -1;
#endif
}

/*
 * [D9] addrinfo 所有权：Unix 上 uv_freeaddrinfo() == freeaddrinfo()
 * （libuv getaddrinfo.c 末尾），调用方仍按原契约 freeaddrinfo。
 * 拷贝方式按 event 版"prepend 逆序"以保持对外顺序，但用 ai_addrlen
 * 而不是 sizeof(struct sockaddr) 拷贝——event 版对 IPv6 只拷 16 字节
 * （< 28），是既有越界 bug，这里不复刻。
 *
 * 内存用 crn_raw_malloc（= calloc），因此调用方 freeaddrinfo/free 都能释放。
 */
static struct addrinfo* np_dump_addrinfo(const struct addrinfo* src) {
    struct addrinfo* resout = nilptr;
    const struct addrinfo* ai = src;
    for (; ai != nilptr; ai = ai->ai_next) {
        struct addrinfo* tai = crn_raw_malloc(sizeof(struct addrinfo));
        if (tai == nilptr) {
            break;
        }
        tai->ai_flags = ai->ai_flags;
        tai->ai_family = ai->ai_family;
        tai->ai_socktype = ai->ai_socktype;
        tai->ai_protocol = ai->ai_protocol;
        tai->ai_addrlen = ai->ai_addrlen;
        tai->ai_addr = nilptr;
        tai->ai_canonname = nilptr;
        if (ai->ai_addr != nilptr && ai->ai_addrlen > 0) {
            tai->ai_addr = crn_raw_malloc((size_t)ai->ai_addrlen);
            if (tai->ai_addr == nilptr) {
                crn_raw_free(tai);
                break;
            }
            memcpy(tai->ai_addr, ai->ai_addr, (size_t)ai->ai_addrlen);
        }
        if (ai->ai_canonname != nilptr) {
            size_t n = strlen(ai->ai_canonname) + 1;
            tai->ai_canonname = crn_raw_malloc(n);
            if (tai->ai_canonname != nilptr) {
                memcpy(tai->ai_canonname, ai->ai_canonname, n);
            }
        }
        tai->ai_next = resout;
        resout = tai;
    }
    return resout;
}

static void np_dns_complete(uvdns* r, int status, struct addrinfo* res) {
    struct addrinfo* outres = nilptr;
    int errcode = 0;

    // 先认领令牌：0=取消已接管（obj 由取消路径善后）；2=销毁中跳过唤醒
    int nprv = crn_npreg_claim(r->tok);
    if (nprv == 0) { return; }

    if (status == 0) {
        outres = np_dump_addrinfo(res);
        uv_freeaddrinfo(res);
    } else {
        errcode = np_eai_from_uv(status);
        lwarn("%s -> %s\n", uv_strerror(status), r->node ? r->node : "(null)");
    }

    if (nprv == 2) {
        crn_npreg_done(r->tok);
        return;
    }
    if (crn_procer_resume_prechk(r->data, r->ytype, r->grid, r->mcid)) {
        if (r->out != nilptr) {
            *r->out = outres;
        }
        if (r->errcode != nilptr) {
            *r->errcode = errcode;
        }
        crn_npreg_done(r->tok);
        crn_procer_resume_one(r->data, r->ytype, r->grid, r->mcid);
    } else {
        lwarn("oops, seems gr gone %d\n", r->grid);
        if (outres != nilptr) {
            freeaddrinfo(outres);
        }
    }
    crn_npreg_done(r->tok);
    if (r->node != nilptr) {
        crn_raw_free(r->node);
        r->node = nilptr;
    }
    crn_raw_free(r);
}

static void np_dns_cb(uv_getaddrinfo_t* req, int status, struct addrinfo* res) {
    uvdns* r = (uvdns*)req->data;
    if (r == nilptr) {
        return;
    }
    np_dns_complete(r, status, res);
}

/* ------------------------------------------------------------------ */
/* job 分发（只在 loop 线程、uv_prepare 中执行）                         */
/* ------------------------------------------------------------------ */

static int np_start_io(uvjob* j) {
    netpoller* np = gnpl__;
    assert(np != 0);
    int fd = (int)j->fd;
    int want = (j->typ == UVJOB_WRITE) ? UV_WRITABLE : UV_READABLE;

    uvfd* f = np_fd_get(np, fd);
    if (f == nilptr) {
        lwarn("add error fd=%d grid=%d (poll handle)\n", fd, j->grid);
        return 1;
    }

    if (f->closing) {
        /* [D7] 旧 poll handle 正在 close、close_cb 还未执行：此刻直接建新
         * handle 会被旧 handle 之后的 invalidate 误删。先挂到 f->deferred，
         * 由 np_uvfd_free_cb 重新调用 np_start_io（那时 table 条目已清空，
         * 会为同一 fd 新建 handle）。返回 0 表示 job 所有权已转移。 */
        j->next = nilptr;
        if (f->deferred_tail != nilptr) {
            f->deferred_tail->next = j;
        } else {
            f->deferred_head = j;
        }
        f->deferred_tail = j;
        return 0;
    }

    uvwaiter* w = crn_raw_malloc(sizeof(uvwaiter));
    if (w == nilptr) {
        np_fd_maybe_close(np, f);
        return 1;
    }
    w->want = want;
    w->grid = j->grid;
    w->mcid = j->mcid;
    w->ytype = j->ytype;
    w->data = j->data;
    w->fdref = f;
    w->d = evdata_new(CXEV_IO, j->data);
    w->d->grid = j->grid;
    w->d->mcid = j->mcid;
    w->d->ytype = j->ytype;
    w->d->fd = j->fd;
    w->tok = j->tok;
    if (j->tok != nilptr) { j->tok->obj = w; }

    int rv = uv_timer_init(&np->loop, &w->tmo);
    if (rv != 0) {
        lwarn("uv_timer_init failed fd=%d rv=%d %s\n", fd, rv, uv_strerror(rv));
        evdata_free(w->d);
        crn_raw_free(w);
        np_fd_maybe_close(np, f);
        return 1;
    }
    w->tmo.data = w;
    rv = uv_timer_start(&w->tmo, np_io_timeout_cb, CRN_UV_IO_TIMEOUT_MS, 0);
    if (rv != 0) {
        lwarn("uv_timer_start failed fd=%d rv=%d %s\n", fd, rv, uv_strerror(rv));
        uv_close((uv_handle_t*)&w->tmo, np_waiter_free_cb);
        evdata_free(w->d);
        w->d = nilptr;
        np_fd_maybe_close(np, f);
        return 1;
    }

    /* 头插 => LIFO，对齐 libevent LIST_INSERT_HEAD */
    w->next = f->waiters;
    f->waiters = w;

    rv = np_fd_apply(np, f);
    if (rv != 0) {
        lwarn("add error fd=%d grid=%d rv=%d %s\n", fd, j->grid, rv, uv_strerror(rv));
        np_unlink(np, f, w);
        uv_timer_stop(&w->tmo);
        uv_close((uv_handle_t*)&w->tmo, np_waiter_free_cb);
        evdata_free(w->d);
        w->d = nilptr;
        np_fd_maybe_close(np, f);
        return 1;
    }
    return 1;
}

static void np_start_timer(uvjob* j) {
    netpoller* np = gnpl__;
    assert(np != 0);

    uint64_t now = uv_hrtime();
    uint64_t remain = (j->deadline_ns > now) ? (j->deadline_ns - now) : 0;
    /*
     * [D5] 粒度：event 版把 ns 写进 timeval 后，epoll_wait 的超时由
     * evutil_tv_to_msec_() 计算 = (usec+999)/1000，即"向上取整到 ms"。
     * libuv 的 uv_timer 同样是 ms 精度，因此这里先截到 usec 再 ceil 到 ms，
     * 与默认后端同粒度，无需任何亚毫秒自旋。
     * （两边都可能在 ms 边界上有 ±1ms 的取整差异，这是 API 固有。）
     */
    uint64_t usec = remain / 1000;
    uint64_t ms = (usec + 999) / 1000;

    uvtimer* ut = crn_raw_malloc(sizeof(uvtimer));
    if (ut == nilptr) {
        return;
    }
    ut->d = evdata_new(CXEV_TIMER, j->data);
    ut->d->grid = j->grid;
    ut->d->mcid = j->mcid;
    ut->d->ytype = j->ytype;
    ut->d->fd = j->fd;
    ut->d->deadline_ns = j->deadline_ns;
    ut->tok = j->tok;
    if (j->tok != nilptr) { j->tok->obj = ut; }

    int rv = uv_timer_init(&np->loop, &ut->t);
    if (rv != 0) {
        lwarn("uv_timer_init failed ns=%" PRId64 " rv=%d %s\n",
              j->fd, rv, uv_strerror(rv));
        evdata_free(ut->d);
        crn_raw_free(ut);
        return;
    }
    ut->t.data = ut;
    rv = uv_timer_start(&ut->t, np_timer_cb, ms, 0);
    if (rv != 0) {
        lwarn("add error ns=%" PRId64 " rv=%d %s\n", j->fd, rv, uv_strerror(rv));
        evdata* d = ut->d;
        ut->d = nilptr;
        uv_close((uv_handle_t*)&ut->t, np_uvtimer_free_cb);
        evdata_free(d);
        return;
    }
}

static void np_start_dns(uvjob* j) {
    netpoller* np = gnpl__;
    assert(np != 0);
    uvdns* r = j->dns;
    if (r == nilptr) {
        return;
    }

    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_flags = AI_CANONNAME;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_protocol = IPPROTO_TCP;

    int rv = uv_getaddrinfo(&np->loop, &r->req, np_dns_cb,
                            r->node, nilptr, &hints);
    r->req.data = r;
    /* uv_getaddrinfo 会复制 node/hints，返回后即可释放自己的副本。 */
    if (r->node != nilptr) {
        crn_raw_free(r->node);
        r->node = nilptr;
    }
    if (rv != 0) {
        /* [D9] 回调永不触发：走与回调相同的完成路径（非 NULL 返回值仍由调用方 yield）。 */
        np_dns_complete(r, rv, nilptr);
    }
}

static void np_cancel_tokens(uvjob* j) {
    netpoller* np = gnpl__;
    // loop 线程：释放机器线程摘除的未决令牌所挂接的对象。
    // DONE=唤醒路径已自行释放，跳过；在途 DNS 只能等完成回调善后。
    for (int i = 0; i < j->ccnt; i++) {
        np_token* tok = (np_token*)j->carr[i];
        if (tok == nilptr) { continue; }
        int st = __atomic_load_n(&tok->np.npstate, __ATOMIC_ACQUIRE);
        if (st == NPREG_DONE) { continue; }
        if (tok->kind == NPT_IO && tok->obj != nilptr) {
            uvwaiter* w = (uvwaiter*)tok->obj;
            uvfd* f = w->fdref;
            /* 未认领=唤醒从未发生：与 np_io_timeout_cb 同序：
             * 摘链 -> 注销fd -> 停timer/释放（含evdata） */
            (void)np_unlink(np, f, w);
            np_fd_maybe_close(np, f);
            evdata* d = w->d;
            w->d = nilptr;
            uv_timer_stop(&w->tmo);
            uv_close((uv_handle_t*)&w->tmo, np_waiter_free_cb);
            if (d != nilptr) { evdata_free(d); }
        } else if (tok->kind == NPT_TIMER && tok->obj != nilptr) {
            uvtimer* ut = (uvtimer*)tok->obj;
            evdata* d = ut->d;
            ut->d = nilptr;
            uv_close((uv_handle_t*)&ut->t, np_uvtimer_free_cb);
            if (d != nilptr) { evdata_free(d); }
        } else if (tok->kind == NPT_DNS && tok->obj != nilptr) {
            // uv_getaddrinfo 在途无法安全取消（uv_cancel 仅特定阶段有效）：
            // 留给完成回调；其 claim 会因 DRAINING/DYING 而跳过唤醒。
            // 这里不释放 r（uv 线程池仍持有）。
        }
        // 令牌本体是 GC 内存，弃置给 GC
    }
    crn_raw_free(j->carr);
}

static int np_dispatch(uvjob* j) {
    switch (j->typ) {
    case UVJOB_READ:
    case UVJOB_WRITE:
        return np_start_io(j);
    case UVJOB_TIMER:
        np_start_timer(j);
        return 1;
    case UVJOB_DNS:
        np_start_dns(j);
        return 1;
    case UVJOB_CANCEL:
        np_cancel_tokens(j);
        return 1;
    default:
        lwarn("unknown job typ %d\n", j->typ);
        return 1;
    }
}

/*
 * [D13] 所有 uv_poll_init/start、uv_timer_init/start、uv_getaddrinfo 都在
 * uv_prepare 中执行：
 *   - prepare 在 poll 之前、上一轮 close 之后，所以"关闭旧 poll handle"
 *     与"为同一 fd 建新 handle"永远不会落在同一轮 poll 批次里，
 *     避免 uv_poll_stop->uv_close 的二次 invalidate 误删新注册，
 *     也避免同一轮内 close 与 init 的 fd 复用竞态。
 *   - uv_async 只负责把 loop 从阻塞中唤醒，不做任何注册动作。
 */
static void np_prepare_cb(uv_prepare_t* ph) {
    netpoller* np = gnpl__;
    (void)ph;
    assert(np != 0);

    uvjob* q = nilptr;
    pthread_mutex_lock(&np->qmu);
    q = np->qhead;
    np->qhead = nilptr;
    np->qtail = nilptr;
    pthread_mutex_unlock(&np->qmu);

    while (q != nilptr) {
        uvjob* next = q->next;
        int own = np_dispatch(q);
        if (own) {
            crn_raw_free(q);
        }
        q = next;
    }
}

static void np_async_cb(uv_async_t* ah) {
    (void)ah;
    /* [D1][D13] 只负责唤醒；实际工作留给 np_prepare_cb。 */
}

static void np_heartbeat_cb(uv_timer_t* t) {
    (void)t;
}

/* ------------------------------------------------------------------ */
/* 对外接口                                                            */
/* ------------------------------------------------------------------ */

/*
 * [D1] event 版在这里做 evthread_use_pthreads() + event_set_mem_functions()。
 * libuv 没有"线程安全开关"，跨线程能力由 uv_async_send 提供，
 * 线程池/内存分配也由 libuv 自己管理，故此处为空实现。
 */
void netpoller_use_threads() {
}

/*
 * [D11] event 版用 crn_gc_malloc 分配 np，并把 libevent 的内存函数
 * 换成 crn_gc_*（event_set_mem_functions）。libuv 没有分配器钩子，
 * 内部一律 libc malloc；这里只保证我们自己的 np/队列/等待者内存策略
 * 与仓库一致（np 用 crn_gc_malloc，队列节点用 crn_raw_malloc）。
 *
 * uv_loop_t / uv_async_t / uv_prepare_t / uv_timer_t 先 memset 再 init
 * （GC_malloc 不清零）；初始化放在 netpoller_new（与 event 版
 * event_base_new 同位置），uv_run 在 netpoller_loop（netpoller 线程）。
 * libuv 只要求 loop 绑定单线程、不同时使用；源码无 thread-id 校验，
 * 顺序交接是安全的。
 */
netpoller* netpoller_new() {
    assert(gnpl__ == 0);

    netpoller* np = crn_gc_malloc(sizeof(netpoller));
    memset(np, 0, sizeof(netpoller));
    pmutex_init(&np->qmu, nilptr);
    np->qhead = nilptr;
    np->qtail = nilptr;
    np->fds = nilptr;
    np->nfds = 0;

    int rv = uv_loop_init(&np->loop);
    assert(rv == 0);

    rv = uv_async_init(&np->loop, &np->async, np_async_cb);
    assert(rv == 0);

    rv = uv_prepare_init(&np->loop, &np->prepare);
    assert(rv == 0);
    rv = uv_prepare_start(&np->prepare, np_prepare_cb);
    assert(rv == 0);

    rv = uv_timer_init(&np->loop, &np->heartbeat);
    assert(rv == 0);
    (void)rv; /* NDEBUG 下 assert 消失，避免 set-but-unused */

    gnpl__ = np;
    log_info("using backend libuv");
    return np;
}

/*
 * [D10] event 版每轮：
 *     struct timeval tv = {0, 586000};
 *     event_base_loopexit(np->loop, &tv);
 *     event_base_dispatch(np->loop);
 * 即"空闲最多 586ms 醒一次，有事件立刻返回"。
 * uv 版用一次性 586ms uv_timer + uv_run(UV_RUN_ONCE) 表达同一语义：
 * 每轮重设（忙时不会额外多醒），uv_run(ONCE) 对应 dispatch。
 * 尾部 assert(1==2) 与 event 版一致（本函数不返回）。
 */
void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != 0);

    for (;;) {
        uv_timer_start(&np->heartbeat, np_heartbeat_cb, CRN_UV_HEARTBEAT_MS, 0);
        int rv = uv_run(&np->loop, UV_RUN_ONCE);
        linfo("ohno, rv=%d\n", rv);
        (void)rv; /* NDEBUG 下 linfo 可能展开为空，避免 unused warning */
    }
    assert(1==2);
}

/*
 * 与 event 版完全相同的 ytype 分派（含 YIELD_TYPE_MSLEEP / TICK_WAKE /
 * GETADDRINFO 空操作 / CHAN_* 断言 / LOCK 不注册 fd）。
 * 唯一区别：这里不直接调用 uv_*，只把 "READ/WRITE/TIMER + 绝对 deadline"
 * 作为 job 入队，见 [D1]。
 *
 * deadline 在入队时用 uv_hrtime() 绝对化，排队延迟不会拉长 sleep；
 * event 版的 event_add 是相对超时，但同样在 yield 时计算。
 */
void netpoller_cancelfiber(fiber* gr) {
    // 机器线程摘令牌（单写者），经 uv_async 桥转 loop 线程释放挂接对象
    netpoller* np = gnpl__;
    if (np == 0 || gr == nilptr) { return; }
    for (;;) {
        void** arr = crn_raw_malloc(sizeof(void*) * 64);
        if (arr == nilptr) { return; }
        int n = crn_npreg_take(gr, arr, 64);
        if (n == 0) {
            crn_raw_free(arr);
            return;
        }
        uvjob* j = np_job_new(UVJOB_CANCEL, 0, 0, gr);
        if (j == nilptr) {
            crn_raw_free(arr);
            return;
        }
        j->carr = arr;
        j->ccnt = n;
        np_enqueue(j);
        if (n < 64) { return; }
    }
}
void netpoller_yieldfd(int64_t fd, int ytype, fiber* gr) {
    netpoller* np = gnpl__;
    assert(np != 0);
    (void)np;
    assert(ytype > YIELD_TYPE_NONE);
    assert(ytype < YIELD_TYPE_MAX);

    int64_t ns = 0;
    switch (ytype) {
    case YIELD_TYPE_SLEEP:
        ns = fd * 1000000000LL;
        break;
    case YIELD_TYPE_MSLEEP:
        ns = fd * 1000000LL;
        break;
    case YIELD_TYPE_USLEEP:
        ns = fd * 1000LL;
        break;
    case YIELD_TYPE_NANOSLEEP:
        ns = fd;
        break;
    case YIELD_TYPE_TICK_WAKE:
        ns = fd;
        break;
    case YIELD_TYPE_CHAN_SEND:
    case YIELD_TYPE_CHAN_RECV:
        /* event 版同样断言：这两个类型不应走到 netpoller。 */
        assert(1==2);
        ns = 1000;
        break;
    case YIELD_TYPE_LOCK:
        /* 互斥量停车：没有 fd 可注册，crn_mutex_unlock 会直接 resume。 */
        return;
    case YIELD_TYPE_CONNECT:
    case YIELD_TYPE_WRITE:
    case YIELD_TYPE_WRITEV:
    case YIELD_TYPE_SEND:
    case YIELD_TYPE_SENDTO:
    case YIELD_TYPE_SENDMSG: {
        uvjob* j = np_job_new(UVJOB_WRITE, fd, ytype, gr);
        if (j != nilptr) {
            np_enqueue(j);
        }
        return;
    }
    case YIELD_TYPE_GETADDRINFO:
        /* DNS 已在 netpoller_dnsresolv 里注册；这里与 event 版一样什么都不做。 */
        return;
    default:
        if (fd < 0) {
            lwarn("wtf yield fd=%" PRId64 " ytype=%d grid=%d\n", fd, ytype, gr->id);
            assert(fd >= 0);
        }
        {
            uvjob* j = np_job_new(UVJOB_READ, fd, ytype, gr);
            if (j != nilptr) {
                np_enqueue(j);
            }
        }
        return;
    }

    uvjob* j = np_job_new(UVJOB_TIMER, ns, ytype, gr);
    if (j == nilptr) {
        return;
    }
    j->deadline_ns = uv_hrtime() + (uint64_t)ns;
    np_enqueue(j);
}

/*
 * [D9] 与 event 版 evdns 的差异（唯一无法消除的实现差异，务必知悉）：
 *  - event: evdns_base_new(base,1) 自带 DNS 客户端，读 /etc/resolv.conf，
 *           且 DNS_OPTIONS_ALL 含 HOSTSFILE，所以也读 /etc/hosts。
 *  - uv   : uv_getaddrinfo 在线程池里调系统 getaddrinfo()，走 NSS
 *           （systemd-resolved / mDNS / LDAP / SSSD 等）。
 *  常见 hosts + A/AAAA 解析结果一致；NSS 插件、search/ndots 边界、
 *  超时重试策略、地址排序可能不同。异步契约一致。
 *  并发度也不同：event 版 evdns 允许 64 个 inflight 请求；libuv 使用
 *  全局线程池（默认 UV_THREADPOOL_SIZE=4，且与 fs 操作共享），
 *  大量并发解析时吞吐/排队行为会有差异。
 *
 * [D9] 与 event 版的"立即完成"契约差异：
 * event 的 evdns_getaddrinfo 对 /etc/hosts 命中或错误可能同步回调并
 * 返回 NULL，调用方（iohook2.c）据此不 yield；libuv 的回调永远异步，
 * 所以这里恒返回非 NULL，调用方走正常 yield 路径。
 * iohook2.c 已同时支持这两种返回值，行为等价。
 */
void* netpoller_dnsresolv(const char* hostname, int ytype, fiber* gr,
                          struct addrinfo** addr, int* errcode) {
    netpoller* np = gnpl__;
    assert(np != 0);
    (void)np;

    uvdns* r = crn_raw_malloc(sizeof(uvdns));
    if (r == nilptr) {
        return nilptr;
    }
    r->ytype = ytype;
    r->grid = gr->id;
    r->mcid = gr->mcid;
    r->data = gr;
    r->out = (void**)addr;
    r->errcode = errcode;

    /* 令牌：机器线程入列（单写者），DNS 完成时认领 */
    np_token* tok = (np_token*)crn_gc_malloc(sizeof(np_token));
    if (tok != nilptr) {
        tok->kind = NPT_DNS;
        tok->obj = r;
        crn_npreg_add(gr, tok);
        r->tok = tok;
    }

    if (hostname != nilptr) {
        size_t n = strlen(hostname) + 1;
        r->node = crn_raw_malloc(n);
        if (r->node != nilptr) {
            memcpy(r->node, hostname, n);
        }
    }

    uvjob* j = np_job_new(UVJOB_DNS, 0, ytype, gr);
    if (j == nilptr) {
        /* 极端 OOM：返回 NULL 会让 iohook2.c 走"立即完成"分支，
         * 那里在 errcode==0 时会 assert(res2 != nilptr)。event 版在
         * 同等 OOM 下也没有干净路径；此处只保证不泄漏。 */
        if (r->node != nilptr) {
            crn_raw_free(r->node);
        }
        crn_raw_free(r);
        return nilptr;
    }
    j->dns = r;
    np_enqueue(j);
    return r;
}
