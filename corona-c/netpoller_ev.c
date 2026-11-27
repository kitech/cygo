#include <ev.h>
#include <stdint.h>
#include <inttypes.h>
#include <stdbool.h>
#include <string.h>
#include <stdlib.h>
#include <errno.h>
#include <pthread.h>
#include <netdb.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <netinet/in.h>

#include "coronapriv.h"
#include "rxilog.h"

/*
 * libev 后端（未接入构建目标）。
 *
 * 与 netpoller_event.c 的语义差异：
 *  1) libev 没有独立 EV_CLOSED：用 EV_ERROR 以及读/写事件近似 event 版的
 *     EV_CLOSED，写方向的半关闭唤醒不如 libevent 精确。
 *  2) libev 每个 fd 每个方向只允许一个 ev_io watcher，因此这里按 fd 聚合
 *     一个 ev_io + rd/wr 等待者链表；event 版是每个 yield 一个 struct event。
 *  3) DNS 用 worker pthread 调 getaddrinfo，不是 event 版的 evdns；
 *     异步契约一致，NSS/结果顺序可能不同。
 *  4) 所有 ev_* 调用只在 loop 线程执行；machine 线程通过 ev_async 入队。
 *  5) IO 超时用每个等待者独立的 3.456s ev_timer，对齐 event 版。
 */

extern void crn_pre_gclock_proc(const char* funcname);
extern void crn_post_gclock_proc(const char* funcname);
extern bool crn_procer_resume_prechk(void* gr_, int ytype, int grid, int mcid);

#define EV_IO_TIMEOUT_S 3.456

typedef struct netpoller netpoller;
typedef struct evfd evfd;
typedef struct evwaiter evwaiter;

typedef struct evdata {
    void* data; // fiber*
    int grid;
    int mcid;
    int ytype;
    int64_t fd; // fd or ns
} evdata;

typedef struct evwaiter {
    struct evwaiter* next;
    struct evwaiter* gnext;
    evfd* f;
    int want; // EV_READ / EV_WRITE
    int done;
    evdata d;
    ev_timer tmo; // [差异 5] 每个等待者独立的 3.456s 超时
} evwaiter;

struct evfd {
    int fd;
    int events;
    ev_io io; // [差异 2] 同一 fd 的聚合 watcher
    evwaiter* rds;
    evwaiter* wrs;
};

typedef struct evsleep {
    ev_timer t;
    struct evsleep* gnext;
    evdata d;
} evsleep;

typedef struct evdns_req {
    char* node;
    int ytype;
    int grid;
    int mcid;
    void* data; // fiber*
    void** out; // struct addrinfo**
    int* errcode;
    struct addrinfo* res;
    int status;
    struct evdns_req* next;
} evdns_req;

enum evjobtyp {
    EVJOB_READ = 1,
    EVJOB_WRITE,
    EVJOB_TIMER,
};

typedef struct evjob {
    struct evjob* next;
    int typ;
    int64_t fd; // fd
    int64_t ns; // timer
    int ytype;
    int grid;
    int mcid;
    void* data; // fiber*
} evjob;

struct netpoller {
    struct ev_loop* loop;
    ev_async async;     // [差异 4] 唯一允许跨线程的唤醒方式
    ev_prepare prepare; // 在 poll 前统一处理 job/garbage/dns
    ev_timer heartbeat; // 586ms，对齐 event 版
    pmutex_t qmu;       // job 队列 + dns 完成队列
    evjob* qhead;
    evjob* qtail;
    evdns_req* dhead;
    evdns_req* dtail;
    evfd** fds; // fd -> 聚合 watcher（只增不回收）
    size_t nfds;
    evwaiter* gwaiters; // 待释放的等待者（callback 已返回）
    evsleep* gsleeps;   // 待释放的 sleep timer
};

static netpoller* gnpl__ = 0;

static void ev_io_cb(struct ev_loop* loop, ev_io* w, int revents);
static void ev_io_timeout_cb(struct ev_loop* loop, ev_timer* w, int revents);
static void ev_sleep_cb(struct ev_loop* loop, ev_timer* w, int revents);
static void ev_async_cb(struct ev_loop* loop, ev_async* w, int revents);
static void ev_prepare_cb(struct ev_loop* loop, ev_prepare* w, int revents);
static void ev_heartbeat_cb(struct ev_loop* loop, ev_timer* w, int revents);

/* ------------------------------------------------------------------ */
/* fd -> 聚合 watcher                                                   */
/* ------------------------------------------------------------------ */

static evfd* ev_fd_get(netpoller* np, int fd) {
    if (fd < 0) {
        return nilptr;
    }
    if ((size_t)fd >= np->nfds) {
        size_t ncap = np->nfds ? np->nfds : 64;
        while (ncap <= (size_t)fd) {
            ncap *= 2;
        }
        evfd** nfds2 = realloc(np->fds, ncap * sizeof(evfd*));
        if (nfds2 == nilptr) {
            return nilptr;
        }
        memset(nfds2 + np->nfds, 0, (ncap - np->nfds) * sizeof(evfd*));
        np->fds = nfds2;
        np->nfds = ncap;
    }

    evfd* f = np->fds[fd];
    if (f != nilptr) {
        return f;
    }
    f = crn_raw_malloc(sizeof(evfd));
    if (f == nilptr) {
        return nilptr;
    }
    f->fd = fd;
    f->events = 0;
    f->rds = nilptr;
    f->wrs = nilptr;
    ev_io_init(&f->io, ev_io_cb, fd, EV_READ);
    f->io.data = f;
    np->fds[fd] = f;
    return f;
}

static void ev_fd_update(netpoller* np, evfd* f) {
    int ev = 0;
    if (f->rds != nilptr) {
        ev |= EV_READ;
    }
    if (f->wrs != nilptr) {
        ev |= EV_WRITE;
    }
    if (ev == f->events) {
        return;
    }
    if (ev == 0) {
        ev_io_stop(np->loop, &f->io);
        f->events = 0;
        return;
    }
    ev_io_stop(np->loop, &f->io);
    ev_io_set(&f->io, f->fd, ev); // 与 event 版重新 event_add 同义
    f->events = ev;
    ev_io_start(np->loop, &f->io);
}

/* ------------------------------------------------------------------ */
/* 唤醒 / 回收                                                         */
/* ------------------------------------------------------------------ */

static evwaiter* ev_unlink(evwaiter** head, evwaiter* w) {
    evwaiter** pp = head;
    while (*pp != nilptr) {
        if (*pp == w) {
            *pp = w->next;
            w->next = nilptr;
            return w;
        }
        pp = &(*pp)->next;
    }
    return nilptr;
}

static void ev_waiter_free_later(netpoller* np, evwaiter* w) {
    w->gnext = np->gwaiters;
    np->gwaiters = w;
}

static void ev_sleep_free_later(netpoller* np, evsleep* s) {
    s->gnext = np->gsleeps;
    np->gsleeps = s;
}

/*
 * event 版是 event_del+event_free 先于 resume；这里先 stop/摘链再 resume，
 * watcher 内存延后到 prepare 释放，避免 libev 在 callback 返回后访问。
 */
static void ev_waiter_resume(netpoller* np, evwaiter* w) {
    if (w->done) {
        return;
    }
    w->done = 1;
    ev_timer_stop(np->loop, &w->tmo);
    void* data = w->d.data;
    int grid = w->d.grid;
    int mcid = w->d.mcid;
    int ytype = w->d.ytype;
    ev_waiter_free_later(np, w);
    crn_procer_resume_one(data, ytype, grid, mcid);
}

static void ev_wake_list(netpoller* np, evwaiter** head) {
    while (*head != nilptr) {
        evwaiter* w = *head;
        *head = w->next;
        w->next = nilptr;
        ev_waiter_resume(np, w);
    }
}

static void ev_io_cb(struct ev_loop* loop, ev_io* w, int revents) {
    netpoller* np = gnpl__;
    evfd* f = (evfd*)w->data;
    assert(np != nilptr);
    assert(f != nilptr);
    (void)loop;

    /* [差异 1] libev 没有 EV_CLOSED：EV_ERROR 时按关闭处理。 */
    if (revents & EV_ERROR) {
        ev_wake_list(np, &f->rds);
        ev_wake_list(np, &f->wrs);
    } else {
        if (revents & EV_READ) {
            ev_wake_list(np, &f->rds);
        }
        if (revents & EV_WRITE) {
            ev_wake_list(np, &f->wrs);
        }
    }
    ev_fd_update(np, f);
}

static void ev_io_timeout_cb(struct ev_loop* loop, ev_timer* w, int revents) {
    netpoller* np = gnpl__;
    evwaiter* x = (evwaiter*)w->data;
    assert(np != nilptr);
    assert(x != nilptr);
    (void)loop;
    (void)revents;
    if (x->done) {
        return;
    }
    evfd* f = x->f;
    if (ev_unlink(&f->rds, x) == nilptr) {
        ev_unlink(&f->wrs, x);
    }
    ev_waiter_resume(np, x);
    ev_fd_update(np, f);
}

static void ev_sleep_cb(struct ev_loop* loop, ev_timer* w, int revents) {
    netpoller* np = gnpl__;
    evsleep* s = (evsleep*)w->data;
    assert(np != nilptr);
    assert(s != nilptr);
    (void)loop;
    (void)revents;

    ev_timer_stop(np->loop, &s->t);
    void* data = s->d.data;
    int grid = s->d.grid;
    int mcid = s->d.mcid;
    int ytype = s->d.ytype;
    ev_sleep_free_later(np, s);
    crn_procer_resume_one(data, ytype, grid, mcid);
}

/* ------------------------------------------------------------------ */
/* job 处理（只在 loop 线程、ev_prepare 中）                            */
/* ------------------------------------------------------------------ */

static void ev_start_io(netpoller* np, evjob* j) {
    evfd* f = ev_fd_get(np, (int)j->fd);
    if (f == nilptr) {
        lwarn("add error fd=%" PRId64 " grid=%d\n", j->fd, j->grid);
        return;
    }
    evwaiter* w = crn_raw_malloc(sizeof(evwaiter));
    if (w == nilptr) {
        return;
    }
    memset(w, 0, sizeof(*w));
    w->f = f;
    w->want = (j->typ == EVJOB_WRITE) ? EV_WRITE : EV_READ;
    w->d.data = j->data;
    w->d.grid = j->grid;
    w->d.mcid = j->mcid;
    w->d.ytype = j->ytype;
    w->d.fd = j->fd;

    ev_timer_init(&w->tmo, ev_io_timeout_cb, EV_IO_TIMEOUT_S, 0);
    w->tmo.data = w;
    ev_timer_start(np->loop, &w->tmo);

    /* 头插 => LIFO，与 event 版一致 */
    if (w->want == EV_READ) {
        w->next = f->rds;
        f->rds = w;
    } else {
        w->next = f->wrs;
        f->wrs = w;
    }
    ev_fd_update(np, f);
}

static void ev_start_timer(netpoller* np, evjob* j) {
    evsleep* s = crn_raw_malloc(sizeof(evsleep));
    if (s == nilptr) {
        return;
    }
    memset(s, 0, sizeof(*s));
    s->d.data = j->data;
    s->d.grid = j->grid;
    s->d.mcid = j->mcid;
    s->d.ytype = j->ytype;
    s->d.fd = j->ns;
    ev_timer_init(&s->t, ev_sleep_cb, (double)j->ns / 1000000000.0, 0);
    s->t.data = s;
    ev_timer_start(np->loop, &s->t);
}

static void ev_prepare_cb(struct ev_loop* loop, ev_prepare* pw, int revents) {
    netpoller* np = gnpl__;
    assert(np != nilptr);
    (void)loop;
    (void)pw;
    (void)revents;

    evjob* q = nilptr;
    pthread_mutex_lock(&np->qmu);
    q = np->qhead;
    np->qhead = nilptr;
    np->qtail = nilptr;
    pthread_mutex_unlock(&np->qmu);

    while (q != nilptr) {
        evjob* next = q->next;
        if (q->typ == EVJOB_TIMER) {
            ev_start_timer(np, q);
        } else {
            ev_start_io(np, q);
        }
        crn_raw_free(q);
        q = next;
    }

    evdns_req* dq = nilptr;
    pthread_mutex_lock(&np->qmu);
    dq = np->dhead;
    np->dhead = nilptr;
    np->dtail = nilptr;
    pthread_mutex_unlock(&np->qmu);

    while (dq != nilptr) {
        evdns_req* r = dq;
        dq = r->next;
        if (crn_procer_resume_prechk(r->data, r->ytype, r->grid, r->mcid)) {
            if (r->out != nilptr) {
                *r->out = r->res;
            }
            if (r->errcode != nilptr) {
                *r->errcode = r->status;
            }
            crn_procer_resume_one(r->data, r->ytype, r->grid, r->mcid);
        } else {
            lwarn("oops, seems gr gone %d\n", r->grid);
            if (r->res != nilptr) {
                freeaddrinfo(r->res);
            }
        }
        if (r->node != nilptr) {
            crn_raw_free(r->node);
        }
        crn_raw_free(r);
    }

    while (np->gwaiters != nilptr) {
        evwaiter* w = np->gwaiters;
        np->gwaiters = w->gnext;
        crn_raw_free(w);
    }
    while (np->gsleeps != nilptr) {
        evsleep* s = np->gsleeps;
        np->gsleeps = s->gnext;
        crn_raw_free(s);
    }
}

static void ev_async_cb(struct ev_loop* loop, ev_async* w, int revents) {
    (void)loop;
    (void)w;
    (void)revents;
    /* 只负责唤醒；实际工作留给 ev_prepare_cb */
}

static void ev_heartbeat_cb(struct ev_loop* loop, ev_timer* w, int revents) {
    (void)loop;
    (void)w;
    (void)revents;
}

/* ------------------------------------------------------------------ */
/* DNS：worker 线程调 getaddrinfo，完成结果回到 loop 线程              */
/* ------------------------------------------------------------------ */

/* [差异 3] event 版用 evdns；这里用 worker pthread + getaddrinfo。 */
static void* ev_dns_worker(void* arg) {
    evdns_req* r = (evdns_req*)arg;
    netpoller* np = gnpl__;
    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    hints.ai_protocol = IPPROTO_TCP;
    hints.ai_flags = AI_CANONNAME;

    r->status = getaddrinfo(r->node, nilptr, &hints, &r->res);

    pthread_mutex_lock(&np->qmu);
    r->next = nilptr;
    if (np->dtail != nilptr) {
        np->dtail->next = r;
    } else {
        np->dhead = r;
    }
    np->dtail = r;
    pthread_mutex_unlock(&np->qmu);
    ev_async_send(np->loop, &np->async);
    return nilptr;
}

/* ------------------------------------------------------------------ */
/* 对外接口                                                            */
/* ------------------------------------------------------------------ */

void netpoller_use_threads() {
}

netpoller* netpoller_new() {
    assert(gnpl__ == 0);
    netpoller* np = crn_raw_malloc(sizeof(netpoller));
    assert(np != nilptr);
    memset(np, 0, sizeof(*np));
    pmutex_init(&np->qmu, nilptr);

    np->loop = ev_loop_new(ev_recommended_backends());
    assert(np->loop != nilptr);

    ev_async_init(&np->async, ev_async_cb);
    np->async.data = np;
    ev_async_start(np->loop, &np->async);

    ev_prepare_init(&np->prepare, ev_prepare_cb);
    np->prepare.data = np;
    ev_prepare_start(np->loop, &np->prepare);

    ev_timer_init(&np->heartbeat, ev_heartbeat_cb, 0.586, 0.586);

    gnpl__ = np;
    log_info("using backend libev");
    return np;
}

void netpoller_loop() {
    netpoller* np = gnpl__;
    assert(np != nilptr);

    ev_timer_start(np->loop, &np->heartbeat);
    for (;;) {
        ev_run(np->loop, 0);
    }
    assert(1==2);
}

/* 与 event 版相同的 ytype 分派；这里只入队，所有 ev_* 在 loop 线程执行。 */
void netpoller_yieldfd(int64_t fd, int ytype, fiber* gr) {
    netpoller* np = gnpl__;
    assert(np != nilptr);
    assert(ytype > YIELD_TYPE_NONE);
    assert(ytype < YIELD_TYPE_MAX);

    int64_t ns = 0;
    int typ = -1;
    switch (ytype) {
    case YIELD_TYPE_SLEEP:
        ns = fd * 1000000000LL;
        typ = EVJOB_TIMER;
        break;
    case YIELD_TYPE_MSLEEP:
        ns = fd * 1000000LL;
        typ = EVJOB_TIMER;
        break;
    case YIELD_TYPE_USLEEP:
        ns = fd * 1000LL;
        typ = EVJOB_TIMER;
        break;
    case YIELD_TYPE_NANOSLEEP:
    case YIELD_TYPE_TICK_WAKE:
        ns = fd;
        typ = EVJOB_TIMER;
        break;
    case YIELD_TYPE_CHAN_SEND:
    case YIELD_TYPE_CHAN_RECV:
        assert(1==2); // event 版同样断言：不应走到 netpoller
        return;
    case YIELD_TYPE_LOCK:
        // 没有 fd 可注册，crn_mutex_unlock 直接 resume
        return;
    case YIELD_TYPE_CONNECT:
    case YIELD_TYPE_WRITE:
    case YIELD_TYPE_WRITEV:
    case YIELD_TYPE_SEND:
    case YIELD_TYPE_SENDTO:
    case YIELD_TYPE_SENDMSG:
        typ = EVJOB_WRITE;
        break;
    case YIELD_TYPE_GETADDRINFO:
        // DNS 已在 netpoller_dnsresolv 注册；这里与 event 版一样什么都不做
        return;
    default:
        if (fd < 0) {
            lwarn("wtf yield fd=%" PRId64 " ytype=%d grid=%d\n", fd, ytype, gr->id);
            assert(fd >= 0);
        }
        typ = EVJOB_READ;
        break;
    }

    evjob* j = crn_raw_malloc(sizeof(evjob));
    if (j == nilptr) {
        return;
    }
    memset(j, 0, sizeof(*j));
    j->typ = typ;
    j->fd = fd;
    j->ns = ns;
    j->ytype = ytype;
    j->grid = gr->id;
    j->mcid = gr->mcid;
    j->data = gr;

    crn_pre_gclock_proc(__func__);
    pthread_mutex_lock(&np->qmu);
    if (np->qtail != nilptr) {
        np->qtail->next = j;
    } else {
        np->qhead = j;
    }
    np->qtail = j;
    pthread_mutex_unlock(&np->qmu);
    crn_post_gclock_proc(__func__);
    ev_async_send(np->loop, &np->async);
}

void* netpoller_dnsresolv(const char* hostname, int ytype, fiber* gr,
                          struct addrinfo** addr, int* errcode) {
    netpoller* np = gnpl__;
    assert(np != nilptr);

    evdns_req* r = crn_raw_malloc(sizeof(evdns_req));
    if (r == nilptr) {
        return nilptr;
    }
    memset(r, 0, sizeof(*r));
    r->ytype = ytype;
    r->grid = gr->id;
    r->mcid = gr->mcid;
    r->data = gr;
    r->out = (void**)addr;
    r->errcode = errcode;

    if (hostname != nilptr) {
        size_t n = strlen(hostname) + 1;
        r->node = crn_raw_malloc(n);
        if (r->node != nilptr) {
            memcpy(r->node, hostname, n);
        }
    }

    pthread_t th;
    int rv = pthread_create(&th, nilptr, ev_dns_worker, r);
    if (rv != 0) {
        lwarn("dns pthread_create failed %d\n", rv);
        if (r->node != nilptr) {
            crn_raw_free(r->node);
        }
        crn_raw_free(r);
        if (errcode != nilptr) {
            *errcode = EAI_FAIL;
        }
        if (addr != nilptr) {
            *addr = nilptr;
        }
        return nilptr;
    }
    pthread_detach(th);
    return r;
}
